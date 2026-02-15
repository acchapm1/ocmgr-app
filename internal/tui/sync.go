package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/acchapm1/ocmgr/internal/config"
	gh "github.com/acchapm1/ocmgr/internal/github"
)

// syncStatus holds state for the sync status view.
type syncStatus struct {
	lines  []string
	errMsg string
	loaded bool
}

// syncLoadedMsg is sent when sync status finishes loading.
type syncLoadedMsg struct {
	lines []string
	err   error
}

// ── Load ─────────────────────────────────────────────────────────────

func (m Model) loadSyncStatus() (tea.Model, tea.Cmd) {
	m.currentView = viewSync
	m.syncSt = &syncStatus{loaded: false}
	return m, m.fetchSyncStatus()
}

func (m Model) fetchSyncStatus() tea.Cmd {
	storeDir := m.store.Dir
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return syncLoadedMsg{err: fmt.Errorf("loading config: %w", err)}
		}
		if cfg.GitHub.Repo == "" {
			return syncLoadedMsg{err: fmt.Errorf("github.repo is not configured; run: ocmgr config set github.repo <owner/repo>")}
		}

		status, err := gh.Status(storeDir, cfg.GitHub.Repo, cfg.GitHub.Auth)
		if err != nil {
			return syncLoadedMsg{err: err}
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("Repository: %s", cfg.GitHub.Repo))
		lines = append(lines, "")

		total := len(status.InSync) + len(status.Modified) + len(status.LocalOnly) + len(status.RemoteOnly)
		if total == 0 {
			lines = append(lines, "No profiles found locally or remotely.")
		} else {
			lines = append(lines, fmt.Sprintf("%-20s %s", "PROFILE", "STATUS"))
			lines = append(lines, strings.Repeat("─", 45))

			for _, n := range status.InSync {
				lines = append(lines, fmt.Sprintf("%-20s ✓ in sync", n))
			}
			for _, n := range status.Modified {
				lines = append(lines, fmt.Sprintf("%-20s ~ modified", n))
			}
			for _, n := range status.LocalOnly {
				lines = append(lines, fmt.Sprintf("%-20s ● local only", n))
			}
			for _, n := range status.RemoteOnly {
				lines = append(lines, fmt.Sprintf("%-20s ○ remote only", n))
			}
		}

		return syncLoadedMsg{lines: lines}
	}
}

// ── Update ───────────────────────────────────────────────────────────

func (m Model) updateSync(msg tea.Msg) (tea.Model, tea.Cmd) {
	ss := m.syncSt
	if ss == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case syncLoadedMsg:
		ss.loaded = true
		if msg.err != nil {
			ss.errMsg = msg.err.Error()
		} else {
			ss.lines = msg.lines
		}
		return m, nil

	case tea.KeyMsg:
		if ss.loaded {
			if key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))) {
				m.currentView = viewMenu
				m.syncSt = nil
				return m, nil
			}
		}
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────

func (m Model) viewSync() string {
	ss := m.syncSt
	if ss == nil {
		return ""
	}

	var b strings.Builder

	title := SubtitleStyle.Render("Sync Status")
	b.WriteString(title)
	b.WriteString("\n\n")

	if !ss.loaded {
		b.WriteString(StatusStyle.Render("⏳ Loading sync status..."))
		return b.String()
	}

	if ss.errMsg != "" {
		b.WriteString(ErrorStyle.Render("✗ " + ss.errMsg))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("esc: back"))
		return b.String()
	}

	for _, line := range ss.lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(MutedStyle.Render("  Use CLI for push/pull: ocmgr sync push|pull <name>"))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("esc: back • q: back"))
	return b.String()
}
