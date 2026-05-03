package gui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type analysisModel struct {
	viewport viewport.Model
	width    int
	height   int
	theme    Theme
	entries  []string
}

func NewAnalysisModel(w, h int) analysisModel {
	vp := viewport.New(w, h-4)
	vp.SetContent("Traffic Analysis Feed waiting for Burp data...")
	return analysisModel{
		viewport: vp,
		width:    w,
		height:   h,
		theme:    ModernTheme,
		entries:  []string{},
	}
}

func (m *analysisModel) AddEntry(entry string) {
	style := lipgloss.NewStyle()
	if strings.Contains(entry, "ERRO") || strings.Contains(entry, "CRITICAL") {
		style = style.Foreground(lipgloss.Color("1")).Bold(true)
	} else if strings.Contains(entry, "ALER") || strings.Contains(entry, "AI ALERT") {
		style = style.Foreground(lipgloss.Color("3"))
	} else if strings.Contains(entry, "INFO") {
		style = style.Foreground(lipgloss.Color("6"))
	} else if strings.Contains(entry, "[AI Fleet]") {
		style = style.Foreground(lipgloss.Color("6")).Italic(true)
	} else if strings.HasPrefix(entry, "AI THINKING:") {
		style = style.Foreground(lipgloss.Color("244")).Italic(true)
	} else if strings.HasPrefix(entry, "REQ:") {
		style = style.Foreground(lipgloss.Color("2"))
	}

	// Truncate to width - 4 for border/padding
	maxWidth := m.width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}
	if len(entry) > maxWidth {
		entry = entry[:maxWidth-3] + "..."
	}

	styledEntry := style.Render(entry)

	m.entries = append(m.entries, styledEntry)
	if len(m.entries) > 500 { // Increased buffer
		m.entries = m.entries[1:]
	}
	m.viewport.SetContent(strings.Join(m.entries, "\n"))
	m.viewport.GotoBottom()
}

func (m *analysisModel) Clear() {
	m.entries = []string{}
	m.viewport.SetContent("Traffic Analysis Feed cleared. Waiting for new data...")
}

func (m analysisModel) Init() tea.Cmd {
	return nil
}

func (m analysisModel) vw(p float64) int {
	return int(float64(m.width) * p / 100.0)
}

func (m analysisModel) vh(p float64) int {
	return int(float64(m.height) * p / 100.0)
}

func (m analysisModel) Update(msg tea.Msg) (analysisModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		// Height is now managed in View() or via a calculation here
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m analysisModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Underline(true)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	// Math for stability:
	// title (1) + \n (1) + feedContainer (vpHeight + 2) + \n (1) + help (1) = TotalHeight
	// 1 + 1 + vpHeight + 2 + 1 + 1 = vpHeight + 6
	vpHeight := m.height - 6
	if vpHeight < 2 {
		vpHeight = 2
	}
	m.viewport.Height = vpHeight
	m.viewport.Width = m.width - 2

	feedContainer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(m.width).
		Height(vpHeight + 2).
		MaxHeight(vpHeight + 2).
		Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("INTERCEPTED TRAFFIC ANALYSIS"),
		feedContainer,
		helpStyle.Render(" [c] Clear Feed   [up/down] Scroll"),
	)
}
