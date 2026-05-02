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
)

type TickMsg time.Time

type dashboardModel struct {
	dialog     dialog
	app_status bool
	viewport   viewport.Model
	ready      bool
	logPath    string
	theme      Theme
	stats      db.Stats
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

	vp := viewport.New(w, h-12)
	vp.SetContent("Waiting for system events...")

	return dashboardModel{
		dialog:   dialog{width: w, height: h, id: "dash"},
		logPath:  logPath,
		viewport: vp,
		ready:    true,
		theme:    ModernTheme,
	}
}

func runScheduler(status bool) tea.Cmd {
	return func() tea.Msg {
		scheduler.Skibbidi(status)
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
		m.viewport.Height = msg.Height - 12

	case tea.MouseMsg:
		// We handle mouse clicks in the root model using zone.Get
		// But we need to make sure the coordinates are right.
		// Actually, let's just keep the logic in the root model.

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.app_status = !m.app_status
			cmds = append(cmds, runScheduler(m.app_status))
		}

	case TickMsg:
		// Refresh stats with error handling to prevent UI freeze
		if s, err := db.GetStats(); err == nil {
			m.stats = s
		}
		
		// Robust log tailing
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

	statusText := "OFFLINE"
	statusColor := lipgloss.Color("1") // Red
	if m.app_status {
		statusText = "RUNNING"
		statusColor = lipgloss.Color("2") // Green
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(statusColor).
		Render(fmt.Sprintf("SYSTEM STATUS: %s", statusText))

	btnText := "[ START SERVICE ]"
	if m.app_status {
		btnText = "[ STOP SERVICE ]"
	}
	
	// Use the passed manager
	btn := zm.Mark(m.dialog.id+"ToggleStart", lipgloss.NewStyle().
		Background(m.theme.Accent).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(0, 1).
		Render(btnText))

	stats := fmt.Sprintf("Targets: %d | Subdomains: %d | Alive URLs: %d", m.stats.Targets, m.stats.Subs, m.stats.URLs)
	
	logTitle := lipgloss.NewStyle().
		Foreground(m.theme.Accent).
		Bold(true).
		Render("--- LIVE SYSTEM FEED ---")
	
	// Ensure viewport doesn't overflow
	logs := m.viewport.View()

	help := lipgloss.NewStyle().
		Foreground(m.theme.InactiveTabFG).
		Render(" [s] Start/Stop   [1-4] Tabs   [tab] Panels   [q] Quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		btn,
		"",
		lipgloss.NewStyle().Foreground(m.theme.Highlight).Render(stats),
		"",
		logTitle,
		logs,
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
