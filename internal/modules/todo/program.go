// Package todo is a demonstration module providing a simple todo list.
//
// It provides:
//   - A full-page GET (PageProgram) that lists todos with checkboxes.
//   - A single HTMX-aware POST Program that handles add, toggle, and clear
//     mutations through the Msg/Cmd/Update cycle, persists to the store via
//     Cmds, and renders the updated list fragment.
//   - Persistence across page refreshes via an in-memory store.
package todo

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	rt "github.com/ryanfaerman/banana/internal/kernel/runtime"
)

// ---------------------------------------------------------------------------
// Shared model
// ---------------------------------------------------------------------------

// Model is the page-local state for the todo module.
type Model struct {
	Todos []Todo
}

// ---------------------------------------------------------------------------
// PageProgram – GET /todo
// ---------------------------------------------------------------------------

// PageMsg is the sealed interface for PageProgram messages.
type PageMsg interface{ isPageMsg() }

// PageOpened is sent by Init; triggers loading todos from the store.
type PageOpened struct{}

func (PageOpened) isPageMsg() {}

// TodosLoaded carries the todos retrieved from the store.
type TodosLoaded struct{ Todos []Todo }

func (TodosLoaded) isPageMsg() {}

// PageProgram renders the full todo page.
type PageProgram struct{ Store *Store }

// Init starts with an empty model and immediately sends PageOpened.
func (p PageProgram) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[PageMsg]) {
	return Model{}, func(_ context.Context) []PageMsg {
		return []PageMsg{PageOpened{}}
	}
}

// Update loads todos on PageOpened, then populates the model on TodosLoaded.
func (p PageProgram) Update(_ context.Context, m Model, msg PageMsg) (Model, rt.Cmd[PageMsg], rt.Outcome) {
	switch msg.(type) {
	case PageOpened:
		store := p.Store
		return m, func(_ context.Context) []PageMsg {
			return []PageMsg{TodosLoaded{Todos: store.List()}}
		}, rt.Outcome{}
	case TodosLoaded:
		m.Todos = msg.(TodosLoaded).Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the full todo page inside the application shell.
func (p PageProgram) View(_ context.Context, m Model) templ.Component {
	return Page(m.Todos)
}

// ---------------------------------------------------------------------------
// Program – POST /todo, POST /todo/clear, POST /todo/{id}/toggle
// ---------------------------------------------------------------------------

// PostMsg is the sealed interface for Program messages.
type PostMsg interface{ isPostMsg() }

// AddRequested is decoded from POST /todo; carries the new todo text.
type AddRequested struct{ Text string }

func (AddRequested) isPostMsg() {}

// AddCompleted carries the refreshed todo list after adding.
type AddCompleted struct{ Todos []Todo }

func (AddCompleted) isPostMsg() {}

// ToggleRequested carries the ID of the todo to flip, decoded from POST /todo/{id}/toggle.
type ToggleRequested struct{ ID string }

func (ToggleRequested) isPostMsg() {}

// ToggleCompleted carries the refreshed list after the toggle.
type ToggleCompleted struct{ Todos []Todo }

func (ToggleCompleted) isPostMsg() {}

// ClearRequested triggers clearing all todos, decoded from POST /todo/clear.
type ClearRequested struct{}

func (ClearRequested) isPostMsg() {}

// ClearCompleted carries the (now empty) refreshed list.
type ClearCompleted struct{ Todos []Todo }

func (ClearCompleted) isPostMsg() {}

// Program handles all todo POST mutations: add, toggle, and clear.
type Program struct{ Store *Store }

func (p Program) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[PostMsg]) {
	return Model{}, nil
}

// DecodeMsg inspects the request to determine which action to perform:
//   - POST /todo/{id}/toggle → ToggleRequested (chi "id" URL param is set)
//   - POST /todo/clear       → ClearRequested  (path ends with "/clear")
//   - POST /todo             → AddRequested
func (p Program) DecodeMsg(r *http.Request) (PostMsg, error) {
	if id := chi.URLParam(r, "id"); id != "" {
		return ToggleRequested{ID: id}, nil
	}
	if strings.HasSuffix(r.URL.Path, "/clear") {
		return ClearRequested{}, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("todo: parse form: %w", err)
	}
	return AddRequested{Text: r.FormValue("text")}, nil
}

// Update handles all todo mutations.
func (p Program) Update(_ context.Context, m Model, msg PostMsg) (Model, rt.Cmd[PostMsg], rt.Outcome) {
	switch msg := msg.(type) {
	case AddRequested:
		store := p.Store
		text := msg.Text
		return m, func(_ context.Context) []PostMsg {
			if text != "" {
				store.Add(text)
			}
			return []PostMsg{AddCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case AddCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	case ToggleRequested:
		store := p.Store
		id := msg.ID
		return m, func(_ context.Context) []PostMsg {
			store.Toggle(id)
			return []PostMsg{ToggleCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ToggleCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	case ClearRequested:
		store := p.Store
		return m, func(_ context.Context) []PostMsg {
			store.Clear()
			return []PostMsg{ClearCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ClearCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the todo list fragment (used by HandlePost for HTMX responses).
func (p Program) View(_ context.Context, m Model) templ.Component {
	return TodoList(m.Todos)
}

