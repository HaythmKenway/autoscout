package gui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"gopkg.in/yaml.v3"
)

type settingsModel struct {
	form         *huh.Form
	discordModel *Discord
}

type Discord struct {
	ID                string `yaml:"id"`
	DiscordChannel    string `yaml:"discord_channel"`
	DiscordUsername   string `yaml:"discord_username"`
	DiscordFormat     string `yaml:"discord_format"`
	DiscordWebhookURL string `yaml:"discord_webhook_url"`
}

type Settings struct {
	Agent string `yaml:"agent"`
	Data  string `yaml:"data"`
}

type DiscordConfig struct {
	Discord []Discord `yaml:"discord"`
}

type SettingsConfig struct {
	Settings Settings `yaml:"settings"`
}

func NewSettingsModel(width int, height int) settingsModel {
	discordPath := os.ExpandEnv("$HOME/.config/notify/provider-config.yaml")
	settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")

	discordConfig, err := readDiscordConfig(discordPath)
	if err != nil {
		// Log error but continue with empty config
		// log.Fatalf("Error reading discord config: %v", err)
		discordConfig = &DiscordConfig{}
	}

	settingsConfig, err := readSettingsConfig(settingsPath)
	if err != nil {
		settingsConfig = &SettingsConfig{}
	}

	discordModel := &Discord{}
	if len(discordConfig.Discord) > 0 {
		discordModel = &discordConfig.Discord[0]
	}

	settings := settingsConfig.Settings

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("agent").
				Options(huh.NewOptions("Gemini", "Ollama", "Skibbidi")...).
				Title("Choose your model").
				Description("Make sure to go bankrupt").
				Value(&settings.Agent),
			huh.NewSelect[string]().
				Key("data").
				Options(huh.NewOptions("yes ofcourse!", "yes")...).
				Title("Sell your data").
				Value(&settings.Data),
			huh.NewInput().Key("Did").Title("Bot Id").Value(&discordModel.ID),
			huh.NewInput().Key("Dchannel").Title("Discord channel").Value(&discordModel.DiscordChannel),
			huh.NewInput().Key("Dame").Title("Discord Username").Value(&discordModel.DiscordUsername),
			huh.NewInput().Key("Dformat").Title("Discord Text format").Value(&discordModel.DiscordFormat),
			huh.NewInput().Key("Dwebhook").Title("Discord Webhookurl").Value(&discordModel.DiscordWebhookURL),
		),
	).WithWidth(width - 5).WithHeight(height - 2) // Adjusted margins

	return settingsModel{
		form:         form,
		discordModel: discordModel,
	}
}

func (m settingsModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Make form responsive
		m.form = m.form.WithWidth(msg.Width - 5).WithHeight(msg.Height - 2)
	}

	updatedForm, formCmd := m.form.Update(msg)
	if f, ok := updatedForm.(*huh.Form); ok {
		m.form = f
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Only save if a field is focused
			if m.form.GetFocusedField() != nil {
				key := m.form.GetFocusedField().GetKey()
				value := m.form.GetFocusedField().GetValue()

				var valueStr string
				switch v := value.(type) {
				case string:
					valueStr = v
				case int:
					valueStr = fmt.Sprintf("%d", v)
				case float64:
					valueStr = fmt.Sprintf("%f", v)
				default:
					valueStr = fmt.Sprintf("%v", v)
				}
				storetodb(m.discordModel, key, valueStr)
			}
		}
	}

	return m, formCmd
}

func (m settingsModel) View() string {
	return m.form.View()
}

func storetodb(discord *Discord, key string, value string) {
	switch key {
	case "Did", "Dchannel", "Dame", "Dformat", "Dwebhook":
		updateDiscordConfig(discord, key, value)
	case "agent", "data":
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
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		localUtils.Logger(fmt.Sprintf("Error creating config directory: %v", err), 1)
		return
	}

	if err := writeDiscordConfig(path, config); err != nil {
		localUtils.Logger(fmt.Sprintf("Error writing Discord config: %v", err), 1)
	}
}

func updateSettingsConfig(key string, value string) {
	filePath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
	dirPath := filepath.Dir(filePath)

	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		localUtils.Logger(fmt.Sprintf("Error creating settings directory: %v", err), 1)
		return
	}

	config, err := readSettingsConfig(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			config = &SettingsConfig{}
		} else {
			// Don't log read errors if file is just empty/corrupt, reset instead
			config = &SettingsConfig{}
		}
	}

	switch key {
	case "agent":
		config.Settings.Agent = value
	case "data":
		config.Settings.Data = value
	}

	if err := writeSettingsConfig(filePath, config); err != nil {
		localUtils.Logger(fmt.Sprintf("Error writing Settings config: %v", err), 1)
	}
}

func readDiscordConfig(filePath string) (*DiscordConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var config DiscordConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func readSettingsConfig(filePath string) (*SettingsConfig, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var config SettingsConfig
	err = yaml.Unmarshal(data, &config)
	return &config, err
}

func writeDiscordConfig(filePath string, config *DiscordConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}

func writeSettingsConfig(filePath string, config *SettingsConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, os.ModePerm)
}
