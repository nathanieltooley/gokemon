package pokeselection

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/views"
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

	moveTeamDown = key.NewBinding(
		key.WithKeys("j", "down"),
	)

	moveTeamUp = key.NewBinding(
		key.WithKeys("k", "up"),
	)

	openSaveTeam = key.NewBinding(
		key.WithKeys("s"),
	)
)

var editors = [...]string{"Details", "Moves", "Item", "Ability", "EV/IV"}

const (
	MODE_ADDPOKE = iota
	MODE_EDITPOKE
	MODE_SAVETEAM
)

type SelectionModel struct {
	Team   []*game.Pokemon
	Choice *game.BasePokemon

	list                list.Model
	mode                int
	currentPokemonIndex int
	currentEditorIndex  int
	moveRegistry        *game.MoveRegistry
	editorModels        [len(editors)]editor
	listeningForEscape  bool
	addingNewPokemon    bool
	abilities           map[string][]string
	saveNameInput       textinput.Model
}

func NewModel(pokemon game.PokemonRegistry, moves *game.MoveRegistry, abilities map[string][]string) SelectionModel {
	items := make([]list.Item, len(pokemon))
	for i, pkm := range pokemon {
		items[i] = item{&pkm}
	}

	list := list.New(items, itemDelegate{}, 20, 24)
	list.Title = "Pokemon Selection"
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(true)
	list.SetShowFilter(true)

	var editorModels [len(editors)]editor

	return SelectionModel{
		list:               list,
		editorModels:       editorModels,
		moveRegistry:       moves,
		listeningForEscape: true,
		addingNewPokemon:   true,
		abilities:          abilities,

		Choice: list.Items()[0].(item).BasePokemon, // grab first pokemon as default
	}
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) View() string {
	switch m.mode {
	case MODE_ADDPOKE:
		return RenderAddMode(m)
	case MODE_EDITPOKE:
		return RenderEditMode(m)
	case MODE_SAVETEAM:
		return RenderTeamMode(m)
	}

	return ""
}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	// Update add pokemon list in addpoke mode
	if m.mode == MODE_ADDPOKE && m.addingNewPokemon {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		choice, _ := m.list.Items()[m.list.Index()].(item)

		m.Choice = choice.BasePokemon
	}

	if m.mode == MODE_SAVETEAM {
		var cmd tea.Cmd
		m.saveNameInput, cmd = m.saveNameInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update Current Editor
	if m.mode == MODE_EDITPOKE {
		currentModel := m.editorModels[m.currentEditorIndex]

		if currentModel != nil {
			newModel, cmdFromEditor := currentModel.Update(&m, msg)
			m.editorModels[m.currentEditorIndex] = newModel
			cmds = append(cmds, cmdFromEditor)
		}

	}

	// Listen to key presses
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Select pokemon and enter editing view
			if m.mode == MODE_ADDPOKE {
				if m.addingNewPokemon {
					if m.Choice != nil && len(m.Team) < 6 {
						newPokemon := game.NewPokeBuilder(m.Choice).Build()
						m.Team = append(m.Team, newPokemon)
					}

					m.currentPokemonIndex = len(m.Team) - 1
				}

				currentPokemon := m.GetCurrentPokemon()

				if currentPokemon != nil {
					m.mode = MODE_EDITPOKE

					m.editorModels[0] = newDetailsEditor(currentPokemon)
					m.editorModels[1] = newMoveEditor(currentPokemon, m.moveRegistry.GetFullMovesForPokemon(m.Choice.Name))
					m.editorModels[3] = newAbilityEditor(m.abilities[strings.ToLower(currentPokemon.Base.Name)])
					m.editorModels[4] = newEVIVEditor(currentPokemon)
				}

			}

			if m.mode == MODE_SAVETEAM {
				if m.saveNameInput.Value() != "" {
					if err := SaveTeam(m.saveNameInput.Value(), m.Team); err != nil {
						log.Fatalln("Failed to save team: ", err)
					}

					m.mode = MODE_ADDPOKE
				}
			}

		case tea.KeyEscape:
			// Leave editing mode and go back to add mode
			if (m.mode == MODE_EDITPOKE || m.mode == MODE_SAVETEAM) && m.listeningForEscape {
				m.mode = MODE_ADDPOKE
			}

		case tea.KeyTab:
			if m.mode == MODE_ADDPOKE {
				m.addingNewPokemon = !m.addingNewPokemon
			}
		}

		// Listen to key presses for edit mode
		if m.mode == MODE_EDITPOKE {
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
		}

		if m.mode == MODE_ADDPOKE && !m.addingNewPokemon {
			if key.Matches(msg, moveTeamDown) {
				m.currentPokemonIndex++

				if m.currentPokemonIndex > len(m.Team)-1 {
					m.currentPokemonIndex = 0
				}
			}

			if key.Matches(msg, moveTeamUp) {
				m.currentPokemonIndex--

				if m.currentPokemonIndex < 0 {
					m.currentPokemonIndex = len(m.Team) - 1
				}
			}
		}

		if m.mode == MODE_ADDPOKE {
			if key.Matches(msg, openSaveTeam) {
				m.mode = MODE_SAVETEAM

				newInput := textinput.New()
				newInput.CharLimit = 20
				newInput.Width = 20
				newInput.Placeholder = "Team Name"
				cmds = append(cmds, newInput.Focus())
				m.saveNameInput = newInput
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (s SelectionModel) GetCurrentPokemon() *game.Pokemon {
	if len(s.Team) > 0 {
		return s.Team[s.currentPokemonIndex]
	}

	return nil
}

func RenderAddMode(m SelectionModel) string {
	var body string
	var header string

	if m.Choice != nil {
		body = fmt.Sprintf("Hp: %d\nAttack: %d\nDef: %d\nSpAttack: %d\nSpDef: %d\nSpeed: %d\n",
			m.Choice.Hp,
			m.Choice.Attack,
			m.Choice.Def,
			m.Choice.SpAttack,
			m.Choice.SpDef,
			m.Choice.Speed)
		header = fmt.Sprintf("Pokemon: %s", m.Choice.Name)
	} else {
		body = ""
	}

	dialog := lipgloss.JoinVertical(lipgloss.Left, infoHeaderStyle.Render(header), body)
	selection := lipgloss.JoinVertical(lipgloss.Center, infoStyle.Render(dialog), m.list.View())

	teamPanels := make([]string, 0)

	for i, pokemon := range m.Team {
		panel := fmt.Sprintf("%s\nLevel: %d\n", pokemon.Nickname, pokemon.Level)

		if i == m.currentPokemonIndex && !m.addingNewPokemon {
			teamPanels = append(teamPanels, highlightedPokemonTeamStyle.Render(panel))
		} else {
			teamPanels = append(teamPanels, pokemonTeamStyle.Render(panel))
		}
	}

	teamView := lipgloss.JoinVertical(lipgloss.Center, teamPanels...)

	return views.Center(lipgloss.JoinHorizontal(lipgloss.Center, selection, teamView))
}

func RenderEditMode(m SelectionModel) string {
	// header := "Editing Pokemon"
	// var body string

	currentPokemon := m.Team[m.currentPokemonIndex]
	currentEditor := m.editorModels[m.currentEditorIndex]

	type1 := currentPokemon.Base.Type1.Name
	type2 := ""

	if currentPokemon.Base.Type2 != nil {
		type2 = currentPokemon.Base.Type2.Name
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
            Type: %s | %s
            Ability: %s
            Item: %s

            Move 1: %s
            Move 2: %s
            Move 3: %s
            Move 4: %s

            MAX EVS: %d
            `,
		currentPokemon.Nickname,
		currentPokemon.Level,

		currentPokemon.Hp.Value,
		currentPokemon.Hp.Iv,
		currentPokemon.Hp.Ev,

		currentPokemon.Attack.Value,
		currentPokemon.Attack.Iv,
		currentPokemon.Attack.Ev,

		currentPokemon.Def.Value,
		currentPokemon.Def.Iv,
		currentPokemon.Def.Ev,

		currentPokemon.SpAttack.Value,
		currentPokemon.SpAttack.Iv,
		currentPokemon.SpAttack.Ev,

		currentPokemon.SpDef.Value,
		currentPokemon.SpDef.Iv,
		currentPokemon.SpDef.Ev,

		currentPokemon.Speed.Value,
		currentPokemon.Speed.Iv,
		currentPokemon.Speed.Ev,

		type1,
		type2,
		currentPokemon.Ability,
		"",

		"",
		"",
		"",
		"",
		game.MAX_TOTAL_EV-currentPokemon.GetCurrentEvTotal(),
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
	return lipgloss.JoinVertical(lipgloss.Center, info, tabs, editorView)
}

func RenderTeamMode(m SelectionModel) string {
	// teams, err := LoadTeamMap()
	// if err != nil {
	// 	return views.Center(fmt.Sprintf("Could not load teams: %s", err))
	// }

	promptStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Align(lipgloss.Center).Padding(2, 15)
	prompt := promptStyle.Render(lipgloss.JoinVertical(lipgloss.Center, "Save Team", m.saveNameInput.View()))

	// return views.Center(lipgloss.JoinVertical(lipgloss.Center, slices.Collect(maps.Keys(teams))...))
	return views.Center(prompt)
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
