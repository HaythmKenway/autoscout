package gui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/HaythmKenway/autoscout/pkg/burp"
)

type proxyMode int

const (
	modeProxyTable proxyMode = iota
	modeSessionMenu
	modeCreateSession
)

type proxyModel struct {
	table         table.Model
	width         int
	height        int
	theme         Theme
	zm            *zone.Manager
	mode          proxyMode
	
	// Session Menu
	sessions      []string
	selectedSess  int
	
	// Create Session
	input         textinput.Model
}

func NewProxyModel(w, h int, zm *zone.Manager) proxyModel {
	t := table.New(
		table.WithFocused(true),
		table.WithHeight(h-6),
	)

	ti := textinput.New()
	ti.Placeholder = "Session Name"
	ti.CharLimit = 50
	ti.Width = 30

	m := proxyModel{
		table:  t,
		width:  w,
		height: h,
		theme:  ModernTheme,
		zm:     zm,
		mode:   modeProxyTable,
		input:  ti,
	}

	m.handleResize(w, h)
	m.applyStyles()
	return m
}

func (m *proxyModel) applyStyles() {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderColor).
		BorderBottom(true).
		Bold(true).
		Foreground(m.theme.Accent).
		Background(lipgloss.Color("236"))
	
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#000000")).
		Background(m.theme.Accent).
		Bold(true)
	
	m.table.SetStyles(s)
}

func (m proxyModel) Init() tea.Cmd {
	return nil
}

