package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/acchapm1/ocmgr-app/internal/config"
	"github.com/acchapm1/ocmgr-app/internal/github"
	"github.com/acchapm1/ocmgr-app/internal/store"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize profiles with GitHub",
	Long: `Sync profiles between the local store (~/.ocmgr/profiles) and a
remote GitHub repository. The repository and auth method are
read from ~/.ocmgr/config.toml (see "ocmgr config show").

Use "ocmgr sync push" to upload a profile and "ocmgr sync pull"
to download. "ocmgr sync status" shows which profiles differ.`,
}

// ── sync push ─────────────────────────────────────────────────────

var syncPushCmd = &cobra.Command{
	Use:   "push <name>",
	Short: "Push a local profile to GitHub",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := s.Get(name)
		if err != nil {
			return err
		}

		fmt.Printf("Pushing profile %q to %s …\n", name, cfg.GitHub.Repo)

		if err := github.PushProfile(name, p.Path, cfg.GitHub.Repo, cfg.GitHub.Auth); err != nil {
			return fmt.Errorf("push failed: %w", err)
		}

		fmt.Printf("✓ Pushed profile %q\n", name)
		return nil
	},
}

// ── sync pull ─────────────────────────────────────────────────────

var syncPullCmd = &cobra.Command{
	Use:   "pull [name]",
	Short: "Pull a profile from GitHub (or --all)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		if all {
			fmt.Printf("Pulling all profiles from %s …\n", cfg.GitHub.Repo)
			pulled, err := github.PullAll(s.Dir, cfg.GitHub.Repo, cfg.GitHub.Auth)
			if err != nil {
				return fmt.Errorf("pull failed: %w", err)
			}
			if len(pulled) == 0 {
				fmt.Println("No profiles found in remote repository.")
				return nil
			}
			fmt.Printf("✓ Pulled %d profiles:\n", len(pulled))
			for _, name := range pulled {
				fmt.Printf("    %s\n", name)
			}
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("provide a profile name or use --all")
		}

		name := args[0]
		fmt.Printf("Pulling profile %q from %s …\n", name, cfg.GitHub.Repo)

		if err := github.PullProfile(name, s.Dir, cfg.GitHub.Repo, cfg.GitHub.Auth); err != nil {
			return fmt.Errorf("pull failed: %w", err)
		}

		fmt.Printf("✓ Pulled profile %q\n", name)
		return nil
	},
}

// ── sync status ───────────────────────────────────────────────────

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show sync status between local and remote profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		fmt.Printf("Comparing local profiles with %s …\n\n", cfg.GitHub.Repo)

		st, err := github.Status(s.Dir, cfg.GitHub.Repo, cfg.GitHub.Auth)
		if err != nil {
			return fmt.Errorf("status check failed: %w", err)
		}

		empty := len(st.InSync) == 0 && len(st.Modified) == 0 &&
			len(st.LocalOnly) == 0 && len(st.RemoteOnly) == 0

		if empty {
			fmt.Println("No profiles found locally or remotely.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "PROFILE\tSTATUS\n")

		for _, n := range st.InSync {
			fmt.Fprintf(w, "%s\t✓ in sync\n", n)
		}
		for _, n := range st.Modified {
			fmt.Fprintf(w, "%s\t~ modified (push or pull to sync)\n", n)
		}
		for _, n := range st.LocalOnly {
			fmt.Fprintf(w, "%s\t● local only (push to sync)\n", n)
		}
		for _, n := range st.RemoteOnly {
			fmt.Fprintf(w, "%s\t○ remote only (pull to sync)\n", n)
		}

		w.Flush()
		return nil
	},
}

func init() {
	syncPullCmd.Flags().Bool("all", false, "pull all remote profiles")

	syncCmd.AddCommand(syncPushCmd)
	syncCmd.AddCommand(syncPullCmd)
	syncCmd.AddCommand(syncStatusCmd)
}
