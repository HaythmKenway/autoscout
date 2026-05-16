package gui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HaythmKenway/autoscout/pkg/ai"
	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TriggerManualMsg struct {
	SessionID string
	TargetURL string
}

type AIResultMsg struct {
	Plan *ai.AIPlan
	Err  error
}

type manualFocus int

const (
	focusReq manualFocus = iota
	focusRes
	focusAI
	focusInput
)

type manualOverlayModel struct {
	reqViewport viewport.Model
	resViewport viewport.Model
	aiViewport  viewport.Model
	input       textinput.Model
	width       int
	height      int
	theme       Theme
	active      bool
	sessionID   string
	targetURL   string
	thinking    string
	focus       manualFocus

	// Raw data for re-wrapping
	rawReq string
	rawRes string
	rawAI  string
}

func NewManualOverlayModel(w, h int) manualOverlayModel {
	rvp := viewport.New(w/2-2, h-14)
	rvp.SetContent("No request data.")

	svp := viewport.New(w/2-2, h-14)
	svp.SetContent("No response data.")

	avp := viewport.New(w-10, 6)
	avp.SetContent("AI output will appear here...")

	ti := textinput.New()
	ti.Placeholder = "Enter AI instructions (e.g., 'Analyze for IDOR')..."
	ti.Focus()
	ti.Width = w - 10

	return manualOverlayModel{
		reqViewport: rvp,
		resViewport: svp,
		aiViewport:  avp,
		input:       ti,
		width:       w,
		height:      h,
		theme:       ModernTheme,
		active:      false,
		focus:       focusInput,
		rawReq:      "No request data.",
		rawRes:      "No response data.",
		rawAI:       "AI output will appear here...",
	}
}

func (m *manualOverlayModel) SetSession(id string) {
	m.sessionID = id
	m.targetURL = ""
	dir := "/tmp/autoscout"

	reqPath := filepath.Join(dir, id+".request")
	if reqData, err := os.ReadFile(reqPath); err == nil {
		m.rawReq = string(reqData)
	} else {
		m.rawReq = fmt.Sprintf("Error reading request: %v", err)
	}

	resPath := filepath.Join(dir, id+".response")
	if resData, err := os.ReadFile(resPath); err == nil {
		m.rawRes = string(resData)
	} else {
		m.rawRes = "No response data available for this session."
	}
	m.rawAI = "Ready for manual investigation."
	m.rebuildViewports()
}

func (m *manualOverlayModel) SetTarget(url string) {
	m.targetURL = url
	m.sessionID = ""
	m.rawReq = fmt.Sprintf("Target: %s\n\nNo request/response traffic captured for this target yet.\nSend traffic from Burp or ask AI what to do with this target.", url)
	m.rawRes = "N/A"
	m.rawAI = "Target focus set. Awaiting instructions."
	m.rebuildViewports()
}

func (m *manualOverlayModel) rebuildViewports() {
	// Frame overhead: 2 (border) + 2 (padding) = 4
	leftW := m.width / 2
	rightW := m.width - leftW

	innerLeftW := leftW - 4
	innerRightW := rightW - 4
	innerAIW := m.width - 8

	if innerLeftW < 5 { innerLeftW = 5 }
	if innerRightW < 5 { innerRightW = 5 }
	if innerAIW < 10 { innerAIW = 10 }

	// 1. Wrap and Set Request
	m.reqViewport.Width = innerLeftW
	m.reqViewport.SetContent(lipgloss.NewStyle().Width(innerLeftW).Render(m.rawReq))

	// 2. Wrap and Set Response
	m.resViewport.Width = innerRightW
	m.resViewport.SetContent(lipgloss.NewStyle().Width(innerRightW).Render(m.rawRes))

	// 3. Wrap and Set AI Output
	m.aiViewport.Width = innerAIW
	m.aiViewport.SetContent(lipgloss.NewStyle().Width(innerAIW).Render(m.rawAI))
}

