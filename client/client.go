package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/views/pokeSelection"
)

type model struct {
	currentView tea.Model
	quitting    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newView, cmd := m.currentView.Update(msg)

	m.currentView = newView

	// Disables the closing of the program when pressing ESC
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEscape {
			return m, nil
		}
	}

	return m, cmd
}

func (m model) View() string {
	return m.currentView.View()
}

func main() {
	log.Println("Loading Pokemon Data")
	basePokemon, err := game.LoadBasePokemon("./data/gen1-data.csv")

	log.Printf("Loaded %d pokemon\n", len(basePokemon))

	if err != nil {
		log.Fatalf("Failed to load pokemon data: %s\n", err)
	}

	log.Println("Loading Move Data")
	moves, err := game.LoadMoves("./data/moves.json", "./data/movesMap.json")

	if err != nil {
		log.Fatalf("Failed to load move data: %s\n", err)
	}

	log.Printf("Loaded %d moves\n", len(moves.MoveList))
	log.Printf("Loaded move info for %d pokemon\n", len(moves.MoveMap))

	m := model{
		currentView: pokeselection.NewModel(basePokemon),
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalln("Error running program: ", err)
	}
}
