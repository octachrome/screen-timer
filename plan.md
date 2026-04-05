# Screen Timer — Post-MVP Plan

What's built: per-app tracking, single daily budgets, force-close enforcement, toast notifications, basic web UI, file-based persistence. See `mvp.md` for details.

What remains: application groups, weekday/weekend budgets, ad-hoc extensions, fullscreen-aware notifications, persistent overlay experimentation, historical CSV export, and auto-start on boot.

---

## Phase 1 — Application Groups & the "All" Group

Add the concept of named application groups to both server and client.

**Server:**
- New data model: `Group` with a name, a list of member exe names, and its own daily budget.
- A built-in "All" group that implicitly contains every tracked application.
- CRUD API for groups: `POST/GET/PUT/DELETE /api/groups`.
- Assign/remove applications to groups.
- Extend usage tracking to compute per-group usage (sum of member usage).
- Usage summary endpoint returns both per-app and per-group data.

**Client (agent):**
- Config response includes group definitions and budgets alongside app budgets.
- Engine enforces all applicable budgets independently: if an app belongs to groups "Games" and "All", it's blocked when *any* of its budgets (app, Games, or All) is exhausted.
- Notification thresholds fire per-budget (the child gets the most urgent notification across all applicable budgets).

**UI:**
- Create/edit/delete groups.
- Assign applications to groups.
- View per-group usage alongside per-app usage.

## Phase 2 — Weekday / Weekend Budgets

Replace the single daily budget with separate weekday (Mon–Fri) and weekend (Sat–Sun) defaults, at every budget level (per-app, per-group, and "All").

**Server:**
- Model change: replace `DailyBudget` with `WeekdayBudget` and `WeekendBudget` on both `Application` and `Group`.
- Agent config endpoint returns both budgets; the agent picks the correct one based on the current day of week.
- UI API and usage summary include both budget values.

**Client:**
- Engine selects the active budget (weekday or weekend) based on the local date.

**UI:**
- Budget inputs become two fields (weekday / weekend) everywhere budgets are set.

## Phase 3 — Ad-Hoc Daily Budget Extensions

Let the manager extend today's budget for any application, group, or "All" without changing the default.

**Server:**
- New field: `ExtensionMinutes` per app/group (resets daily).
- API endpoint: `POST /api/apps/{exe}/extend`, `POST /api/groups/{name}/extend` with `{ "extra_minutes": 30 }`.
- Effective budget = default budget (weekday/weekend) + extension for today.

**Client:**
- Config response includes extensions; engine uses effective budget.

**UI:**
- "+30 min" / custom extension button on each app/group row.

## Phase 4 — Fullscreen-Aware Notifications

Replace or supplement standard Windows toast notifications with a notification mechanism that is visible over fullscreen games and does not steal keyboard focus.

**Investigation (do first):**
- Test standard toasts with the fullscreen harness (`ScreenTimer.FullscreenHarness`) in borderless and exclusive fullscreen modes.
- If toasts work in borderless fullscreen (which most modern games use), document that and consider MVP-complete for common cases.

**Implementation (if toasts fail in target scenarios):**
- Build a topmost transparent overlay window (WPF or WinForms) that renders above fullscreen content.
- The overlay shows the notification text for a few seconds then fades.
- Use `WS_EX_NOACTIVATE` / `WS_EX_TRANSPARENT` to prevent focus stealing.
- Swap `INotificationSink` implementation — core engine unchanged.

## Phase 5 — Persistent Time-Remaining Overlay

Experiment with a persistent on-screen widget that shows remaining time for the currently focused tracked application (requirement F6).

**Experiment:**
- Build a small always-on-top, click-through overlay window (e.g., a corner HUD showing "Fortnite: 42 min left").
- Test visibility across windowed, borderless fullscreen, and exclusive fullscreen modes.
- Measure whether it causes input lag, stutter, or focus issues.

**If viable:** ship it as the default display, controlled by a setting.
**If not viable:** document findings; rely on pop-up notifications only (Phase 4).

## Phase 6 — Historical Usage & CSV Export

Record daily usage history and allow export.

**Server:**
- Store daily usage history (per-app, per-group) — append a summary at end-of-day or on each usage push.
- Storage: append to a CSV file or a JSON-lines file, one row per app per day.
- New endpoint: `GET /api/usage/history?days=30` — returns historical data.
- New endpoint: `GET /api/usage/export` — returns CSV download.

**UI (optional):**
- No UI required per the requirements, but a "Download CSV" button would be simple to add.

## Phase 7 — Auto-Start on Boot & Polish

Make the agent start automatically and apply final polish.

**Auto-start:**
- Register the agent with Windows Task Scheduler (run at logon, non-elevated).
- Alternatively, add a registry entry under `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`.
- Make this configurable (an installer step, or a `--install` / `--uninstall` CLI flag on the agent).

**Polish:**
- Structured logging review (ensure all key events are logged).
- Error handling for edge cases (agent crash recovery, stale state files).
- Configuration: make server URL, poll intervals, notification thresholds configurable via `appsettings.json`.
- README / user-facing documentation for setup and usage.
