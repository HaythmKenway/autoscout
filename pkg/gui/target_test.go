package gui

import (
	"fmt"
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/table"
	zone "github.com/lrstanley/bubblezone"
)

func TestTargetPanelView(t *testing.T) {
	zm := zone.New()
	
	testCases := []struct {
		name string
		w, h int
	}{
		{"Standard 120x30", 120, 30},
		{"Small 80x24", 80, 24},
		{"Very Small 50x15", 50, 15},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// main GUI layout math: sidebar(25%) + separator(1) + margin(4)
			sidebarWidth := int(float64(tc.w) * 0.25)
			if sidebarWidth < 15 { sidebarWidth = 15 }
			if sidebarWidth > 30 { sidebarWidth = 30 }
			rightPanelTotalWidth := tc.w - sidebarWidth - 1
			contentWidth := rightPanelTotalWidth - 4
			contentHeight := tc.h - 2

			m := NewTargetModel(contentWidth, contentHeight, zm)
			
			// Inject mock data
			m.domainTable.SetRows([]table.Row{
				{"example.com"},
				{"  └── api.example.com"},
			})
			m.urlTable.SetRows([]table.Row{
				{"https://api.example.com/v1/users", "200", "Express"},
				{"https://example.com/admin", "403", "Nginx"},
			})

			rendered := m.View()
			
			fileName := fmt.Sprintf("debug_target_%dx%d.txt", tc.w, tc.h)
			_ = os.WriteFile(fileName, []byte(rendered), 0644)
			t.Logf("Wrote %s", fileName)
		})
	}
}
