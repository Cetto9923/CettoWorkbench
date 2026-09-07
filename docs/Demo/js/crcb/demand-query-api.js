// =============================================================================
// 文件: web/static/workbench/po/demand-query-api.js
// 模块: PO · 需求查询 V1.3
// 职责: 需求工作区列表 search/count fetch、query 参数构建、API 行映射。
// 依赖: po-core.js (getWorkbenchApiBase)
//       po-data.js (可选 redirectToLogin)
// =============================================================================

/** 禅道 status key → chip 展示文案（与 PRD §3.3 一致；列表 statusLabel 以 API 为准） */
const DEMAND_ZT_STATUS_CHIPS = [
  { key: 'draft', label: '草稿' },
  { key: 'wait', label: '待评审' },
  { key: 'active', label: '已评审' },
  { key: 'clarified', label: '已澄清' },
  { key: 'developing', label: '开发中' },
  { key: 'testing', label: '测试中' },
  { key: 'waitacceptance', label: '待验收' },
  { key: 'acceptanced', label: '已验收' },
  { key: 'waitdeliver', label: '待交付' },
  { key: 'released', label: '已发布' },
  { key: 'closed', label: '已关闭' },
  { key: 'refuse', label: '已驳回' }
];

let demandWorkspaceRows = [];
let demandWorkspaceLoading = false;
let demandWorkspaceInitialized = false;
let demandWorkspaceRefreshTimer = null;

function demandStatusChipLabel(key) {
  const hit = DEMAND_ZT_STATUS_CHIPS.find((it) => it.key === key);
  return hit ? hit.label : key;
}

function demandStatusCssClass(statusKey) {
  const k = String(statusKey || '').toLowerCase().trim();
  if (!k) return 'status';
  if (k === 'reject' || k === 'rejected') return 'status refuse';
  return 'status ' + k;
}

function mapDemandSearchItemToRow(r) {
  const idNum = Number(r && r.id) || 0;
  const rawId = idNum ? ('US' + idNum) : String(r && r.id || '');
  const priNum = Number(r && r.pri);
  const priLabel = (r && r.priLabel) || (Number.isFinite(priNum) ? ('P' + priNum) : 'P2');
  return {
    id: rawId,
    reqCode: rawId,
    title: (r && r.title) || (r && r.name) || '-',
    agile: (r && r.agileGroup) || '-',
    agileId: (r && r.agileId) || 0,
    systems: (r && r.system) ? [r.system] : [],
    mainSystem: (r && r.system) || '-',
    status: (r && r.status) || '',
    statusLabel: (r && r.statusLabel) || (r && r.status) || '-',
    zentaoStatus: (r && r.status) || '',
    zentaoStatusLabel: (r && r.statusLabel) || '',
    valueStreamStage: (r && r.valueStreamStage) || '',
    valueStageLabel: (function () {
      const k = String((r && r.valueStreamStage) || '');
      if (k && typeof VALUE_STREAM_STAGES === 'object' && VALUE_STREAM_STAGES[k]) return VALUE_STREAM_STAGES[k].shortLabel;
      return k;
    })(),
    priority: priLabel,
    priNum: Number.isFinite(priNum) ? priNum : 2,
    owner: (r && r.ownerName) || (r && r.ownerAccount) || '-',
    demandOwner: (r && r.ownerName) || (r && r.ownerAccount) || '-',
    assignedTo: (r && r.ownerName) || (r && r.ownerAccount) || '-',
    estimateLaunch: (r && r.estimateLaunch) || '-',
    hang: (r && r.hang) ? '1' : '0',
    category: (r && r.category) || '',
    poolId: (r && r.poolId) || 0,
    poolName: (r && r.poolName) || '',
    nodeType: 'biz',
    parentId: null
  };
}

function buildDemandWorkspaceMyRelation() {
  if (demandCommonFilter === 'myLead') return 'lead';
  if (demandCommonFilter === 'needFocus') return 'follow';
  return '';
}

