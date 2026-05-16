package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type analysisModel struct {
	viewport   viewport.Model
	width      int
	height     int
	theme      Theme
	rawEntries []string
	wrap       bool
}

func NewAnalysisModel(w, h int) analysisModel {
	vp := viewport.New(w, h-4)
	vp.SetContent("Traffic Analysis Feed waiting for Burp data...")
	return analysisModel{
		viewport:   vp,
		width:      w,
		height:     h,
		theme:      ModernTheme,
		rawEntries: []string{},
		wrap:       false,
	}
}

func (m *analysisModel) AddEntry(entry string) {
	m.rawEntries = append(m.rawEntries, entry)
	if len(m.rawEntries) > 500 {
		m.rawEntries = m.rawEntries[1:]
	}
	m.rebuildViewport()
}

func (m *analysisModel) rebuildViewport() {
	var styledEntries []string
	maxWidth := m.width - 4
	if maxWidth < 10 {
		maxWidth = 10
	}

	for _, entry := range m.rawEntries {
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

		var processed string
		if m.wrap {
			style = style.Width(maxWidth)
			processed = style.Render(entry)
		} else {
			display := entry
			if len(display) > maxWidth {
				display = display[:maxWidth-3] + "..."
			}
			processed = style.Render(display)
		}
		styledEntries = append(styledEntries, processed)
	}

	m.viewport.SetContent(strings.Join(styledEntries, "\n"))
	m.viewport.GotoBottom()
}

func (m *analysisModel) ToggleWrap() {
	m.wrap = !m.wrap
	m.rebuildViewport()
}

func (m *analysisModel) Clear() {
	m.rawEntries = []string{}
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
		m.rebuildViewport()
	}
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m analysisModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Underline(true).PaddingLeft(1)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(1)

	vpHeight := m.height - 2
	if vpHeight < 2 {
		vpHeight = 2
	}
	m.viewport.Height = vpHeight
	m.viewport.Width = m.width

	wrapText := "Off"
	if m.wrap {
		wrapText = "On"
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("INTERCEPTED TRAFFIC ANALYSIS"),
		m.viewport.View(),
		helpStyle.Render(fmt.Sprintf(" [c] Clear Feed   [w] Wrap: %s   [up/down] Scroll", wrapText)),
	)
}
