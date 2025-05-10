package gui

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

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

type Config struct {
	Discord []Discord `yaml:"discord"`
}

func NewSettingsModel(width int, height int) settingsModel {
	filePath := os.ExpandEnv("$HOME/.config/notify/provider-config.yaml")
	config, err := readConfig(filePath)
	fmt.Print(config)
	if err != nil {
		log.Fatalf("Error reading config: %v", err)
	}

	discordModel := &Discord{}
	if len(config.Discord) > 0 {
		discordModel = &config.Discord[0]
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("agent").
				Options(huh.NewOptions("Gemini", "Ollama", "Skibbidi")...).
				Title("Choose your model").
				Description("Make sure to go bankrupt"),
			huh.NewSelect[string]().
				Key("data").
				Options(huh.NewOptions("yes ofcourse!", "yes")...).
				Title("Sell your data"),
			huh.NewInput().Key("Did").Title("Bot Id").Value(&discordModel.ID),
			huh.NewInput().Key("Dchannel").Title("Discord channel").Value(&discordModel.DiscordChannel),
			huh.NewInput().Key("Dame").Title("Discord Username").Value(&discordModel.DiscordUsername),
			huh.NewInput().Key("Dformat").Title("Discord Text format").Value(&discordModel.DiscordFormat),
			huh.NewInput().Key("Dwebhook").Title("Discord Webhookurl").Value(&discordModel.DiscordWebhookURL),
		),
	).WithWidth(width - 40).WithHeight(height - 10)

	return settingsModel{
		form:         form,
		discordModel: discordModel,
	}
}

func (m settingsModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m settingsModel) Update(msg tea.Msg) (settingsModel, tea.Cmd) {
	updatedForm, formCmd := m.form.Update(msg)
	if f, ok := updatedForm.(*huh.Form); ok {
		m.form = f
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.form.GetFocusedField().GetKey() != "Dwebhook" {
				m.form.NextField()
			}
		case "shift+tab":
			if m.form.GetFocusedField().GetKey() != "agent" {
				m.form.PrevField()
			}
		case "enter":
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
	return m, formCmd
}

func storetodb(m *Discord, key string, value string) {
	switch key {
	case "Did":
		m.ID = value
	case "Dchannel":
		m.DiscordChannel = value
	case "Dame":
		m.DiscordUsername = value
	case "Dformat":
		m.DiscordFormat = value
	case "Dwebhook":
		m.DiscordWebhookURL = value
	}

	config := &Config{Discord: []Discord{*m}}
	err := writeConfig(os.ExpandEnv("$HOME/.config/notify/provider-config.yaml"), config)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error writing config: %v", err), 1)
	}
}

func readConfig(filePath string) (*Config, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func writeConfig(filePath string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func updateDiscordChannel(config *Config, id string, newChannel string) {
	for i, d := range config.Discord {
		if d.ID == id {
			config.Discord[i].DiscordChannel = newChannel
			break
		}
	}
}

func (m settingsModel) View() string {
	return m.form.View()
}
