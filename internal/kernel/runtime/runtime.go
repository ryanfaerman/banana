// Package runtime provides an Elm/Bubble Tea–inspired SSR kernel.
// A Program is a typed page-level state machine:
//
//	Init  → (Model, Cmd)
//	Update(Model, Msg) → (Model, Cmd)
//	View(Model) → templ.Component
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

// Cmd is a side-effecting function that returns zero or more follow-up messages.
// Returning nil (or an empty slice) means "no further messages".
type Cmd[Msg any] func(ctx context.Context) []Msg

// NoCmd returns a nil Cmd, signalling no side effects.
func NoCmd[Msg any]() Cmd[Msg] { return nil }

// BatchCmd combines multiple Cmds into one.
func BatchCmd[Msg any](cmds ...Cmd[Msg]) Cmd[Msg] {
	return func(ctx context.Context) []Msg {
		var msgs []Msg
		for _, c := range cmds {
			if c != nil {
				msgs = append(msgs, c(ctx)...)
			}
		}
		return msgs
	}
}

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
// M is the page Model, Msg is the page's local sum type.
type Program[M any, Msg any] interface {
	// Init creates the initial model and an optional first command.
	Init(ctx context.Context, r *http.Request) (M, Cmd[Msg])
	// Update applies a message to the model and returns the updated model,
	// an optional next command, and the accumulated Outcome (redirect, flashes).
	Update(ctx context.Context, model M, msg Msg) (M, Cmd[Msg], Outcome)
	// View renders the model into a templ Component.
	View(ctx context.Context, model M) templ.Component
}

// MsgDecoder decodes an *http.Request into a Msg.
// Pass one to HandlePost or HandleGet at route-registration time so that
// programs remain pure state machines with no HTTP awareness.
type MsgDecoder[Msg any] func(r *http.Request) (Msg, error)

// maxLoopIterations is the maximum number of Update/Cmd cycles allowed per
// request.  This prevents infinite loops caused by a Cmd producing a Msg that
// always returns another Cmd.  32 is a generous but bounded ceiling – real
// request flows rarely exceed a handful of iterations.
const maxLoopIterations = 32

// Runner executes the Elm-style update/cmd loop for a single HTTP interaction.
type Runner[M any, Msg any] struct {
	Program Program[M, Msg]
	Logger  *slog.Logger
}

// NewRunner constructs a Runner for the given Program.
func NewRunner[M any, Msg any](p Program[M, Msg]) *Runner[M, Msg] {
	return &Runner[M, Msg]{Program: p, Logger: slog.Default()}
}

// RunInit executes Init and returns the model ready for rendering.
func (r *Runner[M, Msg]) RunInit(ctx context.Context, req *http.Request) (M, error) {
	model, cmd := r.Program.Init(ctx, req)
	if cmd == nil {
		return model, nil
	}
	msgs := cmd(ctx)
	var outcome Outcome
	var err error
	model, outcome, err = r.drainMsgs(ctx, model, msgs)
	_ = outcome // init outcomes (redirects, flashes) are ignored for GET
	return model, err
}

// RunInitWithMsg is like RunInit but, after draining the initial Cmd, it
// decodes a Msg from the request using dec and runs one more Update/Cmd cycle.
// If dec is nil, it behaves identically to RunInit.
func (r *Runner[M, Msg]) RunInitWithMsg(ctx context.Context, req *http.Request, dec MsgDecoder[Msg]) (M, error) {
	model, err := r.RunInit(ctx, req)
	if err != nil || dec == nil {
		return model, err
	}
	msg, err := dec(req)
	if err != nil {
		return model, fmt.Errorf("runtime: decode msg: %w", err)
	}
	updatedModel, nextCmd, _ := r.Program.Update(ctx, model, msg)
	model = updatedModel
	if nextCmd != nil {
		msgs := nextCmd(ctx)
		model, _, err = r.drainMsgs(ctx, model, msgs)
	}
	return model, err
}

// RunPostWithModel is like RunPost but also returns the final model so that
// fragment handlers (e.g. HandleFragment) can call View after the update/cmd loop.
func (r *Runner[M, Msg]) RunPostWithModel(ctx context.Context, req *http.Request, dec MsgDecoder[Msg]) (M, Outcome, error) {
	model, cmd := r.Program.Init(ctx, req)
	if cmd != nil {
		initMsgs := cmd(ctx)
		var err error
		var o Outcome
		model, o, err = r.drainMsgs(ctx, model, initMsgs)
		if err != nil {
			return model, o, err
		}
	}

	msg, err := dec(req)
	if err != nil {
		var zero M
		return zero, Outcome{}, fmt.Errorf("runtime: decode msg: %w", err)
	}

	updatedModel, nextCmd, outcome := r.Program.Update(ctx, model, msg)
	model = updatedModel

	if nextCmd != nil {
		msgs := nextCmd(ctx)
		var moreOutcome Outcome
		model, moreOutcome, err = r.drainMsgs(ctx, model, msgs)
		if err != nil {
			return model, outcome, err
		}
		outcome = mergeOutcome(outcome, moreOutcome)
	}
	return model, outcome, nil
}

// RunPost executes the decoder → Update → Cmd loop and returns the accumulated Outcome.
func (r *Runner[M, Msg]) RunPost(ctx context.Context, req *http.Request, dec MsgDecoder[Msg]) (Outcome, error) {
	model, cmd := r.Program.Init(ctx, req)
	if cmd != nil {
		initMsgs := cmd(ctx)
		var err error
		var o Outcome
		model, o, err = r.drainMsgs(ctx, model, initMsgs)
		if err != nil {
			return o, err
		}

	}

	msg, err := dec(req)
	if err != nil {
		return Outcome{}, fmt.Errorf("runtime: decode msg: %w", err)
	}

	updatedModel, nextCmd, outcome := r.Program.Update(ctx, model, msg)
	model = updatedModel

	if nextCmd != nil {
		msgs := nextCmd(ctx)
		var moreOutcome Outcome
		model, moreOutcome, err = r.drainMsgs(ctx, model, msgs)
		if err != nil {
			return outcome, err
		}
		outcome = mergeOutcome(outcome, moreOutcome)
	}
	_ = model
	return outcome, nil
}

// drainMsgs processes a slice of messages through the bounded update/cmd loop.
func (r *Runner[M, Msg]) drainMsgs(ctx context.Context, model M, msgs []Msg) (M, Outcome, error) {
	var accumulated Outcome
	queue := msgs
	iterations := 0
	for len(queue) > 0 {
		if iterations >= maxLoopIterations {
			return model, accumulated, fmt.Errorf("runtime: update loop exceeded %d iterations", maxLoopIterations)
		}
		msg := queue[0]
		queue = queue[1:]
		var cmd Cmd[Msg]
		var outcome Outcome
		model, cmd, outcome = r.Program.Update(ctx, model, msg)
		accumulated = mergeOutcome(accumulated, outcome)
		if cmd != nil {
			more := cmd(ctx)
			queue = append(queue, more...)
		}
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
