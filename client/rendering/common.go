package rendering

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
)

var (
	HighlightedColor = lipgloss.Color("33")
	BlackTextColor   = lipgloss.Color("0")

	ButtonStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Width(30).Padding(1, 3).Align(lipgloss.Center)
	HighlightedButtonStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Width(30).Padding(1, 3).Align(lipgloss.Center).Foreground(HighlightedColor)

	HighlightedItemStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(HighlightedColor)
	ItemStyle            = lipgloss.NewStyle().PaddingLeft(4)
)

func Center(width int, height int, text string) string {
	return lipgloss.PlaceVertical(height, lipgloss.Center, lipgloss.PlaceHorizontal(width, lipgloss.Center, text))
}

func GlobalCenter(text string) string {
	return Center(global.TERM_WIDTH, global.TERM_HEIGHT, text)
}

func CenterBlock(block string, text string) string {
	w, h := lipgloss.Size(block)
	return Center(w, h, text)
}
