package rendering

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
)

var (
	HighlightedColor = lipgloss.Color("45")

	ButtonStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1, 3).Align(lipgloss.Center)
	HighlightedButtonStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Padding(1, 3).Align(lipgloss.Center).Foreground(HighlightedColor)

	HighlightedItemStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(HighlightedColor)
	ItemStyle            = lipgloss.NewStyle().PaddingLeft(4)
)

func Center(text string) string {
	return lipgloss.PlaceVertical(global.TERM_HEIGHT, lipgloss.Center, lipgloss.PlaceHorizontal(global.TERM_WIDTH, lipgloss.Center, text))
}
