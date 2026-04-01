# Screen Timer — MVP C# Client Plan

## Architecture

Three projects, cleanly separated:

| Project | Responsibility | Dependencies |
|---------|---------------|--------------|
| **ScreenTimer.Agent.Core** | All business logic: tracking, budgets, notification scheduling, enforcement decisions | None (pure .NET, no OS/UI/HTTP calls) |
| **ScreenTimer.Agent.Windows** | OS adapters: foreground detection, toast display, process kill, HTTP client, file persistence | Core |
| **ScreenTimer.Agent.Host** | Background process wiring (`BackgroundService` / `GenericHost`) | Core, Windows |

A separate **ScreenTimer.FullscreenHarness** test utility (not production code) is used for manual full-screen validation.

### Core engine design: tick-driven, command-based

The host runs a loop every ~1 second. Each iteration:

1. Samples the foreground window (via an adapter).
2. Passes the sample + current time + current state + config into a **pure** `AgentEngine.Tick()` method.
3. `Tick()` returns an `EngineResult` containing updated state and a list of **commands** to execute:
   - `ShowToast(exeName, remaining)`
   - `PushUsage(usageBatch)`
   - `ForceClose(exeName)`
   - `PersistState`
4. The host dispatches commands to the appropriate adapters.

Because the engine is a pure function of its inputs, all logic is testable headlessly with fake clocks, fake probes, and in-memory sinks.

### Interfaces (defined in Core, implemented in Windows)

```csharp
IForegroundWindowProbe   // returns ForegroundSample { ExeName, Timestamp }
IAgentApiClient          // GET /api/agent/config, POST /api/agent/usage
INotificationSink        // show a toast / overlay (abstracted)
IProcessController       // graceful close + hard kill
IStateStore              // load/save agent state (JSON file)
IClock                   // DateTimeOffset.Now (fakeable)
```

### Domain state

Per tracked app, per day:
- `usedTodaySeconds`
- `pendingUploadSeconds`
- notification flags: `sent10Min`, `sent5Min`, `sent1Min`
- `exhausted` flag
- last sample info (foreground exe + timestamp)
- last config poll time, last usage flush time
- current local date (for midnight reset)

### Tracking algorithm

Use **elapsed-time attribution**, not "+1s if focused":
- On each tick, compute `elapsed = now - previousTickTime`.
- Attribute elapsed time to the **previous** foreground tracked exe (if any).
- This stays accurate even if the loop stalls for 2–3 seconds.

### Polling cadence

- Foreground sampling: every **1 second**
- Usage push: every **10–15 seconds**
- Config poll: every **30–60 seconds**

### Local persistence

Persist to a JSON file in `%LocalAppData%` on every usage flush:
- `usedTodaySeconds` per app
- `pendingUploadSeconds` per app
- notification flags and exhausted flags
- date stamp (for midnight reset detection)

This is necessary because the server API does not return "used today" — the agent is the source of truth for local usage.

---

## Testing Strategy

### 1. Headless unit tests (ScreenTimer.Agent.Core.Tests)

All business logic tested with fake dependencies — fast, deterministic, runs in CI.

**Tracking tests:**
- Counts usage only when a tracked exe is foreground
- Ignores untracked apps
- Correctly switches attribution when foreground changes
- Does not count time when no valid app is foreground
- Handles variable tick intervals (1s, 2.5s, etc.)

**Notification tests:**
- Fires toast at 10, 5, 1 minute thresholds (only once each)
- Does not re-fire every tick after threshold crossed
- Does not fire if budget is already past threshold on first tick

**Enforcement tests:**
- Fires ForceClose when budget reaches zero
- Re-fires ForceClose if exhausted app reappears in foreground
- Does not fire ForceClose for untracked apps

**Reset tests:**
- Midnight date change clears usage, notification flags, exhausted flags

**Sync tests:**
- Pending usage batches correctly for upload
- Failed push preserves pending usage
- Successful push clears only the uploaded portion
- Config change updates budgets for existing apps
- Deleted app stops being tracked

**Persistence tests:**
- State round-trips through save/load
- Restart restores persisted state correctly

