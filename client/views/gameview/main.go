package gameview

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/game/state"
	"github.com/nathanieltooley/gokemon/client/rendering"
)

var panelStyle = lipgloss.NewStyle().Border(lipgloss.BlockBorder(), true).Padding(2)

type MainGameModel struct {
	state state.GameState
}

func NewMainGameModel(state state.GameState) MainGameModel {
	return MainGameModel{
		state: state,
	}
}

func (m MainGameModel) Init() tea.Cmd { return nil }
func (m MainGameModel) View() string {
	return rendering.Center(
		lipgloss.JoinHorizontal(
			lipgloss.Center,
			playerPanel("HOST", m.state.GetPlayer(state.HOST)),
			playerPanel("PEER", m.state.GetPlayer(state.PEER)),
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

	return m, nil
}

func playerPanel(name string, player *state.Player) string {
	currentPokemon := player.Team[player.ActivePokeIndex]
	pokeInfo := fmt.Sprintf(`
		%s
		%d / %d
		Level: %d
	`,
		currentPokemon.Nickname,
		currentPokemon.Hp.Value,
		currentPokemon.MaxHp,
		currentPokemon.Level,
	)

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Center, name, pokeInfo))
}
