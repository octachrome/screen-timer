# Screen Timer — Plan: Consolidate Processes into Groups

## Goal

Replace the current flat "tracked application" model with a **group-centric** model. Every tracked item becomes a **group** with a name, a budget, and one or more member processes. The "Add Application" form stays the same (enter a process name + budget, click Add) but under the hood creates a single-member group named after the process. The Edit button expands to let the user add more processes to the group. The client/agent becomes group-aware so notifications say *"Gaming: 1 minute remaining"* instead of *"Fortnite: 1 minute(s) remaining"*.

---

## Phase 1 — Server: data model & persistence

### Model changes (`model.go`)

- Replace `Application` with a `Group` struct:
  ```go
  type Group struct {
      Name          string
      Processes     []string       // process names
      DailyBudget   time.Duration
      UsedToday     time.Duration
      LastResetDate string
  }
  ```
- Update `UsageSummary` to include group name and member list:
  ```go
  type UsageSummary struct {
      Name               string   `json:"name"`
      Processes          []string `json:"processes"`
      DailyBudgetMinutes int      `json:"daily_budget_minutes"`
      UsedTodayMinutes   int      `json:"used_today_minutes"`
      RemainingMinutes   int      `json:"remaining_minutes"`
  }
  ```
- Update `AddAppRequest` → `AddGroupRequest`: fields `name`, `process` (single exe), `daily_budget_minutes`.
- Add `UpdateGroupRequest` with fields for budget and process list.
- Update `AgentConfigResponse` to include groups:
  ```go
  type GroupConfig struct {
      Name               string   `json:"name"`
      Processes          []string `json:"processes"`
      DailyBudgetMinutes int      `json:"daily_budget_minutes"`
  }
  type AgentConfigResponse struct {
      Groups      []GroupConfig `json:"groups"`
      TestPopupAt string        `json:"test_popup_at,omitempty"`
  }
  ```

### Store changes (`store.go`)

- Change internal map: `apps map[string]*Application` → `groups map[string]*Group` (keyed by group name).
- Update persistence structs (`persistedData`, `persistedGroup`) to include the process list.
- Update `save()`/`load()` to serialise the new shape.
- **Migration**: when loading an old-format JSON file (with `"apps"` key), convert each app into a single-process group with `name = exeName`.
- Rename store methods:
  - `AddApp()` → `AddGroup(name, process, budget)` — creates a group with one process; error if name already exists.
  - `UpdateBudget()` → `UpdateGroup(name, budget, processes)` — updates budget and/or process list.
  - `DeleteApp()` → `DeleteGroup(name)`.
  - `ListApps()` → `ListGroups()`.
  - `GetUsageSummary()` — returns `[]UsageSummary` with group-level data.
- `RecordUsage(exeName, seconds, totalSeconds)` — stays process-based (agent reports per-exe). The store looks up **all groups containing that exe** and adds usage to each.

### Handler changes (`handlers.go`)

- `POST /api/apps` — accept `AddGroupRequest`; call `store.AddGroup()`.
- `PUT /api/apps/{name}` — accept `UpdateGroupRequest`; call `store.UpdateGroup()`.
- `DELETE /api/apps/{name}` — call `store.DeleteGroup()`.
- `GET /api/usage/today` — return group-level summaries.
- `GET /api/agent/config` — return `GroupConfig` list so the agent knows group names and which processes belong to which groups.
- Keep URL paths as `/api/apps` to avoid frontend URL changes (or rename to `/api/groups` and update frontend — either works; keeping `/api/apps` is simpler).

### Server tests

- Update `store_test.go` and `handlers_test.go` to exercise the new group-based methods.

---

## Phase 2 — Web UI changes (`app.js`, `index.html`)

### Add form (unchanged UX, different payload)

- The form still has "Process name" + "Daily budget" + Add button.
- On submit, POST to `/api/apps` with `{ "name": "<process>", "process": "<process>", "daily_budget_minutes": N }`. (The group name defaults to the process name.)

### Tracked Applications table

- Rename column "Process" → "Name".
- Add a "Processes" column showing the comma-separated list of member exe names.
- Edit button opens an inline editor that shows:
  - Budget (number input, as today).
  - Process list (comma-separated text input or a small editable list).
  - Save sends `PUT /api/apps/{name}` with `{ "daily_budget_minutes": N, "processes": ["a.exe", "b.exe"] }`.

### Rendering

- `renderTable()` reads `name` and `processes` from the new summary shape.
- `startEdit()` adds a processes text field alongside the budget input.

---

## Phase 3 — Client/agent: group-aware config & notifications

### DTO changes

- `AgentConfigResponseDto` gains a `Groups` list (replacing or alongside `Apps`).
  ```csharp
  public sealed class GroupConfigDto
  {
      [JsonPropertyName("name")]
      public string Name { get; set; } = "";
      [JsonPropertyName("processes")]
      public List<string> Processes { get; set; } = new();
      [JsonPropertyName("daily_budget_minutes")]
      public int DailyBudgetMinutes { get; set; }
  }
  ```
- `AgentConfigResponseDto.Apps` → `AgentConfigResponseDto.Groups`.

### Model changes

- `AppRule` → `GroupRule`: `Name`, `Processes` (list), `DailyBudgetMinutes`.
- `AgentState.Apps` (per-exe usage) stays as-is — usage is still tracked per-exe.
- Add `AgentState.GroupUsage`: `Dictionary<string, GroupUsageState>` keyed by group name, tracking notification flags and exhaustion per group.
  ```csharp
  public sealed class GroupUsageState
  {
      public bool Sent10Min { get; set; }
      public bool Sent5Min { get; set; }
      public bool Sent1Min { get; set; }
      public bool Exhausted { get; set; }
  }
  ```

### Engine changes (`AgentEngine.cs`)

- **Usage attribution** (unchanged): attribute elapsed seconds to the per-exe `AppUsageState` as today.
- **Budget checking**: iterate `CurrentRules` (now `List<GroupRule>`). For each group, sum `UsedTodaySeconds` across all member processes. Compare against the group budget.
- **Notifications**: fire `ShowToastCommand(groupName, remainingMinutes)` — using the **group name** so the toast reads *"1 minute remaining for Gaming"*.
- **Enforcement**: if a group's budget is exhausted and the current foreground exe is a member of that group, emit `ForceCloseCommand(currentExe)`.

### Notification sink

- `ShowToast(string label, int remainingMinutes)` — change the message from `"{exeName}: {n} minute(s) remaining"` to `"{n} minute(s) remaining for {label}"`. The `label` is now the group name.

### Worker (`AgentWorker.cs`)

- Config poll: map `GroupConfigDto` list → `List<GroupRule>`.
- Ensure per-exe `AppUsageState` entries are created for every exe mentioned in any group.

### Client tests

- Update `NotificationTests`, `EnforcementTests`, etc. to use group rules.
- Add test: two exes in one group, usage from both counts toward the shared budget.
- Add test: notification message uses group name.

---

## Migration / backwards compatibility

- **Server persistence**: `load()` expects the new groups format. Old-format JSON files (with the `"apps"` key) are ignored and will be overwritten on the next save.
- **Agent config**: the server always returns the new `groups` format. Old agents that only understand `apps` will see an empty app list and stop enforcing — acceptable since both are deployed together.

## Out of scope

- The built-in "All" group (separate feature, can be added later).
- Weekday/weekend budgets, ad-hoc extensions (later phases from the old plan).
