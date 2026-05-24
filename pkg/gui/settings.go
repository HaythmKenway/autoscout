package gui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
	"gopkg.in/yaml.v3"
)

type settingsCategory int

const (
	CatUI settingsCategory = iota
	CatAI
	CatWordlists
	CatDiscord
)

type settingsModel struct {
	activeCat    settingsCategory
	focusEditor  bool
	
	// UI Settings
	themeOptions []string
	themeCursor  int
	
	// AI Settings
	agentOptions []string
	agentCursor  int
	
	// Wordlist Settings
	wordlistCursor int
	wordlistInputs []textinput.Model
	
	// Discord Settings
	discordInputs []textinput.Model
	
	width        int
	height       int
	theme        Theme
	discordModel *Discord
	userSettings *Settings
	saveMessage  string
	saveTimer    *time.Timer
}

func NewSettingsModel(width int, height int) settingsModel {
	discordPath := os.ExpandEnv("$HOME/.config/notify/provider-config.yaml")
	settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")

	discordConfig, _ := readDiscordConfig(discordPath)
	if discordConfig == nil { discordConfig = &DiscordConfig{} }
	settingsConfig, _ := readSettingsConfig(settingsPath)
	if settingsConfig == nil { settingsConfig = &SettingsConfig{} }

	discordModel := &Discord{}
	if len(discordConfig.Discord) > 0 {
		discordModel = &discordConfig.Discord[0]
	}

	userSettings := &settingsConfig.Settings
	if userSettings.Theme == "" { userSettings.Theme = "Modern" }
	if userSettings.Agent == "" { userSettings.Agent = "ClaudeCode" }
	if userSettings.ProxyURL == "" { userSettings.ProxyURL = "http://127.0.0.1:8080" }
	if userSettings.FuzzWordlist == "" { userSettings.FuzzWordlist = "/usr/share/wordlists/dirb/common.txt" }
	if userSettings.ParamWordlist == "" { userSettings.ParamWordlist = "/usr/share/wordlists/seclists/Discovery/Web-Content/burp-parameter-names.txt" }

	// UI Options
	themes := []string{"Modern", "Cyberpunk", "Matrix"}
	tCursor := 0
	for i, t := range themes {
		if t == userSettings.Theme {
			tCursor = i
			break
		}
	}

	// AI Options
	agents := []string{"ClaudeCode", "Claude", "Gemini", "Ollama", "Codex"}
	aCursor := 0
	for i, a := range agents {
		if a == userSettings.Agent {
			aCursor = i
			break
		}
	}

	// Wordlist Inputs
	fi := textinput.New()
	fi.Placeholder = "/usr/share/wordlists/..."
	fi.SetValue(userSettings.FuzzWordlist)
	fi.CharLimit = 256
	fi.Width = 50

	pi2 := textinput.New()
	pi2.Placeholder = "/usr/share/wordlists/..."
	pi2.SetValue(userSettings.ParamWordlist)
	pi2.CharLimit = 256
	pi2.Width = 50

	// Discord Inputs
	di := textinput.New()
	di.Placeholder = "Webhook URL"
	di.SetValue(discordModel.DiscordWebhookURL)
	di.CharLimit = 256
	di.Width = 40

	// Proxy Input
	pi := textinput.New()
	pi.Placeholder = "http://127.0.0.1:8080"
	pi.SetValue(userSettings.ProxyURL)
	pi.CharLimit = 128
	pi.Width = 30

	// Rate Limit Input
	ri := textinput.New()
	ri.Placeholder = "5"
	if userSettings.RateLimit == "" { userSettings.RateLimit = "5" }
	ri.SetValue(userSettings.RateLimit)
	ri.CharLimit = 10
	ri.Width = 10

	return settingsModel{
		activeCat:     CatUI,
		themeOptions:  themes,
		themeCursor:   tCursor,
		agentOptions:  agents,
		agentCursor:   aCursor,
		wordlistInputs: []textinput.Model{fi, pi2},
		discordInputs: []textinput.Model{di, pi, ri},
		width:         width,
		height:        height,
		theme:         ModernTheme,
		discordModel:  discordModel,
		userSettings:  userSettings,
	}
}

func (m settingsModel) vw(p float64) int {
	return int(float64(m.width) * p / 100.0)
}

func (m settingsModel) vh(p float64) int {
	return int(float64(m.height) * p / 100.0)
}

func (m settingsModel) Init() tea.Cmd {
	return nil
}

