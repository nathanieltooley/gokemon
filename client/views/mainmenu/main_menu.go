package mainmenu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/views"
	"github.com/nathanieltooley/gokemon/client/views/pokeselection"
)

var (
	buttonStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	highlightedButtonStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true)
)

type MainMenuModel struct {
	focusIndex int
}

func NewModel() MainMenuModel {
	return MainMenuModel{}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) View() string {
	buttons := []string{"Play Game", "Create Team"}

	for i, button := range buttons {
		if i == m.focusIndex {
			buttons[i] = highlightedButtonStyle.Render(button)
		} else {
			buttons[i] = buttonStyle.Render(button)
		}
	}

	items := []string{"Gokemon!"}
	items = append(items, buttons...)
	return views.Center(lipgloss.JoinVertical(lipgloss.Center, items...))
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			m.focusIndex++

			if m.focusIndex > 1 {
				m.focusIndex = 0
			}
		case tea.KeyShiftTab:
			m.focusIndex--

			if m.focusIndex < 0 {
				m.focusIndex = 1
			}

		case tea.KeyEnter:
			switch m.focusIndex {
			case 1:
				return pokeselection.NewModel(global.POKEMON, global.MOVES, global.ABILITIES), nil
			}
		}
	}

	return m, nil
}
