package gui

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HaythmKenway/autoscout/pkg/ai"
	"github.com/HaythmKenway/autoscout/pkg/burp"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/reflow/wrap"
)

type TriggerManualMsg struct {
	SessionID string
	TargetURL string
}

type AIResultMsg struct {
	Plan *ai.AIPlan
	Err  error
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
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
	spinner     spinner.Model
	width       int
	height      int
	theme       Theme
	active      bool
	sessionID   string
	targetURL   string
	thinking    string
	focus       manualFocus
	zm          *zone.Manager

	// Raw data for re-wrapping
	rawReq string
	rawRes string
	rawAI  string
	chat   []ChatMessage
}

func NewManualOverlayModel(w, h int, zm *zone.Manager) manualOverlayModel {
	rvp := viewport.New(1, 1)
	svp := viewport.New(1, 1)
	avp := viewport.New(1, 1)

	ti := textinput.New()
	ti.Placeholder = "Enter AI instructions..."
	ti.Focus()
	ti.Prompt = " AI > "

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ModernTheme.Accent)

	m := manualOverlayModel{
		reqViewport: rvp,
		resViewport: svp,
		aiViewport:  avp,
		input:       ti,
		spinner:     s,
		width:       w,
		height:      h,
		theme:       ModernTheme,
		active:      false,
		focus:       focusInput,
		zm:          zm,
		rawReq:      "No request data.",
		rawRes:      "No response data.",
		rawAI:       "Ready for investigation.",
		chat:        []ChatMessage{},
	}
	m.rebuildViewports()
	return m
}

func (m *manualOverlayModel) SetSession(id string) {
	m.sessionID = id
	m.targetURL = ""
	
	// Determine directory
	var dir string
	if burp.CurrentSession != "" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), burp.CurrentSession)
	} else {
		dir = "/tmp/autoscout"
	}

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
		m.rawRes = "No response data available."
	}
	
	// Load Chat History
	m.chat = []ChatMessage{}
	chatPath := filepath.Join(dir, id+".chat.json")
	if chatData, err := os.ReadFile(chatPath); err == nil {
		json.Unmarshal(chatData, &m.chat)
	}

	m.rebuildViewports()
}

func (m *manualOverlayModel) SetTarget(url string) {
	m.targetURL = url
	m.sessionID = ""
	m.rawReq = fmt.Sprintf("Target: %s\n\nAwaiting traffic...", url)
	m.rawRes = "N/A"
	m.chat = []ChatMessage{}
	m.rebuildViewports()
}

func (m *manualOverlayModel) rebuildViewports() {
	// 1. Calculate Modal Size (Centered 95% Box)
	modalW := int(float64(m.width) * 0.95)
	modalH := int(float64(m.height) * 0.9)
	if modalW < 40 { modalW = m.width }
	if modalH < 15 { modalH = m.height }

	// 2. Inner budget (Subtract 2 for the single outer border)
	innerW := modalW - 2
	innerH := modalH - 2

	primaryH := innerH - 5
	if primaryH < 5 { primaryH = 5 }

	// 4. Update Viewports
	m.reqViewport.Width = innerW
	m.reqViewport.Height = primaryH
	m.reqViewport.SetContent(m.highlightHTTP(m.rawReq, innerW))

	m.resViewport.Width = innerW
	m.resViewport.Height = primaryH
	m.resViewport.SetContent(m.highlightHTTP(m.rawRes, innerW))

	// Render Chat in AI Viewport
	var sb strings.Builder
	if len(m.chat) == 0 {
		sb.WriteString("Ready for investigation.")
	} else {
		for _, msg := range m.chat {
			roleStyle := lipgloss.NewStyle().Bold(true)
			if msg.Role == "user" {
				roleStyle = roleStyle.Foreground(lipgloss.Color("2"))
				sb.WriteString(roleStyle.Render("YOU > ") + "\n")
			} else {
				roleStyle = roleStyle.Foreground(m.theme.Accent)
				sb.WriteString(roleStyle.Render("AI > ") + "\n")
			}
			sb.WriteString(wrap.String(msg.Content, innerW) + "\n\n")
		}
	}

	m.aiViewport.Width = innerW
	m.aiViewport.Height = primaryH
	m.aiViewport.SetContent(sb.String())
	m.aiViewport.GotoBottom()

	m.input.Width = innerW - 10
}

