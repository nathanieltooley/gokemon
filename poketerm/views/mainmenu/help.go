package mainmenu

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/poketerm/global"
	"github.com/nathanieltooley/gokemon/poketerm/rendering"
	"github.com/nathanieltooley/gokemon/poketerm/rendering/components"
)

type helpMenuModel struct {
	backtrack components.Breadcrumbs
}

func newHelpMenu(backtrack components.Breadcrumbs) helpMenuModel {
	return helpMenuModel{backtrack}
}

func (m helpMenuModel) Init() tea.Cmd { return nil }
func (m helpMenuModel) View() string {
	return rendering.GlobalCenter(
		lipgloss.JoinVertical(lipgloss.Center, "Help",
			"Up / K to move up",
			"Down / J to move down",
			"H / Left to move left",
			"L / Right to move right",
			"Enter to select an item in a menu",
			"Esc to move to a previous menu",
		),
	)
}

func (m helpMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.BackKey) {
			return m.backtrack.PopDefault(func() tea.Model { return NewModel() }), nil
		}
	}

	return m, nil
}
