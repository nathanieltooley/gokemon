package gameview

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

type playerPanel struct {
	gameState state.GameState

	player *state.Player
	name   string
}

func newPlayerPanel(gameState state.GameState, name string, player *state.Player) playerPanel {
	return playerPanel{
		gameState: gameState,

		player: player,
		name:   name,
	}
}

func (m playerPanel) Init() tea.Cmd { return nil }
func (m playerPanel) View() string {
	currentPokemon := m.player.Team[m.player.ActivePokeIndex]
	pokeInfo := fmt.Sprintf("%s\n%d / %d\nLevel: %d",
		currentPokemon.Nickname,
		currentPokemon.Hp.Value,
		currentPokemon.MaxHp,
		currentPokemon.Level,
	)

	pokeStyle := lipgloss.NewStyle().Align(lipgloss.Center).Border(lipgloss.NormalBorder(), true).Width(20).Height(5)

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Center, m.name, pokeStyle.Render(pokeInfo)))
}

func (m playerPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

type actionPanel struct {
	ctx *gameContext

	actionFocus int
}

func (m actionPanel) Init() tea.Cmd { return nil }
func (m actionPanel) View() string {
	var fight string
	var pokemon string

	if m.actionFocus == 0 {
		fight = highlightedPanelStyle.Width(15).Render("Fight")
	} else {
		fight = panelStyle.Width(15).Render("Fight")
	}

	if m.actionFocus == 1 {
		pokemon = highlightedPanelStyle.Width(15).Render("Pokemon")
	} else {
		pokemon = panelStyle.Width(15).Render("Pokemon")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, fight, pokemon)
}

func (m actionPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			switch m.actionFocus {
			case 0:
				return movePanel{
					ctx:   m.ctx,
					moves: m.ctx.state.LocalPlayer.GetActivePokemon().Moves,
				}, nil
			case 1:
				return newPokemonPanel(m.ctx, m.ctx.state.LocalPlayer.Team), nil
			}
		}

		if key.Matches(msg, global.MoveLeftKey) {
			m.actionFocus--

			if m.actionFocus < 0 {
				m.actionFocus = 1
			}
		}

		if key.Matches(msg, global.MoveRightKey) {
			m.actionFocus++

			if m.actionFocus > 1 {
				m.actionFocus = 0
			}
		}
	}

	return m, nil
}

type movePanel struct {
	ctx           *gameContext
	moveGridFocus int

	moves [4]*game.Move
}

func (m movePanel) Init() tea.Cmd { return nil }
func (m movePanel) View() string {
	grid := make([]string, 0)

	// Move grid
	// TODO: Maybe refactor this into a separate component?
	for i := 0; i < 2; i++ {
		row := make([]string, 0)
		for j := 0; j < 2; j++ {
			arrayIndex := (i * 2) + j
			style := panelStyle

			if arrayIndex == m.moveGridFocus {
				style = style.Background(rendering.HighlightedColor)
			}

			if m.moves[arrayIndex] == nil {
				row = append(row, style.Render("Empty"))
			} else {
				row = append(row, style.Render(m.moves[i].Name))
			}
		}

		grid = append(grid, lipgloss.JoinHorizontal(lipgloss.Center, row...))
	}

	return lipgloss.JoinVertical(lipgloss.Center, grid...)
}

func (m movePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.MoveLeftKey) {
			m.moveGridFocus = int(math.Max(0, float64(m.moveGridFocus-1)))
		}

		if key.Matches(msg, global.MoveRightKey) {
			m.moveGridFocus = int(math.Min(3, float64(m.moveGridFocus+1)))
		}

		if key.Matches(msg, global.MoveDownKey) {
			m.moveGridFocus = int(math.Min(3, float64(m.moveGridFocus+2)))
		}

		if key.Matches(msg, global.MoveUpKey) {
			m.moveGridFocus = int(math.Max(0, float64(m.moveGridFocus-2)))
		}

		if key.Matches(msg, global.SelectKey) {
			move := m.ctx.state.LocalPlayer.GetActivePokemon().Moves[m.moveGridFocus]

			if move != nil {
				attack := state.NewAttackAction(state.HOST, m.moveGridFocus)
				m.ctx.chosenAction = attack
			}
		}
	}

	return m, nil
}

type pokemonPanel struct {
	ctx          *gameContext
	pokemon      [6]*game.Pokemon
	validPokemon []*game.Pokemon

	selectedPokemon int
}

func newPokemonPanel(ctx *gameContext, pokemon [6]*game.Pokemon) pokemonPanel {
	panel := pokemonPanel{
		ctx:     ctx,
		pokemon: pokemon,
	}

	panel.UpdateValidPokemon()
	return panel
}

func (m pokemonPanel) Init() tea.Cmd { return nil }
func (m pokemonPanel) View() string {
	pokeStyle := lipgloss.NewStyle().Width(10).Border(lipgloss.NormalBorder(), true)

	panels := make([]string, 0)
	for i, pokemon := range m.validPokemon {
		if i == m.selectedPokemon {
			panels = append(panels, highlightedPanelStyle.Width(10).Render(pokemon.Nickname))
		} else {
			panels = append(panels, pokeStyle.Render(pokemon.Nickname))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, panels[:]...)
}

func (m pokemonPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.MoveLeftKey) {
			m.selectedPokemon--
			if m.selectedPokemon < 0 {
				m.selectedPokemon = len(m.validPokemon)
			}
		}

		if key.Matches(msg, global.MoveRightKey) {
			m.selectedPokemon++

			if m.selectedPokemon >= len(m.validPokemon) {
				m.selectedPokemon = 0
			}
		}

		if key.Matches(msg, global.SelectKey) {
			// All this extra stuff with valid pokemon
			// is just to make the menus nicer to look at (no gaps)
			currentValidPokemon := m.pokemon[m.selectedPokemon]

			// m.selectedPokemon is based off of the validPokemon slice
			// which has different indices than the actual team array
			// so we find the actual index by comparing pointers
			var currentPokemonIndex int

			for i, pokemon := range m.pokemon {
				if pokemon == currentValidPokemon {
					currentPokemonIndex = i
				}
			}

			// Only allow switches to alive and existing pokemon
			if currentValidPokemon != nil && currentValidPokemon.Hp.Value > 0 {
				action := state.NewSwitchAction(m.ctx.state, state.HOST, currentPokemonIndex)

				m.ctx.chosenAction = action
			}
		}
	}

	m.UpdateValidPokemon()

	return m, nil
}

func (m *pokemonPanel) UpdateValidPokemon() {
	newValidPokemon := make([]*game.Pokemon, 0)
	for _, pokemon := range m.pokemon {
		if pokemon != nil && pokemon.Hp.Value > 0 {
			newValidPokemon = append(newValidPokemon, pokemon)
		}
	}

	m.validPokemon = newValidPokemon
}
