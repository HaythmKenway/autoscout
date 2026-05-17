package gui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/HaythmKenway/autoscout/pkg/burp"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
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
	fileModel       fileModel
	settingsModel   settingsModel
	dashboardModel  dashboardModel
	targetModel     targetModel
	proxyModel      proxyModel
	analysisModel   analysisModel
	pushModel       pushModel
	docsModel       docsModel
	manualOverlay   manualOverlayModel
	helpModel       helpModel
	quitDialog      quitDialogModel
	zm              *zone.Manager
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.fileModel.Init(),
		m.dashboardModel.Init(),
		m.settingsModel.Init(),
		m.targetModel.Init(),
		m.proxyModel.Init(),
		m.analysisModel.Init(),
		m.pushModel.Init(),
		m.docsModel.Init(),
		m.helpModel.Init(),
		m.quitDialog.Init(),
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

	// 1. Global System Handlers (Always processed)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		cmds = append(cmds, m.handleResize(msg.Width, msg.Height))
	case TickMsg:
		for _, entry := range m.dashboardModel.burp_queue {
			m.analysisModel.AddEntry(entry)
		}
		m.dashboardModel.burp_queue = []string{}
		
		// Always sync models in background to keep heartbeat alive and data fresh
		var dCmd, tCmd, pCmd tea.Cmd
		m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
		m.targetModel, tCmd = m.targetModel.Update(msg)
		m.proxyModel, pCmd = m.proxyModel.Update(msg)
		cmds = append(cmds, dCmd, tCmd, pCmd)
	case TriggerManualMsg:
		m.manualOverlay.active = true
		if msg.SessionID != "" {
			m.manualOverlay.SetSession(msg.SessionID)
		} else if msg.TargetURL != "" {
			m.manualOverlay.SetTarget(msg.TargetURL)
		} else {
			latest := GetLatestSessionID()
			if latest != "" {
				m.manualOverlay.SetSession(latest)
			}
		}
		return m, nil
	case TriggerCreateSessionMsg:
		m.activeTab = 2 // Proxy
		m.proxyModel.mode = modeCreateSession
		m.proxyModel.input.Focus()
		return m, textinput.Blink
	case TriggerOpenSessionMsg:
		m.activeTab = 2 // Proxy
		m.proxyModel.mode = modeSessionMenu
		m.proxyModel.sessions = burp.GetSessions()
		return m, nil
	case TriggerSaveSessionMsg:
		// Logic to save session is usually auto or manually triggered in proxy
		return m, nil
	}

	// 2. Modal/Overlay Handling (Captures Focus)
	if m.quitDialog.active {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			case "n", "N", "esc":
				m.quitDialog.active = false
				return m, nil
			}
		}
		var qCmd tea.Cmd
		m.quitDialog, qCmd = m.quitDialog.Update(msg)
		return m, qCmd
	}

	if m.helpModel.active {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "?", "esc", "q":
				m.helpModel.active = false
				return m, nil
			}
		}
		var hCmd tea.Cmd
		m.helpModel, hCmd = m.helpModel.Update(msg)
		return m, hCmd
	}

	if m.manualOverlay.active {
		var moCmd tea.Cmd
		m.manualOverlay, moCmd = m.manualOverlay.Update(msg)
		cmds = append(cmds, moCmd)
		// Modal captures all input, but we still allow system handlers like resizing
		if !msgTypeIs(msg, "WindowSizeMsg", "TriggerManualMsg") {
			return m, tea.Batch(cmds...)
		}
	}

	// 3. Main UI Interaction Handlers
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			// Sidebar Navigation
			for i := range m.NavItems {
				if m.zm.Get(fmt.Sprintf("nav-%d", i)).InBounds(msg) {
					m.activeTab = i
					m.activePanel = PanelLeft
					return m, nil
				}
			}

			// Settings Navigation (Index 5)
			if m.activeTab == 5 {
				for i := 0; i < 4; i++ {
					if m.zm.Get(fmt.Sprintf("set-cat-%d", i)).InBounds(msg) {
						m.settingsModel.activeCat = settingsCategory(i)
						m.settingsModel.focusEditor = false
					}
				}
			}

			// Dashboard Controls (Index 1)
			if m.activeTab == 1 {
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

	case tea.KeyMsg:
		// Check if any sub-model is in an input mode to prevent shortcut collision
		isInputMode := m.targetModel.adding || m.targetModel.filtering || m.proxyModel.mode == modeCreateSession
		
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !isInputMode {
				m.quitDialog.active = true
				return m, nil
			}
		case "?":
			if !isInputMode {
				m.helpModel.active = true
				m.helpModel.SetContext(m.activeTab)
				return m, nil
			}
		case "m":
			if !isInputMode {
				m.manualOverlay.active = !m.manualOverlay.active
				if m.manualOverlay.active {
					latest := GetLatestSessionID()
					if latest != "" {
						m.manualOverlay.SetSession(latest)
					}
				}
				return m, nil
			}
		case "tab":
			if !isInputMode {
				if m.activePanel == PanelLeft {
					m.activePanel = PanelRight
				} else {
					m.activePanel = PanelLeft
				}
				return m, nil
			}
		case "1", "2", "3", "4", "5", "6":
			if !isInputMode {
				m.activeTab = int(msg.String()[0] - '1')
				m.activePanel = PanelRight
			}
		case "up", "k":
			if m.activePanel == PanelLeft && !isInputMode {
				m.activeTab = max(m.activeTab-1, 0)
				return m, nil
			}
		case "down", "j":
			if m.activePanel == PanelLeft && !isInputMode {
				m.activeTab = min(m.activeTab+1, len(m.NavItems)-1)
				return m, nil
			}
		case "enter":
			if m.activePanel == PanelLeft {
				if m.activeTab == 5 && m.NavItems[m.activeTab] == "Exit" { // Backup check for exit
					m.quitDialog.active = true
				} else {
					m.activePanel = PanelRight
				}
				return m, nil
			}
		case "c":
			if m.activeTab == 4 && !isInputMode { // Analysis tab
				m.analysisModel.Clear()
			}
		case "w":
			if m.activeTab == 4 && !isInputMode { // Analysis tab
				m.analysisModel.ToggleWrap()
			}
		}
	}

	// 4. Delegate to Active Tab Model (Skip TickMsg as it's handled globally above)
	if !msgTypeIs(msg, "TickMsg") && (m.activePanel == PanelRight || msgTypeIs(msg, "MouseMsg")) {
		switch m.activeTab {
		case 0:
			var fCmd tea.Cmd
			m.fileModel, fCmd = m.fileModel.Update(msg)
			cmds = append(cmds, fCmd)
		case 1:
			var dCmd tea.Cmd
			m.dashboardModel, dCmd = m.dashboardModel.Update(msg)
			cmds = append(cmds, dCmd)
		case 2:
			var pCmd tea.Cmd
			m.proxyModel, pCmd = m.proxyModel.Update(msg)
			cmds = append(cmds, pCmd)
		case 3:
			var tCmd tea.Cmd
			m.targetModel, tCmd = m.targetModel.Update(msg)
			cmds = append(cmds, tCmd)
		case 4:
			var aCmd tea.Cmd
			m.analysisModel, aCmd = m.analysisModel.Update(msg)
			cmds = append(cmds, aCmd)
		case 5:
			var pCmd tea.Cmd
			m.pushModel, pCmd = m.pushModel.Update(msg)
			cmds = append(cmds, pCmd)
		case 6:
			var dCmd tea.Cmd
			m.docsModel, dCmd = m.docsModel.Update(msg)
			cmds = append(cmds, dCmd)
		case 7:
			var sCmd tea.Cmd
			m.settingsModel, sCmd = m.settingsModel.Update(msg)
			cmds = append(cmds, sCmd)

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
				m.proxyModel.theme = m.theme
				m.targetModel.applyStyles()
				m.proxyModel.applyStyles()
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func msgTypeIs(msg tea.Msg, types ...string) bool {
	t := fmt.Sprintf("%T", msg)
	for _, expected := range types {
		if strings.Contains(t, expected) {
			return true
		}
	}
	return false
}

func (m model) vw(p float64) int {
	return int(float64(m.width) * p / 100.0)
}

func (m model) vh(p float64) int {
	return int(float64(m.height) * p / 100.0)
}

func getSidebarWidth(w int) int {
	sidebarWidth := int(float64(w) * 0.25)
	if sidebarWidth < 15 {
		sidebarWidth = 15
	}
	if sidebarWidth > 30 {
		sidebarWidth = 30
	}
	return sidebarWidth
}

func (m *model) handleResize(w, h int) tea.Cmd {
	m.width = w
	m.height = h

	sidebarWidth := getSidebarWidth(w)
	rightPanelTotalWidth := w - sidebarWidth

	// Subtract 2 for the outer borders of the content pane
	contentWidth := rightPanelTotalWidth - 2
	// Subtract 2 for borders, 2 for top bar = 4 lines total overhead
	contentHeight := h - 4

	if contentWidth < 10 { contentWidth = 10 }
	if contentHeight < 5 { contentHeight = 5 }

	subMsg := tea.WindowSizeMsg{Width: contentWidth, Height: contentHeight}
	var fCmd, dCmd, pCmd, sCmd, tCmd, aCmd, pushCmd, docCmd tea.Cmd
	m.fileModel, fCmd = m.fileModel.Update(subMsg)
	m.dashboardModel, dCmd = m.dashboardModel.Update(subMsg)
	m.proxyModel, pCmd = m.proxyModel.Update(subMsg)
	m.settingsModel, sCmd = m.settingsModel.Update(subMsg)
	m.targetModel, tCmd = m.targetModel.Update(subMsg)
	m.analysisModel, aCmd = m.analysisModel.Update(subMsg)
	m.pushModel, pushCmd = m.pushModel.Update(subMsg)
	m.docsModel, docCmd = m.docsModel.Update(subMsg)
	
	var moCmd, hCmd, qCmd tea.Cmd
	m.manualOverlay, moCmd = m.manualOverlay.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m.helpModel, hCmd = m.helpModel.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m.quitDialog, qCmd = m.quitDialog.Update(tea.WindowSizeMsg{Width: w, Height: h})
	
	return tea.Batch(fCmd, dCmd, pCmd, sCmd, tCmd, aCmd, pushCmd, docCmd, moCmd, hCmd, qCmd)
}

func (m model) View() string {
	if m.width < 30 || m.height < 10 {
		return "Terminal too small"
	}

	if m.quitDialog.active {
		return m.zm.Scan(m.quitDialog.View())
	}

	if m.helpModel.active {
		return m.zm.Scan(m.helpModel.View())
	}

	if m.manualOverlay.active {
		return m.zm.Scan(m.manualOverlay.View())
	}

	sidebarWidth := getSidebarWidth(m.width)
	layoutHeight := m.height - 1

	// Panel Styles based on Focus
	sidebarBorderCol := m.theme.BorderColor
	contentBorderCol := m.theme.BorderColor
	if m.activePanel == PanelLeft {
		sidebarBorderCol = m.theme.Accent
	} else {
		contentBorderCol = m.theme.Accent
	}

	// 1. Sidebar Rendering
	var navItems []string
	for i, item := range m.NavItems {
		style := lipgloss.NewStyle().Padding(0, 1)
		indicator := "  "
		
		if i == m.activeTab {
			if m.activePanel == PanelLeft {
				style = style.Background(m.theme.Accent).Foreground(lipgloss.Color("#000000")).Bold(true)
				indicator = lipgloss.NewStyle().Foreground(m.theme.Accent).Render("┃ ")
			} else {
				style = style.Foreground(m.theme.Accent).Bold(true)
				indicator = "┃ "
			}
		} else {
			style = style.Foreground(m.theme.InactiveTabFG)
		}
		
		label := fmt.Sprintf("%s %s", m.NavIcons[i], item)
		// Inner width: sidebarWidth - 2 (borders) - 2 (indicator) - 2 (padding)
		renderedLabel := style.Width(sidebarWidth - 6).Render(label)
		fullRow := lipgloss.JoinHorizontal(lipgloss.Center, indicator, renderedLabel)
		navItems = append(navItems, m.zm.Mark(fmt.Sprintf("nav-%d", i), fullRow))
	}

	sidebarHeader := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Padding(0, 1).
		MarginBottom(1).
		Render(" AUTOSCOUT ")

	sidebarContent := lipgloss.JoinVertical(lipgloss.Left,
		sidebarHeader,
		lipgloss.JoinVertical(lipgloss.Left, navItems...),
	)
	
	leftPanel := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(layoutHeight - 2). // -2 for borders
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sidebarBorderCol).
		Padding(1, 0).
		Render(sidebarContent)

	// 2. Main Panel Rendering
	rightWidth := m.width - sidebarWidth
	tabTitle := strings.ToUpper(m.NavItems[m.activeTab])
	
	// Top Bar Style
	topBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(m.theme.Accent).
		Bold(true).
		Width(rightWidth - 2).
		Padding(0, 1)

	header := topBarStyle.Render(" CONTEXT: " + tabTitle)

	var content string
	switch m.activeTab {
	case 0: content = m.fileModel.View()
	case 1: content = m.dashboardModel.View(m.zm)
	case 2: content = m.proxyModel.View()
	case 3: content = m.targetModel.View()
	case 4: content = m.analysisModel.View()
	case 5: content = m.pushModel.View()
	case 6: content = m.docsModel.View()
	case 7: content = m.settingsModel.View(m.zm)
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left, header, content)

	rightPanel := lipgloss.NewStyle().
		Width(rightWidth).
		Height(layoutHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(contentBorderCol).
		Render(mainView)

	layout := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	// 3. Global Footer
	footerStyle := lipgloss.NewStyle().
		Background(contentBorderCol).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Bold(true)
	
	helpHint := lipgloss.NewStyle().
		Foreground(m.theme.InactiveTabFG).
		MarginLeft(2).
		Render("Press [?] for Help")

	footer := lipgloss.JoinHorizontal(lipgloss.Top, footerStyle.Render("AUTOSCOUT READY"), helpHint)

	return m.zm.Scan(lipgloss.JoinVertical(lipgloss.Left, layout, footer))
}

func LoadGui(port string, workQueue chan burp.BurpRequest) error {
	w, h := getTerminalSize()
	zm := zone.New() // Create a local manager
	
	m := model{
		NavItems:        []string{"File", "Dashboard", "Proxy", "Targets", "Analysis", "Settings"},
		NavIcons:        []string{"⊞", "⌂", "⇆", "⌖", "≡", "⛭"},
		activePanel:     PanelLeft,
		sidebarExpanded: true,
		theme:           ModernTheme,
		width:           w,
		height:          h,
		zm:              zm,
	}
	
	sidebarWidth := getSidebarWidth(w)
	rightWidth := w - sidebarWidth

	m.fileModel = NewFileModel(rightWidth-2, h-3, m.zm)
	m.settingsModel = NewSettingsModel(rightWidth-2, h-3)
	m.dashboardModel = NewDashboardModel(rightWidth-2, h-3, port, workQueue)
	m.proxyModel = NewProxyModel(rightWidth-2, h-3, m.zm)
	m.targetModel = NewTargetModel(rightWidth-2, h-3, m.zm)
	m.analysisModel = NewAnalysisModel(rightWidth-2, h-3)
	m.pushModel = NewPushModel(rightWidth-2, h-3)
	m.docsModel = NewDocsModel(rightWidth-2, h-3)
	m.manualOverlay = NewManualOverlayModel(w, h, m.zm)
	m.helpModel = NewHelpModel(w, h)
	m.quitDialog = NewQuitDialogModel(w, h)

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
		NavItems:        []string{"File", "Dashboard", "Proxy", "Targets", "Analysis", "Settings"},
		NavIcons:        []string{"⊞", "⌂", "⇆", "⌖", "≡", "⛭"},
		activePanel:     PanelLeft,
		sidebarExpanded: true,
		theme:           ModernTheme,
		width:           w,
		height:          h,
		zm:              zm,
	}

	sidebarWidth := getSidebarWidth(w)
	rightWidth := w - sidebarWidth

	m.fileModel = NewFileModel(rightWidth-2, h-3, m.zm)
	m.settingsModel = NewSettingsModel(rightWidth-2, h-3)
	m.dashboardModel = NewDashboardModel(rightWidth-2, h-3, "8081", nil)
	m.proxyModel = NewProxyModel(rightWidth-2, h-3, m.zm)
	m.targetModel = NewTargetModel(rightWidth-2, h-3, m.zm)
	m.analysisModel = NewAnalysisModel(rightWidth-2, h-3)
	m.pushModel = NewPushModel(rightWidth-2, h-3)
	m.docsModel = NewDocsModel(rightWidth-2, h-3)
	m.manualOverlay = NewManualOverlayModel(w, h, m.zm)
	m.helpModel = NewHelpModel(w, h)
	m.quitDialog = NewQuitDialogModel(w, h)
	
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}
