package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/acchapm1/ocmgr-app/internal/copier"
	"github.com/acchapm1/ocmgr-app/internal/resolver"
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
they are applied in order so later profiles override earlier ones.

If a profile has an "extends" field in its profile.toml, the parent
profile is automatically included before the child. Circular
dependencies are detected and reported as errors.

Use --only or --exclude to limit which content directories are copied
(agents, commands, skills, plugins).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceP("profile", "p", nil, "profile name(s) to apply (required, may be repeated)")
	initCmd.Flags().BoolP("force", "f", false, "overwrite existing files without prompting")
	initCmd.Flags().BoolP("merge", "m", false, "only copy new files, skip existing ones")
	initCmd.Flags().BoolP("dry-run", "d", false, "preview changes without copying")
	initCmd.Flags().StringP("only", "o", "", "content dirs to include (comma-separated: agents,commands,skills,plugins)")
	initCmd.Flags().StringP("exclude", "e", "", "content dirs to exclude (comma-separated: agents,commands,skills,plugins)")
	_ = initCmd.MarkFlagRequired("profile")
}

func runInit(cmd *cobra.Command, args []string) error {
	profileNames, _ := cmd.Flags().GetStringSlice("profile")
	force, _ := cmd.Flags().GetBool("force")
	merge, _ := cmd.Flags().GetBool("merge")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	onlyRaw, _ := cmd.Flags().GetString("only")
	excludeRaw, _ := cmd.Flags().GetString("exclude")

	// Validate mutually exclusive flags.
	if force && merge {
		return fmt.Errorf("--force and --merge are mutually exclusive")
	}
	if onlyRaw != "" && excludeRaw != "" {
		return fmt.Errorf("--only and --exclude are mutually exclusive")
	}

	// Parse and validate --only / --exclude values.
	includeDirs, err := parseContentDirs(onlyRaw)
	if err != nil {
		return fmt.Errorf("--only: %w", err)
	}
	excludeDirs, err := parseContentDirs(excludeRaw)
	if err != nil {
		return fmt.Errorf("--exclude: %w", err)
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

	// Resolve the extends dependency chain for all requested profiles.
	// This expands "go" (extends "base") into ["base", "go"] so parents
	// are applied first.
	resolved, err := resolver.Resolve(profileNames, func(name string) (string, error) {
		p, err := s.Get(name)
		if err != nil {
			return "", err
		}
		return p.Extends, nil
	})
	if err != nil {
		return fmt.Errorf("resolving profile dependencies: %w", err)
	}

	// If the resolved list differs from what the user requested, show
	// the full chain so the user knows what will be applied.
	if len(resolved) != len(profileNames) || !slicesEqual(resolved, profileNames) {
		fmt.Printf("Resolved dependency chain: %s\n", strings.Join(resolved, " → "))
	}

	// Load every resolved profile up-front so we fail fast.
	type loadedProfile struct {
		name string
		path string
	}
	profiles := make([]loadedProfile, 0, len(resolved))
	for _, name := range resolved {
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
		Strategy:    strategy,
		DryRun:      dryRun,
		IncludeDirs: includeDirs,
		ExcludeDirs: excludeDirs,
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

// parseContentDirs splits a comma-separated string of content directory
// names, validates each one, and returns the list. An empty input returns
// nil.
func parseContentDirs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var dirs []string
	for _, part := range strings.Split(raw, ",") {
		d := strings.TrimSpace(part)
		if d == "" {
			continue
		}
		if !copier.ValidContentDirs[d] {
			return nil, fmt.Errorf("invalid content directory %q; must be one of: agents, commands, skills, plugins", d)
		}
		dirs = append(dirs, d)
	}
	return dirs, nil
}

// slicesEqual reports whether two string slices have the same elements
// in the same order.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
