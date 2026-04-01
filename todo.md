# Screen Timer Client — TODO

## Phase 1: Core Domain + Headless Tests
- [x] Create solution and project structure (sln, Core, Core.Tests)
- [x] Define domain models (AppRule, AgentState, ForegroundSample, AppUsageState)
- [x] Define interfaces (IClock, IForegroundWindowProbe, IAgentApiClient, INotificationSink, IProcessController, IStateStore)
- [x] Define DTOs (AppConfigDto, UsageReportDto, UsagePushDto)
- [x] Define EngineResult and command types
- [x] Implement AgentEngine.Tick() — tracking logic (elapsed-time attribution)
- [x] Implement notification policy (10, 5, 1 min thresholds)
- [x] Implement enforcement logic (ForceClose on budget exhaustion)
- [x] Implement usage batching (PushUsage command generation)
- [x] Implement midnight reset logic
- [x] Implement config sync handling
- [x] Write tracking tests (8 tests)
- [x] Write notification tests (7 tests)
- [x] Write enforcement tests (5 tests)
- [x] Write reset tests (5 tests)
- [x] Write sync tests (7 tests)
- [x] Write persistence round-trip tests (3 tests)

## Phase 2: HTTP Client + Persistence Adapters
- [x] Create ScreenTimer.Agent.Windows project
- [x] Implement AgentApiClient (HttpClient wrapper)
- [x] Implement JsonStateStore (read/write JSON in %LocalAppData%)
- [x] Add adapter-level tests (JSON serialization, file round-trip)

## Phase 3: Windows Adapters + Background Host
- [x] Implement Win32ForegroundWindowProbe (P/Invoke)
- [x] Implement ToastNotificationSink (console placeholder — TODO: real toast)
- [x] Implement WindowsProcessController (graceful close → kill)
- [x] Create ScreenTimer.Agent.Host project with GenericHost + BackgroundService
- [x] Implement AgentWorker (1-second tick loop)
- [x] Wire DI and configuration (appsettings.json)

## Phase 4: Fullscreen Harness
- [ ] Build ScreenTimer.FullscreenHarness

## Phase 5: Integration Tests + Stabilization
- [ ] Create ScreenTimer.Agent.IntegrationTests
- [x] Add cross-language contract smoke test
- [x] Add retry/backoff for network failures
- [x] Polish logging

## Discovered Tasks
- [x] Replace console ToastNotificationSink with real Windows toast notifications
- [x] Add .gitignore for bin/obj
- [x] Fix WindowsProcessController exe name bug (GetProcessesByName needs name without .exe extension)
- [x] Fix WindowsProcessController to use async WaitForExitAsync instead of blocking WaitForExit
- [x] Set Product name in Host csproj for toast notification title
- [ ] Write README with startup/packaging instructions
