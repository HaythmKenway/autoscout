package gui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/HaythmKenway/autoscout/internal/db"
)

var (
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1).
			Align(lipgloss.Center)

	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
)

type targetModel struct {
	table  table.Model
	input  textinput.Model
	adding bool
	width  int
	height int
	err    error
}

func NewTargetModel(w, h int) targetModel {
	// 1. Configure Table
	columns := []table.Column{
		{Title: "Target Domain", Width: w - 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(h-8), // Reserve space for help text/headers
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// 2. Configure Input (for adding targets)
	ti := textinput.New()
	ti.Placeholder = "example.com"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 30

	m := targetModel{
		table:  t,
		input:  ti,
		adding: false,
		width:  w,
		height: h,
	}

	// Load initial data
	m.refreshTargets()
	return m
}

func (m targetModel) Init() tea.Cmd {
	return nil
}

func (m targetModel) Update(msg tea.Msg) (targetModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Resize table dynamically
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 8)

		// Resize column to fit width
		cols := m.table.Columns()
		if len(cols) > 0 {
			cols[0].Width = msg.Width - 10
			m.table.SetColumns(cols)
		}

	case tea.KeyMsg:
		// === Input Mode ===
		if m.adding {
			switch msg.String() {
			case "enter":
				// Save to DB
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

		// === Table Mode ===
		switch msg.String() {
		case "a":
			m.adding = true
			m.input.Focus()
			return m, textinput.Blink
		case "d", "backspace", "delete":
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				db.RemoveTarget(selected[0])
				m.refreshTargets()
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m targetModel) View() string {
	if m.adding {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			inputBoxStyle.Render(
				fmt.Sprintf("Add New Target\n\n%s\n\n(Enter to Save, Esc to Cancel)", m.input.View()),
			),
		)
	}

	return baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.table.View(),
			"\n [a] Add Target   [d] Delete Target   [↑/↓] Navigate",
		),
	)
}

// Helper to reload data from DB
func (m *targetModel) refreshTargets() {
	database, err := db.OpenDatabase()
	if err != nil {
		m.err = err
		return
	}
	defer database.Close()

	targets, err := db.GetTargetsFromTable(database)
	if err != nil {
		m.err = err
		return
	}

	rows := []table.Row{}
	for _, t := range targets {
		rows = append(rows, table.Row{t})
	}

	m.table.SetRows(rows)
}
