package gameview

import (
	"maps"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/global"
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
	backtrack        *components.Breadcrumbs
	selectingStarter bool
}

type teamItem struct {
	Name    string
	Pokemon []game.Pokemon
}

var (
	switchFocusKey = key.NewBinding(
		key.WithKeys(tea.KeyTab.String(), tea.KeyShiftTab.String()),
	)
	selectKey = key.NewBinding(
		key.WithKeys(tea.KeyEnter.String()),
	)
)

func (t teamItem) FilterValue() string { return t.Name }
func (t teamItem) Value() string       { return t.Name }

func NewTeamSelectModel(backtrack *components.Breadcrumbs) TeamSelectModel {
	button := components.ViewButton{
		Name: "New Team",
		OnClick: func() tea.Model {
			backtrack.PushNew(func() tea.Model {
				return NewTeamSelectModel(backtrack)
			})
			return teameditor.NewTeamEditorModel(backtrack, make([]game.Pokemon, 0))
		},
	}

	buttons := components.NewMenuButton([]components.ViewButton{button})

	teams, err := teamfs.LoadTeamMap()
	// TODO: Error handling
	if err != nil {
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
	return TeamSelectModel{
		teamList:  list,
		buttons:   buttons,
		backtrack: backtrack,
		teamView:  components.NewTeamView(list.SelectedItem().(teamItem).Pokemon),
	}
}

func (m TeamSelectModel) Init() tea.Cmd { return nil }
func (m TeamSelectModel) View() string {
	teamSelectionView := lipgloss.JoinVertical(lipgloss.Center, m.teamList.View(), m.buttons.View())
	view := lipgloss.JoinHorizontal(lipgloss.Center, teamSelectionView, m.teamView.View())
	if m.selectingStarter {
		view = m.teamView.View()
	}
	return rendering.GlobalCenter(view)
}

func (m TeamSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, switchFocusKey) {
			m.focus++
			if m.focus > 1 {
				m.focus = 0
			}
		}

		if m.selectingStarter && key.Matches(msg, selectKey) {
			selectedTeam := m.teamList.SelectedItem().(teamItem)
			defaultEnemyTeam := state.RandomTeam()

			gameState := state.NewState(selectedTeam.Pokemon, defaultEnemyTeam)
			// update the starter to what the player selects in this screen
			gameState.LocalPlayer.ActivePokeIndex = m.teamView.CurrentPokemonIndex

			return NewMainGameModel(gameState, state.HOST), nil
		}

		if m.focus == 0 && key.Matches(msg, selectKey) && !m.selectingStarter {
			m.selectingStarter = true
			m.teamView.Focused = true
		}

		if msg.Type == tea.KeyEsc {
			if m.selectingStarter {
				m.selectingStarter = false
			} else {
				return m.backtrack.PopDefault(func() tea.Model { return m }), nil
			}
		}
	}

	if !m.selectingStarter {
		switch m.focus {
		case 0:
			m.buttons.Unfocus()

			var cmd tea.Cmd
			m.teamList, cmd = m.teamList.Update(msg)

			return m, cmd
		case 1:
			m.buttons.Focus()

			newModel := m.buttons.Update(msg)
			if newModel != nil {
				return newModel, nil
			}
		}
	}

	m.teamView.Team = m.teamList.SelectedItem().(teamItem).Pokemon

	log.Debug().Msg("help me!!!")
	if m.selectingStarter {
		newTeamView, _ := m.teamView.Update(msg)
		m.teamView = newTeamView.(components.TeamView)
	}

	return m, nil
}
