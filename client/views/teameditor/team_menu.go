package teameditor

import (
	"errors"
	"io/fs"
	"maps"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
	"github.com/nathanieltooley/gokemon/client/shared/teamfs"
	"github.com/rs/zerolog/log"
)

type startTeamMenu struct {
	backtrace components.Breadcrumbs
	buttons   components.MenuButtons
}

// Allows the user to select an already existing team for editing
type teamSelectionMenu struct {
	backtrace components.Breadcrumbs

	teamList list.Model
	teams    teamfs.SavedTeams
}

type teamItem struct {
	Name    string
	Pokemon []game.Pokemon
}

func (t teamItem) FilterValue() string { return t.Name }
func (t teamItem) Value() string       { return t.Name }

func newTeamMainMenu(backtrace components.Breadcrumbs) startTeamMenu {
	startMenu := startTeamMenu{}
	buttons := []components.ViewButton{
		{
			Name: "Create New Team",
			OnClick: func() (tea.Model, tea.Cmd) {
				backtrace.Push(startMenu)
				return NewTeamEditorModel(backtrace, make([]game.Pokemon, 0)), nil
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() (tea.Model, tea.Cmd) {
				return newTeamSelectionMenu(backtrace), nil
			},
		},
	}

	startMenu.backtrace = backtrace
	startMenu.buttons = components.NewMenuButton(buttons)

	return startMenu
}

func newTeamSelectionMenu(backtrace components.Breadcrumbs) teamSelectionMenu {
	teams, err := teamfs.LoadTeamMap(global.TeamSaveLocation)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if err := teamfs.NewTeamSave(global.TeamSaveLocation); err != nil {
				log.Panic().Msgf("Could not create team saves file: %s", err)
			}
		} else {
			// TODO: Show error message
			log.Panic().Msgf("Could not load team saves file: %s", err)
		}
	}
	items := make([]list.Item, 0)
	for team := range maps.Keys(teams) {
		items = append(items, teamItem{
			Name:    team,
			Pokemon: teams[team],
		})
	}

	teamList := list.New(items, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT)
	teamList.DisableQuitKeybindings()

	return teamSelectionMenu{
		backtrace: backtrace,

		teams:    teams,
		teamList: teamList,
	}
}

func (m startTeamMenu) Init() tea.Cmd { return nil }
func (m startTeamMenu) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, m.buttons.View()))
}

func (m startTeamMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, startCmd := m.buttons.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			return m.backtrace.PopDefault(func() tea.Model { return m }), nil
		}
	}

	if newModel != nil {
		return newModel, startCmd
	}

	return m, nil
}

func (t teamSelectionMenu) Init() tea.Cmd { return nil }
func (t teamSelectionMenu) View() string {
	return rendering.GlobalCenter(t.teamList.View())
}

func (t teamSelectionMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := t.teamList.Update(msg)
	t.teamList = newList

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			teamItem := t.teamList.SelectedItem().(teamItem)
			team := slices.Clone(teamItem.Pokemon)

			t.backtrace.PushNew(func() tea.Model {
				return newTeamSelectionMenu(t.backtrace)
			})

			editor := NewTeamEditorModel(t.backtrace, team)

			return editor, nil
		}

		if msg.Type == tea.KeyEsc {
			return newTeamMainMenu(t.backtrace), nil
		}
	}

	return t, cmd
}

func NewTeamMenu(mainMenuFunction func() tea.Model) startTeamMenu {
	backtrack := components.NewBreadcrumb()

	return newTeamMainMenu(backtrack.PushNew(mainMenuFunction))
}
