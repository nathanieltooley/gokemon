package rendering

import (
	"fmt"
	"io"
	"math"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	if index == m.Index() {
		fmt.Fprint(w, d.HighlightedItemStyle.Render(listItem.FilterValue()))
	} else {
		fmt.Fprint(w, d.ItemStyle.Render(listItem.FilterValue()))
	}
}

func (d *simpleDelegate) SetSpacing(spacing int) {
	d.spacing = spacing
}

func NewSimpleListDelegate() simpleDelegate {
	return simpleDelegate{HighlightedItemStyle, ItemStyle, 0}
}
