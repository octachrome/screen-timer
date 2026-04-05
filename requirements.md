# Screen Timer — Detailed Requirements

## Overview

A screen time management system for Windows, allowing a parent (manager) to set per-application daily time budgets for a child, monitor usage remotely via a web UI, and enforce limits on the child's PC.

## Users

- **Manager (parent):** Configures limits and monitors usage remotely via a web interface.
- **Child:** Uses the PC normally; sees remaining time and receives notifications as limits approach.

---

## Functional Requirements

### F1: Application Time Tracking

- The Windows agent tracks time spent in specific applications, identified by process name (e.g., `Fortnite`).
- Time is only tracked while the application window is **in focus** (foreground).
- Usage data is recorded per-application, per-day.
- Historical usage data is exported/stored as a **CSV file** (no UI required for history).

### F2: Daily Time Budgets

- Budgets can be set at three levels, from most specific to least:
  1. **Per-application** — a budget for an individual application (e.g., `Fortnite`).
  2. **Per-group** — a budget for a named group of applications (e.g., a "Games" group containing `Fortnite` and `Minecraft`). An application can belong to multiple groups.
  3. **"All" group** — a built-in group that always exists and covers all tracked applications, setting an overall daily screen time budget.
- When multiple budgets apply to an application, **all** of them are enforced independently. For example, if "Games" has a 2-hour budget and "All" has a 4-hour budget, a game is blocked when *either* budget is exhausted.
- **Separate default budgets** can be configured for **weekdays** (Mon–Fri) and **weekends** (Sat–Sun) at each level (application, group, and "All").
- The manager can **extend the budget for the current day** on an ad-hoc basis (e.g., add 30 minutes today only) for any application, group, or "All", without changing the default.

### F3: Manager Web Interface

- A web UI served from a remote backend (hosted on EC2).
- **No authentication** required.
- The manager can:
  - **Add/remove tracked applications** (by executable name).
  - **Create/edit/delete groups** of applications (assign applications to named groups).
  - **Configure weekday and weekend default budgets** per application, per group, and for the "All" group.
  - **Extend the current day's budget** for any application, group, or "All".
  - **View today's usage** per application and per group (time used vs. budget remaining).

### F4: Child Notifications

- Notifications appear when there are **10 minutes, 5 minutes, and 1 minute** remaining on an application's budget.
- Notifications must be:
  - Clearly visible but **unobtrusive**.
  - Must **not steal keyboard focus** from the active application.
  - **Visible even when a game is in full-screen mode**.

### F5: Enforcement on Budget Expiry

- When an application's daily budget is exhausted, the application should be **made unusable**.
- **[OPEN QUESTION]** The exact enforcement mechanism needs further investigation. Options include:
  - Force-closing the application.
  - Minimizing and preventing re-focus.
  - Covering the window with an unmovable overlay.
  - Other approaches — to be decided after experimentation.

### F6: Persistent Time Remaining Display

- **[OPEN QUESTION / EXPERIMENT NEEDED]** Ideally, a persistent on-screen overlay widget shows remaining time for the currently focused tracked application.
- Windows overlays can be inconsistent with full-screen games; **experiments are needed** to determine feasibility.
- **Fallback:** If a reliable overlay is not achievable, use pop-up notifications only (at the 10/5/1 minute marks per F4).

---

## Non-Functional Requirements

### NF1: Architecture

| Component        | Technology                          | Hosting              |
|------------------|-------------------------------------|----------------------|
| Web backend      | **Go** (lightweight on CPU/memory)  | EC2 instance         |
| Web frontend     | **TypeScript** with a reactive framework (e.g., React, Svelte, Vue) + minimal CSS library | Served by Go backend |
| Windows agent    | **C#** (.NET, modern)               | Child's PC           |
| Low-level hooks  | **C/C++** only if needed for overlay/notification APIs that require lower-level Windows access; kept minimal | Child's PC (called from C#) |

### NF2: Windows Agent Startup

- The Windows agent must **auto-start on boot** (e.g., via Windows Task Scheduler or startup registry entry).
- It runs as a background process.

### NF3: Tamper Resistance

- **Not required.** No special measures needed to prevent the child from closing the agent.

### NF4: Communication

- The Windows agent communicates with the web backend to:
  - **Pull** application configuration and budgets.
  - **Push** usage data.
- Exact protocol TBD (HTTP polling or WebSocket are likely candidates).

---

## Open Questions

1. **Enforcement mechanism (F5):** How exactly should an application be made unusable when its budget runs out? Needs experimentation.
2. **Overlay feasibility (F6):** Can a persistent overlay reliably display over full-screen games on Windows? Needs experimentation to determine approach or whether to fall back to notifications only.
