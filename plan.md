# Plan: Convert Console App to Windowless Tray App with File Logging

## Goal

Convert `ScreenTimer.Agent.Host` from a console application to a windowless Windows app with a system tray icon, and redirect all logging from the console to a log file.

## Changes

### 1. Change project output type to Windows app

**File:** `client/src/ScreenTimer.Agent.Host/ScreenTimer.Agent.Host.csproj`

- Add `<OutputType>WinExe</OutputType>` to suppress the console window.
- Add `<UseWindowsForms>true</UseWindowsForms>` to enable Windows Forms (needed for `NotifyIcon` and `ApplicationContext`).

### 2. Add file logging

**File:** `client/src/ScreenTimer.Agent.Host/ScreenTimer.Agent.Host.csproj`

- Add a package reference to `Serilog.Extensions.Hosting` and `Serilog.Sinks.File` (or alternatively, use the simpler built-in approach with `AddFile` from `Microsoft.Extensions.Logging` — but that has no first-party rolling-file provider, so Serilog is the pragmatic choice).

**File:** `client/src/ScreenTimer.Agent.Host/Program.cs`

- Remove the console logging provider.
- Configure Serilog (or a file logging provider) to write to a well-known log file path, e.g. `%LOCALAPPDATA%/ScreenTimer/logs/agent.log`.
- Enable rolling/size-limited logs to avoid unbounded growth.

**File:** `client/src/ScreenTimer.Agent.Host/appsettings.json`

- Remove the `Console` formatter configuration (no longer needed).

### 3. Create a tray application context

**New file:** `client/src/ScreenTimer.Agent.Host/TrayApplicationContext.cs`

Create a class inheriting from `ApplicationContext` that:

- Creates a `NotifyIcon` with an icon and tooltip (e.g. "Screen Timer").
- Adds a `ContextMenuStrip` with two items:
  - **Open Logs** — calls `Process.Start("notepad.exe", logFilePath)`.
  - **Exit** — calls `Application.Exit()`.
- Disposes the `NotifyIcon` on exit.

### 4. Rewrite Program.cs entry point

**File:** `client/src/ScreenTimer.Agent.Host/Program.cs`

- Build and start the `IHost` (with the `AgentWorker` background service) without calling `host.Run()` (which blocks and expects console lifetime).
- Instead, after `host.StartAsync()`, launch the Windows Forms message loop via `Application.Run(new TrayApplicationContext(...))`.
- When `Application.Run` returns (user clicked Exit), call `host.StopAsync()` and dispose the host.

### 5. Add an application icon

**New file:** `client/src/ScreenTimer.Agent.Host/Resources/tray.ico`

- Add a simple icon file for the tray. Can be a placeholder initially.
- Embed it as a resource in the `.csproj`.

## Order of Implementation

1. Add Serilog packages and configure file logging (step 2).
2. Change project to WinExe with WinForms (step 1).
3. Add icon resource (step 5).
4. Create `TrayApplicationContext` (step 3).
5. Rewrite `Program.cs` entry point (step 4).
6. Build and test.
