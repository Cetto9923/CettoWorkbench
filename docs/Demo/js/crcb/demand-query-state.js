// =============================================================================
// 文件: web/static/workbench/po/demand-query-state.js
// 模块: PO · 需求查询 V1.3
// 职责: chip 状态、默认口径、条件条、清空/重置、价值流高级筛选写入。
// 依赖: demand-query-api.js (DEMAND_ZT_STATUS_CHIPS)
//       demand-workspace.js (getDemandAgileOptions 等)
// =============================================================================

window.demandZentaoStatusSelection = window.demandZentaoStatusSelection || new Set();
window.demandPoolSelection = window.demandPoolSelection || new Set();
window.demandPoolLabelMap = window.demandPoolLabelMap || {};
window.demandValueStreamStageSelection = window.demandValueStreamStageSelection || '';

/** 价值流阶段：仅高级筛选，不进顶部 status chip */
const DEMAND_VALUE_STREAM_FILTER_OPTIONS = typeof getValueStreamStageOptions === 'function'
  ? getValueStreamStageOptions().map(function (label) {
      const keyMap = {
        '受理': 'accept', '澄清': 'clarify', '排期': 'schedule', '提测': 'developing',
        '测试': 'testing', '验收': 'acceptance', '发起交付': 'deliver', '发布': 'release', '评价反馈': 'feedback'
      };
      return { key: keyMap[label] || label, label: label };
    })
  : [
      { key: 'accept', label: '受理' }, { key: 'clarify', label: '澄清' }, { key: 'schedule', label: '排期' },
      { key: 'developing', label: '提测' }, { key: 'testing', label: '测试' }, { key: 'acceptance', label: '验收' },
      { key: 'deliver', label: '发起交付' }, { key: 'release', label: '发布' }, { key: 'feedback', label: '评价反馈' }
    ];

function getDemandPoolLabel(poolId) {
  const key = String(poolId || '').trim();
  if (!key) return '';
  return (window.demandPoolLabelMap && window.demandPoolLabelMap[key]) || key;
}

function mapValueStageLabelToFilterKey(label) {
  const hit = DEMAND_VALUE_STREAM_FILTER_OPTIONS.find(function (o) { return o.label === label; });
  return hit ? hit.key : '';
}

function setDemandValueStreamStageFilter(key) {
  window.demandValueStreamStageSelection = key || '';
  const el = document.getElementById('demandValueStreamStageFilter');
  if (el) el.value = window.demandValueStreamStageSelection;
}

function applyDemandValueStreamStageFromLabels(labels) {
  const list = Array.isArray(labels) ? labels : [];
  const key = list.length ? mapValueStageLabelToFilterKey(list[0]) : '';
  setDemandValueStreamStageFilter(key);
}

