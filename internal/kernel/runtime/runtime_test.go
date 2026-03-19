package runtime_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-h/templ"

	"github.com/ryanfaerman/banana/internal/kernel/runtime"
)

// ---------------------------------------------------------------------------
// Minimal test program
// ---------------------------------------------------------------------------

type testModel struct{ count int }

type incrementMsg struct{}
type doneMsg struct{ value int }

type testProgram struct{}

func (testProgram) Init(_ context.Context, _ *http.Request) (any, runtime.Cmd) {
	return testModel{}, nil
}

func (testProgram) Update(_ context.Context, model any, msg runtime.Msg) (any, runtime.Cmd, runtime.Outcome) {
	m := model.(testModel)
	switch msg.(type) {
	case incrementMsg:
		m.count++
		// produce a doneMsg via a Cmd
		count := m.count
		return m, func() runtime.Msg { return doneMsg{value: count} }, runtime.Outcome{}
	case doneMsg:
		return m, nil, runtime.Outcome{RedirectTo: "/done"}
	}
	return m, nil, runtime.Outcome{}
}

func (testProgram) View(_ context.Context, _ any) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return nil
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRunInit_noCmd(t *testing.T) {
	runner := runtime.NewRunner(testProgram{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	model, err := runner.RunInit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.(testModel).count != 0 {
		t.Errorf("expected count 0, got %d", model.(testModel).count)
	}
}

func TestRunPost_updateCmdLoop(t *testing.T) {
	prog := testProgram{}
	runner := runtime.NewRunner(prog)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	outcome, err := runner.RunPost(context.Background(), req, func(_ *http.Request) (runtime.Msg, error) {
		return incrementMsg{}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// doneMsg sets RedirectTo "/done"
	if outcome.RedirectTo != "/done" {
		t.Errorf("expected redirect '/done', got %q", outcome.RedirectTo)
	}
}

func TestCmd_nilTerminatesChain(t *testing.T) {
	called := false
	cmd := runtime.Cmd(func() runtime.Msg {
		called = true
		return nil // nil terminates the chain
	})
	if cmd == nil {
		t.Error("Cmd should not be nil")
	}
	msg := cmd()
	if !called {
		t.Error("Cmd was not called")
	}
	if msg != nil {
		t.Errorf("expected nil msg, got %v", msg)
	}
}
