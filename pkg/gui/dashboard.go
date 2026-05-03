package gui

import (
	"fmt"
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
)

type TickMsg time.Time

type dashboardModel struct {
	dialog      dialog
	app_status  bool
	burp_status bool
	viewport    viewport.Model
	ready       bool
	logPath     string
	theme       Theme
	stats       db.Stats
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

	vp := viewport.New(w, h-14)
	vp.SetContent("Waiting for system events...")

	return dashboardModel{
		dialog:      dialog{width: w, height: h, id: "dash"},
		logPath:     logPath,
		viewport:    vp,
		ready:       true,
		theme:       ModernTheme,
		burp_status: burp.IsRunning(),
	}
}

func runScheduler(status bool) tea.Cmd {
	return func() tea.Msg {
		scheduler.Skibbidi(status)
		return nil
	}
}

func toggleBurp(status bool) tea.Cmd {
	return func() tea.Msg {
		if status {
			burp.StartServer("8081")
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
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 14

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.app_status = !m.app_status
			cmds = append(cmds, runScheduler(m.app_status))
		case "b":
			m.burp_status = !m.burp_status
			cmds = append(cmds, toggleBurp(m.burp_status))
		}

	case TickMsg:
		if s, err := db.GetStats(); err == nil {
			m.stats = s
		}
		m.burp_status = burp.IsRunning()
		content := getFormattedLogsTailSafe(m.logPath, 20)
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		return m, tickEvery()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m dashboardModel) View(zm *zone.Manager) string {
	if !m.ready {
		return "Initializing Dashboard..."
	}

	// 1. Service Status
	statusText := "OFFLINE"
	statusColor := lipgloss.Color("1")
	if m.app_status {
		statusText = "RUNNING"
		statusColor = lipgloss.Color("2")
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(statusColor).
		Render(fmt.Sprintf("SCANNER SERVICE: %s", statusText))

	btnText := "[ START SCANNER ]"
	if m.app_status {
		btnText = "[ STOP SCANNER ] "
	}
	btn := zm.Mark(m.dialog.id+"ToggleStart", lipgloss.NewStyle().
		Background(m.theme.Accent).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render(btnText))

	// 2. Burp Status
	burpStatusText := "OFFLINE"
	burpStatusColor := lipgloss.Color("1")
	if m.burp_status {
		burpStatusText = "ACTIVE"
		burpStatusColor = lipgloss.Color("2")
	}

	burpHeader := lipgloss.NewStyle().
		Bold(true).
		Foreground(burpStatusColor).
		Render(fmt.Sprintf("BURP INTEGRATION: %s", burpStatusText))

	burpBtnText := "[ START API ]"
	if m.burp_status {
		burpBtnText = "[ STOP API ] "
	}
	burpBtn := zm.Mark(m.dialog.id+"ToggleBurp", lipgloss.NewStyle().
		Background(lipgloss.Color("6")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render(burpBtnText))

	stats := fmt.Sprintf("Targets: %d | Subdomains: %d | Alive URLs: %d", m.stats.Targets, m.stats.Subs, m.stats.URLs)
	
	logTitle := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("--- LIVE SYSTEM FEED ---")
	
	help := lipgloss.NewStyle().
		Foreground(m.theme.InactiveTabFG).
		Render(" [s] Scanner   [b] Burp API   [1-4] Tabs   [tab] Panels   [q] Quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, header, lipgloss.NewStyle().Width(5).Render(""), btn),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, burpHeader, lipgloss.NewStyle().Width(5).Render(""), burpBtn),
		"",
		lipgloss.NewStyle().Foreground(m.theme.Highlight).Render(stats),
		"",
		logTitle,
		m.viewport.View(),
		"",
		help,
	)
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Ultra-robust log tailing
func getFormattedLogsTailSafe(path string, maxLines int) string {
	file, err := os.Open(path)
	if err != nil {
		return "Waiting for logs..."
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "Log error"
	}

	// Tail the last 8KB
	chunkSize := int64(8192)
	if stat.Size() < chunkSize {
		chunkSize = stat.Size()
	}
	
	buf := make([]byte, chunkSize)
	_, err = file.ReadAt(buf, stat.Size()-chunkSize)
	if err != nil && err != os.ErrExist {
		// Ignore read errors if file is growing
	}

	raw := string(buf)
	lines := strings.Split(raw, "\n")
	
	// Discard first line as it might be a partial line from the seek
	if len(lines) > 1 {
		lines = lines[1:]
	}

	var filtered []string
	for _, line := range lines {
		clean := sanitizeLine(line)
		if clean == "" {
			continue
		}
		
		// Subtle colorization
		if strings.Contains(clean, "ERRO") || strings.Contains(clean, "CRIT") {
			clean = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(clean)
		} else if strings.Contains(clean, "ALER") {
			clean = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(clean)
		} else if strings.Contains(clean, "INFO") {
			clean = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(clean)
		}
		
		filtered = append(filtered, clean)
	}

	if len(filtered) == 0 {
		return "Scanning log feed..."
	}

	if len(filtered) > maxLines {
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
