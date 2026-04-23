# TODO: Weekday / Weekend Budgets

## Server Data Model (model.go)
- [x] 1. Add `WeekendBudget time.Duration` field to `Group` struct
- [x] 2. Add `weekend_budget_minutes` field to `AddGroupRequest` DTO
- [x] 3. Add `weekend_budget_minutes` field to `UpdateGroupRequest` DTO
- [x] 4. Add `weekend_budget_minutes` field to `GroupConfig` DTO (agent-facing)
- [x] 5. Add `weekend_budget_minutes` field to `UsageSummary` DTO (UI-facing)
- [x] 6. Add `isWeekend(t time.Time) bool` helper function
- [x] 7. Update `ToUsageSummary()` to accept `time.Time`, return weekend budget, and set `DailyBudgetMinutes` to active budget
- [x] 8. Update `ToGroupConfig()` to return `weekend_budget_minutes`

## Server Persistence (store.go)
- [x] 9. Add `WeekendBudget int64` field to `persistedGroup`
- [x] 10. Update `save()` to write `WeekendBudget`
- [x] 11. Update `load()` to read `WeekendBudget` (zero = same as weekday, no migration needed)
- [x] 12. Update `AddGroup()` signature to accept weekend budget parameter
- [x] 13. Update `UpdateGroup()` signature to accept weekend budget parameter
- [x] 14. Update `GetUsageSummary()` to pass current time to `ToUsageSummary()`

## Server Handlers (handlers.go)
- [x] 15. Update `handleAddApp`: read `weekend_budget_minutes`, pass to store; default to weekday if zero/absent
- [x] 16. Update `handleUpdateApp`: read `weekend_budget_minutes`, pass to store; default to weekday if zero/absent
- [x] 17. Update `handleListApps` to pass current time to `ToUsageSummary()`

## Mock Client (mockclient/client.go)
- [x] 18. Add `WeekendBudgetMinutes` to mockclient `GroupConfig`, `UsageSummary`, `AddGroupRequest`, `UpdateGroupRequest`

## Server Tests
- [ ] 19. Store test: create group with weekend budget, verify `GetUsageSummary` returns weekend budget on Saturday
- [ ] 20. Store test: verify `GetUsageSummary` returns weekday budget on Monday
- [ ] 21. Store test: omitting weekend budget defaults to weekday value
- [ ] 22. Store test: persistence round-trip preserves weekend budget
- [ ] 23. Handler test: POST `/api/apps` with `weekend_budget_minutes`, verify response includes both
- [ ] 24. Handler test: PUT `/api/apps/{name}` updating `weekend_budget_minutes`
- [ ] 25. Handler test: omitting `weekend_budget_minutes` defaults to weekday value
- [ ] 26. Integration test: add group with weekend budget, agent polls config, sees both budgets

## UI — Add Application Form (index.html)
- [ ] 27. Add "Different on weekends" checkbox to the Add Application form
- [ ] 28. Add "Weekend budget (minutes)" input, hidden by default

## UI — Add Form JS Logic (app.js)
- [ ] 29. Wire checkbox to show/hide weekend budget input
- [ ] 30. On form submit: if checkbox unchecked, omit or send weekday value for `weekend_budget_minutes`; if checked, send the weekend input value

## UI — Tracked Applications Table (app.js)
- [ ] 31. Split "Budget" column into "Weekday" and "Weekend" columns
- [ ] 32. Bold/highlight the column that applies today (weekday or weekend)

## UI — Inline Edit (app.js)
- [ ] 33. In `startEdit()`, show two number inputs for weekday + weekend budgets
- [ ] 34. On save, send both budget values in the PUT request

## C# Agent — DTOs and Models
- [ ] 35. Add `WeekendBudgetMinutes` property to `GroupConfigDto.cs` with `[JsonPropertyName("weekend_budget_minutes")]`
- [ ] 36. Add `WeekendBudgetMinutes` property to `GroupRule.cs`
- [ ] 37. Update `AgentWorker.cs` mapping to copy `WeekendBudgetMinutes` from DTO to GroupRule

## C# Agent — Engine Logic
- [ ] 38. Update `AgentEngine.Tick()` to pick correct budget based on day of week
- [ ] 39. Make day-of-week determination injectable/testable (e.g. pass current time or DayOfWeek)

## C# Agent Tests
- [ ] 40. Test: engine applies weekday budget on a weekday
- [ ] 41. Test: engine applies weekend budget on a weekend
- [ ] 42. Test: notifications fire at correct thresholds for weekend budget on weekend