function buildDemandWorkspaceQueryParams(opts) {
  opts = opts || {};
  const params = new URLSearchParams();
  const keyword = opts.keyword != null
    ? String(opts.keyword)
    : String(document.getElementById('demandKeywordFilter')?.value || '').trim();
  if (keyword) params.set('keyword', keyword);

  const statuses = opts.statuses != null
    ? opts.statuses
    : Array.from(window.demandZentaoStatusSelection || []);
  if (Array.isArray(statuses) && statuses.length) {
    params.set('statuses', statuses.map((s) => String(s).toLowerCase()).join(','));
  }

  const agiles = opts.agiles != null
    ? opts.agiles
    : Array.from(window.demandAgileSelection || []);
  if (Array.isArray(agiles) && agiles.length) {
    params.set('agiles', agiles.join(','));
  }

  const systems = opts.systems != null
    ? opts.systems
    : Array.from(window.demandSystemSelection || []);
  if (Array.isArray(systems) && systems.length) {
    params.set('systems', systems.join(','));
  }

  const pools = opts.poolIds != null
    ? opts.poolIds
    : Array.from(window.demandPoolSelection || []);
  if (Array.isArray(pools) && pools.length) {
    params.set('poolIds', pools.join(','));
  }

  const suspended = opts.suspended != null
    ? opts.suspended
    : ((document.getElementById('demandSuspendedFilter')?.value || '') === 'yes');
  if (suspended) params.set('suspended', 'yes');

  const myRelation = opts.myRelation != null ? opts.myRelation : buildDemandWorkspaceMyRelation();
  if (myRelation) params.set('myRelation', myRelation);

  const vs = opts.valueStreamStage != null
    ? opts.valueStreamStage
    : (window.demandValueStreamStageSelection || document.getElementById('demandValueStreamStageFilter')?.value || '');
  if (vs) params.set('valueStreamStage', String(vs));

  const scopeEl = document.getElementById('demandScopeFilter');
  const objectScope = opts.objectScope != null ? opts.objectScope : (scopeEl ? scopeEl.value : '');
  if (objectScope) params.set('objectScope', String(objectScope));

  const quickScope = opts.quickScope != null ? opts.quickScope : (demandCommonFilter || 'allDemands');
  if (quickScope) params.set('quickScope', String(quickScope));

  const pri = (document.getElementById('demandPriorityFilter')?.value || '').trim();
  if (pri) {
    const priMap = { P0: '0', P1: '1', P2: '2', P3: '3' };
    params.set('pris', priMap[pri] != null ? priMap[pri] : pri);
  }

  params.set('page', String(opts.page != null ? opts.page : (demandPage || 1)));
  params.set('pageSize', String(opts.pageSize != null ? opts.pageSize : (demandPageSize || 20)));
  return params;
}

async function requestDemandWorkspaceSearch(opts) {
  opts = opts || {};
  const params = buildDemandWorkspaceQueryParams(opts);
  const url = (typeof getWorkbenchApiBase === 'function' ? getWorkbenchApiBase() : '') + '/workbench/api/demands/search?' + params.toString();
  demandWorkspaceLoading = true;
  let res;
  try {
    res = await fetch(url, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' }
    });
  } catch (e) {
    console.warn('[demand-query] demands/search 不可用', e);
    applyDemandWorkspaceSearchResults({ total: 0, items: [] });
    return { total: 0, items: [] };
  }
  if (res.status === 401) {
    if (typeof redirectToLogin === 'function') redirectToLogin();
    applyDemandWorkspaceSearchResults({ total: 0, items: [] });
    return { total: 0, items: [] };
  }
  if (!res.ok) {
    console.warn('[demand-query] demands/search 返回异常', res.status);
    applyDemandWorkspaceSearchResults({ total: 0, items: [] });
    return { total: 0, items: [] };
  }
  let json;
  try {
    json = await res.json();
  } catch (e) {
    console.warn('[demand-query] demands/search 解析失败', e);
    applyDemandWorkspaceSearchResults({ total: 0, items: [] });
    return { total: 0, items: [] };
  }
  const payload = (json && json.success === true)
    ? { total: json.total || 0, items: json.items || [] }
    : { total: 0, items: [] };
  applyDemandWorkspaceSearchResults(payload);
  return payload;
}

