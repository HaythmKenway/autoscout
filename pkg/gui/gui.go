package gui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	zone "github.com/lrstanley/bubblezone"
)

type model struct {
	Tabs           []string
	activeTab      int
	width          int
	height         int
	settingsModel  settingsModel
	dashboardModel dashboardModel
	targetModel    targetModel
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.dashboardModel.Init(),
		m.settingsModel.Init(),
		m.targetModel.Init(),
	)
}

func getTerminalSize() (width int, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return width, height
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// CRITICAL FIX: Handle TickMsg explicitly so the loop never dies
	case TickMsg:
		var dCmd tea.Cmd
		m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
		cmds = append(cmds, dCmd)

	case tea.KeyMsg:
		if m.activeTab == 1 && m.targetModel.adding {
			var tCmd tea.Cmd
			m.targetModel, tCmd = m.targetModel.Update(msg)
			cmds = append(cmds, tCmd)
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "right", "tab":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
			return m, nil
		case "left", "shift+tab":
			m.activeTab = max(m.activeTab-1, 0)
			return m, nil
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			for i := range m.Tabs {
				if zone.Get(fmt.Sprintf("tab-%d", i)).InBounds(msg) {
					m.activeTab = i
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		tabAreaHeight := 4
		contentHeight := m.height - tabAreaHeight
		if contentHeight < 10 {
			contentHeight = 10
		}

		subMsg := tea.WindowSizeMsg{Width: m.width, Height: contentHeight}

		var dCmd, sCmd, tCmd tea.Cmd
		m.dashboardModel, dCmd = m.dashboardModel.Update(subMsg)
		m.settingsModel, sCmd = m.settingsModel.Update(subMsg)
		m.targetModel, tCmd = m.targetModel.Update(subMsg)
		cmds = append(cmds, dCmd, sCmd, tCmd)
	}

	// Pass other messages to active tab
	switch m.activeTab {
	case 0:
		// We already handled TickMsg, but dashboard might need keys/mouse
		if _, ok := msg.(TickMsg); !ok {
			var dCmd tea.Cmd
			m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
			cmds = append(cmds, dCmd)
		}
	case 1:
		var tCmd tea.Cmd
		m.targetModel, tCmd = m.targetModel.Update(msg)
		cmds = append(cmds, tCmd)
	case 3:
		var sCmd tea.Cmd
		m.settingsModel, sCmd = m.settingsModel.Update(msg)
		cmds = append(cmds, sCmd)
	}

	return m, tea.Batch(cmds...)
}

var (
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	inactiveTabStyle = lipgloss.NewStyle().
				Border(tabBorderWithBottom("┴", "─", "┴"), true).
				BorderForeground(highlightColor).
				Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Border(tabBorderWithBottom("┘", " ", "└"), true).
			BorderForeground(highlightColor).
			Padding(0, 1)

	windowStyle = lipgloss.NewStyle().Padding(0, 1)
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m model) View() string {
	var renderedTabs []string
	tabRowWidth := 0

	for i, tab := range m.Tabs {
		style := inactiveTabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		tabStr := zone.Mark(fmt.Sprintf("tab-%d", i), style.Render(tab))
		renderedTabs = append(renderedTabs, tabStr)
		tabRowWidth += lipgloss.Width(tabStr)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	if tabRowWidth < m.width {
		gap := m.width - tabRowWidth - 4
		if gap > 0 {
			row += lipgloss.NewStyle().
				Foreground(highlightColor).
				Render(strings.Repeat("─", gap))
		}
	}

	content := ""
	switch m.activeTab {
	case 0:
		content = m.dashboardModel.View()
	case 1:
		content = m.targetModel.View()
	case 2:
		content = "\n  ≡ Analysis Page (Coming Soon)"
	case 3:
		content = m.settingsModel.View()
	}

	contentView := windowStyle.Width(m.width).Render(content)
	return zone.Scan(lipgloss.JoinVertical(lipgloss.Left, row, contentView))
}

func LoadGui() error {
	w, h := getTerminalSize()
	zone.NewGlobal()
	m := model{
		Tabs:           []string{"⌂ Dashboard", "➤ Targets", "≡ Analysis", "☰ Settings"},
		width:          w,
		height:         h,
		settingsModel:  NewSettingsModel(w, h),
		dashboardModel: NewDashboardModel(w, h),
		targetModel:    NewTargetModel(w, h),
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running program: %w", err)
	}
	return nil
}

func SShHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	w, h := 80, 24
	if active {
		w = pty.Window.Width
		h = pty.Window.Height
	}

	zone.NewGlobal()
	m := model{
		Tabs:           []string{"⌂ Dashboard", "➤ Targets", "≡ Analysis", "☰ Settings"},
		width:          w,
		height:         h,
		settingsModel:  NewSettingsModel(w, h),
		dashboardModel: NewDashboardModel(w, h),
		targetModel:    NewTargetModel(w, h),
	}
	return m, []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
