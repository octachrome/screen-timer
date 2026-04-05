# Phase 2 — Web UI Changes: Todo List

## Add Form (unchanged UX, different payload)
- [x] 1. Update POST payload to send `{ name, process, daily_budget_minutes }` instead of `{ exe_name, daily_budget_minutes }`

## Tracked Applications Table
- [x] 2. Rename table column header "Process" → "Name"
- [x] 3. Add a "Processes" column showing comma-separated member process names
- [x] 4. Update `renderTable()` to read `name` and `processes` from the new summary shape (replace `row.exe_name` → `row.name`)
- [x] 5. Update Edit button to pass group name instead of exe_name
- [x] 6. Update Delete button to pass group name instead of exe_name

## Inline Edit Expansion
- [x] 7. Expand `startEdit()` to show a processes text field alongside the budget input
- [x] 8. Save button sends `PUT /api/apps/{name}` with `{ daily_budget_minutes, processes: [...] }`
- [x] 9. Add CSS styling for the inline edit form (processes field layout)

## Delete
- [x] 10. Update `deleteApp()` to use group name in confirmation prompt and API call

## Data Fetching
- [x] 11. Verify `fetchUsage()` works with new response shape (name, processes fields)

## Add Form — Multiple Processes
- [x] 21. Change `AddGroupRequest.Process` (string) → `Processes` ([]string) across server, mockclient, tests
- [x] 22. Rename label "Process name" → "Process names", placeholder "e.g. Fortnite, Minecraft"
- [x] 23. Parse comma-separated input, set group name to the joined process names

## Group Rename
- [x] 15. Add `name` field to `UpdateGroupRequest` on server
- [x] 16. Implement rename logic in `store.UpdateGroup` (with conflict detection)
- [x] 17. Make name cell editable in `startEdit()`
- [x] 18. Add placeholder text to processes input (`e.g. Fortnite, Minecraft`)

## Tests
- [x] 12. Add integration test: POST /api/apps with name+process payload, verify response has `name` and `processes` fields
- [x] 13. Add integration test: PUT /api/apps/{name} with processes list, verify processes are updated
- [x] 14. Add integration test: full round-trip — add group, edit processes, verify usage shows updated process list
- [x] 19. Add handler test: PUT rename returns new name, old name gone
- [x] 20. Add handler test: PUT rename conflict returns 409
