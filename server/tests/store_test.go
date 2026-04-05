package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/octachrome/screen-timer/server/internal/server"
)

func TestAddGroup(t *testing.T) {
	s := server.NewStore()
	g, err := s.AddGroup("firefox", "firefox", 60*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name != "firefox" {
		t.Errorf("Name = %q, want %q", g.Name, "firefox")
	}
	if g.DailyBudget != 60*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", g.DailyBudget, 60*time.Minute)
	}

	groups := s.ListGroups()
	if len(groups) != 1 {
		t.Fatalf("ListGroups returned %d groups, want 1", len(groups))
	}
	if groups[0].Name != "firefox" {
		t.Errorf("ListGroups[0].Name = %q, want %q", groups[0].Name, "firefox")
	}
	if groups[0].DailyBudget != 60*time.Minute {
		t.Errorf("ListGroups[0].DailyBudget = %v, want %v", groups[0].DailyBudget, 60*time.Minute)
	}
}

func TestAddDuplicateGroup(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddGroup("firefox", "firefox", 60*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error on first add: %v", err)
	}

	_, err = s.AddGroup("firefox", "firefox", 30*time.Minute)
	if err == nil {
		t.Fatal("expected error on duplicate add, got nil")
	}
}

func TestGetGroup(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddGroup("chrome", "chrome", 45*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := s.GetGroup("chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.Name != "chrome" {
		t.Errorf("Name = %q, want %q", g.Name, "chrome")
	}
	if g.DailyBudget != 45*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", g.DailyBudget, 45*time.Minute)
	}

	_, err = s.GetGroup("unknown")
	if err == nil {
		t.Fatal("expected error for unknown group, got nil")
	}
}