func (m *manualOverlayModel) highlightHTTP(raw string, width int) string {
	if width < 5 { return raw }
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	var highlighted []string
	
	headerStyle := lipgloss.NewStyle().Foreground(m.theme.Accent).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(m.theme.Accent).Padding(0, 1).Bold(true)
	
	bodyStarted := false
	for i, line := range lines {
		if strings.TrimSpace(line) == "" && !bodyStarted {
			bodyStarted = true
			highlighted = append(highlighted, "")
			continue
		}

		if bodyStarted {
			// Body content - simple wrap
			highlighted = append(highlighted, wrap.String(line, width))
			continue
		}

		if i == 0 {
			// Request line / Status line
			txt := strings.TrimSpace(line)
			if len(txt) > width { txt = txt[:width-3] + "..." }
			highlighted = append(highlighted, methodStyle.Render(txt))
			continue
		}

		if idx := strings.Index(line, ": "); idx != -1 {
			// Header line - Wrap and style
			k := line[:idx+2]
			v := line[idx+2:]
			styled := headerStyle.Render(k) + valueStyle.Render(v)
			highlighted = append(highlighted, wrap.String(styled, width))
		} else {
			highlighted = append(highlighted, wrap.String(valueStyle.Render(line), width))
		}
	}
	return strings.Join(highlighted, "\n")
}

func (m manualOverlayModel) Update(msg tea.Msg) (manualOverlayModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.rebuildViewports()
	case spinner.TickMsg:
		if m.thinking != "" {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	case AIResultMsg:
		m.thinking = ""
		content := ""
		if msg.Err != nil {
			content = fmt.Sprintf("AI ERROR: %v", msg.Err)
		} else {
			var sb strings.Builder
			sb.WriteString("THINKING:\n" + msg.Plan.Thinking + "\n\nACTIONS:\n")
			if len(msg.Plan.Actions) == 0 { sb.WriteString("- No automated tools recommended.") }
			for _, act := range msg.Plan.Actions { sb.WriteString(fmt.Sprintf("- Run %s on %s\n", act.Tool, act.Target)) }
			content = sb.String()
		}
		m.chat = append(m.chat, ChatMessage{Role: "ai", Content: content})
		m.saveChatHistory()
		m.rebuildViewports()
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if m.zm.Get("tab-req").InBounds(msg) { m.focus = focusReq; m.input.Blur() }
			if m.zm.Get("tab-res").InBounds(msg) { m.focus = focusRes; m.input.Blur() }
			if m.zm.Get("tab-ai").InBounds(msg) { m.focus = focusAI; m.input.Blur() }
			if m.zm.Get("modal-input").InBounds(msg) { m.focus = focusInput; m.input.Focus() }
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": m.active = false; return m, nil
		case "tab":
			m.focus = (m.focus + 1) % 4
			if m.focus == focusInput { m.input.Focus() } else { m.input.Blur() }
		case "enter":
			if m.focus == focusInput && m.input.Value() != "" && m.thinking == "" {
				m.thinking = "AI is analyzing..."
				instruction := m.input.Value()
				m.input.Reset()
				m.focus = focusAI // Auto-switch to AI panel
				m.input.Blur()
				
				m.chat = append(m.chat, ChatMessage{Role: "user", Content: instruction})
				m.saveChatHistory()
				m.rebuildViewports()
				
				return m, tea.Batch(m.executeAI(instruction), m.spinner.Tick)
			}
		}
	}
	
	if m.focus == focusInput {
		m.input, cmd = m.input.Update(msg)
	} else if m.focus == focusReq {
		m.reqViewport, cmd = m.reqViewport.Update(msg)
	} else if m.focus == focusRes {
		m.resViewport, cmd = m.resViewport.Update(msg)
	} else if m.focus == focusAI {
		m.aiViewport, cmd = m.aiViewport.Update(msg)
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *manualOverlayModel) saveChatHistory() {
	if m.sessionID == "" { return }
	var dir string
	if burp.CurrentSession != "" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), burp.CurrentSession)
	} else {
		dir = "/tmp/autoscout"
	}
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(m.chat, "", "  ")
	os.WriteFile(filepath.Join(dir, m.sessionID+".chat.json"), data, 0644)
}

func (m manualOverlayModel) executeAI(instruction string) tea.Cmd {
	return func() tea.Msg {
		agent := ai.LoadBackend()
		req := burp.BurpRequest{URL: m.targetURL, Tool: "MANUAL_GUI", UserContext: instruction}
		if m.sessionID != "" {
			var dir string
			if burp.CurrentSession != "" {
				dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), burp.CurrentSession)
			} else {
				dir = "/tmp/autoscout"
			}
			reqD, _ := os.ReadFile(filepath.Join(dir, m.sessionID+".request"))
			req.Body = base64.StdEncoding.EncodeToString(reqD)
			if resD, err := os.ReadFile(filepath.Join(dir, m.sessionID+".response")); err == nil {
				req.ResponseStatus = 200
				req.ResponseBody = base64.StdEncoding.EncodeToString(resD)
			}
		}
		plan, err := agent.Analyze(req)
		return AIResultMsg{Plan: plan, Err: err}
	}
}

