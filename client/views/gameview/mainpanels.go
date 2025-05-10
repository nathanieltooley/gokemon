package gameview

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/rs/zerolog/log"
)

const playerPanelWidth = 40

var (
	playerPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder(), true).
				Padding(1, 2).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center).
				Width(playerPanelWidth).
				MarginLeft(5).
				MarginRight(5).
				Height(20)

	panelStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1, 2).AlignHorizontal(lipgloss.Center)
	highlightedPanelStyle = panelStyle.Background(rendering.HighlightedColor).Foreground(lipgloss.Color("255"))

	pokeInfoStyle = lipgloss.NewStyle().Align(lipgloss.Center).Border(lipgloss.RoundedBorder(), true).Height(5).Padding(5, 0).Width(playerPanelWidth - 8)
)

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

var typeColors map[string]lipgloss.Color = map[string]lipgloss.Color{
	game.TYPENAME_NORMAL:   lipgloss.Color("#99a2a5"),
	game.TYPENAME_FIRE:     lipgloss.Color("#e31c1c"),
	game.TYPENAME_WATER:    lipgloss.Color("#1461eb"),
	game.TYPENAME_GRASS:    lipgloss.Color("#26bd45"),
	game.TYPENAME_ELECTRIC: lipgloss.Color("#FFD400"),
	game.TYPENAME_PSYCHIC:  lipgloss.Color("#dd228d"),
	game.TYPENAME_ICE:      lipgloss.Color("#31BBCE"),
	game.TYPENAME_DRAGON:   lipgloss.Color("#1d3be2"),
	game.TYPENAME_DARK:     lipgloss.Color("#5c4733"),
	game.TYPENAME_FAIRY:    lipgloss.Color("#e66fc3"),
	game.TYPENAME_FIGHTING: lipgloss.Color("#cf8530"),
	game.TYPENAME_FLYING:   lipgloss.Color("#51b2e8"),
	game.TYPENAME_POISON:   lipgloss.Color("#A61AE5"),
	game.TYPENAME_GROUND:   lipgloss.Color("#9a6b25"),
	game.TYPENAME_ROCK:     lipgloss.Color("#d5c296"),
	game.TYPENAME_BUG:      lipgloss.Color("#99e14b"),
	game.TYPENAME_GHOST:    lipgloss.Color("#8606e6"),
	// bulbapedia has this as a light blue
	// it does look similar to normal, so maybe change?
	// but ice and flying already have light blue
	game.TYPENAME_STEEL: lipgloss.Color("#74868b"),
}

type playerPanel struct {
	gameState state.GameState

	player    *state.Player
	name      string
	healthBar progress.Model
	timer     *int64
}

func newPlayerPanel(gameState state.GameState, name string, player *state.Player, timer *int64) playerPanel {
	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.Width = playerPanelWidth * .50

	return playerPanel{
		gameState: gameState,

		player:    player,
		name:      name,
		healthBar: progressBar,
		timer:     timer,
	}
}

func pokemonEffects(pokemon game.Pokemon) string {
	panels := make([]string, 0)

	negativePanel := lipgloss.NewStyle().Background(lipgloss.Color("#ff2f2f")).Foreground(lipgloss.Color("#ffffff"))
	positivePanel := lipgloss.NewStyle().Background(lipgloss.Color("#00cf00")).Foreground(lipgloss.Color("#000000"))

	writeStat := func(statName string, statStage int) {
		statMod := game.StageMultipliers[statStage]
		if statStage > 0 {
			panels = append(panels, positivePanel.Render(fmt.Sprintf("%s: x%.2f", statName, statMod)))
		} else if statStage < 0 {
			panels = append(panels, negativePanel.Render(fmt.Sprintf("%s: x%.2f", statName, statMod)))
		}
	}

	if pokemon.ConfusionCount > 0 {
		panels = append(panels, negativePanel.Render("Confusion"))
	}

	writeStat("Attack", pokemon.Attack.GetStage())
	writeStat("Defense", pokemon.Def.GetStage())
	writeStat("SpAttack", pokemon.SpAttack.GetStage())
	writeStat("SpDef", pokemon.SpDef.GetStage())
	writeStat("Speed", pokemon.RawSpeed.GetStage())

	acc := pokemon.Accuracy()
	evasion := pokemon.Evasion()

	if acc > 1 {
		panels = append(panels, positivePanel.Render(fmt.Sprintf("Accuracy: x%.2f", acc)))
	} else if acc < 1 {
		panels = append(panels, negativePanel.Render(fmt.Sprintf("Accuracy: x%.2f", acc)))
	}

	// if evasion is greater than 1, thats bad
	if evasion > 1 {
		panels = append(panels, negativePanel.Render(fmt.Sprintf("Evasion: x%.2f", evasion)))
	} else if evasion < 1 {
		panels = append(panels, positivePanel.Render(fmt.Sprintf("Evasion: x%.2f", evasion)))
	}

	return lipgloss.JoinVertical(lipgloss.Center, panels...)
}

