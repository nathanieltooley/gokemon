package components

import tea "github.com/charmbracelet/bubbletea"

type Focus struct {
	Index int
	Items []Focusable
}

func NewFocus(items ...Focusable) Focus {
	return Focus{Items: items}
}

type Focusable interface {
	OnFocus(tea.Model, tea.Msg) (tea.Model, tea.Cmd)
	Blur()
	View() string
	FocusedView() string
}

func (f *Focus) Next() {
	f.Index++
	if f.Index >= len(f.Items) {
		f.Index = 0
	}
}

func (f *Focus) Prev() {
	f.Index--
	if f.Index < 0 {
		f.Index = len(f.Items) - 1
	}
}

func (f *Focus) UpdateFocused(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	for i, item := range f.Items {
		if i != f.Index {
			item.Blur()
		}
	}
	return f.Items[f.Index].OnFocus(m, msg)
}

func (f *Focus) Views() []string {
	views := make([]string, 0)
	for i, item := range f.Items {
		if i == f.Index {
			views = append(views, item.FocusedView())
		} else {
			views = append(views, item.View())
		}
	}

	return views
}
