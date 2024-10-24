package gameview

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/ai"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/rs/zerolog/log"
)

// BIG PROBLEM IM GONNA HAVE TO FIGURE OUT
//
// bubbletea doesn't have a framework for creating "panels" or "windows" at arbitrary locations.
// I'm gonna have to find a way to atleast put these panels in decent locations.
// Maybe a module that creates these panels at a location and makes a string of it?
// I may not deal with it ever and just make two panels with pokemon info in the center
// and a panel with pokemon actions at the bottom ¯\_(ツ)_/¯

var (
	panelStyle            = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).Padding(1, 2).AlignHorizontal(lipgloss.Center)
	highlightedPanelStyle = panelStyle.Background(rendering.HighlightedColor).Foreground(lipgloss.Color("255"))
)

type MainGameModel struct {
	state      *state.GameState
	playerSide int

	panel tea.Model
}

func NewMainGameModel(state state.GameState, playerSide int) MainGameModel {
	return MainGameModel{
		state:      &state,
		playerSide: playerSide,
		panel: actionPanel{
			state: &state,
		},
	}
}

func (m MainGameModel) Init() tea.Cmd { return nil }
func (m MainGameModel) View() string {
	panelView := ""
	if m.state.Turn() == m.playerSide {
		panelView = m.panel.View()
	} else {
		log.Debug().Msg("not your turn")
	}

	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(m.state, "HOST", m.state.GetPlayer(state.HOST)).View(),
				newPlayerPanel(m.state, "PEER", m.state.GetPlayer(state.PEER)).View(),
			),

			panelView,
		),
	)
}

type tickMsg struct {
	t time.Time
}

func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// Debug switch action
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlA {
			m.state.Update(state.SwitchAction{
				PlayerIndex: state.HOST,
				SwitchIndex: 1,
			})
		}
	case tickMsg:
		log.Debug().Msgf("Tick: %#v", msg.t.String())
	}

	if m.state.Turn() == m.playerSide {
		m.panel, _ = m.panel.Update(msg)
		cmd = tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg{t}
		})
		// Assuming singleplayer
	} else {
		m.state.RunAction(ai.BestMove(m.state))
	}

	return m, cmd
}
