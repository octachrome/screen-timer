# Phase 1 — Server: Data Model & Persistence — TODO

## Model changes (`model.go`)

- [x] 1. Replace `Application` struct with `Group` struct (Name, Processes []string, DailyBudget, UsedToday, LastResetDate)
- [x] 2. Update `UsageSummary` to include `name` and `processes` fields (replace `exe_name`)
- [x] 3. Replace `AddAppRequest` with `AddGroupRequest` (fields: name, process, daily_budget_minutes)
- [x] 4. Add `UpdateGroupRequest` with fields for budget and process list
- [x] 5. Replace `AppConfig`/`AgentConfigResponse` with `GroupConfig`/updated `AgentConfigResponse` using groups
- [x] 6. Update `ToUsageSummary()` and `ToAppConfig()` conversion methods for group model

## Store changes (`store.go`)

- [x] 7. Change internal map from `apps map[string]*Application` to `groups map[string]*Group` (keyed by group name)
- [x] 8. Update persistence structs (`persistedData`, `persistedGroup`) to include process list
- [x] 9. Update `save()`/`load()` to serialise the new group shape
- [x] 10. Add migration: when loading old-format JSON (with `"apps"` key), convert each app into a single-process group
- [x] 11. Rename `AddApp()` → `AddGroup(name, process, budget)` — creates group with one process; error if name exists
- [x] 12. Rename `UpdateBudget()` → `UpdateGroup(name, budget, processes)` — updates budget and/or process list
- [x] 13. Rename `DeleteApp()` → `DeleteGroup(name)`
- [x] 14. Rename `ListApps()` → `ListGroups()`
- [x] 15. Update `GetUsageSummary()` to return group-level data with name and processes
- [x] 16. Update `RecordUsage()` — stays process-based; looks up all groups containing that exe and adds usage to each

## Handler changes (`handlers.go`)

- [x] 17. Update `POST /api/apps` to accept `AddGroupRequest` and call `store.AddGroup()`
- [x] 18. Update `PUT /api/apps/{name}` to accept `UpdateGroupRequest` and call `store.UpdateGroup()`
- [x] 19. Update `DELETE /api/apps/{name}` to call `store.DeleteGroup()`
- [x] 20. Update `GET /api/usage/today` to return group-level summaries
- [x] 21. Update `GET /api/agent/config` to return `GroupConfig` list
- [x] 22. Update `GET /api/apps` to return group-level list
- [x] 23. Update route path parameter from `{exe}` to `{name}`

## Mock client changes (`mockclient/client.go`)

- [x] 24. Update mock client types to match new group-based API (AddGroupRequest, UpdateGroupRequest, GroupConfig, etc.)
- [x] 25. Update mock client methods (AddApp→AddGroup, UpdateApp→UpdateGroup, DeleteApp→DeleteGroup, etc.)

## Test changes

- [x] 26. Update `store_test.go` to exercise new group-based store methods
- [x] 27. Update `handlers_test.go` to exercise new group-based handlers
- [x] 28. Update `integration_test.go` to exercise new group-based end-to-end flows
- [x] 29. Add test: RecordUsage adds usage to all groups containing the process
- [x] 30. Add test: persistence migration from old app-format JSON to group format
- [x] 31. Add test: group with multiple processes — usage from each counts toward shared budget
