# Screen Timer — Server

REST API backend for the Screen Timer application. Manages tracked applications, daily time budgets, and usage data reported by the Windows agent.

## Project Structure

```
server/
├── cmd/server/main.go             # Entrypoint: wires router, starts HTTP server
├── internal/server/                # Production code (not importable outside this module)
│   ├── model.go                    # Data types (Application, AppConfig, UsageSummary, etc.)
│   ├── store.go                    # In-memory store with mutex-protected map
│   └── handlers.go                 # HTTP handlers and router setup
├── mockclient/client.go            # Mock client simulating the Windows agent's HTTP calls
├── tests/                          # All tests (unit, handler, integration)
│   ├── store_test.go               # Store unit tests
│   ├── handlers_test.go            # HTTP handler tests
│   └── integration_test.go         # End-to-end tests using mock client
├── static/                         # Frontend assets (served at /)
└── go.mod
```

## Prerequisites

- Go 1.22 or later (uses `http.ServeMux` method-based routing)

## Running the Server

```sh
cd server
go run ./cmd/server
```

The server starts on port **8080** by default. Set the `PORT` environment variable to use a different port:

```sh
PORT=3000 go run ./cmd/server
```

## API Endpoints

### Manager (Web UI)

| Method   | Path                | Purpose                    |
|----------|---------------------|----------------------------|
| `GET`    | `/api/apps`         | List all tracked apps      |
| `POST`   | `/api/apps`         | Add a tracked app          |
| `PUT`    | `/api/apps/{exe}`   | Update an app's budget     |
| `DELETE` | `/api/apps/{exe}`   | Remove a tracked app       |
| `GET`    | `/api/usage/today`  | Today's usage summary      |

### Agent (Windows client)

| Method | Path               | Purpose                       |
|--------|---------------------|-------------------------------|
| `GET`  | `/api/agent/config` | Poll app configs and budgets  |
| `POST` | `/api/agent/usage`  | Push accumulated usage data   |

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
