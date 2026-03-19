// Package todo is a demonstration module providing a simple todo list.
//
// It provides:
//   - A full-page GET (PageProgram) that lists todos with checkboxes.
//   - Three HTMX-aware POST Programs (AddProgram, ToggleProgram, ClearProgram)
//     that process mutations through the Msg/Cmd/Update cycle, persist to the
//     store via Cmds, and render the updated list fragment.
//   - Persistence across page refreshes via an in-memory store.
package todo

import (
	"context"
	"fmt"
	"net/http"

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
// AddProgram – POST /todo
// ---------------------------------------------------------------------------

// AddMsg is the sealed interface for AddProgram messages.
type AddMsg interface{ isAddMsg() }

// AddRequested is decoded from the POST form; carries the new todo text.
type AddRequested struct{ Text string }

func (AddRequested) isAddMsg() {}

// AddCompleted carries the refreshed todo list after the store mutation.
type AddCompleted struct{ Todos []Todo }

func (AddCompleted) isAddMsg() {}

// AddProgram handles adding a new todo item.
type AddProgram struct{ Store *Store }

func (p AddProgram) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[AddMsg]) {
	return Model{}, nil
}

// DecodeMsg parses the "text" form field into an AddRequested message.
func (p AddProgram) DecodeMsg(r *http.Request) (AddMsg, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("todo add: parse form: %w", err)
	}
	return AddRequested{Text: r.FormValue("text")}, nil
}

// Update: AddRequested → Cmd (stores item, returns AddCompleted with fresh list).
//
//	AddCompleted → populate model.
func (p AddProgram) Update(_ context.Context, m Model, msg AddMsg) (Model, rt.Cmd[AddMsg], rt.Outcome) {
	switch msg := msg.(type) {
	case AddRequested:
		store := p.Store
		text := msg.Text
		return m, func(_ context.Context) []AddMsg {
			if text != "" {
				store.Add(text)
			}
			return []AddMsg{AddCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case AddCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the todo list fragment (used by HandlePost for HTMX responses).
func (p AddProgram) View(_ context.Context, m Model) templ.Component {
	return TodoList(m.Todos)
}

// ---------------------------------------------------------------------------
// ToggleProgram – POST /todo/{id}/toggle
// ---------------------------------------------------------------------------

// ToggleMsg is the sealed interface for ToggleProgram messages.
type ToggleMsg interface{ isToggleMsg() }

// ToggleRequested carries the ID of the todo to flip.
type ToggleRequested struct{ ID string }

func (ToggleRequested) isToggleMsg() {}

// ToggleCompleted carries the refreshed list after the toggle.
type ToggleCompleted struct{ Todos []Todo }

func (ToggleCompleted) isToggleMsg() {}

// ToggleProgram handles toggling the done state of a todo item.
type ToggleProgram struct{ Store *Store }

func (p ToggleProgram) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[ToggleMsg]) {
	return Model{}, nil
}

// DecodeMsg reads the "id" URL parameter set by chi routing.
func (p ToggleProgram) DecodeMsg(r *http.Request) (ToggleMsg, error) {
	return ToggleRequested{ID: chi.URLParam(r, "id")}, nil
}

// Update: ToggleRequested → Cmd (toggles item, returns ToggleCompleted with fresh list).
//
//	ToggleCompleted → populate model.
func (p ToggleProgram) Update(_ context.Context, m Model, msg ToggleMsg) (Model, rt.Cmd[ToggleMsg], rt.Outcome) {
	switch msg := msg.(type) {
	case ToggleRequested:
		store := p.Store
		id := msg.ID
		return m, func(_ context.Context) []ToggleMsg {
			store.Toggle(id)
			return []ToggleMsg{ToggleCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ToggleCompleted:
		m.Todos = msg.Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the todo list fragment.
func (p ToggleProgram) View(_ context.Context, m Model) templ.Component {
	return TodoList(m.Todos)
}

// ---------------------------------------------------------------------------
// ClearProgram – POST /todo/clear
// ---------------------------------------------------------------------------

// ClearMsg is the sealed interface for ClearProgram messages.
type ClearMsg interface{ isClearMsg() }

// ClearRequested triggers clearing all todos.
type ClearRequested struct{}

func (ClearRequested) isClearMsg() {}

// ClearCompleted carries the (now empty) refreshed list.
type ClearCompleted struct{ Todos []Todo }

func (ClearCompleted) isClearMsg() {}

// ClearProgram handles clearing all todo items.
type ClearProgram struct{ Store *Store }

func (p ClearProgram) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[ClearMsg]) {
	return Model{}, nil
}

// DecodeMsg always returns ClearRequested; there is no form body to parse.
func (p ClearProgram) DecodeMsg(_ *http.Request) (ClearMsg, error) {
	return ClearRequested{}, nil
}

// Update: ClearRequested → Cmd (clears store, returns ClearCompleted with fresh list).
//
//	ClearCompleted → populate model.
func (p ClearProgram) Update(_ context.Context, m Model, msg ClearMsg) (Model, rt.Cmd[ClearMsg], rt.Outcome) {
	switch msg.(type) {
	case ClearRequested:
		store := p.Store
		return m, func(_ context.Context) []ClearMsg {
			store.Clear()
			return []ClearMsg{ClearCompleted{Todos: store.List()}}
		}, rt.Outcome{}
	case ClearCompleted:
		m.Todos = msg.(ClearCompleted).Todos
		return m, nil, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the todo list fragment.
func (p ClearProgram) View(_ context.Context, m Model) templ.Component {
	return TodoList(m.Todos)
}

