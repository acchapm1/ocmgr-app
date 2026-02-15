package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/acchapm1/ocmgr/internal/copier"
	"github.com/acchapm1/ocmgr/internal/resolver"
)

// initStep tracks the current step in the init wizard.
type initStep int

const (
	initStepProfile initStep = iota
	initStepDir
	initStepPreview
	initStepRunning
	initStepDone
)

// ── Init Wizard state (stored in Model) ──────────────────────────────

// initWizard holds all state for the init wizard flow.
type initWizard struct {
	step          initStep
	profileList   list.Model
	dirInput      textinput.Model
	selectedNames []string
	resolvedNames []string
	previewLines  []string
	resultLines   []string
	errMsg        string
	copyCount     int
	skipCount     int
}

// ── Messages ─────────────────────────────────────────────────────────

type initCopyDoneMsg struct {
	copied  int
	skipped int
	errors  []string
}

type initCopyErrMsg struct {
	err error
}

// ── Load ─────────────────────────────────────────────────────────────

func (m Model) loadInitWizard() (tea.Model, tea.Cmd) {
	profiles, err := m.store.List()
	if err != nil {
		m.errMsg = fmt.Sprintf("loading profiles: %v", err)
		return m, nil
	}

	if len(profiles) == 0 {
		m.statusMsg = "No profiles found. Create one first with: ocmgr profile create <name>"
		return m, nil
	}

	m.profiles = profiles
	m.currentView = viewInit
	m.statusMsg = ""
	m.errMsg = ""

	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		items[i] = profileItem{profile: p}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ColorPrimary).BorderLeftForeground(ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(ColorSecondary).BorderLeftForeground(ColorPrimary)

	pl := list.New(items, delegate, m.width, m.height-4)
	pl.Title = "Select Profile"
	pl.SetShowStatusBar(true)
	pl.SetFilteringEnabled(true)

	ti := textinput.New()
	ti.Placeholder = "."
	ti.CharLimit = 256
	ti.Width = 50

	m.initWiz = &initWizard{
		step:        initStepProfile,
		profileList: pl,
		dirInput:    ti,
	}

	return m, nil
}

// ── Update ───────────────────────────────────────────────────────────

func (m Model) updateInit(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.initWiz
	if wiz == nil {
		return m, nil
	}

	switch wiz.step {
	case initStepProfile:
		return m.updateInitProfile(msg)
	case initStepDir:
		return m.updateInitDir(msg)
	case initStepPreview:
		return m.updateInitPreview(msg)
	case initStepRunning:
		// Wait for result
		switch msg := msg.(type) {
		case initCopyDoneMsg:
			wiz.step = initStepDone
			wiz.copyCount = msg.copied
			wiz.skipCount = msg.skipped
			wiz.resultLines = msg.errors
			return m, nil
		case initCopyErrMsg:
			wiz.step = initStepDone
			wiz.errMsg = msg.err.Error()
			return m, nil
		}
		return m, nil
	case initStepDone:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter", "esc", "q"))) {
				m.currentView = viewMenu
				m.initWiz = nil
				return m, nil
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateInitProfile(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.initWiz

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if wiz.profileList.FilterState() == list.Filtering {
			break
		}
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			selected, ok := wiz.profileList.SelectedItem().(profileItem)
			if !ok {
				return m, nil
			}
			wiz.selectedNames = []string{selected.profile.Name}

			// Resolve extends chain
			resolved, err := resolver.Resolve(wiz.selectedNames, func(name string) (string, error) {
				p, err := m.store.Get(name)
				if err != nil {
					return "", err
				}
				return p.Extends, nil
			})
			if err != nil {
				wiz.errMsg = fmt.Sprintf("resolving profile: %v", err)
				return m, nil
			}
			wiz.resolvedNames = resolved

			// Move to directory step
			wiz.step = initStepDir
			wiz.dirInput.Focus()
			return m, wiz.dirInput.Focus()
		}
	}

	var cmd tea.Cmd
	wiz.profileList, cmd = wiz.profileList.Update(msg)
	return m, cmd
}

func (m Model) updateInitDir(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.initWiz

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			dir := strings.TrimSpace(wiz.dirInput.Value())
			if dir == "" {
				dir = "."
			}

			absDir, err := filepath.Abs(dir)
			if err != nil {
				wiz.errMsg = fmt.Sprintf("invalid directory: %v", err)
				return m, nil
			}

			targetOpencode := filepath.Join(absDir, ".opencode")

			// Build preview
			wiz.previewLines = []string{}
			wiz.previewLines = append(wiz.previewLines,
				fmt.Sprintf("Target: %s", targetOpencode))
			wiz.previewLines = append(wiz.previewLines, "")

			if len(wiz.resolvedNames) > 1 {
				wiz.previewLines = append(wiz.previewLines,
					fmt.Sprintf("Profiles (resolved): %s", strings.Join(wiz.resolvedNames, " → ")))
			} else {
				wiz.previewLines = append(wiz.previewLines,
					fmt.Sprintf("Profile: %s", wiz.resolvedNames[0]))
			}
			wiz.previewLines = append(wiz.previewLines, "")

			// Dry-run to preview files
			for _, name := range wiz.resolvedNames {
				p, err := m.store.Get(name)
				if err != nil {
					continue
				}
				result, _ := copier.CopyProfile(p.Path, targetOpencode, copier.Options{
					Strategy: copier.StrategyOverwrite,
					DryRun:   true,
				})
				if result != nil {
					wiz.previewLines = append(wiz.previewLines,
						fmt.Sprintf("  %s: %d files", name, len(result.Copied)))
					for _, f := range result.Copied {
						rel, _ := filepath.Rel(targetOpencode, f)
						wiz.previewLines = append(wiz.previewLines,
							fmt.Sprintf("    %s", rel))
					}
				}
			}

			wiz.step = initStepPreview
			return m, nil
		}
	}

	var cmd tea.Cmd
	wiz.dirInput, cmd = wiz.dirInput.Update(msg)
	return m, cmd
}

