// Package menu provides a simple registry for navigation menu items.
// Modules register items during startup; the kernel layout renders them.
package menu

import (
	"sort"
	"sync"

	"github.com/ryanfaerman/banana/internal/kernel/access"
)

// Item represents a single navigation link.
type Item struct {
	ID    string
	Label string
	Href  string
	Order int
	// Icon is an optional CSS class or SVG string.
	Icon string
	// Access describes who may see this menu item.
	// The kernel evaluates this before passing items to the renderer.
	Access access.Access
}

// Registry holds all registered menu items.
type Registry struct {
	mu    sync.RWMutex
	items []Item
}

// DefaultRegistry is the package-level singleton used by modules.
var DefaultRegistry = &Registry{}

// Register adds an item to the registry.
func (reg *Registry) Register(item Item) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.items = append(reg.items, item)
}

// Items returns all registered items sorted by Order then ID.
func (reg *Registry) Items() []Item {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	out := make([]Item, len(reg.items))
	copy(out, reg.items)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order != out[j].Order {
			return out[i].Order < out[j].Order
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Register adds an item to the default registry.
func Register(item Item) { DefaultRegistry.Register(item) }

// Items returns items from the default registry.
func Items() []Item { return DefaultRegistry.Items() }