func (m proxyModel) Update(msg tea.Msg) (proxyModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.handleResize(msg.Width, msg.Height)
		m.refreshTable()

	case TickMsg:
		m.refreshTable()

	case tea.KeyMsg:
		if m.mode == modeCreateSession {
			switch msg.String() {
			case "enter":
				name := m.input.Value()
				if name != "" {
					burp.SetSession(name)
					m.mode = modeProxyTable
					m.input.Reset()
					m.refreshTable()
				}
			case "esc":
				m.mode = modeProxyTable
				m.input.Reset()
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		if m.mode == modeSessionMenu {
			switch msg.String() {
			case "up", "k":
				m.selectedSess = max(0, m.selectedSess-1)
			case "down", "j":
				m.selectedSess = min(len(m.sessions), m.selectedSess+1) // +1 for "Create New" option
			case "enter":
				if m.selectedSess == 0 {
					m.mode = modeCreateSession
					m.input.Focus()
					return m, textinput.Blink
				} else {
					burp.SetSession(m.sessions[m.selectedSess-1])
					m.mode = modeProxyTable
					m.refreshTable()
				}
			case "n":
				m.mode = modeCreateSession
				m.input.Focus()
				return m, textinput.Blink
			case "esc", "s", "f":
				m.mode = modeProxyTable
			}
			return m, nil
		}

		switch msg.String() {
		case "s", "f":
			m.mode = modeSessionMenu
			m.sessions = burp.GetSessions()
			m.selectedSess = 0
			return m, nil
		case "c":
			burp.ClearProxyHistory()
			m.refreshTable()
		case "d", "backspace", "delete":
			history := burp.GetProxyHistory()
			idx := m.table.Cursor()
			if idx >= 0 && idx < len(history) {
				burp.RemoveProxyEntry(history[idx].SessionID)
				m.refreshTable()
			}
		case "enter":
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				history := burp.GetProxyHistory()
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(history) {
					id := history[idx].SessionID
					url := history[idx].URL
					return m, func() tea.Msg {
						return TriggerManualMsg{SessionID: id, TargetURL: url}
					}
				}
			}
		}

		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *proxyModel) handleResize(w, h int) {
	m.table.SetWidth(w - 2)
	m.table.SetHeight(h - 6)

	timeW := 12
	methodW := 8
	statusW := 8
	toolW := 10
	urlW := w - timeW - methodW - statusW - toolW - 6
	if urlW < 20 { urlW = 20 }

	m.table.SetColumns([]table.Column{
		{Title: "Time", Width: timeW},
		{Title: "Method", Width: methodW},
		{Title: "Status", Width: statusW},
		{Title: "Tool", Width: toolW},
		{Title: "URL", Width: urlW},
	})
}

func (m *proxyModel) refreshTable() {
	history := burp.GetProxyHistory()
	rows := []table.Row{}

	for _, e := range history {
		methodStyle := lipgloss.NewStyle().Bold(true)
		switch e.Method {
		case "GET": methodStyle = methodStyle.Foreground(lipgloss.Color("4"))
		case "POST": methodStyle = methodStyle.Foreground(lipgloss.Color("2"))
		case "PUT", "PATCH": methodStyle = methodStyle.Foreground(lipgloss.Color("3"))
		case "DELETE": methodStyle = methodStyle.Foreground(lipgloss.Color("1"))
		}

		statusStyle := lipgloss.NewStyle()
		if e.Status == "200" { statusStyle = statusStyle.Foreground(lipgloss.Color("2")) }
		if e.Status == "404" { statusStyle = statusStyle.Foreground(lipgloss.Color("1")) }

		rows = append(rows, table.Row{
			e.Timestamp.Format("15:04:05"),
			methodStyle.Render(e.Method),
			statusStyle.Render(e.Status),
			e.Tool,
			e.URL,
		})
	}
	m.table.SetRows(rows)
}

func (m proxyModel) View() string {
	if m.mode == modeCreateSession {
		boxStyle := lipgloss.NewStyle().
			Border(m.theme.Border).
			BorderForeground(m.theme.Accent).
			Padding(1).
			Align(lipgloss.Center)

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			boxStyle.Render(fmt.Sprintf("Create New Session\n\n%s\n\n(Enter to Save, Esc to Cancel)", m.input.View())),
		)
	}

	if m.mode == modeSessionMenu {
		title := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Accent).Render("SESSION MANAGER")
		var sessViews []string
		
		// First entry: Create New
		createStyle := lipgloss.NewStyle()
		createPrefix := "  "
		if m.selectedSess == 0 {
			createPrefix = "> "
			createStyle = createStyle.Foreground(m.theme.Accent).Bold(true)
		}
		sessViews = append(sessViews, createStyle.Render(createPrefix+"[ + Create New Session ]"))
		sessViews = append(sessViews, "") // Spacer

		for i, s := range m.sessions {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i+1 == m.selectedSess {
				prefix = "> "
				style = style.Foreground(m.theme.Accent).Bold(true)
			}
			sessViews = append(sessViews, style.Render(prefix+s))
		}

		menuContent := lipgloss.JoinVertical(lipgloss.Left, 
			title, 
			"", 
			lipgloss.JoinVertical(lipgloss.Left, sessViews...),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" [enter] Select  [esc] Close"),
		)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Accent).
			Padding(1, 2).
			Background(lipgloss.Color("234"))

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, boxStyle.Render(menuContent))
	}

	// 1. Status Bar
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	statusText := "ONLINE"
	if !burp.IsRunning() {
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)
		statusText = "OFFLINE"
	}
	
	history := burp.GetProxyHistory()
	countStr := fmt.Sprintf(" | CAPTURED: %d", len(history))
	
	statusBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("7")).
		Padding(0, 1).
		Width(m.width)
	
	statusBar := statusBarStyle.Render("STATUS: " + statusStyle.Render(statusText) + countStr)

	// 2. Table or Placeholder
	var tableView string
	if len(history) == 0 {
		tableView = lipgloss.NewStyle().
			Height(m.height - 6).
			Width(m.width).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("240")).
			Render("\n\n\n( No traffic captured yet )\nEnsure Burp Server is STARTed in Dashboard")
	} else {
		tableView = m.table.View()
	}

	// 3. Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(1).
		Render(" [s] Sessions  [enter] Analyze  [d] Delete  [c] Clear ")

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, tableView, footer)
}
