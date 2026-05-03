package gui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name            string
	Background      lipgloss.Color
	Foreground      lipgloss.Color
	Accent          lipgloss.Color
	Highlight       lipgloss.Color
	Border          lipgloss.Border
	BorderColor     lipgloss.Color
	ActiveTabBG     lipgloss.Color
	ActiveTabFG     lipgloss.Color
	InactiveTabBG   lipgloss.Color
	InactiveTabFG   lipgloss.Color
	SidebarBG       lipgloss.Color
	SidebarFG       lipgloss.Color
	SidebarSelect   lipgloss.Color
}

var (
	ModernTheme = Theme{
		Name:          "Modern",
		Background:    lipgloss.Color("#1a1b26"),
		Foreground:    lipgloss.Color("#a9b1d6"),
		Accent:        lipgloss.Color("#7aa2f7"),
		Highlight:     lipgloss.Color("#bb9af7"),
		Border:        lipgloss.RoundedBorder(),
		BorderColor:   lipgloss.Color("#444b6a"),
		ActiveTabBG:   lipgloss.Color("#7aa2f7"),
		ActiveTabFG:   lipgloss.Color("#1a1b26"),
		InactiveTabBG: lipgloss.Color("#24283b"),
		InactiveTabFG: lipgloss.Color("#565f89"),
		SidebarBG:     lipgloss.Color("#16161e"),
		SidebarFG:     lipgloss.Color("#a9b1d6"),
		SidebarSelect: lipgloss.Color("#7aa2f7"),
	}

	CyberpunkTheme = Theme{
		Name:          "Cyberpunk",
		Background:    lipgloss.Color("#000000"),
		Foreground:    lipgloss.Color("#00ff00"),
		Accent:        lipgloss.Color("#ff00ff"),
		Highlight:     lipgloss.Color("#00ffff"),
		Border:        lipgloss.DoubleBorder(),
		BorderColor:   lipgloss.Color("#ff00ff"),
		ActiveTabBG:   lipgloss.Color("#ff00ff"),
		ActiveTabFG:   lipgloss.Color("#000000"),
		InactiveTabBG: lipgloss.Color("#1a1a1a"),
		InactiveTabFG: lipgloss.Color("#ff00ff"),
		SidebarBG:     lipgloss.Color("#0a0a0a"),
		SidebarFG:     lipgloss.Color("#00ffff"),
		SidebarSelect: lipgloss.Color("#ff00ff"),
	}

	MatrixTheme = Theme{
		Name:          "Matrix",
		Background:    lipgloss.Color(""),
		Foreground:    lipgloss.Color("2"),
		Accent:        lipgloss.Color("2"),
		Highlight:     lipgloss.Color("10"),
		Border:        lipgloss.NormalBorder(),
		BorderColor:   lipgloss.Color("2"),
		ActiveTabBG:   lipgloss.Color("2"),
		ActiveTabFG:   lipgloss.Color("0"),
		InactiveTabBG: lipgloss.Color("0"),
		InactiveTabFG: lipgloss.Color("22"),
		SidebarBG:     lipgloss.Color("0"),
		SidebarFG:     lipgloss.Color("2"),
		SidebarSelect: lipgloss.Color("22"),
	}
)

func (t Theme) BaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Foreground)
}

func (t Theme) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(t.Border).BorderForeground(t.BorderColor)
}
