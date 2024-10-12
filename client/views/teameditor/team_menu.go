package teameditor

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
)

type model struct {
	buttons components.MenuButtons
}

type menuItem struct {
	string
}

func (m menuItem) FilterValue() string { return m.string }
func (m menuItem) Value() string       { return m.string }

func NewTeamMenu() model {
	buttons := []components.ViewButton{
		{
			Name: "Create New Team",
			OnClick: func() tea.Model {
				return NewModel(global.POKEMON, global.MOVES, global.ABILITIES)
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() tea.Model {
				return nil
			},
		},
	}

	return model{
		components.NewMenuButton(buttons),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() string {
	return rendering.Center(lipgloss.JoinVertical(lipgloss.Center, m.buttons.View()))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel := m.buttons.Update(msg)
	if newModel != nil {
		return newModel, nil
	}

	return m, nil
}
