package gameview

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

const playerPanelWidth = 20

var statusColors map[int]lipgloss.Color = map[int]lipgloss.Color{
	game.STATUS_BURN:   lipgloss.Color("#E36D1C"),
	game.STATUS_PARA:   lipgloss.Color("#FFD400"),
	game.STATUS_TOXIC:  lipgloss.Color("#A61AE5"),
	game.STATUS_POISON: lipgloss.Color("#A61AE5"),
	game.STATUS_FROZEN: lipgloss.Color("#31BBCE"),
	game.STATUS_SLEEP:  lipgloss.Color("#BCE9EF"),
}

var statusTxt map[int]string = map[int]string{
	game.STATUS_BURN:   "BRN",
	game.STATUS_PARA:   "PAR",
	game.STATUS_FROZEN: "FRZ",
	game.STATUS_TOXIC:  "TOX",
	game.STATUS_POISON: "PSN",
	game.STATUS_SLEEP:  "SLP",
}

type playerPanel struct {
	gameState state.GameState

	player    *state.Player
	name      string
	healthBar progress.Model
}

func newPlayerPanel(gameState state.GameState, name string, player *state.Player) playerPanel {
	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.Width = playerPanelWidth * .75

	return playerPanel{
		gameState: gameState,

		player:    player,
		name:      name,
		healthBar: progressBar,
	}
}

func (m playerPanel) Init() tea.Cmd { return nil }
func (m playerPanel) View() string {
	currentPokemon := m.player.Team[m.player.ActivePokeIndex]
	statusText := ""
	if currentPokemon.Status != game.STATUS_NONE {
		statusStyle := lipgloss.NewStyle().Background(statusColors[currentPokemon.Status])
		statusText = statusStyle.Render(statusTxt[currentPokemon.Status])
	}

	pokeInfo := fmt.Sprintf("%s %s\nLevel: %d",
		statusText,
		currentPokemon.Nickname,
		currentPokemon.Level,
	)

	healthPerc := float64(currentPokemon.Hp.Value) / float64(currentPokemon.MaxHp)

	pokeStyle := lipgloss.NewStyle().Align(lipgloss.Center).Border(lipgloss.NormalBorder(), true).Width(playerPanelWidth).Height(5)
	pokeInfo = pokeStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pokeInfo, m.healthBar.ViewAs(healthPerc)))

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Center, m.name, pokeInfo))
}

func (m playerPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	progressModel, _ := m.healthBar.Update(msg)
	m.healthBar = progressModel.(progress.Model)

	return m, nil
}

type actionPanel struct {
	ctx *gameContext

	actionFocus int
}

func newActionPanel(ctx *gameContext) actionPanel {
	return actionPanel{
		ctx: ctx,
	}
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
				row = append(row, style.Render(m.moves[arrayIndex].Name))
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
	ctx     *gameContext
	pokemon []game.Pokemon

	selectedPokemon int
}

func newPokemonPanel(ctx *gameContext, pokemon []game.Pokemon) pokemonPanel {
	panel := pokemonPanel{
		ctx:     ctx,
		pokemon: pokemon,
	}

	return panel
}

func (m pokemonPanel) Init() tea.Cmd { return nil }
func (m pokemonPanel) View() string {
	pokeStyle := lipgloss.NewStyle().Width(10).Border(lipgloss.NormalBorder(), true)

	panels := make([]string, 0)
	// TODO: Gray out or get rid of dead pokemon
	for i, pokemon := range m.pokemon {
		if i == m.selectedPokemon {
			panels = append(panels, highlightedPanelStyle.Width(10).Render(pokemon.Nickname))
		} else {
			panels = append(panels, pokeStyle.Render(pokemon.Nickname))
		}
	}

	forcedSwitch := ""
	if m.ctx.forcedSwitch {
		forcedSwitch = "Your Pokemon fainted, please select a new one to switch in"
	}

	return lipgloss.JoinVertical(lipgloss.Center, forcedSwitch, lipgloss.JoinHorizontal(lipgloss.Center, panels[:]...))
}

func (m pokemonPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// TODO: Add backing out of the menu

		if key.Matches(msg, global.MoveLeftKey) {
			m.selectedPokemon--
			if m.selectedPokemon < 0 {
				m.selectedPokemon = len(m.pokemon)
			}
		}

		if key.Matches(msg, global.MoveRightKey) {
			m.selectedPokemon++

			if m.selectedPokemon >= len(m.pokemon) {
				m.selectedPokemon = 0
			}
		}

		if key.Matches(msg, global.SelectKey) {
			currentValidPokemon := m.pokemon[m.selectedPokemon]

			// Only allow switches to alive and existing pokemon
			if currentValidPokemon.Alive() {
				action := state.NewSwitchAction(m.ctx.state, state.HOST, m.selectedPokemon)

				m.ctx.chosenAction = action
			}
		}
	}

	return m, nil
}
