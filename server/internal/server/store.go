package server

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Store is the in-memory data store for tracked applications and their usage.
// All methods are safe for concurrent access; reads use an RWMutex so
// multiple readers can proceed in parallel.
// Note: data is not persisted — restarting the server clears all state.
type Store struct {
	mu   sync.RWMutex
	apps map[string]*Application // keyed by ExeName
}

// NewStore creates an empty Store ready for use.
func NewStore() *Store {
	return &Store{apps: make(map[string]*Application)}
}

// AddApp registers a new application to track. Returns an error if an
// application with the same exeName already exists (duplicate → 409 in the API).
func (s *Store) AddApp(exeName string, budget time.Duration) (*Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.apps[exeName]; exists {
		return nil, fmt.Errorf("app already exists: %s", exeName)
	}

	app := &Application{
		ExeName:     exeName,
		DailyBudget: budget,
	}
	s.apps[exeName] = app
	return app, nil
}

// GetApp returns the Application with the given exeName, or an error if not found.
func (s *Store) GetApp(exeName string) (*Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.apps[exeName]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", exeName)
	}
	return app, nil
}

// ListApps returns all tracked applications sorted alphabetically by ExeName.
func (s *Store) ListApps() []*Application {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Application, 0, len(s.apps))
	for _, app := range s.apps {
		result = append(result, app)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ExeName < result[j].ExeName
	})
	return result
}

// UpdateBudget changes the daily budget for an existing application.
// Returns an error if the app is not found (→ 404 in the API).
func (s *Store) UpdateBudget(exeName string, budget time.Duration) (*Application, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.apps[exeName]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", exeName)
	}
	app.DailyBudget = budget
	return app, nil
}

// DeleteApp removes a tracked application. Returns an error if not found.
func (s *Store) DeleteApp(exeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[exeName]; !ok {
		return fmt.Errorf("app not found: %s", exeName)
	}
	delete(s.apps, exeName)
	return nil
}

// RecordUsage adds the given number of seconds to an application's UsedToday.
// If the current date differs from LastResetDate, UsedToday is reset to zero
// first (automatic daily reset). Returns an error for unknown apps; the
// handler silently ignores that error so the agent doesn't fail when it
// reports usage for an app the manager has already deleted.
func (s *Store) RecordUsage(exeName string, seconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.apps[exeName]
	if !ok {
		return fmt.Errorf("app not found: %s", exeName)
	}

	// Reset accumulated usage when the date rolls over
	today := time.Now().Format("2006-01-02")
	if app.LastResetDate != today {
		app.UsedToday = 0
		app.LastResetDate = today
	}
	app.UsedToday += time.Duration(seconds) * time.Second
	return nil
}

// GetUsageSummary returns a UsageSummary for every tracked app, sorted by
// ExeName. This is the primary data source for the UI's "Tracked Applications"
// table (GET /api/usage/today).
func (s *Store) GetUsageSummary() []UsageSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]UsageSummary, 0, len(s.apps))
	for _, app := range s.apps {
		result = append(result, app.ToUsageSummary())
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ExeName < result[j].ExeName
	})
	return result
}
