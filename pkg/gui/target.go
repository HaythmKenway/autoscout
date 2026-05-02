package gui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/HaythmKenway/autoscout/internal/db"
)

type targetModel struct {
	table  table.Model
	input  textinput.Model
	adding bool
	width  int
	height int
	theme  Theme
	err    error
}

func NewTargetModel(w, h int) targetModel {
	columns := []table.Column{
		{Title: "Target Domain", Width: w - 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(h-8),
	)

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
		theme:  ModernTheme,
	}

	m.refreshTargets()
	m.applyStyles()
	return m
}

func (m *targetModel) applyStyles() {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(m.theme.Border).
		BorderForeground(m.theme.BorderColor).
		BorderBottom(true).
		Bold(true).
		Foreground(m.theme.Accent)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffff")).
		Background(m.theme.Highlight).
		Bold(true)
	m.table.SetStyles(s)
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
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 8)
		cols := m.table.Columns()
		if len(cols) > 0 {
			cols[0].Width = msg.Width - 10
			m.table.SetColumns(cols)
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
		inputBoxStyle := lipgloss.NewStyle().
			Border(m.theme.Border).
			BorderForeground(m.theme.Accent).
			Padding(1).
			Align(lipgloss.Center)

		tw := m.width
		if tw < 0 { tw = 0 }
		th := m.height
		if th < 0 { th = 0 }

		return lipgloss.Place(
			tw, th,
			lipgloss.Center, lipgloss.Center,
			inputBoxStyle.Render(
				fmt.Sprintf("Add New Target\n\n%s\n\n(Enter to Save, Esc to Cancel)", m.input.View()),
			),
		)
	}

	tw := m.width - 4
	if tw < 0 { tw = 0 }

	baseStyle := lipgloss.NewStyle().
		BorderStyle(m.theme.Border).
		BorderForeground(m.theme.InactiveTabFG).
		Width(tw)

	return baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			m.table.View(),
			lipgloss.NewStyle().
				Foreground(m.theme.InactiveTabFG).
				MarginTop(1).
				Render(" [a] Add Target   [d] Delete Target   [↑/↓] Navigate"),
		),
	)
}

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
