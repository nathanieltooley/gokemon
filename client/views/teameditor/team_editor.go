package teameditor

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/reg"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/shared/teamfs"
	"github.com/rs/zerolog/log"
)

var (
	infoStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Margin(2).Width(30)
	infoHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Border(lipgloss.NormalBorder(), false, false, true).Width(30).Align(lipgloss.Center)

	unselectedEditorStyle = lipgloss.NewStyle().Margin(2)
	selectedEditorStyle   = lipgloss.NewStyle().Margin(2).Bold(true).Border(lipgloss.BlockBorder(), true)
)

var (
	selectEditorLeft = key.NewBinding(
		key.WithKeys("left"),
	)

	selectEditorRight = key.NewBinding(
		key.WithKeys("right"),
	)

	enterPokeEditor = key.NewBinding(
		key.WithKeys("enter"),
	)

	openSaveTeam = key.NewBinding(
		key.WithKeys("s"),
	)

	toggleAddingPokemon = key.NewBinding(
		key.WithKeys("tab"),
	)

	goToPreviousPage = key.NewBinding(
		key.WithKeys(tea.KeyEsc.String()),
	)

	confirm = key.NewBinding(
		key.WithKeys("y", tea.KeyEnter.String()),
	)

	deletePokemonKey = key.NewBinding(
		key.WithKeys("d", tea.KeyDelete.String()),
	)
)

var editors = [...]string{"Details", "Moves", "Item", "Ability", "EV/IV"}

const (
	MODE_ADDPOKE = iota
	MODE_EDITPOKE
	MODE_SAVETEAM
)

type teamEditorCtx struct {
	// The current team we're editing
	team               []game.Pokemon
	listeningForEscape bool
	// HACK: needed to probably back out into team selection
	backtrack *components.Breadcrumbs
}

type (
	editTeamModel struct {
		ctx *teamEditorCtx

		addPokemonList   list.Model
		addingNewPokemon bool
		choice           *game.BasePokemon
		teamView         components.TeamView
	}
	editPokemonModel struct {
		ctx *teamEditorCtx

		editorModels       [len(editors)]editor
		moveRegistry       *reg.MoveRegistry
		abilities          map[string][]string
		currentPokemon     *game.Pokemon
		currentEditorIndex int
	}
	saveTeamModel struct {
		ctx *teamEditorCtx

		saveNameInput textinput.Model
		confirming    bool
		erroring      bool
		displayError  error
	}
)

func NewTeamEditorModel(backtrack *components.Breadcrumbs, team []game.Pokemon) editTeamModel {
	ctx := teamEditorCtx{
		team:               team,
		listeningForEscape: true,
		backtrack:          backtrack,
	}

	return newEditTeamModelCtx(&ctx)
}

func newEditTeamModelCtx(ctx *teamEditorCtx) editTeamModel {
	pokemon := global.POKEMON

	items := make([]list.Item, len(pokemon))
	for i, pkm := range pokemon {
		items[i] = item{&pkm}
	}

	list := list.New(items, itemDelegate{}, 20, 24)
	list.Title = "Pokemon Selection"
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(true)
	list.SetShowFilter(true)
	list.DisableQuitKeybindings()

	choice := list.Items()[0].(item).BasePokemon // grab first pokemon as default

	return editTeamModel{
		ctx: ctx,

		addPokemonList:   list,
		addingNewPokemon: true,
		choice:           choice,
		teamView:         components.NewTeamView(ctx.team),
	}
}

