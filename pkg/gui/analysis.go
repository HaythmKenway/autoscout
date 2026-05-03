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
	styledEntry := entry
	if strings.Contains(entry, "AI ALERT") || strings.Contains(entry, "CRITICAL") {
		styledEntry = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render(entry)
	} else if strings.Contains(entry, "AI:") || strings.Contains(entry, "modified") {
		styledEntry = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(entry)
	} else if strings.Contains(entry, "[AI Fleet]") {
		styledEntry = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true).Render(entry)
	} else if strings.HasPrefix(entry, "AI THINKING:") {
		styledEntry = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Italic(true).Render(entry)
	} else if strings.HasPrefix(entry, "REQ:") {
		styledEntry = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(entry)
	}

	m.entries = append(m.entries, styledEntry)
	if len(m.entries) > 200 {
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

func (m analysisModel) Update(msg tea.Msg) (analysisModel, tea.Cmd) {
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

func (m analysisModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Underline(true).MarginBottom(1)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).MarginTop(1)
	
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("INTERCEPTED TRAFFIC ANALYSIS"),
		m.viewport.View(),
		helpStyle.Render(" [c] Clear Feed   [up/down] Scroll"),
	)
}
