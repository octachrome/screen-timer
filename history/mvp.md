# Screen Timer — MVP

The minimum subset to prove the end-to-end concept: a Windows agent that tracks and enforces time limits, controlled by a remote web UI.

## In Scope

### Windows Agent (C#)

- Track time spent in configured applications (by executable name), **only while in focus**.
- A single daily budget per application (no weekday/weekend distinction).
- **Toast notifications** at 10, 5, and 1 minute remaining (using standard Windows toast notifications — no overlay, no full-screen support).
- **Force-close** the application when the budget is exhausted (simplest viable enforcement).
- Poll the backend on a regular interval to fetch configuration and push usage data.
- Runs as a normal background process (no auto-start, no tamper resistance).

### Web Backend (Go)

- REST API for:
  - CRUD tracked applications and their daily budget.
  - Receiving usage data from the agent.
  - Serving today's usage summary.
- Serve the web frontend as static files.
- In-memory or simple file-based storage (no database).

### Web Frontend (TypeScript + reactive framework + minimal CSS)

- Add/remove tracked applications by executable name.
- Set a single daily budget per application.
- View today's usage per application (time used vs. time remaining).

## Out of Scope (deferred to later iterations)

- Separate weekday/weekend budgets
- Ad-hoc daily budget extensions
- Historical usage data / CSV export
- Persistent on-screen overlay
- Full-screen-aware notifications
- Non-focus-stealing notification behaviour
- Auto-start on boot
- Sophisticated enforcement (anything beyond force-close)