type hideSaveMsg struct{}
func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case hideSaveMsg:
		m.saveMessage = ""

	case tea.KeyMsg:
		// Navigation between categories
		if !m.focusEditor {
			switch msg.String() {
			case "up", "k":
				if m.activeCat > 0 { m.activeCat-- }
			case "down", "j":
				if m.activeCat < CatDiscord { m.activeCat++ }
			case "right", "l", "enter", "tab":
				m.focusEditor = true
				if m.activeCat == CatDiscord {
					m.discordInputs[0].Focus()
				} else if m.activeCat == CatWordlists {
					m.wordlistInputs[m.wordlistCursor].Focus()
				}
			}
		} else {
			// Inside Editor
			switch m.activeCat {
			case CatUI:
				switch msg.String() {
				case "up", "k":
					if m.themeCursor > 0 { m.themeCursor-- }
				case "down", "j":
					if m.themeCursor < len(m.themeOptions)-1 { m.themeCursor++ }
				case "left", "h", "esc":
					m.focusEditor = false
				}
			case CatAI:
				switch msg.String() {
				case "up", "k":
					if m.agentCursor > 0 { m.agentCursor-- }
				case "down", "j":
					if m.agentCursor < len(m.agentOptions)-1 { m.agentCursor++ }
				case "left", "h", "esc":
					m.focusEditor = false
				}
			case CatWordlists:
				switch msg.String() {
				case "tab":
					m.wordlistInputs[m.wordlistCursor].Blur()
					m.wordlistCursor = (m.wordlistCursor + 1) % len(m.wordlistInputs)
					m.wordlistInputs[m.wordlistCursor].Focus()
				case "esc":
					m.focusEditor = false
					m.wordlistInputs[m.wordlistCursor].Blur()
				default:
					m.wordlistInputs[m.wordlistCursor], cmd = m.wordlistInputs[m.wordlistCursor].Update(msg)
					cmds = append(cmds, cmd)
				}
			case CatDiscord:
				switch msg.String() {
				case "tab":
					m.discordInputs[m.wordlistCursor].Blur()
					m.wordlistCursor = (m.wordlistCursor + 1) % len(m.discordInputs)
					m.discordInputs[m.wordlistCursor].Focus()
				case "esc":
					m.focusEditor = false
					m.discordInputs[m.wordlistCursor].Blur()
				default:
					m.discordInputs[m.wordlistCursor], cmd = m.discordInputs[m.wordlistCursor].Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}

		// Global Hotkeys
		switch msg.String() {
		case "ctrl+s":
			m.save()
			m.saveMessage = " CONFIG SAVED! "
			return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
				return hideSaveMsg{}
			})
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *settingsModel) save() {
	// Sync UI State to UserSettings
	m.userSettings.Theme = m.themeOptions[m.themeCursor]
	m.userSettings.Agent = m.agentOptions[m.agentCursor]
	m.userSettings.ProxyURL = m.discordInputs[1].Value()
	m.userSettings.RateLimit = m.discordInputs[2].Value()
	m.userSettings.FuzzWordlist = m.wordlistInputs[0].Value()
	m.userSettings.ParamWordlist = m.wordlistInputs[1].Value()
	m.discordModel.DiscordWebhookURL = m.discordInputs[0].Value()

	// Persist to Disk
	storetodb(m.discordModel, "Dwebhook", m.discordModel.DiscordWebhookURL)
	updateSettingsConfig("theme", m.userSettings.Theme)
	updateSettingsConfig("agent", m.userSettings.Agent)
	updateSettingsConfig("proxy", m.userSettings.ProxyURL)
	updateSettingsConfig("rate_limit", m.userSettings.RateLimit)
	updateSettingsConfig("wordlist", m.userSettings.FuzzWordlist)
	updateSettingsConfig("param_wordlist", m.userSettings.ParamWordlist)
	localUtils.Logger("Settings persisted to disk", 1)
}

