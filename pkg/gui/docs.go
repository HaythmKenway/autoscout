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
		"multi-panel features and the AI Proxy integrations.",
		"",
		headerStyle.Render("1. GLOBAL NAVIGATION"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [Tab]   "), "Switch focus between Navigation (Left) and Content (Right)"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [1-4]   "), "Quick-jump between tabs (Dash, Targets, Docs, Settings)"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [j/k]   "), "Navigate lists/categories when panel is focused"),
		lipgloss.JoinHorizontal(lipgloss.Top, keyStyle.Render(" [q]     "), "Quit application safely (restores your terminal)"),
		"",
		headerStyle.Render("2. AI MITM PROXY (-p)"),
		"The proxy intercepts HTTPS traffic and sends it to the AI Fleet.",
		"Usage: "+cmdStyle.Render("./autoscout -p -port 8081"),
		"",
		"To generate the CA certificate initially:",
		"Command: "+cmdStyle.Render("./autoscout -gencert"),
		"",
		"IMPORTANT: For HTTPS interception, you must import the Root CA:",
		"Path: "+cmdStyle.Render("~/ca.crt"),
		"Import this into your Browser's 'Authorities' certificate store.",
		"",
		headerStyle.Render("3. BURP SUITE INTEGRATION"),
		"To use Burp Suite -> Autoscout workflow:",
		"1. Start proxy: "+cmdStyle.Render("./autoscout -p -port 8081"),
		"2. In Burp: Settings -> Network -> Connections -> Upstream Proxy",
		"3. Add Destination: "+keyStyle.Render("*")+" | Host: "+keyStyle.Render("127.0.0.1")+" | Port: "+keyStyle.Render("8081"),
		"",
		headerStyle.Render("4. SETTINGS (Press 4)"),
		"The new settings panel uses a split-view. Navigate categories on",
		"the left, press "+keyStyle.Render("[Right/Enter]")+" to edit. Press "+keyStyle.Render("[Ctrl+S]")+" to save.",
		"",
		headerStyle.Render("5. SCANNER DAEMON (-d)"),
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
