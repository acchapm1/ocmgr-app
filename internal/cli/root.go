package cli

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/acchapm1/ocmgr/internal/tui"
)

// Version is set via ldflags at build time.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "ocmgr",
	Short:   "OpenCode Profile Manager",
	Long:    "ocmgr manages .opencode directory profiles.\n\nIt lets you create, snapshot, and apply reusable configuration\nprofiles for OpenCode projects so every repo starts with the\nright set of instructions, skills, and MCP servers.\n\nRun with no arguments to launch the interactive TUI.",
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := tui.NewModel()
		if err != nil {
			return fmt.Errorf("initializing TUI: %w", err)
		}
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running TUI: %w", err)
		}
		return nil
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
	// Subcommands
	rootCmd.AddCommand(initCmd, profileCmd, snapshotCmd, configCmd, syncCmd)
}
