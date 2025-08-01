package mainmenu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/views/teameditor"
)

type MainMenuModel struct {
	buttons components.MenuButtons
}

func NewModel() MainMenuModel {
	buttons := []components.ViewButton{
		{
			Name: "Play Game",
			OnClick: func() (tea.Model, tea.Cmd) {
				backtrack := components.NewBreadcrumb()
				return NewPlaySelection(backtrack.PushNew(func() tea.Model { return NewModel() })), nil
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() (tea.Model, tea.Cmd) {
				// lol at not using breadcrumbs for this
				return teameditor.NewTeamMenu(func() tea.Model {
					return NewModel()
				}), nil
			},
		},
		{
			Name: "Options",
			OnClick: func() (tea.Model, tea.Cmd) {
				backtrack := components.NewBreadcrumb()
				return newOptionsMenu(backtrack.PushNew(func() tea.Model { return NewModel() })), nil
			},
		},
	}

	return MainMenuModel{
		buttons: components.NewMenuButton(buttons),
	}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) View() string {
	header := "Gokemon!"
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.buttons.View()))
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, startCmd := m.buttons.Update(msg)
	if newModel != nil {
		return newModel, startCmd
	}

	return m, nil
}
