# Screen Timer Agent

A Windows background agent that tracks time spent in configured applications (by executable name) while they are in the foreground. It shows toast notifications as daily time budgets run low (at 10, 5, and 1 minute remaining) and force-closes applications when the budget is exhausted. The agent polls a Go backend server for configuration and pushes usage data back to it.

## Prerequisites

- [.NET 9.0 SDK](https://dotnet.microsoft.com/download/dotnet/9.0)
- Windows 10 or later
- The [Go server](../server/) running at the configured URL

## Building

```
cd client
dotnet publish src/ScreenTimer.Agent.Host -o build
```

This publishes the agent and all its dependencies — including `appsettings.json` — into the `build/` folder. The entire folder is self-contained and can be copied to another machine without the source tree. To run the built agent:

```
build\ScreenTimer.Agent.Host.exe
```

The agent loads `appsettings.json` from the same directory as the executable, so edit the copy in `build/` (or on the target machine) to change configuration.

## Running Tests

```
cd client
dotnet test
```

This runs unit tests in `tests/ScreenTimer.Agent.Core.Tests` and adapter/contract tests in `tests/ScreenTimer.Agent.Windows.Tests`.

## Running the Agent

Make sure the Go server is running first, then:

```
cd client
dotnet run --project src/ScreenTimer.Agent.Host
```

To override the server URL on the command line:

```
dotnet run --project src/ScreenTimer.Agent.Host -- --ServerUrl http://192.168.1.50:8080
```

## Configuration

Configuration lives in `appsettings.json` next to the executable (source default: `src/ScreenTimer.Agent.Host/appsettings.json`):

| Setting     | Default                  | Description                          |
|-------------|--------------------------|--------------------------------------|
| `ServerUrl` | `http://localhost:8080`  | Base URL of the Go backend server    |

You can also set `ServerUrl` via command-line argument (shown above) or environment variable (`ServerUrl=http://...`).

The agent uses the following fixed intervals:

| Interval            | Value | Description                                     |
|---------------------|-------|-------------------------------------------------|
| Tick interval       | 1 s   | How often the engine samples the foreground app  |
| Config poll         | 30 s  | How often app rules are fetched from the server  |
| Usage push          | 15 s  | How often accumulated usage is pushed to the server |

Config poll and usage push use exponential backoff (up to 1 minute) on repeated failures.

## How It Works

The agent runs as a .NET `BackgroundService` with a tick-driven architecture:

1. **Every second**, the worker samples the foreground window to determine which executable is active.
2. The pure `AgentEngine.Tick()` method (in `ScreenTimer.Agent.Core`) receives the current state and foreground sample, then returns an updated state plus a list of commands (show toast, push usage, force-close, persist state).
3. The `AgentWorker` dispatches each command to the appropriate Windows adapter—Win32 APIs for foreground detection and process control, `Microsoft.Toolkit.Uwp.Notifications` for toast notifications, and an HTTP client for the server API.
4. State is persisted to `%LocalAppData%\ScreenTimer\agent-state.json` so usage tracking survives restarts.

This separation keeps all business logic in the `Core` project with no OS or network dependencies, making it fully unit-testable.

## Solution Structure

| Project | Description |
|---------|-------------|
| `src/ScreenTimer.Agent.Core` | Pure business logic — engine, models, DTOs, interfaces |
| `src/ScreenTimer.Agent.Windows` | Windows adapters — Win32 foreground probe, toast notifications, process controller, HTTP client, JSON state store |
| `src/ScreenTimer.Agent.Host` | Runnable `BackgroundService` host |
| `src/ScreenTimer.FullscreenHarness` | Manual testing utility for fullscreen scenarios |
| `tests/ScreenTimer.Agent.Core.Tests` | Core engine unit tests |
| `tests/ScreenTimer.Agent.Windows.Tests` | Adapter and contract tests |
| `tests/ScreenTimer.Agent.IntegrationTests` | Integration tests for the worker loop |

## Packaging for Deployment

To publish a self-contained single-file executable (no .NET runtime required on the target machine):

```
cd client
dotnet publish src/ScreenTimer.Agent.Host -c Release -r win-x64 --self-contained
```

The output will be in `src/ScreenTimer.Agent.Host/bin/Release/net9.0-windows10.0.19041.0/win-x64/publish/`.

## Testing with the Fullscreen Harness

The `ScreenTimer.FullscreenHarness` is a small WinForms app for manually testing toast visibility and force-close behavior in different window modes:

```
dotnet run --project src/ScreenTimer.FullscreenHarness -- --mode borderless
dotnet run --project src/ScreenTimer.FullscreenHarness -- --mode exclusive --resist-close 3
```

Modes: `windowed` (default), `borderless`, `exclusive`. Use `--resist-close <N>` to ignore the first N close requests (tests hard-kill fallback).
