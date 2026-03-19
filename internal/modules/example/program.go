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

	"github.com/ryanfaerman/banana/internal/kernel/tea"
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

// Program implements tea.Program[Model, Msg] for the example page.
type Program struct{}

// Init creates the initial model and sends PageOpened to the update loop.
func (p Program) Init(_ context.Context, _ *http.Request) (Model, tea.Cmd[Msg]) {
	return Model{}, func(_ context.Context) []Msg {
		return []Msg{PageOpened{}}
	}
}

// Update handles messages and returns the updated model, optional next Cmd,
// and any Outcome (redirect, flashes).
func (p Program) Update(_ context.Context, m Model, msg Msg) (Model, tea.Cmd[Msg], tea.Outcome) {
	switch msg := msg.(type) {
	case PageOpened:
		// nothing to do on initial open
		return m, nil, tea.Outcome{}

	case ActionSubmitted:
		m.LastMsg = msg.Message
		// Return a Cmd that produces CounterIncremented.
		cmd := func(_ context.Context) []Msg {
			return []Msg{CounterIncremented{}}
		}
		outcome := tea.Outcome{
			// Redirect to the page itself (could be left empty to use Referer).
			RedirectTo: "/example",
			Flashes: []tea.Flash{
				{Level: "success", Message: fmt.Sprintf("Submitted: %q", msg.Message)},
			},
		}
		return m, cmd, outcome

	case CounterIncremented:
		m.Counter++
		return m, nil, tea.Outcome{}
	}
	return m, nil, tea.Outcome{}
}

// View renders the model into a templ Component.
func (p Program) View(_ context.Context, m Model) templ.Component {
	return Page(m)
}

// DecodeMsg parses the POST form body into an ActionSubmitted message.
func (p Program) DecodeMsg(r *http.Request) (Msg, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("example: parse form: %w", err)
	}
	return ActionSubmitted{Message: r.FormValue("message")}, nil
}
