// Package defaultui provides the default full-page renderer for the banana
// kernel.  It implements web.Renderer using templ-based HTML templates.
//
// Wire it in from cmd/server:
//
//	web.SetRenderer(defaultui.New())
package defaultui

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"github.com/ryanfaerman/banana/internal/kernel/web"
)

// Renderer is the default implementation of web.Renderer.
// It renders full pages using the templ shell defined in layout.templ and
// renders fragments with no surrounding shell (for HTMX / widget endpoints).
type Renderer struct{}

// New returns a new default Renderer ready for use.
func New() *Renderer { return &Renderer{} }

// RenderFullPage renders input.Body inside the application shell, including
// navigation (already filtered by the kernel) and any flash messages.
func (rend *Renderer) RenderFullPage(ctx context.Context, w http.ResponseWriter, r *http.Request, input web.FullPageInput) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return shell(input.Title, input.Menus, input.Body, input.Flashes).Render(ctx, w)
}

// RenderPageFragment renders only the body component with no surrounding shell.
// Use this for HTMX partial updates or widget endpoints that return HTML
// fragments rather than complete pages.
func (rend *Renderer) RenderPageFragment(ctx context.Context, w http.ResponseWriter, _ *http.Request, body templ.Component) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return body.Render(ctx, w)
}
