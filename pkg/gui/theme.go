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
		Background:    lipgloss.Color(""),
		Foreground:    lipgloss.Color("7"),
		Accent:        lipgloss.Color("#874BFD"),
		Highlight:     lipgloss.Color("#7D56F4"),
		Border:        lipgloss.NormalBorder(),
		BorderColor:   lipgloss.Color("#874BFD"),
		ActiveTabBG:   lipgloss.Color("#874BFD"),
		ActiveTabFG:   lipgloss.Color("7"),
		InactiveTabBG: lipgloss.Color("235"),
		InactiveTabFG: lipgloss.Color("243"),
		SidebarBG:     lipgloss.Color("233"),
		SidebarFG:     lipgloss.Color("7"),
		SidebarSelect: lipgloss.Color("#874BFD"),
	}

	NeonTheme = Theme{
		Name:          "Neon",
		Background:    lipgloss.Color(""),
		Foreground:    lipgloss.Color("7"),
		Accent:        lipgloss.Color("#ff00ff"),
		Highlight:     lipgloss.Color("#00ffff"),
		Border:        lipgloss.ThickBorder(),
		BorderColor:   lipgloss.Color("#ff00ff"),
		ActiveTabBG:   lipgloss.Color("#ff00ff"),
		ActiveTabFG:   lipgloss.Color("0"),
		InactiveTabBG: lipgloss.Color("234"),
		InactiveTabFG: lipgloss.Color("#ff00ff"),
		SidebarBG:     lipgloss.Color("0"),
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
