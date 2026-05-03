package gui

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

type settingsCategory int

const (
	CatUI settingsCategory = iota
	CatAI
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
	
	// Discord Settings
	discordInputs []textinput.Model
	
	width        int
	height       int
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
	if userSettings.Agent == "" { userSettings.Agent = "Gemini" }
	if userSettings.ProxyURL == "" { userSettings.ProxyURL = "http://127.0.0.1:8080" }

	// UI Options
	themes := []string{"Modern", "Neon", "Matrix"}
	tCursor := 0
	for i, t := range themes {
		if t == userSettings.Theme {
			tCursor = i
			break
		}
	}

	// AI Options
	agents := []string{"Gemini", "Ollama", "Codex", "Skibbidi"}
	aCursor := 0
	for i, a := range agents {
		if a == userSettings.Agent {
			aCursor = i
			break
		}
	}

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

	return settingsModel{
		activeCat:     CatUI,
		themeOptions:  themes,
		themeCursor:   tCursor,
		agentOptions:  agents,
		agentCursor:   aCursor,
		discordInputs: []textinput.Model{di, pi},
		width:         width,
		height:        height,
		discordModel:  discordModel,
		userSettings:  userSettings,
	}
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
			case CatDiscord:
				if msg.String() == "esc" {
					m.focusEditor = false
					m.discordInputs[0].Blur()
				}
				m.discordInputs[0], cmd = m.discordInputs[0].Update(msg)
				cmds = append(cmds, cmd)
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
	m.discordModel.DiscordWebhookURL = m.discordInputs[0].Value()

	// Persist to Disk
	storetodb(m.discordModel, "Dwebhook", m.discordModel.DiscordWebhookURL)
	updateSettingsConfig("theme", m.userSettings.Theme)
	updateSettingsConfig("agent", m.userSettings.Agent)
	updateSettingsConfig("proxy", m.userSettings.ProxyURL)
	localUtils.Logger("Settings persisted to disk", 1)
}

func (m settingsModel) View() string {
	catWidth := 18
	editorWidth := m.width - catWidth - 4
	if editorWidth < 0 { editorWidth = 0 }

	// Categories List
	categories := []string{" UI / Appearance ", " AI Fleet Mode ", " Discord / Notify "}
	var catViews []string
	for i, cat := range categories {
		style := lipgloss.NewStyle().Padding(0, 1).MarginBottom(1)
		if settingsCategory(i) == m.activeCat {
			if !m.focusEditor {
				style = style.Background(lipgloss.Color("5")).Foreground(lipgloss.Color("7")).Bold(true)
			} else {
				style = style.Background(lipgloss.Color("8")).Foreground(lipgloss.Color("7"))
			}
		} else {
			style = style.Foreground(lipgloss.Color("240"))
		}
		catViews = append(catViews, style.Width(catWidth).Render(cat))
	}
	catList := lipgloss.JoinVertical(lipgloss.Left, catViews...)

	// Editor Area
	var editorContent string
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5")).MarginBottom(1)
	
	switch m.activeCat {
	case CatUI:
		var opts []string
		for i, opt := range m.themeOptions {
			prefix := "[ ] "
			style := lipgloss.NewStyle()
			if i == m.themeCursor {
				prefix = "[*] "
				if m.focusEditor {
					style = style.Foreground(lipgloss.Color("5")).Bold(true)
				}
			}
			opts = append(opts, style.Render(prefix+opt))
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
					style = style.Foreground(lipgloss.Color("5")).Bold(true)
				}
			}
			opts = append(opts, style.Render(prefix+opt))
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
	case CatDiscord:
		editorContent = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render("DISCORD CONFIGURATION"),
			"Webhook URL:",
			m.discordInputs[0].View(),
		)
	}

	editorStyle := lipgloss.NewStyle().
		Width(editorWidth).
		Height(m.height - 4).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("237"))

	if m.focusEditor {
		editorStyle = editorStyle.BorderForeground(lipgloss.Color("5"))
	}

	editor := editorStyle.Render(editorContent)

	// Save Banner
	footer := ""
	if m.saveMessage != "" {
		footer = "\n" + lipgloss.NewStyle().
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("0")).
			Bold(true).
			Padding(0, 2).
			Render(m.saveMessage)
	} else {
		footer = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" [Tab] Switch Zone  [Ctrl+S] Save  [Esc] Back")
	}

	main := lipgloss.JoinHorizontal(lipgloss.Top, catList, editor)
	return main + footer
}

func storetodb(discord *Discord, key string, value string) {
	switch key {
	case "Did", "Dchannel", "Dame", "Dformat", "Dwebhook":
		updateDiscordConfig(discord, key, value)
	case "agent", "data", "theme":
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
}

type DiscordConfig struct {
	Discord []Discord `yaml:"discord"`
}

type SettingsConfig struct {
	Settings Settings `yaml:"settings"`
}
