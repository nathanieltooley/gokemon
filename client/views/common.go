package views

import (
	"fmt"
	"io"
	"math"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathanieltooley/gokemon/client/global"
)

var (
	HighlightedColor = lipgloss.Color("45")

	ButtonStyle            = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1, 3)
	HighlightedButtonStyle = lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true).Padding(1, 3).Foreground(HighlightedColor)

	HighlightedItemStyle = lipgloss.NewStyle().PaddingLeft(4).Foreground(HighlightedColor)
	ItemStyle            = lipgloss.NewStyle().PaddingLeft(4)
)

func Center(text string) string {
	return lipgloss.PlaceVertical(global.TERM_HEIGHT, lipgloss.Center, lipgloss.PlaceHorizontal(global.TERM_WIDTH, lipgloss.Center, text))
}

type SimpleItem interface {
	Value() string
}
type simpleDelegate struct {
	HighlightedItemStyle lipgloss.Style
	ItemStyle            lipgloss.Style

	spacing int
}

func (d simpleDelegate) Height() int {
	// Get the smaller style's height
	height := math.Min(float64(d.ItemStyle.GetHeight()), float64(d.HighlightedItemStyle.GetHeight()))
	// Make sure the height is atleast 1
	intHeight := int(math.Max(1, height))
	return intHeight
}
func (d simpleDelegate) Spacing() int                            { return d.spacing }
func (d simpleDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d simpleDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(SimpleItem)
	if !ok {
		fmt.Fprint(w, "Invalid Item!")
		return
	}

	if index == m.Index() {
		fmt.Fprint(w, d.HighlightedItemStyle.Render(i.Value()))
	} else {
		fmt.Fprint(w, d.ItemStyle.Render(i.Value()))
	}
}

func (d *simpleDelegate) SetSpacing(spacing int) {
	d.spacing = spacing
}

func NewSimpleListDelegate() simpleDelegate {
	return simpleDelegate{HighlightedItemStyle, ItemStyle, 0}
}