func (m editTeamModel) Init() tea.Cmd { return nil }
func (m editTeamModel) View() string {
	var body string
	var header string

	if m.choice != nil {
		body = fmt.Sprintf("Hp: %d\nAttack: %d\nDef: %d\nSpAttack: %d\nSpDef: %d\nSpeed: %d\n",
			m.choice.Hp,
			m.choice.Attack,
			m.choice.Def,
			m.choice.SpAttack,
			m.choice.SpDef,
			m.choice.Speed)
		header = fmt.Sprintf("Pokemon: %s", m.choice.Name)
	} else {
		body = ""
	}

	dialog := lipgloss.JoinVertical(lipgloss.Left, infoHeaderStyle.Render(header), body)
	selection := lipgloss.JoinVertical(lipgloss.Center, infoStyle.Render(dialog), m.addPokemonList.View())

	return rendering.GlobalCenter(lipgloss.JoinHorizontal(lipgloss.Center, selection, m.teamView.View()))
}

func (m editTeamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.addingNewPokemon {
		m.addPokemonList, cmd = m.addPokemonList.Update(msg)
		choice, _ := m.addPokemonList.Items()[m.addPokemonList.Index()].(item)

		m.choice = choice.BasePokemon
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, enterPokeEditor) {
			if m.addingNewPokemon {
				if m.choice != nil && len(m.ctx.team) < 6 {
					newPokemon := game.NewPokeBuilder(m.choice).SetLevel(100).Build()
					m.ctx.team = append(m.ctx.team, newPokemon)
				}

				m.teamView.CurrentPokemonIndex = len(m.ctx.team) - 1
			}

			var currentPokemon *game.Pokemon

			if len(m.ctx.team) > 0 {
				currentPokemon = &m.ctx.team[m.teamView.CurrentPokemonIndex]
			}

			if currentPokemon != nil {
				m.addingNewPokemon = true
				m.ctx.backtrack.PushNew(func() tea.Model {
					return newEditTeamModelCtx(m.ctx)
				})

				log.Debug().Msgf("len %d", len(m.ctx.team))
				return newEditPokemonModel(m.ctx, currentPokemon), nil
			}
		}

		if key.Matches(msg, deletePokemonKey) && !m.addingNewPokemon && len(m.ctx.team) > 0 {
			m.ctx.team = slices.Delete(m.ctx.team, m.teamView.CurrentPokemonIndex, m.teamView.CurrentPokemonIndex+1)
			newPkmIndex := int(math.Max(0, float64(m.teamView.CurrentPokemonIndex-1)))
			m.teamView = components.NewTeamView(m.ctx.team)
			m.teamView.CurrentPokemonIndex = newPkmIndex
		}

		if key.Matches(msg, toggleAddingPokemon) {
			// Toggle adding pokemon
			if len(m.ctx.team) > 0 {
				m.addingNewPokemon = !m.addingNewPokemon
			}
		}

		if key.Matches(msg, openSaveTeam) {
			m.ctx.backtrack.Push(m)
			return newSaveTeamModel(m.ctx), nil
		}

		if msg.Type == tea.KeyEsc {
			return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}

	m.teamView.Focused = !m.addingNewPokemon
	newTeamView, _ := m.teamView.Update(msg)
	m.teamView = newTeamView.(components.TeamView)

	return m, cmd
}

func (m editTeamModel) GetCurrentPokemon() game.Pokemon {
	return game.Pokemon{}
}

func newEditPokemonModel(ctx *teamEditorCtx, currentPokemon *game.Pokemon) editPokemonModel {
	moveRegistry := global.MOVES
	abilities := global.ABILITIES

	var editorModels [len(editors)]editor
	editorModels[0] = newDetailsEditor(*currentPokemon)
	editorModels[1] = newMoveEditor(*currentPokemon, moveRegistry.GetFullMovesForPokemon(currentPokemon.Base.Name))
	editorModels[2] = newItemEditor()
	editorModels[3] = newAbilityEditor(abilities[strings.ToLower(currentPokemon.Base.Name)])
	editorModels[4] = newEVIVEditor(*currentPokemon)

	return editPokemonModel{
		ctx: ctx,

		moveRegistry:   global.MOVES,
		abilities:      global.ABILITIES,
		currentPokemon: currentPokemon,
		editorModels:   editorModels,
	}
}

