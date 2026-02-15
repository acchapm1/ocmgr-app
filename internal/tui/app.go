// Package tui implements the interactive terminal UI for ocmgr.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/acchapm1/ocmgr/internal/profile"
	"github.com/acchapm1/ocmgr/internal/store"
)

// view represents which screen is currently displayed.
type view int

const (
	viewMenu view = iota
	viewProfiles
	viewProfileDetail
	viewInit
	viewEditor
	viewSync
	viewSnapshot
)

// menuItem implements list.Item for the main menu.
type menuItem struct {
	title string
	desc  string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

// profileItem implements list.Item for the profile browser.
type profileItem struct {
	profile *profile.Profile
}

func (i profileItem) Title() string       { return i.profile.Name }
func (i profileItem) Description() string { return i.profile.Description }
func (i profileItem) FilterValue() string { return i.profile.Name }

// Model is the top-level Bubble Tea model for the ocmgr TUI.
type Model struct {
	// Current view
	currentView view

	// Main menu
	menuList list.Model

	// Profile browser
	profileList     list.Model
	profiles        []*profile.Profile
	selectedProfile *profile.Profile
	profileDetail   string

	// Init wizard
	initWiz *initWizard

	// Profile editor
	editor *profileEditor

	// Sync status
	syncSt *syncStatus

	// Snapshot wizard
	snapWiz *snapshotWizard

	// Dimensions
	width  int
	height int

	// Status / errors
	statusMsg string
	errMsg    string

	// Store
	store *store.Store
}

// NewModel creates and returns a new TUI model.
func NewModel() (Model, error) {
	s, err := store.NewStore()
	if err != nil {
		return Model{}, fmt.Errorf("opening store: %w", err)
	}

	m := Model{
		currentView: viewMenu,
		store:       s,
	}

	// Build main menu
	menuItems := []list.Item{
		menuItem{title: "Init", desc: "Initialize .opencode/ from a profile"},
		menuItem{title: "Profiles", desc: "Browse and manage profiles"},
		menuItem{title: "Sync", desc: "Synchronize profiles with GitHub"},
		menuItem{title: "Snapshot", desc: "Capture .opencode/ as a new profile"},
		menuItem{title: "Config", desc: "View and edit configuration"},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ColorPrimary).BorderLeftForeground(ColorPrimary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(ColorSecondary).BorderLeftForeground(ColorPrimary)

	m.menuList = list.New(menuItems, delegate, 40, 14)
	m.menuList.Title = "ocmgr"
	m.menuList.SetShowStatusBar(false)
	m.menuList.SetFilteringEnabled(false)
	m.menuList.Styles.Title = TitleStyle.Copy().
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	return m, nil
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menuList.SetSize(msg.Width, msg.Height-2)
		if m.profileList.Items() != nil {
			m.profileList.SetSize(msg.Width, msg.Height-2)
		}
		return m, nil

	case tea.KeyMsg:
		// Global keys
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
			if m.currentView == viewMenu {
				return m, tea.Quit
			}
			// Don't intercept 'q' in text inputs or filtering
			if m.isTextInputActive() {
				break
			}
			return m.goBack(), nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			if m.isTextInputActive() {
				break
			}
			if m.currentView != viewMenu {
				return m.goBack(), nil
			}
		}
	}

	// Delegate to current view
	switch m.currentView {
	case viewMenu:
		return m.updateMenu(msg)
	case viewProfiles:
		return m.updateProfiles(msg)
	case viewProfileDetail:
		return m.updateProfileDetail(msg)
	case viewInit:
		return m.updateInit(msg)
	case viewEditor:
		return m.updateEditor(msg)
	case viewSync:
		return m.updateSync(msg)
	case viewSnapshot:
		return m.updateSnapshot(msg)
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.currentView {
	case viewMenu:
		return m.viewMenu()
	case viewProfiles:
		return m.viewProfiles()
	case viewProfileDetail:
		return m.viewProfileDetail()
	case viewInit:
		return m.viewInit()
	case viewEditor:
		return m.viewEditor()
	case viewSync:
		return m.viewSync()
	case viewSnapshot:
		return m.viewSnapshot()
	}
	return ""
}

// isTextInputActive returns true if a text input or list filter is focused.
func (m Model) isTextInputActive() bool {
	switch m.currentView {
	case viewProfiles:
		return m.profileList.FilterState() == list.Filtering
	case viewInit:
		if m.initWiz != nil {
			switch m.initWiz.step {
			case initStepDir:
				return true
			case initStepProfile:
				return m.initWiz.profileList.FilterState() == list.Filtering
			}
		}
	case viewEditor:
		if m.editor != nil {
			return m.editor.fileList.FilterState() == list.Filtering
		}
	case viewSnapshot:
		if m.snapWiz != nil {
			switch m.snapWiz.step {
			case snapStepName, snapStepDir, snapStepMeta:
				return true
			}
		}
	}
	return false
}

// goBack returns to the previous view.
func (m Model) goBack() Model {
	switch m.currentView {
	case viewProfileDetail:
		m.currentView = viewProfiles
		m.selectedProfile = nil
		m.profileDetail = ""
	case viewProfiles:
		m.currentView = viewMenu
	case viewInit:
		m.currentView = viewMenu
		m.initWiz = nil
	case viewEditor:
		m.currentView = viewProfiles
		m.editor = nil
	case viewSync:
		m.currentView = viewMenu
		m.syncSt = nil
	case viewSnapshot:
		m.currentView = viewMenu
		m.snapWiz = nil
	}
	return m
}

// ── Menu ─────────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			selected, ok := m.menuList.SelectedItem().(menuItem)
			if !ok {
				return m, nil
			}
			m.statusMsg = ""
			m.errMsg = ""
			switch selected.title {
			case "Init":
				return m.loadInitWizard()
			case "Profiles":
				return m.loadProfiles()
			case "Sync":
				return m.loadSyncStatus()
			case "Snapshot":
				return m.loadSnapshotWizard()
			case "Config":
				m.statusMsg = "Use CLI: ocmgr config show|set|init"
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.menuList, cmd = m.menuList.Update(msg)
	return m, cmd
}

func (m Model) viewMenu() string {
	var b strings.Builder
	b.WriteString(m.menuList.View())

	if m.statusMsg != "" {
		b.WriteString("\n")
		b.WriteString(StatusStyle.Render("ℹ " + m.statusMsg))
	}
	if m.errMsg != "" {
		b.WriteString("\n")
		b.WriteString(ErrorStyle.Render("✗ " + m.errMsg))
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("enter: select • q: quit"))

	return b.String()
}

// ── Profile Browser ──────────────────────────────────────────────────

func (m Model) loadProfiles() (tea.Model, tea.Cmd) {
	profiles, err := m.store.List()
	if err != nil {
		m.errMsg = fmt.Sprintf("loading profiles: %v", err)
		return m, nil
	}

	m.profiles = profiles
	m.currentView = viewProfiles
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

	m.profileList = list.New(items, delegate, m.width, m.height-2)
	m.profileList.Title = "Profiles"
	m.profileList.SetShowStatusBar(true)
	m.profileList.SetFilteringEnabled(true)
	m.profileList.Styles.Title = TitleStyle.Copy().
		Background(ColorSecondary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1)

	return m, nil
}

func (m Model) updateProfiles(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't intercept keys when filtering
		if m.profileList.FilterState() == list.Filtering {
			break
		}
		if key.Matches(msg, key.NewBinding(key.WithKeys("enter"))) {
			selected, ok := m.profileList.SelectedItem().(profileItem)
			if !ok {
				return m, nil
			}
			return m.loadProfileDetail(selected.profile)
		}
		if key.Matches(msg, key.NewBinding(key.WithKeys("e"))) {
			selected, ok := m.profileList.SelectedItem().(profileItem)
			if !ok {
				return m, nil
			}
			return m.loadEditor(selected.profile)
		}
	}

	var cmd tea.Cmd
	m.profileList, cmd = m.profileList.Update(msg)
	return m, cmd
}

