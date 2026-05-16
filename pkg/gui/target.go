package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/HaythmKenway/autoscout/internal/db"
)

type targetPane int

const (
	PaneDomains targetPane = iota
	PaneURLs
	PaneTools
)

type targetModel struct {
	domainTable    table.Model
	urlTable       table.Model
	input          textinput.Model
	adding         bool
	width          int
	height         int
	theme          Theme
	err            error
	activePane     targetPane
	selectedTarget string
	zm             *zone.Manager

	// Session tracking
	urlToSession map[string]string

	// Tool Sidebar
	toolList     []string
	selectedTool int
	aiMode       bool

	// Layout widths
	leftWidth    int
	middleWidth  int
	sidebarWidth int
}

func NewTargetModel(w, h int, zm *zone.Manager) targetModel {
	// Domain Table (Left)
	dt := table.New(table.WithFocused(true))

	// URL Table (Middle)
	ut := table.New(table.WithFocused(false))

	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30

	m := targetModel{
		domainTable:  dt,
		urlTable:     ut,
		input:        ti,
		adding:       false,
		width:        w,
		height:       h,
		theme:        ModernTheme,
		activePane:   PaneDomains,
		zm:           zm,
		toolList:     []string{"DalFox", "SQLMap", "Nuclei", "Katana", "FFUF", "Arjun", "GoSpider", "AI Mode"},
		aiMode:       true,
		urlToSession: make(map[string]string),
	}

	m.handleResize(w, h)
	m.refreshTargets()
	m.applyStyles()
	return m
}

func (m *targetModel) applyStyles() {
	s := table.DefaultStyles()
	// Disable all internal borders to prevent double-border glitches
	s.Header = s.Header.
		BorderStyle(lipgloss.HiddenBorder()).
		BorderBottom(false).
		Bold(true).
		Foreground(m.theme.Accent)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffff")).
		Background(m.theme.Highlight).
		Bold(true)
	
	m.domainTable.SetStyles(s)
	m.urlTable.SetStyles(s)
}

func (m targetModel) Init() tea.Cmd {
	return nil
}

func (m targetModel) vw(p float64) int {
	return int(float64(m.width) * p / 100.0)
}

func (m targetModel) vh(p float64) int {
	return int(float64(m.height) * p / 100.0)
}

func (m targetModel) Update(msg tea.Msg) (targetModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.handleResize(msg.Width, msg.Height)

	case TickMsg:
		m.refreshTargets()
		m.refreshURLs()

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if m.zm.Get("target-ai-btn").InBounds(msg) {
				selectedURLRow := m.urlTable.SelectedRow()
				var sid, target string
				if len(selectedURLRow) > 0 {
					target = selectedURLRow[0]
					sid = m.urlToSession[target]
				} else {
					target = m.selectedTarget
					sid = m.urlToSession[target]
				}

				return m, func() tea.Msg {
					return TriggerManualMsg{SessionID: sid, TargetURL: target}
				}
			}
		}

	case tea.KeyMsg:
		if m.adding {
			switch msg.String() {
			case "enter":
				target := m.input.Value()
				if target != "" {
					if _, err := db.AddTarget(target); err != nil {
						m.err = err
					} else {
						m.input.Reset()
						m.adding = false
						m.refreshTargets()
					}
				}
			case "esc":
				m.adding = false
				m.input.Reset()
			}
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "tab":
			m.cyclePane()
		case "a":
			m.adding = true
			m.input.Focus()
			return m, textinput.Blink
		case "d", "backspace", "delete":
			if m.activePane == PaneDomains {
				selected := m.domainTable.SelectedRow()
				if len(selected) > 0 {
					target := strings.TrimPrefix(selected[0], "  └── ")
					db.RemoveTarget(target)
					m.refreshTargets()
				}
			}
		case "up", "down", "j", "k":
			if m.activePane == PaneDomains {
				m.domainTable, cmd = m.domainTable.Update(msg)
				cmds = append(cmds, cmd)
				
				selected := m.domainTable.SelectedRow()
				if len(selected) > 0 {
					newTarget := strings.TrimPrefix(selected[0], "  └── ")
					if newTarget != m.selectedTarget {
						m.selectedTarget = newTarget
						m.refreshURLs()
					}
				}
				return m, tea.Batch(cmds...)
			} else if m.activePane == PaneTools {
				if msg.String() == "up" || msg.String() == "k" {
					m.selectedTool = max(0, m.selectedTool-1)
				} else {
					m.selectedTool = min(len(m.toolList)-1, m.selectedTool+1)
				}
				return m, nil
			}
		case "enter":
			if m.activePane == PaneTools {
				toolName := m.toolList[m.selectedTool]
				if toolName == "AI Mode" {
					m.aiMode = !m.aiMode
				} else {
					// Execute tool logic
					m.executeSelectedTool()
				}
				return m, nil
			}
		}
	}

	if m.activePane == PaneDomains {
		m.domainTable, cmd = m.domainTable.Update(msg)
	} else if m.activePane == PaneURLs {
		m.urlTable, cmd = m.urlTable.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *targetModel) cyclePane() {
	m.domainTable.Blur()
	m.urlTable.Blur()
	
	switch m.activePane {
	case PaneDomains:
		m.activePane = PaneURLs
		m.urlTable.Focus()
	case PaneURLs:
		m.activePane = PaneTools
	case PaneTools:
		m.activePane = PaneDomains
		m.domainTable.Focus()
	}
}

func (m *targetModel) executeSelectedTool() {
	if m.selectedTarget == "" { return }
}