func (m editPokemonModel) Init() tea.Cmd { return nil }
func (m editPokemonModel) View() string {
	// header := "Editing Pokemon"
	// var body string

	currentEditor := m.editorModels[m.currentEditorIndex]

	type1 := m.currentPokemon.Base.Type1.Name
	type2 := ""

	if m.currentPokemon.Base.Type2 != nil {
		type2 = m.currentPokemon.Base.Type2.Name
	}

	typeString := ""
	if type2 == "" {
		typeString = type1
	} else {
		typeString = fmt.Sprintf("%s | %s", type1, type2)
	}

	info := fmt.Sprintf(`
            Name: %s
            Level: %d
            HP: %d:%d:%d
            Attack: %d:%d:%d
            Defense: %d:%d:%d
            Special Attack: %d:%d:%d
            Special Defense: %d:%d:%d
            Speed: %d:%d:%d
            Type: %s
            Ability: %s
            Item: %s
			Nature %s

            Move 1: %s
            Move 2: %s
            Move 3: %s
            Move 4: %s

            MAX EVS: %d
            `,
		m.currentPokemon.Nickname,
		m.currentPokemon.Level,

		m.currentPokemon.Hp.Value,
		m.currentPokemon.Hp.Iv,
		m.currentPokemon.Hp.Ev,

		m.currentPokemon.Attack.RawValue,
		m.currentPokemon.Attack.Iv,
		m.currentPokemon.Attack.Ev,

		m.currentPokemon.Def.RawValue,
		m.currentPokemon.Def.Iv,
		m.currentPokemon.Def.Ev,

		m.currentPokemon.SpAttack.RawValue,
		m.currentPokemon.SpAttack.Iv,
		m.currentPokemon.SpAttack.Ev,

		m.currentPokemon.SpDef.RawValue,
		m.currentPokemon.SpDef.Iv,
		m.currentPokemon.SpDef.Ev,

		m.currentPokemon.RawSpeed.RawValue,
		m.currentPokemon.RawSpeed.Iv,
		m.currentPokemon.RawSpeed.Ev,

		typeString,
		m.currentPokemon.Ability,
		m.currentPokemon.Item,
		m.currentPokemon.Nature.Name,

		getMoveName(m.currentPokemon.Moves[0]),
		getMoveName(m.currentPokemon.Moves[1]),
		getMoveName(m.currentPokemon.Moves[2]),
		getMoveName(m.currentPokemon.Moves[3]),
		game.MAX_TOTAL_EV-m.currentPokemon.GetCurrentEvTotal(),
	)

	var newEditors [len(editors)]string
	// editorTabs := strings.Builder{}

	for i, editor := range editors {
		if i == m.currentEditorIndex {
			newEditors[i] = selectedEditorStyle.Render(editor + "\t")
		} else {
			newEditors[i] = unselectedEditorStyle.Render(editor + "\t")
		}
	}

	var editorView string

	if currentEditor != nil {
		editorView = currentEditor.View()
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Center, newEditors[0:]...)

	fullEditorStyle := lipgloss.NewStyle().
		Width(int(float32(global.TERM_WIDTH-2)*0.75)).
		Height(global.TERM_HEIGHT-2).
		// AlignHorizontal(lipgloss.Center).
		Border(lipgloss.NormalBorder(), true).
		Padding(4)

	fullEditorView := fullEditorStyle.Render(lipgloss.JoinVertical(lipgloss.Left, tabs, editorView))

	editorInfoStyle := lipgloss.NewStyle().
		Width(int(float32(global.TERM_WIDTH)*0.25)).
		Height(global.TERM_HEIGHT-6).
		AlignHorizontal(lipgloss.Center).
		Border(lipgloss.InnerHalfBlockBorder(), true).
		Padding(4)

	view := lipgloss.JoinHorizontal(lipgloss.Center, editorInfoStyle.Render(info), fullEditorView)
	return rendering.GlobalCenter(view)
}

