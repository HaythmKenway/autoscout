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
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
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
		app_status:  scheduler.IsRunning(),
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
		m.updateViewportHeight()
		m.viewport.Width = m.dialog.width - 6

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
			if len(m.activeJobs) > 0 && m.selectedJob < len(m.activeJobs) {
				job := m.activeJobs[m.selectedJob]
				localUtils.Logger(fmt.Sprintf("[Dashboard] User requested kill for job %s (%s)", job.ID, job.Tool), 1)
				tools.DefaultJobManager.StopJob(job.ID)
			} else {
				localUtils.Logger("[Dashboard] No job selected or job list empty", 2)
			}
		}

	case TickMsg:
		if s, err := db.GetStats(); err == nil {
			m.stats = s
		}
		m.app_status = scheduler.IsRunning()
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
	// Math: topRow (10) + spacing(1) + feedTitle(1) + feedContainer border (2) + help (1) = 15 lines total overhead
	vpHeight := m.dialog.height - 15
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

	// 1. Precise Math
	w := m.dialog.width
	
	col1W := int(float64(w) * 0.33)
	col2W := int(float64(w) * 0.33)
	col3W := w - col1W - col2W

	topRowH := 8

	accent := m.theme.Accent
	headerStyle := lipgloss.NewStyle().Foreground(accent).Bold(true).Underline(true).PaddingBottom(1)
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1)

	// --- 1. Service Controls ---
	renderStatus := func(active bool) string {
		if active {
			return lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(" ACTIVE ")
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true).Render("OFFLINE ")
	}

	renderBtn := func(label string, active bool, markID string) string {
		text := " START "
		color := accent
		if active {
			text = " STOP  "
			color = lipgloss.Color("1")
		}
		style := lipgloss.NewStyle().Background(color).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
		return zm.Mark(markID, style.Render(text))
	}

	controls := cardStyle.Width(col1W - 2).Height(topRowH - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("SYSTEM SERVICES"),
			fmt.Sprintf("Scanner: %s", renderStatus(m.app_status)),
			renderBtn("START", m.app_status, m.dialog.id+"ToggleStart"),
			"",
			fmt.Sprintf("Burp API: %s", renderStatus(m.burp_status)),
			renderBtn("START", m.burp_status, m.dialog.id+"ToggleBurp"),
		),
	)

	// --- 2. Stats ---
	statLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	statVal := lipgloss.NewStyle().Foreground(m.theme.Highlight).Bold(true)
	
	renderStat := func(label string, val int) string {
		return fmt.Sprintf("%s %s", statLabel.Render(fmt.Sprintf("%-10s", label)), statVal.Render(fmt.Sprintf("%d", val)))
	}

	stats := cardStyle.Width(col2W - 2).Height(topRowH - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render("LIVE STATISTICS"),
			renderStat("Targets", m.stats.Targets),
			renderStat("Subdomains", m.stats.Subs),
			renderStat("Requests", m.burp_reqs),
			renderStat("Responses", m.burp_resps),
		),
	)

	// --- 3. Active Jobs ---
	var jobsView string
	if len(m.activeJobs) == 0 {
		jobsView = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render("\n  ( •_•)\n  No active tasks...")
	} else {
		var rows []string
		for i := 0; i < len(m.activeJobs) && i < 4; i++ {
			job := m.activeJobs[i]
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
			if i == m.selectedJob {
				style = style.Foreground(m.theme.Accent).Bold(true)
			}
			row := fmt.Sprintf("• %-8s [%s]", job.Tool, job.ID[:min(6, len(job.ID))])
			if job.IsKilling {
				style = style.Foreground(lipgloss.Color("1"))
				row += " [!]"
			}
			rows = append(rows, style.Render(row))
		}
		jobsView = strings.Join(rows, "\n")
	}

	activeJobsCard := cardStyle.Width(col3W - 2).Height(topRowH - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			headerStyle.Render(fmt.Sprintf("ACTIVE JOBS (%d)", len(m.activeJobs))),
			jobsView,
		),
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, controls, stats, activeJobsCard)

	// --- 4. System Feed ---
	m.updateViewportHeight()
	
	feedContainer := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderColor).
		Width(w - 2).
		Height(m.viewport.Height).
		Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		topRow,
		feedContainer,
	)
}

func tickEvery() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
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
