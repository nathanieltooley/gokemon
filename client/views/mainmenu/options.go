package mainmenu

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
	"github.com/nathanieltooley/gokemon/client/rendering"
	"github.com/nathanieltooley/gokemon/client/rendering/components"
)

type optionsMenuModel struct {
	backtrack components.Breadcrumbs

	saveLocationPrompt textinput.Model
	showError          bool
	err                error
}

type clearErrorMessage struct {
	t time.Time
}

func newOptionsMenu(backtrack components.Breadcrumbs) optionsMenuModel {
	prompt := textinput.New()
	prompt.Focus()
	prompt.SetValue(global.TeamSaveLocation)

	return optionsMenuModel{
		backtrack:          backtrack,
		saveLocationPrompt: prompt,
	}
}

func (m optionsMenuModel) Init() tea.Cmd { return nil }
func (m optionsMenuModel) View() string {
	if m.showError {
		return rendering.GlobalCenter(fmt.Sprintf("An Error has occured!: %s", m.err))
	}

	return rendering.GlobalCenter(lipgloss.JoinVertical(lipgloss.Center, "Team Save Location", m.saveLocationPrompt.View()))
}

func (m optionsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case clearErrorMessage:
		m.showError = false
		m.err = nil
	case tea.KeyMsg:
		if m.showError {
			return m, cmd
		}

		switch msg.Type {
		case tea.KeyEsc:
			return m.backtrack.PopDefault(func() tea.Model { return m }), nil
		case tea.KeyEnter:
			saveLocation := m.saveLocationPrompt.Value()
			if saveLocation != "" {
				// TODO: Validation!
				saveLocation := path.Clean(saveLocation)
				_, err := os.ReadDir(saveLocation)
				if err != nil {
					m.showError = true
					m.err = err

					cmd = tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
						return clearErrorMessage{t}
					})

					return m, cmd
				}

				global.TeamSaveLocation = m.saveLocationPrompt.Value()
			}
		}
	}

	m.saveLocationPrompt, cmd = m.saveLocationPrompt.Update(msg)

	return m, cmd
}
