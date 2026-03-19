// Package todo is a demonstration module providing a simple todo list.
//
// It provides:
//   - A single Program that handles both GET (full page) and POST (add,
//     toggle, clear) operations through the Msg/Cmd/Update cycle.
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
// Msg (unified sum type)
// ---------------------------------------------------------------------------

// Msg is the sealed interface for all todo messages.
type Msg interface{ isTodoMsg() }

// PageOpened is sent by Init; triggers loading todos from the store.
type PageOpened struct{}

func (PageOpened) isTodoMsg() {}

// TodosLoaded carries the todos retrieved from the store.
type TodosLoaded struct{ Todos []Todo }

func (TodosLoaded) isTodoMsg() {}

// AddRequested carries the new todo text from POST /todo.
type AddRequested struct{ Text string }

func (AddRequested) isTodoMsg() {}

// AddCompleted carries the refreshed todo list after adding.
type AddCompleted struct{ Todos []Todo }

func (AddCompleted) isTodoMsg() {}

// ToggleRequested carries the ID of the todo to flip from POST /todo/{id}/toggle.
type ToggleRequested struct{ ID string }

func (ToggleRequested) isTodoMsg() {}

// ToggleCompleted carries the refreshed list after the toggle.
type ToggleCompleted struct{ Todos []Todo }

func (ToggleCompleted) isTodoMsg() {}

// ClearRequested triggers clearing all todos from POST /todo/clear.
type ClearRequested struct{}

func (ClearRequested) isTodoMsg() {}

// ClearCompleted carries the (now empty) refreshed list.
type ClearCompleted struct{ Todos []Todo }

func (ClearCompleted) isTodoMsg() {}

// ---------------------------------------------------------------------------
// Program – GET /todo, POST /todo, POST /todo/clear, POST /todo/{id}/toggle
// ---------------------------------------------------------------------------

// Program handles all todo operations: page load, add, toggle, and clear.
type Program struct{ Store *Store }

// Init starts with an empty model and immediately sends PageOpened to trigger
// loading todos from the store.
func (p Program) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[Msg]) {
	return Model{}, func(_ context.Context) []Msg {
		return []Msg{PageOpened{}}
	}
}

// Update handles all todo messages.
func (p Program) Update(_ context.Context, m Model, msg Msg) (Model, rt.Cmd[Msg], rt.Outcome) {
	switch msg := msg.(type) {
	case PageOpened:
		store := p.Store
		return m, func(_ context.Context) []Msg {
			return []Msg{TodosLoaded{Todos: store.List()}}
		}, rt.Outcome{}
	case TodosLoaded:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	case AddRequested:
		store := p.Store
		text := msg.Text
		return m, func(_ context.Context) []Msg {
			if text != "" {
				store.Add(text)
			}
			return []Msg{AddCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case AddCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	case ToggleRequested:
		store := p.Store
		id := msg.ID
		return m, func(_ context.Context) []Msg {
			store.Toggle(id)
			return []Msg{ToggleCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ToggleCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	case ClearRequested:
		store := p.Store
		return m, func(_ context.Context) []Msg {
			store.Clear()
			return []Msg{ClearCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ClearCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the full todo page.
// For GET requests this is wrapped in the application shell by HandleGet.
// For HTMX POST requests HandlePost returns this directly; the templates use
// hx-select="#todo-list" so HTMX extracts just the list fragment from the
// full-page response.
func (p Program) View(_ context.Context, m Model) templ.Component {
	return Page(m.Todos)
}

