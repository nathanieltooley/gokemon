package gameview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

// BIG PROBLEM IM GONNA HAVE TO FIGURE OUT
//
// bubbletea doesn't have a framework for creating "panels" or "windows" at arbitrary locations.
// I'm gonna have to find a way to atleast put these panels in decent locations.
// Maybe a module that creates these panels at a location and makes a string of it?
// I may not deal with it ever and just make two panels with pokemon info in the center
// and a panel with pokemon actions at the bottom ¯\_(ツ)_/¯

var (
	panelStyle            = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).Padding(1, 2).AlignHorizontal(lipgloss.Center)
	highlightedPanelStyle = panelStyle.Background(rendering.HighlightedColor).Foreground(lipgloss.Color("255"))
)

type MainGameModel struct {
	state *state.GameState

	panel tea.Model
}

func NewMainGameModel(state state.GameState) MainGameModel {
	return MainGameModel{
		state: &state,
		panel: actionPanel{
			state: &state,
		},
	}
}

func (m MainGameModel) Init() tea.Cmd { return nil }
func (m MainGameModel) View() string {
	return rendering.GlobalCenter(
		lipgloss.JoinVertical(
			lipgloss.Center,
			lipgloss.JoinHorizontal(
				lipgloss.Center,
				newPlayerPanel(m.state, "HOST", m.state.GetPlayer(state.HOST)).View(),
				newPlayerPanel(m.state, "PEER", m.state.GetPlayer(state.PEER)).View(),
			),

			m.panel.View(),
		),
	)
}

func (m MainGameModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlA {
			m.state.Update(state.SwitchAction{
				PlayerIndex: state.HOST,
				SwitchIndex: 1,
			})
		}
	}

	m.panel, _ = m.panel.Update(msg)

	return m, nil
}
