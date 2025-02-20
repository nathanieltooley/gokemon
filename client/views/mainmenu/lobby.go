package mainmenu

import (
	"net"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/views/gameview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var enterKey = key.NewBinding(
	key.WithKeys("enter"),
)

var lobbyLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "lobby").Logger()
	return &logger
}

type (
	LobbyModel struct {
		backtrack *components.Breadcrumbs

		conn net.Conn
	}
	JoinLobbyModel struct {
		backtrack *components.Breadcrumbs

		conn        net.Conn
		ipTextInput textinput.Model
	}
)

type connectionAcceptedMsg net.Conn

func listenForConnection(address string) tea.Msg {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error listening on %s", address)
		return nil
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			lobbyLogger().Err(err).Str("addr", conn.RemoteAddr().String()).Msg("Error accepting connection")
			continue
		}

		lobbyLogger().Info().Str("addr", conn.RemoteAddr().String()).Msg("Host accepted connection!")

		return conn
	}
}

func connect(address string) tea.Msg {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error connection to lobby addr: %s", address)
		return nil
	}

	return conn
}

func NewLobbyHost(backtrack *components.Breadcrumbs) LobbyModel {
	return LobbyModel{backtrack: backtrack}
}

func (m LobbyModel) Init() tea.Cmd {
	return func() tea.Msg {
		log.Info().Msg("Waiting for connection")
		return listenForConnection("localhost:7777")
	}
}

func (m LobbyModel) View() string {
	header := "Lobby"
	if m.conn != nil {
		return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, "Connection Established!"))
	}

	return rendering.GlobalCenter(header)
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connectionAcceptedMsg:
		m.conn = msg

		// Send to team selection
		return gameview.NewTeamSelectModel(m.backtrack, true, m.conn, state.HOST), nil
	}

	return m, nil
}

func NewLobbyJoiner(backtrack *components.Breadcrumbs) JoinLobbyModel {
	textInput := textinput.New()
	textInput.Focus()

	return JoinLobbyModel{backtrack: backtrack, ipTextInput: textInput}
}

func (m JoinLobbyModel) Init() tea.Cmd { return nil }
func (m JoinLobbyModel) View() string {
	header := "Join Lobby"

	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.ipTextInput.View()))
}

func (m JoinLobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	newInput, tiCmd := m.ipTextInput.Update(msg)
	m.ipTextInput = newInput
	cmds = append(cmds, tiCmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, enterKey) {
			cmds = append(cmds, func() tea.Msg {
				// TODO: Input validation!
				return connect(m.ipTextInput.Value())
			})
		}
	case connectionAcceptedMsg:
		m.conn = msg

		return gameview.NewTeamSelectModel(m.backtrack, true, m.conn, state.PEER), nil
	}

	return m, tea.Batch(cmds...)
}