### 2. Headless integration tests (ScreenTimer.Agent.IntegrationTests)

Test the real `AgentWorker` loop with fake OS adapters but real HTTP against a test server:

- Polling cadence works correctly
- Usage flush cadence works correctly
- JSON contract matches the Go server's expectations
- Retry/backoff on network failure
- Persistence survives simulated restart

**Cross-language contract smoke test:**
- Start the real Go server in test setup
- Call `GET /api/agent/config` and `POST /api/agent/usage`
- Verify DTO serialization/deserialization matches

### 3. Full-screen UI validation (manual + semi-automated)

This is the critical part that **cannot** be tested headlessly. The strategy:

#### A. Build a FullscreenHarness test utility

A small standalone app that:
- Can launch in **windowed**, **borderless fullscreen**, or **exclusive fullscreen** mode
- Displays an obvious on-screen animation so you know it's active and rendering
- Optionally ignores graceful close (WM_CLOSE) for a configurable delay — to test hard-kill fallback
- Logs when it receives close events

This gives you a reproducible, controlled target app with short budgets (2–3 minutes) instead of needing to launch real games every time.

#### B. Semi-automated test script

A script that:
1. Configures the server with a tiny budget (e.g., 2 minutes) for the harness exe
2. Launches the harness in a given fullscreen mode
3. Starts the agent
4. Waits for threshold times
5. **Automatically verifies** via agent logs:
   - Toast command was emitted at correct thresholds
   - ForceClose command was emitted at budget exhaustion
   - Target process exited
   - Usage was pushed to server correctly
6. **Manual verification** (by the tester watching the screen):
   - Was the toast actually visible during fullscreen?
   - Did it steal focus or cause stutter?
   - Did force-close feel abrupt or broken?

#### C. Real game validation

After harness testing, validate with 1–2 real games:
- One **borderless fullscreen** title
- One **exclusive fullscreen** title (if available)

#### D. Test matrix

For each target (harness windowed / harness borderless / harness exclusive / real game):

| Scenario | Verify |
|----------|--------|
| App focused for full countdown | All 3 toasts appear, force-close works |
| Alt-tab away mid-countdown, then back | Time only counts while focused |
| Budget exhausted while app is focused | Force-close triggers |
| App ignores graceful close | Hard kill fallback works |
| Network disconnected during play | Pending usage preserved, pushed on reconnect |
| Agent restart during session | State restored, tracking resumes |

#### E. Instrumentation (built from day 1)

Structured logs with timestamps for:
- Foreground exe changes
- Tracked seconds increments
- Threshold crossings
- Toast requests
- Config poll success/failure
- Usage push success/failure
- Enforcement start/end
- Process IDs targeted

For full-screen testing, **logs are ground truth**. Use a phone camera or secondary monitor to observe toast visibility — desktop screen capture is unreliable for exclusive fullscreen content.

#### F. Decision rule

- If toasts work acceptably in **borderless fullscreen** (which most modern games use), keep MVP as-is.
- If toasts consistently fail in **exclusive fullscreen** and that's a problem for the target games, that's the trigger to pull a **minimal overlay** into scope as a post-MVP iteration. The `INotificationSink` abstraction means the core engine doesn't change.

---

## Implementation Phases

### Phase 1 — Core domain + headless tests (~1–2 days)

1. Create solution and `ScreenTimer.Agent.Core` project.
2. Define domain models: `AppRule`, `AgentState`, `ForegroundSample`, `UsageBucket`.
3. Define interfaces: `IClock`, `IForegroundWindowProbe`, `IAgentApiClient`, `INotificationSink`, `IProcessController`, `IStateStore`.
4. Define DTOs: `AppConfigDto`, `UsageReportDto`.
5. Implement `AgentEngine.Tick()` — tracking, thresholds, enforcement, batching.
6. Implement `EngineResult` and command types.
7. Create `ScreenTimer.Agent.Core.Tests` with full headless test coverage (tracking, notifications, enforcement, reset, sync, persistence round-trip).

### Phase 2 — HTTP client + persistence adapters (~0.5–1 day)

