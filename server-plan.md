# Server-Side MVP — Development Plan

## Technology Choices

- **Language:** Go
- **Storage:** In-memory with a sync.RWMutex (no database, no file persistence for MVP)
- **HTTP:** Standard library `net/http` + a lightweight router (e.g., `chi` or just `http.ServeMux` from Go 1.22+)
- **Testing:** `testing` package + `net/http/httptest`
- **Mock client:** A small Go package (`mockclient`) that wraps `net/http.Client` and speaks the same REST protocol the C# agent will, used exclusively in tests

---

## Data Model

```go
// Application represents a tracked application with its budget and today's usage.
type Application struct {
    ExeName       string        // e.g. "Fortnite.exe" — unique identifier
    DailyBudget   time.Duration // e.g. 2h
    UsedToday     time.Duration // accumulated usage pushed by agent
    LastResetDate string        // "2006-01-02"; usage resets when date changes
}
```

Storage is a simple `map[string]*Application` keyed by `ExeName`, guarded by a mutex.

---

## REST API

Two prefixes separate the two consumers:

- **`/api/`** — called by the web frontend (manager)
- **`/api/agent/`** — called by the Windows agent (C# client)

### UI endpoints (`/api/`)

| Method | Path | Purpose | Request Body | Response |
|--------|------|---------|-------------|----------|
| `GET` | `/api/apps` | List all tracked apps with budget & usage | — | `[]Application` |
| `POST` | `/api/apps` | Add a tracked app | `{ "exe_name": "X", "daily_budget_minutes": 120 }` | created `Application` |
| `PUT` | `/api/apps/{exe}` | Update an app's budget | `{ "daily_budget_minutes": 90 }` | updated `Application` |
| `DELETE` | `/api/apps/{exe}` | Remove a tracked app | — | 204 |
| `GET` | `/api/usage/today` | Today's usage summary | — | `[]UsageSummary` |

### Agent endpoints (`/api/agent/`)

| Method | Path | Purpose | Request Body | Response |
|--------|------|---------|-------------|----------|
| `GET` | `/api/agent/config` | Poll: get all apps + budgets | — | `[]AppConfig` |
| `POST` | `/api/agent/usage` | Push accumulated usage | `{ "usage": [{ "exe_name": "X", "seconds": 30 }] }` | 200 |

This separation lets us evolve, rate-limit, or authenticate the two surfaces independently in the future.

---

## Mock Client (`mockclient` package)

A Go package that simulates the C# Windows agent's HTTP behaviour:

```go
type MockClient struct {
    BaseURL    string
    HTTPClient *http.Client
}

func (c *MockClient) GetConfig() ([]AppConfig, error)
func (c *MockClient) PushUsage(usage []UsageReport) error
```

This client:
- Calls `GET /api/config` and deserialises the response.
- Calls `POST /api/usage` with a JSON body identical to what the real C# agent will send.
- Is used in integration-style tests where we spin up `httptest.NewServer`, point the mock client at it, and exercise full request/response cycles.

---

## Implementation Phases

### Phase 1 — Project skeleton & storage layer

1. `go mod init` with module path.
2. Define data model types in `model.go`.
3. Implement `Store` in `store.go`:
   - `AddApp(exeName string, budget time.Duration) (*Application, error)`
   - `GetApp(exeName string) (*Application, error)`
   - `ListApps() []*Application`
   - `UpdateBudget(exeName string, budget time.Duration) (*Application, error)`
   - `DeleteApp(exeName string) error`
   - `RecordUsage(exeName string, seconds int) error` — adds to `UsedToday`, resets if date changed
   - `GetUsageSummary() []UsageSummary`

**Tests (`store_test.go`):**
- Add an app, verify it appears in ListApps.
- Add duplicate app → error.
- Update budget → verify new value.
- Delete app → verify removed.
- Record usage → verify UsedToday increments.
- Record usage on a new day → verify UsedToday resets.
- Record usage for unknown app → error.

### Phase 2 — HTTP handlers

Implement handlers in `handlers.go`, each a thin wrapper that validates input, calls the store, and writes JSON.

**Tests (`handlers_test.go`):**

Each handler tested via `httptest.NewRecorder`:

- `POST /api/apps` — valid payload → 201 + correct body.
- `POST /api/apps` — missing exe_name → 400.
- `POST /api/apps` — duplicate → 409.
- `GET /api/apps` — returns list (empty, then populated).
- `PUT /api/apps/{exe}` — updates budget, returns updated app.
- `PUT /api/apps/{exe}` — unknown app → 404.
- `DELETE /api/apps/{exe}` — success → 204.
- `DELETE /api/apps/{exe}` — unknown → 404.
- `GET /api/usage/today` — returns summary.
- `GET /api/agent/config` — returns config for agent.
- `POST /api/agent/usage` — valid usage report → 200, verify store updated.
- `POST /api/agent/usage` — unknown app in report → still 200 (ignore unknown; agent may be out of sync).

### Phase 3 — Mock client & integration tests

Implement `mockclient/client.go`.

**Tests (`integration_test.go`):**

Full round-trip tests using `httptest.NewServer` + mock client:

- Manager adds an app via API → agent polls config → app appears.
- Agent pushes usage → manager views usage summary → values match.
- Manager updates budget → agent polls config → new budget reflected.
- Manager deletes app → agent polls config → app gone.
- Agent pushes usage for unknown app → no error (graceful handling).
- Simulated session: add app, push usage multiple times, check accumulation, verify budget-remaining calculation.

### Phase 4 — Server wiring & static file serving

1. Wire routes in `main.go`.
2. Serve static files from an `./static` directory for the frontend (just the wiring; frontend itself is out of scope for this plan).
3. Add a health-check endpoint (`GET /healthz`).

**Tests:**
- Health check returns 200.
- Request to `/` serves static files (test with a dummy file).

---

## File Layout

```
server/
├── main.go              # Entrypoint: wires router, starts server
├── model.go             # Data types
├── store.go             # In-memory store
├── store_test.go        # Store unit tests
├── handlers.go          # HTTP handler functions
├── handlers_test.go     # Handler unit tests
├── integration_test.go  # End-to-end tests using mock client
├── mockclient/
│   └── client.go        # Mock C# agent client
├── static/              # (placeholder for frontend assets)
└── go.mod
```

---

## Test Summary

| Layer | File | # Tests (approx) |
|-------|------|-------------------|
| Store | `store_test.go` | 7 |
| Handlers | `handlers_test.go` | 12 |
| Integration | `integration_test.go` | 6 |
| Server wiring | `handlers_test.go` | 2 |
| **Total** | | **~27** |

Every piece of business logic and every API endpoint will have dedicated test coverage. The mock client ensures the agent-facing contract is verified end-to-end.