func TestUpdateGroup(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddGroup("slack", "slack", 30*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := s.UpdateGroup("slack", 90*time.Minute, []string{"slack"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.DailyBudget != 90*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", g.DailyBudget, 90*time.Minute)
	}

	got, _ := s.GetGroup("slack")
	if got.DailyBudget != 90*time.Minute {
		t.Errorf("GetGroup after update: DailyBudget = %v, want %v", got.DailyBudget, 90*time.Minute)
	}

	_, err = s.UpdateGroup("unknown", 10*time.Minute, []string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown group, got nil")
	}
}

func TestDeleteGroup(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("discord", "discord", 20*time.Minute)

	err := s.DeleteGroup("discord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	groups := s.ListGroups()
	if len(groups) != 0 {
		t.Errorf("ListGroups returned %d groups after delete, want 0", len(groups))
	}

	err = s.DeleteGroup("unknown")
	if err == nil {
		t.Fatal("expected error for unknown group, got nil")
	}
}

func TestRecordUsage(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("spotify", "spotify", 60*time.Minute)

	err := s.RecordUsage("spotify", 30, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, _ := s.GetGroup("spotify")
	if g.UsedToday != 30*time.Second {
		t.Errorf("UsedToday = %v, want %v", g.UsedToday, 30*time.Second)
	}
}

func TestRecordUsageAccumulates(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("vscode", "vscode", 120*time.Minute)

	_ = s.RecordUsage("vscode", 10, 0)
	_ = s.RecordUsage("vscode", 20, 0)
	_ = s.RecordUsage("vscode", 30, 0)

	g, _ := s.GetGroup("vscode")
	if g.UsedToday != 60*time.Second {
		t.Errorf("UsedToday = %v, want %v", g.UsedToday, 60*time.Second)
	}
}

func TestRecordUsageDayReset(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("zoom", "zoom", 60*time.Minute)

	_ = s.RecordUsage("zoom", 100, 0)

	g, _ := s.GetGroup("zoom")
	if g.UsedToday != 100*time.Second {
		t.Fatalf("UsedToday = %v, want %v", g.UsedToday, 100*time.Second)
	}

	// Simulate a day change by setting LastResetDate to yesterday.
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	g.LastResetDate = yesterday

	_ = s.RecordUsage("zoom", 25, 0)

	g, _ = s.GetGroup("zoom")
	if g.UsedToday != 25*time.Second {
		t.Errorf("UsedToday after day reset = %v, want %v", g.UsedToday, 25*time.Second)
	}
}

func TestRecordUsageUnknownApp(t *testing.T) {
	s := server.NewStore()

	err := s.RecordUsage("nonexistent", 10, 0)
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

func TestRecordUsageTotalRecovery(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("game", "game", 60*time.Minute)

	// Simulate: server knows 10s usage, client reports 5s delta but 100s total
	_ = s.RecordUsage("game", 10, 0)  // initial 10s
	_ = s.RecordUsage("game", 5, 100) // 5s delta, 100s total → should use 100s

	g, _ := s.GetGroup("game")
	if g.UsedToday != 100*time.Second {
		t.Errorf("UsedToday = %v, want %v", g.UsedToday, 100*time.Second)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "test.json")
	s1 := server.NewStoreWithFile(fp)
	_, _ = s1.AddGroup("chrome", "chrome", 60*time.Minute)
	_, _ = s1.AddGroup("firefox", "firefox", 120*time.Minute)
	_ = s1.RecordUsage("chrome", 300, 0)

	// Create a new store from the same file
	s2 := server.NewStoreWithFile(fp)
	groups := s2.ListGroups()
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups after reload, got %d", len(groups))
	}
	g, err := s2.GetGroup("chrome")
	if err != nil {
		t.Fatalf("chrome not found after reload: %v", err)
	}
	if g.DailyBudget != 60*time.Minute {
		t.Errorf("chrome DailyBudget = %v, want %v", g.DailyBudget, 60*time.Minute)
	}
	if g.UsedToday != 300*time.Second {
		t.Errorf("chrome UsedToday = %v, want %v", g.UsedToday, 300*time.Second)
	}
}

func TestPersistenceNonExistentFile(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "subdir", "nonexistent.json")
	s := server.NewStoreWithFile(fp)
	groups := s.ListGroups()
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for new file, got %d", len(groups))
	}
	// Should work fine — adding a group creates the file
	_, err := s.AddGroup("test", "test", 30*time.Minute)
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
}

func TestGetUsageSummary(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddGroup("alpha", "alpha", 60*time.Minute)
	_, _ = s.AddGroup("beta", "beta", 120*time.Minute)

	_ = s.RecordUsage("alpha", 600, 0) // 10 minutes
	_ = s.RecordUsage("beta", 1800, 0) // 30 minutes

	summaries := s.GetUsageSummary()
	if len(summaries) != 2 {
		t.Fatalf("GetUsageSummary returned %d entries, want 2", len(summaries))
	}

	// Sorted by Name: alpha, beta
	if summaries[0].Name != "alpha" {
		t.Errorf("summaries[0].Name = %q, want %q", summaries[0].Name, "alpha")
	}
	if summaries[0].DailyBudgetMinutes != 60 {
		t.Errorf("alpha DailyBudgetMinutes = %d, want 60", summaries[0].DailyBudgetMinutes)
	}
	if summaries[0].UsedTodayMinutes != 10 {
		t.Errorf("alpha UsedTodayMinutes = %d, want 10", summaries[0].UsedTodayMinutes)
	}
	if summaries[0].RemainingMinutes != 50 {
		t.Errorf("alpha RemainingMinutes = %d, want 50", summaries[0].RemainingMinutes)
	}

	if summaries[1].Name != "beta" {
		t.Errorf("summaries[1].Name = %q, want %q", summaries[1].Name, "beta")
	}
	if summaries[1].DailyBudgetMinutes != 120 {
		t.Errorf("beta DailyBudgetMinutes = %d, want 120", summaries[1].DailyBudgetMinutes)
	}
	if summaries[1].UsedTodayMinutes != 30 {
		t.Errorf("beta UsedTodayMinutes = %d, want 30", summaries[1].UsedTodayMinutes)
	}
	if summaries[1].RemainingMinutes != 90 {
		t.Errorf("beta RemainingMinutes = %d, want 90", summaries[1].RemainingMinutes)
	}
}

func TestRecordUsageMultipleGroups(t *testing.T) {
	s := server.NewStore()
	// Two groups both contain the same process "shared.exe"
	_, _ = s.AddGroup("group-a", "shared.exe", 60*time.Minute)
	_, _ = s.AddGroup("group-b", "shared.exe", 120*time.Minute)

	err := s.RecordUsage("shared.exe", 45, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ga, _ := s.GetGroup("group-a")
	if ga.UsedToday != 45*time.Second {
		t.Errorf("group-a UsedToday = %v, want %v", ga.UsedToday, 45*time.Second)
	}
	gb, _ := s.GetGroup("group-b")
	if gb.UsedToday != 45*time.Second {
		t.Errorf("group-b UsedToday = %v, want %v", gb.UsedToday, 45*time.Second)
	}
}

func TestPersistenceMigration(t *testing.T) {
	fp := filepath.Join(t.TempDir(), "legacy.json")

	// Write old-format JSON with "apps" key
	oldData := map[string]any{
		"apps": map[string]any{
			"chrome.exe": map[string]any{
				"exe_name":        "chrome.exe",
				"daily_budget_ns": int64(60 * time.Minute),
				"used_today_ns":   int64(10 * time.Minute),
				"last_reset_date": "2025-01-01",
			},
			"firefox.exe": map[string]any{
				"exe_name":        "firefox.exe",
				"daily_budget_ns": int64(120 * time.Minute),
				"used_today_ns":   int64(0),
				"last_reset_date": "",
			},
		},
	}
	b, err := json.Marshal(oldData)
	if err != nil {
		t.Fatalf("failed to marshal old data: %v", err)
	}
	if err := os.WriteFile(fp, b, 0o644); err != nil {
		t.Fatalf("failed to write old file: %v", err)
	}

	// Load with NewStoreWithFile — should migrate
	s := server.NewStoreWithFile(fp)
	groups := s.ListGroups()
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups after migration, got %d", len(groups))
	}

	g, err := s.GetGroup("chrome.exe")
	if err != nil {
		t.Fatalf("chrome.exe not found after migration: %v", err)
	}
	if g.Name != "chrome.exe" {
		t.Errorf("Name = %q, want %q", g.Name, "chrome.exe")
	}
	if g.DailyBudget != 60*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", g.DailyBudget, 60*time.Minute)
	}
	if g.UsedToday != 10*time.Minute {
		t.Errorf("UsedToday = %v, want %v", g.UsedToday, 10*time.Minute)
	}
	if len(g.Processes) != 1 || g.Processes[0] != "chrome.exe" {
		t.Errorf("Processes = %v, want [chrome.exe]", g.Processes)
	}
}

func TestGroupWithMultipleProcesses(t *testing.T) {
	s := server.NewStore()
	// Create group with one process
	_, err := s.AddGroup("browsers", "chrome.exe", 60*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Update to have multiple processes
	_, err = s.UpdateGroup("browsers", 60*time.Minute, []string{"chrome.exe", "firefox.exe", "edge.exe"})
	if err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}

	// Record usage for each process
	_ = s.RecordUsage("chrome.exe", 10, 0)
	_ = s.RecordUsage("firefox.exe", 20, 0)
	_ = s.RecordUsage("edge.exe", 30, 0)

	g, _ := s.GetGroup("browsers")
	if g.UsedToday != 60*time.Second {
		t.Errorf("UsedToday = %v, want %v", g.UsedToday, 60*time.Second)
	}
}
