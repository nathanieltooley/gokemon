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
	state *state.GameState

	player *state.Player
	name   string
}

func newPlayerPanel(state *state.GameState, name string, player *state.Player) playerPanel {
	return playerPanel{
		state:  state,
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
	state *state.GameState

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
					state: m.state,
					moves: m.state.LocalPlayer.GetActivePokemon().Moves,
				}, nil
			case 1:
				return pokemonPanel{
					state:   m.state,
					pokemon: m.state.LocalPlayer.Team,
				}, nil
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
	state         *state.GameState
	moveGridFocus int

	moves [4]*game.MoveFull
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
			move := m.state.LocalPlayer.GetActivePokemon().Moves[m.moveGridFocus]

			if move != nil {
				attack := state.NewAttackAction(state.HOST, m.moveGridFocus)
				m.state.LocalSubmittedAction = attack
			}
		}
	}

	return m, nil
}

type pokemonPanel struct {
	state   *state.GameState
	pokemon [6]*game.Pokemon

	selectedPokemon int
}

func (m pokemonPanel) Init() tea.Cmd { return nil }
func (m pokemonPanel) View() string {
	pokeStyle := lipgloss.NewStyle().Width(10).Border(lipgloss.NormalBorder(), true)

	var panel [6]string
	for i := 0; i < 6; i++ {
		pokemon := m.pokemon[i]

		if pokemon != nil {
			if i == m.selectedPokemon {
				panel[i] = highlightedPanelStyle.Width(10).Render(pokemon.Nickname)
			} else {
				panel[i] = pokeStyle.Render(pokemon.Nickname)
			}
		} else {
			panel[i] = pokeStyle.Render()
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, panel[:]...)
}

func (m pokemonPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.MoveLeftKey) {
			m.selectedPokemon--
			if m.selectedPokemon < 0 {
				m.selectedPokemon = 5
			}
		}

		if key.Matches(msg, global.MoveRightKey) {
			m.selectedPokemon++

			if m.selectedPokemon > 5 {
				m.selectedPokemon = 0
			}
		}

		if key.Matches(msg, global.SelectKey) {
			currentPokemon := m.pokemon[m.selectedPokemon]

			if currentPokemon != nil && currentPokemon.Hp.Value > 0 {
				action := state.SwitchAction{
					PlayerIndex: state.HOST,
					SwitchIndex: m.selectedPokemon,
				}

				m.state.LocalSubmittedAction = action
			}

		}
	}

	return m, nil
}
