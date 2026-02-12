package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "ocmgr",
	Short:   "OpenCode Profile Manager",
	Long:    "ocmgr manages .opencode directory profiles.\n\nIt lets you create, snapshot, and apply reusable configuration\nprofiles for OpenCode projects so every repo starts with the\nright set of instructions, skills, and MCP servers.",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Run 'ocmgr --help' for usage information.")
		fmt.Println("TUI mode coming soon — use subcommands for now.")
	},
}

// Execute runs the root command and exits on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Persistent flags (none yet — structure ready for future additions).

	// Subcommands
	rootCmd.AddCommand(initCmd, profileCmd, snapshotCmd, configCmd, syncCmd)
}
