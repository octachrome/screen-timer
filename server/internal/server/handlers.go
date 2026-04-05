// Package server implements the Screen Timer HTTP server.
// It provides REST endpoints for two consumers:
//   - /api/        — the web-based management UI (add/edit/delete apps, view usage)
//   - /api/agent/  — the Windows agent (poll config, push usage data)
//
// Handlers are thin wrappers: validate input → call Store → write JSON response.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// writeJSON serialises v as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError sends a JSON error response: {"error": "msg"}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// EnableCORS wraps an http.Handler with CORS headers.
func EnableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// NewRouter creates and returns a configured ServeMux for the application.
func NewRouter(store *Store) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// UI endpoints
	mux.HandleFunc("GET /api/apps", handleListApps(store))
	mux.HandleFunc("POST /api/apps", handleAddApp(store))
	mux.HandleFunc("PUT /api/apps/{name}", handleUpdateApp(store))
	mux.HandleFunc("DELETE /api/apps/{name}", handleDeleteApp(store))
	mux.HandleFunc("GET /api/usage/today", handleUsageToday(store))

	// Agent endpoints
	mux.HandleFunc("GET /api/agent/config", handleAgentConfig(store))
	mux.HandleFunc("POST /api/agent/usage", handleAgentUsage(store))
	mux.HandleFunc("POST /api/agent/test-popup", handleTestPopup(store))

	// Static file serving
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	return mux
}

// handleListApps returns all tracked groups as UsageSummary objects.
func handleListApps(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groups := store.ListGroups()
		summaries := make([]UsageSummary, len(groups))
		for i := range groups {
			summaries[i] = groups[i].ToUsageSummary()
		}
		writeJSON(w, http.StatusOK, summaries)
	}
}

// handleAddApp adds a new group to track.
// Returns 201 on success, 400 for validation errors, 409 if the group already exists.
func handleAddApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name must not be empty")
			return
		}
		if len(req.Processes) == 0 {
			writeError(w, http.StatusBadRequest, "at least one process is required")
			return
		}
		if req.DailyBudgetMinutes <= 0 {
			writeError(w, http.StatusBadRequest, "daily_budget_minutes must be greater than 0")
			return
		}
		budget := time.Duration(req.DailyBudgetMinutes) * time.Minute
		group, err := store.AddGroup(req.Name, req.Processes, budget)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		summary := group.ToUsageSummary()
		writeJSON(w, http.StatusCreated, summary)
	}
}

// handleUpdateApp updates the daily budget and processes for a tracked group.
func handleUpdateApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		var req UpdateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.DailyBudgetMinutes <= 0 {
			writeError(w, http.StatusBadRequest, "daily_budget_minutes must be greater than 0")
			return
		}
		budget := time.Duration(req.DailyBudgetMinutes) * time.Minute
		newName := req.Name
		if newName == "" {
			newName = name
		}
		group, err := store.UpdateGroup(name, newName, budget, req.Processes)
		if err != nil {
			if strings.HasPrefix(err.Error(), "group already exists") {
				writeError(w, http.StatusConflict, err.Error())
			} else {
				writeError(w, http.StatusNotFound, err.Error())
			}
			return
		}
		summary := group.ToUsageSummary()
		writeJSON(w, http.StatusOK, summary)
	}
}

// handleDeleteApp removes a group from tracking.
func handleDeleteApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if err := store.DeleteGroup(name); err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleUsageToday returns today's usage summary for all tracked apps.
func handleUsageToday(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summaries := store.GetUsageSummary()
		writeJSON(w, http.StatusOK, summaries)
	}
}

// handleAgentConfig returns the group configs for the agent.
func handleAgentConfig(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groups := store.ListGroups()
		configs := make([]GroupConfig, len(groups))
		for i := range groups {
			configs[i] = groups[i].ToGroupConfig()
		}
		resp := AgentConfigResponse{Groups: configs}
		if t := store.GetTestPopupRequestedAt(); !t.IsZero() {
			resp.TestPopupAt = t.Format(time.RFC3339)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

// handleTestPopup triggers a test popup request.
func handleTestPopup(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := store.RequestTestPopup()
		writeJSON(w, http.StatusOK, map[string]string{
			"status":       "ok",
			"requested_at": t.Format(time.RFC3339),
		})
	}
}

// handleAgentUsage records usage data pushed by the agent.
// Errors for individual apps (e.g. unknown exe) are silently ignored so the
// agent doesn't fail when it reports usage for an app the manager has deleted.
func handleAgentUsage(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var push UsagePush
		if err := json.NewDecoder(r.Body).Decode(&push); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		for _, report := range push.Usage {
			// Ignore errors — the agent may report usage for apps already removed
			store.RecordUsage(report.ExeName, report.Seconds, report.TotalSeconds)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
