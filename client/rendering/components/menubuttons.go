package components

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

type ViewButton struct {
	Name    string
	OnClick func() (tea.Model, tea.Cmd)
}

type MenuButtons struct {
	buttons []ViewButton
	index   int
}

func NewMenuButton(buttons []ViewButton) MenuButtons {
	return MenuButtons{
		buttons: buttons,
	}
}

// MenuButtons only return a non nil value when a button is selected
func (m *MenuButtons) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.MoveDownKey) {
			m.index++
			if m.index >= len(m.buttons) {
				m.index = 0
			}
		}

		if key.Matches(msg, global.MoveUpKey) {
			m.index--

			if m.index < 0 {
				m.index = len(m.buttons) - 1
			}
		}

		switch msg.Type {
		case tea.KeyEnter:
			return m.buttons[m.index].OnClick()
		}
	}

	return nil, nil
}

func (m MenuButtons) View() string {
	views := make([]string, len(m.buttons))
	for i, button := range m.buttons {
		if i == m.index {
			views[i] = rendering.HighlightedButtonStyle.Render(button.Name)
		} else {
			views[i] = rendering.ButtonStyle.Render(button.Name)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Center, views...)
}

func (m *MenuButtons) Unfocus() {
	m.index = -1
}

func (m *MenuButtons) Focus() {
	if m.index == -1 {
		m.index = 0
	}
}
