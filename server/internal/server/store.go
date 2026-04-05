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

// Store is the in-memory data store for tracked groups and their usage.
// All methods are safe for concurrent access; reads use an RWMutex so
// multiple readers can proceed in parallel.
// When created with NewStoreWithFile, mutations are persisted to a JSON file.
type Store struct {
	mu                   sync.RWMutex
	groups               map[string]*Group // keyed by group Name
	filePath             string            // empty means no persistence
	clock                func() time.Time
	testPopupRequestedAt time.Time
}

// persistedGroup is the JSON-serialisable representation of a Group.
type persistedGroup struct {
	Name          string   `json:"name"`
	Processes     []string `json:"processes"`
	DailyBudget   int64    `json:"daily_budget_ns"`
	UsedToday     int64    `json:"used_today_ns"`
	LastResetDate string   `json:"last_reset_date"`
}

// persistedData is the top-level structure written to the JSON file.
type persistedData struct {
	Groups map[string]persistedGroup `json:"groups"`
}

// persistedApp is the old JSON format for migration purposes.
type persistedApp struct {
	ExeName       string `json:"exe_name"`
	DailyBudget   int64  `json:"daily_budget_ns"`
	UsedToday     int64  `json:"used_today_ns"`
	LastResetDate string `json:"last_reset_date"`
}

// NewStore creates an empty Store ready for use (no file persistence).
func NewStore() *Store {
	return &Store{
		groups: make(map[string]*Group),
		clock:  time.Now,
	}
}

// NewStoreWithFile creates a Store backed by the given JSON file.
// If the file exists, state is loaded from it; otherwise the store starts empty.
func NewStoreWithFile(filePath string) *Store {
	s := &Store{
		groups:   make(map[string]*Group),
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

	data := persistedData{Groups: make(map[string]persistedGroup, len(s.groups))}
	for k, g := range s.groups {
		data.Groups[k] = persistedGroup{
			Name:          g.Name,
			Processes:     g.Processes,
			DailyBudget:   int64(g.DailyBudget),
			UsedToday:     int64(g.UsedToday),
			LastResetDate: g.LastResetDate,
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
// Supports migration from the old "apps" format to the new "groups" format.
func (s *Store) load() {
	if s.filePath == "" {
		return
	}

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		return
	}

	// Two-step unmarshal to support migration from old format.
	var raw struct {
		Apps   map[string]persistedApp   `json:"apps"`
		Groups map[string]persistedGroup `json:"groups"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return
	}

	if len(raw.Groups) > 0 {
		for k, pg := range raw.Groups {
			s.groups[k] = &Group{
				Name:          pg.Name,
				Processes:     pg.Processes,
				DailyBudget:   time.Duration(pg.DailyBudget),
				UsedToday:     time.Duration(pg.UsedToday),
				LastResetDate: pg.LastResetDate,
			}
		}
		return
	}

	// Migration: convert old apps into single-process groups.
	if len(raw.Apps) > 0 {
		for _, pa := range raw.Apps {
			s.groups[pa.ExeName] = &Group{
				Name:          pa.ExeName,
				Processes:     []string{pa.ExeName},
				DailyBudget:   time.Duration(pa.DailyBudget),
				UsedToday:     time.Duration(pa.UsedToday),
				LastResetDate: pa.LastResetDate,
			}
		}
		// Re-save in the new format so future loads use the new schema.
		s.save()
	}
}

// AddGroup registers a new group to track. Returns an error if a
// group with the same name already exists (duplicate → 409 in the API).
func (s *Store) AddGroup(name string, process string, budget time.Duration) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.groups[name]; exists {
		return nil, fmt.Errorf("group already exists: %s", name)
	}

	g := &Group{
		Name:        name,
		Processes:   []string{process},
		DailyBudget: budget,
	}
	s.groups[name] = g
	s.save()
	return g, nil
}

// GetGroup returns the Group with the given name, or an error if not found.
func (s *Store) GetGroup(name string) (*Group, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.groups[name]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	return g, nil
}

// ListGroups returns all tracked groups sorted alphabetically by Name.
func (s *Store) ListGroups() []*Group {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		result = append(result, g)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// UpdateGroup changes the daily budget and process list for an existing group.
// Returns an error if the group is not found (→ 404 in the API).
func (s *Store) UpdateGroup(name string, budget time.Duration, processes []string) (*Group, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.groups[name]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", name)
	}
	g.DailyBudget = budget
	g.Processes = processes
	s.save()
	return g, nil
}

// DeleteGroup removes a tracked group. Returns an error if not found.
func (s *Store) DeleteGroup(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.groups[name]; !ok {
		return fmt.Errorf("group not found: %s", name)
	}
	delete(s.groups, name)
	s.save()
	return nil
}

// RecordUsage adds the given number of seconds to the usage of any group
// whose Processes slice contains exeName. If the current date differs from
// LastResetDate, UsedToday is reset to zero first (automatic daily reset).
// Returns an error if no group contains the given exeName.
func (s *Store) RecordUsage(exeName string, seconds int, totalSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for _, g := range s.groups {
		if !containsProcess(g.Processes, exeName) {
			continue
		}
		found = true

		// Reset accumulated usage when the date rolls over
		today := s.clock().Format("2006-01-02")
		if g.LastResetDate != today {
			g.UsedToday = 0
			g.LastResetDate = today
		}
		g.UsedToday += time.Duration(seconds) * time.Second

		// If the client reports a total that's higher than what we have, use it
		// (allows recovery after server restart)
		if totalSeconds > 0 {
			clientTotal := time.Duration(totalSeconds) * time.Second
			if clientTotal > g.UsedToday {
				g.UsedToday = clientTotal
			}
		}
	}

	if !found {
		return fmt.Errorf("no group contains process: %s", exeName)
	}

	s.save()
	return nil
}

// containsProcess checks whether the slice contains the given process name.
func containsProcess(processes []string, name string) bool {
	for _, p := range processes {
		if p == name {
			return true
		}
	}
	return false
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

// GetUsageSummary returns a UsageSummary for every tracked group, sorted by
// Name. This is the primary data source for the UI's "Tracked Applications"
// table (GET /api/usage/today).
func (s *Store) GetUsageSummary() []UsageSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]UsageSummary, 0, len(s.groups))
	for _, g := range s.groups {
		result = append(result, g.ToUsageSummary())
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}
