// ═══ QUERY PAGE (需求综合查询 R2A) ═══
// R5-F MOVE FIRST 拆分: 从 po-query-notifications.js 整体迁出需求综合查询域
// (query state / multi-select / filter / renderQuery / ensureQueryInitialLoad)。
// 数据源: requestDemandSearch() → /workbench/api/demands/search (po-data provider)。
// 页面渲染用 canonical WBRenderPagination 与 viewInZentao /
// openBusinessDemandDetail 导航。与 Notification 域零运行时交叉引用。
// 加载契约: 晚于 po-data.js / po-core.js, 早于 po-workbench.js (navTo('query'))。
// ═══ QUERY PAGE (需求综合查询 R2A) ═══
// 2026-08-16 R2A 重写: 综合查询 = 当前账号在 zt_demandpool ACL 范围内可见的需求全集。
// 数据源走 requestDemandSearch() 单独拉 /workbench/api/demands/search, 不再派生自 work-items。
// queryData 元素契约: {id, title, agile, system, status, statusLabel, pri, priLabel,
//   owner, ownerName, estimateLaunch, hang, poolId, category}

// ════════════════════════════════════════════════════════════════════════════
// TODO [V0.8-INCIDENT] 通知中心未读数 362 异常大 (客户端 worker 标记 2026-08-17)
// ════════════════════════════════════════════════════════════════════════════
// 现象:
//   - 我的待办 chip: 8 / 待我处理 90 (合理)
//   - 通知中心 chip: 362 (失衡 45 倍 — 单页面塞不下这么多条)
//   - syncNoticeBadge() 曾把 unread 误写到待办侧栏徽章, 现已拆开; 通知侧仍有 99+ 上限
//   - 用户视角: 点开通知中心看到几百条历史遗留通知, 全是已读/无关, 没阅读价值
// 可能的根因 (后端 SQL / 聚合层 4 个嫌疑点):
//   1. SQL 没按 role/AC 过滤  →  全公司所有 zt_demand 的历史动作通知全部返给当前账号
//   2. SQL 没按 readStatus='unread' 去重  →  把已读通知也当成未读计数
//   3. SQL 没按 (object_id, action) 去重  →  同一条需求被多角色多次提及, 产生 N 条
//   4. 返回全量 (没 LIMIT)  →  数据量本身合理, 是前端展示/聚合错误
// 排查入口 (后端 owner 调研, 客户端不擅改 Go 代码):
//   - `GET /workbench/api/notifications?account=admin&unreadOnly=true`
//     直接看返多少条 / 返哪些 object_id / 是否带 readStatus
//   - 修复位置 (后端, 不在本次 scope):
//       internal/module/workbench/handler_demands.go  → notifications handler
//       internal/module/workbench/service_datasets.go  → notifications service
//       internal/module/workbench/repo_notifications.go → 实际 SQL
// 客户端容错 (本次仅做展示层, 不改数据):
//   - syncNoticeBadge() 显示上限: unread > 100 时写 "99+"
//   - 仅 UI 收敛, 真实 362 仍然来自后端, 仍需后端 owner 修根因
// 业务/数据真源: V0.8 文档 §通知中心口径
// 工单留痕: 见 artifacts/v0.8-tab-move/RISK.md §362-notification-anomaly
// ════════════════════════════════════════════════════════════════════════════
let queryData = [];
let queryAgileSelected = [];
let querySystemSelected = [];
let queryStatusSelected = [];
let queryPriSelected = [];
let queryMyRelation = 'all';
let querySuspended = false;
let queryKeyword = '';
let queryInitialized = false;
if (!window.queryPaginationState) window.queryPaginationState = { page: 1, pageSize: 20, total: 0 };

/**
 * 通用 multi-select 折叠按钮: 点击切换展开。
 */
function toggleMultiSelect(id) {
  const el = document.getElementById(id);
  if (!el) return;
  document.querySelectorAll('.multi-select').forEach(function (m) { if (m !== el) m.classList.remove('open'); });
  el.classList.toggle('open');
}

/**
 * 通用多选变更 (敏捷小组 / 涉及系统):
 * - 复用 queryAgileSelected / querySystemSelected
 * - 触发 requestDemandSearch 重新拉取
 */
