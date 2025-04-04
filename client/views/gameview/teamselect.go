package gameview

import (
	"maps"
	"net"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/networking"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/shared/teamfs"
	"github.com/nathanieltooley/gokemon/client/views/teameditor"
	"github.com/rs/zerolog/log"
)

type TeamSelectModel struct {
	teamList         list.Model
	teamView         components.TeamView
	buttons          components.MenuButtons
	focus            int
	backtrack        components.Breadcrumbs
	selectingStarter bool

	networkInfo      *NetworkingInfo
	waitingOnNetwork bool
	teamSent         bool
}

type teamItem struct {
	Name    string
	Pokemon []game.Pokemon
}

var switchFocusKey = key.NewBinding(
	key.WithKeys(tea.KeyTab.String(), tea.KeyShiftTab.String()),
)

func (t teamItem) FilterValue() string { return t.Name }
func (t teamItem) Value() string       { return t.Name }

type NetworkingInfo struct {
	Conn   net.Conn
	ConnId int
	// This value is only relevant for the host's TeamSelectModel
	ClientName string
}

func NewTeamSelectModel(backtrack components.Breadcrumbs, netInfo *NetworkingInfo) TeamSelectModel {
	button := components.ViewButton{
		Name: "New Team",
		OnClick: func() (tea.Model, tea.Cmd) {
			return teameditor.NewTeamEditorModel(backtrack.PushNew(func() tea.Model {
				return NewTeamSelectModel(backtrack, netInfo)
			}), make([]game.Pokemon, 0)), nil
		},
	}

	buttons := components.NewMenuButton([]components.ViewButton{button})

	teams, err := teamfs.LoadTeamMap(global.Opt.TeamSaveLocation)
	if err != nil {
		// TODO: Show error messages, and better handling (not crashing)
		log.Panic().Msgf("Could not load Teams: %s", err)
	}

	items := make([]list.Item, 0)
	for team := range maps.Keys(teams) {
		items = append(items, teamItem{
			Name:    team,
			Pokemon: teams[team],
		})
	}

	buttons.Unfocus()

	list := list.New(items, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT/2)
	list.DisableQuitKeybindings()

	startingTeam := make([]game.Pokemon, 0)

	if list.SelectedItem() != nil {
		startingTeam = list.SelectedItem().(teamItem).Pokemon
	}

	return TeamSelectModel{
		teamList:    list,
		buttons:     buttons,
		backtrack:   backtrack,
		teamView:    components.NewTeamView(startingTeam),
		networkInfo: netInfo,
	}
}

func (m TeamSelectModel) Init() tea.Cmd { return nil }
func (m TeamSelectModel) View() string {
	teamSelectionView := lipgloss.JoinVertical(lipgloss.Center, "Select a Team!", m.teamList.View(), m.buttons.View())
	view := lipgloss.JoinHorizontal(lipgloss.Center, teamSelectionView, m.teamView.View())
	if m.selectingStarter {
		view = lipgloss.JoinVertical(lipgloss.Center, "Select your starting Pokemon!", m.teamView.View())
	}
	return rendering.GlobalCenter(view)
}

func (m TeamSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.waitingOnNetwork {
			return m, nil
		}

		if key.Matches(msg, switchFocusKey) {
			m.focus++
			if m.focus > 1 {
				m.focus = 0
			}
		}

		if m.selectingStarter && key.Matches(msg, global.SelectKey) && !m.teamSent {
			selectedTeam := m.teamList.SelectedItem().(teamItem)

			// TODO: Check for HOST/PEER
			if m.networkInfo != nil {
				m.teamSent = true
				switch m.networkInfo.ConnId {
				case state.HOST:
					// Wait for opponent to send team over
					return m, func() tea.Msg {
						peerTeam, err := networking.AcceptData[networking.TeamSelectionPacket](m.networkInfo.Conn)
						if err != nil {
							// TODO: Change all log.Fatal() to something that doesn't crash
							log.Fatal().Err(err).Msg("Error trying to get team from opponent")
						}

						return peerTeam
					}
				case state.PEER:
					return m, func() tea.Msg {
						// Send team to host
						log.Debug().Msgf("Sent team: %+v", selectedTeam.Pokemon)

						err := networking.SendData(m.networkInfo.Conn, networking.TeamSelectionPacket{Team: selectedTeam.Pokemon, StartingIndex: m.teamView.CurrentPokemonIndex})
						if err != nil {
							log.Fatal().Err(err).Msg("Error trying to send team to host")
						}

						// Wait for starting state
						state, err := networking.AcceptData[state.GameState](m.networkInfo.Conn)
						if err != nil {
							log.Fatal().Err(err).Msg("Error trying to get gamestate from host")
						}

						return state
					}
				}
			} else {
				defaultEnemyTeam := state.RandomTeam()

				gameState := state.NewState(selectedTeam.Pokemon, defaultEnemyTeam)
				// update the starter to what the player selects in this screen
				gameState.LocalPlayer.ActivePokeIndex = m.teamView.CurrentPokemonIndex

				return NewMainGameModel(gameState, state.HOST, nil), nil
			}
		}

		if m.focus == 0 && key.Matches(msg, global.SelectKey) && !m.selectingStarter && m.teamList.SelectedItem() != nil {
			m.selectingStarter = true
			m.teamView.Focused = true
		}

		// TODO: Allow backtracking but close connection for multiplayer games
		if msg.Type == tea.KeyEsc && m.networkInfo == nil {
			if m.selectingStarter {
				m.selectingStarter = false
			} else {
				return m.backtrack.PopDefault(func() tea.Model { return m }), nil
			}
		}
	case state.GameState: // PEER CASE
		return NewMainGameModel(msg, state.PEER, m.networkInfo.Conn), nil

	case networking.TeamSelectionPacket: // HOST CASE
		selectedTeam := m.teamList.SelectedItem().(teamItem)

		gameState := state.NewState(selectedTeam.Pokemon, msg.Team)

		gameState.LocalPlayer.ActivePokeIndex = m.teamView.CurrentPokemonIndex
		gameState.LocalPlayer.Name = global.Opt.LocalPlayerName

		gameState.OpposingPlayer.ActivePokeIndex = msg.StartingIndex
		gameState.OpposingPlayer.Name = m.networkInfo.ClientName

		// TODO: Make this a cmd so that it doesn't block
		networking.SendData(m.networkInfo.Conn, gameState)

		return NewMainGameModel(gameState, state.HOST, m.networkInfo.Conn), nil
	}

	// Update team selection view
	if !m.selectingStarter {
		switch m.focus {
		case 0:
			m.buttons.Unfocus()

			var cmd tea.Cmd
			m.teamList, cmd = m.teamList.Update(msg)

			if m.teamList.SelectedItem() != nil {
				m.teamView.Team = m.teamList.SelectedItem().(teamItem).Pokemon
			} else {
				m.teamView.Team = make([]game.Pokemon, 0)
			}

			return m, cmd
		case 1:
			m.buttons.Focus()

			newModel, startCmd := m.buttons.Update(msg)
			if newModel != nil {
				return newModel, startCmd
			}
		}
	}

	if m.selectingStarter {
		newTeamView, _ := m.teamView.Update(msg)
		m.teamView = newTeamView.(components.TeamView)
	}

	return m, nil
}
