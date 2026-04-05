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

func addApp(t *testing.T, store *server.Store, exe string, budgetMinutes int) {
	t.Helper()
	_, err := store.AddApp(exe, time.Duration(budgetMinutes)*time.Minute)
	if err != nil {
		t.Fatalf("failed to add app %s: %v", exe, err)
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

	body := jsonBody(t, server.AddAppRequest{ExeName: "Fortnite.exe", DailyBudgetMinutes: 60})
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
	if resp.ExeName != "Fortnite.exe" {
		t.Errorf("expected exe_name Fortnite.exe, got %s", resp.ExeName)
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

	body := jsonBody(t, server.AddAppRequest{ExeName: "", DailyBudgetMinutes: 60})
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
	addApp(t, store, "Fortnite.exe", 60)

	body := jsonBody(t, server.AddAppRequest{ExeName: "Fortnite.exe", DailyBudgetMinutes: 60})
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
	addApp(t, store, "Fortnite.exe", 60)
	addApp(t, store, "Minecraft.exe", 120)

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
		t.Fatalf("expected 2 apps, got %d", len(summaries))
	}
	if summaries[0].ExeName != "Fortnite.exe" {
		t.Errorf("expected first app Fortnite.exe, got %s", summaries[0].ExeName)
	}
	if summaries[1].ExeName != "Minecraft.exe" {
		t.Errorf("expected second app Minecraft.exe, got %s", summaries[1].ExeName)
	}
}

func TestPutAppsValid(t *testing.T) {
	store, router := setupRouter()
	addApp(t, store, "Fortnite.exe", 60)

	body := jsonBody(t, server.UpdateBudgetRequest{DailyBudgetMinutes: 90})
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

func TestPutAppsNotFound(t *testing.T) {
	_, router := setupRouter()

	body := jsonBody(t, server.UpdateBudgetRequest{DailyBudgetMinutes: 90})
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
	addApp(t, store, "Fortnite.exe", 60)

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
	addApp(t, store, "Fortnite.exe", 60)
	addApp(t, store, "Minecraft.exe", 120)
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

	// Sorted by exe_name: Fortnite.exe first
	fortnite := summaries[0]
	if fortnite.ExeName != "Fortnite.exe" {
		t.Errorf("expected Fortnite.exe, got %s", fortnite.ExeName)
	}
	if fortnite.UsedTodayMinutes != 10 {
		t.Errorf("expected used_today_minutes 10, got %d", fortnite.UsedTodayMinutes)
	}
	if fortnite.RemainingMinutes != 50 {
		t.Errorf("expected remaining_minutes 50, got %d", fortnite.RemainingMinutes)
	}

	minecraft := summaries[1]
	if minecraft.ExeName != "Minecraft.exe" {
		t.Errorf("expected Minecraft.exe, got %s", minecraft.ExeName)
	}
	if minecraft.UsedTodayMinutes != 0 {
		t.Errorf("expected used_today_minutes 0, got %d", minecraft.UsedTodayMinutes)
	}
}

func TestGetAgentConfig(t *testing.T) {
	store, router := setupRouter()
	addApp(t, store, "Fortnite.exe", 60)
	addApp(t, store, "Minecraft.exe", 120)

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
	if len(resp.Apps) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(resp.Apps))
	}
	if resp.Apps[0].ExeName != "Fortnite.exe" || resp.Apps[0].DailyBudgetMinutes != 60 {
		t.Errorf("unexpected first config: %+v", resp.Apps[0])
	}
	if resp.Apps[1].ExeName != "Minecraft.exe" || resp.Apps[1].DailyBudgetMinutes != 120 {
		t.Errorf("unexpected second config: %+v", resp.Apps[1])
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
	addApp(t, store, "Fortnite.exe", 60)

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
	app, err := store.GetApp("Fortnite.exe")
	if err != nil {
		t.Fatalf("failed to get app: %v", err)
	}
	if app.UsedToday != 5*time.Minute {
		t.Errorf("expected 5m usage, got %v", app.UsedToday)
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
