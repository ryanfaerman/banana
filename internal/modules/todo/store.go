package todo

import (
	"strconv"
	"sync"
)

// Todo is a single todo item.
type Todo struct {
	ID   string
	Text string
	Done bool
}

// Store is a thread-safe in-memory todo store.
// In a real application this would be backed by a database.
type Store struct {
	mu     sync.RWMutex
	todos  []Todo
	nextID int
}

// NewStore returns a new empty Store.
func NewStore() *Store { return &Store{} }

// List returns a copy of all todos in insertion order.
func (s *Store) List() []Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Todo, len(s.todos))
	copy(out, s.todos)
	return out
}

// Add appends a new todo with the given text.
func (s *Store) Add(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	s.todos = append(s.todos, Todo{ID: strconv.Itoa(s.nextID), Text: text})
}

// Toggle flips the Done state of the todo with the given ID.
func (s *Store) Toggle(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.todos {
		if s.todos[i].ID == id {
			s.todos[i].Done = !s.todos[i].Done
			return
		}
	}
}

// Clear removes all todos.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.todos = nil
}
