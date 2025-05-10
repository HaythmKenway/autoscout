package gui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type settingsModel struct {
	form *huh.Form
}

func NewSettingsModel(width int) settingsModel {
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
		),
	).WithWidth(width - 40).WithHeight(10)

	return settingsModel{form: form}
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
		switch msg.String(){
			case "tab":
				if(m.form.GetFocusedField().GetKey()!="data"){	
				m.form.NextField()}
			case "shift+tab":
				if(m.form.GetFocusedField().GetKey()!="agent"){	
				m.form.PrevField()
			}
}}
	return m, formCmd
}

func (m settingsModel) View() string {
	return m.form.View()
}
