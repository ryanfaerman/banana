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

type testMsg interface{ isTestMsg() }

type incrementMsg struct{}

func (incrementMsg) isTestMsg() {}

type doneMsg struct{ value int }

func (doneMsg) isTestMsg() {}

type testProgram struct{}

func (testProgram) Init(_ context.Context, _ *http.Request) (testModel, runtime.Cmd[testMsg]) {
	return testModel{}, nil
}

func (testProgram) Update(_ context.Context, m testModel, msg testMsg) (testModel, runtime.Cmd[testMsg], runtime.Outcome) {
	switch msg.(type) {
	case incrementMsg:
		m.count++
		// produce a doneMsg via a Cmd
		return m, func(_ context.Context) []testMsg { return []testMsg{doneMsg{value: m.count}} }, runtime.Outcome{}
	case doneMsg:
		return m, nil, runtime.Outcome{RedirectTo: "/done"}
	}
	return m, nil, runtime.Outcome{}
}

func (testProgram) View(_ context.Context, _ testModel) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, _ io.Writer) error {
		return nil
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRunInit_noCmd(t *testing.T) {
	runner := runtime.NewRunner[testModel, testMsg](testProgram{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	model, err := runner.RunInit(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.count != 0 {
		t.Errorf("expected count 0, got %d", model.count)
	}
}

func TestRunPost_updateCmdLoop(t *testing.T) {
	prog := testProgram{}
	runner := runtime.NewRunner[testModel, testMsg](prog)
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	outcome, err := runner.RunPost(context.Background(), req, func(_ *http.Request) (testMsg, error) {
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

func TestBatchCmd(t *testing.T) {
	cmd1 := func(_ context.Context) []testMsg { return []testMsg{incrementMsg{}} }
	cmd2 := func(_ context.Context) []testMsg { return []testMsg{incrementMsg{}} }
	batched := runtime.BatchCmd(cmd1, cmd2)
	msgs := batched(context.Background())
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestNoCmd(t *testing.T) {
	cmd := runtime.NoCmd[testMsg]()
	if cmd != nil {
		t.Error("NoCmd should return nil")
	}
}
