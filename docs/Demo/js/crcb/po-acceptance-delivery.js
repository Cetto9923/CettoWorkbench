(function () {
  'use strict';

  const STAGES = [
    { key: 'submit', tone: 'blue', label: '待提交验收', description: '测试完成后，等待提交业务验收' },
    { key: 'accept', tone: 'orange', label: '待验收', description: '已提交验收，等待验收人处理' },
    { key: 'deliver', tone: 'purple', label: '待发起交付', description: '验收已通过，等待 PO 发起交付' },
    { key: 'release', tone: 'green', label: '待发布', description: '已发起交付，等待发布负责人上线' },
    { key: 'verify', tone: 'red', label: '待验证', description: '已发布，等待生产验证确认' },
    { key: 'done', tone: 'gray', label: '已完成', description: '已发布且有真实人工评价，可退出本工作池' }
  ];
  const state = { stage: 'submit', relation: 'all', version: '', system: '', owner: '', keyword: '', page: 1, pageSize: 15, loading: false, data: null };
  const adItemCache = {};

  function h(v) {
    return String(v == null ? '' : v).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
  }
  function meaningful(v) {
    const s = String(v == null ? '' : v).trim();
    if (!s || s === '-' || s === '—' || /^undefined$/i.test(s) || /^null$/i.test(s)) return '';
    return s;
  }
  function js(v) {
    const raw = String(v == null ? '' : v)
      .replace(/\\/g, '\\\\')
      .replace(/\r/g, '\\r')
      .replace(/\n/g, '\\n')
      .replace(/'/g, "\\'");
    return h(raw);
  }
  function byId(id) { return document.getElementById(id); }
  function demandId(item) { return Number(item && item.demandId || (typeof extractDemandId === 'function' ? extractDemandId(item && item.id) : 0)) || 0; }
  function show(msg) { if (typeof showToast === 'function') showToast(msg); }

  function ensureShell() {
    const host = byId('acceptanceDeliveryBody');
    if (!host) return null;
    if (host.dataset.ready === '1') return host;
    host.dataset.ready = '1';
    host.innerHTML =
      '<div class="ad-notice"><i class="fas fa-circle-info"></i><span>按“提交验收 → 验收 → 发起交付 → 发布 → 验证”闭环推进；前置条件不足时展示卡点并限制动作。</span></div>' +
      '<div class="ad-summary" id="adSummary"></div>' +
      WBFilterShell.primary('<div class="ad-filterbar filter-bar filter-bar--common">' +
      '<div class="ad-relation" role="group" aria-label="责任视角">' +
      '<button type="button" class="ad-chip active" data-relation="all">全部</button>' +
      '<button type="button" class="ad-chip" data-relation="mine">待我处理</button>' +
      '<button type="button" class="ad-chip" data-relation="follow">待我跟进</button></div>' +
      '<select class="form-select" id="adVersionFilter" aria-label="版本窗口"><option value="">全部版本窗口</option></select>' +
      '<span class="ad-scope-hint" id="adScopeHint"></span>' +
      '<select class="form-select" id="adSystemFilter" aria-label="产品或系统"><option value="">系统：全部</option></select>' +
      '<select class="form-select" id="adOwnerFilter" aria-label="当前责任人"><option value="">责任人：全部</option></select>' +
      '<input class="form-input" id="adKeywordFilter" placeholder="搜索 ID / 标题" aria-label="搜索 ID 或标题">' +
      '<button type="button" class="action-btn" id="adClearBtn"><i class="fas fa-rotate-left"></i>重置</button>' +
      '<span class="ad-status" id="adStatus"></span></div>') +
      WBListShell.render({
        surfaceClass: 'ad-surface',
        scrollClass: 'ad-scroll',
        content: '<table class="tbl ad-table"><thead><tr>' +
      '<th class="c-id">ID</th><th class="ad-c-title">需求标题</th><th>当前阶段</th><th>版本窗口</th><th class="ad-c-state">当前状态 / 前置条件</th><th>当前责任人</th><th>计划时间</th><th>卡点 / 风险</th><th class="ad-c-action">操作</th>' +
      '</tr></thead><tbody id="adTableBody"></tbody></table>',
        empty: '<div class="po-list-empty" id="adEmpty" hidden><i class="fas fa-inbox"></i><div class="t">当前条件下暂无待处理事项</div><div class="h">可清空筛选条件，或切换其他处理阶段。</div></div>',
        pagerId: 'adPagination'
      });
    host.querySelectorAll('[data-relation]').forEach((btn) => btn.addEventListener('click', () => {
      state.relation = btn.dataset.relation || 'all'; state.page = 1; loadAcceptanceDelivery();
    }));
    byId('adVersionFilter').addEventListener('change', (e) => { state.version = e.target.value; state.page = 1; loadAcceptanceDelivery(); });
    byId('adSystemFilter').addEventListener('change', (e) => { state.system = e.target.value; state.page = 1; loadAcceptanceDelivery(); });
    byId('adOwnerFilter').addEventListener('change', (e) => { state.owner = e.target.value; state.page = 1; loadAcceptanceDelivery(); });
    byId('adKeywordFilter').addEventListener('input', debounce((e) => { state.keyword = e.target.value; state.page = 1; loadAcceptanceDelivery(); }, 250));
    byId('adClearBtn').addEventListener('click', () => {
      Object.assign(state, { relation: 'all', version: '', system: '', owner: '', keyword: '', page: 1 });
      loadAcceptanceDelivery();
    });
    return host;
  }

  function debounce(fn, wait) {
    let t = null;
    return function () { clearTimeout(t); const args = arguments; t = setTimeout(() => fn.apply(null, args), wait); };
  }

  async function loadAcceptanceDelivery() {
    if (!ensureShell()) return;
    // V1.2.1：避免重复并发 load 时状态错位；同一实例不会因快速切换/重复进入而保留"加载中…"残留。
    if (state.loading) return;
    state.loading = true;
    renderAcceptanceDelivery();
    const qs = new URLSearchParams();
    ['stage', 'relation', 'version', 'system', 'owner', 'keyword', 'page', 'pageSize'].forEach((k) => {
      if (state[k] !== '' && state[k] != null) qs.set(k, state[k]);
    });
    // V1.2.1：版本跟进 → 验收交付深链。
    if (state.__deepLink && state.__deepLink.demandId) qs.set('demandId', state.__deepLink.demandId);
    if (state.__deepLink && state.__deepLink.windowId) qs.set('windowId', state.__deepLink.windowId);
    try {
      const json = await apiFetch('/workbench/api/po/acceptance-delivery?' + qs.toString());
      if (!json || json.success !== true) throw new Error((json && json.message) || '接口失败');
      state.data = json;
      // 单次深链消费成功 → 清空，避免影响下次手动加载
      if (state.__deepLink && state.__deepLink.demandId && json.highlightDemandId === state.__deepLink.demandId) {
        state.__deepLink = null;
      }
    } catch (e) {
      state.data = { success: false, summary: [], items: [], pagination: { page: state.page, pageSize: state.pageSize, total: 0 }, filters: {}, message: e.message || '获取验收交付数据失败' };
    } finally {
      state.loading = false;
      renderAcceptanceDelivery();
    }
  }

  // V1.2.1：版本跟进深链 — 设置 state.stage 与 AD state.__deepLink 后重新加载。
  function applyAcceptanceDeliveryDeepLink(ctx) {
    if (!ctx || !ctx.demandId) return;
    state.__deepLink = { demandId: Number(ctx.demandId) || 0, windowId: Number(ctx.windowId) || 0, intent: String(ctx.intent || '') };
    if (ctx.intent && /^(submit|accept|deliver|release|verify)$/.test(ctx.intent)) {
      state.stage = ctx.intent;
    }
    state.page = 1;
  }
  window.applyAcceptanceDeliveryDeepLink = applyAcceptanceDeliveryDeepLink;

  function renderAcceptanceDelivery() {
    ensureShell();
    const data = state.data || {};
    renderSummary(data.summary || []);
    renderFilters(data.filters || {});
    renderRows(data.items || []);
    renderPager(data.pagination || { page: state.page, pageSize: state.pageSize, total: 0 });
    const status = byId('adStatus');
    if (status) {
      status.textContent = state.loading ? '加载中…' : (data.success === false ? (data.message || '加载失败') : '');
      status.hidden = !status.textContent;
    }
    document.querySelectorAll('#acceptanceDeliveryBody [data-relation]').forEach((btn) => btn.classList.toggle('active', btn.dataset.relation === state.relation));
  }

  function renderSummary(summary) {
    const host = byId('adSummary');
    if (!host) return;
    const byStage = {};
    summary.forEach((s) => { byStage[s.stage] = s; });
    host.innerHTML = STAGES.map((def) => {
      const s = byStage[def.key];
      const label = (s && s.label) ? s.label : def.label;
      const description = (s && s.description) ? s.description : def.description;
      const count = (s && typeof s.count === 'number') ? s.count : 0;
      return '<button type="button" class="ad-card ' + (state.stage === def.key ? 'active ' : '') + 'tone-' + def.tone + '" data-stage="' + h(def.key) + '">' +
        '<span class="ad-card-label"><i></i>' + h(label) + '</span><strong>' + Number(count) + '</strong><span>' + h(description) + '</span></button>';
    }).join('');
    host.querySelectorAll('[data-stage]').forEach((btn) => btn.addEventListener('click', () => { state.stage = btn.dataset.stage; state.page = 1; loadAcceptanceDelivery(); }));
  }

  function renderFilters(filters) {
    renderVersionOptions(filters.versions || [], state.version);
    setOptions('adSystemFilter', '系统：全部', filters.systems || [], state.system);
    setOptions('adOwnerFilter', '责任人：全部', filters.owners || [], state.owner);
    renderScopeHint(filters.versions || []);
    const kw = byId('adKeywordFilter');
    if (kw && kw.value !== state.keyword) kw.value = state.keyword;
  }
  // 版本窗口下拉分组渲染（Goal §六）：进行中 / 即将上线 / 历史窗口 / 未关联版本窗口。
  function renderVersionOptions(list, value) {
    const el = byId('adVersionFilter');
    if (!el) return;
    const current = value || '';
    const isPlanning = (w) => w.status === 'planning' || w.status === 'current' || w.status === 'next';
    const groups = [
      { label: '进行中', cond: (w) => !w.isHistory && isPlanning(w) },
      { label: '即将上线', cond: (w) => !w.isHistory && !isPlanning(w) },
      { label: '历史窗口', cond: (w) => w.isHistory }
    ];
    let html = '<option value="">全部版本窗口</option>';
    groups.forEach((g) => {
      const opts = (list || []).filter((o) => Number(o.id) !== 0 && g.cond(o));
      if (!opts.length) return;
      html += '<optgroup label="' + h(g.label) + '">' + opts.map((o) => {
        const release = o.releaseDate ? ' · 上线 ' + String(o.releaseDate).slice(5) : '';
        return '<option value="' + h(o.name) + '"' + (String(o.name) === String(current) ? ' selected' : '') + '>' + h(o.name) + ' · ' + Number(o.count || 0) + release + '</option>';
      }).join('') + '</optgroup>';
    });
    const none = (list || []).find((o) => Number(o.id) === 0);
    if (none) {
      html += '<optgroup label="其他"><option value="未关联版本窗口"' + ('未关联版本窗口' === String(current) ? ' selected' : '') + '>未关联版本窗口 · ' + Number(none.count || 0) + '</option></optgroup>';
    }
    el.innerHTML = html;
    el.value = current;
  }
  // 统计范围弱提示（Goal §七）：不额外占行，弱化文字。
  function renderScopeHint(list) {
    const host = byId('adScopeHint');
    if (!host) return;
    if (!state.version) {
      host.textContent = '统计范围：全部版本窗口';
      return;
    }
    const w = (list || []).find((o) => String(o.name) === String(state.version));
    if (w && Number(w.id) === 0) {
      host.textContent = '统计范围：未关联版本窗口';
    } else if (w && w.releaseDate) {
      host.textContent = '统计范围：' + w.name + ' · 上线 ' + String(w.releaseDate).slice(5);
    } else {
      host.textContent = '统计范围：' + (state.version || '');
    }
  }
  function setOptions(id, first, list, value) {
    const el = byId(id);
    if (!el) return;
    const current = value || '';
    el.innerHTML = '<option value="">' + h(first) + '</option>' + list.map((o) => '<option value="' + h(o.value) + '">' + h(o.label) + '</option>').join('');
    el.value = current;
  }

  function renderRows(items) {
    const body = byId('adTableBody');
    const empty = byId('adEmpty');
    if (!body || !empty) return;
    for (const k in adItemCache) delete adItemCache[k];
    items.forEach(function (it) { adItemCache[it.id] = it; });
    const highlightId = (state.data && state.data.highlightDemandId) ? Number(state.data.highlightDemandId) : 0;
    body.innerHTML = items.map(function (it) { return rowHtml(it, highlightId); }).join('');
    const isError = state.data && state.data.success === false;
    empty.hidden = items.length > 0 || state.loading;
    const emptyEl = empty;
    if (isError) {
      const title = emptyEl.querySelector('.t');
      const hint = emptyEl.querySelector('.h');
      if (title) title.textContent = '加载失败';
      if (hint) hint.textContent = state.data.message || '请稍后重试';
    } else if (!state.loading && items.length === 0) {
      const title = emptyEl.querySelector('.t');
      const hint = emptyEl.querySelector('.h');
      if (title) title.textContent = '当前条件下暂无待处理事项';
      if (hint) hint.textContent = '可清空筛选条件，或切换其他处理阶段。';
    }
    // 深链高亮行 scrollIntoView
    if (highlightId) {
      const tr = body.querySelector('tr.ad-row-highlight');
      if (tr && typeof tr.scrollIntoView === 'function') {
        try { tr.scrollIntoView({ block: 'center', behavior: 'smooth' }); } catch (e) {}
      }
    }
  }
  function rowHtml(item, highlightId) {
    const actions = (item.availableActions || []).map((a) => actionHtml(item, a)).join('');
    const riskClass = item.riskTone === 'danger' ? 'st-danger' : (item.riskTone === 'warning' ? 'st-warning' : 'st-ok');
    const isHi = highlightId && Number(item.demandId) === Number(highlightId);
    const cls = isHi ? 'ad-row-highlight' : '';
    const meta = [item.system, item.priority].map(meaningful).filter(Boolean).join(' · ');
    return '<tr class="' + cls + '" data-demand-id="' + h(item.demandId) + '">' +
      '<td class="c-id"><button type="button" class="query-id-link" onclick="event.stopPropagation();viewInZentao(\'' + js(item.id) + '\',\'demand\')">' + h(item.id) + '</button></td>' +
      '<td><button type="button" class="query-title-link ad-title" onclick="event.stopPropagation();openBusinessDemandDetail(\'' + js(item.id) + '\')">' + h(item.title || '-') + '</button>' + (meta ? '<div class="ad-sub">' + h(meta) + '</div>' : '') + '</td>' +
      '<td><span class="status-tag">' + h(item.stageLabel) + '</span></td>' +
      '<td>' + h(item.versionName || '-') + '</td>' +
      '<td><div class="ad-state">' + h(item.state || '-') + '</div><div class="ad-sub">' + h(item.stateDetail || '') + '</div></td>' +
      '<td>' + h(item.owner || '-') + '</td>' +
      '<td>' + h(shortDate(item.planDate) || '-') + '</td>' +
      '<td><span class="status-tag ' + riskClass + '">' + h(item.risk || '无阻塞') + '</span></td>' +
      '<td><div class="ad-actions">' + actions + '</div></td></tr>';
  }
  function actionHtml(item, action) {
    const cls = 'action-btn' + (action.primary ? ' primary' : '');
    const disabled = action.enabled === false ? ' disabled title="' + h(action.reason || '当前不可操作') + '"' : '';
    return '<button type="button" class="' + cls + '"' + disabled + ' onclick="event.stopPropagation();handleAcceptanceDeliveryAction(\'' + js(action.key) + '\',' + demandId(item) + ',\'' + js(item.id) + '\',' + Number(item.versionId || 0) + ',event)">' + h(action.label) + '</button>';
  }
  function shortDate(v) { const m = String(v || '').match(/\d{4}-(\d{2})-(\d{2})/); return m ? m[1] + '-' + m[2] : ''; }

  function renderPager(p) {
    const host = byId('adPagination');
    if (!host) return;
    const total = Number(p.total || 0);
    const pages = Math.max(1, Math.ceil(total / state.pageSize));
    host.innerHTML = '<div class="pagination"><span>共 ' + total + ' 条</span><div class="pagination-controls">' +
      '<button type="button" class="action-btn" ' + (state.page <= 1 ? 'disabled' : '') + ' onclick="setAcceptanceDeliveryPage(' + (state.page - 1) + ')">上一页</button>' +
      '<span>' + state.page + ' / ' + pages + '</span>' +
      '<button type="button" class="action-btn" ' + (state.page >= pages ? 'disabled' : '') + ' onclick="setAcceptanceDeliveryPage(' + (state.page + 1) + ')">下一页</button>' +
      '</div></div>';
  }

  window.setAcceptanceDeliveryPage = function (page) { state.page = Math.max(1, Number(page || 1)); loadAcceptanceDelivery(); };
  window.renderAcceptanceDelivery = renderAcceptanceDelivery;
  window.loadAcceptanceDelivery = loadAcceptanceDelivery;
  window.handleAcceptanceDeliveryAction = async function (key, id, displayId, windowId, evt) {
    // 重复点击防护：找到触发按钮并禁用直到请求结束
    const btn = (evt && evt.currentTarget) || (typeof event !== 'undefined' ? event && event.target : null);
    if (btn && btn.tagName === 'BUTTON') btn.disabled = true;
    try {
      // V1.2.3 fix-accept: 「去验收」直弹需求详情，让用户进详情走真实验收流程，
      // 不再一键 POST /acceptance 把需求标 accepted（之前实现是 bug）。
      if (key === 'detail' || key === 'followTest' || key === 'viewRelease' || key === 'accept') return openBusinessDemandDetail(displayId);
      if (key === 'urgeAcceptance') {
        // 催验收：不把 CurrentHandler(验收阶段常为 RD)当推荐对象；
        // 系统推荐由服务端 accepter→openedBy；前端可搜索修改。
        return openRemind('acceptance', displayId, {
          windowId: windowId,
          reason: '请尽快完成业务验收'
        });
      }
      if (key === 'urgeApproval') return openRemind('manager-review', displayId, { windowId: windowId, reason: '主管部门审批未完成，阻塞交付' });
      if (key === 'urgeRelease') return openRemind('release', displayId, { windowId: windowId, reason: '已发起交付，等待发布上线', requireRecipient: true });
      if (key === 'submitAcceptance') return await postDemandAction(id, '/submit-acceptance', '已提交验收');
      if (key === 'confirmVerify') return await postDemandAction(id, '/appraise', '生产验证已确认');
      if (key === 'deliver') return openAcceptanceDeliveryDrawer(displayId);
      show('当前动作不可用');
    } finally {
      if (btn && btn.tagName === 'BUTTON') btn.disabled = false;
    }
  };
  async function postDemandAction(id, suffix, okMsg) {
    if (!id) return show('需求 ID 无效');
    try {
      const res = await apiFetch('/workbench/api/demands/' + id + suffix, { method: 'POST', body: { comment: okMsg } });
      if (!res || res.success === false) return show((res && res.message) || '操作失败');
      show(okMsg);
      loadAcceptanceDelivery();
    } catch (e) { show('操作失败，请稍后重试'); }
  }
  window.openAcceptanceDeliveryDrawer = function (displayId) {
    if (typeof window.openPoDeliverModal === 'function') {
      return window.openPoDeliverModal(displayId);
    }
    show('发起交付表单未加载，请刷新页面后重试');
  };
  window.getAcceptanceDeliveryItem = function (displayId) {
    return adItemCache[String(displayId || '')] || null;
  };
})();
