package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/acchapm1/ocmgr/internal/copier"
	"github.com/acchapm1/ocmgr/internal/profile"
)

// snapStep tracks the current step in the snapshot wizard.
type snapStep int

const (
	snapStepName snapStep = iota
	snapStepDir
	snapStepMeta
	snapStepPreview
	snapStepRunning
	snapStepDone
)

// snapshotWizard holds state for the snapshot wizard flow.
type snapshotWizard struct {
	step      snapStep
	nameInput textinput.Model
	dirInput  textinput.Model
	descInput textinput.Model
	tagsInput textinput.Model
	metaField int // 0=desc, 1=tags
	name      string
	sourceDir string
	preview   []string
	errMsg    string
	resultMsg string
}

// snapDoneMsg is sent when the snapshot completes.
type snapDoneMsg struct {
	msg string
	err error
}

// ── Load ─────────────────────────────────────────────────────────────

func (m Model) loadSnapshotWizard() (tea.Model, tea.Cmd) {
	m.currentView = viewSnapshot

	ni := textinput.New()
	ni.Placeholder = "my-profile"
	ni.CharLimit = 64
	ni.Width = 40
	ni.Focus()

	di := textinput.New()
	di.Placeholder = "."
	di.CharLimit = 256
	di.Width = 50

	desc := textinput.New()
	desc.Placeholder = "A brief description"
	desc.CharLimit = 200
	desc.Width = 50

	tags := textinput.New()
	tags.Placeholder = "go, backend, api"
	tags.CharLimit = 200
	tags.Width = 50

	m.snapWiz = &snapshotWizard{
		step:      snapStepName,
		nameInput: ni,
		dirInput:  di,
		descInput: desc,
		tagsInput: tags,
	}

	return m, ni.Focus()
}

// ── Update ───────────────────────────────────────────────────────────

func (m Model) updateSnapshot(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.snapWiz
	if wiz == nil {
		return m, nil
	}

	switch wiz.step {
	case snapStepName:
		return m.updateSnapName(msg)
	case snapStepDir:
		return m.updateSnapDir(msg)
	case snapStepMeta:
		return m.updateSnapMeta(msg)
	case snapStepPreview:
		return m.updateSnapPreview(msg)
	case snapStepRunning:
		switch msg := msg.(type) {
		case snapDoneMsg:
			wiz.step = snapStepDone
			if msg.err != nil {
				wiz.errMsg = msg.err.Error()
			} else {
				wiz.resultMsg = msg.msg
			}
			return m, nil
		}
		return m, nil
	case snapStepDone:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter", "esc", "q"))) {
				m.currentView = viewMenu
				m.snapWiz = nil
				return m, nil
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateSnapName(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.snapWiz
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			name := strings.TrimSpace(wiz.nameInput.Value())
			if name == "" {
				wiz.errMsg = "Name is required"
				return m, nil
			}
			if m.store.Exists(name) {
				wiz.errMsg = fmt.Sprintf("Profile %q already exists", name)
				return m, nil
			}
			wiz.name = name
			wiz.errMsg = ""
			wiz.step = snapStepDir
			wiz.dirInput.Focus()
			return m, wiz.dirInput.Focus()
		}
	}
	var cmd tea.Cmd
	wiz.nameInput, cmd = wiz.nameInput.Update(msg)
	return m, cmd
}

func (m Model) updateSnapDir(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.snapWiz
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			dir := strings.TrimSpace(wiz.dirInput.Value())
			if dir == "" {
				dir = "."
			}
			absDir, err := filepath.Abs(dir)
			if err != nil {
				wiz.errMsg = fmt.Sprintf("Invalid directory: %v", err)
				return m, nil
			}
			openCodeDir := filepath.Join(absDir, ".opencode")
			if _, err := os.Stat(openCodeDir); os.IsNotExist(err) {
				wiz.errMsg = fmt.Sprintf("No .opencode directory found in %s", absDir)
				return m, nil
			}
			wiz.sourceDir = absDir
			wiz.errMsg = ""
			wiz.step = snapStepMeta
			wiz.descInput.Focus()
			return m, wiz.descInput.Focus()
		}
	}
	var cmd tea.Cmd
	wiz.dirInput, cmd = wiz.dirInput.Update(msg)
	return m, cmd
}

func (m Model) updateSnapMeta(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.snapWiz
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("tab"))) {
			if wiz.metaField == 0 {
				wiz.metaField = 1
				wiz.descInput.Blur()
				wiz.tagsInput.Focus()
				return m, wiz.tagsInput.Focus()
			}
			wiz.metaField = 0
			wiz.tagsInput.Blur()
			wiz.descInput.Focus()
			return m, wiz.descInput.Focus()
		}
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			// Build preview
			openCodeDir := filepath.Join(wiz.sourceDir, ".opencode")
			wiz.preview = []string{
				fmt.Sprintf("Name:   %s", wiz.name),
				fmt.Sprintf("Source: %s", openCodeDir),
			}
			desc := strings.TrimSpace(wiz.descInput.Value())
			if desc != "" {
				wiz.preview = append(wiz.preview, fmt.Sprintf("Desc:   %s", desc))
			}
			tags := strings.TrimSpace(wiz.tagsInput.Value())
			if tags != "" {
				wiz.preview = append(wiz.preview, fmt.Sprintf("Tags:   %s", tags))
			}
			wiz.preview = append(wiz.preview, "")

			// Count files
			for _, dir := range profile.ContentDirs() {
				srcDir := filepath.Join(openCodeDir, dir)
				entries, err := os.ReadDir(srcDir)
				if err != nil {
					continue
				}
				count := 0
				for _, e := range entries {
					if !e.IsDir() {
						count++
					}
				}
				if count > 0 {
					wiz.preview = append(wiz.preview, fmt.Sprintf("  %s/: %d files", dir, count))
				}
			}

			wiz.step = snapStepPreview
			return m, nil
		}
	}

	var cmd tea.Cmd
	if wiz.metaField == 0 {
		wiz.descInput, cmd = wiz.descInput.Update(msg)
	} else {
		wiz.tagsInput, cmd = wiz.tagsInput.Update(msg)
	}
	return m, cmd
}

