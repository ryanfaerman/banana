package todo

import (
	"fmt"
	"net/http"

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

	// POST endpoints — a single Program handles all mutations; decoders are
	// provided inline at route registration so the Program stays HTTP-free.
	// web.HandlePost detects HX-Request and either renders the View as a
	// fragment (HTMX) or performs a PRG redirect.
	program := Program{Store: store}
	r.Post("/todo", web.HandlePost(program, func(r *http.Request) (PostMsg, error) {
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("todo: parse form: %w", err)
		}
		return AddRequested{Text: r.FormValue("text")}, nil
	}))
	r.Post("/todo/clear", web.HandlePost(program, func(_ *http.Request) (PostMsg, error) {
		return ClearRequested{}, nil
	}))
	r.Post("/todo/{id}/toggle", web.HandlePost(program, func(r *http.Request) (PostMsg, error) {
		return ToggleRequested{ID: chi.URLParam(r, "id")}, nil
	}))
}

