package gui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	zone "github.com/lrstanley/bubblezone"
)

type model struct {
	Tabs          []string
	activeTab     int
	width         int
	height        int
	settingsModel settingsModel
}

func (m model) Init() tea.Cmd {
	return m.settingsModel.Init()

}

func getTerminalSize() (width int, height int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return width, height
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		updatedModel, settingsCmd := m.settingsModel.Update(msg)
		m.settingsModel = updatedModel
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "right":
			m.activeTab = min(m.activeTab+1, len(m.Tabs)-1)
		case "left":
			m.activeTab = max(m.activeTab-1, 0)
		}
		cmd = tea.Batch(cmd, settingsCmd)

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionRelease || msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		for i := range m.Tabs {
			if zone.Get(fmt.Sprintf("tab-%d", i)).InBounds(msg) {
				m.activeTab = i
				break
			}
		}
		//  And also update on mouse events (might be needed for edge cases)
		updatedModel, settingsCmd := m.settingsModel.Update(msg)
		m.settingsModel = updatedModel
		cmd = tea.Batch(cmd, settingsCmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, cmd
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

	windowStyle = lipgloss.NewStyle().Padding(1, 2)
	docStyle    = lipgloss.NewStyle().PaddingTop(1)
)

func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}

func (m model) View() string {
	var renderedTabs []string
	tabRowWidth := 0

	for i, tab := range m.Tabs {
		style := inactiveTabStyle
		if i == m.activeTab {
			style = activeTabStyle
		}
		tabStr := zone.Mark(fmt.Sprintf("tab-%d", i), style.Render(tab))
		renderedTabs = append(renderedTabs, tabStr)
		tabRowWidth += lipgloss.Width(tabStr)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	if tabRowWidth < m.width {
		row += lipgloss.NewStyle().
			Foreground(highlightColor).
			Render(strings.Repeat("─", m.width-tabRowWidth))
	}

	content := ""
	switch m.activeTab {
	case 0:
		content = "This will be dashboard someday"
	case 1:
		content = "This will be Target page in future"
	case 2:
		content = "This will be Analyzing page"
	case 3:
		content = m.settingsModel.View()
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
		width:         w,
		height:        h,
		settingsModel: NewSettingsModel(w, h),
	}
	m.settingsModel.Init()

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running program: %w", err)
	}
	return nil
}

func SShHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	w, h := getTerminalSize()
	zone.NewGlobal()
	m := model{
		Tabs: []string{
			"⌂ Dashboard",
			"➤ Targets",
			"≡ Analysis",
			"☰ Settings",
		},
		width:         w,
		height:        h,
		settingsModel: NewSettingsModel(w, h),
	}
	return m, []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
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
