# Screen Timer — Issues

## UI
- [x] "Executable name" label and "e.g. Fortnite.exe" placeholder are misleading — the agent matches on process name (without .exe extension). Change label to "Process name" and placeholder to "e.g. notepad" or "e.g. Fortnite". See `server/static/index.html` lines 17-18.

## Agent
- [x] Toast notifications fire even when the app is not in the foreground. Only show notifications when the tracked app is the current foreground process. The enforcement check (line 80) already gates on `currentTrackedExe` — notifications (lines 60-74) should do the same. See `client/src/ScreenTimer.Agent.Core/Engine/AgentEngine.cs`.
- [x] Max backoff for network failures is 5 minutes — reduce to 1 minute. See `MaxBackoff` in `client/src/ScreenTimer.Agent.Host/AgentWorker.cs` line 21.

## Server
- [x] Server state is entirely in-memory — restarting the server loses all app configuration and usage data. Add file-based persistence (JSON file) so state survives restarts. See `server/internal/server/store.go`.
- [x] After a server restart, the "used today" display only shows usage accumulated since the restart, not the full day's total. The client is the source of truth for usage but only pushes deltas. Consider having the client push total usage (not just deltas) so the server can recover, or persist server-side.
- [x] Add "Send Test Popup" button to UI for testing toast visibility in fullscreen games. Implementation plan:
  - Server: add `testPopupRequestedAt time.Time` to Store. New endpoint `POST /api/agent/test-popup` sets it to `time.Now()`.
  - Change `GET /api/agent/config` response from bare array to `{"apps": [...], "test_popup_at": "..."}`.
  - Client: add `LastTestPopupTime` to AgentState (persisted). On config poll, if `test_popup_at` parses to a time after `LastTestPopupTime`, show a toast and update `LastTestPopupTime`.
  - Uses timestamps (not counters) for resilience across server/client restarts.
  - Piggybacks on existing 30-second config poll — no extra HTTP calls.
  - UI: add button in `server/static/index.html` that POSTs to `/api/agent/test-popup`.
