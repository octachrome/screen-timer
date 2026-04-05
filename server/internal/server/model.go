package server

import "time"

// Group represents a tracked group of processes with its budget and today's usage.
type Group struct {
	Name          string
	Processes     []string
	DailyBudget   time.Duration
	UsedToday     time.Duration
	LastResetDate string
}

// GroupConfig is the agent-facing view of a group's configuration.
type GroupConfig struct {
	Name               string   `json:"name"`
	Processes          []string `json:"processes"`
	DailyBudgetMinutes int      `json:"daily_budget_minutes"`
}

// AgentConfigResponse is the wrapper object returned by GET /api/agent/config.
type AgentConfigResponse struct {
	Groups      []GroupConfig `json:"groups"`
	TestPopupAt string        `json:"test_popup_at,omitempty"`
}

// UsageSummary is the UI-facing view of today's usage for a group.
type UsageSummary struct {
	Name               string   `json:"name"`
	Processes          []string `json:"processes"`
	DailyBudgetMinutes int      `json:"daily_budget_minutes"`
	UsedTodayMinutes   int      `json:"used_today_minutes"`
	RemainingMinutes   int      `json:"remaining_minutes"`
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

// AddGroupRequest is the request body for POST /api/groups.
type AddGroupRequest struct {
	Name               string `json:"name"`
	Process            string `json:"process"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
}

// UpdateGroupRequest is the request body for PUT /api/groups/{name}.
type UpdateGroupRequest struct {
	DailyBudgetMinutes int      `json:"daily_budget_minutes"`
	Processes          []string `json:"processes"`
}

// ToUsageSummary converts a Group to a UsageSummary for the UI.
func (g *Group) ToUsageSummary() UsageSummary {
	budget := int(g.DailyBudget.Minutes())
	used := int(g.UsedToday.Minutes())
	remaining := budget - used
	if remaining < 0 {
		remaining = 0
	}
	return UsageSummary{
		Name:               g.Name,
		Processes:          g.Processes,
		DailyBudgetMinutes: budget,
		UsedTodayMinutes:   used,
		RemainingMinutes:   remaining,
	}
}

// ToGroupConfig converts a Group to a GroupConfig for the agent.
func (g *Group) ToGroupConfig() GroupConfig {
	return GroupConfig{
		Name:               g.Name,
		Processes:          g.Processes,
		DailyBudgetMinutes: int(g.DailyBudget.Minutes()),
	}
}
