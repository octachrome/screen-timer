package server

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Store struct {
	mu   sync.RWMutex
	apps map[string]*Application
}

func NewStore() *Store {
	return &Store{apps: make(map[string]*Application)}
}

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

func (s *Store) GetApp(exeName string) (*Application, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.apps[exeName]
	if !ok {
		return nil, fmt.Errorf("app not found: %s", exeName)
	}
	return app, nil
}

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

func (s *Store) DeleteApp(exeName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.apps[exeName]; !ok {
		return fmt.Errorf("app not found: %s", exeName)
	}
	delete(s.apps, exeName)
	return nil
}

func (s *Store) RecordUsage(exeName string, seconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.apps[exeName]
	if !ok {
		return fmt.Errorf("app not found: %s", exeName)
	}

	today := time.Now().Format("2006-01-02")
	if app.LastResetDate != today {
		app.UsedToday = 0
		app.LastResetDate = today
	}
	app.UsedToday += time.Duration(seconds) * time.Second
	return nil
}

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
