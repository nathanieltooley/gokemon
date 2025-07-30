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
	"github.com/google/uuid"
	stateCore "github.com/nathanieltooley/gokemon/client/game/state/core"
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
	// What port the actual game connection will be sent through
	connPort = ":7777"
	// The port the broadcast searches will be sent to and the host will listen to
	broadPort = ":7778"
	// The port the client listens to for responses to the search
	broadResponsePort = ":7779"
	// broadcastAddr     = "255.255.255.255"
	broadcastMessage = "GOKEMON|SEARCH"
)

type (
	CreateLobbyModel struct {
		backtrack components.Breadcrumbs

		nameInput textinput.Model
		focus     int
	}
	LobbyModel struct {
		backtrack components.Breadcrumbs

		conn      net.Conn
		lobbyInfo lobby
		hosting   bool
		host      lobbyPlayer
		opponent  lobbyPlayer
		focus     int
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

func (l lobbyPlayer) IsNil() bool {
	return l.Name == "" && l.Addr == ""
}

type lobby struct {
	Name     string
	Addr     string
	HostName string
	Id       uuid.UUID
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

	if err := networking.SendData(conn, lobbyPlayer{playerName, conn.LocalAddr().String(), stateCore.PEER}); err != nil {
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
	for _, ip := range networking.GetAllBroadcastAddrs() {
		raddr, _ := net.ResolveUDPAddr("udp4", ip.String()+broadPort)
		conn, err := net.DialUDP("udp4", nil, raddr)
		if err != nil {
			lobbyLogger().Err(err).Msgf("Error trying to connect to UDP broadcastAddr: %s", ip.String()+broadPort)
			continue
		}

		_, err = conn.Write([]byte(broadcastMessage))
		if err != nil {
			lobbyLogger().Warn().Msgf("Error trying to send broadcast: %s", err)
		} else {
			lobbyLogger().Debug().Msgf("lan search broadcast sent on: %s", conn.RemoteAddr())
		}

	}

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

	lobbyLogger().Info().Msgf("client (IP: %s) listening for response on: %s", conn.LocalAddr(), conn.RemoteAddr())

	broadResponse := make([]byte, 1024)
	n, responderAddr, err := conn.ReadFromUDP(broadResponse)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Error trying to read message from: %s", responderAddr)
	}

	lobbyLogger().Info().Msgf("Client got message from: %s", responderAddr)

	// Response should look like: GOKEMON:NAME:PORT
	responseParts := strings.Split(string(broadResponse[0:n]), "|")
	if len(responseParts) < 3 || responseParts[0] != "GOKEMON" {
		lobbyLogger().Debug().Msgf("Read malformed or non Gokemon broadcast message: %s", broadResponse)
		return lanSearchResult{
			lob:  nil,
			conn: conn,
		}
	}

	lobbyName := responseParts[1]
	lobbyPort := responseParts[2]
	lobbyId := responseParts[3]

	lobbyAddr := responderAddr.IP.String() + lobbyPort

	lob := lobby{
		Name: lobbyName,
		Addr: lobbyAddr,
		Id:   uuid.MustParse(lobbyId),
	}

	lobbyLogger().Info().Msgf("Found lobby: %+v", lob)

	return lanSearchResult{
		&lob,
		conn,
	}
}

func listenForSearch(searchConn *net.UDPConn, lobbyInfo lobby) tea.Msg {
	broadcastedName := lobbyInfo.Name
	if broadcastedName == "" {
		broadcastedName = "new_lobby"
	}

	broadcastedName = strings.ReplaceAll(broadcastedName, "|", "")

	// Setup connection if it is the first time this function is called
	if searchConn == nil {
		// I tried doing broadAddr + broadPort but windows doesn't like it
		// I think the host should still be able to read the messages though by just connecting to a port with UDP
		laddr, _ := net.ResolveUDPAddr("udp4", broadPort)
		var err error
		searchConn, err = net.ListenUDP("udp4", laddr)
		if err != nil {
			lobbyLogger().Err(err).Msgf("Error for host trying to connect to UDP addr: %s", laddr.String())
			return nil
		}
		lobbyLogger().Debug().Msgf("host listening for searches on: %s", searchConn.LocalAddr())
	}

	buf := make([]byte, 1024)
	n, senderAddr, err := searchConn.ReadFromUDP(buf)
	if err != nil {
		lobbyLogger().Err(err).Msgf("Host failed to listen to UDP broadcast")
		return continueLanSearchMsg{conn: searchConn}
	}

	lobbyLogger().Debug().Msgf("HIT! %s; FROM!: %s", buf[0:n], senderAddr.String())
	message := string(buf[0:n])

	// Send Response
	if message == broadcastMessage {
		returnUdpAddr, _ := net.ResolveUDPAddr("udp4", senderAddr.IP.String()+broadResponsePort)

		// Send host's port to the searcher's IP (IP is implied)
		_, err = searchConn.WriteToUDP(fmt.Appendf(nil, "GOKEMON|%s|%s|%s", broadcastedName, connPort, lobbyInfo.Id.String()), returnUdpAddr)
		if err != nil {
			lobbyLogger().Err(err).Msgf("Host failed to send LAN search response")
		} else {
			lobbyLogger().Info().Msgf("Sent broadcast response to: %s", returnUdpAddr.String())
		}

	}

	return continueLanSearchMsg{conn: searchConn}
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
			lobbyLogger().Info().Str("msg", string(data)).Msg("Host sent incorrect message!")
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

			newId := uuid.New()
			lobbyInfo := lobby{
				Name:     lobbyName,
				Addr:     addr.String(),
				HostName: global.Opt.LocalPlayerName,
				Id:       newId,
			}

			cmds = append(cmds, func() tea.Msg {
				lobbyLogger().Info().Msg("Waiting for connection")
				return listenForConnection(connPort, lobbyInfo)
			})

			cmds = append(cmds, func() tea.Msg {
				lobbyLogger().Info().Msg("Waiting for LAN searches")
				return listenForSearch(nil, lobbyInfo)
			})

			return LobbyModel{
				backtrack: m.backtrack,
				hosting:   true,
				host:      lobbyPlayer{Name: global.Opt.LocalPlayerName, Addr: addr.String()},
				lobbyInfo: lobbyInfo,
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
	nameInput.Prompt = global.Opt.LocalPlayerName

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
		host := lobbyPlayer{Name: msg.lobbyData.HostName, Addr: msg.lobbyData.Addr}
		opponent := lobbyPlayer{Name: m.playerName, Addr: msg.conn.LocalAddr().String()}

		cmds = append(cmds, func() tea.Msg {
			return listenForStart(msg.conn)
		})

		return LobbyModel{
			backtrack: components.Breadcrumbs{},
			conn:      msg.conn,
			hosting:   false,
			lobbyInfo: msg.lobbyData,
			host:      host,
			opponent:  opponent,
		}, tea.Batch(cmds...)
	case lanSearchResult:
		log.Debug().Msgf("Got search result: %+v", *msg.lob)
		if msg.lob != nil {
			var lobItem list.Item = *msg.lob

			alreadyAdded := false
			for _, item := range m.lobbyList.Items() {
				itemAsLob := item.(lobby)
				if itemAsLob.Id == msg.lob.Id {
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
	header := fmt.Sprintf("Lobby: %s", m.lobbyInfo.Name)

	startGameButton := rendering.HighlightedButtonStyle.Render("Start Game!")
	if !m.hosting {
		startGameButton = ""
	}

	nameStyle := lipgloss.NewStyle().Margin(2).Padding(5).Border(lipgloss.NormalBorder(), true)

	hostView := nameStyle.Render("Host: " + m.host.Name)
	opponentView := nameStyle.Render("Client: " + m.opponent.Name)
	if m.opponent.IsNil() {
		opponentView = "Waiting for client . . . "
	}

	nameViews := lipgloss.JoinHorizontal(lipgloss.Center, hostView, opponentView)

	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, header, nameViews, startGameButton))
}

func (m LobbyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Start game on enter
		if key.Matches(msg, global.SelectKey) && !m.opponent.IsNil() && m.hosting {
			// TODO: Get rid of blocking code
			_, err := m.conn.Write([]byte("GOKEMON|START"))
			lobbyLogger().Debug().Msg("host sent start msg")

			netInfo := gameview.NetworkingInfo{
				Conn:         m.conn,
				ConnId:       stateCore.HOST,
				OpposingName: m.opponent.Name,
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
			return listenForSearch(msg.conn, m.lobbyInfo)
		})
	case connectionAcceptedMsgHost:
		m.conn = msg.conn

		m.opponent = msg.clientData
	case startGameMsg:
		return gameview.NewTeamSelectModel(components.NewBreadcrumb(), &gameview.NetworkingInfo{Conn: m.conn, ConnId: stateCore.PEER, OpposingName: m.host.Name}), nil
	}

	return m, tea.Batch(cmds...)
}
