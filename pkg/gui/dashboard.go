package gui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"

	"github.com/HaythmKenway/autoscout/internal/db"
	scheduler "github.com/HaythmKenway/autoscout/internal/scheduler"
	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/HaythmKenway/autoscout/pkg/tools"
)

type TickMsg time.Time

type dashboardModel struct {
	dialog      dialog
	app_status  bool
	burp_status bool
	burp_reqs    int
	burp_resps   int
	burp_queue   []string
	burp_port    string
	workQueue    chan burp.BurpRequest
	viewport     viewport.Model
	ready       bool
	logPath     string
	theme       Theme
	stats       db.Stats
	activeJobs  []*tools.Job
	selectedJob int
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

func NewDashboardModel(w int, h int, port string, workQueue chan burp.BurpRequest) dashboardModel {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".autoscout", "go.log")

	m := dashboardModel{
		dialog:      dialog{width: w, height: h, id: "dash"},
		logPath:     logPath,
		ready:       true,
		theme:       ModernTheme,
		burp_status: burp.IsRunning(),
		burp_port:   port,
		workQueue:   workQueue,
	}

	m.viewport = viewport.New(w, 0)
	m.updateViewportHeight()
	m.viewport.SetContent("Waiting for system events...")

	return m
}

func runScheduler(status bool) tea.Cmd {
	return func() tea.Msg {
		scheduler.Skibbidi(status)
		return nil
	}
}

func toggleBurp(status bool, port string, workQueue chan burp.BurpRequest) tea.Cmd {
	return func() tea.Msg {
		if status {
			burp.StartServer(port, workQueue)
		} else {
			burp.StopServer()
		}
		return nil
	}
}

func (m dashboardModel) Update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dialog.width = msg.Width
		m.dialog.height = msg.Height
		m.viewport.Width = msg.Width - 2 // Adjust for feedContainer borders
		m.updateViewportHeight()

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.app_status = !m.app_status
			cmds = append(cmds, runScheduler(m.app_status))
		case "b":
			m.burp_status = !m.burp_status
			cmds = append(cmds, toggleBurp(m.burp_status, m.burp_port, m.workQueue))
		case "up":
			if m.selectedJob > 0 {
				m.selectedJob--
			}
		case "down":
			if m.selectedJob < len(m.activeJobs)-1 {
				m.selectedJob++
			}
		case "x": // Kill selected job
			if len(m.activeJobs) > 0 {
				job := m.activeJobs[m.selectedJob]
				tools.DefaultJobManager.StopJob(job.ID)
			}
		}

	case TickMsg:
		if s, err := db.GetStats(); err == nil {
			m.stats = s
		}
		m.burp_status = burp.IsRunning()
		m.burp_reqs, m.burp_resps, m.burp_queue = burp.GetStats()
		
		// Sort jobs: most recent first
		jobs := tools.DefaultJobManager.ListJobs()
		for i := 0; i < len(jobs); i++ {
			for j := i + 1; j < len(jobs); j++ {
				if jobs[i].StartTime.Before(jobs[j].StartTime) {
					jobs[i], jobs[j] = jobs[j], jobs[i]
				}
			}
		}
		m.activeJobs = jobs

		if m.selectedJob >= len(m.activeJobs) {
			m.selectedJob = 0
		}
		// Fetch EXACTLY the number of lines the viewport can show
		// Use maxWidth - 4 to account for rounded border and padding
		content := getFormattedLogsTailSafe(m.logPath, m.viewport.Height, m.viewport.Width-4)
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		return m, tickEvery()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *dashboardModel) updateViewportHeight() {
	// Math: topRow (8) + feedHeader (2) + feedContainer border (2) + help (1) = 13 lines total overhead
	vpHeight := m.dialog.height - 13
	if vpHeight < 2 {
		vpHeight = 2
	}
	m.viewport.Height = vpHeight
}

func (m dashboardModel) vw(p float64) int {
	return int(float64(m.dialog.width) * p / 100.0)
}

func (m dashboardModel) vh(p float64) int {
	return int(float64(m.dialog.height) * p / 100.0)
}

