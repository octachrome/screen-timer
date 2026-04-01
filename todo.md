# Screen Timer — UI Implementation TODO

## Phase 1: Scaffold & Read-only View
- [x] 1. Create `index.html` with Pico CSS CDN, page structure (heading, form, table placeholders)
- [x] 2. Create `app.js` with `fetchUsage()` and `renderTable()` — load & display tracked apps on page load
- [x] 3. Create `style.css` with red-text styling for exhausted budgets

## Phase 2: Add App Form
- [x] 4. Wire up "Add Application" form — POST /api/apps, handle success (clear form, re-fetch), handle errors (inline message)

## Phase 3: Delete App
- [x] 5. Add "Delete" button to each table row — confirm() → DELETE /api/apps/{exe} → re-fetch table

## Phase 4: Edit Budget (Inline)
- [x] 6. Add "Edit" button to each table row — replace budget cell with number input + Save/Cancel
- [x] 7. On save: PUT /api/apps/{exe} → re-fetch table

## Phase 5: Polish
- [x] 8. Auto-refresh polling every 30 seconds
- [x] 9. "Last updated" timestamp display
- [x] 10. Empty state messaging ("No applications tracked yet")
- [x] 11. Loading state while fetching
- [x] 12. Add `class="secondary outline"` to Delete buttons to visually distinguish from Edit
- [x] 13. Visual polish and final review
