package mainmenu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/views/gameview"
)

type PlaySelection struct {
	buttons components.MenuButtons
}

func NewPlaySelection(backtrack components.Breadcrumbs) PlaySelection {
	buttons := []components.ViewButton{
		{
			Name: "Singleplayer",
			OnClick: func() (tea.Model, tea.Cmd) {
				return gameview.NewTeamSelectModel(backtrack.PushNew(func() tea.Model { return NewPlaySelection(backtrack) }), false, nil, 0), nil
			},
		},
		{
			Name: "Host Lobby",
			OnClick: func() (tea.Model, tea.Cmd) {
				lh := NewLobbyHost(backtrack.PushNew(func() tea.Model { return NewPlaySelection(backtrack) }))
				return lh, lh.Init()
			},
		},
		{
			Name: "Join Lobby",
			OnClick: func() (tea.Model, tea.Cmd) {
				backtrack.PushNew(func() tea.Model { return NewPlaySelection(backtrack) })
				lj := NewLobbyJoiner(backtrack)
				return lj, lj.Init()
			},
		},
	}

	return PlaySelection{
		buttons: components.NewMenuButton(buttons),
	}
}

func (m PlaySelection) Init() tea.Cmd { return nil }
func (m PlaySelection) View() string  { return rendering.GlobalCenter(m.buttons.View()) }
func (m PlaySelection) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, startCmd := m.buttons.Update(msg)
	if newModel != nil {
		return newModel, startCmd
	}
	return m, nil
}
