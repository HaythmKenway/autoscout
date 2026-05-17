package gui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type quitDialogModel struct {
	width  int
	height int
	active bool
	theme  Theme
}

func NewQuitDialogModel(w, h int) quitDialogModel {
	return quitDialogModel{
		width:  w,
		height: h,
		theme:  ModernTheme,
		active: false,
	}
}

func (m quitDialogModel) Init() tea.Cmd {
	return nil
}

func (m quitDialogModel) Update(msg tea.Msg) (quitDialogModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m quitDialogModel) View() string {
	if !m.active {
		return ""
	}

	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Padding(1, 4).
		Align(lipgloss.Center)

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(m.theme.Accent).
		Padding(0, 1).
		Bold(true)

	cancelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")).
		Padding(0, 1)

	content := lipgloss.JoinVertical(lipgloss.Center,
		fmt.Sprintf("Are you sure you want to quit?\n"),
		lipgloss.JoinHorizontal(lipgloss.Center,
			buttonStyle.Render(" (y) Yes "),
			"  ",
			cancelStyle.Render(" (n) No "),
		),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		dialogBoxStyle.Render(content),
		lipgloss.WithWhitespaceBackground(lipgloss.Color("0")),
		lipgloss.WithWhitespaceChars(" "),
	)
}