func (m manualOverlayModel) Update(msg tea.Msg) (manualOverlayModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Responsive Height Budgeting
		// Overhead: Title(1), Help(1), Margins/Padding(6) = 8 lines
		availableH := m.height - 8
		if availableH < 10 {
			availableH = 10
		}

		reqH := int(float64(availableH) * 0.40)
		aiH := int(float64(availableH) * 0.40)
		inputH := availableH - reqH - aiH

		if reqH < 3 { reqH = 3 }
		if aiH < 4 { aiH = 4 }
		if inputH < 3 { inputH = 3 }

		m.reqViewport.Height = reqH
		m.resViewport.Height = reqH
		m.aiViewport.Height = aiH
		m.input.Width = m.width - 10
		m.rebuildViewports()

	case AIResultMsg:
		m.thinking = ""
		if msg.Err != nil {
			m.rawAI = fmt.Sprintf("AI ERROR: %v", msg.Err)
		} else {
			var sb strings.Builder
			sb.WriteString("THINKING:\n")
			sb.WriteString(msg.Plan.Thinking)
			sb.WriteString("\n\nACTIONS:\n")
			if len(msg.Plan.Actions) == 0 {
				sb.WriteString("- No automated tools recommended.")
			}
			for _, act := range msg.Plan.Actions {
				sb.WriteString(fmt.Sprintf("- Run %s on %s\n", act.Tool, act.Target))
			}
			m.rawAI = sb.String()
		}
		m.rebuildViewports()

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.active = false
			return m, nil
		case "tab":
			m.focus = (m.focus + 1) % 4
			if m.focus == focusInput {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil
		case "enter":
			if m.focus == focusInput && m.input.Value() != "" && m.thinking == "" {
				m.thinking = "AI is analyzing context..."
				m.rawAI = "Processing instruction..."
				m.rebuildViewports()
				instruction := m.input.Value()
				m.input.Reset()
				return m, m.executeAI(instruction)
			}
		}
	}

	if m.focus == focusInput {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.focus == focusReq {
		m.reqViewport, cmd = m.reqViewport.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.focus == focusRes {
		m.resViewport, cmd = m.resViewport.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.focus == focusAI {
		m.aiViewport, cmd = m.aiViewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m manualOverlayModel) executeAI(instruction string) tea.Cmd {
	return func() tea.Msg {
		agent := ai.LoadBackend()
		
		// Prepare a dummy request for context
		req := burp.BurpRequest{
			URL:         m.targetURL,
			Tool:        "MANUAL_GUI",
			UserContext: instruction,
		}

		if m.sessionID != "" {
			dir := "/tmp/autoscout"
			reqData, _ := os.ReadFile(filepath.Join(dir, m.sessionID+".request"))
			req.Body = base64.StdEncoding.EncodeToString(reqData)
			req.URL = "Captured Context"

			if resData, err := os.ReadFile(filepath.Join(dir, m.sessionID+".response")); err == nil {
				req.ResponseStatus = 200
				req.ResponseBody = base64.StdEncoding.EncodeToString(resData)
			}
		}

		plan, err := agent.Analyze(req)
		return AIResultMsg{Plan: plan, Err: err}
	}
}

func (m manualOverlayModel) View() string {
	if !m.active {
		return ""
	}

	title := "  MANUAL INVESTIGATION  "
	if m.sessionID != "" {
		title = "  INVESTIGATION: " + m.sessionID + "  "
	} else if m.targetURL != "" {
		title = "  TARGET: " + m.targetURL + "  "
	}

	titleStyle := lipgloss.NewStyle().
		Background(m.theme.Accent).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Align(lipgloss.Center).
		Width(m.width)

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderColor).
		Padding(0, 1)

	activePaneStyle := paneStyle.Copy().BorderForeground(m.theme.Accent)

	leftPaneStyle := paneStyle
	rightPaneStyle := paneStyle
	aiBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("240")). 
		Padding(0, 1)
	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("240")). 
		Padding(0, 1)

	focusColor := lipgloss.Color("5") 
	switch m.focus {
	case focusReq:
		leftPaneStyle = activePaneStyle
	case focusRes:
		rightPaneStyle = activePaneStyle
	case focusAI:
		aiBoxStyle = aiBoxStyle.BorderForeground(focusColor)
	case focusInput:
		inputBoxStyle = inputBoxStyle.BorderForeground(lipgloss.Color("2")) 
	}

	leftW := m.width / 2
	rightW := m.width - leftW

	leftPane := leftPaneStyle.Width(leftW - 2).Height(m.reqViewport.Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Underline(true).Render("REQUEST"),
			m.reqViewport.View(),
		),
	)

	rightPane := rightPaneStyle.Width(rightW - 2).Height(m.resViewport.Height + 2).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Underline(true).Render("RESPONSE"),
			m.resViewport.View(),
		),
	)

	sideBySide := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	aiBox := aiBoxStyle.
		Width(m.width - 2).
		Height(m.aiViewport.Height + 2).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Foreground(focusColor).Bold(true).Render("AI AGENT OUTPUT"),
				m.aiViewport.View(),
			),
		)

	inputBox := inputBoxStyle.
		Width(m.width - 2).
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("240")).Render("AI Instructions:"),
				m.input.View(),
			),
		)

	helpBar := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width).
		Align(lipgloss.Center).
		Render("[tab] Focus   [enter] Execute   [esc] Close")

	content := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(title),
		sideBySide,
		aiBox,
		inputBox,
		helpBar,
	)

	return lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Width(m.width).
		Height(m.height).
		Render(content)
}

func GetLatestSessionID() string {
	dir := "/tmp/autoscout"
	files, err := os.ReadDir(dir)
	if err != nil || len(files) == 0 {
		return ""
	}

	var sessions []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".request") {
			sessions = append(sessions, strings.TrimSuffix(f.Name(), ".request"))
		}
	}
	sort.Strings(sessions)
	if len(sessions) > 0 {
		return sessions[len(sessions)-1]
	}
	return ""
}
