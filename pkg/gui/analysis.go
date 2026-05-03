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
	m.entries = append(m.entries, entry)
	if len(m.entries) > 100 {
		m.entries = m.entries[1:]
	}
	m.viewport.SetContent(strings.Join(m.entries, "\n"))
	m.viewport.GotoBottom()
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
	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("INTERCEPTED TRAFFIC ANALYSIS"),
		m.viewport.View(),
	)
}