function onQueryGenericMultiChange(hostId) {
  const selected = Array.prototype.slice.call(document.querySelectorAll('#' + hostId + ' input[type="checkbox"]:checked')).map(function (c) { return c.value; });
  if (hostId === 'queryAgileMulti') queryAgileSelected = selected;
  if (hostId === 'querySystemMulti') querySystemSelected = selected;
  const btn = document.querySelector('#' + hostId + ' .multi-select-btn');
  if (btn) {
    if (hostId === 'queryAgileMulti') {
      btn.textContent = queryAgileSelected.length ? '敏捷小组(' + queryAgileSelected.length + ')' : '全部敏捷小组';
    } else if (hostId === 'querySystemMulti') {
      btn.textContent = querySystemSelected.length ? '系统(' + querySystemSelected.length + ')' : '全部涉及系统';
    }
  }
  triggerQueryRefresh();
}

function onQueryStatusMultiChange() {
  queryStatusSelected = Array.prototype.slice.call(document.querySelectorAll('#queryStatusMulti input[type="checkbox"]:checked')).map(function (c) { return c.value; });
  const btn = document.querySelector('#queryStatusMulti .multi-select-btn');
  if (btn) btn.textContent = queryStatusSelected.length ? '状态(' + queryStatusSelected.length + ')' : '全部状态';
  if (window.queryPaginationState) window.queryPaginationState.page = 1;
  triggerQueryRefresh();
}

function onQueryPriMultiChange() {
  // pri 多选: 与 status / agile / system 同维度 OR; 不再强制单选。
  // 收集所有 checked 值, 保留顺序, 全空时还原。
  const checked = Array.prototype.slice.call(document.querySelectorAll('#queryPriMulti input[type="checkbox"]:checked')).map(function (c) { return c.value; });
  queryPriSelected = checked;
  const btn = document.querySelector('#queryPriMulti .multi-select-btn');
  if (btn) btn.textContent = queryPriSelected.length ? '优先级(' + queryPriSelected.length + ')' : '全部优先级';
  if (window.queryPaginationState) window.queryPaginationState.page = 1;
  triggerQueryRefresh();
}

function onQuerySuspendedChange(checked) {
  querySuspended = !!checked;
  if (window.queryPaginationState) window.queryPaginationState.page = 1;
  triggerQueryRefresh();
}

function onQueryKeywordChange(value) {
  queryKeyword = String(value || '').trim();
  if (window.queryPaginationState) window.queryPaginationState.page = 1;
  triggerQueryRefresh();
}

function resetQueryFilters() {
  queryAgileSelected = [];
  querySystemSelected = [];
  queryStatusSelected = [];
  queryPriSelected = [];
  queryMyRelation = 'all';
  querySuspended = false;
  queryKeyword = '';
  if (window.queryPaginationState) {
    window.queryPaginationState.page = 1;
    window.queryPaginationState.total = 0;
  }
  // 清掉所有 checkbox 勾选
  document.querySelectorAll('#queryAgileMulti input[type="checkbox"], #querySystemMulti input[type="checkbox"], #queryStatusMulti input[type="checkbox"], #queryPriMulti input[type="checkbox"]').forEach(function (c) { c.checked = false; });
  const kwInput = document.getElementById('queryKeywordFilter');
  if (kwInput) kwInput.value = '';
  const susChk = document.getElementById('querySuspended');
  if (susChk) susChk.checked = false;
  // reset 快捷视图
  document.querySelectorAll('.sv-pill').forEach(function (p) { p.classList.remove('active'); });
  const allPill = document.querySelector('.sv-pill[data-sv="all"]');
  if (allPill) allPill.classList.add('active');
  triggerQueryRefresh();
}

let queryRefreshTimer = null;
function triggerQueryRefresh() {
  if (queryRefreshTimer) clearTimeout(queryRefreshTimer);
  queryRefreshTimer = setTimeout(function () {
    if (typeof requestDemandSearch === 'function') {
      requestDemandSearch({
        keyword: queryKeyword,
        statuses: queryStatusSelected,
        agiles: queryAgileSelected,
        systems: querySystemSelected,
        pris: queryPriSelected.join(','),
        suspended: querySuspended,
        myRelation: queryMyRelation,
        page: window.queryPaginationState ? window.queryPaginationState.page : 1,
        pageSize: window.queryPaginationState ? window.queryPaginationState.pageSize : 20
      });
    }
  }, 120);
}