func (m Model) updateSnapPreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	wiz := m.snapWiz
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter", "y"))):
			wiz.step = snapStepRunning
			return m, m.runSnapshot()
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "esc"))):
			m.currentView = viewMenu
			m.snapWiz = nil
			return m, nil
		}
	}
	return m, nil
}

func (m Model) runSnapshot() tea.Cmd {
	wiz := m.snapWiz
	name := wiz.name
	sourceDir := wiz.sourceDir
	desc := strings.TrimSpace(wiz.descInput.Value())
	tagsRaw := strings.TrimSpace(wiz.tagsInput.Value())
	storeDir := m.store.Dir

	return func() tea.Msg {
		openCodeDir := filepath.Join(sourceDir, ".opencode")

		p, err := profile.ScaffoldProfile(storeDir, name)
		if err != nil {
			return snapDoneMsg{err: fmt.Errorf("creating profile: %w", err)}
		}

		success := false
		defer func() {
			if !success {
				_ = os.RemoveAll(p.Path)
			}
		}()

		totalFiles := 0
		for _, dir := range profile.ContentDirs() {
			srcDir := filepath.Join(openCodeDir, dir)
			if _, err := os.Stat(srcDir); os.IsNotExist(err) {
				continue
			}
			err := filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if info.IsDir() {
					return nil
				}
				// Skip infrastructure files
				switch info.Name() {
				case "node_modules", "package.json", "bun.lock", ".gitignore":
					return nil
				}
				rel, err := filepath.Rel(srcDir, path)
				if err != nil {
					return err
				}
				dst := filepath.Join(p.Path, dir, rel)
				if err := copier.CopyFile(path, dst); err != nil {
					return err
				}
				totalFiles++
				return nil
			})
			if err != nil {
				return snapDoneMsg{err: fmt.Errorf("copying %s: %w", dir, err)}
			}
		}

		p.Description = desc
		if tagsRaw != "" {
			for _, t := range strings.Split(tagsRaw, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					p.Tags = append(p.Tags, t)
				}
			}
		}
		if err := profile.SaveProfile(p); err != nil {
			return snapDoneMsg{err: fmt.Errorf("saving metadata: %w", err)}
		}

		success = true
		return snapDoneMsg{msg: fmt.Sprintf("Snapshot '%s' created with %d files", name, totalFiles)}
	}
}

// ── View ─────────────────────────────────────────────────────────────

func (m Model) viewSnapshot() string {
	wiz := m.snapWiz
	if wiz == nil {
		return ""
	}

	switch wiz.step {
	case snapStepName:
		return m.viewSnapName()
	case snapStepDir:
		return m.viewSnapDir()
	case snapStepMeta:
		return m.viewSnapMeta()
	case snapStepPreview:
		return m.viewSnapPreview()
	case snapStepRunning:
		return StatusStyle.Render("⏳ Creating snapshot...")
	case snapStepDone:
		return m.viewSnapDone()
	}
	return ""
}

func (m Model) viewSnapName() string {
	wiz := m.snapWiz
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Snapshot — Profile Name"))
	b.WriteString("\n\n")
	b.WriteString("  Name: ")
	b.WriteString(wiz.nameInput.View())
	if wiz.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(ErrorStyle.Render("  ✗ " + wiz.errMsg))
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("enter: continue • esc: cancel"))
	return b.String()
}

func (m Model) viewSnapDir() string {
	wiz := m.snapWiz
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Snapshot — Source Directory"))
	b.WriteString("\n\n")
	b.WriteString("  Name: ")
	b.WriteString(DetailValueStyle.Render(wiz.name))
	b.WriteString("\n")
	b.WriteString("  Source (default: current): ")
	b.WriteString(wiz.dirInput.View())
	if wiz.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(ErrorStyle.Render("  ✗ " + wiz.errMsg))
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("enter: continue • esc: cancel"))
	return b.String()
}

func (m Model) viewSnapMeta() string {
	wiz := m.snapWiz
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Snapshot — Metadata"))
	b.WriteString("\n\n")
	b.WriteString("  Description: ")
	b.WriteString(wiz.descInput.View())
	b.WriteString("\n")
	b.WriteString("  Tags:        ")
	b.WriteString(wiz.tagsInput.View())
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("tab: next field • enter: continue • esc: cancel"))
	return b.String()
}

func (m Model) viewSnapPreview() string {
	wiz := m.snapWiz
	var b strings.Builder
	b.WriteString(SubtitleStyle.Render("Snapshot — Confirm"))
	b.WriteString("\n\n")
	for _, line := range wiz.preview {
		b.WriteString("  ")
		b.WriteString(MutedStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(StatusStyle.Render("Create this snapshot?"))
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("y/enter: create • n/esc: cancel"))
	return b.String()
}

func (m Model) viewSnapDone() string {
	wiz := m.snapWiz
	var b strings.Builder
	if wiz.errMsg != "" {
		b.WriteString(ErrorStyle.Render("✗ " + wiz.errMsg))
	} else {
		b.WriteString(StatusStyle.Render("✓ " + wiz.resultMsg))
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("press any key to return to menu"))
	return b.String()
}
