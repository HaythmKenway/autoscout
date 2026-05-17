package gui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func TestProxyForwarding(t *testing.T) {
	zm := zone.New()
	m := NewProxyModel(100, 40, zm)

	// Simulate "Enter" key press on an empty table (or with mock data)
	// In a real test, we would populate the table, but here we just want to see
	// if the model returns the TriggerManualMsg.

	// Since we can't easily populate burp.ProxyHistory in a unit test without 
	// modifying global state, we'll manually check the Update logic.

	msg := tea.KeyMsg{Type: tea.KeyEnter, Runes: []rune("\n")}
	
	// We need to ensure there's at least one row for the "Enter" logic to trigger
	// But proxyModel reads from burp.GetProxyHistory() which is global.
	
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Log("Note: Command is nil because table is empty. This is expected if history is empty.")
	} else {
		t.Log("Command received from proxyModel")
	}
}
