package gui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	charm "github.com/HaythmKenway/autoscout/pkg/gui/settings"
)

type model struct {
	Tabs       []string
	TabContent []string
	activeTab  int
	width      int
	height     int
}

func (m model) Init() tea.Cmd {
	return nil
}

func getTerminalSize() (width int, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return width, height
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "right", "l", "n", "tab":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
		case "left", "h", "p", "shift+tab":
			m.activeTab = max(m.activeTab-1, 0)
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		for i := range m.Tabs {
			zoneID := fmt.Sprintf("tab-%d", i)
			if zone.Get(zoneID).InBounds(msg) {
				m.activeTab = i
				break
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

var (
	highlightColor = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	inactiveTabStyle = lipgloss.NewStyle().
				Border(tabBorderWithBottom("┴", "─", "┴"), true).
				BorderForeground(highlightColor).
				Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
				Border(tabBorderWithBottom("┘", " ", "└"), true).
				BorderForeground(highlightColor).
				Padding(0, 1)

	windowStyle = lipgloss.NewStyle().
				Padding(1, 2)

	docStyle = lipgloss.NewStyle().PaddingTop(1)
)

func (m model) View() string {
	var renderedTabs []string
	tabRowWidth := 0

	for i, tab := range m.Tabs {
		style := inactiveTabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		zoneID := fmt.Sprintf("tab-%d", i)
		tabStr := zone.Mark(zoneID, style.Render(tab))
		renderedTabs = append(renderedTabs, tabStr)
		tabRowWidth += lipgloss.Width(tabStr)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	if tabRowWidth < m.width {
		row += lipgloss.NewStyle().
			Foreground(highlightColor).
			Render(strings.Repeat("─", m.width-tabRowWidth))
	}

	// Handle rendering of content based on active tab
	content := ""
	switch m.activeTab {
	case 0:
		content = "This will be dashboard someday"
	case 1:
		content = "This will be Target page in future"
	case 2:
		content = "This will be Analyzing page"
	case 3:
		// Settings Tab (Using charm settings page here)
		settingsModel := charm.New()
		content = settingsModel.View() // Show settings page content
	}

	contentView := windowStyle.Width(m.width).Render(content)

	return zone.Scan(docStyle.Render(row + "\n" + contentView))
}

func LoadGui() error {
	w, h := getTerminalSize()
	zone.NewGlobal()

	m := model{
		Tabs: []string{
			"⌂ Dashboard",
			"➤ Targets",
			"≡ Analysis",
			"☰ Settings",
		},
		TabContent: []string{
			"This will be dashboard someday",
			"This will be Target page in future",
			"This will be Analyzing page",
			"This will be settings page", // Placeholder for settings tab
		},
		width:  w,
		height: h,
	}

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running program: %w", err)
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

