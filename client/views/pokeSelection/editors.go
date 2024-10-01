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

type detailsEditor struct {
	focused int
	inputs  []textinput.Model
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
		0,
		[]textinput.Model{nameInput, levelInput},
	}
}

func (e detailsEditor) View() string {
	views := make([]string, 0)
	for _, input := range e.inputs {
		views = append(views, input.View())
	}
	return lipgloss.JoinVertical(lipgloss.Center, views...)
}

func (e detailsEditor) Update(rootModel *SelectionModel, msg tea.Msg) (editor, tea.Cmd) {
	cmds := make([]tea.Cmd, DE_LEN)
	currentPokemon := rootModel.Team[rootModel.currentPokemonIndex]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyTab {
			e.focused++

			if e.focused > DE_LEN-1 {
				e.focused = 0
			}
		}

		if msg.Type == tea.KeyShiftTab {
			e.focused--
			if e.focused < 0 {
				e.focused = DE_LEN - 1
			}
		}
	}

	for i := range e.inputs {
		switch i {
		case DE_NAME:
			nameValue := e.inputs[i].Value()

			if nameValue != "" {
				currentPokemon.Nickname = nameValue
			}

		case DE_LEVEL:
			levelValue := e.inputs[i].Value()

			if levelValue != "" {
				parsedLevel, err := strconv.ParseInt(levelValue, 0, 64)

				if err != nil {
					invalidValue := e.inputs[i].Value()
					e.inputs[i].SetValue(invalidValue[:len(invalidValue)-1])
				} else {
					parsedLevel = int64(math.Min(100, float64(parsedLevel)))

					currentPokemon.Level = uint8(parsedLevel)
					currentPokemon.ReCalcStats()
				}
			}

		}
	}

	for i, input := range e.inputs {
		if i == e.focused {
			input.Focus()
		} else {
			input.Blur()
		}

		e.inputs[i], cmds[i] = input.Update(msg)
	}

	return e, tea.Batch(cmds...)
}
