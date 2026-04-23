/**
 * Screen Timer — MVP Frontend
 *
 * Single-page vanilla JS app (no framework, no build step) that lets a
 * manager add/remove tracked applications, set daily screen-time budgets,
 * and view today's usage.  Served as a static file by the Go backend.
 *
 * API endpoints used:
 *   GET    /api/usage/today   — load all tracked groups with budget + usage
 *   POST   /api/apps          — add a new tracked group
 *   PUT    /api/apps/{name}   — update a group's daily budget and processes
 *   DELETE /api/apps/{name}   — remove a tracked group
 */
'use strict';

// ---------------------------------------------------------------------------
// Data fetching
// ---------------------------------------------------------------------------

/**
 * Fetch today's usage summaries from the server.
 * Returns an array of UsageSummary objects, each containing:
 *   name, processes, daily_budget_minutes, used_today_minutes, remaining_minutes
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
    <th>Name</th>
    <th>Processes</th>
    <th>Weekday Budget</th>
    <th>Weekend Budget</th>
    <th>Used Today</th>
    <th>Remaining</th>
    <th>Actions</th>
  </tr>`;
  table.appendChild(thead);

  const tbody = document.createElement('tbody');
  for (const row of summaries) {
    const tr = document.createElement('tr');

    const tdName = document.createElement('td');
    tdName.textContent = row.name;

    const tdProcesses = document.createElement('td');
    tdProcesses.textContent = row.processes.join(', ');

    const tdWeekday = document.createElement('td');
    tdWeekday.textContent = `${row.daily_budget_minutes} min`;

    const tdWeekend = document.createElement('td');
    tdWeekend.textContent = `${row.weekend_budget_minutes} min`;

    // Highlight active budget column based on today's day
    const today = new Date().getDay();
    const isWeekend = (today === 0 || today === 6);
    if (isWeekend) {
      tdWeekend.style.fontWeight = 'bold';
    } else {
      tdWeekday.style.fontWeight = 'bold';
    }

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
      startEdit(tdName, tdWeekday, tdWeekend, tdProcesses, row.name, row.daily_budget_minutes, row.weekend_budget_minutes, row.processes);
    });

    const deleteBtn = document.createElement('button');
    deleteBtn.className = 'secondary outline';
    deleteBtn.textContent = 'Delete';
    deleteBtn.addEventListener('click', () => {
      deleteApp(row.name);
    });

    tdActions.appendChild(editBtn);
    tdActions.appendChild(deleteBtn);

    tr.appendChild(tdName);
    tr.appendChild(tdProcesses);
    tr.appendChild(tdWeekday);
    tr.appendChild(tdWeekend);
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
// Delete App — DELETE /api/apps/{name}
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
// Inline Edit — PUT /api/apps/{name}
// ---------------------------------------------------------------------------

/**
 * Replace the name, budget and processes cells with inputs + Save/Cancel
 * buttons so the user can edit inline without leaving the page.
 * On save: sends PUT /api/apps/{name} with the new name, budget and processes.
 * On cancel: simply re-renders the table to restore the original cells.
 */
function startEdit(nameCell, weekdayCell, weekendCell, processesCell, groupName, currentWeekdayBudget, currentWeekendBudget, currentProcesses) {
  nameCell.innerHTML = '';
  weekdayCell.innerHTML = '';
  weekendCell.innerHTML = '';
  processesCell.innerHTML = '';

  const nameInput = document.createElement('input');
  nameInput.type = 'text';
  nameInput.value = groupName;
  nameCell.appendChild(nameInput);

  const weekdayInput = document.createElement('input');
  weekdayInput.type = 'number';
  weekdayInput.min = '1';
  weekdayInput.value = currentWeekdayBudget;
  weekdayCell.appendChild(weekdayInput);

  const weekendInput = document.createElement('input');
  weekendInput.type = 'number';
  weekendInput.min = '1';
  weekendInput.value = currentWeekendBudget;
  weekendCell.appendChild(weekendInput);

  const processesInput = document.createElement('input');
  processesInput.type = 'text';
  processesInput.placeholder = 'e.g. Fortnite, Minecraft';
  processesInput.value = currentProcesses.join(', ');
  processesCell.appendChild(processesInput);

  const saveBtn = document.createElement('button');
  saveBtn.textContent = 'Save';
  saveBtn.addEventListener('click', async () => {
    const processes = processesInput.value.split(',').map(s => s.trim()).filter(s => s.length > 0);
    await fetch(`/api/apps/${encodeURIComponent(groupName)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: nameInput.value.trim(),
        daily_budget_minutes: parseInt(weekdayInput.value, 10),
        weekend_budget_minutes: parseInt(weekendInput.value, 10),
        processes,
      }),
    });
    await refreshData();
  });

  const cancelBtn = document.createElement('button');
  cancelBtn.textContent = 'Cancel';
  cancelBtn.addEventListener('click', () => {
    refreshData();
  });

  weekdayCell.appendChild(saveBtn);
  weekdayCell.appendChild(cancelBtn);
}

// ---------------------------------------------------------------------------
// Initialization — "Add Application" form (Section 1) + auto-refresh
// ---------------------------------------------------------------------------

document.addEventListener('DOMContentLoaded', () => {
  /**
   * Add App form handler — POST /api/apps
   * Reads name, process and daily_budget_minutes from the form inputs.
   * On success (201): clears the form and refreshes the table.
   * On error (409 duplicate / 400 validation): shows an inline error message.
   */
  document.getElementById('add-app-form').addEventListener('submit', async (e) => {
    e.preventDefault();

    const exeInput = document.getElementById('exe-name');
    const budgetInput = document.getElementById('daily-budget');
    const formError = document.getElementById('form-error');

    const processes = exeInput.value.split(',').map(s => s.trim()).filter(s => s.length > 0);
    const diffWeekend = document.getElementById('diff-weekend').checked;
    const weekendInput = document.getElementById('weekend-budget');

    const body = {
      name: processes.join(', '),
      processes,
      daily_budget_minutes: parseInt(budgetInput.value, 10),
    };
    if (diffWeekend) {
      body.weekend_budget_minutes = parseInt(weekendInput.value, 10);
    }

    const res = await fetch('/api/apps', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (res.status === 201) {
      exeInput.value = '';
      budgetInput.value = '';
      document.getElementById('diff-weekend').checked = false;
      document.getElementById('weekend-budget-group').style.display = 'none';
      weekendInput.value = '';
      formError.textContent = '';
      await refreshData();
    } else {
      const data = await res.json();
      formError.textContent = data.error;
    }
  });

  document.getElementById('test-popup-btn').addEventListener('click', async () => {
    const status = document.getElementById('test-popup-status');
    try {
      const res = await fetch('/api/agent/test-popup', { method: 'POST' });
      if (res.ok) {
        status.textContent = 'Test popup sent!';
        status.className = 'success-msg';
      } else {
        status.textContent = 'Failed to send test popup';
        status.className = 'error-msg';
      }
    } catch (e) {
      status.textContent = 'Network error';
      status.className = 'error-msg';
    }
    setTimeout(() => { status.textContent = ''; }, 3000);
  });

  document.getElementById('diff-weekend').addEventListener('change', (e) => {
    document.getElementById('weekend-budget-group').style.display = e.target.checked ? '' : 'none';
  });

  // Initial data load
  refreshData();

  // Auto-refresh: poll usage data every 30 seconds to keep numbers current
  setInterval(refreshData, 30000);
});
