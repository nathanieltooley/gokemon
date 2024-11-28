package gameview

import (
	"log"
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
)

type TeamSelectModel struct {
	teamList list.Model
	buttons  components.MenuButtons
	focus    int
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

func NewTeamSelectModel() TeamSelectModel {
	button := components.ViewButton{
		Name: "New Team",
		OnClick: func() tea.Model {
			backtrack := components.NewBreadcrumb()

			backtrack.PushNew(func() tea.Model {
				return NewTeamSelectModel()
			})
			return teameditor.NewTeamEditorModel(&backtrack)
		},
	}

	buttons := components.NewMenuButton([]components.ViewButton{button})

	teams, err := teamfs.LoadTeamMap()
	// TODO: Error handling
	if err != nil {
		log.Panicln("Could not load Teams: ", err)
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
	return TeamSelectModel{
		teamList: list,
		buttons:  buttons,
	}
}

func (m TeamSelectModel) Init() tea.Cmd { return nil }
func (m TeamSelectModel) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, m.teamList.View(), m.buttons.View()))
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

		if m.focus == 0 && key.Matches(msg, selectKey) {
			selectedTeam := m.teamList.SelectedItem().(teamItem)
			defaultEnemyTeam := state.RandomTeam()

			return NewMainGameModel(state.NewState(selectedTeam.Pokemon, defaultEnemyTeam), state.HOST), nil
		}
	}

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

	return m, nil
}