function syncDemandStatusTabs() {
  const host = document.getElementById('demandStatusTabs');
  if (!host) return;
  const selected = Array.from(window.demandZentaoStatusSelection || []);
  const active = selected.length === 1 ? selected[0] : '';
  const options = DEMAND_ZT_STATUS_CHIPS || [];
  host.innerHTML = [
    '<button type="button" class="dw-scope-chip' + (!active ? ' active' : '') + '" data-dw-status="" onclick="setDemandStatusFilter(\'\')">全部</button>',
    ...options.map(function (it) {
      const esc = String(it.key).replace(/'/g, "\\'");
      return '<button type="button" class="dw-scope-chip' + (active === it.key ? ' active' : '') + '" data-dw-status="' + it.key + '" onclick="setDemandStatusFilter(\'' + esc + '\')">' + it.label + '</button>';
    })
  ].join('');
}

function setDemandStatusFilter(key) {
  const next = key || '';
  window.demandZentaoStatusSelection = next ? new Set([next]) : new Set();
  syncDemandStatusTabs();
  if (typeof applyDemandFilters === 'function') applyDemandFilters();
}

function setDemandCommonFilter(key) {
  demandCommonFilter = key || '';
  document.querySelectorAll('.dw-common-chip').forEach(function (el) {
    const filterKey = el.dataset.dwFilter || '';
    const isAllDemands = filterKey === 'allDemands' && (demandCommonFilter === 'allDemands' || !demandCommonFilter);
    const isActive = isAllDemands || (!!demandCommonFilter && filterKey === demandCommonFilter);
    el.classList.toggle('active', isActive);
  });
  if (key === 'allDemands' || !key) {
    currentDemandPerspective = 'all';
    currentDemandActionFilter = 'all';
  } else if (key === 'myRelated') {
    currentDemandPerspective = 'my';
    currentDemandActionFilter = 'all';
  } else if (key === 'myLead') {
    currentDemandPerspective = 'myLead';
    currentDemandActionFilter = 'all';
  } else if (key === 'blocked') {
    currentDemandActionFilter = 'blocked';
  } else if (key === 'overdue') {
    currentDemandActionFilter = 'overdue';
  } else if (key === 'needFocus') {
    currentDemandActionFilter = 'issueRisk';
  }
  if (typeof syncDemandActionChips === 'function') syncDemandActionChips();
}

function applyDemandCommonFilter(key) {
  const next = demandCommonFilter === key ? '' : key;
  if (next === '') {
    setDemandCommonFilter('allDemands');
  } else {
    setDemandCommonFilter(next);
  }
  if (key === 'thisWeek' && next === 'thisWeek') {
    currentDemandActionFilter = 'all';
    const now = new Date();
    const day = now.getDay() || 7;
    const mon = new Date(now);
    mon.setDate(now.getDate() - day + 1);
    const sun = new Date(mon);
    sun.setDate(mon.getDate() + 6);
    const fmt = function (d) { return d.toISOString().slice(0, 10); };
    const from = document.getElementById('demandOnlineFromFilter');
    const to = document.getElementById('demandOnlineToFilter');
    if (from) from.value = fmt(mon);
    if (to) to.value = fmt(sun);
  }
  if (typeof applyDemandFilters === 'function') applyDemandFilters();
}

function renderDemandConditionChips() {
  const host = document.getElementById('demandConditionChips');
  if (!host) return;
  const chips = [];
  const quickMap = {
    allDemands: '全部需求',
    myRelated: '与我相关',
    myLead: '我负责',
    blocked: '有卡点',
    overdue: '超期',
    needFocus: '我关注'
  };
  if (demandCommonFilter && quickMap[demandCommonFilter]) {
    chips.push({ key: 'quick', label: quickMap[demandCommonFilter], clear: "dismissDemandCondition('quick')" });
  }
  const scopeRaw = document.getElementById('demandScopeFilter')?.value || '';
  const scope = scopeRaw === 'subBiz' ? 'biz' : scopeRaw;
  if (scope) {
    const scopeLab = ({ biz: '业务需求', rd: '研发需求' })[scope] || scope;
    chips.push({ key: 'scope', label: scopeLab, clear: "dismissDemandCondition('scope')" });
  }
// V1.3 残留修复: 防御 DOM 对象 / 非字符串值, 防止 chip 文案退化成 toString 默认形式。
// raw 值只接受 string; 若被外部错误地写入 select 元素本身, 降级为空字符串, chip 不渲染。
  const vsKeyRaw = window.demandValueStreamStageSelection || document.getElementById('demandValueStreamStageFilter')?.value || '';
  const vsKey = typeof vsKeyRaw === 'string'
    ? vsKeyRaw
    : ((vsKeyRaw && (vsKeyRaw.value || vsKeyRaw.text || vsKeyRaw.getAttribute && vsKeyRaw.getAttribute('value'))) || '');
  const vsKeyStr = typeof vsKey === 'string' ? vsKey : String(vsKey || '');
  if (vsKeyStr) {
    const hit = DEMAND_VALUE_STREAM_FILTER_OPTIONS.find(function (o) { return o.key === vsKeyStr; }) || {};
    const rawLab = hit.label || vsKeyStr;
    const vsLab = typeof rawLab === 'string' ? rawLab : (rawLab && (rawLab.value || rawLab.text || String(rawLab))) || vsKeyStr;
    chips.push({ key: 'vs', label: '价值流：' + vsLab, clear: "dismissDemandCondition('vs')" });
  }
  const zts = Array.from(window.demandZentaoStatusSelection || []);
  if (zts.length) {
    const ztLabel = zts.length === 1
      ? (demandStatusChipLabel(zts[0]) || zts[0])
      : zts.map(function (k) { return demandStatusChipLabel(k) || k; }).join('/');
    chips.push({ key: 'zt', label: '状态：' + ztLabel, clear: "dismissDemandCondition('zt')" });
  }
  const pools = Array.from(window.demandPoolSelection || []);
  if (pools.length) {
    const poolLabels = pools.map(getDemandPoolLabel).filter(Boolean);
    chips.push({ key: 'pool', label: '需求池：' + (poolLabels.length > 2 ? poolLabels.length + '项' : poolLabels.join('/')), clear: "dismissDemandCondition('pool')" });
  }
  const systems = Array.from(window.demandSystemSelection || []);
  if (systems.length) chips.push({ key: 'system', label: '系统：' + (systems.length > 2 ? systems.length + '项' : systems.join('/')), clear: "dismissDemandCondition('system')" });
  const teams = Array.from(window.demandTeamSelection || []);
  if (teams.length) chips.push({ key: 'team', label: '团队：' + (teams.length > 2 ? teams.length + '项' : teams.join('/')), clear: "dismissDemandCondition('team')" });
  const agiles = Array.from(window.demandAgileSelection || []);
  if (agiles.length) chips.push({ key: 'agile', label: '小组：' + agiles.join('/'), clear: "dismissDemandCondition('agile')" });
  const kw = (document.getElementById('demandKeywordFilter')?.value || '').trim();
  if (kw) chips.push({ key: 'kw', label: '关键词：' + kw, clear: "dismissDemandCondition('kw')" });

  if (!chips.length) {
    host.innerHTML = '';
    host.style.display = 'none';
    return;
  }
  host.style.display = 'flex';
  host.innerHTML = '<span class="dw-condition-label">当前条件</span>' + chips.map(function (c) {
    return '<button type="button" class="dw-condition-chip" onclick="' + c.clear + '">' + escapeHtml(c.label) + ' <span class="dw-condition-x" aria-hidden="true">×</span></button>';
  }).join('') + '<button type="button" class="dw-condition-clear" onclick="clearDemandFilters()">清空全部</button>';
}

function dismissDemandCondition(kind, value) {
  if (kind === 'quick') {
    setDemandCommonFilter('allDemands');
  } else if (kind === 'scope') {
    if (typeof setDemandScopeFilter === 'function') setDemandScopeFilter('');
    return;
  } else if (kind === 'vs') {
    setDemandValueStreamStageFilter('');
  } else if (kind === 'zt') {
    window.demandZentaoStatusSelection = new Set();
    syncDemandStatusTabs();
  } else if (kind === 'pool') {
    window.demandPoolSelection = new Set();
  } else if (kind === 'system') {
    window.demandSystemSelection = new Set();
  } else if (kind === 'team') {
    window.demandTeamSelection = new Set();
  } else if (kind === 'agile') {
    window.demandAgileSelection = new Set();
    if (typeof syncDemandAgileTabs === 'function') syncDemandAgileTabs();
  } else if (kind === 'kw') {
    const el = document.getElementById('demandKeywordFilter');
    if (el) el.value = '';
  }
  if (typeof renderDemandDimensionSummary === 'function') renderDemandDimensionSummary();
  if (typeof applyDemandFilters === 'function') applyDemandFilters();
}

function resetDemandFiltersToDefault() {
  currentDemandPerspective = 'all';
  currentDemandActionFilter = 'all';
  demandCommonFilter = 'allDemands';
  window.demandSystemSelection = new Set(DEMAND_DEFAULT_SCOPE.systems);
  window.demandAgileSelection = new Set(DEMAND_DEFAULT_SCOPE.agiles);
  window.demandTeamSelection = new Set(DEMAND_DEFAULT_SCOPE.teams);
  window.demandPoolSelection = new Set();
  window.demandZentaoStatusSelection = new Set();
  setDemandValueStreamStageFilter('');
  const defaults = {
    demandScopeFilter: '',
    demandKeywordFilter: '',
    demandCategoryFilter: '',
    demandImportanceFilter: '',
    demandPriorityFilter: '',
    demandDeptFilter: '',
    demandAssigneeFilter: '',
    demandCreatedByFilter: '',
    demandProposerFilter: '',
    demandOwnerFilter: '',
    demandTesterFilter: '',
    demandDelayedFilter: '',
    demandSuspendedFilter: '',
    demandChangingFilter: '',
    demandApprovalFilter: '',
    demandIterationFilter: '',
    demandWindowFilter: '',
    demandCreatedFromFilter: '',
    demandCreatedToFilter: '',
    demandDeliveryFromFilter: '',
    demandDeliveryToFilter: '',
    demandOnlineFromFilter: '',
    demandOnlineToFilter: '',
    demandContainerFilter: ''
  };
  Object.entries(defaults).forEach(function (entry) {
    const el = document.getElementById(entry[0]);
    if (el) el.value = entry[1];
  });
  document.querySelectorAll('.perspective-tab').forEach(function (tab) {
    tab.classList.toggle('active', tab.dataset.perspective === 'all');
  });
  setDemandCommonFilter('allDemands');
  document.querySelectorAll('.dw-scope-seg .dw-scope-chip').forEach(function (chip) {
    chip.classList.toggle('active', (chip.dataset.dwScope || '') === '');
  });
  syncDemandStatusTabs();
  if (typeof syncDemandAgileTabs === 'function') syncDemandAgileTabs();
  if (typeof syncDemandActionChips === 'function') syncDemandActionChips();
  if (typeof renderDemandDimensionSummary === 'function') renderDemandDimensionSummary();
  renderDemandConditionChips();
}

function initDemandDefaultFilters() {
  if (!(window.demandAgileSelection && window.demandAgileSelection.size)) window.demandAgileSelection = new Set(DEMAND_DEFAULT_SCOPE.agiles);
  if (!(window.demandTeamSelection && window.demandTeamSelection.size)) window.demandTeamSelection = new Set(DEMAND_DEFAULT_SCOPE.teams);
  if (!(window.demandSystemSelection && window.demandSystemSelection.size)) window.demandSystemSelection = new Set(DEMAND_DEFAULT_SCOPE.systems);
  if (!(window.demandPoolSelection && window.demandPoolSelection.size)) window.demandPoolSelection = new Set();
  if (!(window.demandZentaoStatusSelection && window.demandZentaoStatusSelection.size)) window.demandZentaoStatusSelection = new Set();
  if (!demandCommonFilter) demandCommonFilter = 'allDemands';
  setDemandCommonFilter(demandCommonFilter);
  parseDemandQueryDeepLink();
  syncDemandStatusTabs();
}

function parseDemandQueryDeepLink() {
  try {
    const params = new URLSearchParams(location.search);
    const vs = String(params.get('valueStreamStage') || '').trim();
    if (vs) setDemandValueStreamStageFilter(vs);
    const st = String(params.get('status') || params.get('statuses') || '').trim();
    if (st) window.demandZentaoStatusSelection = new Set([st.split(',')[0].trim().toLowerCase()]);
  } catch (e) {}
}

function clearDemandFilters() {
  resetDemandFiltersToDefault();
  demandPage = 1;
  if (typeof triggerDemandWorkspaceRefresh === 'function') triggerDemandWorkspaceRefresh();
  if (typeof showToast === 'function') showToast('已恢复默认工作口径');
}

async function refreshDemandWorkspaceHeaderCounts() {
  const counts = typeof requestDemandWorkspaceCount === 'function'
    ? await requestDemandWorkspaceCount()
    : null;
  const setCount = function (id, count) {
    const el = document.getElementById(id);
    if (el && count != null) el.textContent = String(count);
  };
  if (counts) {
    setCount('dwCountScopeAll', counts.scopeAll);
    setCount('dwCountScopeBiz', counts.scopeBiz);
    setCount('dwCountScopeRd', counts.scopeRd);
    setCount('dwCountMyRelated', counts.myRelated);
    setCount('dwCountMyLead', counts.myLead);
    setCount('dwCountBlocked', counts.blocked);
    setCount('dwCountOverdue', counts.overdue);
    setCount('dwCountNeedFocus', counts.needFocus);
    return;
  }
  if (typeof renderDemandHeaderCountsLegacy === 'function') renderDemandHeaderCountsLegacy();
}

window.syncDemandStatusTabs = syncDemandStatusTabs;
window.setDemandStatusFilter = setDemandStatusFilter;
window.setDemandCommonFilter = setDemandCommonFilter;
window.applyDemandCommonFilter = applyDemandCommonFilter;
window.renderDemandConditionChips = renderDemandConditionChips;
window.dismissDemandCondition = dismissDemandCondition;
window.resetDemandFiltersToDefault = resetDemandFiltersToDefault;
window.initDemandDefaultFilters = initDemandDefaultFilters;
window.clearDemandFilters = clearDemandFilters;
window.setDemandValueStreamStageFilter = setDemandValueStreamStageFilter;
window.applyDemandValueStreamStageFromLabels = applyDemandValueStreamStageFromLabels;
window.refreshDemandWorkspaceHeaderCounts = refreshDemandWorkspaceHeaderCounts;
window.getDemandPoolLabel = getDemandPoolLabel;
