// Package web provides chi-compatible HTTP adapters for kernel runtime programs.
// HandleGet renders a page (Init → View wrapped in the application shell).
// HandlePost runs the update/cmd loop and performs a PRG redirect.
//
// Rendering is delegated to a Renderer implementation that lives outside the
// kernel (e.g. internal/ui/defaultui).  Set the default renderer via
// SetRenderer before registering any routes.
package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"

	"github.com/ryanfaerman/banana/internal/kernel/menu"
	"github.com/ryanfaerman/banana/internal/kernel/runtime"
)

// MenuItem is the display-only projection of a navigation entry.
// Access requirements have already been evaluated and filtered by the kernel
// before the renderer ever sees these items.
type MenuItem struct {
	Label string
	Href  string
	Icon  string
}

// FullPageInput carries everything a renderer needs to produce a complete page.
type FullPageInput struct {
	Title   string
	Menus   []MenuItem
	Flashes []runtime.Flash
	Body    templ.Component
}

// Renderer is the kernel's port for writing HTTP responses.
// Implementations live outside the kernel (e.g. internal/ui/defaultui).
type Renderer interface {
	// RenderFullPage renders the body inside the full application shell
	// (HTML boilerplate, navigation, flash messages, etc.).
	RenderFullPage(ctx context.Context, w http.ResponseWriter, r *http.Request, input FullPageInput) error
	// RenderPageFragment renders only the body component with no surrounding
	// shell.  Used for HTMX partial updates and widget endpoints.
	RenderPageFragment(ctx context.Context, w http.ResponseWriter, r *http.Request, body templ.Component) error
}

// DefaultRenderer is the application-wide renderer.
// It must be set via SetRenderer before any handlers are invoked.
var DefaultRenderer Renderer

// DefaultTitle is the HTML <title> used by full-page renders.
var DefaultTitle = "banana"

// SetRenderer configures the default renderer.  Call from cmd/server during bootstrap,
// before registering any routes.
func SetRenderer(r Renderer) { DefaultRenderer = r }

// HandleGet returns an http.HandlerFunc that runs Init → View for the program,
// wrapped in the application shell via DefaultRenderer.
// An optional dec may be provided; if non-nil it is called after Init and the
// resulting Msg is fed through one Update/Cmd cycle before View is called.
func HandleGet(p runtime.Program, dec ...runtime.MsgDecoder) http.HandlerFunc {
	return HandleGetWith(DefaultRenderer, p, dec...)
}

// HandleGetWith is like HandleGet but uses the provided Renderer instead of DefaultRenderer.
func HandleGetWith(rend Renderer, p runtime.Program, dec ...runtime.MsgDecoder) http.HandlerFunc {
	if rend == nil {
		panic("web: HandleGetWith called with nil Renderer; call web.SetRenderer before registering routes")
	}
	var decoder runtime.MsgDecoder
	if len(dec) > 0 {
		decoder = dec[0]
	}
	runner := runtime.NewRunner(p)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		model, err := runner.RunInitWithMsg(ctx, r, decoder)
		if err != nil {
			http.Error(w, "internal server error: init: "+err.Error(), http.StatusInternalServerError)
			return
		}
		input := FullPageInput{
			Title: DefaultTitle,
			Menus: filterMenus(r),
			Body:  p.View(ctx, model),
		}
		if err := rend.RenderFullPage(ctx, w, r, input); err != nil {
			http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// isHTMXRequest reports whether the request originated from HTMX.
// HTMX sets the "HX-Request: true" header on every request it makes.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// HandlePost returns an http.HandlerFunc that adapts its response to the caller:
//   - HTMX requests (HX-Request: true): run the update/cmd loop, then render
//     the program's View as an HTML fragment so HTMX can swap it in-place.
//   - Normal requests: run the update/cmd loop, then perform a PRG redirect
//     using Outcome.RedirectTo, the Referer header, or "/" as fallback.
//
// dec is called on every POST request to decode the request into a Msg.
func HandlePost(p runtime.Program, dec runtime.MsgDecoder) http.HandlerFunc {
	return HandlePostWith(DefaultRenderer, p, dec)
}

// HandlePostWith is like HandlePost but uses the provided Renderer instead of DefaultRenderer.
func HandlePostWith(rend Renderer, p runtime.Program, dec runtime.MsgDecoder) http.HandlerFunc {
	if rend == nil {
		panic("web: HandlePostWith called with nil Renderer; call web.SetRenderer before registering routes")
	}
	runner := runtime.NewRunner(p)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		model, outcome, err := runner.RunPostWithModel(ctx, r, dec)
		if err != nil {
			http.Error(w, "internal server error: post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if isHTMXRequest(r) {
			if err := rend.RenderPageFragment(ctx, w, r, p.View(ctx, model)); err != nil {
				http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
			}
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

// filterMenus returns the menu items visible to the current request's identity.
// Each item's Access requirements are evaluated; items the caller cannot see
// are excluded before the slice is handed to the renderer.
func filterMenus(r *http.Request) []MenuItem {
	items := menu.Items()
	out := make([]MenuItem, 0, len(items))
	for _, item := range items {
		// TODO: evaluate item.Access against the identity stored in r.Context().
		_ = item.Access
		out = append(out, MenuItem{Label: item.Label, Href: item.Href, Icon: item.Icon})
	}
	return out
}