1. Create `ScreenTimer.Agent.Windows` project.
2. Implement `AgentApiClient` (HttpClient wrapper for `GET /api/agent/config`, `POST /api/agent/usage`).
3. Implement `JsonStateStore` (read/write JSON in `%LocalAppData%`).
4. Add adapter-level tests (JSON serialization, file round-trip).

### Phase 3 — Windows adapters + background host (~1–2 days)

1. Implement `Win32ForegroundWindowProbe` (P/Invoke: `GetForegroundWindow`, `GetWindowThreadProcessId`, process lookup).
2. Implement `ToastNotificationSink` (Windows toast notification API).
3. Implement `WindowsProcessController` (graceful close → wait → kill).
4. Create `ScreenTimer.Agent.Host` project with `GenericHost` + `BackgroundService`.
5. Implement `AgentWorker` — the 1-second tick loop that calls `AgentEngine.Tick()` and dispatches commands.
6. Wire DI and configuration (`appsettings.json` with server URL, poll intervals).

### Phase 4 — Fullscreen harness + end-to-end validation (~1 day)

1. Build `ScreenTimer.FullscreenHarness` (windowed / borderless / exclusive modes, close-resistance option).
2. Write semi-automated test script (set tiny budget, launch harness, verify via logs).
3. Run manual validation with harness in all 3 modes.
4. Run manual validation with 1–2 real games.
5. Document observed toast behavior per mode.
6. Decide: are toasts sufficient, or is overlay needed post-MVP?

### Phase 5 — Integration tests + stabilization (~0.5–1 day)

1. Create `ScreenTimer.Agent.IntegrationTests` — real worker loop with fake OS + test HTTP server.
2. Add cross-language contract smoke test against the real Go server.
3. Add retry/backoff for network failures.
4. Polish logging (structured, timestamped).
5. Add error handling for kill failures (log, don't crash).
6. Write startup/packaging instructions.

---

## File Layout

```
client/
├── src/
│   ├── ScreenTimer.Agent.Core/
│   │   ├── Models/
│   │   │   ├── AppRule.cs
│   │   │   ├── AgentState.cs
│   │   │   ├── ForegroundSample.cs
│   │   │   └── UsageBucket.cs
│   │   ├── Interfaces/
│   │   │   ├── IClock.cs
│   │   │   ├── IForegroundWindowProbe.cs
│   │   │   ├── IAgentApiClient.cs
│   │   │   ├── INotificationSink.cs
│   │   │   ├── IProcessController.cs
│   │   │   └── IStateStore.cs
│   │   ├── Engine/
│   │   │   ├── AgentEngine.cs
│   │   │   ├── EngineResult.cs
│   │   │   ├── NotificationPolicy.cs
│   │   │   └── UsageAccumulator.cs
│   │   └── Dtos/
│   │       ├── AppConfigDto.cs
│   │       └── UsageReportDto.cs
│   │
│   ├── ScreenTimer.Agent.Windows/
│   │   ├── Foreground/
│   │   │   └── Win32ForegroundWindowProbe.cs
│   │   ├── Notifications/
│   │   │   └── ToastNotificationSink.cs
│   │   ├── Processes/
│   │   │   └── WindowsProcessController.cs
│   │   ├── Storage/
│   │   │   └── JsonStateStore.cs
│   │   └── Http/
│   │       └── AgentApiClient.cs
│   │
│   ├── ScreenTimer.Agent.Host/
│   │   ├── AgentWorker.cs
│   │   ├── Program.cs
│   │   └── appsettings.json
│   │
│   └── ScreenTimer.FullscreenHarness/
│       └── Program.cs
│
├── tests/
│   ├── ScreenTimer.Agent.Core.Tests/
│   │   ├── TrackingTests.cs
│   │   ├── NotificationTests.cs
│   │   ├── EnforcementTests.cs
│   │   ├── ResetTests.cs
│   │   └── SyncTests.cs
│   │
│   └── ScreenTimer.Agent.IntegrationTests/
│       ├── WorkerPollingTests.cs
│       ├── UsagePushTests.cs
│       ├── PersistenceTests.cs
│       └── GoServerContractTests.cs
│
└── ScreenTimer.sln
```
