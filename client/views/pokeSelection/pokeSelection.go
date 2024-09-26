package pokeselection

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
)

var (
	infoStyle       = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Margin(2).Width(30)
	infoHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Border(lipgloss.NormalBorder(), false, false, true).Width(30).Align(lipgloss.Center)
)

type SelectionModel struct {
	Team   []*game.Pokemon
	Choice *game.BasePokemon

	list list.Model
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
	var body string
	header := "Pokemon Selection"
	if m.Choice != nil {
		body = fmt.Sprintf("Hp: %d\nAttack: %d\nDef: %d\nSpAttack: %d\nSpDef: %d\nSpeed: %d\n", m.Choice.Hp, m.Choice.Attack, m.Choice.Def, m.Choice.SpAttack, m.Choice.SpDef, m.Choice.Speed)
		header = fmt.Sprintf("Pokemon: %s", m.Choice.Name)
	} else {
		body = ""
	}

	dialog := lipgloss.JoinVertical(lipgloss.Left, infoHeaderStyle.Render(header), body)

	return lipgloss.JoinVertical(lipgloss.Center, infoStyle.Render(dialog), m.list.View())
}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	choice, _ := m.list.Items()[m.list.Index()].(item)
	m.Choice = choice.BasePokemon

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
