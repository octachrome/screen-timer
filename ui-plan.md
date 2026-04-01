# Screen Timer — MVP Frontend UI Plan

## Overview

A single-page web app served as static files from `server/static/`. No build step — plain HTML + vanilla TypeScript (compiled to JS) or plain JS with a minimal CSS library. The UI lets the manager add/remove tracked apps, set budgets, and view today's usage.

Given the MVP scope is small (3 features, 5 API calls), a framework like React/Svelte is overkill. Use **vanilla JS** with a single HTML file and a small CSS library for clean styling.

---

## Technology Choices

| Concern | Choice | Rationale |
|---------|--------|-----------|
| Markup | Single `index.html` | Simplest delivery; served by Go's `http.FileServer` |
| Scripting | Vanilla JS (`app.js`) | No build step; MVP has ~3 interactions |
| Styling | [Pico CSS](https://picocss.com/) (classless/minimal) | Clean defaults with zero config; CDN link |
| HTTP | `fetch()` API | Built-in, no dependencies |

---

## UI Layout

Single page with two sections:

### Section 1: "Add Application"

A form with two inputs and a submit button:
- **Executable name** — text input (e.g., `Fortnite.exe`)
- **Daily budget (minutes)** — number input
- **Add** button

On submit: `POST /api/apps` → refresh the app list.

### Section 2: "Tracked Applications"

A table (or card list) showing all tracked apps. Each row displays:

| Column | Source |
|--------|--------|
| Executable name | `exe_name` |
| Budget | `daily_budget_minutes` min |
| Used today | `used_today_minutes` min |
| Remaining | `remaining_minutes` min (highlight red when ≤ 0) |
| Actions | **Edit budget** (inline or modal), **Delete** button |

Data source: `GET /api/usage/today` (returns `UsageSummary[]` which has all needed fields).

---

## Interactions

### Add App
1. User fills in exe name + budget, clicks "Add".
2. `POST /api/apps` with `{ "exe_name": "...", "daily_budget_minutes": N }`.
3. On success (201): clear form, re-fetch and re-render the table.
4. On error (409 duplicate, 400 validation): show inline error message.

### Edit Budget
1. User clicks "Edit" on a row.
2. Budget cell becomes an editable number input (inline editing).
3. User changes value, presses Enter or clicks "Save".
4. `PUT /api/apps/{exe}` with `{ "daily_budget_minutes": N }`.
5. On success: re-render row with new values.

### Delete App
1. User clicks "Delete" on a row.
2. Confirmation prompt (`confirm()`).
3. `DELETE /api/apps/{exe}`.
4. On success (204): remove row from table.

### Auto-refresh
- Poll `GET /api/usage/today` every **30 seconds** to keep usage numbers current.
- Show a small "last updated" timestamp.

---

## File Structure

```
server/static/
├── index.html      # Single page: markup + CDN CSS link + <script src="app.js">
├── app.js          # All application logic (~150-200 lines)
└── style.css       # Minimal overrides/custom styles (if needed beyond Pico)
```

---

## Implementation Phases

### Phase 1: Scaffold & Read-only view
1. Create `index.html` with Pico CSS CDN, page structure (heading, form placeholder, table placeholder).
2. Create `app.js` with:
   - `fetchUsage()` — calls `GET /api/usage/today`, returns JSON.
   - `renderTable(summaries)` — builds the table DOM from the array.
   - On page load: fetch and render.
3. Create `style.css` with any minor overrides (e.g., red text for zero remaining).

### Phase 2: Add App form
1. Wire up the "Add Application" form.
2. On submit: call `POST /api/apps`, handle success/error, re-fetch table.

### Phase 3: Delete App
1. Add a "Delete" button to each table row.
2. On click: `confirm()` → `DELETE /api/apps/{exe}` → re-fetch table.

### Phase 4: Edit Budget (inline)
1. Add an "Edit" button to each row.
2. On click: replace budget cell with `<input type="number">` + Save/Cancel.
3. On save: `PUT /api/apps/{exe}` → re-fetch row.

### Phase 5: Polish
1. Auto-refresh polling (30s interval).
2. "Last updated" timestamp display.
3. Visual indicator for exhausted budgets (remaining = 0).
4. Empty state messaging ("No applications tracked yet").
5. Loading state while fetching.

---

## API Calls Summary

| Action | Method | Endpoint | Request Body |
|--------|--------|----------|-------------|
| Load usage table | `GET` | `/api/usage/today` | — |
| Add app | `POST` | `/api/apps` | `{ "exe_name", "daily_budget_minutes" }` |
| Edit budget | `PUT` | `/api/apps/{exe}` | `{ "daily_budget_minutes" }` |
| Delete app | `DELETE` | `/api/apps/{exe}` | — |

`GET /api/apps` is not needed separately since `GET /api/usage/today` returns all the data the UI requires (budget + usage + remaining).
