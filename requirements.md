# Screen Timer — Detailed Requirements

## Overview

A screen time management system for Windows, allowing a parent (manager) to set per-application daily time budgets for a child, monitor usage remotely via a web UI, and enforce limits on the child's PC.

## Users

- **Manager (parent):** Configures limits and monitors usage remotely via a web interface.
- **Child:** Uses the PC normally; sees remaining time and receives notifications as limits approach.

---

## Functional Requirements

### F1: Application Time Tracking

- The Windows agent tracks time spent in specific applications, identified by executable name (e.g., `Fortnite.exe`).
- Time is only tracked while the application window is **in focus** (foreground).
- Usage data is recorded per-application, per-day.
- Historical usage data is exported/stored as a **CSV file** (no UI required for history).

### F2: Daily Time Budgets

- Each tracked application has a daily time budget (e.g., 2 hours).
- **Separate default budgets** can be configured for **weekdays** (Mon–Fri) and **weekends** (Sat–Sun).
- The manager can **extend the budget for the current day** on an ad-hoc basis (e.g., add 30 minutes today only), without changing the default.

### F3: Manager Web Interface

- A web UI served from a remote backend (hosted on EC2).
- **No authentication** required.
- The manager can:
  - **Add/remove tracked applications** (by executable name).
  - **Configure weekday and weekend default budgets** per application.
  - **Extend the current day's budget** for any application.
  - **View today's usage** per application (time used vs. budget remaining).

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
