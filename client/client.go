package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/views/mainmenu"
)

type RootModel struct {
	currentView tea.Model
	quitting    bool
}

func (m RootModel) Init() tea.Cmd {
	return nil
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newView, cmd := m.currentView.Update(msg)

	m.currentView = newView

	// Disables the closing of the program when pressing ESC
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			return m, nil

		case tea.KeyCtrlC:
			cmd = tea.Quit
		}
	case tea.WindowSizeMsg:
		global.TERM_HEIGHT = msg.Height
		global.TERM_WIDTH = msg.Width
	}

	return m, cmd
}

func (m RootModel) View() string {
	return m.currentView.View()
}

func main() {
	m := RootModel{
		// currentView: pokeselection.NewModel(basePokemon, &moves, abilities),
		currentView: nil,
	}

	m.currentView = mainmenu.NewModel()

	log.Printf("Term Size: %d X %d\n", global.TERM_WIDTH, global.TERM_HEIGHT)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalln("Error running program: ", err)
	}
}
