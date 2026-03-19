package menu_test

import (
	"testing"

	"github.com/ryanfaerman/banana/internal/kernel/menu"
)

func TestRegistry_RegisterAndItems(t *testing.T) {
	reg := &menu.Registry{}
	reg.Register(menu.Item{ID: "b", Label: "B", Order: 2})
	reg.Register(menu.Item{ID: "a", Label: "A", Order: 1})
	reg.Register(menu.Item{ID: "c", Label: "C", Order: 2})

	items := reg.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// sorted: a(1) < b(2) < c(2) — b before c because "b" < "c"
	if items[0].ID != "a" || items[1].ID != "b" || items[2].ID != "c" {
		t.Errorf("unexpected order: %v", items)
	}
}
