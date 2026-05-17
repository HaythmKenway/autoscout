package gui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpModel struct {
	viewport viewport.Model
	width    int
	height   int
	theme    Theme
	active   bool
}

func NewHelpModel(w, h int) helpModel {
	vp := viewport.New(w-10, h-6)
	return helpModel{
		viewport: vp,
		width:    w,
		height:   h,
		theme:    ModernTheme,
		active:   false,
	}
}

func (m helpModel) Init() tea.Cmd {
	return nil
}

func (m *helpModel) SetContext(tab int) {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Accent).Render("AUTOSCOUT GLOBAL HELP"))
	sb.WriteString("\n\n")

	// Global bindings
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Global Shortcuts:") + "\n")
	sb.WriteString("  [1-6] Switch Tabs      [tab] Cycle focus\n")
	sb.WriteString("  [?]   Toggle Help      [q]   Quit application\n")
	sb.WriteString("  [m]   Manual Invest.   [Esc] Close overlay\n\n")

	// Contextual bindings
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Context-Specific Shortcuts:") + "\n")
	switch tab {
	case 0: // File
		sb.WriteString("  [Enter] Select Option  [Up/Down] Navigate\n")
	case 1: // Dashboard
		sb.WriteString("  [s] Toggle Scanner     [b] Toggle Burp Server\n")
		sb.WriteString("  [x] Terminate Task     [Up/Down] Select Task\n")
	case 2: // Proxy
		sb.WriteString("  [s] Session Menu       [Enter] AI Analyze\n")
		sb.WriteString("  [d] Delete Entry       [c] Clear History\n")
	case 3: // Targets
		sb.WriteString("  [a] Add Target         [d] Delete Domain\n")
		sb.WriteString("  [tab] Switch Panes     [Enter] Run Tool/AI\n")
	case 4: // Analysis
		sb.WriteString("  [w] Toggle Word-Wrap   [c] Clear Feed\n")
		sb.WriteString("  [Up/Down] Scroll Feed\n")
	case 5: // Push
		sb.WriteString("  [p] Start Sync Process\n")
	case 6: // Docs
		sb.WriteString("  [Up/Down] Scroll Docs\n")
	case 7: // Settings
		sb.WriteString("  [Enter] Select Option  [Arrows] Navigate\n")
	}

	m.viewport.SetContent(sb.String())
}

func (m helpModel) Update(msg tea.Msg) (helpModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width - 10
		m.viewport.Height = msg.Height - 6
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m helpModel) View() string {
	if !m.active {
		return ""
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.theme.Accent).
		Padding(1, 2).
		Background(lipgloss.Color("0")).
		Render(m.viewport.View())

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
		lipgloss.WithWhitespaceChars(" "),
	)
}