/**
 * chip 计数：GET /workbench/api/demands/count（后端 PR-1 并行开发）。
 * 未就绪时返回 null，调用方保留既有数字或跳过更新。
 * 期望响应: { success, counts: { scopeAll, scopeBiz, scopeRd, myRelated, myLead, blocked, overdue, needFocus, status: { draft: N, ... } } }
 */
async function requestDemandWorkspaceCount(opts) {
  opts = opts || {};
  const params = buildDemandWorkspaceQueryParams(Object.assign({}, opts, { page: 1, pageSize: 1 }));
  params.delete('page');
  params.delete('pageSize');
  const url = (typeof getWorkbenchApiBase === 'function' ? getWorkbenchApiBase() : '') + '/workbench/api/demands/count?' + params.toString();
  try {
    const res = await fetch(url, {
      credentials: 'include',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' }
    });
    if (!res.ok) return null;
    const json = await res.json();
    if (!json || json.success !== true) return null;
    return json.counts || json;
  } catch (e) {
    return null;
  }
}

function applyDemandWorkspaceSearchResults(payload) {
  demandWorkspaceLoading = false;
  const list = (payload && Array.isArray(payload.items)) ? payload.items : [];
  window.demandPoolLabelMap = window.demandPoolLabelMap || {};
  list.forEach(function (r) {
    const id = String((r && r.poolId) || '').trim();
    const name = String((r && r.poolName) || '').trim();
    if (id && name) window.demandPoolLabelMap[id] = name;
  });
  demandWorkspaceRows = list.map(mapDemandSearchItemToRow);
  demandTotal = Number(payload && payload.total) || list.length;
  DEMAND_TREE_DATA = demandWorkspaceRows.slice();
  if (demandCurrentView === 'list' && typeof renderDemandListView === 'function') {
    try { renderDemandListView(); } catch (e) { console.warn('renderDemandListView failed', e); }
  } else if (demandCurrentView === 'kanban' && typeof renderDemandKanbanView === 'function') {
    try { renderDemandKanbanView(); } catch (e) {}
  } else if (demandCurrentView === 'group' && typeof renderDemandGroupView === 'function') {
    try { renderDemandGroupView(); } catch (e) {}
  }
  if (typeof renderDemandAnalyticsPanel === 'function') {
    try { renderDemandAnalyticsPanel(); } catch (e) {}
  }
}

function triggerDemandWorkspaceRefresh(opts) {
  if (demandWorkspaceRefreshTimer) clearTimeout(demandWorkspaceRefreshTimer);
  demandWorkspaceRefreshTimer = setTimeout(function () {
    requestDemandWorkspaceSearch(opts || {});
    if (typeof refreshDemandWorkspaceHeaderCounts === 'function') {
      refreshDemandWorkspaceHeaderCounts();
    }
  }, 120);
}

function ensureDemandWorkspaceInitialLoad() {
  if (demandWorkspaceInitialized) return;
  demandWorkspaceInitialized = true;
  triggerDemandWorkspaceRefresh({ page: 1 });
}

window.buildDemandWorkspaceQueryParams = buildDemandWorkspaceQueryParams;
window.requestDemandWorkspaceSearch = requestDemandWorkspaceSearch;
window.requestDemandWorkspaceCount = requestDemandWorkspaceCount;
window.ensureDemandWorkspaceInitialLoad = ensureDemandWorkspaceInitialLoad;
window.triggerDemandWorkspaceRefresh = triggerDemandWorkspaceRefresh;
window.demandStatusCssClass = demandStatusCssClass;
