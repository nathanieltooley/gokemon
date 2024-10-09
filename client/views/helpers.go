package views

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
)

func Center(text string) string {
	return lipgloss.PlaceVertical(global.TERM_HEIGHT, lipgloss.Center, lipgloss.PlaceHorizontal(global.TERM_WIDTH, lipgloss.Center, text))
}
