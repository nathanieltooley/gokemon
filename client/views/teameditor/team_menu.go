package teameditor

import (
	"maps"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
)

// Root Model that holds the Main Team Menu (create new, edit existing team)
// and the Team Selection submodels
type rootTeamModel struct {
	subModel tea.Model
}

type startTeamMenu struct {
	buttons components.MenuButtons
}

// Allows the user to select an already existing team for editing
type teamSelectionMenu struct {
	teamList list.Model
	teams    SavedTeams
}

type menuItem struct {
	string
}

func (m menuItem) FilterValue() string { return m.string }
func (m menuItem) Value() string       { return m.string }

type teamItem struct {
	Name    string
	Pokemon []game.Pokemon
}

func (t teamItem) FilterValue() string { return t.Name }
func (t teamItem) Value() string       { return t.Name }

func (m rootTeamModel) Init() tea.Cmd { return nil }
func (m rootTeamModel) View() string {
	return m.subModel.View()
}

func (m rootTeamModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.subModel.Update(msg)
}

func newTeamMainMenu() startTeamMenu {
	buttons := []components.ViewButton{
		{
			Name: "Create New Team",
			OnClick: func() tea.Model {
				return NewTeamEditorModel(global.POKEMON, global.MOVES, global.ABILITIES)
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() tea.Model {
				teams, err := LoadTeamMap()
				if err != nil {
					// TODO: Show error message
					return nil
				}
				items := make([]list.Item, 0)
				for team := range maps.Keys(teams) {
					items = append(items, teamItem{
						Name:    team,
						Pokemon: teams[team],
					})
				}

				return teamSelectionMenu{
					teams:    teams,
					teamList: list.New(items, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT),
				}
			},
		},
	}

	return startTeamMenu{
		buttons: components.NewMenuButton(buttons),
	}
}
func (m startTeamMenu) Init() tea.Cmd { return nil }
func (m startTeamMenu) View() string {
	return rendering.Center(lipgloss.JoinVertical(lipgloss.Center, m.buttons.View()))
}

func (m startTeamMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel := m.buttons.Update(msg)

	if newModel != nil {
		return newModel, nil
	}

	return m, nil
}

func (t teamSelectionMenu) Init() tea.Cmd { return nil }
func (t teamSelectionMenu) View() string {
	return t.teamList.View()
}

func (t teamSelectionMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newList, cmd := t.teamList.Update(msg)
	t.teamList = newList

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter {
			editor := NewTeamEditorModel(global.POKEMON, global.MOVES, global.ABILITIES)
			teamItem := t.teamList.SelectedItem().(teamItem)
			team := teamItem.Pokemon
			pokePointers := make([]*game.Pokemon, len(team))

			for i, poke := range team {
				pointer := &poke
				pokePointers[i] = pointer
			}

			return editor.AddStartingTeam(pokePointers), nil
		}
	}

	return t, cmd
}

func NewTeamMenu() rootTeamModel {
	return rootTeamModel{
		subModel: newTeamMainMenu(),
	}
}
