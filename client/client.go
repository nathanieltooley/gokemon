package main

import (
	"fmt"
	"io"
	"log"
	// "strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
)

type model struct {
	list     list.Model
	choice   *game.BasePokemon
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)
	choice, _ := m.list.Items()[m.list.Index()].(item)
	m.choice = choice.BasePokemon

	return m, cmd
}

func (m model) View() string {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Margin(2).Width(30)
	headerStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true).Width(30).Align(lipgloss.Center)

	var body string
	header := "Pokemon Selection"
	if m.choice != nil {
		body = fmt.Sprintf("Hp: %d\nAttack: %d\nDef: %d\nSpAttack: %d\nSpDef: %d\nSpeed: %d\n", m.choice.Hp, m.choice.Attack, m.choice.Def, m.choice.SpAttack, m.choice.SpDef, m.choice.Speed)
		header = fmt.Sprintf("Pokemon: %s", m.choice.Name)
	} else {
		body = ""
	}

	dialog := lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(header), body)

	return lipgloss.JoinVertical(lipgloss.Center, style.Render(dialog), m.list.View())
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

func main() {
	log.Println("Loading Pokemon Data")
	basePokemon, err := game.LoadBasePokemon("./data/gen1-data.csv")

	log.Printf("Loaded %d pokemon\n", len(basePokemon))

	if err != nil {
		log.Fatalf("Failed to load pokemon data: %s\n", err)
	}

	log.Println("Loading Move Data")
	moves, err := game.LoadMoves("./data/moves.json")

	if err != nil {
		log.Fatalf("Failed to load move data: %s\n", err)
	}

	log.Printf("Loaded %d moves\n", len(moves))

	// bulba := basePokemon.GetPokemonByName("bulbasaur")
	//
	// b1 := game.NewPokeBuilder(bulba).SetPerfectIvs().SetLevel(100).Build()
	// b2 := game.NewPokeBuilder(bulba).SetPerfectIvs().SetLevel(100).Build()
	//
	// log.Fatalln(game.Damage(b1, b2, moves.GetMove("pound")))

	items := make([]list.Item, len(basePokemon))
	for i, pkm := range basePokemon {
		items[i] = item{&pkm}
	}

	list := list.New(items, itemDelegate{}, 20, 24)
	list.Title = "Pokemon Selection"
	list.SetShowStatusBar(false)
	list.SetFilteringEnabled(true)
	list.SetShowFilter(true)

	m := model{list: list}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalln("Error running program: ", err)
	}
}
