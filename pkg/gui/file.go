package gui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

type fileItem struct {
	title, desc string
}

func (i fileItem) Title() string       { return i.title }
func (i fileItem) Description() string { return i.desc }
func (i fileItem) FilterValue() string { return i.title }

type fileModel struct {
	list   list.Model
	width  int
	height int
	theme  Theme
	zm     *zone.Manager
}

func NewFileModel(w, h int, zm *zone.Manager) fileModel {
	items := []list.Item{
		fileItem{title: "New Session", desc: "Create a fresh investigation workspace"},
		fileItem{title: "Open Session", desc: "Load a previous session from disk"},
		fileItem{title: "Save Session", desc: "Persist current history and findings"},
		fileItem{title: "Exit", desc: "Safely shut down Autoscout"},
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ModernTheme.Accent).BorderForeground(ModernTheme.Accent)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(ModernTheme.Highlight)

	l := list.New(items, d, w, h-4)
	l.Title = "FILE / SESSION MANAGEMENT"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().Background(ModernTheme.Accent).Foreground(lipgloss.Color("0")).Padding(0, 1).Bold(true)

	return fileModel{
		list:   l,
		width:  w,
		height: h,
		theme:  ModernTheme,
		zm:     zm,
	}
}

func (m fileModel) Init() tea.Cmd {
	return nil
}

func (m fileModel) Update(msg tea.Msg) (fileModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-2, msg.Height-4)
	case tea.KeyMsg:
		if msg.String() == "enter" {
			selected := m.list.SelectedItem().(fileItem)
			switch selected.title {
			case "Exit":
				return m, tea.Quit
			case "New Session":
				return m, func() tea.Msg { return TriggerCreateSessionMsg{} }
			case "Open Session":
				return m, func() tea.Msg { return TriggerOpenSessionMsg{} }
			case "Save Session":
				return m, func() tea.Msg { return TriggerSaveSessionMsg{} }
			}
		}
	}
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m fileModel) View() string {
	return lipgloss.NewStyle().Padding(1, 2).Render(m.list.View())
}

type TriggerCreateSessionMsg struct{}
type TriggerOpenSessionMsg struct{}
type TriggerSaveSessionMsg struct{}
