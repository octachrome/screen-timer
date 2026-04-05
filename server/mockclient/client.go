package mockclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// AppConfig mirrors the server's AppConfig type.
type AppConfig struct {
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
}

// AgentConfigResponse mirrors the server's AgentConfigResponse type.
type AgentConfigResponse struct {
	Apps        []AppConfig `json:"apps"`
	TestPopupAt string      `json:"test_popup_at,omitempty"`
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
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
	UsedTodayMinutes   int    `json:"used_today_minutes"`
	RemainingMinutes   int    `json:"remaining_minutes"`
}

// AddAppRequest mirrors the server's AddAppRequest.
type AddAppRequest struct {
	ExeName            string `json:"exe_name"`
	DailyBudgetMinutes int    `json:"daily_budget_minutes"`
}

// UpdateBudgetRequest mirrors the server's UpdateBudgetRequest.
type UpdateBudgetRequest struct {
	DailyBudgetMinutes int `json:"daily_budget_minutes"`
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

// AddApp creates a new tracked application (POST /api/apps).
func (c *Client) AddApp(req AddAppRequest) (*UsageSummary, error) {
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

// ListApps lists all tracked applications (GET /api/apps).
func (c *Client) ListApps() ([]UsageSummary, error) {
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

// UpdateApp updates the budget for a tracked application (PUT /api/apps/{exeName}).
func (c *Client) UpdateApp(exeName string, req UpdateBudgetRequest) (*UsageSummary, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPut, c.BaseURL+"/api/apps/"+exeName, bytes.NewReader(body))
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

// DeleteApp removes a tracked application (DELETE /api/apps/{exeName}).
func (c *Client) DeleteApp(exeName string) error {
	req, err := http.NewRequest(http.MethodDelete, c.BaseURL+"/api/apps/"+exeName, nil)
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
