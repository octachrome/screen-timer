package mockclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// GroupConfig mirrors the server's GroupConfig type.
type GroupConfig struct {
	Name               string   `json:"name"`
	Processes          []string `json:"processes"`
	DailyBudgetMinutes   int      `json:"daily_budget_minutes"`
	WeekendBudgetMinutes int      `json:"weekend_budget_minutes"`
}

// AgentConfigResponse mirrors the server's AgentConfigResponse type.
type AgentConfigResponse struct {
	Groups      []GroupConfig `json:"groups"`
	TestPopupAt string        `json:"test_popup_at,omitempty"`
}

// UsageReport mirrors the server's UsageReport type.
type UsageReport struct {
	ExeName string `json:"exe_name"`
	Seconds int    `json:"seconds"`
}

type usagePush struct {
	Usage []UsageReport `json:"usage"`
}

// UsageSummary mirrors the server's UsageSummary type.
type UsageSummary struct {
	Name               string   `json:"name"`
	Processes          []string `json:"processes"`
	DailyBudgetMinutes   int      `json:"daily_budget_minutes"`
	WeekendBudgetMinutes int      `json:"weekend_budget_minutes"`
	UsedTodayMinutes     int      `json:"used_today_minutes"`
	RemainingMinutes   int      `json:"remaining_minutes"`
}

// AddGroupRequest mirrors the server's AddGroupRequest.
type AddGroupRequest struct {
	Name               string   `json:"name"`
	Processes          []string `json:"processes"`
	DailyBudgetMinutes   int      `json:"daily_budget_minutes"`
	WeekendBudgetMinutes int      `json:"weekend_budget_minutes"`
}

// UpdateGroupRequest mirrors the server's UpdateGroupRequest.
type UpdateGroupRequest struct {
	Name                 string   `json:"name"`
	DailyBudgetMinutes   int      `json:"daily_budget_minutes"`
	WeekendBudgetMinutes int      `json:"weekend_budget_minutes"`
	Processes            []string `json:"processes"`
}

// Client is a mock HTTP client that speaks the same REST protocol as the C# agent.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new mock client pointed at the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: http.DefaultClient,
	}
}

// GetConfig fetches the agent configuration (GET /api/agent/config).
func (c *Client) GetConfig() (*AgentConfigResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/agent/config")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var configResp AgentConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return nil, err
	}
	return &configResp, nil
}

// RequestTestPopup triggers a test popup (POST /api/agent/test-popup).
func (c *Client) RequestTestPopup() error {
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/agent/test-popup", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// PushUsage sends a usage report to the server (POST /api/agent/usage).
func (c *Client) PushUsage(usage []UsageReport) error {
	body, err := json.Marshal(usagePush{Usage: usage})
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/agent/usage", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// AddGroup creates a new tracked group (POST /api/apps).
func (c *Client) AddGroup(req AddGroupRequest) (*UsageSummary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Post(c.BaseURL+"/api/apps", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var summary UsageSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// ListGroups lists all tracked groups (GET /api/apps).
func (c *Client) ListGroups() ([]UsageSummary, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/apps")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var summaries []UsageSummary
	if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

// UpdateGroup updates a tracked group (PUT /api/apps/{name}).
func (c *Client) UpdateGroup(name string, req UpdateGroupRequest) (*UsageSummary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPut, c.BaseURL+"/api/apps/"+name, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var summary UsageSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, err
	}
	return &summary, nil
}

// DeleteGroup removes a tracked group (DELETE /api/apps/{name}).
func (c *Client) DeleteGroup(name string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/api/apps/"+name, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

// GetUsageToday fetches today's usage for all tracked apps (GET /api/usage/today).
func (c *Client) GetUsageToday() ([]UsageSummary, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/usage/today")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var summaries []UsageSummary
	if err := json.NewDecoder(resp.Body).Decode(&summaries); err != nil {
		return nil, err
	}
	return summaries, nil
}

// HealthCheck pings the server's health endpoint (GET /healthz).
func (c *Client) HealthCheck() error {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
