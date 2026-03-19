// Package runtime provides an Elm/Bubble Tea–inspired SSR kernel.
// A Program is a page-level state machine:
//
//	Init  → (model, Cmd)
//	Update(model, Msg) → (model, Cmd)
//	View(model) → templ.Component
//
// POST handlers run the update loop, execute commands, and redirect (PRG).
// GET handlers call Init + View.
package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
)

// Msg is the type of all program messages.  Any value is a valid Msg; nil
// means "no message" and terminates a Cmd chain.
type Msg = any

// Cmd is a side-effecting function that produces an optional follow-up message.
// Returning nil means "no further messages".
type Cmd func() Msg

// Outcome carries the side-effect decisions a POST program can make.
// RedirectTo overrides the default Referer redirect; Flashes are stored for the next request.
type Outcome struct {
	// RedirectTo is the URL to redirect to after a POST. If empty, the kernel
	// falls back to the HTTP Referer header.
	RedirectTo string
	// Flashes are short-lived notification messages to surface to the user.
	Flashes []Flash
}

// Flash is a short-lived UI notification.
type Flash struct {
	Level   string // "info", "success", "warning", "error"
	Message string
}

// Program is the page-level state machine interface.
// model is an opaque value owned by the program implementation.
type Program interface {
	// Init creates the initial model and an optional first command.
	Init(ctx context.Context, r *http.Request) (model any, cmd Cmd)
	// Update applies a message to the model and returns the updated model,
	// an optional next command, and the accumulated Outcome (redirect, flashes).
	Update(ctx context.Context, model any, msg Msg) (any, Cmd, Outcome)
	// View renders the model into a templ Component.
	View(ctx context.Context, model any) templ.Component
}

// MsgDecoder decodes an *http.Request into a Msg.
// Pass one to HandlePost or HandleGet at route-registration time so that
// programs remain pure state machines with no HTTP awareness.
type MsgDecoder func(r *http.Request) (Msg, error)

// maxLoopIterations is the maximum number of Update/Cmd cycles allowed per
// request.  This prevents infinite loops caused by a Cmd producing a Msg that
// always returns another Cmd.  32 is a generous but bounded ceiling – real
// request flows rarely exceed a handful of iterations.
const maxLoopIterations = 32

// Runner executes the Elm-style update/cmd loop for a single HTTP interaction.
type Runner struct {
	Program Program
	Logger  *slog.Logger
}

// NewRunner constructs a Runner for the given Program.
func NewRunner(p Program) *Runner {
	return &Runner{Program: p, Logger: slog.Default()}
}

// RunInit executes Init and drains any initial Cmd chain.
func (r *Runner) RunInit(ctx context.Context, req *http.Request) (any, error) {
	model, cmd := r.Program.Init(ctx, req)
	if cmd == nil {
		return model, nil
	}
	var err error
	model, _, err = r.drainCmd(ctx, model, cmd)
	return model, err
}

// RunInitWithMsg is like RunInit but, after draining the initial Cmd, it
// decodes a Msg from the request using dec and runs one more Update/Cmd cycle.
// If dec is nil, it behaves identically to RunInit.
func (r *Runner) RunInitWithMsg(ctx context.Context, req *http.Request, dec MsgDecoder) (any, error) {
	model, err := r.RunInit(ctx, req)
	if err != nil || dec == nil {
		return model, err
	}
	msg, err := dec(req)
	if err != nil {
		return model, fmt.Errorf("runtime: decode msg: %w", err)
	}
	if msg == nil {
		return model, nil
	}
	updatedModel, nextCmd, _ := r.Program.Update(ctx, model, msg)
	model = updatedModel
	if nextCmd != nil {
		model, _, err = r.drainCmd(ctx, model, nextCmd)
	}
	return model, err
}

// RunPostWithModel runs Init → drain Cmds → decode → Update → drain Cmds and
// returns the final model so that fragment handlers can call View afterwards.
func (r *Runner) RunPostWithModel(ctx context.Context, req *http.Request, dec MsgDecoder) (any, Outcome, error) {
	model, initCmd := r.Program.Init(ctx, req)
	if initCmd != nil {
		var err error
		var o Outcome
		model, o, err = r.drainCmd(ctx, model, initCmd)
		if err != nil {
			return model, o, err
		}
	}

	msg, err := dec(req)
	if err != nil {
		return nil, Outcome{}, fmt.Errorf("runtime: decode msg: %w", err)
	}

	updatedModel, nextCmd, outcome := r.Program.Update(ctx, model, msg)
	model = updatedModel

	if nextCmd != nil {
		var moreOutcome Outcome
		model, moreOutcome, err = r.drainCmd(ctx, model, nextCmd)
		if err != nil {
			return model, outcome, err
		}
		outcome = mergeOutcome(outcome, moreOutcome)
	}
	return model, outcome, nil
}

// RunPost executes the decoder → Update → Cmd loop and returns the accumulated Outcome.
func (r *Runner) RunPost(ctx context.Context, req *http.Request, dec MsgDecoder) (Outcome, error) {
	_, outcome, err := r.RunPostWithModel(ctx, req, dec)
	return outcome, err
}

// drainCmd executes a Cmd and follows the resulting Msg → Update → Cmd chain
// until a Cmd returns nil or the iteration limit is reached.
func (r *Runner) drainCmd(ctx context.Context, model any, cmd Cmd) (any, Outcome, error) {
	accumulated := Outcome{}
	iterations := 0
	for cmd != nil {
		if iterations >= maxLoopIterations {
			return model, accumulated, fmt.Errorf("runtime: update loop exceeded %d iterations", maxLoopIterations)
		}
		msg := cmd()
		if msg == nil {
			break
		}
		var outcome Outcome
		var nextCmd Cmd
		model, nextCmd, outcome = r.Program.Update(ctx, model, msg)
		accumulated = mergeOutcome(accumulated, outcome)
		cmd = nextCmd
		iterations++
	}
	return model, accumulated, nil
}

func mergeOutcome(base, overlay Outcome) Outcome {
	if overlay.RedirectTo != "" {
		base.RedirectTo = overlay.RedirectTo
	}
	base.Flashes = append(base.Flashes, overlay.Flashes...)
	return base
}