func (m Model) updateInitPreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.initWiz

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "y"))):
			// Run the copy
			wiz.step = initStepRunning
			return m, m.runInitCopy()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "esc"))):
			m.currentView = viewMenu
			m.initWiz = nil
			return m, nil
		}
	}

	return m, nil
}

// runInitCopy returns a tea.Cmd that performs the actual copy.
func (m Model) runInitCopy() tea.Cmd {
	wiz := m.initWiz
	dir := strings.TrimSpace(wiz.dirInput.Value())
	if dir == "" {
		dir = "."
	}
	absDir, _ := filepath.Abs(dir)
	targetOpencode := filepath.Join(absDir, ".opencode")
	resolvedNames := wiz.resolvedNames
	st := m.store

	return func() tea.Msg {
		totalCopied := 0
		totalSkipped := 0
		var allErrors []string

		for _, name := range resolvedNames {
			p, err := st.Get(name)
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: %v", name, err))
				continue
			}

			result, err := copier.CopyProfile(p.Path, targetOpencode, copier.Options{
				Strategy: copier.StrategyOverwrite,
			})
			if err != nil {
				return initCopyErrMsg{err: err}
			}
			totalCopied += len(result.Copied)
			totalSkipped += len(result.Skipped)
			allErrors = append(allErrors, result.Errors...)
		}

		return initCopyDoneMsg{
			copied:  totalCopied,
			skipped: totalSkipped,
			errors:  allErrors,
		}
	}
}

// ── View ─────────────────────────────────────────────────────────────

func (m Model) viewInit() string {
	wiz := m.initWiz
	if wiz == nil {
		return ""
	}

	switch wiz.step {
	case initStepProfile:
		return m.viewInitProfile()
	case initStepDir:
		return m.viewInitDir()
	case initStepPreview:
		return m.viewInitPreview()
	case initStepRunning:
		return m.viewInitRunning()
	case initStepDone:
		return m.viewInitDone()
	}
	return ""
}

func (m Model) viewInitProfile() string {
	wiz := m.initWiz
	var b strings.Builder
	b.WriteString(wiz.profileList.View())
	b.WriteString("\n")
	if wiz.errMsg != "" {
		b.WriteString(ErrorStyle.Render("✗ " + wiz.errMsg))
		b.WriteString("\n")
	}
	b.WriteString(HelpStyle.Render("enter: select • /: filter • esc: cancel"))
	return b.String()
}

func (m Model) viewInitDir() string {
	wiz := m.initWiz
	var b strings.Builder

	title := SubtitleStyle.Render("Target Directory")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString("  Profile: ")
	b.WriteString(DetailValueStyle.Render(strings.Join(wiz.resolvedNames, " → ")))
	b.WriteString("\n\n")
	b.WriteString("  Directory (default: current): ")
	b.WriteString(wiz.dirInput.View())
	b.WriteString("\n")

	if wiz.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render("✗ " + wiz.errMsg))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("enter: continue • esc: cancel"))
	return b.String()
}

func (m Model) viewInitPreview() string {
	wiz := m.initWiz
	var b strings.Builder

	title := SubtitleStyle.Render("Preview Changes")
	b.WriteString(title)
	b.WriteString("\n\n")

	for _, line := range wiz.previewLines {
		b.WriteString("  ")
		b.WriteString(MutedStyle.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("Apply these changes?"))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("y/enter: apply • n/esc: cancel"))
	return b.String()
}

func (m Model) viewInitRunning() string {
	return StatusStyle.Render("⏳ Copying files...")
}

func (m Model) viewInitDone() string {
	wiz := m.initWiz
	var b strings.Builder

	if wiz.errMsg != "" {
		b.WriteString(ErrorStyle.Render("✗ " + wiz.errMsg))
	} else {
		b.WriteString(StatusStyle.Render(fmt.Sprintf("✓ Copied %d files", wiz.copyCount)))
		if wiz.skipCount > 0 {
			b.WriteString("\n")
			b.WriteString(MutedStyle.Render(fmt.Sprintf("  Skipped %d existing files", wiz.skipCount)))
		}
		if len(wiz.resultLines) > 0 {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render(fmt.Sprintf("  %d errors:", len(wiz.resultLines))))
			for _, e := range wiz.resultLines {
				b.WriteString("\n    ")
				b.WriteString(MutedStyle.Render(e))
			}
		}

		dir := strings.TrimSpace(wiz.dirInput.Value())
		if dir == "" {
			dir = "."
		}
		absDir, _ := filepath.Abs(dir)
		targetOpencode := filepath.Join(absDir, ".opencode")

		// Check if opencode.json exists
		if _, err := os.Stat(filepath.Join(targetOpencode, "opencode.json")); os.IsNotExist(err) {
			b.WriteString("\n\n")
			b.WriteString(MutedStyle.Render("  Tip: Run 'ocmgr init -p <profile> .' to also configure plugins and MCPs"))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("press any key to return to menu"))
	return b.String()
}
