// cmd/server is the application entry point.
// It wires up the chi router, registers modules, and starts the HTTP server.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ryanfaerman/banana/internal/kernel/web"
	"github.com/ryanfaerman/banana/internal/modules/example"
	"github.com/ryanfaerman/banana/internal/ui/defaultui"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	// Wire the default full-page renderer into the kernel before routes are registered.
	web.SetRenderer(defaultui.New())

	r := chi.NewRouter()

	// Kernel middleware stack.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Home redirect to example page.
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/example", http.StatusFound)
	})

	// Register modules.
	example.Register(r)

	slog.Info("starting server", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
