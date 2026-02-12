package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/acchapm1/ocmgr-app/internal/copier"
	"github.com/acchapm1/ocmgr-app/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [target-dir]",
	Short: "Initialize .opencode directory from a profile",
	Long: `Initialize a .opencode directory by copying one or more profile
contents into the target directory. If no target directory is
specified, the current working directory is used.

Multiple profiles can be layered by passing --profile more than once;
they are applied in order so later profiles override earlier ones.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceP("profile", "p", nil, "profile name(s) to apply (required, may be repeated)")
	initCmd.Flags().BoolP("force", "f", false, "overwrite existing files without prompting")
	initCmd.Flags().BoolP("merge", "m", false, "only copy new files, skip existing ones")
	initCmd.Flags().BoolP("dry-run", "d", false, "preview changes without copying")
	_ = initCmd.MarkFlagRequired("profile")
}

func runInit(cmd *cobra.Command, args []string) error {
	profileNames, _ := cmd.Flags().GetStringSlice("profile")
	force, _ := cmd.Flags().GetBool("force")
	merge, _ := cmd.Flags().GetBool("merge")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Validate mutually exclusive flags.
	if force && merge {
		return fmt.Errorf("--force and --merge are mutually exclusive")
	}

	// Resolve target directory.
	targetDir := "."
	if len(args) == 1 {
		targetDir = args[0]
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("cannot resolve target directory: %w", err)
	}
	targetOpencode := filepath.Join(absTarget, ".opencode")

	// Open the profile store.
	s, err := store.NewStore()
	if err != nil {
		return fmt.Errorf("cannot open store: %w", err)
	}

	// Load every requested profile up-front so we fail fast.
	type loadedProfile struct {
		name string
		path string
	}
	profiles := make([]loadedProfile, 0, len(profileNames))
	for _, name := range profileNames {
		p, err := s.Get(name)
		if err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		profiles = append(profiles, loadedProfile{name: name, path: p.Path})
	}

	// Determine copy strategy.
	var strategy copier.Strategy
	switch {
	case force:
		strategy = copier.StrategyOverwrite
	case merge:
		strategy = copier.StrategyMerge
	default:
		strategy = copier.StrategyPrompt
	}

	// Build copy options.
	opts := copier.Options{
		Strategy: strategy,
		DryRun:   dryRun,
		OnConflict: func(src, dst string) (copier.ConflictChoice, error) {
			relPath, _ := filepath.Rel(targetOpencode, dst)
			fmt.Fprintf(os.Stderr, "Conflict: %s\n", relPath)
			fmt.Fprintf(os.Stderr, "  [o]verwrite  [s]kip  [c]ompare  [a]bort\n")
			scanner := bufio.NewScanner(os.Stdin)
			for {
				fmt.Fprintf(os.Stderr, "Choice: ")
				if !scanner.Scan() {
					return copier.ChoiceCancel, nil
				}
				switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
				case "o":
					return copier.ChoiceOverwrite, nil
				case "s":
					return copier.ChoiceSkip, nil
				case "c":
					diff := exec.Command("diff", "--color=always", src, dst)
					diff.Stdout = os.Stdout
					diff.Stderr = os.Stderr
					if err := diff.Run(); err != nil {
						// diff returns exit code 1 when files differ — that's expected.
						// Only warn if the command itself failed to run.
						if diff.ProcessState == nil || !diff.ProcessState.Exited() {
							fmt.Fprintf(os.Stderr, "  (diff command failed: %v)\n", err)
						}
					}
					return copier.ChoiceCompare, nil
				case "a":
					return copier.ChoiceCancel, nil
				default:
					continue
				}
			}
		},
	}

	prefix := ""
	if dryRun {
		prefix = "[dry run] "
	}

	// Apply each profile in order.
	for _, lp := range profiles {
		fmt.Printf("%sApplying profile %q …\n", prefix, lp.name)

		result, err := copier.CopyProfile(lp.path, targetOpencode, opts)
		if err != nil {
			return fmt.Errorf("copying profile %q: %w", lp.name, err)
		}

		// Summary: copied files.
		if len(result.Copied) > 0 {
			fmt.Printf("%s✓ Copied %d files\n", prefix, len(result.Copied))
			for _, f := range result.Copied {
				fmt.Printf("    %s\n", f)
			}
		}

		// Summary: skipped files.
		if len(result.Skipped) > 0 {
			fmt.Printf("%s→ Skipped %d files\n", prefix, len(result.Skipped))
			for _, f := range result.Skipped {
				fmt.Printf("    %s\n", f)
			}
		}

		// Summary: errors.
		if len(result.Errors) > 0 {
			fmt.Printf("%s✗ %d errors\n", prefix, len(result.Errors))
			for _, e := range result.Errors {
				fmt.Printf("    %s\n", e)
			}
		}
	}

	// Check for plugin dependencies.
	if copier.DetectPluginDeps(targetOpencode) {
		fmt.Fprintf(os.Stderr, "Plugin dependencies detected. Install now? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer == "y" {
				if dryRun {
					fmt.Printf("[dry run] Would run: bun install in %s\n", targetOpencode)
				} else {
					install := exec.Command("bun", "install")
					install.Dir = targetOpencode
					install.Stdout = os.Stdout
					install.Stderr = os.Stderr
					if err := install.Run(); err != nil {
						return fmt.Errorf("bun install failed: %w", err)
					}
				}
			} else {
				fmt.Printf("To install later, run: cd %s && bun install\n", targetOpencode)
			}
		}
	}

	return nil
}
