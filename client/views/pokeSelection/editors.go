package pokeselection

import (
	"math"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
)

type editor interface {
	View() string
	Update(*SelectionModel, tea.Msg) (editor, tea.Cmd)
}

// Component that regulates focus of text inputs
type inputSelector struct {
	focused int
	inputs  []textinput.Model
}

func newInputSelector(inputs []textinput.Model) inputSelector {
	return inputSelector{
		0,
		inputs,
	}
}

func (is inputSelector) Update(msg tea.Msg) (inputSelector, tea.Cmd) {
	cmds := make([]tea.Cmd, len(is.inputs))

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyTab {
			is.focused++

			if is.focused > DE_LEN-1 {
				is.focused = 0
			}
		}

		if msg.Type == tea.KeyShiftTab {
			is.focused--
			if is.focused < 0 {
				is.focused = DE_LEN - 1
			}
		}
	}

	for i, input := range is.inputs {
		if i == is.focused {
			input.Focus()
		} else {
			input.Blur()
		}

		is.inputs[i], cmds[i] = input.Update(msg)
	}

	return is, tea.Batch(cmds...)
}

type detailsEditor struct {
	is inputSelector
}

const (
	DE_NAME = iota
	DE_LEVEL

	DE_LEN
)

func newDetailsEditor(pokeInfo *game.Pokemon) detailsEditor {
	nameInput := textinput.New()
	nameInput.Placeholder = "Nickname"
	nameInput.Focus()
	nameInput.CharLimit = 16
	nameInput.SetValue(pokeInfo.Nickname)

	levelInput := textinput.New()
	levelInput.Blur()
	levelInput.Placeholder = "Level"
	levelInput.CharLimit = 3
	levelInput.SetValue(string(pokeInfo.Level))

	return detailsEditor{
		newInputSelector([]textinput.Model{nameInput, levelInput}),
	}
}

func (e detailsEditor) View() string {
	views := make([]string, 0)
	for _, input := range e.is.inputs {
		views = append(views, input.View())
	}
	return lipgloss.JoinVertical(lipgloss.Center, views...)
}

func (e detailsEditor) Update(rootModel *SelectionModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd
	currentPokemon := rootModel.Team[rootModel.currentPokemonIndex]

	e.is, cmd = e.is.Update(msg)

	for i := range e.is.inputs {
		switch i {
		case DE_NAME:
			nameValue := e.is.inputs[i].Value()

			if nameValue != "" {
				currentPokemon.Nickname = nameValue
			}

		case DE_LEVEL:
			levelValue := e.is.inputs[i].Value()

			if levelValue != "" {
				parsedLevel, err := strconv.ParseInt(levelValue, 0, 64)

				if err != nil {
					invalidValue := e.is.inputs[i].Value()
					e.is.inputs[i].SetValue(invalidValue[:len(invalidValue)-1])
				} else {
					parsedLevel = int64(math.Min(100, float64(parsedLevel)))

					currentPokemon.Level = uint8(parsedLevel)
					currentPokemon.ReCalcStats()
				}
			}

		}
	}

	return e, cmd
}
