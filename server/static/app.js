/**
 * Screen Timer — MVP Frontend
 *
 * Single-page vanilla JS app (no framework, no build step) that lets a
 * manager add/remove tracked applications, set daily screen-time budgets,
 * and view today's usage.  Served as a static file by the Go backend.
 *
 * API endpoints used:
 *   GET    /api/usage/today   — load all tracked apps with budget + usage
 *   POST   /api/apps          — add a new tracked app
 *   PUT    /api/apps/{exe}    — update an app's daily budget
 *   DELETE /api/apps/{exe}    — remove a tracked app
 */
'use strict';

// ---------------------------------------------------------------------------
// Data fetching
// ---------------------------------------------------------------------------

/**
 * Fetch today's usage summaries from the server.
 * Returns an array of UsageSummary objects, each containing:
 *   exe_name, daily_budget_minutes, used_today_minutes, remaining_minutes
 */
async function fetchUsage() {
  const res = await fetch('/api/usage/today');
  return res.json();
}

// ---------------------------------------------------------------------------
// Rendering — "Tracked Applications" table (Section 2 of the UI)
// ---------------------------------------------------------------------------

/**
 * Build and insert the tracked-apps table into the #app-list container.
 * Each row shows: Executable | Budget | Used Today | Remaining | Actions.
 * When remaining ≤ 0 the cell is highlighted with the "exhausted" class.
 * An empty-state message is shown when there are no tracked apps.
 */
function renderTable(summaries) {
  const container = document.getElementById('app-list');
  container.innerHTML = '';

  // Empty state — no apps tracked yet
  if (summaries.length === 0) {
    const p = document.createElement('p');
    p.className = 'empty-state';
    p.textContent = 'No applications tracked yet.';
    container.appendChild(p);
    return;
  }

  const table = document.createElement('table');
  const thead = document.createElement('thead');
  thead.innerHTML = `<tr>
    <th>Executable</th>
    <th>Budget</th>
    <th>Used Today</th>
    <th>Remaining</th>
    <th>Actions</th>
  </tr>`;
  table.appendChild(thead);

  const tbody = document.createElement('tbody');
  for (const row of summaries) {
    const tr = document.createElement('tr');

    const tdExe = document.createElement('td');
    tdExe.textContent = row.exe_name;

    const tdBudget = document.createElement('td');
    tdBudget.textContent = `${row.daily_budget_minutes} min`;

    const tdUsed = document.createElement('td');
    tdUsed.textContent = `${row.used_today_minutes} min`;

    // Highlight remaining time red when budget is exhausted (≤ 0)
    const tdRemaining = document.createElement('td');
    tdRemaining.textContent = `${row.remaining_minutes} min`;
    if (row.remaining_minutes <= 0) {
      tdRemaining.classList.add('exhausted');
    }

    // Actions: Edit (inline budget editing) and Delete
    const tdActions = document.createElement('td');

    const editBtn = document.createElement('button');
    editBtn.textContent = 'Edit';
    editBtn.addEventListener('click', () => {
      startEdit(tdBudget, row.exe_name, row.daily_budget_minutes);
    });

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'secondary outline';
    deleteBtn.textContent = 'Delete';
    deleteBtn.addEventListener('click', () => {
      deleteApp(row.exe_name);
    });

    tdActions.appendChild(editBtn);
    tdActions.appendChild(deleteBtn);

    tr.appendChild(tdExe);
    tr.appendChild(tdBudget);
    tr.appendChild(tdUsed);
    tr.appendChild(tdRemaining);
    tr.appendChild(tdActions);
    tbody.appendChild(tr);
  }

  table.appendChild(tbody);
  container.appendChild(table);
}

// ---------------------------------------------------------------------------
// Data refresh & "last updated" timestamp
// ---------------------------------------------------------------------------

/**
 * Re-fetch usage data and re-render the table.
 * Shows a loading indicator on the first load (when the container is empty).
 * Updates the "last updated" timestamp after every successful fetch.
 */
async function refreshData() {
  const container = document.getElementById('app-list');

  // Show a loading state only on the initial page load
  if (!container.hasChildNodes()) {
    container.setAttribute('aria-busy', 'true');
    container.textContent = 'Loading…';
  }

  const summaries = await fetchUsage();
  container.removeAttribute('aria-busy');
  renderTable(summaries);

  // Update the "last updated" timestamp (HH:MM:SS)
  const now = new Date();
  const timestamp = now.toTimeString().split(' ')[0];
  const el = document.getElementById('last-updated');
  if (el) {
    el.textContent = `Last updated: ${timestamp}`;
  }
}

// ---------------------------------------------------------------------------
// Delete App — DELETE /api/apps/{exe}
// ---------------------------------------------------------------------------

/**
 * Prompt the user for confirmation, then delete the tracked app.
 * On success (204) the table is refreshed to remove the row.
 */
async function deleteApp(exeName) {
  if (!confirm(`Delete ${exeName}?`)) {
    return;
  }
  await fetch(`/api/apps/${encodeURIComponent(exeName)}`, { method: 'DELETE' });
  await refreshData();
}

// ---------------------------------------------------------------------------
// Inline Edit Budget — PUT /api/apps/{exe}
// ---------------------------------------------------------------------------

/**
 * Replace the budget cell with a number input + Save/Cancel buttons so the
 * user can edit the daily budget inline without leaving the page.
 * On save: sends PUT /api/apps/{exe} with the new budget value.
 * On cancel: simply re-renders the table to restore the original cell.
 */
function startEdit(budgetCell, exeName, currentValue) {
  budgetCell.innerHTML = '';

  const input = document.createElement('input');
  input.type = 'number';
  input.min = '1';
  input.value = currentValue;

  const saveBtn = document.createElement('button');
  saveBtn.textContent = 'Save';
  saveBtn.addEventListener('click', async () => {
    await fetch(`/api/apps/${encodeURIComponent(exeName)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ daily_budget_minutes: parseInt(input.value, 10) }),
    });
    await refreshData();
  });

  const cancelBtn = document.createElement('button');
  cancelBtn.textContent = 'Cancel';
  cancelBtn.addEventListener('click', () => {
    refreshData();
  });

  budgetCell.appendChild(input);
  budgetCell.appendChild(saveBtn);
  budgetCell.appendChild(cancelBtn);
}

// ---------------------------------------------------------------------------
// Initialization — "Add Application" form (Section 1) + auto-refresh
// ---------------------------------------------------------------------------

document.addEventListener('DOMContentLoaded', () => {
  /**
   * Add App form handler — POST /api/apps
   * Reads exe_name and daily_budget_minutes from the form inputs.
   * On success (201): clears the form and refreshes the table.
   * On error (409 duplicate / 400 validation): shows an inline error message.
   */
  document.getElementById('add-app-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const exeInput = document.getElementById('exe-name');
    const budgetInput = document.getElementById('daily-budget');
    const formError = document.getElementById('form-error');

    const body = {
      exe_name: exeInput.value,
      daily_budget_minutes: parseInt(budgetInput.value, 10),
    };

    const res = await fetch('/api/apps', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (res.status === 201) {
      exeInput.value = '';
      budgetInput.value = '';
      formError.textContent = '';
      await refreshData();
    } else {
      const data = await res.json();
      formError.textContent = data.error;
    }
  });

  // Initial data load
  refreshData();

  // Auto-refresh: poll usage data every 30 seconds to keep numbers current
  setInterval(refreshData, 30000);
});
