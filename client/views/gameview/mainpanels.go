package gameview

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
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
					moves: m.state.GetCurrentPlayer().GetActivePokemon().Moves,
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
	state *state.GameState

	moveGridFocusRow int
	moveGridFocusCol int

	moves [4]*game.MoveFull
}

func (m movePanel) Init() tea.Cmd { return nil }
func (m movePanel) View() string {
	panels := make([]string, 0)

	for i := 0; i < 4; i++ {
		if m.moves[i] == nil {
			panels = append(panels, "Empty")
		} else {
			panels = append(panels, panelStyle.Render(m.moves[i].Name))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Center, lipgloss.JoinHorizontal(lipgloss.Center, panels[0], panels[1]), lipgloss.JoinHorizontal(lipgloss.Center, panels[2], panels[3]))
}

func (m movePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