/**
 * 5 个快捷视图 chip (R2A 真接通):
 *   - all      = 不带 myRelation, 拉全集
 *   - lead     = BRA = me (我牵头)
 *   - support  = QD/RD = me (我配合)
 *   - follow   = createdBy = me (我关注)
 *   - hang     = 仅看挂起 (suspeded=yes, 不限 myRelation)
 * "保存视图" / "导出" 按钮已移除 (二期提供)。
 */
function applySavedView(el, type) {
  document.querySelectorAll('.sv-pill').forEach(function (p) { p.classList.remove('active'); });
  el.classList.add('active');
  if (window.queryPaginationState) window.queryPaginationState.page = 1;
  if (type === 'hang') {
    querySuspended = true;
    queryMyRelation = 'all';
    const susChk = document.getElementById('querySuspended');
    if (susChk) susChk.checked = true;
  } else {
    querySuspended = false;
    const susChk = document.getElementById('querySuspended');
    if (susChk) susChk.checked = false;
    queryMyRelation = (type === 'lead' || type === 'support' || type === 'follow') ? type : 'all';
  }
  triggerQueryRefresh();
}

/**
 * 渲染 queryData 到 #queryBody 表格。
 * 状态列走 statusLabel (后端已翻译); 优先级列三色 P0(红) / P1(橙) / P2(灰)。
 * "主配" / "部门" / "重要程度" / "需求池" 已删 (backend 无字段)。
 */
