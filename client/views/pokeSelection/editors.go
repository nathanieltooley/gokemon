package pokeselection

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type editor interface {
	View() string
	Update(*SelectionModel, tea.Msg) (editor, tea.Cmd)
}

type detailsEditor struct {
	nameInput  textinput.Model
	levelInput textinput.Model
}

func newDetailsEditor() detailsEditor {
	nameInput := textinput.New()
	nameInput.Placeholder = "Nickname"
	nameInput.Focus()
	nameInput.CharLimit = 16

	levelInput := textinput.New()
	levelInput.Placeholder = "Level"

	return detailsEditor{
		nameInput,
		levelInput,
	}
}

func (e detailsEditor) View() string {
	return lipgloss.JoinVertical(lipgloss.Center, e.nameInput.View(), e.levelInput.View())
}

func (e detailsEditor) Update(rootModel *SelectionModel, msg tea.Msg) (editor, tea.Cmd) {
	return e, nil
}
