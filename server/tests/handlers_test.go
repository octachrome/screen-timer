package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/octachrome/screen-timer/server/internal/server"
)

func setupRouter() (*server.Store, *http.ServeMux) {
	store := server.NewStore()
	router := server.NewRouter(store)
	return store, router
}

func addGroup(t *testing.T, store *server.Store, name string, process string, budget int) {
	t.Helper()
	_, err := store.AddGroup(name, []string{process}, time.Duration(budget)*time.Minute, 0)
	if err != nil {
		t.Fatalf("failed to add group %s: %v", name, err)
	}
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	return bytes.NewReader(b)
}

func TestPostAppsValid(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.AddGroupRequest{Name: "Fortnite.exe", Processes: []string{"Fortnite.exe"}, DailyBudgetMinutes: 60})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Fortnite.exe" {
		t.Errorf("expected name Fortnite.exe, got %s", resp.Name)
	}
	if resp.DailyBudgetMinutes != 60 {
		t.Errorf("expected daily_budget_minutes 60, got %d", resp.DailyBudgetMinutes)
	}
	if resp.UsedTodayMinutes != 0 {
		t.Errorf("expected used_today_minutes 0, got %d", resp.UsedTodayMinutes)
	}
	if resp.RemainingMinutes != 60 {
		t.Errorf("expected remaining_minutes 60, got %d", resp.RemainingMinutes)
	}
}

func TestPostAppsMissingExeName(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.AddGroupRequest{Name: "", Processes: []string{}, DailyBudgetMinutes: 60})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPostAppsDuplicate(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)

	body := jsonBody(t, server.AddGroupRequest{Name: "Fortnite.exe", Processes: []string{"Fortnite.exe"}, DailyBudgetMinutes: 60})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetAppsEmpty(t *testing.T) {
	_, router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := bytes.TrimSpace(rr.Body.Bytes())
	if string(body) != "[]" {
		t.Errorf("expected empty JSON array [], got %s", string(body))
	}
}

func TestGetAppsPopulated(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)
	addGroup(t, store, "Minecraft.exe", "Minecraft.exe", 120)

	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var summaries []server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&summaries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(summaries))
	}
	if summaries[0].Name != "Fortnite.exe" {
		t.Errorf("expected first group Fortnite.exe, got %s", summaries[0].Name)
	}
	if summaries[1].Name != "Minecraft.exe" {
		t.Errorf("expected second group Minecraft.exe, got %s", summaries[1].Name)
	}
}

func TestPutAppsValid(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)

	body := jsonBody(t, server.UpdateGroupRequest{DailyBudgetMinutes: 90, Processes: []string{"Fortnite.exe"}})
	req := httptest.NewRequest(http.MethodPut, "/api/apps/Fortnite.exe", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DailyBudgetMinutes != 90 {
		t.Errorf("expected daily_budget_minutes 90, got %d", resp.DailyBudgetMinutes)
	}
}

func TestPutAppsRename(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)

	body := jsonBody(t, server.UpdateGroupRequest{Name: "Games", DailyBudgetMinutes: 60, Processes: []string{"Fortnite.exe"}})
	req := httptest.NewRequest(http.MethodPut, "/api/apps/Fortnite.exe", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Games" {
		t.Errorf("expected name Games, got %s", resp.Name)
	}

	// Old name should no longer exist
	_, err := store.GetGroup("Fortnite.exe")
	if err == nil {
		t.Error("expected old group name to be gone")
	}
}

