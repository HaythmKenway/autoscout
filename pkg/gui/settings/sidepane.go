package settings

import (
	// "fmt"
	// "os"
	// "strings"

	tea "github.com/charmbracelet/bubbletea"
	// "github.com/charmbracelet/lipgloss"
	// "github.com/charmbracelet/huh"
	// zone "github.com/lrstanley/bubblezone"
)

type model struct {
	FontColor      string
	MetricsEnabled bool
	GrafanaEnabled bool
	width          int
}

func New() model {
	return model{
		FontColor:      "Blue",
		MetricsEnabled: true,
		GrafanaEnabled: false,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() string {
 return "This aint working"
}
