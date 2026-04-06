// DDNS Updater Interactive WebUI
(function () {
  'use strict';

  let providers = {};
  let configEntries = [];
  let editIndex = -1;
  let deleteIndex = -1;

  function $(sel) { return document.querySelector(sel); }
  function $$(sel) { return document.querySelectorAll(sel); }

  function showToast(msg) {
    const toast = $('#toast');
    toast.textContent = msg;
    toast.style.display = 'block';
    toast.style.opacity = '1';
    setTimeout(() => {
      toast.style.opacity = '0';
      setTimeout(() => { toast.style.display = 'none'; }, 300);
    }, 2500);
  }

  async function api(method, path, body) {
    const opts = { method, headers: {} };
    if (body) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const res = await fetch(path, opts);
    if (res.status === 204) return null;
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Request failed');
    return data;
  }

  function initTabs() {
    $$('.tab').forEach(tab => {
      tab.addEventListener('click', () => {
        $$('.tab').forEach(t => t.classList.remove('active'));
        $$('.tab-content').forEach(tc => tc.classList.remove('active'));
        tab.classList.add('active');
        const target = tab.dataset.tab;
        $('#' + target).classList.add('active');
        window.location.hash = target;
        if (target === 'dashboard') loadStatus();
        if (target === 'configuration') loadConfig();
      });
    });
    const hash = window.location.hash.slice(1);
    if (hash === 'configuration') {
      $$('.tab').forEach(t => t.classList.remove('active'));
      $$('.tab-content').forEach(tc => tc.classList.remove('active'));
      $('[data-tab="configuration"]').classList.add('active');
      $('#configuration').classList.add('active');
    }
  }

  function statusClass(status) {
    const s = (status || '').toLowerCase();
    if (s === 'success') return 'success';
    if (s === 'fail') return 'fail';
    if (s === 'uptodate') return 'uptodate';
    if (s === 'updating') return 'updating';
    return 'unset';
  }

  function statusLabel(status) {
    const s = (status || '').toLowerCase();
    if (s === 'success') return 'Success';
    if (s === 'fail') return 'Failure';
    if (s === 'uptodate') return 'Up to date';
    if (s === 'updating') return 'Updating';
    return 'Unset';
  }

  function renderRecords(records) {
    const grid = $('#records-grid');
    if (!records || records.length === 0) {
      grid.innerHTML = '<p class="loading">No DNS records configured.</p>';
      return;
    }
    grid.innerHTML = records.map(rec => {
      const prevIPs = (rec.previous_ips || []).slice(0, 3).join(', ') || 'N/A';
      const ipLink = rec.current_ip
        ? '<a href="https://ipinfo.io/' + rec.current_ip + '" target="_blank">' + rec.current_ip + '</a>'
        : 'N/A';
      const sc = statusClass(rec.status);
      const timeAgo = rec.last_updated ? timeSince(rec.last_updated) : '';
      return '<div class="card">' +
        '<div class="card-header">' +
          '<span class="card-domain">' + escHtml(rec.domain) + '</span>' +
          '<span class="badge badge-provider">' + escHtml(rec.provider) + '</span>' +
        '</div>' +
        '<div class="card-body">' +
          '<div class="card-row"><span class="card-label">Owner</span><span class="card-value">' + escHtml(rec.owner) + '</span></div>' +
          '<div class="card-row"><span class="card-label">IP Version</span><span class="badge">' + escHtml(rec.ip_version) + '</span></div>' +
          '<div class="card-row"><span class="card-label">Current IP</span><span class="card-value">' + ipLink + '</span></div>' +
          '<div class="card-row"><span class="card-label">Previous IPs</span><span class="card-value">' + escHtml(prevIPs) + '</span></div>' +
        '</div>' +
        '<div class="card-footer">' +
          '<span class="status-dot ' + sc + '"></span>' +
          '<span class="status-text">' + statusLabel(rec.status) +
            (rec.message ? ' (' + escHtml(rec.message) + ')' : '') +
            (timeAgo ? ' &middot; ' + timeAgo : '') +
          '</span>' +
        '</div>' +
      '</div>';
    }).join('');
  }

  function timeSince(isoStr) {
    const diff = Date.now() - new Date(isoStr).getTime();
    const secs = Math.floor(diff / 1000);
    if (secs < 60) return secs + 's ago';
    const mins = Math.floor(secs / 60);
    if (mins < 60) return mins + 'm ago';
    const hours = Math.floor(mins / 60);
    if (hours < 24) return hours + 'h ago';
    return Math.floor(hours / 24) + 'd ago';
  }

  function escHtml(str) {
    const d = document.createElement('div');
    d.textContent = str || '';
    return d.innerHTML;
  }

  async function loadStatus() {
    try {
      const data = await api('GET', 'api/status');
      renderRecords(data.records);
    } catch (e) {
      $('#records-grid').innerHTML = '<p class="loading">Failed to load: ' + escHtml(e.message) + '</p>';
    }
  }

  function renderConfig(settings) {
    const list = $('#config-list');
    if (!settings || settings.length === 0) {
      list.innerHTML = '<p class="loading">No entries configured. Click "+ Add Entry" to get started.</p>';
      return;
    }
    list.innerHTML = settings.map((entry, i) => {
      return '<div class="card">' +
        '<div class="card-header">' +
          '<span class="card-domain">' + escHtml(entry.domain || '') + '</span>' +
          '<div class="card-actions">' +
            '<button class="btn-icon" onclick="window._editEntry(' + i + ')" title="Edit">&#9998;</button>' +
            '<button class="btn-icon danger" onclick="window._deleteEntry(' + i + ')" title="Delete">&#128465;</button>' +
          '</div>' +
        '</div>' +
        '<div class="card-body">' +
          '<div class="card-row"><span class="card-label">Provider</span><span class="badge badge-provider">' + escHtml(entry.provider || '') + '</span></div>' +
          '<div class="card-row"><span class="card-label">IP Version</span><span class="badge">' + escHtml(entry.ip_version || 'ipv4 or ipv6') + '</span></div>' +
        '</div>' +
      '</div>';
    }).join('');
  }

  async function loadConfig() {
    try {
      const data = await api('GET', 'api/config');
      configEntries = data.settings || [];
      renderConfig(configEntries);
    } catch (e) {
      $('#config-list').innerHTML = '<p class="loading">Failed to load: ' + escHtml(e.message) + '</p>';
    }
  }

  async function loadProviders() {
    try {
      const data = await api('GET', 'api/providers');
      providers = data.providers || {};
    } catch (e) {
      console.error('Failed to load providers', e);
    }
  }

  function openModal(title, entry, index) {
    editIndex = index;
    $('#modal-title').textContent = title;
    $('#modal-overlay').style.display = 'flex';

    const sel = $('#provider-select');
    sel.innerHTML = '<option value="">Select a provider...</option>';
    Object.keys(providers).sort().forEach(key => {
      const opt = document.createElement('option');
      opt.value = key;
      opt.textContent = providers[key].name || key;
      sel.appendChild(opt);
    });

    $('#domain-input').value = '';
    $('#ip-version-select').value = 'ipv4 or ipv6';
    $('#ipv6-suffix-input').value = '';
    $('#ipv6-suffix-group').style.display = 'none';
    $('#provider-fields-container').innerHTML = '';
    $('#auth-groups-container').innerHTML = '';

    if (entry) {
      sel.value = entry.provider || '';
      $('#domain-input').value = entry.domain || '';
      $('#ip-version-select').value = entry.ip_version || 'ipv4 or ipv6';
      $('#ipv6-suffix-input').value = entry.ipv6_suffix || '';
      if (entry.provider) renderProviderFields(entry.provider, entry);
    }
    updateIpv6Visibility();
  }

  function closeModal() {
    $('#modal-overlay').style.display = 'none';
    editIndex = -1;
  }

  function updateIpv6Visibility() {
    const v = $('#ip-version-select').value;
    $('#ipv6-suffix-group').style.display = (v === 'ipv6' || v === 'ipv4 or ipv6') ? '' : 'none';
  }

  function renderProviderFields(providerKey, existingEntry) {
    const def = providers[providerKey];
    if (!def) return;

    const fieldsContainer = $('#provider-fields-container');
    const authContainer = $('#auth-groups-container');
    fieldsContainer.innerHTML = '';
    authContainer.innerHTML = '';

    if (def.auth_groups && def.auth_groups.length > 0) {
      let selectedGroup = 0;
      if (existingEntry) {
        for (let g = 0; g < def.auth_groups.length; g++) {
          const group = def.auth_groups[g];
          const hasField = group.fields.some(f => existingEntry[f.name] && existingEntry[f.name] !== '');
          if (hasField) { selectedGroup = g; break; }
        }
      }

      let html = '<fieldset class="auth-group-selector"><legend>Authentication Method</legend>';
      html += '<div class="auth-radio-group">';
      def.auth_groups.forEach((group, i) => {
        html += '<label><input type="radio" name="auth-group" value="' + i + '"' +
          (i === selectedGroup ? ' checked' : '') + '> ' + escHtml(group.name) + '</label>';
      });
      html += '</div>';
      html += '<div class="auth-fields" id="auth-fields"></div>';
      html += '</fieldset>';
      authContainer.innerHTML = html;

      renderAuthFields(def.auth_groups[selectedGroup], existingEntry);

      authContainer.querySelectorAll('input[name="auth-group"]').forEach(radio => {
        radio.addEventListener('change', () => {
          renderAuthFields(def.auth_groups[parseInt(radio.value)], existingEntry);
        });
      });
    }

    fieldsContainer.innerHTML = def.fields.map(f => renderField(f, existingEntry)).join('');
  }

  function renderAuthFields(group, existingEntry) {
    const container = document.getElementById('auth-fields');
    if (!container) return;
    container.innerHTML = group.fields.map(f => renderField(f, existingEntry)).join('');
  }

  function renderField(f, existingEntry) {
    const val = existingEntry ? (existingEntry[f.name] || '') : '';
    const req = f.required ? ' required' : '';

    if (f.type === 'boolean') {
      const checked = val === true || val === 'true' ? ' checked' : '';
      return '<div class="form-group"><label class="checkbox-label">' +
        '<input type="checkbox" data-field="' + f.name + '"' + checked + '> ' + escHtml(f.label) +
        '</label>' +
        (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
        '</div>';
    }

    if (f.type === 'select' && f.options) {
      let opts = f.options.map(o =>
        '<option value="' + escHtml(o) + '"' + (val === o ? ' selected' : '') + '>' + escHtml(o) + '</option>'
      ).join('');
      return '<div class="form-group"><label>' + escHtml(f.label) + '</label>' +
        '<select data-field="' + f.name + '"' + req + '>' +
        '<option value="">Select...</option>' + opts + '</select>' +
        (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
        '</div>';
    }

    const inputType = f.type === 'password' ? 'password' : f.type === 'number' ? 'number' : 'text';
    return '<div class="form-group"><label>' + escHtml(f.label) + '</label>' +
      '<input type="' + inputType + '" data-field="' + f.name + '" value="' + escHtml(String(val)) + '"' +
      (f.placeholder ? ' placeholder="' + escHtml(f.placeholder) + '"' : '') +
      req + '>' +
      (f.help ? '<div class="help-text">' + escHtml(f.help) + '</div>' : '') +
      '</div>';
  }

  function collectFormData() {
    const data = {};
    data.provider = $('#provider-select').value;
    data.domain = $('#domain-input').value;
    const ipv = $('#ip-version-select').value;
    if (ipv !== 'ipv4 or ipv6') data.ip_version = ipv;
    const ipv6s = $('#ipv6-suffix-input').value.trim();
    if (ipv6s) data.ipv6_suffix = ipv6s;

    $$('#entry-form [data-field]').forEach(el => {
      const name = el.dataset.field;
      if (el.type === 'checkbox') {
        if (el.checked) data[name] = true;
      } else if (el.type === 'number' && el.value) {
        data[name] = parseInt(el.value, 10);
      } else if (el.value) {
        data[name] = el.value;
      }
    });
    return data;
  }

  async function saveEntry(e) {
    e.preventDefault();
    const data = collectFormData();
    if (!data.provider || !data.domain) {
      showToast('Provider and domain are required');
      return;
    }
    try {
      if (editIndex >= 0) {
        await api('PUT', 'api/config/' + editIndex, data);
        showToast('Entry updated');
      } else {
        await api('POST', 'api/config', data);
        showToast('Entry added');
      }
      closeModal();
      loadConfig();
      $('#restart-banner').style.display = 'block';
    } catch (e) {
      showToast('Error: ' + e.message);
    }
  }

  function openDeleteDialog(index) {
    deleteIndex = index;
    const entry = configEntries[index];
    $('#delete-message').textContent = 'Delete entry for ' +
      (entry.domain || 'unknown') + ' (' + (entry.provider || 'unknown') + ')?';
    $('#delete-overlay').style.display = 'flex';
  }

  async function confirmDelete() {
    if (deleteIndex < 0) return;
    try {
      await api('DELETE', 'api/config/' + deleteIndex);
      showToast('Entry deleted');
      $('#delete-overlay').style.display = 'none';
      deleteIndex = -1;
      loadConfig();
      $('#restart-banner').style.display = 'block';
    } catch (e) {
      showToast('Error: ' + e.message);
    }
  }

  async function forceUpdate() {
    const btn = $('#force-update-btn');
    btn.disabled = true;
    btn.textContent = 'Updating...';
    try {
      const res = await fetch('update');
      const text = await res.text();
      if (res.ok) {
        showToast(text);
      } else {
        showToast('Update failed');
      }
      loadStatus();
    } catch (e) {
      showToast('Error: ' + e.message);
    } finally {
      btn.disabled = false;
      btn.textContent = 'Force Update All';
    }
  }

  window._editEntry = function (i) {
    openModal('Edit Entry', configEntries[i], i);
  };
  window._deleteEntry = function (i) {
    openDeleteDialog(i);
  };

  async function init() {
    initTabs();
    await loadProviders();

    $('#force-update-btn').addEventListener('click', forceUpdate);
    $('#add-entry-btn').addEventListener('click', () => openModal('Add Entry', null, -1));
    $('#modal-close').addEventListener('click', closeModal);
    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-overlay').addEventListener('click', (e) => {
      if (e.target === $('#modal-overlay')) closeModal();
    });
    $('#entry-form').addEventListener('submit', saveEntry);
    $('#provider-select').addEventListener('change', (e) => {
      renderProviderFields(e.target.value, null);
    });
    $('#ip-version-select').addEventListener('change', updateIpv6Visibility);
    $('#delete-cancel').addEventListener('click', () => {
      $('#delete-overlay').style.display = 'none';
    });
    $('#delete-confirm').addEventListener('click', confirmDelete);

    const hash = window.location.hash.slice(1);
    if (hash === 'configuration') {
      loadConfig();
    } else {
      loadStatus();
    }

    setInterval(() => {
      if ($('#dashboard').classList.contains('active')) loadStatus();
    }, 30000);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