func TestPutAppsRenameConflict(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)
	addGroup(t, store, "Games", "Minecraft.exe", 120)

	body := jsonBody(t, server.UpdateGroupRequest{Name: "Games", DailyBudgetMinutes: 60, Processes: []string{"Fortnite.exe"}})
	req := httptest.NewRequest(http.MethodPut, "/api/apps/Fortnite.exe", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPutAppsNotFound(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.UpdateGroupRequest{DailyBudgetMinutes: 90, Processes: []string{"Unknown.exe"}})
	req := httptest.NewRequest(http.MethodPut, "/api/apps/Unknown.exe", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteAppsValid(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)

	req := httptest.NewRequest(http.MethodDelete, "/api/apps/Fortnite.exe", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDeleteAppsNotFound(t *testing.T) {
	_, router := setupRouter()

	req := httptest.NewRequest(http.MethodDelete, "/api/apps/Unknown.exe", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestGetUsageToday(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)
	addGroup(t, store, "Minecraft.exe", "Minecraft.exe", 120)
	if err := store.RecordUsage("Fortnite.exe", 600, 0); err != nil {
		t.Fatalf("failed to record usage: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/usage/today", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var summaries []server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&summaries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(summaries))
	}

	// Sorted by name: Fortnite.exe first
	fortnite := summaries[0]
	if fortnite.Name != "Fortnite.exe" {
		t.Errorf("expected Fortnite.exe, got %s", fortnite.Name)
	}
	if fortnite.UsedTodayMinutes != 10 {
		t.Errorf("expected used_today_minutes 10, got %d", fortnite.UsedTodayMinutes)
	}
	if fortnite.RemainingMinutes != 50 {
		t.Errorf("expected remaining_minutes 50, got %d", fortnite.RemainingMinutes)
	}

	minecraft := summaries[1]
	if minecraft.Name != "Minecraft.exe" {
		t.Errorf("expected Minecraft.exe, got %s", minecraft.Name)
	}
	if minecraft.UsedTodayMinutes != 0 {
		t.Errorf("expected used_today_minutes 0, got %d", minecraft.UsedTodayMinutes)
	}
}

func TestGetAgentConfig(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)
	addGroup(t, store, "Minecraft.exe", "Minecraft.exe", 120)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/config", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.AgentConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Groups) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(resp.Groups))
	}
	if resp.Groups[0].Name != "Fortnite.exe" || resp.Groups[0].DailyBudgetMinutes != 60 {
		t.Errorf("unexpected first config: %+v", resp.Groups[0])
	}
	if resp.Groups[1].Name != "Minecraft.exe" || resp.Groups[1].DailyBudgetMinutes != 120 {
		t.Errorf("unexpected second config: %+v", resp.Groups[1])
	}
	if resp.TestPopupAt != "" {
		t.Errorf("expected empty test_popup_at, got %s", resp.TestPopupAt)
	}
}

func TestPostTestPopup(t *testing.T) {
	_, router := setupRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/agent/test-popup", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
	if resp["requested_at"] == "" {
		t.Error("expected non-empty requested_at")
	}
}

func TestAgentConfigIncludesTestPopupAt(t *testing.T) {
	store, router := setupRouter()

	// Request a test popup first
	store.RequestTestPopup()

	req := httptest.NewRequest(http.MethodGet, "/api/agent/config", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.AgentConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TestPopupAt == "" {
		t.Error("expected non-empty test_popup_at after requesting test popup")
	}
}

func TestPostAgentUsage(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Fortnite.exe", "Fortnite.exe", 60)

	push := server.UsagePush{
		Usage: []server.UsageReport{
			{ExeName: "Fortnite.exe", Seconds: 300},
		},
	}
	body := jsonBody(t, push)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/usage", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify store was updated
	g, err := store.GetGroup("Fortnite.exe")
	if err != nil {
		t.Fatalf("failed to get group: %v", err)
	}
	if g.UsedToday != 5*time.Minute {
		t.Errorf("expected 5m usage, got %v", g.UsedToday)
	}
}

func TestPostAgentUsageUnknownApp(t *testing.T) {
	_, router := setupRouter()

	push := server.UsagePush{
		Usage: []server.UsageReport{
			{ExeName: "Unknown.exe", Seconds: 300},
		},
	}
	body := jsonBody(t, push)
	req := httptest.NewRequest(http.MethodPost, "/api/agent/usage", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPostAppsWithWeekendBudget(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.AddGroupRequest{Name: "Game", Processes: []string{"Game"}, DailyBudgetMinutes: 60, WeekendBudgetMinutes: 30})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WeekendBudgetMinutes != 30 {
		t.Errorf("expected weekend_budget_minutes 30, got %d", resp.WeekendBudgetMinutes)
	}
}

func TestPutAppsWeekendBudget(t *testing.T) {
	store, router := setupRouter()
	addGroup(t, store, "Game", "Game", 60)

	body := jsonBody(t, server.UpdateGroupRequest{DailyBudgetMinutes: 60, WeekendBudgetMinutes: 45, Processes: []string{"Game"}})
	req := httptest.NewRequest(http.MethodPut, "/api/apps/Game", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WeekendBudgetMinutes != 45 {
		t.Errorf("expected weekend_budget_minutes 45, got %d", resp.WeekendBudgetMinutes)
	}
}

func TestPostAppsWeekendBudgetDefaults(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.AddGroupRequest{Name: "Game", Processes: []string{"Game"}, DailyBudgetMinutes: 60})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp server.UsageSummary
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.WeekendBudgetMinutes != 60 {
		t.Errorf("expected weekend_budget_minutes to default to 60, got %d", resp.WeekendBudgetMinutes)
	}
}

func TestHealthCheck(t *testing.T) {
	_, router := setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got %q", rr.Body.String())
	}
}
