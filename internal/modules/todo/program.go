// Package todo is a demonstration module providing a simple todo list.
//
// It provides:
//   - A full-page GET that lists todos with checkboxes.
//   - HTMX POST endpoints (add / toggle / clear) that return HTML fragments,
//     updating the list in-place without a full page refresh.
//   - Persistence across page refreshes via an in-memory store.
package todo

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	rt "github.com/ryanfaerman/banana/internal/kernel/runtime"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the page-local state for the todo module.
type Model struct {
	Todos []Todo
}

// ---------------------------------------------------------------------------
// Msg (local sum type)
// ---------------------------------------------------------------------------

// Msg is the sealed interface for all todo page messages.
type Msg interface{ isTodoMsg() }

// PageOpened is sent on GET (via Init).
type PageOpened struct{}

func (PageOpened) isTodoMsg() {}

// ---------------------------------------------------------------------------
// Program implementation (GET only)
// ---------------------------------------------------------------------------

// Program implements rt.Program[Model, Msg] for the todo page.
type Program struct {
	Store *Store
}

// Init loads the current todo list from the store and sends PageOpened.
func (p Program) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[Msg]) {
	todos := p.Store.List()
	return Model{Todos: todos}, func(_ context.Context) []Msg {
		return []Msg{PageOpened{}}
	}
}

// Update handles messages.  Currently only PageOpened is used; all mutations
// are handled by dedicated HTMX endpoints.
func (p Program) Update(_ context.Context, m Model, msg Msg) (Model, rt.Cmd[Msg], rt.Outcome) {
	switch msg.(type) {
	case PageOpened:
		// nothing to do; model is already populated from Init
	}
	return m, nil, rt.Outcome{}
}

// View renders the full todo page.
func (p Program) View(_ context.Context, m Model) templ.Component {
	return Page(m.Todos)
}
