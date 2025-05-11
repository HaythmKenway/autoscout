package gui

import (
 tea "github.com/charmbracelet/bubbletea"
 	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1).
			MarginRight(2)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)
)

type dashboardModel struct{
	dialog dialog
	app_status bool
}

type dialog struct{
	id string
	height int 
	width  int 
	active string
	question string
}

func (m dashboardModel) Init() tea.Cmd {
	return nil
}

func NewDashboardModel(w int,h int) dashboardModel {
	return dashboardModel{dialog:dialog{width: 30,height: 20}}
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dialog.width = msg.Width
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}

		if zone.Get(m.dialog.id + "ToggleStart").InBounds(msg) {
			m.app_status=!m.app_status	
		}

		return m, nil
	}
	return m, nil

}

func (m dashboardModel) View() string {
	var startButton,question string	
	if m.app_status {
		startButton=activeButtonStyle.Render("Stop")
		question = lipgloss.NewStyle().Width(27).Align(lipgloss.Center).Render("Stop Services")
	} else {
		startButton=buttonStyle.Render("Start")
		question = lipgloss.NewStyle().Width(27).Align(lipgloss.Center).Render("Start Services")
	}
	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
			zone.Mark(m.dialog.id+"ToggleStart",startButton),
	)
	return dialogBoxStyle.Render(lipgloss.JoinVertical(lipgloss.Center, question, buttons))
}
