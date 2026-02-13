package cli

import (
	"fmt"
	"strings"

	"github.com/acchapm1/ocmgr/internal/updater"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [version]",
	Short: "Update ocmgr to the latest or a specific version",
	Long: `Update ocmgr to the latest version from GitHub releases.

If a version is specified, update to that specific version:
  ocmgr update          # Update to latest version
  ocmgr update v0.2.0   # Update to specific version

Note: This command only works for installations done via the curl
installer. For Homebrew installations, use: brew upgrade ocmgr
For Go installations, use: go install github.com/acchapm1/ocmgr/cmd/ocmgr@latest`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	u := updater.New(Version)

	// Detect installation method
	method := updater.DetectInstallMethod()

	// Check if we can self-update
	if method == "homebrew" {
		fmt.Println("ocmgr was installed via Homebrew.")
		fmt.Println()
		fmt.Println("To update, run:")
		fmt.Println("  brew upgrade ocmgr")
		return nil
	}

	if method == "go" {
		fmt.Println("ocmgr was installed via 'go install'.")
		fmt.Println()
		fmt.Println("To update, run:")
		fmt.Println("  go install github.com/acchapm1/ocmgr/cmd/ocmgr@latest")
		return nil
	}

	// Determine target version
	var targetVersion string
	if len(args) == 1 {
		targetVersion = args[0]
		// Ensure it starts with 'v'
		if !strings.HasPrefix(targetVersion, "v") {
			targetVersion = "v" + targetVersion
		}
	}

	// Get the release
	var release *updater.Release
	var err error

	if targetVersion != "" {
		fmt.Printf("Looking for release %s...\n", targetVersion)
		release, err = u.GetRelease(targetVersion)
		if err != nil {
			return fmt.Errorf("finding release: %w", err)
		}
	} else {
		fmt.Println("Checking for updates...")
		release, err = u.CheckForUpdate()
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		if release == nil {
			fmt.Printf("✓ ocmgr is already up to date (version %s)\n", Version)
			return nil
		}
	}

	// Show update info
	fmt.Println()
	fmt.Printf("Current version: %s\n", Version)
	fmt.Printf("Available version: %s\n", release.TagName)
	fmt.Println()

	// Perform update
	if err := u.Update(release); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	return nil
}
