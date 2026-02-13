package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/acchapm1/ocmgr/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ocmgr configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		fmt.Printf("Configuration (~/.ocmgr/config.toml):\n\n")
		fmt.Printf("[github]\n")
		fmt.Printf("  %-16s = %s\n", "repo", cfg.GitHub.Repo)
		fmt.Printf("  %-16s = %s\n", "auth", cfg.GitHub.Auth)
		fmt.Printf("\n")
		fmt.Printf("[defaults]\n")
		fmt.Printf("  %-16s = %s\n", "merge_strategy", cfg.Defaults.MergeStrategy)
		fmt.Printf("  %-16s = %s\n", "editor", cfg.Defaults.Editor)
		fmt.Printf("\n")
		fmt.Printf("[store]\n")
		fmt.Printf("  %-16s = %s\n", "path", cfg.Store.Path)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		switch key {
		case "github.repo":
			cfg.GitHub.Repo = value
		case "github.auth":
			validAuth := map[string]bool{"gh": true, "env": true, "ssh": true, "token": true}
			if !validAuth[value] {
				return fmt.Errorf("invalid auth method %q; must be one of: gh, env, ssh, token", value)
			}
			cfg.GitHub.Auth = value
		case "defaults.merge_strategy":
			validStrategies := map[string]bool{"prompt": true, "overwrite": true, "merge": true, "skip": true}
			if !validStrategies[value] {
				return fmt.Errorf("invalid merge strategy %q; must be one of: prompt, overwrite, merge, skip", value)
			}
			cfg.Defaults.MergeStrategy = value
		case "defaults.editor":
			cfg.Defaults.Editor = value
		case "store.path":
			cfg.Store.Path = value
		default:
			return fmt.Errorf("unrecognized key %q\nValid keys: github.repo, github.auth, defaults.merge_strategy, defaults.editor, store.path", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive first-run configuration setup",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		prompt := func(label, defaultVal string) string {
			fmt.Printf("%s [%s]: ", label, defaultVal)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				return defaultVal
			}
			return input
		}

		repo := prompt("GitHub repository (owner/repo)", "acchapm1/opencode-profiles")
		auth := prompt("Auth method (gh/env/ssh/token)", "gh")
		mergeStrategy := prompt("Default merge strategy (prompt/overwrite/merge/skip)", "prompt")
		editor := prompt("Editor", "nvim")

		cfg := &config.Config{
			GitHub: config.GitHub{
				Repo: repo,
				Auth: auth,
			},
			Defaults: config.Defaults{
				MergeStrategy: mergeStrategy,
				Editor:        editor,
			},
			Store: config.Store{
				Path: "~/.ocmgr/profiles",
			},
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Println("Configuration saved to ~/.ocmgr/config.toml")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configInitCmd)
}