func (m dashboardModel) View(zm *zone.Manager) string {
	if !m.ready {
		return "Initializing Dashboard..."
	}

	// 1. Calculate absolute base dimensions
	// rightPanelTotalWidth is the space for the active jobs card (35% of total)
	rightPanelTotalWidth := int(float64(m.dialog.width) * 0.35)
	if rightPanelTotalWidth < 30 {
		rightPanelTotalWidth = 30
	}
	// leftPanelTotalWidth is the remaining space for Services + Stats
	leftPanelTotalWidth := m.dialog.width - rightPanelTotalWidth

	// 2. Calculate card widths (subtracting 2 for borders)
	controlsTotalWidth := leftPanelTotalWidth / 2
	statsTotalWidth := leftPanelTotalWidth - controlsTotalWidth

	controlsWidth := controlsTotalWidth - 2
	statsWidth := statsTotalWidth - 2
	jobsWidth := rightPanelTotalWidth - 2

	// Safety clamping
	if controlsWidth < 4 { controlsWidth = 4 }
	if statsWidth < 4 { statsWidth = 4 }
	if jobsWidth < 4 { jobsWidth = 4 }

	accent := m.theme.Accent
	headerStyle := lipgloss.NewStyle().Foreground(accent).Bold(true).Underline(true)
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1)

	topRowHeight := 8 // Total height with borders

	// --- 1. Service Controls ---
	statusText := "OFFLINE"
	statusColor := lipgloss.Color("1")
	if m.app_status {
		statusText = "RUNNING"
		statusColor = lipgloss.Color("2")
	}

	burpStatusText := "OFFLINE"
	burpStatusColor := lipgloss.Color("1")
	if m.burp_status {
		burpStatusText = "ACTIVE "
		burpStatusColor = lipgloss.Color("2")
	}

	startBtnText := " START "
	if m.app_status {
		startBtnText = " STOP  "
	}
	startBtnStyle := lipgloss.NewStyle().Background(m.theme.Accent).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	startBtn := zm.Mark(m.dialog.id+"ToggleStart", startBtnStyle.Render(startBtnText))

	burpBtnText := " START "
	if m.burp_status {
		burpBtnText = " STOP  "
	}
	burpBtnStyle := lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	burpBtn := zm.Mark(m.dialog.id+"ToggleBurp", burpBtnStyle.Render(burpBtnText))

	controls := cardStyle.Width(controlsWidth).Height(topRowHeight - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("SERVICES"),
			"",
			fmt.Sprintf("Scanner: %s", lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(statusText)),
			startBtn,
			fmt.Sprintf("Burp:    %s", lipgloss.NewStyle().Foreground(burpStatusColor).Bold(true).Render(burpStatusText)),
			burpBtn,
		),
	)

	// --- 2. Stats ---
	stats := cardStyle.Width(statsWidth).Height(topRowHeight - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("STATISTICS"),
			"",
			fmt.Sprintf("Targets: %d", m.stats.Targets),
			fmt.Sprintf("Subs:    %d", m.stats.Subs),
			fmt.Sprintf("Reqs:    %d", m.burp_reqs),
			fmt.Sprintf("Resps:   %d", m.burp_resps),
		),
	)

	// --- 3. Active Jobs ---
	var jobsView string
	if len(m.activeJobs) == 0 {
		jobsView = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render("\n\n  No active tasks...")
	} else {
		var rows []string
		maxJobs := 4
		for i := 0; i < len(m.activeJobs) && len(rows) < maxJobs; i++ {
			job := m.activeJobs[i]
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == m.selectedJob {
				cursor = "> "
				style = style.Foreground(m.theme.Accent).Bold(true)
			}
			elapsed := time.Since(job.StartTime).Round(time.Second)
			row := fmt.Sprintf("%s%-8s | %s (%s)", cursor, job.Tool, job.ID, elapsed)
			// Truncate to fit jobsWidth
			maxLen := jobsWidth
			if lipgloss.Width(row) > maxLen {
				row = row[:maxLen-3] + "..."
			}
			rows = append(rows, style.Render(row))
		}
		jobsView = "\n" + strings.Join(rows, "\n")
	}

	activeJobsCard := cardStyle.Width(jobsWidth).Height(topRowHeight - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(fmt.Sprintf("ACTIVE TASKS (%d)", len(m.activeJobs))),
			jobsView,
		),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		controls,
		stats,
		activeJobsCard,
	)

	m.updateViewportHeight()

	feedHeader := headerStyle.MarginTop(1).Render(" SYSTEM FEED ")

	feedWidth := m.dialog.width - 2
	if feedWidth < 10 { feedWidth = 10 }

	feedContainer := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(m.theme.BorderColor).
		Width(feedWidth).
		Height(m.viewport.Height).
		Render(m.viewport.View())

	help := lipgloss.NewStyle().
		Foreground(m.theme.InactiveTabFG).
		Height(1).
		MaxHeight(1).
		Render(" [s] Scanner  [b] Burp API  [x] Kill Task  [↑/↓] Select  [q] Exit")

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		feedHeader,
		feedContainer,
		help,
	)
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func getFormattedLogsTailSafe(path string, maxLines int, maxWidth int) string {
	file, err := os.Open(path)
	if err != nil {
		return "Waiting for logs..."
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.Size() == 0 {
		return "Log feed empty..."
	}

	chunkSize := int64(32768)
	offset := stat.Size() - chunkSize
	if offset < 0 {
		offset = 0
		chunkSize = stat.Size()
	}

	buf := make([]byte, chunkSize)
	_, err = file.ReadAt(buf, offset)
	if err != nil && err != io.EOF {
		return "Error reading logs"
	}

	raw := string(buf)
	lines := strings.Split(raw, "\n")

	if offset > 0 && len(lines) > 1 {
		lines = lines[1:]
	}

	var filtered []string
	for _, line := range lines {
		clean := sanitizeLine(line)
		if clean == "" {
			continue
		}

		style := lipgloss.NewStyle()
		if strings.Contains(clean, "ERRO") || strings.Contains(clean, "CRIT") {
			style = style.Foreground(lipgloss.Color("1"))
		} else if strings.Contains(clean, "ALER") {
			style = style.Foreground(lipgloss.Color("3"))
		} else if strings.Contains(clean, "INFO") {
			style = style.Foreground(lipgloss.Color("6"))
		}

		displayLine := clean
		if maxWidth > 0 && len(clean) > maxWidth {
			displayLine = clean[:maxWidth-3] + "..."
		}

		filtered = append(filtered, style.Render(displayLine))
	}

	if len(filtered) == 0 {
		return "Scanning log feed..."
	}

	if len(filtered) > maxLines && maxLines > 0 {
		filtered = filtered[len(filtered)-maxLines:]
	}

	return strings.Join(filtered, "\n")
}

func sanitizeLine(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}
