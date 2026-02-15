package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/acchapm1/ocmgr/internal/profile"
)

// editorStep tracks the current step in the profile editor.
type editorStep int

const (
	editorStepFileList editorStep = iota
	editorStepEditing
)

// profileEditor holds state for the profile editor view.
type profileEditor struct {
	step     editorStep
	profile  *profile.Profile
	fileList list.Model
	files    []string
	errMsg   string
}

// fileItem implements list.Item for the file browser.
type fileItem struct {
	path string
}

func (i fileItem) Title() string       { return i.path }
func (i fileItem) Description() string { return "" }
func (i fileItem) FilterValue() string { return i.path }

// editorDoneMsg is sent when the external editor exits.
type editorDoneMsg struct{ err error }

// ── Load ─────────────────────────────────────────────────────────────

func (m Model) loadEditor(p *profile.Profile) (tea.Model, tea.Cmd) {
	contents, err := profile.ListContents(p)
	if err != nil {
		m.errMsg = fmt.Sprintf("listing contents: %v", err)
		return m, nil
	}

	var files []string
	files = append(files, contents.Agents...)
	files = append(files, contents.Commands...)
	files = append(files, contents.Skills...)
	files = append(files, contents.Plugins...)

	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = fileItem{path: f}
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ColorPrimary).BorderLeftForeground(ColorPrimary)
	delegate.ShowDescription = false

	fl := list.New(items, delegate, m.width, m.height-4)
	fl.Title = fmt.Sprintf("Edit: %s", p.Name)
	fl.SetShowStatusBar(true)
	fl.SetFilteringEnabled(true)

	m.currentView = viewEditor
	m.editor = &profileEditor{
		step:     editorStepFileList,
		profile:  p,
		fileList: fl,
		files:    files,
	}

	return m, nil
}

// ── Update ───────────────────────────────────────────────────────────

func (m Model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	ed := m.editor
	if ed == nil {
		return m, nil
	}

	switch ed.step {
	case editorStepFileList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if ed.fileList.FilterState() == list.Filtering {
				break
			}
			if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
				selected, ok := ed.fileList.SelectedItem().(fileItem)
				if !ok {
					return m, nil
				}
				filePath := selected.path
				absPath := fmt.Sprintf("%s/%s", ed.profile.Path, filePath)

				// Check file exists
				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					ed.errMsg = fmt.Sprintf("file not found: %s", absPath)
					return m, nil
				}

				ed.step = editorStepEditing
				// Launch editor
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "nvim"
				}
				c := exec.Command(editor, absPath)
				return m, tea.ExecProcess(c, func(err error) tea.Msg {
					return editorDoneMsg{err: err}
				})
			}
		}

		var cmd tea.Cmd
		ed.fileList, cmd = ed.fileList.Update(msg)
		return m, cmd

	case editorStepEditing:
		switch msg.(type) {
		case editorDoneMsg:
			ed.step = editorStepFileList
			return m, nil
		}
	}

	return m, nil
}

// ── View ─────────────────────────────────────────────────────────────

func (m Model) viewEditor() string {
	ed := m.editor
	if ed == nil {
		return ""
	}

	var b strings.Builder

	switch ed.step {
	case editorStepFileList:
		b.WriteString(ed.fileList.View())
		if ed.errMsg != "" {
			b.WriteString("\n")
			b.WriteString(ErrorStyle.Render("✗ " + ed.errMsg))
		}
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("enter: edit in $EDITOR • /: filter • esc: back"))

	case editorStepEditing:
		b.WriteString(StatusStyle.Render("Editing file... (return to TUI when editor closes)"))
	}

	return b.String()
}
