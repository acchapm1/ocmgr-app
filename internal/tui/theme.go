// Package tui provides the lipgloss theme for the ocmgr TUI.
package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // violet
	ColorSecondary = lipgloss.Color("#06B6D4") // cyan
	ColorSuccess   = lipgloss.Color("#10B981") // green
	ColorWarning   = lipgloss.Color("#F59E0B") // amber
	ColorError     = lipgloss.Color("#EF4444") // red
	ColorMuted     = lipgloss.Color("#6B7280") // gray
	ColorText      = lipgloss.Color("#E5E7EB") // light gray
	ColorBg        = lipgloss.Color("#1F2937") // dark gray
)

// Styles
var (
	// TitleStyle is used for the app title bar.
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			PaddingLeft(1)

	// SubtitleStyle is used for section subtitles.
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			PaddingLeft(1)

	// MenuItemStyle is the default style for menu items.
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// MenuSelectedStyle is the style for the currently selected menu item.
	MenuSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(1)

	// HelpStyle is used for the help bar at the bottom.
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			PaddingLeft(1).
			PaddingTop(1)

	// StatusStyle is used for status messages.
	StatusStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			PaddingLeft(1)

	// ErrorStyle is used for error messages.
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			PaddingLeft(1)

	// MutedStyle is used for secondary/muted text.
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// BorderStyle is used for bordered panels.
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// DetailLabelStyle is used for labels in detail views.
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true).
				Width(14)

	// DetailValueStyle is used for values in detail views.
	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ColorText)
)
