package gui

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/HaythmKenway/autoscout/pkg/burp"
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
	analysisModel   analysisModel
	zm              *zone.Manager
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.dashboardModel.Init(),
		m.settingsModel.Init(),
		m.targetModel.Init(),
		m.analysisModel.Init(),
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

		// Sync analysis feed from dashboard's poll
		for _, entry := range m.dashboardModel.burp_queue {
			m.analysisModel.AddEntry(entry)
		}
		m.dashboardModel.burp_queue = []string{} // Clear after syncing

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
		case "c":
			if m.activeTab == 2 {
				m.analysisModel.Clear()
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
					cmds = append(cmds, toggleBurp(m.dashboardModel.burp_status, m.dashboardModel.burp_port, m.dashboardModel.workQueue))
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
				case "Cyberpunk":
					m.theme = CyberpunkTheme
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

func (m model) vw(p float64) int {
	return int(float64(m.width) * p / 100.0)
}

func (m model) vh(p float64) int {
	return int(float64(m.height) * p / 100.0)
}

func (m *model) handleResize(w, h int) tea.Cmd {
	m.width = w
	m.height = h

	sidebarWidth := int(float64(w) * 0.25)
	if sidebarWidth < 15 { sidebarWidth = 15 }
	if sidebarWidth > 30 { sidebarWidth = 30 }

	// Calculate exact inner dimensions for content
	// rightPanelTotal = Total - Sidebar - SidebarBorder(1)
	rightPanelTotalWidth := w - sidebarWidth - 1
	
	// contentWidth (Inner) = total - Border(2) - Padding(2)
	contentWidth := rightPanelTotalWidth - 4
	contentHeight := h - 2 // Top/Bottom border

	if contentWidth < 10 { contentWidth = 10 }
	if contentHeight < 5 { contentHeight = 5 }

	subMsg := tea.WindowSizeMsg{Width: contentWidth, Height: contentHeight}
	var dCmd, sCmd, tCmd, aCmd tea.Cmd
	m.dashboardModel, dCmd = m.dashboardModel.Update(subMsg)
	m.settingsModel, sCmd = m.settingsModel.Update(subMsg)
	m.targetModel, tCmd = m.targetModel.Update(subMsg)
	m.analysisModel, aCmd = m.analysisModel.Update(subMsg)
	return tea.Batch(dCmd, sCmd, tCmd, aCmd)
}

func (m model) View() string {
	if m.width < 30 || m.height < 10 {
		return "Terminal too small"
	}

	sidebarWidth := int(float64(m.width) * 0.25)
	if sidebarWidth < 15 { sidebarWidth = 15 }
	if sidebarWidth > 30 { sidebarWidth = 30 }

	// Render Left Panel (Navigation)
	var navItems []string
	for i, item := range m.NavItems {
		style := lipgloss.NewStyle().Padding(0, 1).MarginLeft(1)
		if i == m.activeTab {
			if m.activePanel == PanelLeft {
				style = style.Background(m.theme.Accent).Foreground(lipgloss.Color("#ffffff")).Bold(true)
			} else {
				style = style.Foreground(m.theme.Accent).Bold(true)
			}
		} else {
			style = style.Foreground(m.theme.InactiveTabFG)
		}

		label := fmt.Sprintf("%s %s", m.NavIcons[i], item)
		// Width calculation: sidebarWidth - padding(2) - margin(1) = sidebarWidth - 3
		rendered := style.Width(sidebarWidth - 3).Render(label)
		navItems = append(navItems, m.zm.Mark(fmt.Sprintf("nav-%d", i), rendered))
	}

	sidebarHeader := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Padding(1, 2).
		Render("AUTOSCOUT")

	sidebarContent := lipgloss.JoinVertical(lipgloss.Left,
		sidebarHeader,
		lipgloss.JoinVertical(lipgloss.Left, navItems...),
	)
	
	leftPanel := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(m.height).
		Background(m.theme.SidebarBG).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(m.theme.BorderColor).
		Render(sidebarContent)

	// Render Right Panel (Content)
	rightPanelTotalWidth := m.width - sidebarWidth - 1
	contentWidth := rightPanelTotalWidth - 2 // Account for its own borders
	contentHeight := m.height - 2

	// Update inner content dimensions for sub-models
	m.dashboardModel.dialog.width = contentWidth - 2 // Subtract Padding(0,1)
	m.dashboardModel.dialog.height = contentHeight
	
	var content string
	switch m.activeTab {
	case 0:
		content = m.dashboardModel.View(m.zm)
	case 1:
		content = m.targetModel.View()
	case 2:
		content = m.analysisModel.View()
	case 3:
		content = m.settingsModel.View()
	}

	borderColor := m.theme.BorderColor
	if m.activePanel == PanelRight {
		borderColor = m.theme.Accent
	}

	rightPanel := lipgloss.NewStyle().
		Width(contentWidth).
		Height(contentHeight).
		Padding(0, 1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(content)

	return m.zm.Scan(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel))
}

func LoadGui(port string, workQueue chan burp.BurpRequest) error {
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
	
	leftWidth := int(float64(w) * 0.20)
	if leftWidth < 18 { leftWidth = 18 }
	if leftWidth > 30 { leftWidth = 30 }
	rightWidth := w - leftWidth - 1

	m.settingsModel = NewSettingsModel(rightWidth, h-4)
	m.dashboardModel = NewDashboardModel(rightWidth, h-4, port, workQueue)
	m.targetModel = NewTargetModel(rightWidth, h-4)
	m.analysisModel = NewAnalysisModel(rightWidth, h-4)

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

	leftWidth := int(float64(w) * 0.20)
	if leftWidth < 18 { leftWidth = 18 }
	if leftWidth > 30 { leftWidth = 30 }
	rightWidth := w - leftWidth - 1

	m.settingsModel = NewSettingsModel(rightWidth, h-4)
	m.dashboardModel = NewDashboardModel(rightWidth, h-4, "8081", nil)
	m.targetModel = NewTargetModel(rightWidth, h-4)
	m.analysisModel = NewAnalysisModel(rightWidth, h-4)
	
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
