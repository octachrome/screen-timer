package server

import "time"

// Application represents a tracked application with its budget and today's usage.
type Application struct {
	ExeName       string        `json:"exe_name"`
	DailyBudget   time.Duration `json:"daily_budget"`
	UsedToday     time.Duration `json:"used_today"`
	LastResetDate string        `json:"-"` // internal, not exposed via JSON
}

// AppConfig is the agent-facing view of an application's configuration.
type AppConfig struct {
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
}

// AgentConfigResponse is the wrapper object returned by GET /api/agent/config.
type AgentConfigResponse struct {
	Apps        []AppConfig `json:"apps"`
	TestPopupAt string      `json:"test_popup_at,omitempty"`
}

// UsageSummary is the UI-facing view of today's usage for an application.
type UsageSummary struct {
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
	UsedTodayMinutes   int    `json:"used_today_minutes"`
	RemainingMinutes   int    `json:"remaining_minutes"`
}

// UsageReport is a single entry in the agent's usage push.
type UsageReport struct {
	ExeName      string `json:"exe_name"`
	Seconds      int    `json:"seconds"`
	TotalSeconds int    `json:"total_seconds"`
}

// UsagePush is the request body for POST /api/agent/usage.
type UsagePush struct {
	Usage []UsageReport `json:"usage"`
}

// AddAppRequest is the request body for POST /api/apps.
type AddAppRequest struct {
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
}

// UpdateBudgetRequest is the request body for PUT /api/apps/{exe}.
type UpdateBudgetRequest struct {
	DailyBudgetMinutes int `json:"daily_budget_minutes"`
}

// ToUsageSummary converts an Application to a UsageSummary for the UI.
func (a *Application) ToUsageSummary() UsageSummary {
	budget := int(a.DailyBudget.Minutes())
	used := int(a.UsedToday.Minutes())
	remaining := budget - used
	if remaining < 0 {
		remaining = 0
	}
	return UsageSummary{
		ExeName:            a.ExeName,
		DailyBudgetMinutes: budget,
		UsedTodayMinutes:   used,
		RemainingMinutes:   remaining,
	}
}

// ToAppConfig converts an Application to an AppConfig for the agent.
func (a *Application) ToAppConfig() AppConfig {
	return AppConfig{
		ExeName:            a.ExeName,
		DailyBudgetMinutes: int(a.DailyBudget.Minutes()),
	}
}
