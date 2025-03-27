package mainmenu

import (
	"os"
	"path"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
)

type optionsMenuModel struct {
	backtrack components.Breadcrumbs

	focus     components.Focus
	showError bool
	err       error
}

type clearErrorMessage struct {
	t time.Time
}

type saveLocationInput struct {
	inner textinput.Model
}

func (s *saveLocationInput) OnFocus(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	opM := m.(optionsMenuModel)
	cmds := make([]tea.Cmd, 0)

	s.inner.Focus()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			saveLocation := s.inner.Value()
			if saveLocation != "" {
				// TODO: Validation!
				saveLocation := path.Clean(saveLocation)
				_, err := os.ReadDir(saveLocation)
				if err != nil {
					opM.showError = true
					opM.err = err

					cmds = append(cmds, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return clearErrorMessage{t}
					}))
				}

				global.TeamSaveLocation = s.inner.Value()
			}
		}
	}

	var uCmd tea.Cmd
	s.inner, uCmd = s.inner.Update(msg)
	cmds = append(cmds, uCmd)

	return opM, tea.Batch(cmds...)
}

func (p *saveLocationInput) Blur() {
	p.inner.Blur()
}

func (s saveLocationInput) View() string {
	return lipgloss.JoinVertical(lipgloss.Center, "Save Location", s.inner.View())
}

func (s saveLocationInput) FocusedView() string {
	return s.View()
}

type playerNameInput struct {
	inner textinput.Model
}

func (p *playerNameInput) OnFocus(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	fCmd := p.inner.Focus()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, global.SelectKey) {
			playerName := "Player"
			if p.inner.Value() != "" {
				playerName = p.inner.Value()
			}

			global.LocalPlayerName = playerName
		}
	}

	var uCmd tea.Cmd
	p.inner, uCmd = p.inner.Update(msg)

	return m, tea.Batch(fCmd, uCmd)
}

func (p *playerNameInput) Blur() {
	p.inner.Blur()
}

func (p *playerNameInput) View() string {
	return lipgloss.JoinVertical(lipgloss.Center, "Player Name", p.inner.View())
}
func (p *playerNameInput) FocusedView() string { return p.View() }

func newOptionsMenu(backtrack components.Breadcrumbs) optionsMenuModel {
	prompt := textinput.New()
	prompt.Focus()
	prompt.SetValue(global.TeamSaveLocation)

	namePrompt := textinput.New()
	namePrompt.SetValue(global.LocalPlayerName)

	return optionsMenuModel{
		backtrack: backtrack,
		focus:     components.NewFocus(&saveLocationInput{prompt}, &playerNameInput{namePrompt}),
	}
}

func (m optionsMenuModel) Init() tea.Cmd { return nil }
func (m optionsMenuModel) View() string {
	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, m.focus.Views()...))
}

func (m optionsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	switch msg := msg.(type) {
	case clearErrorMessage:
		m.showError = false
		m.err = nil
	case tea.KeyMsg:
		if m.showError {
			return m, nil
		}

		if key.Matches(msg, global.DownTabKey) {
			m.focus.Next()
		}

		if key.Matches(msg, global.UpTabKey) {
			m.focus.Prev()
		}

		if key.Matches(msg, global.BackKey) {
			return m.backtrack.PopDefault(func() tea.Model { return m }), nil
		}
	}

	newModel, focusCmd := m.focus.UpdateFocused(m, msg)
	m = newModel.(optionsMenuModel)
	cmds = append(cmds, focusCmd)

	return m, tea.Batch(cmds...)
}
