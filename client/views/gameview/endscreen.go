package gameview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

type endModel struct {
	message string
}

func newEndScreen(message string) endModel {
	return endModel{message}
}

func (m endModel) Init() tea.Cmd { return nil }
func (m endModel) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, "Game Over", m.message))
}

func (m endModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
