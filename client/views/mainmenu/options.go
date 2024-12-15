package mainmenu

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
)

type optionsMenuModel struct {
	backtrack components.Breadcrumbs

	saveLocationPrompt textinput.Model
}

func newOptionsMenu(backtrack components.Breadcrumbs) optionsMenuModel {
	prompt := textinput.New()
	prompt.Focus()
	prompt.SetValue(global.TeamSaveLocation)

	return optionsMenuModel{
		backtrack:          backtrack,
		saveLocationPrompt: prompt,
	}
}

func (m optionsMenuModel) Init() tea.Cmd { return nil }
func (m optionsMenuModel) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, "Team Save Location", m.saveLocationPrompt.View()))
}

func (m optionsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m.backtrack.PopDefault(func() tea.Model { return m }), nil
		case tea.KeyEnter:
			if m.saveLocationPrompt.Value() != "" {
				// TODO: Validation!
				global.TeamSaveLocation = m.saveLocationPrompt.Value()
			}
		}
	}

	m.saveLocationPrompt, cmd = m.saveLocationPrompt.Update(msg)

	return m, cmd
}
