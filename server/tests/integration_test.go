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

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "chrome.exe", Processes: []string{"chrome.exe"}, DailyBudgetMinutes: 60})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	configResp, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configResp.Groups) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configResp.Groups))
	}
	if configResp.Groups[0].Name != "chrome.exe" {
		t.Errorf("expected name chrome.exe, got %s", configResp.Groups[0].Name)
	}
	if configResp.Groups[0].DailyBudgetMinutes != 60 {
		t.Errorf("expected budget 60, got %d", configResp.Groups[0].DailyBudgetMinutes)
	}
}

func TestIntegrationAgentPushesUsageThenManagerViews(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "firefox.exe", Processes: []string{"firefox.exe"}, DailyBudgetMinutes: 120})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
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

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "slack.exe", Processes: []string{"slack.exe"}, DailyBudgetMinutes: 30})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	_, err = client.UpdateGroup("slack.exe", mockclient.UpdateGroupRequest{DailyBudgetMinutes: 90, Processes: []string{"slack.exe"}})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	configResp, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configResp.Groups) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configResp.Groups))
	}
	if configResp.Groups[0].DailyBudgetMinutes != 90 {
		t.Errorf("expected budget 90, got %d", configResp.Groups[0].DailyBudgetMinutes)
	}
}

func TestIntegrationDeleteAppThenAgentPolls(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "discord.exe", Processes: []string{"discord.exe"}, DailyBudgetMinutes: 45})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	err = client.DeleteGroup("discord.exe")
	if err != nil {
		t.Fatalf("DeleteGroup failed: %v", err)
	}

	configResp, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configResp.Groups) != 0 {
		t.Errorf("expected 0 configs after delete, got %d", len(configResp.Groups))
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

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "game.exe", Processes: []string{"game.exe"}, DailyBudgetMinutes: 120})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
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

	configResp, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configResp.Groups) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configResp.Groups))
	}
	if configResp.Groups[0].DailyBudgetMinutes != 120 {
		t.Errorf("expected budget still 120, got %d", configResp.Groups[0].DailyBudgetMinutes)
	}
}

func TestIntegrationAddGroupResponseHasProcesses(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	summary, err := client.AddGroup(mockclient.AddGroupRequest{Name: "Fortnite", Processes: []string{"Fortnite"}, DailyBudgetMinutes: 60})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}
	if summary.Name != "Fortnite" {
		t.Errorf("expected name Fortnite, got %s", summary.Name)
	}
	if len(summary.Processes) != 1 || summary.Processes[0] != "Fortnite" {
		t.Errorf("expected processes [Fortnite], got %v", summary.Processes)
	}
}

func TestIntegrationUpdateGroupProcesses(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "Games", Processes: []string{"Fortnite"}, DailyBudgetMinutes: 60})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	summary, err := client.UpdateGroup("Games", mockclient.UpdateGroupRequest{DailyBudgetMinutes: 120, Processes: []string{"Fortnite", "Minecraft"}})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}
	if len(summary.Processes) != 2 {
		t.Fatalf("expected 2 processes, got %d", len(summary.Processes))
	}
	processSet := map[string]bool{}
	for _, p := range summary.Processes {
		processSet[p] = true
	}
	if !processSet["Fortnite"] || !processSet["Minecraft"] {
		t.Errorf("expected processes [Fortnite, Minecraft], got %v", summary.Processes)
	}

	summaries, err := client.GetUsageToday()
	if err != nil {
		t.Fatalf("GetUsageToday failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	processSet = map[string]bool{}
	for _, p := range summaries[0].Processes {
		processSet[p] = true
	}
	if !processSet["Fortnite"] || !processSet["Minecraft"] {
		t.Errorf("expected processes [Fortnite, Minecraft] in usage, got %v", summaries[0].Processes)
	}
}

func TestIntegrationGroupProcessesUsageRoundTrip(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "Games", Processes: []string{"Fortnite"}, DailyBudgetMinutes: 60})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	_, err = client.UpdateGroup("Games", mockclient.UpdateGroupRequest{DailyBudgetMinutes: 120, Processes: []string{"Fortnite", "Minecraft"}})
	if err != nil {
		t.Fatalf("UpdateGroup failed: %v", err)
	}

	err = client.PushUsage([]mockclient.UsageReport{
		{ExeName: "Fortnite", Seconds: 300},
		{ExeName: "Minecraft", Seconds: 600},
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
	if summaries[0].UsedTodayMinutes != 15 {
		t.Errorf("expected used 15 minutes, got %d", summaries[0].UsedTodayMinutes)
	}
}

func TestIntegrationWeekendBudgetConfigRoundTrip(t *testing.T) {
	ts, client := setupTestServer()
	defer ts.Close()

	_, err := client.AddGroup(mockclient.AddGroupRequest{Name: "game.exe", Processes: []string{"game.exe"}, DailyBudgetMinutes: 60, WeekendBudgetMinutes: 45})
	if err != nil {
		t.Fatalf("AddGroup failed: %v", err)
	}

	configResp, err := client.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if len(configResp.Groups) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configResp.Groups))
	}
	if configResp.Groups[0].WeekendBudgetMinutes != 45 {
		t.Errorf("expected weekend_budget_minutes 45, got %d", configResp.Groups[0].WeekendBudgetMinutes)
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
