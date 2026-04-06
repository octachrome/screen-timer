# Phase 3 — Client/Agent: Group-Aware Config & Notifications: Todo List

## DTO Changes
- [x] 1. Create `GroupConfigDto` class with `Name`, `Processes`, `DailyBudgetMinutes` (JSON: `name`, `processes`, `daily_budget_minutes`)
- [x] 2. Replace `AgentConfigResponseDto.Apps` (List\<AppConfigDto\>) with `Groups` (List\<GroupConfigDto\>) (JSON: `groups`)
- [x] 3. Delete `AppConfigDto` (no longer needed)

## Model Changes
- [x] 4. Replace `AppRule` with `GroupRule`: fields `Name`, `Processes` (List\<string\>), `DailyBudgetMinutes`
- [x] 5. Add `GroupUsageState` class with notification flags (`Sent10Min`, `Sent5Min`, `Sent1Min`, `Exhausted`)
- [x] 6. Add `AgentState.GroupUsage` dictionary (Dictionary\<string, GroupUsageState\>) keyed by group name
- [x] 7. Move notification/exhaustion flags out of `AppUsageState` (they now live on `GroupUsageState`)

## Engine Changes (AgentEngine.cs)
- [x] 8. Update `ApplyConfigRules` to accept `List<GroupRule>` and create `AppUsageState` entries for every process mentioned in any group
- [x] 9. Update budget checking to iterate groups, summing `UsedTodaySeconds` across all member processes per group
- [x] 10. Update notification logic to use group name in `ShowToastCommand` instead of exe name
- [x] 11. Update enforcement logic: if group budget exhausted and current foreground exe is a member, emit `ForceCloseCommand(currentExe)`
- [x] 12. Update `ResetForNewDay` to also reset `GroupUsageState` flags
- [x] 13. Update `GetTrackedExeName` to check if exe is a member of any group (not just in `Apps` dictionary)

## Command Changes
- [x] 14. Update `ShowToastCommand` parameter from `ExeName` to `Label` (group name)

## Worker Changes (AgentWorker.cs)
- [x] 15. Update config poll to map `GroupConfigDto` list → `List<GroupRule>`
- [x] 16. Update `DispatchCommandAsync` for `ShowToastCommand` to use label (group name) in log message

## Notification Sink Changes
- [x] 17. Update `INotificationSink.ShowToast` signature from `(string exeName, ...)` to `(string label, ...)`
- [x] 18. Update `ToastNotificationSink.ShowToast` message from `"{exeName}: {n} minute(s) remaining"` to `"{n} minute(s) remaining for {label}"`

## Integration Fakes Updates
- [x] 19. Update `FakeApiClient` to use `GroupConfigDto`/`Groups` instead of `AppConfigDto`/`Apps`
- [x] 20. Update `FakeNotificationSink` to track label instead of exe name

## Contract Test Updates (GoServerContractTests)
- [x] 21. Update `GetConfig_Returns_SnakeCase_MatchingDto` to use `groups` format and assert on group fields
- [x] 22. Update `PushUsage_Sends_SnakeCase_AcceptedByGoServer` to create groups via new API and assert on `name` field
- [x] 23. Update `GetConfig_EmptyServer_Returns_EmptyArray` to check `Groups` instead of `Apps`

## API Client Tests (AgentApiClientTests)
- [x] 24. Update `GetConfigAsync_Deserializes_SnakeCaseJson` to use `groups` JSON format
- [x] 25. Update `GetConfigAsync_Returns_EmptyList_For_EmptyApps` to check `Groups`
- [x] 26. Update `GetConfigAsync_Deserializes_TestPopupAt` to use `groups` format

## Core Engine Tests
- [x] 27. Update `NotificationTests` to use `GroupRule` and verify toast uses group name
- [x] 28. Update `EnforcementTests` to use `GroupRule` and verify ForceClose uses current exe name
- [x] 29. Update `TrackingTests` to use `GroupRule`
- [x] 30. Update `SyncTests` to use `GroupRule`
- [x] 31. Update `ResetTests` to use `GroupRule`
- [x] 32. Update `PersistenceTests` to use `GroupRule` and include `GroupUsageState`

## Integration Tests (WorkerIntegrationTests)
- [x] 33. Update all worker tests to use `GroupConfigDto` and `GroupRule`

## New Tests
- [x] 34. Add test: two processes in one group, usage from both counts toward shared budget
- [x] 35. Add test: notification message uses group name (not exe name)
- [x] 36. Add test: enforcement force-closes the current foreground exe (not the group name)
- [x] 37. Add test: process in multiple groups, each group's budget is checked independently
- [x] 38. Add test: `GroupUsageState` resets on new day
- [x] 39. Add test: `GroupUsageState` persists through JSON round-trip

## JsonStateStore
- [x] 40. Update `JsonStateStoreTests` to include `GroupRule` and `GroupUsageState` in round-trip tests
