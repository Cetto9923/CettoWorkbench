/* =============================================================================
 * 文件: web/static/js/permission/members.js
 * 模块: 权限配置
 * 职责: 「角色成员」交互核心：角色卡片 → 成员列表 → 页面权限
 * ============================================================================= */
(function () {
  'use strict';

  const ROLE_LABELS = {
    po: 'PO', sm: 'SM', pm: 'PM', biz: '业务', dev: '开发', qa: '测试', lead: '团队管理', pmo: 'PMO'
  };
  const ROLE_ORDER = ['po', 'sm', 'pm', 'biz', 'dev', 'qa', 'lead', 'pmo'];
  const MANAGED_ROLES = new Set(['lead', 'pmo']);
  const SOURCE_LABELS = {
    user_role_grant: '手工授权',
    profile_preferred: '个人资料',
    role_group_mapping: '组织映射',
    org_auto: '组织映射',
    profile_empty: '兼容来源'
  };

  const state = {
    roleCode: 'pmo',
    page: 1,
    pageSize: 20,
    keyword: '',
    selected: {},
    data: null,
    roles: [],
    addPicker: null,
    scopePicker: null,
    scopeKind: 'dept'
  };

  function ensureStyles() {
    // 刷新本轮权限样式，避免旧 query string 命中缓存。
    [['/static/css/permission/list.css', '20260827-perm-ui3'], ['/static/css/permission/drawer.css', '20260827-perm-ui3']].forEach(function (pair) {
      const link = Array.from(document.querySelectorAll('link[rel="stylesheet"]')).find(function (l) { return (l.getAttribute('href') || '').indexOf(pair[0]) >= 0; });
      if (link) link.setAttribute('href', pair[0] + '?v=' + pair[1]);
    });
    if (!document.querySelector('link[href*="/static/workbench/shared/picker/wb-picker.css"]')) {
      const link = document.createElement('link');
      link.rel = 'stylesheet';
      link.href = '/static/workbench/shared/picker/wb-picker.css?v=20260827-perm-ui3';
      document.head.appendChild(link);
    }
  }

  function apiHeaders() {
    var headers = { Accept: 'application/json', 'X-Requested-With': 'XMLHttpRequest' };
    var csrf = typeof window.getCsrfToken === 'function' ? window.getCsrfToken() : '';
    if (csrf) headers['X-CSRF-Token'] = csrf;
    return headers;
  }

  function isManaged() {
    return !!(state.data && state.data.isManaged) || MANAGED_ROLES.has(state.roleCode);
  }

  function roleLabel() {
    return (state.data && state.data.roleLabel) || ROLE_LABELS[state.roleCode] || state.roleCode;
  }

  function loadRoles() {
    return fetch('/admin/permissions/roles', { headers: apiHeaders() })
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (!json.success) throw new Error(json.message || '加载角色失败');
        const items = (json.data && json.data.items) || [];
        const byCode = {};
        items.forEach(function (r) { byCode[r.code] = r; });
        state.roles = ROLE_ORDER.map(function (code) { return byCode[code]; }).filter(Boolean);
        renderRoleCards();
      })
      .catch(function (err) { window.showToast(err.message || '加载角色失败', 'error'); });
  }

  function renderRoleCards() {
    const host = document.getElementById('permissionSummary');
    if (!host) return;
    host.innerHTML = '';
    state.roles.forEach(function (r) {
      const card = document.createElement('div');
      card.className = 'sum-card' + (state.roleCode === r.code ? ' active' : '');
      card.setAttribute('data-role', r.code);
      card.setAttribute('role', 'button');
      card.tabIndex = 0;
      card.innerHTML =
        '<div class="sum-label">' + window.escapeHtml(r.label || ROLE_LABELS[r.code] || r.code) + '</div>' +
        '<div class="sum-value">' + Number(r.memberCount || 0).toLocaleString('zh-CN') + '</div>' +
        '<div class="sum-tip">' + (MANAGED_ROLES.has(r.code) ? '受控角色' : '当前成员') + '</div>';
      function pick() { selectRole(r.code); }
      card.onclick = pick;
      card.onkeydown = function (e) { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); pick(); } };
      host.appendChild(card);
    });
  }

  function selectRole(code) {
    state.roleCode = code;
    state.page = 1;
    state.keyword = '';
    state.selected = {};
    const kw = document.getElementById('permKeyword');
    if (kw) kw.value = '';
    renderRoleCards();
    loadMembers();
  }

  function loadMembers() {
    const body = document.getElementById('userRows');
    if (!body) return;
    body.innerHTML = '<tr><td colspan="5"><div class="loading">加载 ' + window.escapeHtml(ROLE_LABELS[state.roleCode] || state.roleCode) + ' 成员…</div></td></tr>';
    const qs = 'page=' + state.page + '&pageSize=' + state.pageSize + '&keyword=' + encodeURIComponent(state.keyword || '');
    return fetch('/admin/permissions/roles/' + encodeURIComponent(state.roleCode) + '/members?' + qs, { headers: apiHeaders() })
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (!json.success) throw new Error(json.message || '加载成员失败');
        state.data = json.data || {};
        state.selected = {};
        renderMembers();
        renderPager();
        renderToolbar();
        updateHeadTitle();
      })
      .catch(function (err) {
        body.innerHTML = '<tr><td colspan="5"><div class="empty">' + window.escapeHtml(err.message || '加载成员失败') + '</div></td></tr>';
      });
  }

  function updateHeadTitle() {
    const title = document.getElementById('roleMembersTitle');
    const subtitle = document.getElementById('roleMembersSubtitle');
    const addBtn = document.getElementById('permAddBtn');
    if (!state.data) return;
    if (title) title.textContent = roleLabel() + ' · 成员';
    if (subtitle) subtitle.textContent = state.data.isManaged
      ? '可添加或移除手工授权成员；组织映射成员仅展示，不在这里直接移除。'
      : '当前角色由用户个人资料维护，本页用于查看成员与页面权限。';
    if (addBtn) {
      addBtn.disabled = !state.data.isManaged;
      addBtn.title = state.data.isManaged ? '支持一次选择一人或多人' : '当前角色由用户个人资料维护';
    }
  }

  function sourceView(m) {
    const raw = String(m.source || '').toLowerCase();
    if (raw === 'user_role_grant' || raw.indexOf('grant') >= 0) return { label: '手工授权', cls: 'green' };
    if (raw === 'profile_preferred' || raw.indexOf('profile') >= 0) return { label: '个人资料', cls: '' };
    if (raw.indexOf('mapping') >= 0 || raw.indexOf('org') >= 0) return { label: '组织映射', cls: 'blue' };
    return { label: SOURCE_LABELS[m.source] || m.sourceLabel || m.source || '系统派生', cls: '' };
  }

  function renderMembers() {
    const body = document.getElementById('userRows');
    if (!body) return;
    const members = (state.data && state.data.members) || [];
    if (!members.length) {
      body.innerHTML = '<tr><td colspan="5"><div class="empty">当前条件下暂无成员</div></td></tr>';
      return;
    }
    let html = '';
    members.forEach(function (m) {
      const removable = m.removable !== false && !!state.data.isManaged;
      const source = sourceView(m);
      const first = String(m.realname || m.account || '?').slice(0, 1);
      html += '<tr data-account="' + window.escapeHtml(m.account) + '">' +
        '<td class="check"><input type="checkbox" class="perm-member-cb" data-account="' + window.escapeHtml(m.account) + '"' + (removable ? '' : ' disabled title="仅手工授权成员可批量移除"') + '></td>' +
        '<td><div class="person"><div class="mini-avatar">' + window.escapeHtml(first) + '</div><div style="min-width:0"><div class="name">' + window.escapeHtml(m.realname || m.account) + '</div><div class="sub">' + window.escapeHtml(m.account) + '</div></div></div></td>' +
        '<td>' + window.escapeHtml(m.deptName || '—') + '</td>' +
        '<td><span class="tag ' + source.cls + '">' + window.escapeHtml(source.label) + '</span></td>' +
        '<td><button type="button" class="link" data-act="edit" data-account="' + window.escapeHtml(m.account) + '" data-name="' + window.escapeHtml(m.realname || m.account) + '">编辑</button>' +
        '<button type="button" class="link" data-act="remove" data-account="' + window.escapeHtml(m.account) + '"' + (removable ? '' : ' disabled title="该成员来自组织映射/个人资料，不能在这里移除"') + '>移除</button></td>' +
        '</tr>';
    });
    body.innerHTML = html;
    bindRowEvents();
    syncHeaderCheckbox();
  }

  function bindRowEvents() {
    const body = document.getElementById('userRows');
    if (!body) return;
    body.querySelectorAll('.perm-member-cb:not(:disabled)').forEach(function (cb) {
      cb.onchange = function () {
        const acc = cb.getAttribute('data-account');
        if (cb.checked) state.selected[acc] = true;
        else delete state.selected[acc];
        renderToolbar();
        syncHeaderCheckbox();
      };
    });
    body.querySelectorAll('button[data-act]').forEach(function (btn) {
      btn.onclick = function () {
        if (btn.disabled) return;
        const acc = btn.getAttribute('data-account');
        const act = btn.getAttribute('data-act');
        if (act === 'edit') openUserPagesDrawer(acc, btn.getAttribute('data-name'));
        if (act === 'remove') removeMembers([acc]);
      };
    });
  }

  function syncHeaderCheckbox() {
    const all = document.getElementById('permMemberAll');
    if (!all) return;
    const boxes = Array.from(document.querySelectorAll('#userRows .perm-member-cb:not(:disabled)'));
    const checked = boxes.filter(function (cb) { return cb.checked; });
    all.disabled = boxes.length === 0;
    all.checked = boxes.length > 0 && checked.length === boxes.length;
    all.indeterminate = checked.length > 0 && checked.length < boxes.length;
  }

  function renderToolbar() {
    const bar = document.getElementById('memberBatchBar');
    const cnt = document.getElementById('memberBatchCount');
    const keys = Object.keys(state.selected);
    if (cnt) cnt.textContent = keys.length;
    if (bar) bar.style.display = keys.length ? 'flex' : 'none';
  }

  function clearSelection() {
    state.selected = {};
    document.querySelectorAll('#userRows .perm-member-cb').forEach(function (cb) { cb.checked = false; });
    syncHeaderCheckbox();
    renderToolbar();
  }

  function pagerItems(page, totalPages) {
    if (totalPages <= 7) return Array.from({ length: totalPages }, function (_, i) { return i + 1; });
    const arr = [1];
    const start = Math.max(2, page - 1);
    const end = Math.min(totalPages - 1, page + 1);
    if (start > 2) arr.push('…');
    for (let i = start; i <= end; i++) arr.push(i);
    if (end < totalPages - 1) arr.push('…');
    arr.push(totalPages);
    return arr;
  }

  function renderPager() {
    const host = document.getElementById('memberPager');
    if (!host || !state.data) return;
    const total = Number(state.data.total || 0);
    const size = Number(state.data.pageSize || state.pageSize || 20);
    const page = Math.max(1, Number(state.data.page || state.page || 1));
    const totalPages = Math.max(1, Math.ceil(total / size));
    state.page = Math.min(page, totalPages);
    state.pageSize = size;
    const pageButtons = pagerItems(state.page, totalPages).map(function (p) {
      if (p === '…') return '<span style="padding:0 2px;color:#94a3b8">…</span>';
      return '<button type="button" class="page-btn' + (p === state.page ? ' active' : '') + '" data-page="' + p + '">' + p + '</button>';
    }).join('');
    host.innerHTML =
      '<span>共 <b>' + total.toLocaleString('zh-CN') + '</b> 条</span>' +
      '<select id="permPageSize"><option value="20"' + (size === 20 ? ' selected' : '') + '>20 条/页</option><option value="50"' + (size === 50 ? ' selected' : '') + '>50 条/页</option><option value="100"' + (size === 100 ? ' selected' : '') + '>100 条/页</option></select>' +
      '<span class="pager-spacer"></span>' +
      '<button type="button" class="page-btn" data-page="' + Math.max(1, state.page - 1) + '"' + (state.page <= 1 ? ' disabled' : '') + '>‹</button>' +
      pageButtons +
      '<button type="button" class="page-btn" data-page="' + Math.min(totalPages, state.page + 1) + '"' + (state.page >= totalPages ? ' disabled' : '') + '>›</button>';
    host.querySelectorAll('.page-btn[data-page]:not(:disabled)').forEach(function (btn) {
      btn.onclick = function () {
        const next = Number(btn.getAttribute('data-page'));
        if (!next || next === state.page) return;
        state.page = next;
        clearSelection();
        loadMembers();
      };
    });
    const pageSize = document.getElementById('permPageSize');
    if (pageSize) pageSize.onchange = function () {
      state.pageSize = Number(pageSize.value || 20);
      state.page = 1;
      clearSelection();
      loadMembers();
    };
  }

  function renderAudit() {
    const host = document.getElementById('auditLogsBody');
    if (!host) return;
    host.innerHTML = '<div class="loading">加载中…</div>';
    fetch('/admin/permissions/audit-logs?limit=100', { headers: apiHeaders() })
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (!json.success) throw new Error(json.message || '加载失败');
        const rows = json.data || [];
        if (!rows.length) { host.innerHTML = '<div class="info">暂无操作记录</div>'; return; }
        host.innerHTML = '<table><thead><tr><th>时间</th><th>操作人</th><th>角色</th><th>操作</th><th>目标</th><th>结果</th></tr></thead><tbody>' + rows.map(function (r) {
          return '<tr><td>' + window.escapeHtml(r.createdAt || '') + '</td><td>' + window.escapeHtml(r.operatorAccount || '') + '</td><td>' + window.escapeHtml(ROLE_LABELS[r.roleCode] || r.roleCode || '') + '</td><td>' + window.escapeHtml(r.actionType || '') + '</td><td>' + window.escapeHtml(r.targetName || r.targetType || '') + '</td><td>' + window.escapeHtml(r.remark || '成功') + '</td></tr>';
        }).join('') + '</tbody></table>';
      })
      .catch(function (err) { host.innerHTML = '<div class="info">' + window.escapeHtml(err.message) + '</div>'; });
  }

  function normalizeDirectory(items) {
    return (Array.isArray(items) ? items : []).map(function (item) {
      const value = item.value != null ? item.value : (item.id != null ? item.id : (item.deptId != null ? item.deptId : (item.deptID != null ? item.deptID : (item.teamId != null ? item.teamId : item.teamID))));
      const label = item.label || item.name || item.deptName || item.teamName || String(value == null ? '' : value);
      const opt = { id: String(value == null ? '' : value), value: String(value == null ? '' : value), label: label };
      if (Array.isArray(item.children) && item.children.length) opt.children = normalizeDirectory(item.children);
      return opt;
    }).filter(function (x) { return x.value; });
  }

  let addPicker = null;
  let scopePicker = null;
  let scopeKind = 'dept';

  function mountScopePicker(kind) {
    const host = document.getElementById('permScopePickerHost');
    if (!host) return Promise.resolve();
    scopeKind = kind || 'dept';
    scopePicker = null;
    host.innerHTML = '<div class="loading" style="padding:10px">加载管理范围…</div>';
    if (!window.WbDirectory || !window.WbPicker) {
      host.innerHTML = '<div class="info">组织范围选择器未加载，请刷新后重试。</div>';
      return Promise.resolve();
    }
    const loader = scopeKind === 'team' ? window.WbDirectory.agileTeams : window.WbDirectory.depts;
    return loader().then(function (items) {
      host.innerHTML = '';
      scopePicker = window.WbPicker.mount({
        el: host,
        mode: 'single',
        layout: 'tree',
        options: normalizeDirectory(items),
        value: '',
        placeholder: scopeKind === 'team' ? '请选择敏捷小组' : '请选择部门',
        emptyLabel: '请选择管理范围'
      });
    }).catch(function (err) {
      host.innerHTML = '<div class="info">范围加载失败：' + window.escapeHtml(err.message || '加载失败') + '</div>';
    });
  }

  function openAddDrawer() {
    if (!state.data || !state.data.isManaged) {
      window.showToast('当前角色由用户个人资料维护，暂不支持管理员添加', 'info');
      return;
    }
    const drawer = document.getElementById('permAddDrawer');
    const body = document.getElementById('permAddBody');
    const title = document.getElementById('permAddTitle');
    if (!drawer || !body) return;
    if (title) title.textContent = '添加 ' + roleLabel() + ' 成员';
    let scopeHtml = '';
    if (state.roleCode === 'lead') {
      scopeHtml = '<div class="scope-card">' +
        '<div class="scope-title">团队管理范围</div>' +
        '<div class="scope-type"><label><input type="radio" name="permScopeType" value="dept" checked> 部门</label><label><input type="radio" name="permScopeType" value="team"> 敏捷小组</label></div>' +
        '<div id="permScopePickerHost" class="scope-picker-host"></div>' +
        '<label class="scope-children" id="permScopeChildrenRow"><input type="checkbox" id="permAddIncludeChildren"> 包含下级部门</label>' +
      '</div>';
    } else {
      scopeHtml = '<div class="info" style="margin-top:14px">PMO 固定为全局权限，无需设置管理范围。</div>';
    }
    body.innerHTML = '<div class="info" style="margin-bottom:12px">选择 1 人就是单人添加；可一次选择多人批量添加。已属于当前角色的账号不可重复添加。</div><div id="permAddPickerHost"></div>' + scopeHtml;
    drawer.classList.add('show');
    addPicker = null;
    scopePicker = null;
    if (window.WbPersonPicker) {
      addPicker = window.WbPersonPicker.mount({
        el: body.querySelector('#permAddPickerHost'),
        mode: 'multi',
        placeholder: '搜索姓名 / 账号 / 部门',
        emptyLabel: '请选择人员',
        disabledAccounts: (state.data.members || []).map(function (m) { return m.account; })
      });
    } else {
      body.querySelector('#permAddPickerHost').innerHTML = '<div class="info">人员选择器未加载，请刷新后重试。</div>';
    }
    if (state.roleCode === 'lead') {
      mountScopePicker('dept');
      body.querySelectorAll('input[name="permScopeType"]').forEach(function (radio) {
        radio.onchange = function () {
          if (!radio.checked) return;
          const row = document.getElementById('permScopeChildrenRow');
          if (row) row.style.display = radio.value === 'dept' ? 'inline-flex' : 'none';
          mountScopePicker(radio.value);
        };
      });
    }
  }

  function permissionCloseAdd() {
    const drawer = document.getElementById('permAddDrawer');
    if (drawer) drawer.classList.remove('show');
    addPicker = null;
    scopePicker = null;
  }

  function confirmAdd() {
    if (!addPicker) return;
    const values = addPicker.getValue();
    const accounts = Array.isArray(values) ? values : (values ? [values] : []);
    if (!accounts.length) { window.showToast('请选择至少 1 名人员', 'error'); return; }
    const body = { accounts: accounts };
    if (state.roleCode === 'lead') {
      if (!scopePicker) { window.showToast('请选择团队管理范围', 'error'); return; }
      const v = scopePicker.getValue();
      const id = Number(Array.isArray(v) ? v[0] : v);
      if (!id) { window.showToast('请选择团队管理范围', 'error'); return; }
      if (scopeKind === 'team') body.scopeTeamId = id;
      else body.scopeDeptId = id;
      body.scopeIncludeChildren = scopeKind === 'dept' && !!(document.getElementById('permAddIncludeChildren') && document.getElementById('permAddIncludeChildren').checked);
    }
    fetch('/admin/permissions/roles/' + encodeURIComponent(state.roleCode) + '/members', {
      method: 'POST', headers: Object.assign({ 'Content-Type': 'application/json' }, apiHeaders()), body: JSON.stringify(body)
    }).then(function (r) { return r.json(); }).then(function (json) {
      if (!json.success) throw new Error(json.message || '添加失败');
      window.showToast('已添加 ' + ((json.data && json.data.affected) || accounts.length) + ' 人', 'success');
      permissionCloseAdd();
      state.page = 1;
      state.selected = {};
      return loadRoles().then(loadMembers);
    }).catch(function (err) { window.showToast(err.message || '添加失败', 'error'); });
  }

  function removeMembers(accounts) {
    if (!accounts || !accounts.length) return;
    if (!state.data || !state.data.isManaged) { window.showToast('当前角色由用户个人资料维护，无法移除', 'error'); return; }
    if (!window.confirm('确认将 ' + accounts.length + ' 人移出「' + roleLabel() + '」？\n这里只移除当前角色，不会删除用户账号。')) return;
    fetch('/admin/permissions/roles/' + encodeURIComponent(state.roleCode) + '/members/remove', {
      method: 'POST', headers: Object.assign({ 'Content-Type': 'application/json' }, apiHeaders()), body: JSON.stringify({ accounts: accounts })
    }).then(function (r) { return r.json(); }).then(function (json) {
      if (!json.success) throw new Error(json.message || '移除失败');
      const affected = (json.data && json.data.affected) || 0;
      const skipped = (json.data && json.data.skippedAccounts) || [];
      window.showToast('已移除 ' + affected + ' 人' + (skipped.length ? '，跳过 ' + skipped.length + ' 人' : ''), skipped.length ? 'info' : 'success');
      const totalAfter = Math.max(0, Number(state.data.total || 0) - affected);
      const newPages = Math.max(1, Math.ceil(totalAfter / state.pageSize));
      if (state.page > newPages) state.page = newPages;
      state.selected = {};
      return loadRoles().then(loadMembers);
    }).catch(function (err) { window.showToast(err.message || '移除失败', 'error'); });
  }

  function openUserPagesDrawer(account, displayName) {
    const drawer = document.getElementById('permissionDrawer');
    const title = document.getElementById('drawerTitle');
    const acc = document.getElementById('drawerAccount');
    const body = document.getElementById('drawerBody');
    if (!drawer || !body) return;
    if (title) title.textContent = (displayName || account) + ' · 页面权限';
    if (acc) acc.textContent = account + ' · 当前角色：' + roleLabel();
    body.innerHTML = '<div class="loading">加载页面权限…</div>';
    drawer.classList.add('show');
    fetch('/admin/permissions/roles/' + encodeURIComponent(state.roleCode) + '/users/' + encodeURIComponent(account) + '/pages', { headers: apiHeaders() })
      .then(function (r) { return r.json(); })
      .then(function (json) {
        if (!json.success) throw new Error(json.message || '加载失败');
        renderUserPages(json.data || {}, account);
      })
      .catch(function (err) { body.innerHTML = '<div class="info">' + window.escapeHtml(err.message || '加载失败') + '</div>'; });
  }

  function renderUserPages(data, account) {
    const body = document.getElementById('drawerBody');
    if (!body) return;
    const pages = data.pages || [];
    if (!pages.length) { body.innerHTML = '<div class="info">当前角色暂无可配置页面。</div>'; return; }
    body.innerHTML = '<div class="page-access-note">勾选 = 该用户在「' + window.escapeHtml(data.roleLabel || roleLabel()) + '」下可以看到该页面；取消勾选 = 隐藏入口。写操作仍由系统角色权限与业务规则校验。</div>' +
      '<div class="page-access-list">' + pages.map(function (p) {
        const locked = !!p.locked;
        const checked = !!p.enabled || locked;
        const source = p.source === 'override' ? '个人设置' : '角色默认';
        return '<label class="page-access-row' + (locked ? ' is-locked' : '') + '">' +
          '<span class="page-access-copy"><span class="page-access-name">' + window.escapeHtml(p.pageName || p.pageCode) + '</span><span class="page-access-desc">' + window.escapeHtml(source) + '</span></span>' +
          (locked ? '<span class="page-access-meta">系统保底</span>' : '') +
          '<input type="checkbox" class="perm-page-cb" data-page="' + window.escapeHtml(p.pageCode) + '"' + (checked ? ' checked' : '') + (locked ? ' disabled title="系统保底页面不可关闭"' : '') + '>' +
        '</label>';
      }).join('') + '</div>';
    const btn = document.getElementById('permUserPagesSave');
    if (btn) {
      btn.style.display = 'inline-block';
      btn.onclick = function () { saveUserPages(account); };
    }
  }

  function saveUserPages(account) {
    const pages = Array.from(document.querySelectorAll('#drawerBody .perm-page-cb')).map(function (cb) {
      return { pageCode: cb.getAttribute('data-page'), enabled: cb.disabled ? true : cb.checked };
    });
    fetch('/admin/permissions/roles/' + encodeURIComponent(state.roleCode) + '/users/' + encodeURIComponent(account) + '/pages', {
      method: 'PUT', headers: Object.assign({ 'Content-Type': 'application/json' }, apiHeaders()), body: JSON.stringify({ pages: pages })
    }).then(function (r) { return r.json(); }).then(function (json) {
      if (!json.success) throw new Error(json.message || '保存失败');
      window.showToast('页面权限已保存', 'success');
      permissionCloseDetail();
    }).catch(function (err) { window.showToast(err.message || '保存失败', 'error'); });
  }

  function permissionCloseDetail() {
    const drawer = document.getElementById('permissionDrawer');
    if (drawer) drawer.classList.remove('show');
  }

  window.permissionSwitchTab = function (name) {
    document.querySelectorAll('#page-permission .permission-tabs button').forEach(function (b) {
      b.classList.toggle('active', b.getAttribute('data-tab') === name);
    });
    document.querySelectorAll('#page-permission .permission-tab-panel').forEach(function (p) { p.classList.remove('active'); });
    const panel = document.getElementById('tab-' + name);
    if (panel) panel.classList.add('active');
    if (name === 'audit') renderAudit();
  };
  window.permissionCloseAdd = permissionCloseAdd;
  window.permissionCloseDetail = permissionCloseDetail;
  window.permissionOpenAdd = openAddDrawer;
  window.permissionConfirmAdd = confirmAdd;
  window.permissionBatchRemove = function () { removeMembers(Object.keys(state.selected)); };
  window.permissionClearSelection = clearSelection;
  window.permissionSearch = function () {
    const kw = document.getElementById('permKeyword');
    state.keyword = kw ? kw.value.trim() : '';
    state.page = 1;
    clearSelection();
    loadMembers();
  };
  window.permissionReset = function () {
    state.keyword = '';
    const kw = document.getElementById('permKeyword');
    if (kw) kw.value = '';
    state.page = 1;
    clearSelection();
    loadMembers();
  };

  function bind() {
    ensureStyles();
    const searchBtn = document.getElementById('permSearchBtn');
    const resetBtn = document.getElementById('permResetBtn');
    const addBtn = document.getElementById('permAddBtn');
    const kw = document.getElementById('permKeyword');
    const allCb = document.getElementById('permMemberAll');
    const batchRemove = document.getElementById('memberBatchRemove');
    const batchClear = document.getElementById('memberBatchClear');
    const addConfirm = document.getElementById('permAddConfirm');
    if (searchBtn) searchBtn.onclick = window.permissionSearch;
    if (resetBtn) resetBtn.onclick = window.permissionReset;
    if (addBtn) addBtn.onclick = openAddDrawer;
    if (kw) kw.onkeydown = function (e) { if (e.key === 'Enter') window.permissionSearch(); };
    if (allCb) allCb.onchange = function () {
      document.querySelectorAll('#userRows .perm-member-cb:not(:disabled)').forEach(function (cb) {
        cb.checked = allCb.checked;
        if (allCb.checked) state.selected[cb.getAttribute('data-account')] = true;
        else delete state.selected[cb.getAttribute('data-account')];
      });
      syncHeaderCheckbox();
      renderToolbar();
    };
    if (batchRemove) batchRemove.onclick = window.permissionBatchRemove;
    if (batchClear) batchClear.onclick = clearSelection;
    if (addConfirm) addConfirm.onclick = confirmAdd;
  }

  let mounted = false;
  window.__permissionMembersMount = function () {
    if (!mounted) { bind(); mounted = true; }
    return loadRoles().then(loadMembers);
  };

  document.addEventListener('DOMContentLoaded', function () {
    if (!document.getElementById('permissionEmbedRoot')) return;
    const page = document.getElementById('permissionEmbedRoot').closest('.page');
    if (page && !page.classList.contains('active')) return;
    bind();
    mounted = true;
    loadRoles().then(loadMembers);
  });
})();
