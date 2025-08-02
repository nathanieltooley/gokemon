package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/golurk"
)

type TeamView struct {
	Team    []golurk.Pokemon
	Focused bool

	CurrentPokemonIndex int
}

var (
	pokemonTeamStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Align(lipgloss.Center).Width(20)
	highlightedPokemonTeamStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Align(lipgloss.Center).Width(20)

	moveTeamDown = key.NewBinding(
		key.WithKeys("j", "down"),
	)

	moveTeamUp = key.NewBinding(
		key.WithKeys("k", "up"),
	)
)

func NewTeamView(team []golurk.Pokemon) TeamView {
	return TeamView{
		Team:    team,
		Focused: false,
	}
}

func (m TeamView) Init() tea.Cmd { return nil }
func (m TeamView) View() string {
	teamPanels := make([]string, 0)

	for i, pokemon := range m.Team {
		panel := fmt.Sprintf("%s\nLevel: %d\n", pokemon.Nickname, pokemon.Level)

		if i == m.CurrentPokemonIndex && m.Focused {
			teamPanels = append(teamPanels, highlightedPokemonTeamStyle.Render(panel))
		} else {
			teamPanels = append(teamPanels, pokemonTeamStyle.Render(panel))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Center, teamPanels...)
}

func (m TeamView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.Focused {
			if key.Matches(msg, moveTeamDown) {
				m.CurrentPokemonIndex++

				if m.CurrentPokemonIndex > len(m.Team)-1 {
					m.CurrentPokemonIndex = 0
				}
			}

			if key.Matches(msg, moveTeamUp) {
				m.CurrentPokemonIndex--

				if m.CurrentPokemonIndex < 0 {
					m.CurrentPokemonIndex = len(m.Team) - 1
				}
			}
		}
	}

	return m, nil
}
