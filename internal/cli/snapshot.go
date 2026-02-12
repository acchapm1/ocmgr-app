package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/acchapm1/ocmgr-app/internal/copier"
	"github.com/acchapm1/ocmgr-app/internal/profile"
	"github.com/acchapm1/ocmgr-app/internal/store"
	"github.com/spf13/cobra"
)

// skipFiles is the set of infrastructure files that should not be copied
// when snapshotting a .opencode directory into a profile.
var skipFiles = map[string]bool{
	"node_modules": true,
	"package.json": true,
	"bun.lock":     true,
	".gitignore":   true,
}

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <name> [source-dir]",
	Short: "Capture current .opencode directory as a profile",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		sourceDir := "."
		if len(args) > 1 {
			sourceDir = args[1]
		}

		// Resolve to absolute path.
		sourceDir, err := filepath.Abs(sourceDir)
		if err != nil {
			return fmt.Errorf("resolving source directory: %w", err)
		}

		openCodeDir := filepath.Join(sourceDir, ".opencode")
		if _, err := os.Stat(openCodeDir); os.IsNotExist(err) {
			return fmt.Errorf("no .opencode directory found in %s", sourceDir)
		}

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		if s.Exists(name) {
			return fmt.Errorf("profile %q already exists; delete it first with 'ocmgr profile delete %s' or choose a different name", name, name)
		}

		p, err := profile.ScaffoldProfile(s.Dir, name)
		if err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}

		// Clean up the scaffolded directory if we fail partway through.
		success := false
		defer func() {
			if !success {
				_ = os.RemoveAll(p.Path)
			}
		}()

		// Copy files from each content directory.
		counts := map[string]int{
			"agents":   0,
			"commands": 0,
			"skills":   0,
			"plugins":  0,
		}

		for _, dir := range profile.ContentDirs() {
			srcDir := filepath.Join(openCodeDir, dir)
			if _, err := os.Stat(srcDir); os.IsNotExist(err) {
				continue
			}

			err := filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}

				// Skip infrastructure files and directories.
				if skipFiles[info.Name()] {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				if info.IsDir() {
					return nil
				}

				rel, err := filepath.Rel(srcDir, path)
				if err != nil {
					return fmt.Errorf("computing relative path: %w", err)
				}

				dst := filepath.Join(p.Path, dir, rel)
				if err := copier.CopyFile(path, dst); err != nil {
					return fmt.Errorf("copying %s: %w", rel, err)
				}

				counts[dir]++
				return nil
			})
			if err != nil {
				return fmt.Errorf("walking %s: %w", dir, err)
			}
		}

		// Prompt for description and tags.
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Description []: ")
		description, _ := reader.ReadString('\n')
		description = strings.TrimSpace(description)

		fmt.Print("Tags (comma-separated) []: ")
		tagsInput, _ := reader.ReadString('\n')
		tagsInput = strings.TrimSpace(tagsInput)

		var tags []string
		if tagsInput != "" {
			for _, t := range strings.Split(tagsInput, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		// Update and save profile metadata.
		p.Description = description
		p.Tags = tags
		if err := profile.SaveProfile(p); err != nil {
			return fmt.Errorf("saving profile metadata: %w", err)
		}

		success = true
		fmt.Printf("Snapshot '%s' created with %d agents, %d commands, %d skills, %d plugins\n",
			name, counts["agents"], counts["commands"], counts["skills"], counts["plugins"])

		return nil
	},
}
