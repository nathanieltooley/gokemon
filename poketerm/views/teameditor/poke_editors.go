package teameditor

import (
	"fmt"
	"math"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/golurk"
	"github.com/nathanieltooley/gokemon/poketerm/rendering"
	"github.com/samber/lo"
)

type editor interface {
	View() string
	Update(*editPokemonModel, tea.Msg) (editor, tea.Cmd)
}

// Component that regulates focus of text inputs
// TODO: Refactor this out to a separate component
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

func newDetailsEditor(pokeInfo golurk.Pokemon) detailsEditor {
	nameInput := textinput.New()
	nameInput.Placeholder = "Nickname"
	nameInput.Focus()
	nameInput.CharLimit = 16
	nameInput.SetValue(pokeInfo.Nickname)

	levelInput := textinput.New()
	levelInput.Blur()
	levelInput.Placeholder = "Level"
	levelInput.CharLimit = 3
	levelInput.SetValue(strconv.FormatUint(uint64(pokeInfo.Level), 10))

	return detailsEditor{
		newInputSelector([]textinput.Model{nameInput, levelInput}),
	}
}

func (e detailsEditor) View() string {
	views := make([]string, 0)
	for _, input := range e.is.inputs {
		views = append(views, input.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func (e detailsEditor) Update(rootModel *editPokemonModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd
	currentPokemon := rootModel.currentPokemon

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

					currentPokemon.Level = uint(parsedLevel)
					currentPokemon.ReCalcStats()
				}
			}

		}
	}

	return e, cmd
}

type evivEditor struct {
	is       inputSelector
	evBars   []progress.Model
	evValues []int
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

func newEVIVEditor(pokeInfo golurk.Pokemon) evivEditor {
	barWidth := 35

	hpiv := textinput.New()
	hpiv.Focus()
	hpiv.CharLimit = 3
	hpiv.SetValue(strconv.FormatUint(uint64(pokeInfo.Hp.Iv), 10))
	hpev := textinput.New()
	hpev.CharLimit = 3
	hpev.SetValue(strconv.FormatUint(uint64(pokeInfo.Hp.Ev), 10))
	hpBar := progress.New()
	hpBar.Width = barWidth

	attackIv := textinput.New()
	attackIv.CharLimit = 3
	attackIv.SetValue(strconv.FormatUint(uint64(pokeInfo.Attack.Iv), 10))
	attackEv := textinput.New()
	attackEv.CharLimit = 3
	attackEv.SetValue(strconv.FormatUint(uint64(pokeInfo.Attack.Ev), 10))
	attackBar := progress.New()
	attackBar.Width = barWidth

	defIv := textinput.New()
	defIv.CharLimit = 3
	defIv.SetValue(strconv.FormatUint(uint64(pokeInfo.Def.Iv), 10))
	defEv := textinput.New()
	defEv.CharLimit = 3
	defEv.SetValue(strconv.FormatUint(uint64(pokeInfo.Def.Ev), 10))
	defBar := progress.New()
	defBar.Width = barWidth

	spAttackIv := textinput.New()
	spAttackIv.CharLimit = 3
	spAttackIv.SetValue(strconv.FormatUint(uint64(pokeInfo.SpAttack.Iv), 10))
	spAttackEv := textinput.New()
	spAttackEv.CharLimit = 3
	spAttackEv.SetValue(strconv.FormatUint(uint64(pokeInfo.SpAttack.Ev), 10))
	spAttackBar := progress.New()
	spAttackBar.Width = barWidth

	spDefIv := textinput.New()
	spDefIv.CharLimit = 3
	spDefIv.SetValue(strconv.FormatUint(uint64(pokeInfo.SpDef.Iv), 10))
	spDefEv := textinput.New()
	spDefEv.CharLimit = 3
	spDefEv.SetValue(strconv.FormatUint(uint64(pokeInfo.SpDef.Ev), 10))
	spDefBar := progress.New()
	spDefBar.Width = barWidth

	speedIv := textinput.New()
	speedIv.CharLimit = 3
	speedIv.SetValue(strconv.FormatUint(uint64(pokeInfo.RawSpeed.Iv), 10))
	speedEv := textinput.New()
	speedEv.CharLimit = 3
	speedEv.SetValue(strconv.FormatUint(uint64(pokeInfo.RawSpeed.Ev), 10))
	speedBar := progress.New()
	speedBar.Width = barWidth

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
		[]progress.Model{
			hpBar,
			attackBar,
			defBar,
			spAttackBar,
			spDefBar,
			speedBar,
		},
		make([]int, 6),
	}
}

