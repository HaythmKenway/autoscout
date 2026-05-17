package gui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pushState int

const (
	pushStateIdle pushState = iota
	pushStateSyncing
	pushStateComplete
)

type pushModel struct {
	width    int
	height   int
	theme    Theme
	state    pushState
	progress progress.Model
	spinner  spinner.Model
	results  []string
}

func NewPushModel(w, h int) pushModel {
	p := progress.New(progress.WithDefaultGradient())
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ModernTheme.Accent)

	return pushModel{
		width:    w,
		height:   h,
		theme:    ModernTheme,
		state:    pushStateIdle,
		progress: p,
		spinner:  s,
		results:  []string{},
	}
}

func (m pushModel) Init() tea.Cmd {
	return nil
}

func (m pushModel) Update(msg tea.Msg) (pushModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
	case tea.KeyMsg:
		if m.state == pushStateIdle && msg.String() == "p" {
			m.state = pushStateSyncing
			m.results = append(m.results, "Analyzing active session data...")
			cmds = append(cmds, m.spinner.Tick, m.simulateSync())
		}
	case spinner.TickMsg:
		if m.state == pushStateSyncing {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	case progress.FrameMsg:
		newModel, cmd := m.progress.Update(msg)
		if pm, ok := newModel.(progress.Model); ok {
			m.progress = pm
			cmds = append(cmds, cmd)
		}
	case syncStepMsg:
		m.results = append(m.results, msg.text)
		if msg.percent >= 1.0 {
			m.state = pushStateComplete
		}
		cmds = append(cmds, m.progress.SetPercent(msg.percent))
	}

	return m, tea.Batch(cmds...)
}

type syncStepMsg struct {
	percent float64
	text    string
}

func (m pushModel) simulateSync() tea.Cmd {
	return func() tea.Msg {
		steps := []struct {
			p float64
			t string
		}{
			{0.2, "Collecting request history..."},
			{0.4, "Indexing AI analysis logs..."},
			{0.6, "Packaging forensic artifacts..."},
			{0.8, "Pushing to central AI backend..."},
			{1.0, "Sync complete. 124 entries processed."},
		}
		for range steps {
			time.Sleep(800 * time.Millisecond)
			// In a real app, this would send the actual data
		}
		// Final step
		return syncStepMsg{percent: 1.0, text: "Data successfully synced to ~/.autoscout/cloud"}
	}
}

func (m pushModel) View() string {
	accent := m.theme.Accent
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(accent).
		Underline(true).
		MarginBottom(1)

	var content string
	switch m.state {
	case pushStateIdle:
		content = lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("DATA PUSH & SYNC"),
			"",
			"Use this panel to sync your current session data, AI analysis history,",
			"and forensic artifacts to the centralized investigation backend.",
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true).Render(" [p] Start Sync Process "),
			"",
			"Note: Syncing ensures your findings are backed up and available for",
			"distributed analysis across the AI fleet.",
		)
	case pushStateSyncing, pushStateComplete:
		resView := ""
		if len(m.results) > 0 {
			resView = "\n" + strings.Join(m.results, "\n")
		}
		
		status := "Syncing..."
		if m.state == pushStateComplete {
			status = "SYNC COMPLETE"
		}

		content = lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("SYNC IN PROGRESS"),
			"",
			m.spinner.View()+" "+status,
			"",
			m.progress.View(),
			resView,
		)
	}

	return lipgloss.NewStyle().Padding(2, 4).Render(content)
}