func (m playerPanel) Init() tea.Cmd { return nil }
func (m playerPanel) View() string {
	if m.player.ActivePokeIndex >= len(m.player.Team) || m.player.ActivePokeIndex < 0 {
		return playerPanelStyle.Render(lipgloss.JoinVertical(lipgloss.Center, m.name, "ERROR: Invalid Active PokeIndex"))
	}

	currentPokemon := m.player.Team[m.player.ActivePokeIndex]
	statusText := ""
	if currentPokemon.Status != game.STATUS_NONE {
		statusStyle := lipgloss.NewStyle().Background(statusColors[currentPokemon.Status])
		statusText = statusStyle.Render(statusTxt[currentPokemon.Status])
	}

	pokeInfo := fmt.Sprintf("%s %s\nLevel: %d\n%s",
		statusText,
		currentPokemon.Nickname,
		currentPokemon.Level,
		pokemonEffects(currentPokemon),
	)

	healthPerc := float64(currentPokemon.Hp.Value) / float64(currentPokemon.MaxHp)

	pokeInfo = pokeInfoStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pokeInfo, m.healthBar.ViewAs(healthPerc)))
	timerView := ""

	if m.gameState.Networked {
		timerView = fmt.Sprintf("Timer: %s", state.GetTimerString(*m.timer))
	}

	return playerPanelStyle.Render(lipgloss.JoinVertical(lipgloss.Center, m.name, timerView, pokeInfo))
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
		fight = highlightedPanelStyle.Width(20).Render("Fight")
	} else {
		fight = panelStyle.Width(20).Render("Fight")
	}

	if m.actionFocus == 1 {
		pokemon = highlightedPanelStyle.Width(20).Render("Pokemon")
	} else {
		pokemon = panelStyle.Width(20).Render("Pokemon")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, fight, pokemon)
}

func (m actionPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			switch m.actionFocus {
			case 0:
				switch m.ctx.playerSide {
				case state.HOST:
					return movePanel{
						ctx:   m.ctx,
						moves: m.ctx.state.HostPlayer.GetActivePokemon().InGameMoveInfo,
					}, nil
				case state.PEER:
					return movePanel{
						ctx:   m.ctx,
						moves: m.ctx.state.ClientPlayer.GetActivePokemon().InGameMoveInfo,
					}, nil
				}
			case 1:
				switch m.ctx.playerSide {
				case state.HOST:
					return newPokemonPanel(m.ctx, m.ctx.state.HostPlayer.Team), nil
				case state.PEER:
					return newPokemonPanel(m.ctx, m.ctx.state.HostPlayer.Team), nil
				}
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

	moves [4]game.BattleMove
}

func (m movePanel) Init() tea.Cmd { return nil }
func (m movePanel) View() string {
	grid := make([]string, 0)

	for i := range 2 {
		row := make([]string, 0)
		for j := range 2 {
			arrayIndex := (i * 2) + j
			style := panelStyle.Width(20)

			if arrayIndex == m.moveGridFocus {
				style = style.BorderBackground(rendering.HighlightedColor)
			}

			move := m.moves[arrayIndex]
			if move.Info.IsNil() {
				row = append(row, style.Render("Empty"))
			} else {
				// TODO: Fix centering issues with PP (lol)
				ppInfo := fmt.Sprintf("%d / %d", move.PP, move.Info.PP)
				moveColor, ok := typeColors[move.Info.Type]

				if ok {
					style = style.Background(moveColor).Foreground(rendering.BestTextColor(moveColor))
				}

				row = append(row, style.Render(lipgloss.JoinVertical(lipgloss.Center, move.Info.Name, ppInfo)))
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
			poke := m.ctx.state.HostPlayer.GetActivePokemon()
			move := poke.Moves[m.moveGridFocus]
			pp := poke.InGameMoveInfo[m.moveGridFocus].PP

			if !move.IsNil() && pp > 0 {
				attack := state.NewAttackAction(m.ctx.playerSide, m.moveGridFocus)
				m.ctx.chosenAction = attack
			}

			outOfMoves := true
			for _, m := range poke.Moves {
				if !m.IsNil() && pp > 0 {
					log.Debug().Msgf("Pokemon not out of moves, has: %s", m.Name)
					outOfMoves = false
				}
			}

			if outOfMoves {
				attack := state.NewAttackAction(m.ctx.playerSide, -1)
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
	currentSubtext  string
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
	pokemonWidth := 15
	panels := make([]string, 0)
	for i, pokemon := range m.pokemon {
		style := panelStyle

		if i == m.selectedPokemon {
			style = highlightedPanelStyle
		}

		if !pokemon.Alive() {
			style = style.BorderForeground(lipgloss.Color("#ff2f2f"))
		}

		panels = append(panels, style.Width(pokemonWidth).Render(pokemon.Nickname))
	}

	displayText := m.currentSubtext

	if m.ctx.forcedSwitch && displayText != "" {
		displayText = "Your Pokemon fainted, please select a new one to switch in"
	}

	return lipgloss.JoinVertical(lipgloss.Center, displayText, lipgloss.JoinHorizontal(lipgloss.Center, panels[:]...))
}

func (m pokemonPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearTextMsg:
		m.currentSubtext = ""
	case tea.KeyMsg:
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

			if !currentValidPokemon.Alive() {
				m.currentSubtext = "This pokemon is not alive"
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					return clearTextMsg{t}
				})
			}

			if m.selectedPokemon == m.ctx.state.HostPlayer.ActivePokeIndex {
				m.currentSubtext = "This pokemon is already active!"
				return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
					return clearTextMsg{t}
				})
			}

			// Only allow switches to alive and existing pokemon
			if currentValidPokemon.Alive() {
				action := state.NewSwitchAction(m.ctx.state, state.HOST, m.selectedPokemon)

				m.ctx.chosenAction = action
			}
		}
	}

	return m, nil
}

type clearTextMsg struct {
	time.Time
}