func (e evivEditor) View() string {
	views := make([]string, 0)
	for i := 0; i < len(e.is.inputs); i += 2 {
		header := ""
		evValue := e.evValues[i/2]
		evSum := lo.Sum(e.evValues)
		percent := float64(evValue) / float64(evSum)

		if percent > 1 {
			percent = 1
		} else if percent < 0 {
			percent = 0
		}

		barView := e.evBars[i/2].ViewAs(percent)

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
			lipgloss.JoinHorizontal(lipgloss.Center, e.is.inputs[i].View(), e.is.inputs[i+1].View(), barView)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, views...)
}

func getValidatedEv(inputString string, allowedTotalEvs int) (int, error) {
	parsedValue, error := strconv.ParseInt(inputString, 0, 64)

	if error != nil {
		return 0, error
	}

	allowedEvs := math.Min(golurk.MAX_EV, float64(allowedTotalEvs))

	return int(math.Min(allowedEvs, float64(parsedValue))), nil
}

func getValidatedIv(inputString string) (int, error) {
	parsedValue, error := strconv.ParseInt(inputString, 0, 64)

	if error != nil {
		return 0, error
	}

	return int(math.Min(golurk.MAX_IV, float64(parsedValue))), nil
}

func (e evivEditor) Update(rootModel *editPokemonModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd
	currentPokemon := rootModel.currentPokemon

	// Zero EVs out so that the EVs from last loop
	// dont mess with the values from this loop
	currentPokemon.Hp.Ev = 0
	currentPokemon.Attack.Ev = 0
	currentPokemon.Def.Ev = 0
	currentPokemon.SpAttack.Ev = 0
	currentPokemon.SpDef.Ev = 0
	currentPokemon.RawSpeed.Ev = 0

	e.is, cmd = e.is.Update(msg)

	for i, input := range e.is.inputs {
		allowedEvs := golurk.MAX_TOTAL_EV - currentPokemon.GetCurrentEvTotal()
		currentInputValue := input.Value()

		if allowedEvs <= 0 {
			break
		}

		switch i {
		case EI_HPIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Hp.Iv = uint(parsedIv)
			}
		case EI_HPEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.Hp.Ev = uint(parsedEv)
			}

		case EI_ATTACKIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Attack.Iv = uint(parsedIv)
			}
		case EI_ATTACKEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.Attack.Ev = uint(parsedEv)
			}

		case EI_DEFIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.Def.Iv = uint(parsedIv)
			}
		case EI_DEFEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.Def.Ev = uint(parsedEv)
			}

		case EI_SPAIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.SpAttack.Iv = uint(parsedIv)
			}
		case EI_SPAEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.SpAttack.Ev = uint(parsedEv)
			}

		case EI_SPDEFIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.SpDef.Iv = uint(parsedIv)
			}
		case EI_SPDEFEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.SpDef.Ev = uint(parsedEv)
			}

		case EI_SPEEDIV:
			parsedIv, err := getValidatedIv(currentInputValue)

			if err == nil {
				currentPokemon.RawSpeed.Iv = uint(parsedIv)
			}
		case EI_SPEEDEV:
			parsedEv, err := getValidatedEv(currentInputValue, allowedEvs)

			if err == nil {
				currentPokemon.RawSpeed.Ev = uint(parsedEv)
			}
		}
	}

	currentPokemon.ReCalcStats()
	e.evValues[0] = int(currentPokemon.Hp.Ev)
	e.evValues[1] = int(currentPokemon.Attack.Ev)
	e.evValues[2] = int(currentPokemon.Def.Ev)
	e.evValues[3] = int(currentPokemon.SpAttack.Ev)
	e.evValues[4] = int(currentPokemon.SpDef.Ev)
	e.evValues[5] = int(currentPokemon.RawSpeed.Ev)

	return e, cmd
}

type moveEditor struct {
	validMoves []golurk.Move

	moveIndex     int
	selectedMoves [4]golurk.Move
	lists         [4]list.Model
}

type moveItem struct {
	golurk.Move
}

func (i moveItem) FilterValue() string { return i.Name }
func (i moveItem) Value() string       { return i.Name }

