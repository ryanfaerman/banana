package example

import "sync"

// Store is a simple thread-safe in-memory state store for the example module.
// In a real application this would be backed by a database or session store.
type Store struct {
	mu      sync.RWMutex
	counter int
	lastMsg string
}

func NewStore() *Store { return &Store{} }

func (s *Store) Load() Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Model{Counter: s.counter, LastMsg: s.lastMsg}
}

func (s *Store) Save(m Model) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter = m.Counter
	s.lastMsg = m.LastMsg
}
