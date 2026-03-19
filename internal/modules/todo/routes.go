package todo

import (
	"github.com/go-chi/chi/v5"

	"github.com/ryanfaerman/banana/internal/kernel/access"
	"github.com/ryanfaerman/banana/internal/kernel/menu"
	"github.com/ryanfaerman/banana/internal/kernel/web"
)

// Register wires the todo module into the application.
// Call this once during startup.
func Register(r chi.Router) {
	registerMenu()
	registerRoutes(r)
}

func registerMenu() {
	menu.Register(menu.Item{
		ID:     "todo",
		Label:  "Todo",
		Href:   "/todo",
		Order:  20,
		Access: access.Public(),
	})
}

func registerRoutes(r chi.Router) {
	store := NewStore()

	// Full-page GET — rendered inside the application shell via the kernel runtime.
	r.Get("/todo", web.HandleGet(PageProgram{Store: store}))

	// POST endpoints — web.HandlePost detects the HX-Request header and either
	// renders the View as an HTML fragment (HTMX) or performs a PRG redirect.
	r.Post("/todo", web.HandlePost(AddProgram{Store: store}))
	r.Post("/todo/clear", web.HandlePost(ClearProgram{Store: store}))
	r.Post("/todo/{id}/toggle", web.HandlePost(ToggleProgram{Store: store}))
}

