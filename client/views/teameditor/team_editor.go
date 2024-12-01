package teameditor

import (
	"fmt"
	"io"
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

	pokemonTeamStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Align(lipgloss.Center).Width(20)
	highlightedPokemonTeamStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Align(lipgloss.Center).Width(20)
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

	moveTeamDown = key.NewBinding(
		key.WithKeys("j", "down"),
	)

	moveTeamUp = key.NewBinding(
		key.WithKeys("k", "up"),
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

type TeamEditorModel struct {
	// TODO: Maybe make this a global package var?
	ctx      *teamEditorCtx
	subModel tea.Model
}

type (
	editTeamModel struct {
		ctx *teamEditorCtx

		addPokemonList      list.Model
		addingNewPokemon    bool
		currentPokemonIndex int
		choice              *game.BasePokemon
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
	}
)

func newEditTeamModel(ctx *teamEditorCtx) editTeamModel {
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

	teamPanels := make([]string, 0)

	for i, pokemon := range m.ctx.team {
		panel := fmt.Sprintf("%s\nLevel: %d\n", pokemon.Nickname, pokemon.Level)

		if i == m.currentPokemonIndex && !m.addingNewPokemon {
			teamPanels = append(teamPanels, highlightedPokemonTeamStyle.Render(panel))
		} else {
			teamPanels = append(teamPanels, pokemonTeamStyle.Render(panel))
		}
	}

	teamView := lipgloss.JoinVertical(lipgloss.Center, teamPanels...)

	return rendering.GlobalCenter(lipgloss.JoinHorizontal(lipgloss.Center, selection, teamView))
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

				m.currentPokemonIndex = len(m.ctx.team) - 1
			}

			var currentPokemon *game.Pokemon

			if len(m.ctx.team) > 0 {
				currentPokemon = &m.ctx.team[m.currentPokemonIndex]
			}

			if currentPokemon != nil {
				m.addingNewPokemon = true
				m.ctx.backtrack.Push(m)
				return newEditPokemonModel(m.ctx, currentPokemon), nil
			}
		}

		if !m.addingNewPokemon {
			if key.Matches(msg, moveTeamDown) {
				m.currentPokemonIndex++

				if m.currentPokemonIndex > len(m.ctx.team)-1 {
					m.currentPokemonIndex = 0
				}
			}

			if key.Matches(msg, moveTeamUp) {
				m.currentPokemonIndex--

				if m.currentPokemonIndex < 0 {
					m.currentPokemonIndex = len(m.ctx.team) - 1
				}
			}

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

	return rendering.GlobalCenter(prompt)
}

func (m saveTeamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)
	var cmd tea.Cmd

	cmd = m.saveNameInput.Focus()
	cmds = append(cmds, cmd)

	m.saveNameInput, cmd = m.saveNameInput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, goToPreviousPage) {
			return newEditTeamModel(m.ctx), nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.saveNameInput.Value() != "" {
				teams, _ := teamfs.LoadTeamMap()
				_, ok := teams[m.saveNameInput.Value()]

				// TODO: Add confirmation check if a team with this name already exists
				if ok {
					log.Info().Msgf("Team: %s already exists", m.saveNameInput.Value())
				}

				// TODO: Show error instead of crashing
				if err := teamfs.SaveTeam(m.saveNameInput.Value(), m.ctx.team); err != nil {
					log.Fatal().Msgf("Failed to save team: %s", err)
				}

				return newEditTeamModel(m.ctx), nil
			}
		case tea.KeyEsc:
			return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}

	return m, cmd
}

func NewTeamEditorModel(backtrack *components.Breadcrumbs) TeamEditorModel {
	ctx := teamEditorCtx{
		team:               make([]game.Pokemon, 0),
		listeningForEscape: true,
		backtrack:          backtrack,
	}
	teamEdit := newEditTeamModel(&ctx)

	return TeamEditorModel{
		ctx: &ctx,

		subModel: teamEdit,
	}
}

func (m *TeamEditorModel) AddStartingTeam(team []game.Pokemon) {
	m.ctx.team = team
}

func (m TeamEditorModel) Init() tea.Cmd {
	return nil
}

func (m TeamEditorModel) View() string {
	return m.subModel.View()
}

func (m TeamEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.ctx.listeningForEscape && msg.Type == tea.KeyEsc {
			return m, cmd
		}

		if m.ctx.listeningForEscape && msg.Type == tea.KeyEsc {
			return m.ctx.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}
	m.subModel, cmd = m.subModel.Update(msg)

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
