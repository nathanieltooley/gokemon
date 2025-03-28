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
	"github.com/nathanieltooley/gokemon/client/networking"
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
	CreateLobbyModel struct {
		backtrack components.Breadcrumbs

		nameInput textinput.Model
		focus     int
	}
	LobbyModel struct {
		backtrack components.Breadcrumbs

		conn       net.Conn
		lobbyName  string
		hosting    bool
		playerList list.Model
		focus      int
	}
	JoinLobbyModel struct {
		backtrack components.Breadcrumbs

		conn         net.Conn
		lobbyList    list.Model
		enteringName bool
		nameInput    textinput.Model
		playerName   string
	}
)

type lobbyPlayer struct {
	Name   string
	Addr   string
	ConnId int
}

type lobby struct {
	Name     string
	Addr     string
	HostName string
}

func (l lobby) FilterValue() string {
	return l.Name
}

func (l lobby) Value() string {
	return l.Name
}

func (l lobbyPlayer) FilterValue() string {
	return l.Name
}

func (l lobbyPlayer) Value() string {
	return l.Name
}

type lanSearchResult struct {
	lob  *lobby // pointer for optional
	conn *net.UDPConn
}

// simple bubbletea msgs
type (
	connectionAcceptedMsgHost struct {
		conn       net.Conn
		clientData lobbyPlayer
	}
	connectionAcceptedMsgClient struct {
		conn      net.Conn
		lobbyData lobby
	}
	repeatLanSearchBroadcast time.Time
	startGameMsg             struct {
		_dummy int
	}
)

type continueLanSearchMsg struct {
	// HACK: makes it a different type from connectionAcceptedMsg
	_dummy int
	conn   *net.UDPConn
}

func listenForConnection(address string, lobbyInfo lobby) tea.Msg {
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

		clientData, err := networking.AcceptData[lobbyPlayer](conn)
		if err != nil {
			lobbyLogger().Err(err).Str("addr", conn.RemoteAddr().String()).Msg("Error getting client data")
			continue
		}

		if err := networking.SendData(conn, lobbyInfo); err != nil {
			lobbyLogger().Err(err).Str("addr", conn.RemoteAddr().String()).Msg("Error sending lobby data to client")
			continue
		}

		return connectionAcceptedMsgHost{
			conn,
			clientData,
		}
	}
}

func connect(address string, playerName string) tea.Msg {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error connection to lobby addr: %s", address)
		return nil
	}

	if err := networking.SendData(conn, lobbyPlayer{playerName, conn.LocalAddr().String(), state.PEER}); err != nil {
		lobbyLogger().Err(err).Msg("Error sending client data to host")
		return nil
	}

	lobbyData, err := networking.AcceptData[lobby](conn)
	if err != nil {
		lobbyLogger().Err(err).Msg("Error trying to receive lobby data")
		return nil
	}

	// discard client data part
	return connectionAcceptedMsgClient{
		conn:      conn,
		lobbyData: lobbyData,
	}
}

func sendLanSearchBroadcast() tea.Msg {
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
		Name: responseParts[1],
		Addr: responseParts[2],
	}

	lobbyLogger().Info().Msgf("Found lobby: %+v", lob)

	return lanSearchResult{
		&lob,
		conn,
	}
}

func listenForSearch(conn *net.UDPConn, name string) tea.Msg {
	if name == "" {
		name = "new_lobby"
	}

	name = strings.ReplaceAll(name, "|", "")

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

		_, err = conn.WriteToUDP(fmt.Appendf(nil, "GOKEMON|%s|%s", name, selfAddrTcp.String()), broadcastAddrUdp)
		if err != nil {
			lobbyLogger().Err(err).Msgf("Host failed to send LAN search response")
		}

		lobbyLogger().Info().Msg("Sent broadcast response")
	}

	return continueLanSearchMsg{conn: conn}
}

func listenForStart(conn net.Conn) tea.Msg {
	for {
		log.Debug().Msg("waiting for start")
		data := make([]byte, 512)
		n, err := conn.Read(data)
		if err != nil {
			lobbyLogger().Err(err).Msg("Failed to listen for start message")
			continue
		}

		message := string(data[:n])

		if message != "GOKEMON|START" {
			lobbyLogger().Info().Str("msg", string(data)).Msg("Sent() incorrect message!")
			continue
		}

		lobbyLogger().Debug().Msg("client told to start game")

		return startGameMsg{}
	}
}

func NewLobbyCreater(backtrack components.Breadcrumbs) CreateLobbyModel {
	textInput := textinput.New()
	textInput.Placeholder = "new_lobby"
	textInput.CharLimit = 20
	textInput.Focus()

	return CreateLobbyModel{backtrack: backtrack, nameInput: textInput}
}

func (m CreateLobbyModel) Init() tea.Cmd { return nil }
func (m CreateLobbyModel) View() string {
	header := "Lobby Creation"
	createButton := rendering.ButtonStyle.Render("Create Lobby")
	if m.focus == 1 {
		createButton = rendering.HighlightedButtonStyle.Render("Create Lobby")
	}
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.nameInput.View(), createButton))
}

