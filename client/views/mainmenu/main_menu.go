package mainmenu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/views/teameditor"
)

type MainMenuModel struct {
	buttons rendering.MenuButtons
}

func NewModel() MainMenuModel {
	buttons := []rendering.ViewButton{
		{
			Name: "Play Game",
			OnClick: func() tea.Model {
				return nil
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() tea.Model {
				return teameditor.NewTeamMenu()
			},
		},
	}

	return MainMenuModel{
		buttons: rendering.NewMenuButton(buttons),
	}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) View() string {
	header := "Gokemon!"
	return rendering.Center(lipgloss.JoinVertical(lipgloss.Center, header, m.buttons.View()))
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel := m.buttons.Update(msg)
	if newModel != nil {
		return newModel, nil
	}

	return m, nil
}
