// Package example is a demonstration module showing the Tea-style kernel.
//
// It provides:
//   - A GET page program that renders within the kernel shell.
//   - A POST action that produces a Cmd → []Msg → Update cycle, sets a flash
//     message stub, and redirects back to the page.
//   - Menu registration.
//   - Route access marking (private).
package example

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"

	rt "github.com/ryanfaerman/banana/internal/kernel/runtime"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the page-local state for the example module.
type Model struct {
	Counter int
	LastMsg string
}

// ---------------------------------------------------------------------------
// Msg  (local sum type)
// ---------------------------------------------------------------------------

// Msg is the sealed interface for all example page messages.
type Msg interface{ isExampleMsg() }

// PageOpened is sent when the page loads (via Init).
type PageOpened struct{}

func (PageOpened) isExampleMsg() {}

// ActionSubmitted is the POST form message.
type ActionSubmitted struct{ Message string }

func (ActionSubmitted) isExampleMsg() {}

// CounterIncremented is produced by the Cmd after ActionSubmitted.
type CounterIncremented struct{}

func (CounterIncremented) isExampleMsg() {}

// ---------------------------------------------------------------------------
// Program implementation
// ---------------------------------------------------------------------------

// Program implements rt.Program[Model, Msg] for the example page.
type Program struct {
	Store *Store
}

// Init loads persisted state from the store and sends PageOpened.
func (p Program) Init(_ context.Context, _ *http.Request) (Model, rt.Cmd[Msg]) {
	model := p.Store.Load()
	return model, func(_ context.Context) []Msg {
		return []Msg{PageOpened{}}
	}
}

// Update handles messages and returns the updated model, optional next Cmd,
// and any Outcome (redirect, flashes).
func (p Program) Update(_ context.Context, m Model, msg Msg) (Model, rt.Cmd[Msg], rt.Outcome) {
	switch msg := msg.(type) {
	case PageOpened:
		// nothing to do on initial open
		return m, nil, rt.Outcome{}

	case ActionSubmitted:
		m.LastMsg = msg.Message
		// Return a Cmd that produces CounterIncremented.
		cmd := func(_ context.Context) []Msg {
			return []Msg{CounterIncremented{}}
		}
		outcome := rt.Outcome{
			// Redirect to the page itself (could be left empty to use Referer).
			RedirectTo: "/example",
			Flashes: []rt.Flash{
				{Level: "success", Message: fmt.Sprintf("Submitted: %q", msg.Message)},
			},
		}
		return m, cmd, outcome

	case CounterIncremented:
		m.Counter++
		// Persist updated model to the store as a side effect (Cmd).
		store := p.Store
		return m, func(_ context.Context) []Msg {
			store.Save(m)
			return nil
		}, rt.Outcome{}
	}
	return m, nil, rt.Outcome{}
}

// View renders the model into a templ Component.
func (p Program) View(_ context.Context, m Model) templ.Component {
	return Page(m)
}