func (m manualOverlayModel) View() string {
	if !m.active { return "" }
	
	bgCol := lipgloss.Color("0") 
	accent := m.theme.Accent
	dimFG := lipgloss.Color("240")

	// 1. Modal Dimensions
	modalW := int(float64(m.width) * 0.95)
	modalH := int(float64(m.height) * 0.9)
	if modalW < 40 { modalW = m.width }
	if modalH < 15 { modalH = m.height }
	innerW := modalW - 2
	innerH := modalH - 2

	// 2. Component Styles
	baseStyle := lipgloss.NewStyle().Width(innerW).Background(bgCol)
	
	// Tab Bar
	tabStyle := lipgloss.NewStyle().Padding(0, 2).Background(lipgloss.Color("235")).Foreground(lipgloss.Color("245"))
	activeTabStyle := tabStyle.Copy().Background(accent).Foreground(lipgloss.Color("0")).Bold(true)

	reqTab := m.zm.Mark("tab-req", tabStyle.Render(" REQUEST "))
	resTab := m.zm.Mark("tab-res", tabStyle.Render(" RESPONSE "))
	aiTab := m.zm.Mark("tab-ai", tabStyle.Render(" AI ANALYSIS "))
	
	if m.focus == focusReq { reqTab = m.zm.Mark("tab-req", activeTabStyle.Render(" REQUEST ")) }
	if m.focus == focusRes { resTab = m.zm.Mark("tab-res", activeTabStyle.Render(" RESPONSE ")) }
	if m.focus == focusAI { aiTab = m.zm.Mark("tab-ai", activeTabStyle.Render(" AI ANALYSIS ")) }
	
	tabBar := baseStyle.Copy().Background(lipgloss.Color("234")).Render(lipgloss.JoinHorizontal(lipgloss.Top, reqTab, resTab, aiTab))

	// 3. Primary Content
	var primaryView string
	switch m.focus {
	case focusRes:
		primaryView = m.resViewport.View()
	case focusAI:
		primaryView = m.aiViewport.View()
		if m.thinking != "" {
			spinnerStr := lipgloss.NewStyle().Foreground(accent).Padding(1, 2).Render(m.spinner.View() + " " + m.thinking)
			primaryView = lipgloss.JoinVertical(lipgloss.Left, primaryView, spinnerStr)
		}
	default:
		primaryView = m.reqViewport.View()
	}

	// 4. Separator
	sepStyle := lipgloss.NewStyle().Foreground(m.theme.BorderColor).Width(innerW)
	separator := sepStyle.Render(strings.Repeat("─", innerW))

	// 5. Input Prompt
	inputPrompt := lipgloss.NewStyle().Foreground(dimFG).PaddingLeft(1).Render("AI Instructions:")
	if m.focus == focusInput {
		inputPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).PaddingLeft(1).Render("AI Instructions:")
	}
	inputSection := m.zm.Mark("modal-input", lipgloss.JoinVertical(lipgloss.Left, 
		separator,
		inputPrompt, 
		m.input.View(),
	))

	// 6. Help Footer
	help := baseStyle.Copy().Foreground(dimFG).Align(lipgloss.Center).Render(" [tab] Cycle focus  [enter] Execute  [esc] Close ")

	// 7. Assemble Modal Content
	modalContent := lipgloss.JoinVertical(lipgloss.Left,
		tabBar,
		primaryView,
		inputSection,
		help,
	)

	// 8. Wrap in a SINGLE border
	modalStyle := lipgloss.NewStyle().
		Border(m.theme.Border).
		BorderForeground(accent).
		Background(bgCol).
		Width(innerW).
		Height(innerH)
	
	if m.focus == focusAI { modalStyle = modalStyle.BorderForeground(lipgloss.Color("5")) }
	if m.focus == focusInput { modalStyle = modalStyle.BorderForeground(lipgloss.Color("2")) }

	renderedModal := modalStyle.Render(modalContent)

	// 9. Centered Overlay over dimmed background
	return lipgloss.Place(m.width, m.height, 
		lipgloss.Center, lipgloss.Center, 
		renderedModal,
		lipgloss.WithWhitespaceChars("▒"),
		lipgloss.WithWhitespaceForeground(dimFG),
	)
}

func GetLatestSessionID() string {
	var dir string
	if burp.CurrentSession != "" {
		dir = filepath.Join(os.ExpandEnv("$HOME/.autoscout/sessions"), burp.CurrentSession)
	} else {
		dir = "/tmp/autoscout"
	}
	files, _ := os.ReadDir(dir)
	var s []string
	for _, f := range files { if strings.HasSuffix(f.Name(), ".request") { s = append(s, strings.TrimSuffix(f.Name(), ".request")) } }
	sort.Strings(s)
	if len(s) > 0 { return s[len(s)-1] }
	return ""
}
