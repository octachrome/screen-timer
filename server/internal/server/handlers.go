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
	mux.HandleFunc("PUT /api/apps/{exe}", handleUpdateApp(store))
	mux.HandleFunc("DELETE /api/apps/{exe}", handleDeleteApp(store))
	mux.HandleFunc("GET /api/usage/today", handleUsageToday(store))

	// Agent endpoints
	mux.HandleFunc("GET /api/agent/config", handleAgentConfig(store))
	mux.HandleFunc("POST /api/agent/usage", handleAgentUsage(store))

	// Static file serving
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	return mux
}

// handleListApps returns all tracked apps as UsageSummary objects.
func handleListApps(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps := store.ListApps()
		summaries := make([]UsageSummary, len(apps))
		for i := range apps {
			summaries[i] = apps[i].ToUsageSummary()
		}
		writeJSON(w, http.StatusOK, summaries)
	}
}

// handleAddApp adds a new application to track.
// Returns 201 on success, 400 for validation errors, 409 if the app already exists.
func handleAddApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ExeName == "" {
			writeError(w, http.StatusBadRequest, "exe_name must not be empty")
			return
		}
		if req.DailyBudgetMinutes <= 0 {
			writeError(w, http.StatusBadRequest, "daily_budget_minutes must be greater than 0")
			return
		}
		app, err := store.AddApp(req.ExeName, time.Duration(req.DailyBudgetMinutes)*time.Minute)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		summary := app.ToUsageSummary()
		writeJSON(w, http.StatusCreated, summary)
	}
}

// handleUpdateApp updates the daily budget for a tracked application.
func handleUpdateApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exe := r.PathValue("exe")
		var req UpdateBudgetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.DailyBudgetMinutes <= 0 {
			writeError(w, http.StatusBadRequest, "daily_budget_minutes must be greater than 0")
			return
		}
		app, err := store.UpdateBudget(exe, time.Duration(req.DailyBudgetMinutes)*time.Minute)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		summary := app.ToUsageSummary()
		writeJSON(w, http.StatusOK, summary)
	}
}

// handleDeleteApp removes an application from tracking.
func handleDeleteApp(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exe := r.PathValue("exe")
		if err := store.DeleteApp(exe); err != nil {
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

// handleAgentConfig returns the app configs for the agent.
func handleAgentConfig(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apps := store.ListApps()
		configs := make([]AppConfig, len(apps))
		for i := range apps {
			configs[i] = apps[i].ToAppConfig()
		}
		writeJSON(w, http.StatusOK, configs)
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
			store.RecordUsage(report.ExeName, report.Seconds)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
