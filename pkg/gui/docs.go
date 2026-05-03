package gui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type docsModel struct {
	viewport viewport.Model
	width    int
	height   int
	theme    Theme
}

func NewDocsModel(w, h int) docsModel {
	vp := viewport.New(w, h-4)
	m := docsModel{
		viewport: vp,
		width:    w,
		height:   h,
		theme:    ModernTheme,
	}
	m.refreshContent()
	return m
}

func (m *docsModel) refreshContent() {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).Underline(true).MarginBottom(1)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	cmdStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("7")).Padding(0, 1)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("AUTOSCOUT v2.0 DOCUMENTATION"),
		"",
		"Welcome to the revamped Autoscout. This guide covers all the new",
		"multi-panel features and automated workflows.",
		"",
		headerStyle.Render("1. GLOBAL NAVIGATION"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [Tab]   "), "Switch focus between Navigation (Left) and Content (Right)"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [1-4]   "), "Quick-jump between tabs (Dash, Targets, Docs, Settings)"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [j/k]   "), "Navigate lists/categories when panel is focused"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [q]     "), "Quit application safely (restores your terminal)"),
		"",
		headerStyle.Render("2. SETTINGS (Press 4)"),
		"The new settings panel uses a split-view. Navigate categories on",
		"the left, press "+keyStyle.Render("[Right/Enter]")+" to edit. Press "+keyStyle.Render("[Ctrl+S]")+" to save.",
		"",
		headerStyle.Render("3. SCANNER DAEMON (-d)"),
		"Runs automated workflows defined in the database.",
		"Usage: "+cmdStyle.Render("./autoscout -d"),
	)

	m.viewport.SetContent(content)
}

func (m docsModel) Init() tea.Cmd {
	return nil
}

func (m docsModel) Update(msg tea.Msg) (docsModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m docsModel) View() string {
	return m.viewport.View()
}
