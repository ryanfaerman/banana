package todo

import (
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
	prog := Program{Store: store}

	// Full-page GET — rendered inside the application shell via the kernel runtime.
	r.Get("/todo", web.HandleGet(prog))

	// HTMX fragment endpoints — each returns only the #todo-list fragment,
	// so the browser can swap it in-place without a full page refresh.
	r.Post("/todo", htmxAdd(store))
	r.Post("/todo/clear", htmxClear(store))
	r.Post("/todo/{id}/toggle", htmxToggle(store))
}

// htmxAdd adds a new todo and returns the updated list fragment.
func htmxAdd(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if text := r.FormValue("text"); text != "" {
			store.Add(text)
		}
		renderFragment(w, r, store)
	}
}

// htmxToggle flips the done state of a single todo and returns the updated list fragment.
func htmxToggle(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		store.Toggle(id)
		renderFragment(w, r, store)
	}
}

// htmxClear removes all todos and returns the updated (empty) list fragment.
func htmxClear(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		store.Clear()
		renderFragment(w, r, store)
	}
}

// renderFragment renders the TodoList component as an HTML fragment using the
// kernel's RenderPageFragment port, which writes just the body (no shell).
func renderFragment(w http.ResponseWriter, r *http.Request, store *Store) {
	ctx := r.Context()
	todos := store.List()
	if err := web.DefaultRenderer.RenderPageFragment(ctx, w, r, TodoList(todos)); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}
