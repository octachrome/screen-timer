package tests

import (
	"testing"
	"time"

	"github.com/octachrome/screen-timer/server/internal/server"
)

func TestAddApp(t *testing.T) {
	s := server.NewStore()
	app, err := s.AddApp("firefox", 60*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.ExeName != "firefox" {
		t.Errorf("ExeName = %q, want %q", app.ExeName, "firefox")
	}
	if app.DailyBudget != 60*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", app.DailyBudget, 60*time.Minute)
	}

	apps := s.ListApps()
	if len(apps) != 1 {
		t.Fatalf("ListApps returned %d apps, want 1", len(apps))
	}
	if apps[0].ExeName != "firefox" {
		t.Errorf("ListApps[0].ExeName = %q, want %q", apps[0].ExeName, "firefox")
	}
	if apps[0].DailyBudget != 60*time.Minute {
		t.Errorf("ListApps[0].DailyBudget = %v, want %v", apps[0].DailyBudget, 60*time.Minute)
	}
}

func TestAddDuplicateApp(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddApp("firefox", 60*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error on first add: %v", err)
	}

	_, err = s.AddApp("firefox", 30*time.Minute)
	if err == nil {
		t.Fatal("expected error on duplicate add, got nil")
	}
}

func TestGetApp(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddApp("chrome", 45*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, err := s.GetApp("chrome")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.ExeName != "chrome" {
		t.Errorf("ExeName = %q, want %q", app.ExeName, "chrome")
	}
	if app.DailyBudget != 45*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", app.DailyBudget, 45*time.Minute)
	}

	_, err = s.GetApp("unknown")
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

func TestUpdateBudget(t *testing.T) {
	s := server.NewStore()
	_, err := s.AddApp("slack", 30*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, err := s.UpdateBudget("slack", 90*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.DailyBudget != 90*time.Minute {
		t.Errorf("DailyBudget = %v, want %v", app.DailyBudget, 90*time.Minute)
	}

	got, _ := s.GetApp("slack")
	if got.DailyBudget != 90*time.Minute {
		t.Errorf("GetApp after update: DailyBudget = %v, want %v", got.DailyBudget, 90*time.Minute)
	}

	_, err = s.UpdateBudget("unknown", 10*time.Minute)
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

func TestDeleteApp(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddApp("discord", 20*time.Minute)

	err := s.DeleteApp("discord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	apps := s.ListApps()
	if len(apps) != 0 {
		t.Errorf("ListApps returned %d apps after delete, want 0", len(apps))
	}

	err = s.DeleteApp("unknown")
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

func TestRecordUsage(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddApp("spotify", 60*time.Minute)

	err := s.RecordUsage("spotify", 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, _ := s.GetApp("spotify")
	if app.UsedToday != 30*time.Second {
		t.Errorf("UsedToday = %v, want %v", app.UsedToday, 30*time.Second)
	}
}

func TestRecordUsageAccumulates(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddApp("vscode", 120*time.Minute)

	_ = s.RecordUsage("vscode", 10)
	_ = s.RecordUsage("vscode", 20)
	_ = s.RecordUsage("vscode", 30)

	app, _ := s.GetApp("vscode")
	if app.UsedToday != 60*time.Second {
		t.Errorf("UsedToday = %v, want %v", app.UsedToday, 60*time.Second)
	}
}

func TestRecordUsageDayReset(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddApp("zoom", 60*time.Minute)

	_ = s.RecordUsage("zoom", 100)

	app, _ := s.GetApp("zoom")
	if app.UsedToday != 100*time.Second {
		t.Fatalf("UsedToday = %v, want %v", app.UsedToday, 100*time.Second)
	}

	// Simulate a day change by setting LastResetDate to yesterday.
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	app.LastResetDate = yesterday

	_ = s.RecordUsage("zoom", 25)

	app, _ = s.GetApp("zoom")
	if app.UsedToday != 25*time.Second {
		t.Errorf("UsedToday after day reset = %v, want %v", app.UsedToday, 25*time.Second)
	}
}

func TestRecordUsageUnknownApp(t *testing.T) {
	s := server.NewStore()

	err := s.RecordUsage("nonexistent", 10)
	if err == nil {
		t.Fatal("expected error for unknown app, got nil")
	}
}

func TestGetUsageSummary(t *testing.T) {
	s := server.NewStore()
	_, _ = s.AddApp("alpha", 60*time.Minute)
	_, _ = s.AddApp("beta", 120*time.Minute)

	_ = s.RecordUsage("alpha", 600)  // 10 minutes
	_ = s.RecordUsage("beta", 1800)  // 30 minutes

	summaries := s.GetUsageSummary()
	if len(summaries) != 2 {
		t.Fatalf("GetUsageSummary returned %d entries, want 2", len(summaries))
	}

	// Sorted by ExeName: alpha, beta
	if summaries[0].ExeName != "alpha" {
		t.Errorf("summaries[0].ExeName = %q, want %q", summaries[0].ExeName, "alpha")
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

	if summaries[1].ExeName != "beta" {
		t.Errorf("summaries[1].ExeName = %q, want %q", summaries[1].ExeName, "beta")
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
