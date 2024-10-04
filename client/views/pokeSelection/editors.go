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
	inputLen := len(is.inputs)
	cmds := make([]tea.Cmd, inputLen)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyTab {
			is.focused++

			if is.focused > inputLen-1 {
				is.focused = 0
			}
		}

		if msg.Type == tea.KeyShiftTab {
			is.focused--
			if is.focused < 0 {
				is.focused = inputLen - 1
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

type evivEditor struct {
	is inputSelector
}

const (
	EI_HPIV = iota
	EI_HPEV
	EI_ATTACKIV
	EI_ATTACKEV
	EI_DEFIV
	EI_DEFEV
	EI_SPAIV
	EI_SPAEV
	EI_SPDEFIV
	EI_SPDEFEV
	EI_SPEEDIV
	EI_SPEEDEV
)

func newEVIVEditor(pokeInfo *game.Pokemon) evivEditor {
	hpiv := textinput.New()
	hpiv.Focus()
	hpiv.CharLimit = 3
	hpiv.SetValue(string(pokeInfo.Hp.Iv))
	hpev := textinput.New()
	hpev.CharLimit = 3
	hpev.SetValue(string(pokeInfo.Hp.Ev))

	attackIv := textinput.New()
	attackIv.CharLimit = 3
	attackIv.SetValue(string(pokeInfo.Attack.Iv))
	attackEv := textinput.New()
	attackEv.CharLimit = 3
	attackEv.SetValue(string(pokeInfo.Attack.Ev))

	defIv := textinput.New()
	defIv.CharLimit = 3
	defIv.SetValue(string(pokeInfo.Def.Iv))
	defEv := textinput.New()
	defEv.CharLimit = 3
	defEv.SetValue(string(pokeInfo.Def.Ev))

	spAttackIv := textinput.New()
	spAttackIv.CharLimit = 3
	spAttackIv.SetValue(string(pokeInfo.SpAttack.Iv))
	spAttackEv := textinput.New()
	spAttackEv.CharLimit = 3
	spAttackEv.SetValue(string(pokeInfo.SpAttack.Ev))

	spDefIv := textinput.New()
	spDefIv.CharLimit = 3
	spDefIv.SetValue(string(pokeInfo.SpDef.Iv))
	spDefEv := textinput.New()
	spDefEv.CharLimit = 3
	spDefEv.SetValue(string(pokeInfo.SpDef.Ev))

	speedIv := textinput.New()
	speedIv.CharLimit = 3
	speedIv.SetValue(string(pokeInfo.Speed.Iv))
	speedEv := textinput.New()
	speedEv.CharLimit = 3
	speedEv.SetValue(string(pokeInfo.Speed.Ev))

	return evivEditor{
		newInputSelector([]textinput.Model{
			hpiv,
			hpev,
			attackIv,
			attackEv,
			defIv,
			defEv,
			spAttackIv,
			spAttackEv,
			spDefIv,
			spDefEv,
			speedIv,
			speedEv,
		}),
	}
}

func (e evivEditor) View() string {
	views := make([]string, 0)
	for i := 0; i < len(e.is.inputs); i += 2 {
		header := ""

		switch i {
		case EI_HPIV:
			header = "HP"
		case EI_ATTACKIV:
			header = "ATTACK"
		case EI_DEFIV:
			header = "DEFENSE"
		case EI_SPAIV:
			header = "SPECIAL ATTACK"
		case EI_SPDEFIV:
			header = "SPECIAL DEFENSE"
		case EI_SPEEDIV:
			header = "SPEED"
		}

		views = append(views, lipgloss.JoinVertical(lipgloss.Left, header,
			lipgloss.JoinHorizontal(lipgloss.Center, e.is.inputs[i].View(), e.is.inputs[i+1].View())))
	}
	return lipgloss.JoinVertical(lipgloss.Center, views...)
}

func getValidatedEv(inputString string) (int, error) {
	parsedValue, error := strconv.ParseInt(inputString, 0, 64)

	if error != nil {
		return 0, error
	}

	return int(math.Min(game.MAX_EV, float64(parsedValue))), nil
}

func getValidatedIv(inputString string) (int, error) {
	parsedValue, error := strconv.ParseInt(inputString, 0, 64)

	if error != nil {
		return 0, error
	}

	return int(math.Min(game.MAX_IV, float64(parsedValue))), nil
}

func (e evivEditor) Update(rootModel *SelectionModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd
	currentPokemon := rootModel.Team[rootModel.currentPokemonIndex]

	e.is, cmd = e.is.Update(msg)

	// TODO: Validate Max EV count
	for i, input := range e.is.inputs {
		currentInputValue := input.Value()

		switch i {
		case EI_HPIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Hp.Iv = uint8(parsedIv)
			}
		case EI_HPEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.Hp.Ev = uint8(parsedEv)
			}

		case EI_ATTACKIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Attack.Iv = uint8(parsedIv)
			}
		case EI_ATTACKEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.Attack.Ev = uint8(parsedEv)
			}

		case EI_DEFIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Def.Iv = uint8(parsedIv)
			}
		case EI_DEFEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.Def.Ev = uint8(parsedEv)
			}

		case EI_SPAIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.SpAttack.Iv = uint8(parsedIv)
			}
		case EI_SPAEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.SpAttack.Ev = uint8(parsedEv)
			}

		case EI_SPDEFIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.SpDef.Iv = uint8(parsedIv)
			}
		case EI_SPDEFEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.SpDef.Ev = uint8(parsedEv)
			}

		case EI_SPEEDIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Speed.Iv = uint8(parsedIv)
			}
		case EI_SPEEDEV:
			parsedEv, err := getValidatedEv(currentInputValue)

			if err == nil {
				currentPokemon.Speed.Ev = uint8(parsedEv)
			}
		}

	}

	currentPokemon.ReCalcStats()

	return e, cmd
}
