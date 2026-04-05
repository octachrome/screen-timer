# Screen Timer — Server

REST API backend for the Screen Timer application. Manages tracked applications, daily time budgets, and usage data reported by the Windows agent. Serves the web management UI as static files.

## Project Structure

```
server/
├── cmd/server/main.go             # Entrypoint: wires router, starts HTTP server
├── internal/server/                # Production code (not importable outside this module)
│   ├── model.go                    # Data types (Application, AppConfig, UsageSummary, etc.)
│   ├── store.go                    # Persistent store (JSON file-backed, mutex-protected)
│   └── handlers.go                 # HTTP handlers and router setup
├── mockclient/client.go            # Mock client simulating the Windows agent's HTTP calls
├── tests/                          # All tests (unit, handler, integration)
│   ├── store_test.go               # Store unit tests (including persistence round-trip)
│   ├── handlers_test.go            # HTTP handler tests
│   └── integration_test.go         # End-to-end tests using mock client
├── static/                         # Frontend assets (served at /)
│   ├── index.html
│   ├── app.js
│   └── style.css
├── data/                           # Created at runtime — persistent state (git-ignored)
│   └── screen-timer.json
└── go.mod
```

## Prerequisites

- Go 1.22 or later (uses `http.ServeMux` method-based routing)

## Building

```sh
cd server
go build -o screen-timer-server ./cmd/server
```

This produces a `screen-timer-server` binary (or `screen-timer-server.exe` on Windows). No external dependencies — the Go standard library provides everything.

## Running the Server

From source:

```sh
cd server
go run ./cmd/server
```

Or run the compiled binary:

```sh
./screen-timer-server
```

The server starts on port **8080** by default. Override with environment variables:

| Variable    | Default                    | Description                              |
|-------------|----------------------------|------------------------------------------|
| `PORT`      | `8080`                     | HTTP listen port                         |
| `DATA_FILE` | `data/screen-timer.json`   | Path to the persistent state JSON file   |

Example:

```sh
PORT=3000 DATA_FILE=/var/data/screen-timer.json go run ./cmd/server
```

### Persistence

The server persists all application configuration and usage data to a JSON file (default: `data/screen-timer.json` relative to the working directory). The file is written atomically (write to `.tmp`, then rename) after every mutation. On startup, the server loads existing state from this file if it exists; otherwise it starts empty.

## API Endpoints

### Manager (Web UI)

| Method   | Path                     | Purpose                    |
|----------|--------------------------|----------------------------|
| `GET`    | `/api/apps`              | List all tracked apps      |
| `POST`   | `/api/apps`              | Add a tracked app          |
| `PUT`    | `/api/apps/{exe}`        | Update an app's budget     |
| `DELETE` | `/api/apps/{exe}`        | Remove a tracked app       |
| `GET`    | `/api/usage/today`       | Today's usage summary      |

### Agent (Windows client)

| Method | Path                     | Purpose                                              |
|--------|--------------------------|------------------------------------------------------|
| `GET`  | `/api/agent/config`      | Poll app configs, budgets, and test-popup timestamp   |
| `POST` | `/api/agent/usage`       | Push accumulated usage data (delta + total)           |
| `POST` | `/api/agent/test-popup`  | Request a test toast notification on the agent        |

### Other

| Method | Path       | Purpose      |
|--------|------------|--------------|
| `GET`  | `/healthz` | Health check |

## Running Tests

```sh
cd server
go test ./...
```

Verbose output:

```sh
go test ./... -v
```
