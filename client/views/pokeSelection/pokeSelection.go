package pokeselection

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
)

var (
	infoStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Margin(2).Width(30)
	infoHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Border(lipgloss.NormalBorder(), false, false, true).Width(30).Align(lipgloss.Center)

	unselectedEditorStyle = lipgloss.NewStyle().Margin(2)
	selectedEditorStyle   = lipgloss.NewStyle().Margin(2).Bold(true).Border(lipgloss.BlockBorder(), true)
)

var (
	selectEditorLeft = key.NewBinding(
		key.WithKeys("editorLeft", "h"),
	)

	selectEditorRight = key.NewBinding(
		key.WithKeys("editorRight", "l"),
	)
)

var editors = [...]string{"Details", "Moves", "Item", "Ability", "EV/IV"}

const (
	MODE_ADDPOKE = iota
	MODE_EDITPOKE
)

type SelectionModel struct {
	Team   []*game.Pokemon
	Choice *game.BasePokemon

	list                list.Model
	mode                int
	currentPokemonIndex int
	currentEditor       int
}

func NewModel(pokemon game.PokemonRegistry) SelectionModel {
	items := make([]list.Item, len(pokemon))
	for i, pkm := range pokemon {
		items[i] = item{&pkm}
	}

	list := list.New(items, itemDelegate{}, 20, 24)
	list.Title = "Pokemon Selection"
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(true)
	list.SetShowFilter(true)

	return SelectionModel{list: list}
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) View() string {
	if m.mode == MODE_ADDPOKE {
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

		for _, pokemon := range m.Team {
			panel := fmt.Sprintf(`
                %s
                Level: %d
                `, pokemon.Base.Name, pokemon.Level)
			teamPanels = append(teamPanels, panel)
		}

		teamView := lipgloss.JoinVertical(lipgloss.Center, teamPanels...)

		return lipgloss.JoinHorizontal(lipgloss.Center, selection, teamView)
	} else if m.mode == MODE_EDITPOKE {
		// header := "Editing Pokemon"
		// var body string

		currentPokemon := m.Team[m.currentPokemonIndex]

		type1 := currentPokemon.Base.Type1.Name
		type2 := ""

		if currentPokemon.Base.Type2 != nil {
			type2 = currentPokemon.Base.Type2.Name
		}

		// TODO: Constant panel showing Pokemon Info
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
            `,
			currentPokemon.Base.Name,
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
			"",
			"",

			"",
			"",
			"",
			"",
		)

		var newEditors [len(editors)]string
		// editorTabs := strings.Builder{}

		for i, editor := range editors {
			if i == int(m.currentEditor) {
				newEditors[i] = selectedEditorStyle.Render(editor + "\t")
			} else {
				newEditors[i] = unselectedEditorStyle.Render(editor + "\t")
			}
		}

		tabs := lipgloss.JoinHorizontal(lipgloss.Center, newEditors[0:]...)
		return lipgloss.JoinVertical(lipgloss.Center, info, tabs)

		// TODO: EV / IV editor

		// TODO: Move editor

		// TODO: Ability editor

		// TODO: Item editor

		// TODO: Level, Name, (maybe tera?) editor

		// TODO: Pokemon Replacement panel?
	}

	return ""

}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	cmd = nil

	if m.mode == MODE_ADDPOKE {
		m.list, cmd = m.list.Update(msg)
		choice, _ := m.list.Items()[m.list.Index()].(item)

		m.Choice = choice.BasePokemon
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.mode == MODE_ADDPOKE {
				if m.Choice != nil && len(m.Team) < 6 {
					newPokemon := game.NewPokeBuilder(m.Choice).Build()
					m.Team = append(m.Team, newPokemon)
				}

				m.currentPokemonIndex = len(m.Team) - 1
				m.mode = MODE_EDITPOKE
			}
		case tea.KeyEscape:
			if m.mode == MODE_EDITPOKE {
				m.mode = MODE_ADDPOKE
			}

		case tea.KeyCtrlC:
			cmd = tea.Quit

		}

		if m.mode == MODE_EDITPOKE {
			if key.Matches(msg, selectEditorLeft) {
				m.currentEditor--

				if m.currentEditor < 0 {
					m.currentEditor = len(editors) - 1
				}
			}

			if key.Matches(msg, selectEditorRight) {
				m.currentEditor++

				if m.currentEditor >= len(editors) {
					m.currentEditor = 0
				}
			}
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
