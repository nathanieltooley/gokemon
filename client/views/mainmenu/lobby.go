package mainmenu

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/views/gameview"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var lobbyLogger = func() *zerolog.Logger {
	logger := log.With().Str("location", "lobby").Logger()
	return &logger
}

const (
	connPort          = ":7777"
	broadPort         = ":7778"
	broadResponsePort = ":7779"
	broadcastAddr     = "255.255.255.255"
	broadcastMessage  = "GOKEMON|SEARCH"
)

type (
	LobbyModel struct {
		backtrack components.Breadcrumbs

		conn          net.Conn
		lobbyName     string
		nameInput     textinput.Model
		inputtingName bool
	}
	JoinLobbyModel struct {
		backtrack components.Breadcrumbs

		focus       int
		conn        net.Conn
		lobbyList   list.Model
		ipTextInput textinput.Model
	}
)

type lobby struct {
	name string
	addr string
}

func (l lobby) FilterValue() string {
	return l.name
}

func (l lobby) Value() string {
	return l.name
}

type lanSearchResult struct {
	lob  *lobby // pointer for optional
	conn *net.UDPConn
}

// simple bubbletea msgs
type (
	connectionAcceptedMsg    net.Conn
	repeatLanSearchBroadcast time.Time
)

type continueLanSearchMsg struct {
	// HACK: makes it a different type from connectionAcceptedMsg
	_dummy int
	conn   *net.UDPConn
}

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

func sendLanSearchBroadcast() tea.Msg {
	// local, _ := net.ResolveUDPAddr("udp4", broadPort)
	// remote, _ := net.ResolveUDPAddr("udp4", broadcastAddr+broadPort)
	// conn, err := net.DialUDP("udp", local, remote)
	// if err != nil {
	// 	lobbyLogger().Err(err).Msgf("Error trying to connect to UDP broadcastAddr: %s->%s", "0.0.0.0:"+broadPort, broadcastAddr+broadPort)
	// 	return nil
	// }
	//

	conn, err := net.Dial("udp4", broadcastAddr+broadPort)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error trying to connect to UDP broadcastAddr: %s", broadcastAddr+broadPort)
	}

	_, err = conn.Write([]byte(broadcastMessage))
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error trying to send broadcast")
		return nil
	}

	lobbyLogger().Debug().Msg("lan search broadcast sent!")
	return nil
}

func listenForLanBroadcastResult(conn *net.UDPConn) tea.Msg {
	if conn == nil {
		var connErr error

		laddr, _ := net.ResolveUDPAddr("udp4", broadResponsePort)
		conn, connErr = net.ListenUDP("udp", laddr)
		if connErr != nil {
			lobbyLogger().Err(connErr).Msg("Error trying to listen for LAN servers")
			return nil
		}
	}

	lobbyLogger().Info().Msgf("client listening for response on: %s|%s", conn.LocalAddr(), conn.RemoteAddr())

	broadResponse := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(broadResponse)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error trying to read message from: %s", addr)
	}

	lobbyLogger().Info().Msgf("Client got message from: %s", addr)

	// Response should look like: GOKEMON:NAME:ADDR
	responseParts := strings.Split(string(broadResponse[0:n]), "|")
	if len(responseParts) < 3 || responseParts[0] != "GOKEMON" {
		lobbyLogger().Debug().Msgf("Read malformed or non Gokemon broadcast message: %s", broadResponse)
		return lanSearchResult{
			lob:  nil,
			conn: conn,
		}
	}

	lob := lobby{
		name: responseParts[1],
		addr: responseParts[2],
	}

	lobbyLogger().Info().Msgf("Found lobby: %+v", lob)

	return lanSearchResult{
		&lob,
		conn,
	}
}

func listenForSearch(conn *net.UDPConn) tea.Msg {
	if conn == nil {
		laddr, _ := net.ResolveUDPAddr("udp4", broadPort)
		var err error
		conn, err = net.ListenUDP("udp4", laddr)
		if err != nil {
			lobbyLogger().Err(err).Msg("Error for host trying to connect to UDP broadcastAddr")
			return nil
		}
		lobbyLogger().Debug().Msgf("host listening for searches on: %s", conn.LocalAddr())
	}

	buf := make([]byte, 1024)
	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Host failed to listen to UDP broadcast")
	}
	lobbyLogger().Debug().Msgf("HIT! %s; FROM!: %s", buf[0:n], addr.String())
	message := string(buf[0:n])

	// Send Response
	if message == broadcastMessage {
		broadcastAddrUdp, _ := net.ResolveUDPAddr("udp4", broadcastAddr+broadResponsePort)
		selfAddrTcp, err := net.ResolveTCPAddr("tcp4", connPort)
		// TODO: Maybe remove this? Should probably never actually error though right?
		if err != nil {
			panic(err)
		}

		_, err = conn.WriteToUDP(fmt.Appendf(nil, "GOKEMON|BLANKNAME|%s", selfAddrTcp.String()), broadcastAddrUdp)
		if err != nil {
			lobbyLogger().Err(err).Msgf("Host failed to send LAN search response")
		}

		lobbyLogger().Info().Msg("Sent broadcast response")
	}

	return continueLanSearchMsg{conn: conn}
}