func (m *targetModel) handleResize(w, h int) {
	if w < 80 {
		m.sidebarWidth = 18
		m.leftWidth = int(float64(w-m.sidebarWidth) * 0.4)
		m.middleWidth = w - m.sidebarWidth - m.leftWidth
	} else {
		m.sidebarWidth = 24
		m.leftWidth = int(float64(w-m.sidebarWidth) * 0.35)
		if m.leftWidth < 25 { m.leftWidth = 25 }
		m.middleWidth = w - m.leftWidth - m.sidebarWidth
	}

	if m.middleWidth < 15 { m.middleWidth = 15 }
	
	// Inner content width = PaneWidth - 4 (borders + padding)
	dtWidth := m.leftWidth - 4
	if dtWidth < 5 { dtWidth = 5 }
	m.domainTable.SetWidth(dtWidth)
	m.domainTable.SetHeight(h - 6)
	m.domainTable.SetColumns([]table.Column{{Title: "Domain", Width: dtWidth - 2}})

	utWidth := m.middleWidth - 4
	if utWidth < 10 { utWidth = 10 }
	m.urlTable.SetWidth(utWidth)
	m.urlTable.SetHeight(h - 6)
	
	statusW := 6
	techW := 10
	urlW := utWidth - statusW - techW - 4
	if urlW < 10 { urlW = 10 }
	m.urlTable.SetColumns([]table.Column{
		{Title: "URL", Width: urlW},
		{Title: "Status", Width: statusW},
		{Title: "Tech", Width: techW},
	})
}

func (m targetModel) View() string {
	if m.adding {
		inputBoxStyle := lipgloss.NewStyle().
			Border(m.theme.Border).
			BorderForeground(m.theme.Accent).
			Padding(1).
			Align(lipgloss.Center)

		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			inputBoxStyle.Render(
				fmt.Sprintf("Add New Target\n\n%s\n\n(Enter to Save, Esc to Cancel)", m.input.View()),
			),
		)
	}

	// Styles for the panes - using explicit widths and BorderBox logic
	paneBase := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1)

	activePaneBase := paneBase.Copy().
		BorderForeground(m.theme.Accent)

	// Inner height for tables
	innerH := m.height - 5
	if innerH < 1 { innerH = 1 }

	// Ensure table height matches pane height
	m.domainTable.SetHeight(innerH)
	m.urlTable.SetHeight(innerH)

	leftStyle := paneBase.Width(m.leftWidth - 4).Height(innerH + 1)
	middleStyle := paneBase.Width(m.middleWidth - 4).Height(innerH + 1)
	rightStyle := paneBase.Width(m.sidebarWidth - 4).Height(innerH + 1)

	if m.activePane == PaneDomains {
		leftStyle = activePaneBase.Width(m.leftWidth - 4).Height(innerH + 1)
	} else if m.activePane == PaneURLs {
		middleStyle = activePaneBase.Width(m.middleWidth - 4).Height(innerH + 1)
	} else {
		rightStyle = activePaneBase.Width(m.sidebarWidth - 4).Height(innerH + 1)
	}

	// Tools Sidebar Content
	toolsHeader := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true).Underline(true).PaddingBottom(1).Render("TOOLS")
	var renderedTools []string
	for i, t := range m.toolList {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
		if i == m.selectedTool && m.activePane == PaneTools {
			prefix = "> "
			style = style.Foreground(m.theme.Accent).Bold(true)
		}
		
		text := t
		if t == "AI Mode" {
			status := "[OFF]"
			if m.aiMode { status = "[ON]" }
			text = fmt.Sprintf("AI Mode %s", status)
			style = style.Foreground(lipgloss.Color("5"))
		}
		renderedTools = append(renderedTools, style.Render(prefix+text))
	}

	aiBtnStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("5")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		MarginTop(1).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.sidebarWidth - 8)
	
	aiBtn := aiBtnStyle.Render("AI ANALYZE")
	rightContent := lipgloss.JoinVertical(lipgloss.Left, toolsHeader, strings.Join(renderedTools, "\n"), "", m.zm.Mark("target-ai-btn", aiBtn))

	// Compose layout
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(m.domainTable.View()),
		middleStyle.Render(m.urlTable.View()),
		rightStyle.Render(rightContent),
	)

	help := lipgloss.NewStyle().
		Foreground(m.theme.InactiveTabFG).
		PaddingLeft(1).
		Render(" [tab] Cycle Pane   [a] Add   [d] Delete   [enter] Select/Run")

	return lipgloss.JoinVertical(lipgloss.Left, content, help)
}

func (m *targetModel) refreshTargets() {
	database, err := db.OpenDatabase()
	if err != nil {
		m.err = err
		return
	}
	defer database.Close()

	targets, err := db.GetTargetsWithSubs(database)
	if err != nil {
		m.err = err
		return
	}

	rows := []table.Row{}
	for _, t := range targets {
		rows = append(rows, table.Row{t.Domain})
		for _, s := range t.Subdomains {
			rows = append(rows, table.Row{"  └── " + s})
		}
	}
	m.domainTable.SetRows(rows)
}

func (m *targetModel) refreshURLs() {
	if m.selectedTarget == "" {
		m.urlTable.SetRows([]table.Row{})
		m.urlToSession = make(map[string]string)
		return
	}

	database, err := db.OpenDatabase()
	if err != nil {
		m.err = err
		return
	}
	defer database.Close()

	urls, err := db.GetUrlsBySubdomain(database, m.selectedTarget)
	if err != nil {
		m.err = err
		return
	}

	rows := []table.Row{}
	m.urlToSession = make(map[string]string)
	for _, u := range urls {
		rows = append(rows, table.Row{u["url"], u["status"], u["tech"]})
		if u["session_id"] != "" {
			m.urlToSession[u["url"]] = u["session_id"]
		}
	}
	m.urlTable.SetRows(rows)
}
