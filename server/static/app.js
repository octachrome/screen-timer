'use strict';

async function fetchUsage() {
  const res = await fetch('/api/usage/today');
  return res.json();
}

function renderTable(summaries) {
  const container = document.getElementById('app-list');
  container.innerHTML = '';

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

    const tdRemaining = document.createElement('td');
    tdRemaining.textContent = `${row.remaining_minutes} min`;
    if (row.remaining_minutes <= 0) {
      tdRemaining.classList.add('exhausted');
    }

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

async function refreshData() {
  const container = document.getElementById('app-list');
  if (!container.hasChildNodes()) {
    container.setAttribute('aria-busy', 'true');
    container.textContent = 'Loading…';
  }

  const summaries = await fetchUsage();
  container.removeAttribute('aria-busy');
  renderTable(summaries);

  const now = new Date();
  const timestamp = now.toTimeString().split(' ')[0];
  const el = document.getElementById('last-updated');
  if (el) {
    el.textContent = `Last updated: ${timestamp}`;
  }
}

async function deleteApp(exeName) {
  if (!confirm(`Delete ${exeName}?`)) {
    return;
  }
  await fetch(`/api/apps/${encodeURIComponent(exeName)}`, { method: 'DELETE' });
  await refreshData();
}

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

document.addEventListener('DOMContentLoaded', () => {
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

  refreshData();
  setInterval(refreshData, 30000);
});
