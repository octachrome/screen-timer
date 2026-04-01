package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/octachrome/screen-timer/server/internal/server"
	"github.com/octachrome/screen-timer/server/mockclient"
)

func setupTestServer() (*httptest.Server, *mockclient.Client) {
	store := server.NewStore()
	router := server.NewRouter(store)
	ts := httptest.NewServer(router)
	client := mockclient.NewClient(ts.URL)
	return ts, client
}

func TestIntegrationAddAppThenAgentPollsConfig(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddApp(mockclient.AddAppRequest{ExeName: "chrome.exe", DailyBudgetMinutes: 60})
	if err != nil {
		t.Fatalf("AddApp failed: %v", err)
	}

	configs, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].ExeName != "chrome.exe" {
		t.Errorf("expected exe_name chrome.exe, got %s", configs[0].ExeName)
	}
	if configs[0].DailyBudgetMinutes != 60 {
		t.Errorf("expected budget 60, got %d", configs[0].DailyBudgetMinutes)
	}
}

func TestIntegrationAgentPushesUsageThenManagerViews(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddApp(mockclient.AddAppRequest{ExeName: "firefox.exe", DailyBudgetMinutes: 120})
	if err != nil {
		t.Fatalf("AddApp failed: %v", err)
	}

	err = client.PushUsage([]mockclient.UsageReport{
		{ExeName: "firefox.exe", Seconds: 300},
	})
	if err != nil {
		t.Fatalf("PushUsage failed: %v", err)
	}

	summaries, err := client.GetUsageToday()
	if err != nil {
		t.Fatalf("GetUsageToday failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].UsedTodayMinutes != 5 {
		t.Errorf("expected used 5 minutes, got %d", summaries[0].UsedTodayMinutes)
	}
	if summaries[0].RemainingMinutes != 115 {
		t.Errorf("expected remaining 115 minutes, got %d", summaries[0].RemainingMinutes)
	}
}

func TestIntegrationUpdateBudgetThenAgentPolls(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddApp(mockclient.AddAppRequest{ExeName: "slack.exe", DailyBudgetMinutes: 30})
	if err != nil {
		t.Fatalf("AddApp failed: %v", err)
	}

	_, err = client.UpdateApp("slack.exe", mockclient.UpdateBudgetRequest{DailyBudgetMinutes: 90})
	if err != nil {
		t.Fatalf("UpdateApp failed: %v", err)
	}

	configs, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].DailyBudgetMinutes != 90 {
		t.Errorf("expected budget 90, got %d", configs[0].DailyBudgetMinutes)
	}
}

func TestIntegrationDeleteAppThenAgentPolls(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddApp(mockclient.AddAppRequest{ExeName: "discord.exe", DailyBudgetMinutes: 45})
	if err != nil {
		t.Fatalf("AddApp failed: %v", err)
	}

	err = client.DeleteApp("discord.exe")
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	configs, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(configs))
	}
}

func TestIntegrationAgentPushesUsageUnknownApp(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	err := client.PushUsage([]mockclient.UsageReport{
		{ExeName: "unknown.exe", Seconds: 60},
	})
	if err != nil {
		t.Errorf("expected no error for unknown app usage push, got %v", err)
	}
}

func TestIntegrationFullSession(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddApp(mockclient.AddAppRequest{ExeName: "game.exe", DailyBudgetMinutes: 120})
	if err != nil {
		t.Fatalf("AddApp failed: %v", err)
	}

	err = client.PushUsage([]mockclient.UsageReport{
		{ExeName: "game.exe", Seconds: 30},
	})
	if err != nil {
		t.Fatalf("PushUsage (30s) failed: %v", err)
	}

	err = client.PushUsage([]mockclient.UsageReport{
		{ExeName: "game.exe", Seconds: 60},
	})
	if err != nil {
		t.Fatalf("PushUsage (60s) failed: %v", err)
	}

	summaries, err := client.GetUsageToday()
	if err != nil {
		t.Fatalf("GetUsageToday failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].UsedTodayMinutes != 1 {
		t.Errorf("expected used 1 minute (90s truncated), got %d", summaries[0].UsedTodayMinutes)
	}
	if summaries[0].RemainingMinutes != 119 {
		t.Errorf("expected remaining 119 minutes, got %d", summaries[0].RemainingMinutes)
	}

	configs, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}
	if configs[0].DailyBudgetMinutes != 120 {
		t.Errorf("expected budget still 120, got %d", configs[0].DailyBudgetMinutes)
	}
}

func TestIntegrationHealthCheck(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	err := client.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}
