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

func (b *Breadcrumbs) Push(model tea.Model) {
	b.PushNew(func() tea.Model {
		return model
	})
}

func (b *Breadcrumbs) PushNew(modelFunc func() tea.Model) {
	b.backtrace = append(b.backtrace, modelFunc)
}

// Returns a pointer for an optional nil value
func (b *Breadcrumbs) Pop() *tea.Model {
	l := len(b.backtrace)

	log.Debug().Msgf("Len: %d", l)

	if l == 0 {
		return nil
	}

	modelFunc := b.backtrace[l-1]
	b.backtrace = b.backtrace[0 : l-1]

	model := modelFunc()

	return &model
}

func (b *Breadcrumbs) PopDefault(def func() tea.Model) tea.Model {
	poppedModel := b.Pop()

	if poppedModel == nil {
		return def()
	}

	return *poppedModel
}