func (m Model) viewProfiles() string {
	var b strings.Builder
	b.WriteString(m.profileList.View())
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("enter: view • e: edit • /: filter • esc: back"))
	return b.String()
}

// ── Profile Detail ───────────────────────────────────────────────────

func (m Model) loadProfileDetail(p *profile.Profile) (tea.Model, tea.Cmd) {
	m.selectedProfile = p
	m.currentView = viewProfileDetail

	contents, err := profile.ListContents(p)
	if err != nil {
		m.errMsg = fmt.Sprintf("listing contents: %v", err)
		return m, nil
	}

	var b strings.Builder

	// Header
	title := TitleStyle.Copy().
		Background(ColorPrimary).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render(p.Name)
	b.WriteString(title)
	b.WriteString("\n\n")

	// Metadata
	if p.Description != "" {
		b.WriteString(DetailLabelStyle.Render("Description"))
		b.WriteString(DetailValueStyle.Render(p.Description))
		b.WriteString("\n")
	}
	if p.Extends != "" {
		b.WriteString(DetailLabelStyle.Render("Extends"))
		b.WriteString(DetailValueStyle.Render(p.Extends))
		b.WriteString("\n")
	}
	if len(p.Tags) > 0 {
		b.WriteString(DetailLabelStyle.Render("Tags"))
		b.WriteString(DetailValueStyle.Render(strings.Join(p.Tags, ", ")))
		b.WriteString("\n")
	}
	if p.Version != "" {
		b.WriteString(DetailLabelStyle.Render("Version"))
		b.WriteString(DetailValueStyle.Render(p.Version))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// File tree
	writeSection := func(label string, files []string) {
		if len(files) == 0 {
			return
		}
		b.WriteString(SubtitleStyle.Render(fmt.Sprintf("%s (%d)", label, len(files))))
		b.WriteString("\n")
		for _, f := range files {
			b.WriteString(MutedStyle.Render("    " + f))
			b.WriteString("\n")
		}
	}

	writeSection("Agents", contents.Agents)
	writeSection("Commands", contents.Commands)
	writeSection("Skills", contents.Skills)
	writeSection("Plugins", contents.Plugins)

	m.profileDetail = b.String()
	return m, nil
}

func (m Model) updateProfileDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, key.NewBinding(key.WithKeys("e"))) {
			if m.selectedProfile != nil {
				return m.loadEditor(m.selectedProfile)
			}
		}
	}
	return m, nil
}

func (m Model) viewProfileDetail() string {
	var b strings.Builder
	b.WriteString(m.profileDetail)
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("e: edit files • esc: back • q: back"))
	return b.String()
}