func (m editPokemonModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	currentModel := m.editorModels[m.currentEditorIndex]

	if currentModel != nil {
		var newModel editor
		newModel, cmd = currentModel.Update(&m, msg)
		m.editorModels[m.currentEditorIndex] = newModel
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, selectEditorLeft) {
			m.currentEditorIndex--

			if m.currentEditorIndex < 0 {
				m.currentEditorIndex = len(editors) - 1
			}
		}

		if key.Matches(msg, selectEditorRight) {
			m.currentEditorIndex++

			if m.currentEditorIndex >= len(editors) {
				m.currentEditorIndex = 0
			}
		}

		if key.Matches(msg, goToPreviousPage) {
			log.Debug().Msgf("len %d", len(m.ctx.team))
			return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}

	return m, cmd
}

func newSaveTeamModel(ctx *teamEditorCtx) saveTeamModel {
	newInput := textinput.New()
	newInput.CharLimit = 20
	newInput.Width = 20
	newInput.Placeholder = "Team Name"

	return saveTeamModel{
		ctx: ctx,

		saveNameInput: newInput,
	}
}

func (m saveTeamModel) Init() tea.Cmd { return nil }
func (m saveTeamModel) View() string {
	promptStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Align(lipgloss.Center).Padding(2, 15)
	prompt := promptStyle.Render(lipgloss.JoinVertical(lipgloss.Center, "Save Team", m.saveNameInput.View()))

	if m.confirming {
		prompt = promptStyle.Render(lipgloss.JoinVertical(lipgloss.Center, "A team with this name already exists. Overwrite?", "Y / N"))
	} else if m.erroring {
		// TODO: Better styling
		prompt = promptStyle.Render(lipgloss.JoinVertical(lipgloss.Center, "An error has occured", m.displayError.Error(), "Press ESC to exit"))
	}

	return rendering.GlobalCenter(prompt)
}

func (m saveTeamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	if m.confirming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, confirm) {
				if err := teamfs.SaveTeam(global.TeamSaveLocation, m.saveNameInput.Value(), m.ctx.team); err != nil {
					m.erroring = true
					m.displayError = err
					log.Error().Msgf("Failed to save team: %s", err)

					m.confirming = false

					return m, nil
				}

				return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
			} else {
				m.confirming = false
			}
		}

		return m, cmd
	}

	if m.erroring {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, goToPreviousPage) {
				m.erroring = false
				m.displayError = nil
			}
		}

		return m, cmd
	}

	cmd = m.saveNameInput.Focus()
	cmds = append(cmds, cmd)

	m.saveNameInput, cmd = m.saveNameInput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.saveNameInput.Value() != "" {
				teams, _ := teamfs.LoadTeamMap(global.TeamSaveLocation)
				_, ok := teams[m.saveNameInput.Value()]

				if ok {
					m.confirming = true
					log.Info().Msgf("Team: %s already exists", m.saveNameInput.Value())
					return m, cmd
				}

				if err := teamfs.SaveTeam(global.TeamSaveLocation, m.saveNameInput.Value(), m.ctx.team); err != nil {
					m.erroring = true
					m.displayError = err
					log.Error().Msgf("Failed to save team: %s", err)

					m.confirming = false

					return m, nil
				}

				return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
			}
		case tea.KeyEsc:
			return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}

	return m, cmd
}

type item struct {
	*game.BasePokemon
}

func (i item) FilterValue() string {
	return i.Name
}

type itemDelegate struct{}

func (i itemDelegate) Height() int                             { return 1 }
func (i itemDelegate) Spacing() int                            { return 0 }
func (i itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (i itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item := listItem.(item)

	var renderStr string
	renderStr = fmt.Sprintf("%d. %s", item.PokedexNumber, item.Name)
	if m.Index() == index {
		renderStr = fmt.Sprintf("> %d. %s", item.PokedexNumber, item.Name)
	}

	fmt.Fprint(w, renderStr)
}

func getMoveName(move *game.Move) string {
	if move != nil {
		return move.Name
	} else {
		return ""
	}
}
