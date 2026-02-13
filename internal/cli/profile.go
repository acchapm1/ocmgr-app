package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/acchapm1/ocmgr/internal/github"
	"github.com/acchapm1/ocmgr/internal/profile"
	"github.com/acchapm1/ocmgr/internal/store"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "List, show, create, and delete profiles in the local store.",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles in the local store",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		profiles, err := s.List()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found. Create one with: ocmgr profile create <name>")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "NAME\tVERSION\tDESCRIPTION\tTAGS\n")
		for _, p := range profiles {
			desc := p.Description
			if len(desc) > 42 {
				desc = desc[:42] + "..."
			}
			tags := strings.Join(p.Tags, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Version, desc, tags)
		}
		w.Flush()

		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := s.Get(name)
		if err != nil {
			return err
		}

		fmt.Printf("Profile: %s\n", p.Name)
		if p.Description != "" {
			fmt.Printf("Description: %s\n", p.Description)
		}
		if p.Version != "" {
			fmt.Printf("Version: %s\n", p.Version)
		}
		if p.Author != "" {
			fmt.Printf("Author: %s\n", p.Author)
		}
		if len(p.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(p.Tags, ", "))
		}
		if p.Extends != "" {
			fmt.Printf("Extends: %s\n", p.Extends)
		}

		contents, err := profile.ListContents(p)
		if err != nil {
			return fmt.Errorf("listing contents: %w", err)
		}

		fmt.Println()
		fmt.Println("Contents:")

		if len(contents.Agents) > 0 {
			fmt.Printf("  agents/ (%d files)\n", len(contents.Agents))
			for _, f := range contents.Agents {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "agents/"))
			}
		}

		if len(contents.Commands) > 0 {
			fmt.Printf("  commands/ (%d files)\n", len(contents.Commands))
			for _, f := range contents.Commands {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "commands/"))
			}
		}

		if len(contents.Skills) > 0 {
			fmt.Printf("  skills/ (%d skills)\n", len(contents.Skills))
			for _, f := range contents.Skills {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "skills/"))
			}
		}

		if len(contents.Plugins) > 0 {
			fmt.Printf("  plugins/ (%d files)\n", len(contents.Plugins))
			for _, f := range contents.Plugins {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "plugins/"))
			}
		}

		return nil
	},
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new empty profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := profile.ScaffoldProfile(s.Dir, name)
		if err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}

		fmt.Printf("Created profile '%s' at %s\n", name, p.Path)
		fmt.Println("Add files to agents/, commands/, skills/, plugins/ directories.")
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile from the local store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		if !force {
			fmt.Printf("Delete profile '%s'? This cannot be undone. [y/N] ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := s.Delete(name); err != nil {
			return err
		}

		fmt.Printf("Deleted profile '%s'\n", name)
		return nil
	},
}

// ── profile import ────────────────────────────────────────────────

var profileImportCmd = &cobra.Command{
	Use:   "import <source>",
	Short: "Import a profile from a local directory or GitHub URL",
	Long: `Import a profile into the local store.

The source can be:
  - A local directory containing a valid profile.toml
  - A GitHub URL (https://github.com/<owner>/<repo>/tree/<branch>/profiles/<name>)

Examples:
  ocmgr profile import /path/to/my-profile
  ocmgr profile import https://github.com/user/opencode-profiles/tree/main/profiles/go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		var srcDir string
		var tmpDir string

		if isGitHubURL(source) {
			// Parse the URL and clone the repo to extract the profile.
			repo, branch, profilePath, err := parseGitHubProfileURL(source)
			if err != nil {
				return err
			}

			tmpDir, err = os.MkdirTemp("", "ocmgr-import-*")
			if err != nil {
				return fmt.Errorf("creating temp dir: %w", err)
			}
			defer os.RemoveAll(tmpDir)

			cloneURL := fmt.Sprintf("https://github.com/%s.git", repo)
			cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, cloneURL, tmpDir)
			cloneCmd.Stderr = os.Stderr
			if err := cloneCmd.Run(); err != nil {
				return fmt.Errorf("cloning %s: %w", repo, err)
			}

			srcDir = filepath.Join(tmpDir, profilePath)
		} else {
			// Local directory.
			abs, err := filepath.Abs(source)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}
			srcDir = abs
		}

		// Validate the source is a proper profile.
		p, err := github.ValidateProfileDir(srcDir)
		if err != nil {
			return err
		}

		if s.Exists(p.Name) {
			return fmt.Errorf("profile %q already exists; delete it first with 'ocmgr profile delete %s'", p.Name, p.Name)
		}

		// Copy the profile into the store.
		targetDir := filepath.Join(s.Dir, p.Name)
		if err := github.CopyDirRecursive(srcDir, targetDir); err != nil {
			return fmt.Errorf("importing profile: %w", err)
		}

		fmt.Printf("✓ Imported profile %q to %s\n", p.Name, targetDir)
		return nil
	},
}

// ── profile export ────────────────────────────────────────────────

var profileExportCmd = &cobra.Command{
	Use:   "export <name> <target-dir>",
	Short: "Export a profile to a local directory",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		targetDir := args[1]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := s.Get(name)
		if err != nil {
			return err
		}

		abs, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("resolving target: %w", err)
		}

		dst := filepath.Join(abs, name)
		if err := github.CopyDirRecursive(p.Path, dst); err != nil {
			return fmt.Errorf("exporting profile: %w", err)
		}

		fmt.Printf("✓ Exported profile %q to %s\n", name, dst)
		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────

// isGitHubURL checks if a string looks like a GitHub URL.
func isGitHubURL(s string) bool {
	return strings.HasPrefix(s, "https://github.com/") ||
		strings.HasPrefix(s, "http://github.com/")
}

// parseGitHubProfileURL extracts repo, branch and path from a URL like
// https://github.com/user/repo/tree/main/profiles/go
func parseGitHubProfileURL(url string) (repo, branch, profilePath string, err error) {
	// Strip protocol prefix.
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimSuffix(url, "/")

	// Expected format: <owner>/<repo>/tree/<branch>/<path...>
	parts := strings.SplitN(url, "/", 5)
	if len(parts) < 5 || parts[2] != "tree" {
		return "", "", "", fmt.Errorf("cannot parse GitHub URL; expected format: https://github.com/<owner>/<repo>/tree/<branch>/<path>")
	}

	repo = parts[0] + "/" + parts[1]
	branch = parts[3]
	profilePath = parts[4]
	return repo, branch, profilePath, nil
}

func init() {
	profileDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)
	profileCmd.AddCommand(profileImportCmd)
	profileCmd.AddCommand(profileExportCmd)
}
