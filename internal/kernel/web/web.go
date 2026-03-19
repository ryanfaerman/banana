// Package web provides chi-compatible HTTP adapters for kernel Tea programs.
// HandleGet renders a page (Init → View wrapped in the kernel shell).
// HandlePost runs the update/cmd loop and performs a PRG redirect.
package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/ryanfaerman/banana/internal/kernel/menu"
	"github.com/ryanfaerman/banana/internal/kernel/tea"
	"github.com/ryanfaerman/banana/internal/kernel/ui"
)

// Renderer renders a templ.Component into an http.ResponseWriter inside the
// kernel shell.
type Renderer struct {
	Title    string
	MenusFunc func() []menu.Item
}

// DefaultRenderer uses the default menu registry and a generic app title.
var DefaultRenderer = &Renderer{
	Title:    "banana",
	MenusFunc: menu.Items,
}

// Render wraps body in the kernel shell and writes it to w.
func (rend *Renderer) Render(ctx context.Context, w http.ResponseWriter, r *http.Request, body templ.Component, flashes []ui.Flash) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	menus := rend.MenusFunc()
	var bodyWithFlashes templ.Component
	if len(flashes) > 0 {
		bodyWithFlashes = templ.ComponentFunc(func(ctx context.Context, w2 io.Writer) error {
			if err := ui.FlashList(flashes).Render(ctx, w2); err != nil {
				return err
			}
			return body.Render(ctx, w2)
		})
	} else {
		bodyWithFlashes = body
	}
	return ui.Shell(rend.Title, menus, bodyWithFlashes).Render(ctx, w)
}

// HandleGet returns an http.HandlerFunc that runs Init → View for the program.
func HandleGet[M any, Msg any](p tea.Program[M, Msg]) http.HandlerFunc {
	return HandleGetWith(DefaultRenderer, p)
}

// HandleGetWith is like HandleGet but uses a custom Renderer.
func HandleGetWith[M any, Msg any](rend *Renderer, p tea.Program[M, Msg]) http.HandlerFunc {
	runner := tea.NewRunner(p)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		model, err := runner.RunInit(ctx, r)
		if err != nil {
			http.Error(w, "internal server error: init: "+err.Error(), http.StatusInternalServerError)
			return
		}
		body := p.View(ctx, model)
		if err := rend.Render(ctx, w, r, body, nil); err != nil {
			http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandlePost returns an http.HandlerFunc that runs the update/cmd loop and
// redirects (PRG).  The program's Outcome.RedirectTo is used if set; otherwise
// the request's Referer header is used; finally "/" is the fallback.
func HandlePost[M any, Msg any](p tea.Program[M, Msg]) http.HandlerFunc {
	return HandlePostWith(DefaultRenderer, p)
}

// HandlePostWith is like HandlePost but uses a custom Renderer.
func HandlePostWith[M any, Msg any](rend *Renderer, p tea.Program[M, Msg]) http.HandlerFunc {
	runner := tea.NewRunner(p)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		outcome, err := runner.RunPost(ctx, r)
		if err != nil {
			http.Error(w, "internal server error: post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		target := outcome.RedirectTo
		if target == "" {
			target = r.Referer()
		}
		if target == "" {
			target = "/"
		}

		// TODO: persist flashes to a real session-backed flash store.
		// For now, flashes are intentionally discarded; they would normally be
		// written to a cookie/session and read on the subsequent GET request.
		if len(outcome.Flashes) > 0 {
			slog.DebugContext(ctx, "flashes not yet persisted (stub)", "count", len(outcome.Flashes))
		}

		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}
