package example

import (
	"github.com/go-chi/chi/v5"

	"github.com/ryanfaerman/banana/internal/kernel/access"
	"github.com/ryanfaerman/banana/internal/kernel/menu"
	"github.com/ryanfaerman/banana/internal/kernel/web"
)

// Register wires the example module into the application.
// Call this once during startup.
func Register(r chi.Router) {
	registerMenu()
	registerRoutes(r)
}

func registerMenu() {
	menu.Register(menu.Item{
		ID:    "example",
		Label: "Example",
		Href:  "/example",
		Order: 10,
	})
}

func registerRoutes(r chi.Router) {
	prog := Program{Store: NewStore()}

	// GET /example – private (requires authn); access middleware is applied
	// per-route so it does not bleed onto other routes.
	getAccess := access.Private()
	r.With(access.Middleware(getAccess)...).Get("/example", web.HandleGet(prog))

	// POST /example/action – also private.
	postAccess := access.Private()
	r.With(access.Middleware(postAccess)...).Post("/example/action", web.HandlePost(prog))
}
