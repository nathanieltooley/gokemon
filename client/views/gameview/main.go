package gameview

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/game/state/stateupdater"
	"github.com/nathanieltooley/gokemon/client/rendering"
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

const (
	_MESSAGE_TIME     = time.Second * 2
	_TICKS_PER_SECOND = 20
	_TICK_TIME        = 1000 / _TICKS_PER_SECOND
)

type gameContext struct {
	state        *state.GameState
	chosenAction state.Action
}

type MainGameModel struct {
	ctx                  *gameContext
	stateUpdater         stateupdater.StateUpdater
	playerSide           int
	startedTurnResolving bool
	currentStateMessage  string
	messageQueue         []string

	inited bool

	panel tea.Model
}

func NewMainGameModel(state state.GameState, playerSide int) MainGameModel {
	ctx := &gameContext{
		state:        &state,
		chosenAction: nil,
	}

	return MainGameModel{
		ctx:          ctx,
		playerSide:   playerSide,
		stateUpdater: &stateupdater.LocalUpdater{},
		panel: actionPanel{
			ctx: ctx,
		},
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*_TICK_TIME, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m MainGameModel) Init() tea.Cmd { return nil }

func (m MainGameModel) View() string {
	panelView := ""
	if m.ctx.chosenAction == nil {
		panelView = m.panel.View()
	}

	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,

			rendering.ButtonStyle.Width(40).Render(m.currentStateMessage),

			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(m.ctx, "HOST", m.ctx.state.GetPlayer(state.HOST)).View(),
				newPlayerPanel(m.ctx, "PEER", m.ctx.state.GetPlayer(state.PEER)).View(),
			),

			panelView,
		),
	)
}

type nextNotifMsg struct{}

// Returns true if there was a message in the queue
func (m *MainGameModel) nextStateMsg() bool {
	if len(m.messageQueue) != 0 {
		// Pop queue
		m.currentStateMessage = m.messageQueue[0]
		m.messageQueue = m.messageQueue[1:]

		return true
	}

	return false
}

// TODO: There will have to be A LOT of changes for LAN or P2P Multiplayer
func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	// Debug switch action
	switch msg := msg.(type) {
	case tea.KeyMsg:
	case nextNotifMsg:
		// Only try to update the msg view later if we actually changed it
		if m.nextStateMsg() {
			cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		} else {
			m.currentStateMessage = ""
		}
	case tickMsg:
		cmds = append(cmds, tick())

	case stateupdater.ForceSwitchMessage:
		// TODO: Handle Force switch on player side
	case stateupdater.TurnResolvedMessage:
		m.panel = actionPanel{ctx: m.ctx}
		m.startedTurnResolving = false
		m.ctx.chosenAction = nil

		m.messageQueue = append(m.messageQueue, msg.Messages...)
	}

	// Force the UI into the switch pokemon panel when the player's current pokemon is dead
	if !m.ctx.state.LocalPlayer.GetActivePokemon().Alive() {
		switch m.panel.(type) {
		case pokemonPanel:
		default:
			m.panel = newPokemonPanel(m.ctx, m.ctx.state.LocalPlayer.Team)
		}
	}

	if m.ctx.chosenAction != nil && !m.startedTurnResolving {
		m.stateUpdater.SendAction(m.ctx.chosenAction)
		cmds = append(cmds, m.stateUpdater.Update(m.ctx.state))
		m.startedTurnResolving = true

	} else {
		m.panel, _ = m.panel.Update(msg)
	}

	if m.currentStateMessage == "" {
		// If there was a new message, send a cmd to go to the next one
		if m.nextStateMsg() {
			cmds = append(cmds, tea.Tick(_MESSAGE_TIME, func(t time.Time) tea.Msg {
				return nextNotifMsg{}
			}))
		}
	}

	// Game Over Check
	// NOTE: Assuming singleplayer
	gameOverValue := m.ctx.state.GameOver()
	if gameOverValue != -1 {
		if gameOverValue == m.playerSide {
			return newEndScreen("You Won!"), nil
		} else {
			return newEndScreen("You Lost :("), nil
		}
	}

	// Start tick loop
	if !m.inited {
		cmds = append(cmds, tick())
		m.inited = true
	}

	return m, tea.Batch(cmds...)
}
