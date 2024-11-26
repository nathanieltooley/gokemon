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
	"github.com/nathanieltooley/gokemon/client/shared/teamfs"
)

type teamMenuCtx struct {
	// We have to do this indirection to avoid a dependency cycle
	// TODO: maybe consolidate all of these view packages into a single package?
	mainMenuFunction func() tea.Model
}

type startTeamMenu struct {
	ctx     *teamMenuCtx
	buttons components.MenuButtons
}

// Allows the user to select an already existing team for editing
type teamSelectionMenu struct {
	ctx *teamMenuCtx

	teamList list.Model
	teams    teamfs.SavedTeams
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

func newTeamMainMenu(ctx *teamMenuCtx) startTeamMenu {
	buttons := []components.ViewButton{
		{
			Name: "Create New Team",
			OnClick: func() tea.Model {
				return NewTeamEditorModel()
			},
		},
		{
			Name: "Edit Teams",
			OnClick: func() tea.Model {
				teams, err := teamfs.LoadTeamMap()
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
					ctx: ctx,

					teams:    teams,
					teamList: list.New(items, rendering.NewSimpleListDelegate(), global.TERM_WIDTH, global.TERM_HEIGHT),
				}
			},
		},
	}

	return startTeamMenu{
		ctx:     ctx,
		buttons: components.NewMenuButton(buttons),
	}
}
func (m startTeamMenu) Init() tea.Cmd { return nil }
func (m startTeamMenu) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, m.buttons.View()))
}

func (m startTeamMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel := m.buttons.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEsc {
			return m.ctx.mainMenuFunction(), nil
		}
	}

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
			editor := NewTeamEditorModel()
			teamItem := t.teamList.SelectedItem().(teamItem)
			team := teamItem.Pokemon

			editor.AddStartingTeam(team)
			return editor, nil
		}

		if msg.Type == tea.KeyEsc {
			return newTeamMainMenu(t.ctx), nil
		}
	}

	return t, cmd
}

func NewTeamMenu(backMenuFunction func() tea.Model) startTeamMenu {
	ctx := teamMenuCtx{
		mainMenuFunction: backMenuFunction,
	}

	return newTeamMainMenu(&ctx)
}