func newMoveEditor(pokemon golurk.Pokemon, validMoves []golurk.Move) moveEditor {
	startingMoves := pokemon.Moves
	var lists [4]list.Model

	items := make([]list.Item, len(validMoves))
	for i, move := range validMoves {
		items[i] = moveItem{move}
	}

	for i := 0; i < 4; i++ {
		list := list.New(items, rendering.NewSimpleListDelegate(), 20, 15)
		list.SetFilteringEnabled(true)
		list.SetShowStatusBar(false)
		list.SetShowFilter(true)
		list.SetShowHelp(false)

		// BUG: This gets reenabled if a filter is clear after being applied
		list.KeyMap.Quit.SetEnabled(false)
		list.KeyMap.NextPage = key.NewBinding(key.WithKeys("l"))
		list.KeyMap.PrevPage = key.NewBinding(key.WithKeys("h"))

		list.Title = fmt.Sprintf("Select Move %d", i)
		lists[i] = list
	}

	return moveEditor{
		validMoves: lo.Map(validMoves, func(m golurk.Move, _ int) golurk.Move {
			return m
		}),
		moveIndex:     0,
		selectedMoves: startingMoves,
		lists:         lists,
	}
}

func (e moveEditor) View() string {
	moves := []string{"Move 1", "Move 2", "Move 3", "Move 4"}

	for i := range moves {
		move := "Nothing"

		if !e.selectedMoves[i].IsNil() {
			move = e.selectedMoves[i].Name
		}

		if i == e.moveIndex {
			moves[i] = fmt.Sprintf("> %s: %s", moves[i], move)
		} else {
			moves[i] = fmt.Sprintf("%s: %s", moves[i], move)
		}

	}

	return lipgloss.JoinHorizontal(lipgloss.Center, e.lists[e.moveIndex].View(), lipgloss.JoinVertical(lipgloss.Left, moves...))
}

func (e moveEditor) Update(rootModel *editPokemonModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			e.moveIndex++

			if e.moveIndex > 3 {
				e.moveIndex = 0
			}
		case tea.KeyShiftTab:
			e.moveIndex--

			if e.moveIndex < 0 {
				e.moveIndex = 3
			}
		case tea.KeyEnter:
			currentList := e.lists[e.moveIndex]
			choice := currentList.VisibleItems()[currentList.Index()].(moveItem)

			e.selectedMoves[e.moveIndex] = choice.Move
			// Update the actual pokemon as well
			rootModel.currentPokemon.Moves[e.moveIndex] = choice.Move

			e.moveIndex++

			if e.moveIndex > 3 {
				e.moveIndex = 0
			}
		}
	}

	newList, cmd := e.lists[e.moveIndex].Update(msg)
	e.lists[e.moveIndex] = newList

	switch e.lists[e.moveIndex].FilterState() {
	// Escape is the default keybind for clearing a filter
	// so we stop listening for it in the root selection model
	case list.Filtering:
		fallthrough
	case list.FilterApplied:
		rootModel.ctx.listeningForEscape = false
	default:
		rootModel.ctx.listeningForEscape = true
	}

	return e, cmd
}

type abilityEditor struct {
	abilityListModel list.Model
}

type abilityItem golurk.Ability

func (a abilityItem) FilterValue() string { return a.Name }
func (a abilityItem) Value() string       { return a.Name }

func newAbilityEditor(validAbilities []golurk.Ability) abilityEditor {
	items := make([]list.Item, len(validAbilities))
	for i, ability := range validAbilities {
		items[i] = abilityItem(ability)
	}

	aList := list.New(items, rendering.NewSimpleListDelegate(), 10, 10)
	aList.SetShowStatusBar(false)
	aList.DisableQuitKeybindings()
	return abilityEditor{aList}
}

func (e abilityEditor) View() string {
	return e.abilityListModel.View()
}

func (e abilityEditor) Update(rootModel *editPokemonModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			ability := e.abilityListModel.VisibleItems()[e.abilityListModel.Index()].(abilityItem)
			rootModel.currentPokemon.Ability = golurk.Ability(ability)
		}
	}

	e.abilityListModel, cmd = e.abilityListModel.Update(msg)

	return e, cmd
}

type itemEditor struct {
	itemListModel list.Model
}
type itemItem string

func (i itemItem) FilterValue() string { return string(i) }
func (i itemItem) Value() string       { return string(i) }

func newItemEditor() itemEditor {
	validItems := golurk.GlobalData.Items
	items := make([]list.Item, len(validItems))
	for i, item := range validItems {
		items[i] = itemItem(item)
	}

	iList := list.New(items, rendering.NewSimpleListDelegate(), 10, 10)
	iList.SetShowStatusBar(false)
	iList.DisableQuitKeybindings()
	return itemEditor{iList}
}

func (e itemEditor) View() string {
	return e.itemListModel.View()
}

func (e itemEditor) Update(rootModel *editPokemonModel, msg tea.Msg) (editor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			item := e.itemListModel.VisibleItems()[e.itemListModel.Index()].(itemItem)
			rootModel.currentPokemon.Item = string(item)
		}
	}

	e.itemListModel, cmd = e.itemListModel.Update(msg)

	return e, cmd
}