func (m CreateLobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.BackKey) {
			return m.backtrack.PopDefault(func() tea.Model {
				return NewLobbyCreater(m.backtrack)
			}), nil
		}

		if key.Matches(msg, global.DownTabKey) {
			m.focus++

			if m.focus > 1 {
				m.focus = 0
			}
		}

		if key.Matches(msg, global.UpTabKey) {
			m.focus--
			if m.focus < 0 {
				m.focus = 1
			}
		}

		if m.focus == 1 && key.Matches(msg, global.SelectKey) {
			lobbyName := m.nameInput.Value()
			addr, _ := net.ResolveTCPAddr("tcp4", connPort)

			cmds = append(cmds, func() tea.Msg {
				lobbyLogger().Info().Msg("Waiting for connection")
				return listenForConnection(connPort, lobby{lobbyName, addr.String(), global.LocalPlayerName})
			})

			cmds = append(cmds, func() tea.Msg {
				lobbyLogger().Info().Msg("Waiting for LAN searches")
				return listenForSearch(nil, lobbyName)
			})

			players := make([]list.Item, 0)
			players = append(players, lobbyPlayer{
				Name: global.LocalPlayerName,
				Addr: addr.String(),
			})

			// Height is shorter than client to account for startGameButton
			playerList := list.New(players, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT-15)

			return LobbyModel{
				backtrack:  m.backtrack,
				hosting:    true,
				playerList: playerList,
				lobbyName:  lobbyName,
			}, tea.Batch(cmds...)
		}
	}

	if m.focus == 0 {
		cmds = append(cmds, m.nameInput.Focus())
		newInput, cmd := m.nameInput.Update(msg)
		m.nameInput = newInput
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func NewLobbyJoiner(backtrack components.Breadcrumbs) JoinLobbyModel {
	lobbyList := make([]list.Item, 0)
	list := list.New(lobbyList, rendering.NewSimpleListDelegate(), global.TERM_WIDTH/2, global.TERM_HEIGHT/2)
	list.DisableQuitKeybindings()

	nameInput := textinput.New()
	nameInput.Prompt = global.LocalPlayerName

	return JoinLobbyModel{backtrack: backtrack, lobbyList: list, nameInput: nameInput}
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
	if !m.enteringName {
		header := "Join Lobby"
		return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.lobbyList.View()))
	} else {
		header := "Join with name..."
		return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.nameInput.View()))
	}
}

func (m JoinLobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			if m.enteringName {
				m.playerName = m.nameInput.Value()
				cmds = append(cmds, func() tea.Msg {
					selectedLobby := m.lobbyList.SelectedItem().(lobby)
					return connect(selectedLobby.Addr, m.playerName)
				})
			} else {
				m.enteringName = true
				m.nameInput = textinput.New()
				cmds = append(cmds, m.nameInput.Focus())
			}
		}

		if key.Matches(msg, global.BackKey) {
			if m.enteringName {
				m.enteringName = false
			} else {
				return m.backtrack.PopDefault(func() tea.Model {
					return NewLobbyJoiner(m.backtrack)
				}), nil
			}
		}

	case connectionAcceptedMsgClient:
		players := make([]list.Item, 0)
		players = append(players, lobbyPlayer{
			Name: msg.lobbyData.HostName,
			Addr: msg.lobbyData.Addr,
		})
		players = append(players, lobbyPlayer{
			Name: m.playerName,
			Addr: msg.conn.LocalAddr().String(),
		})

		cmds = append(cmds, func() tea.Msg {
			return listenForStart(msg.conn)
		})

		return LobbyModel{
			backtrack:  components.Breadcrumbs{},
			conn:       msg.conn,
			hosting:    false,
			lobbyName:  msg.lobbyData.Name,
			playerList: list.New(players, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT-5),
		}, tea.Batch(cmds...)
	case lanSearchResult:
		log.Debug().Msgf("Got search result: %+v", *msg.lob)
		if msg.lob != nil {
			var lobItem list.Item = *msg.lob

			alreadyAdded := false
			for _, item := range m.lobbyList.Items() {
				itemAsLob := item.(lobby)
				if itemAsLob.Addr == msg.lob.Addr {
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

	var cmd tea.Cmd
	m.lobbyList, cmd = m.lobbyList.Update(msg)
	cmds = append(cmds, cmd)

	if m.enteringName {
		var cmd tea.Cmd
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m LobbyModel) Init() tea.Cmd { return nil }

func (m LobbyModel) View() string {
	header := fmt.Sprintf("Lobby: %s", m.lobbyName)

	startGameButton := rendering.HighlightedButtonStyle.Render("Start Game!")
	if !m.hosting {
		startGameButton = ""
	}

	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, m.playerList.View(), startGameButton))
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			// TODO: Get rid of blocking code
			_, err := m.conn.Write([]byte("GOKEMON|START"))
			lobbyLogger().Debug().Msg("host sent start msg")

			clientName := ""
			for _, item := range m.playerList.Items() {
				player := item.(lobbyPlayer)
				if player.ConnId == state.PEER {
					clientName = player.Name
				}
			}

			netInfo := gameview.NetworkingInfo{
				Conn:       m.conn,
				ConnId:     state.HOST,
				ClientName: clientName,
			}

			if err == nil {
				// Send to team selection
				// TODO: Have backing out from here send you to main menu
				return gameview.NewTeamSelectModel(components.NewBreadcrumb(), &netInfo), nil
			} else {
				lobbyLogger().Err(err).Msg("Failed to send start message")
			}

		}
	case continueLanSearchMsg:
		cmds = append(cmds, func() tea.Msg {
			return listenForSearch(msg.conn, m.lobbyName)
		})
	case connectionAcceptedMsgHost:
		m.conn = msg.conn

		m.playerList.InsertItem(-1, msg.clientData)
	case startGameMsg:
		return gameview.NewTeamSelectModel(components.NewBreadcrumb(), &gameview.NetworkingInfo{Conn: m.conn, ConnId: state.PEER}), nil
	}

	return m, tea.Batch(cmds...)
}
