package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

type Breadcrumbs struct {
	backtrace []func() tea.Model
}

func NewBreadcrumb() Breadcrumbs {
	return Breadcrumbs{}
}

// Push a model onto the breadcrumb stack.
// Returns the modified copy.
func (b Breadcrumbs) Push(model tea.Model) Breadcrumbs {
	l := len(b.backtrace)
	log.Debug().Msgf("Push, Len: %d", l)

	return b.PushNew(func() tea.Model {
		return model
	})
}

// Push a function that creates a new model onto the stack.
// Returns the modified copy.
func (b Breadcrumbs) PushNew(modelFunc func() tea.Model) Breadcrumbs {
	b.backtrace = append(b.backtrace, modelFunc)

	return b
}

// Returns a pointer for an optional nil value
// Does not return the modified version since the primary use case does not use it,
// it uses an older copy of the struct from a previous push
func (b Breadcrumbs) Pop() *tea.Model {
	l := len(b.backtrace)

	if l == 0 {
		return nil
	}

	modelFunc := b.backtrace[l-1]
	b.backtrace = b.backtrace[0 : l-1]

	model := modelFunc()

	log.Debug().Msgf("Pop, Len: %d", len(b.backtrace))
	return &model
}

func (b Breadcrumbs) PopDefault(def func() tea.Model) tea.Model {
	poppedModel := b.Pop()

	if poppedModel == nil {
		return def()
	}

	return *poppedModel
}
