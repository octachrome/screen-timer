package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store is the in-memory data store for tracked applications and their usage.
// All methods are safe for concurrent access; reads use an RWMutex so
// multiple readers can proceed in parallel.
// When created with NewStoreWithFile, mutations are persisted to a JSON file.
type Store struct {
	mu                   sync.RWMutex
	apps                 map[string]*Application // keyed by ExeName
	filePath             string                  // empty means no persistence
	clock                func() time.Time
	testPopupRequestedAt time.Time
}

// persistedApp is the JSON-serialisable representation of an Application.
type persistedApp struct {
	ExeName       string `json:"exe_name"`
	DailyBudget   int64  `json:"daily_budget_ns"`
	UsedToday     int64  `json:"used_today_ns"`
	LastResetDate string `json:"last_reset_date"`
}

// persistedData is the top-level structure written to the JSON file.
type persistedData struct {
	Apps map[string]persistedApp `json:"apps"`
}

// NewStore creates an empty Store ready for use (no file persistence).
func NewStore() *Store {
	return &Store{
		apps:  make(map[string]*Application),
		clock: time.Now,
	}
}

// NewStoreWithFile creates a Store backed by the given JSON file.
// If the file exists, state is loaded from it; otherwise the store starts empty.
func NewStoreWithFile(filePath string) *Store {
	s := &Store{
		apps:     make(map[string]*Application),
		filePath: filePath,
		clock:    time.Now,
	}
	s.load()
	return s
}

// SetClock overrides the clock function used by RecordUsage for testing.
func (s *Store) SetClock(fn func() time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clock = fn
}

// save writes the current state to the JSON file (atomic: write tmp then rename).
// Must be called while s.mu is held (write lock).
func (s *Store) save() {
	if s.filePath == "" {
		return
	}

	data := persistedData{Apps: make(map[string]persistedApp, len(s.apps))}
	for k, app := range s.apps {
		data.Apps[k] = persistedApp{
			ExeName:       app.ExeName,
			DailyBudget:   int64(app.DailyBudget),
			UsedToday:     int64(app.UsedToday),
			LastResetDate: app.LastResetDate,
		}
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return
	}
	os.Rename(tmp, s.filePath)
}

// load reads state from the JSON file. If the file doesn't exist the store
// stays empty. Must be called during construction (no lock needed).
func (s *Store) load() {
	if s.filePath == "" {
		return
	}

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}

	var data persistedData
	if err := json.Unmarshal(b, &data); err != nil {
		return
	}

	for k, pa := range data.Apps {
		s.apps[k] = &Application{
			ExeName:       pa.ExeName,
			DailyBudget:   time.Duration(pa.DailyBudget),
			UsedToday:     time.Duration(pa.UsedToday),
			LastResetDate: pa.LastResetDate,
		}
	}
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
	s.save()
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
	s.save()
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
	s.save()
	return nil
}

// RecordUsage adds the given number of seconds to an application's UsedToday.
// If the current date differs from LastResetDate, UsedToday is reset to zero
// first (automatic daily reset). Returns an error for unknown apps; the
// handler silently ignores that error so the agent doesn't fail when it
// reports usage for an app the manager has already deleted.
func (s *Store) RecordUsage(exeName string, seconds int, totalSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.apps[exeName]
	if !ok {
		return fmt.Errorf("app not found: %s", exeName)
	}

	// Reset accumulated usage when the date rolls over
	today := s.clock().Format("2006-01-02")
	if app.LastResetDate != today {
		app.UsedToday = 0
		app.LastResetDate = today
	}
	app.UsedToday += time.Duration(seconds) * time.Second

	// If the client reports a total that's higher than what we have, use it
	// (allows recovery after server restart)
	if totalSeconds > 0 {
		clientTotal := time.Duration(totalSeconds) * time.Second
		if clientTotal > app.UsedToday {
			app.UsedToday = clientTotal
		}
	}

	s.save()
	return nil
}

// RequestTestPopup records a test popup request and returns the timestamp.
func (s *Store) RequestTestPopup() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.testPopupRequestedAt = s.clock()
	return s.testPopupRequestedAt
}

// GetTestPopupRequestedAt returns the time of the last test popup request.
func (s *Store) GetTestPopupRequestedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.testPopupRequestedAt
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
