package gui

import (
	"fmt"
	"os"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	zone "github.com/lrstanley/bubblezone"
)

type Panel int

const (
	PanelLeft Panel = iota
	PanelRight
)

type model struct {
	NavItems        []string
	NavIcons        []string
	activeTab       int
	activePanel     Panel
	sidebarExpanded bool
	width           int
	height          int
	theme           Theme
	settingsModel   settingsModel
	dashboardModel  dashboardModel
	targetModel     targetModel
	docsModel       docsModel
	zm              *zone.Manager
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.dashboardModel.Init(),
		m.settingsModel.Init(),
		m.targetModel.Init(),
		m.docsModel.Init(),
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
	case TickMsg:
		var dCmd tea.Cmd
		m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
		cmds = append(cmds, dCmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.activePanel == PanelLeft {
				m.activePanel = PanelRight
			} else {
				m.activePanel = PanelLeft
			}
			return m, nil
		case "1", "2", "3", "4":
			if !m.targetModel.adding {
				m.activeTab = int(msg.String()[0] - '1')
				m.activePanel = PanelRight
			}
		case "5":
			return m, tea.Quit
		case "up", "k":
			if m.activePanel == PanelLeft && !m.targetModel.adding {
				m.activeTab = max(m.activeTab-1, 0)
			}
		case "down", "j":
			if m.activePanel == PanelLeft && !m.targetModel.adding {
				m.activeTab = min(m.activeTab+1, len(m.NavItems)-1)
			}
		case "enter":
			if m.activeTab == 4 {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cmds = append(cmds, m.handleResize(msg.Width, msg.Height))

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			for i := range m.NavItems {
				if m.zm.Get(fmt.Sprintf("nav-%d", i)).InBounds(msg) {
					if i == 4 { // Exit
						return m, tea.Quit
					}
					m.activeTab = i
					m.activePanel = PanelLeft
					return m, nil
				}
			}
			// Dashboard Toggle Button
			if m.activeTab == 0 {
				if m.zm.Get(m.dashboardModel.dialog.id+"ToggleStart").InBounds(msg) {
					m.dashboardModel.app_status = !m.dashboardModel.app_status
					cmds = append(cmds, runScheduler(m.dashboardModel.app_status))
				}
				if m.zm.Get(m.dashboardModel.dialog.id+"ToggleBurp").InBounds(msg) {
					m.dashboardModel.burp_status = !m.dashboardModel.burp_status
					cmds = append(cmds, toggleBurp(m.dashboardModel.burp_status))
				}
			}
		}
	}

	// Delegate updates to components based on focus
	if m.activePanel == PanelRight || m.activeTab == 1 { // Always update target model if it's adding
		switch m.activeTab {
		case 0:
			var dCmd tea.Cmd
			m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
			cmds = append(cmds, dCmd)
		case 1:
			var tCmd tea.Cmd
			m.targetModel, tCmd = m.targetModel.Update(msg)
			cmds = append(cmds, tCmd)
		case 3:
			var sCmd tea.Cmd
			m.settingsModel, sCmd = m.settingsModel.Update(msg)
			cmds = append(cmds, sCmd)

			// Sync theme from settings
			newThemeName := m.settingsModel.userSettings.Theme
			if newThemeName != m.theme.Name {
				switch newThemeName {
				case "Neon":
					m.theme = NeonTheme
				case "Matrix":
					m.theme = MatrixTheme
				default:
					m.theme = ModernTheme
				}
				m.dashboardModel.theme = m.theme
				m.targetModel.theme = m.theme
				m.targetModel.applyStyles()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleResize(w, h int) tea.Cmd {
	leftWidth := int(float64(w) * 0.25)
	if leftWidth < 15 {
		leftWidth = 15
	}
	rightWidth := w - leftWidth - 1

	subMsg := tea.WindowSizeMsg{Width: rightWidth, Height: h - 2}
	var dCmd, sCmd, tCmd, docCmd tea.Cmd
	m.dashboardModel, dCmd = m.dashboardModel.Update(subMsg)
	m.settingsModel, sCmd = m.settingsModel.Update(subMsg)
	m.targetModel, tCmd = m.targetModel.Update(subMsg)
	m.docsModel, docCmd = m.docsModel.Update(subMsg)
	return tea.Batch(dCmd, sCmd, tCmd, docCmd)
}

func (m model) View() string {
	if m.width < 20 || m.height < 10 {
		return "Terminal too small for UI 2.0"
	}

	leftWidth := int(float64(m.width) * 0.25)
	if leftWidth < 15 {
		leftWidth = 15
	}
	rightWidth := m.width - leftWidth - 1

	// Render Left Panel (Navigation)
	var navItems []string
	for i, item := range m.NavItems {
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.activeTab {
			if m.activePanel == PanelLeft {
				style = style.Background(m.theme.Accent).Foreground(lipgloss.Color("#ffffff")).Bold(true)
			} else {
				style = style.Background(m.theme.InactiveTabBG).Foreground(m.theme.Foreground)
			}
		} else {
			style = style.Foreground(m.theme.InactiveTabFG)
		}
		
		// Use the manager from the model
		label := fmt.Sprintf("%s %s", m.NavIcons[i], item)
		navItems = append(navItems, m.zm.Mark(fmt.Sprintf("nav-%d", i), style.Width(leftWidth - 2).Render(label)))
	}

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height - 2).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(m.theme.BorderColor).
		Render(lipgloss.JoinVertical(lipgloss.Left, navItems...))

	// Render Right Panel (Content)
	content := ""
	switch m.activeTab {
	case 0:
		content = m.dashboardModel.View(m.zm)
	case 1:
		content = m.targetModel.View()
	case 2:
		content = m.docsModel.View()
	case 3:
		content = m.settingsModel.View()
	}

	rightPanelStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height - 2).
		Padding(0, 1)
	
	if m.activePanel == PanelRight {
		rightPanelStyle = rightPanelStyle.Border(lipgloss.NormalBorder()).BorderForeground(m.theme.Accent)
	}

	rightPanel := rightPanelStyle.Render(content)

	return m.zm.Scan(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel))
}

func LoadGui() error {
	w, h := getTerminalSize()
	zm := zone.New() // Create a local manager
	
	m := model{
		NavItems:        []string{"Dashboard", "Targets", "Analysis", "Settings", "Exit"},
		NavIcons:        []string{"⌂", "➤", "≡", "⚙", "⏻"},
		activePanel:     PanelLeft,
		sidebarExpanded: true,
		theme:           ModernTheme,
		width:           w,
		height:          h,
		zm:              zm,
	}
	
	leftWidth := int(float64(w) * 0.25)
	if leftWidth < 15 { leftWidth = 15 }
	rightWidth := w - leftWidth - 1

	m.settingsModel = NewSettingsModel(rightWidth, h-2)
	m.dashboardModel = NewDashboardModel(rightWidth, h-2)
	m.targetModel = NewTargetModel(rightWidth, h-2)
	m.docsModel = NewDocsModel(rightWidth, h-2)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := p.Run()
	return err
}

func SShHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, active := s.Pty()
	w, h := 80, 24
	if active {
		w = pty.Window.Width
		h = pty.Window.Height
	}

	zm := zone.New()
	m := model{
		NavItems:        []string{"Dashboard", "Targets", "Analysis", "Settings", "Exit"},
		NavIcons:        []string{"⌂", "➤", "≡", "⚙", "⏻"},
		activePanel:     PanelLeft,
		sidebarExpanded: true,
		theme:           ModernTheme,
		width:           w,
		height:          h,
		zm:              zm,
	}

	leftWidth := int(float64(w) * 0.25)
	if leftWidth < 15 { leftWidth = 15 }
	rightWidth := w - leftWidth - 1

	m.settingsModel = NewSettingsModel(rightWidth, h-2)
	m.dashboardModel = NewDashboardModel(rightWidth, h-2)
	m.targetModel = NewTargetModel(rightWidth, h-2)
	m.docsModel = NewDocsModel(rightWidth, h-2)
	
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