function renderQuery() {
  const qs = window.queryPaginationState || (window.queryPaginationState = { page: 1, pageSize: 20, total: 0 });
  const totalPages = Math.max(1, Math.ceil((Number(qs.total) || 0) / Math.max(1, Number(qs.pageSize) || 20)));
  if (qs.page > totalPages) qs.page = totalPages;
  // 后端契约：queryData 已是当前页 rows，禁止再次 slice。
  const pageList = Array.isArray(queryData) ? queryData : [];
  const body = document.getElementById('queryBody');
  function queryDemandRef(id) {
    const raw = String(id || '').trim();
    if (!raw) return '';
    return /^US\d+$/i.test(raw) ? raw.toUpperCase() : (/^\d+$/.test(raw) ? ('US' + raw) : raw);
  }
  if (body) {
    const canUseDom = typeof document.createElement === 'function'
      && typeof document.createDocumentFragment === 'function'
      && typeof body.replaceChildren === 'function';
    if (canUseDom) {
      const fragment = document.createDocumentFragment();
      pageList.forEach(function (q) {
        const priCls = q.priNum === 1 ? 'p1' : (q.priNum === 0 ? 'p0' : (q.priNum >= 3 ? 'p3' : 'p2'));
        const statusCls = q.hang ? 'st-suspended' : 'st-pending';
        const rawId = queryDemandRef(q.id);
        const row = document.createElement('tr');
        row.setAttribute('data-demand-id', rawId);
        function textCell(value, cssText) {
          const td = document.createElement('td');
          td.textContent = String(value == null || value === '' ? '-' : value);
          if (cssText) td.style.cssText = cssText;
          row.appendChild(td);
        }
        function taggedCell(value, className) {
          const td = document.createElement('td');
          const tag = document.createElement('span');
          tag.className = className;
          tag.textContent = String(value == null || value === '' ? '-' : value);
          td.appendChild(tag);
          row.appendChild(td);
        }
        const idCell = document.createElement('td');
        const idBtn = document.createElement('button');
        idBtn.type = 'button';
        idBtn.className = 'query-id-link';
        idBtn.textContent = rawId || '-';
        idBtn.title = '在禅道中查看原始详情';
        idBtn.addEventListener('click', function (event) {
          if (event && event.stopPropagation) event.stopPropagation();
          if (typeof viewInZentao === 'function') viewInZentao(rawId, 'demand');
        });
        idCell.appendChild(idBtn);
        row.appendChild(idCell);
        const titleCell = document.createElement('td');
        const titleBtn = document.createElement('button');
        titleBtn.type = 'button';
        titleBtn.className = 'query-title-link';
        titleBtn.textContent = q.title || '-';
        titleBtn.title = '查看需求详情';
        titleBtn.addEventListener('click', function (event) {
          if (event && event.stopPropagation) event.stopPropagation();
          if (typeof openBusinessDemandDetail === 'function') openBusinessDemandDetail(rawId);
          else if (typeof viewDetail === 'function') viewDetail(rawId);
        });
        titleCell.appendChild(titleBtn);
        row.appendChild(titleCell);
        taggedCell(q.agile || '-', 'sched-tag sub');
        textCell(q.system || '-');
        taggedCell(q.statusLabel || q.status || '-', 'status-tag ' + statusCls);
        taggedCell(q.pri || 'P2', 'pri-tag ' + priCls);
        textCell(q.owner || '-');
        textCell(q.estimateLaunch || '-', 'font-size:12px;color:var(--t2)');
        const hangCell = document.createElement('td');
        if (q.hang) {
          hangCell.textContent = ' 挂起';
        } else {
          hangCell.textContent = '-';
        }
        row.appendChild(hangCell);
        fragment.appendChild(row);
      });
      body.replaceChildren(fragment);
    } else {
      body.innerHTML = pageList.map(function (q) {
        const priCls = q.priNum === 1 ? 'p1' : (q.priNum === 0 ? 'p0' : (q.priNum >= 3 ? 'p3' : 'p2'));
        const statusCls = q.hang ? 'st-suspended' : 'st-pending';
        const rawId = queryDemandRef(q.id);
        const safeId = escapeHtml(rawId);
        const safeJsId = typeof escapeJsString === 'function'
          ? escapeJsString(rawId)
          : String(rawId).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
        return '<tr data-demand-id="' + safeId + '">'
          + '<td><button type="button" class="query-id-link" title="在禅道中查看原始详情" onclick="event.stopPropagation();viewInZentao(\'' + safeJsId + '\',\'demand\')">' + safeId + '</button></td>'
          + '<td><button type="button" class="query-title-link" title="查看需求详情" onclick="event.stopPropagation();openBusinessDemandDetail(\'' + safeJsId + '\')">' + escapeHtml(q.title || '-') + '</button></td>'
          + '<td><span class="sched-tag sub">' + escapeHtml(q.agile || '-') + '</span></td>'
          + '<td>' + escapeHtml(q.system || '-') + '</td>'
          + '<td><span class="status-tag ' + statusCls + '">' + escapeHtml(q.statusLabel || q.status || '-') + '</span></td>'
          + '<td><span class="pri-tag ' + priCls + '">' + escapeHtml(q.pri || 'P2') + '</span></td>'
          + '<td>' + escapeHtml(q.owner || '-') + '</td>'
          + '<td style="font-size:12px;color:var(--t2)">' + escapeHtml(q.estimateLaunch || '-') + '</td>'
          + '<td>' + (q.hang ? '<span style="color:var(--orange);font-size:11px">挂起</span>' : '-') + '</td>'
          + '</tr>';
      }).join('');
      if (!body.__poQueryDelegated && typeof body.addEventListener === 'function') {
        body.__poQueryDelegated = true;
        body.addEventListener('click', function (event) {
          const target = event && event.target && event.target.closest ? event.target.closest('.query-id-link,.query-title-link') : null;
          if (target && event.stopPropagation) event.stopPropagation();
        });
      }
    }
  }
  const pagerHost = document.getElementById('queryPagination');
  if (pagerHost && typeof renderPagination === 'function') {
    pagerHost.innerHTML = renderPagination({
      total: Number(qs.total) || 0,
      page: Number(qs.page) || 1,
      pageSize: Number(qs.pageSize) || 20,
      pageSizeOptions: [10, 20, 50, 100],
      onPageChange: function (newPage) { qs.page = newPage; if (typeof triggerQueryRefresh === 'function') triggerQueryRefresh(); },
      onPageSizeChange: function (newSize) { qs.page = 1; qs.pageSize = Number(newSize) || 20; if (typeof triggerQueryRefresh === 'function') triggerQueryRefresh(); }
    });
  }
}

/**
 * navTo('query') 触发后立即拉取一次 (R2A P1-3 fix): 不再依赖用户改 filter 才出数据。
 */
function ensureQueryInitialLoad() {
  if (queryInitialized) return;
  queryInitialized = true;
  triggerQueryRefresh();
}

// 在 navTo('query') 里已经调用 renderQuery; 让它也确保拉一次数据。
const _origNavToQueryRender = (typeof renderQuery === 'function') ? renderQuery : null;
// 通过 hook: 把 queryInitialized 在 navTo('query') 时 reset, 让用户每次进入页面都重新拉。
// (避免切走/回来后看到旧数据)