func (m settingsModel) View(zm *zone.Manager) string {
	catWidth := 22
	editorWidth := m.width - catWidth
	if editorWidth < 0 { editorWidth = 0 }

	// Categories List
	categories := []string{" UI / Appearance ", " AI Fleet Mode ", " Wordlists ", " Discord / Notify "}
	var catViews []string
	for i, cat := range categories {
		style := lipgloss.NewStyle().Padding(0, 1).MarginBottom(1)
		if settingsCategory(i) == m.activeCat {
			if !m.focusEditor {
				style = style.Background(m.theme.Accent).Foreground(lipgloss.Color("#ffffff")).Bold(true)
			} else {
				style = style.Background(lipgloss.Color("237")).Foreground(m.theme.Accent)
			}
		} else {
			style = style.Foreground(lipgloss.Color("240"))
		}
		catViews = append(catViews, zm.Mark(fmt.Sprintf("set-cat-%d", i), style.Width(catWidth).Render(cat)))
	}
	catList := lipgloss.JoinVertical(lipgloss.Left, catViews...)

	// Editor Area
	var editorContent string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Accent).MarginBottom(1)
	
	switch m.activeCat {
	case CatUI:
		var opts []string
		for i, opt := range m.themeOptions {
			prefix := "[ ] "
			style := lipgloss.NewStyle()
			if i == m.themeCursor {
				prefix = "[*] "
				if m.focusEditor {
					style = style.Foreground(m.theme.Accent).Bold(true)
				}
			}
			opts = append(opts, zm.Mark(fmt.Sprintf("set-opt-theme-%d", i), style.Render(prefix+opt)))
		}
		editorContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("SELECT THEME"),
			lipgloss.JoinVertical(lipgloss.Left, opts...),
		)
	case CatAI:
		var opts []string
		for i, opt := range m.agentOptions {
			prefix := "( ) "
			style := lipgloss.NewStyle()
			if i == m.agentCursor {
				prefix = "(*) "
				if m.focusEditor {
					style = style.Foreground(m.theme.Accent).Bold(true)
				}
			}
			opts = append(opts, zm.Mark(fmt.Sprintf("set-opt-agent-%d", i), style.Render(prefix+opt)))
		}
		editorContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("SELECT AI AGENT BACKEND"),
			lipgloss.JoinVertical(lipgloss.Left, opts...),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Active Fleet:"),
			" - ParameterFuzzer",
			" - APIAnalyzer",
			" - InfoLeakScanner",
			" - DeepScanner",
		)
	case CatWordlists:
		m.wordlistInputs[0].Width = editorWidth - 4
		m.wordlistInputs[1].Width = editorWidth - 4
		editorContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("WORDLIST CONFIGURATION"),
			"Fuzzing / Directory Wordlist:",
			m.wordlistInputs[0].View(),
			"",
			"Parameter Discovery Wordlist:",
			m.wordlistInputs[1].View(),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" [Tab] Cycle Inputs"),
		)
	case CatDiscord:
		m.discordInputs[0].Width = editorWidth - 4
		m.discordInputs[1].Width = editorWidth - 4
		m.discordInputs[2].Width = editorWidth - 4
		editorContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("SYSTEM CONFIGURATION"),
			"Discord Webhook URL:",
			m.discordInputs[0].View(),
			"",
			"Global Proxy URL:",
			m.discordInputs[1].View(),
			"",
			"Global Rate Limit (req/s):",
			m.discordInputs[2].View(),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" [Tab] Cycle Inputs"),
		)
	}

	editorStyle := lipgloss.NewStyle().
		Width(editorWidth - 2).
		Height(m.height - 2).
		Padding(0, 1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("237"))

	if m.focusEditor {
		editorStyle = editorStyle.BorderForeground(m.theme.Accent)
	}

	editor := editorStyle.Render(editorContent)

	main := lipgloss.JoinHorizontal(lipgloss.Top, catList, editor)
	return main
}

func storetodb(discord *Discord, key string, value string) {
	switch key {
	case "Did", "Dchannel", "Dame", "Dformat", "Dwebhook":
		updateDiscordConfig(discord, key, value)
	case "agent", "data", "theme", "wordlist", "param_wordlist":
		updateSettingsConfig(key, value)
	}
}

func updateDiscordConfig(discord *Discord, key string, value string) {
	switch key {
	case "Did":
		discord.ID = value
	case "Dchannel":
		discord.DiscordChannel = value
	case "Dame":
		discord.DiscordUsername = value
	case "Dformat":
		discord.DiscordFormat = value
	case "Dwebhook":
		discord.DiscordWebhookURL = value
	}

	config := &DiscordConfig{Discord: []Discord{*discord}}

	path := os.ExpandEnv("$HOME/.config/notify/provider-config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return
	}
	writeDiscordConfig(path, config)
}

func updateSettingsConfig(key string, value string) {
	filePath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return
	}

	config, err := readSettingsConfig(filePath)
	if err != nil { config = &SettingsConfig{} }

	switch key {
	case "agent":
		config.Settings.Agent = value
	case "theme":
		config.Settings.Theme = value
	case "proxy":
		config.Settings.ProxyURL = value
	case "rate_limit":
		config.Settings.RateLimit = value
	case "wordlist":
		config.Settings.FuzzWordlist = value
	case "param_wordlist":
		config.Settings.ParamWordlist = value
	}

	writeSettingsConfig(filePath, config)
}

func readDiscordConfig(filePath string) (*DiscordConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil { return nil, err }
	var config DiscordConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func readSettingsConfig(filePath string) (*SettingsConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil { return nil, err }
	var config SettingsConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func writeDiscordConfig(filePath string, config *DiscordConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil { return err }
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

func writeSettingsConfig(filePath string, config *SettingsConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil { return err }
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

type Discord struct {
	ID                string `yaml:"id"`
	DiscordChannel    string `yaml:"discord_channel"`
	DiscordUsername   string `yaml:"discord_username"`
	DiscordFormat     string `yaml:"discord_format"`
	DiscordWebhookURL string `yaml:"discord_webhook_url"`
}

type Settings struct {
	Agent         string `yaml:"agent"`
	Data          string `yaml:"data"`
	Theme         string `yaml:"theme"`
	ProxyURL      string `yaml:"proxy_url"`
	FuzzWordlist  string `yaml:"wordlist_path"`
	ParamWordlist string `yaml:"param_wordlist_path"`
	RateLimit     string `yaml:"rate_limit"`
}

type DiscordConfig struct {
	Discord []Discord `yaml:"discord"`
}

type SettingsConfig struct {
	Settings Settings `yaml:"settings"`
}
