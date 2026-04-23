# TODO: Convert Console App to Windowless Tray App

## Tasks

- [x] 1. Add Serilog NuGet packages (`Serilog.Extensions.Hosting`, `Serilog.Sinks.File`) to `ScreenTimer.Agent.Host.csproj`
- [x] 2. Define log file path (`%LOCALAPPDATA%/ScreenTimer/logs/agent.log`) in `Program.cs`
- [x] 3. Configure Serilog file logging in `Program.cs` (replace console logger)
- [x] 4. Configure rolling file / size limits for Serilog to prevent unbounded log growth
- [x] 5. Remove console-specific logging config from `appsettings.json`
- [x] 6. Add `<OutputType>WinExe</OutputType>` and `<UseWindowsForms>true</UseWindowsForms>` to `.csproj`
- [x] 7. Generate a placeholder tray icon (`Resources/tray.ico`) and embed it as a resource in `.csproj`
- [x] 8. Create `TrayApplicationContext.cs` — `ApplicationContext` subclass with `NotifyIcon`, tooltip, and `ContextMenuStrip`
- [x] 9. Add "Open Logs" menu item that opens the log file in Notepad
- [x] 10. Add "Exit" menu item that calls `Application.Exit()`
- [x] 11. Rewrite `Program.cs` entry point: `host.StartAsync()` → `Application.Run(TrayApplicationContext)` → `host.StopAsync()`
- [x] 12. Ensure `NotifyIcon` is disposed on exit (hide icon from tray on shutdown)
- [x] 13. Build the solution and verify no compile errors
- [x] 14. Run all tests and verify they still pass
- [x] 15. Fix integration test project: add `UseWindowsForms` and `RollForward` so tests run with available .NET desktop runtime