func NewLobbyHost(backtrack components.Breadcrumbs) LobbyModel {
	return LobbyModel{backtrack: backtrack}
}

func (m LobbyModel) Init() tea.Cmd {
	initCmds := make([]tea.Cmd, 0)

	initCmds = append(initCmds, func() tea.Msg {
		lobbyLogger().Info().Msg("Waiting for connection")
		return listenForConnection(connPort)
	})

	initCmds = append(initCmds, func() tea.Msg {
		lobbyLogger().Info().Msg("Waiting for LAN searches")
		return listenForSearch(nil)
	})

	return tea.Batch(initCmds...)
}

func (m LobbyModel) View() string {
	header := "Lobby"
	if m.conn != nil {
		return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, "Connection Established!"))
	}

	return rendering.GlobalCenter(header)
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.BackKey) {
			return m.backtrack.PopDefault(func() tea.Model {
				return NewLobbyHost(m.backtrack)
			}), nil
		}
	case connectionAcceptedMsg:
		m.conn = msg

		// Send to team selection
		return gameview.NewTeamSelectModel(m.backtrack.PushNew(func() tea.Model {
			return NewLobbyHost(m.backtrack)
		}), true, m.conn, state.HOST), nil
	case continueLanSearchMsg:
		cmds = append(cmds, func() tea.Msg {
			return listenForSearch(msg.conn)
		})
	}

	return m, tea.Batch(cmds...)
}

func NewLobbyJoiner(backtrack components.Breadcrumbs) JoinLobbyModel {
	lobbyList := make([]list.Item, 0)
	list := list.New(lobbyList, rendering.NewSimpleListDelegate(), global.TERM_WIDTH/2, global.TERM_HEIGHT/2)
	list.DisableQuitKeybindings()

	textInput := textinput.New()
	textInput.Focus()

	return JoinLobbyModel{backtrack: backtrack, ipTextInput: textInput, lobbyList: list}
}

func (m JoinLobbyModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0)
	cmds = append(cmds, func() tea.Msg {
		return sendLanSearchBroadcast()
	})

	cmds = append(cmds, func() tea.Msg {
		return listenForLanBroadcastResult(nil)
	})

	cmds = append(cmds, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return repeatLanSearchBroadcast(t)
	}))

	return tea.Batch(cmds...)
}

func (m JoinLobbyModel) View() string {
	header := "Join Lobby"

	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.lobbyList.View(), m.ipTextInput.View()))
}

func (m JoinLobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			switch m.focus {
			case 0:
				cmds = append(cmds, func() tea.Msg {
					selectedLobby := m.lobbyList.SelectedItem().(lobby)
					return connect(selectedLobby.addr)
				})
			case 1:
				cmds = append(cmds, func() tea.Msg {
					// TODO: Input validation!
					return connect(m.ipTextInput.Value())
				})
			}
		}

		if key.Matches(msg, global.BackKey) {
			return m.backtrack.PopDefault(func() tea.Model {
				return NewLobbyJoiner(m.backtrack)
			}), nil
		}

		if msg.Type == tea.KeyTab {
			m.focus++

			if m.focus > 1 {
				m.focus = 0
			}
		}

		if msg.Type == tea.KeyShiftTab {
			m.focus--
			if m.focus < 0 {
				m.focus = 1
			}
		}
	case connectionAcceptedMsg:
		m.conn = msg

		return gameview.NewTeamSelectModel(m.backtrack.PushNew(func() tea.Model {
			return NewLobbyJoiner(m.backtrack)
		}), true, m.conn, state.PEER), nil
	case lanSearchResult:
		log.Debug().Msgf("Got search result: %+v", *msg.lob)
		if msg.lob != nil {
			var lobItem list.Item = *msg.lob

			alreadyAdded := false
			for _, item := range m.lobbyList.Items() {
				itemAsLob := item.(lobby)
				if itemAsLob.addr == msg.lob.addr {
					alreadyAdded = true
				}
			}

			if !alreadyAdded {
				cmds = append(cmds, m.lobbyList.InsertItem(0, lobItem))
			}
		}

		cmds = append(cmds, func() tea.Msg {
			return listenForLanBroadcastResult(msg.conn)
		})
	case repeatLanSearchBroadcast:
		sendLanSearchBroadcast()

		cmds = append(cmds, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
			return repeatLanSearchBroadcast(t)
		}))
	}

	switch m.focus {
	case 0:
		var cmd tea.Cmd
		m.lobbyList, cmd = m.lobbyList.Update(msg)
		cmds = append(cmds, cmd)
	case 1:
		newInput, tiCmd := m.ipTextInput.Update(msg)
		m.ipTextInput = newInput
		cmds = append(cmds, tiCmd)
	}

	return m, tea.Batch(cmds...)
}
