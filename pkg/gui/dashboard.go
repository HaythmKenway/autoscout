package gui

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	scheduler "github.com/HaythmKenway/autoscout/internal/scheduler"
)

type TickMsg time.Time

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			Align(lipgloss.Center)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1).
			MarginRight(2)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)

	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	logTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("62"))
)

type dashboardModel struct {
	dialog     dialog
	app_status bool
	viewport   viewport.Model
	ready      bool
	logPath    string
}

type dialog struct {
	id       string
	height   int
	width    int
	active   string
	question string
}

func (m dashboardModel) Init() tea.Cmd {
	return tickEvery()
}

func NewDashboardModel(w int, h int) dashboardModel {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".autoscout", "go.log")

	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}

	headerHeight := 8
	titleHeight := 2
	viewportHeight := 10

	vpWidth := w - 6
	if vpWidth < 0 {
		vpWidth = 0
	}

	vp := viewport.New(vpWidth, viewportHeight)
	vp.YPosition = headerHeight + titleHeight + 1
	vp.SetContent("Loading logs...")

	return dashboardModel{
		dialog:   dialog{width: w, height: h, id: "dash"},
		logPath:  logPath,
		viewport: vp,
		ready:    true,
	}
}

// runScheduler executes the background logic as a tea.Cmd to keep UI responsive
func runScheduler(status bool) tea.Cmd {
	return func() tea.Msg {
		scheduler.Skibbidi(status)
		return nil
	}
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dialog.width = msg.Width
		m.dialog.height = msg.Height
		m.viewport.Width = msg.Width - 6
		m.viewport.Height = 10
		m.ready = true

	case tea.MouseMsg:
		// FIX: Only check for click events here, but DO NOT return early.
		// We must allow the code to fall through to m.viewport.Update(msg)
		// so that mouse wheel scrolling works in the log view.
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if zone.Get(m.dialog.id + "ToggleStart").InBounds(msg) {
				m.app_status = !m.app_status
				cmds = append(cmds, runScheduler(m.app_status))
			}
		}

	case TickMsg:
		content := getLastNLines(m.logPath, 10)
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		return m, tickEvery()
	}

	// Update viewport (handles scrolling)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m dashboardModel) View() string {
	if !m.ready {
		return "Initializing Dashboard..."
	}

	// 1. Control Panel
	var startButton, question string
	if m.app_status {
		startButton = activeButtonStyle.Render("Stop")
		question = lipgloss.NewStyle().Width(27).Align(lipgloss.Center).Render("Service Running")
	} else {
		startButton = buttonStyle.Render("Start")
		question = lipgloss.NewStyle().Width(27).Align(lipgloss.Center).Render("Service Stopped")
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Top,
		// Mark the button with the ID.
		// Note: The ID here matches the one checked in Update
		zone.Mark(m.dialog.id+"ToggleStart", startButton),
	)

	controlPanel := dialogBoxStyle.Width(m.dialog.width - 4).Render(
		lipgloss.JoinVertical(lipgloss.Center, question, buttons),
	)

	// 2. Log Panel
	logWidth := m.dialog.width - 6
	if logWidth < 0 {
		logWidth = 0
	}

	titleText := fmt.Sprintf("Logs (%s)", filepath.Base(m.logPath))
	logTitle := logTitleStyle.Width(logWidth).Render(titleText)

	logContent := lipgloss.JoinVertical(
		lipgloss.Left,
		logTitle,
		m.viewport.View(),
	)

	logPanel := logBoxStyle.
		Width(m.dialog.width - 4).
		Render(logContent)

	ui := lipgloss.JoinVertical(lipgloss.Left, controlPanel, logPanel)

	// CRITICAL FIX: Return the raw UI string.
	// Do NOT call zone.Scan(ui) here, because gui.go (the parent) scans it.
	return ui
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func getLastNLines(path string, n int) string {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(filepath.Dir(path), 0755)
			os.Create(path)
			return "Log file created."
		}
		return "Error reading log."
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return "No logs found."
	}

	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	return strings.Join(lines[start:], "\n")
}
