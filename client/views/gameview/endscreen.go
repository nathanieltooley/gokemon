package gameview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

type endModel struct{}

func newEndScreen() endModel {
	return endModel{}
}

func (m endModel) Init() tea.Cmd { return nil }
func (m endModel) View() string {
	return rendering.GlobalCenter("Game Over")
}

func (m endModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
