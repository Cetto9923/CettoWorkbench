let demandCurrentView = 'list';
let demandPage = 1;
let demandPageSize = 20;
let demandTotal = 0;
let currentDemandActionFilter = 'all';
let demandCommonFilter = 'allDemands';
window.expandedDemandIds = window.expandedDemandIds || new Set();
window.demandSystemSelection = window.demandSystemSelection || new Set();
window.demandAgileSelection = window.demandAgileSelection || new Set();
window.demandTeamSelection = window.demandTeamSelection || new Set();
window.demandSavedQueries = window.demandSavedQueries || [];
const DEMAND_DEFAULT_SCOPE = {
  agiles: [],
  teams: [],
  systems: []
};
/** 价值流阶段：第 3 行 chip + 高级筛选可选（见 demand-query-state.js） */
const DEMAND_VALUE_STAGE_OPTIONS = typeof getValueStreamStageOptions === 'function'
  ? getValueStreamStageOptions()
  : ['受理','澄清','排期','提测','测试','验收','发起交付','发布','评价反馈'];
const DEMAND_ACTION_FILTERS = [
  { key:'all', label:'全部' },
  { key:'today', label:'今日必推' },
  { key:'overdue', label:'超期' },
  { key:'blocked', label:'阻塞' },
  { key:'bizConfirm', label:'待业务确认' },
  { key:'unscheduled', label:'待排期' },
  { key:'unaccepted', label:'待验收' },
  { key:'undelivered', label:'待交付' },
  { key:'issueRisk', label:'问题风险' }
];

let DEMAND_TREE_DATA = [];




function getDemandSystemOptions() {
  const systems = new Set();
  DEMAND_TREE_DATA.forEach((item) => (item.systems || []).forEach((name) => systems.add(name)));
  return Array.from(systems).sort((a, b) => a.localeCompare(b, 'zh-CN'));
}
function normalizeDemandAgile(value) {
  if (!value) return '';
  return /^agile(\d+)$/i.test(value) ? `敏捷小组-${value.match(/^agile(\d+)$/i)[1]}` : value;
}
function getDemandAgileOptions() {
  return Array.from(new Set(DEMAND_TREE_DATA.map((item) => normalizeDemandAgile(item.agile)).filter(Boolean))).sort((a, b) => a.localeCompare(b, 'zh-CN'));
}
/** 当前用户参与的敏捷小组（BUG-18 fix 2026-08-14 Plan-D：读共享选项，空时返 []，不再 hardcode） */
function getDemandMyAgileGroups() {
  try {
    if (typeof getSharedAgileOptions === 'function') {
      return getSharedAgileOptions().map(function(o){ return o.label; });
    }
  } catch (e) {}
  return [];
}
function syncDemandAgileTabs() {
  const host = document.getElementById('demandAgileTabs');
  if (!host) return;
  const selected = Array.from(window.demandAgileSelection || []);
  const active = selected.length === 1 ? selected[0] : '';
  const groups = getDemandMyAgileGroups();
  host.innerHTML = [
    `<button type="button" class="dw-scope-chip${!active ? ' active' : ''}" data-dw-agile="" onclick="setDemandAgileFilter('')">全部</button>`,
    ...groups.map((name) =>
      `<button type="button" class="dw-scope-chip${active === name ? ' active' : ''}" data-dw-agile="${name.replace(/"/g, '&quot;')}" onclick="setDemandAgileFilter('${name.replace(/'/g, "\\'")}')">${name}</button>`
    )
  ].join('');
}
function setDemandAgileFilter(name) {
  const next = name || '';
  window.demandAgileSelection = next ? new Set([next]) : new Set();
  syncDemandAgileTabs();
  applyDemandFilters();
}
function getDemandTeamOptions() {
  return Array.from(new Set(DEMAND_TREE_DATA.map((item) => item.team).filter(Boolean))).sort((a, b) => a.localeCompare(b, 'zh-CN'));
}
function getDemandCurrentUser() {
  return (typeof getWorkbenchCurrentUserName === 'function' ? getWorkbenchCurrentUserName() : getCurrentPoUserLabel());
}
function getDemandItemValueStage(item) {
  // 与首页 VALUE_STREAM_STAGES.shortLabel 对齐；优先复用 computePoValueStage
  if (typeof computePoValueStage === 'function') {
    const bridge = {
      id: item.reqCode || item.id,
      phase: item.phase,
      status: item.status,
      phaseLabel: item.phase,
      risks: item.riskType ? [item.riskType] : [],
      nodeType: item.nodeType,
      // syncDemandTreeFromWorkItems 已写入真实 WorkItem 字段；此处必须透传，
      // 否则 computePoValueStage 会因缺少 valueStreamStage 打 [po-stage] missing 警告。
      valueStreamStage: item.valueStreamStage || '',
      workbenchStatus: item.workbenchStatus || item.status || '',
      zentaoStatus: item.zentaoStatus || '',
      actionType: item.actionType || '',
      isMyAction: item.isMyAction === true,
      isMyFollowUp: item.isMyFollowUp === true
    };
    // 研发需求：DEV- 前缀触发 computeRdPoValueStage
    if (item.nodeType === 'rd' && bridge.id && !String(bridge.id).startsWith('DEV-')) {
      bridge.id = 'DEV-' + bridge.id;
    }
    const key = computePoValueStage(bridge);
    if (key && VALUE_STREAM_STAGES[key]) return VALUE_STREAM_STAGES[key].shortLabel;
  }
  const text = `${item.valueStageLabel || ''} ${item.phase || ''} ${item.status || ''}`;
  if (/受理|草稿|待评审|驳回/.test(text)) return '受理';
  if (/澄清|已评审|梳理/.test(text)) return '澄清';
  if (/排期|已澄清|待排期|转研|建任务/.test(text)) return '排期';
  if (/提测/.test(text)) return '提测';
  if (/开发|研发中|developing/.test(text)) return '提测';
  if (/联调|测试中|testing/.test(text)) return '测试';  // 2026-08-17 V0.8: 联调测试→测试
  if (/验收|待验收|accept/.test(text)) return '验收';
  if (/发起交付|待交付|deliver/.test(text)) return '发起交付';  // V0.8: 交付→发起交付
  if (/发布|上线|released/.test(text)) return '发布';  // V0.8: 新增发布阶段
  if (/评价|反馈|关闭/.test(text)) return '评价反馈';
  return '澄清';
}
function getDemandItemZentaoStatus(item) {
  const raw = `${item.zentaoStatusLabel || ''} ${item.zentaoStatus || ''} ${item.phase || ''} ${item.status || ''}`;
  if (/草稿/.test(raw)) return '草稿';
  if (/待评审|wait/.test(raw)) return '待评审';
  if (/已评审|active/.test(raw)) return '已评审';
  if (/已澄清|clarified/.test(raw)) return '已澄清';
  if (/挂起|suspended/.test(raw)) return '已挂起';
  if (/已关闭|closed|完成|done/.test(raw)) return '已关闭';
  if (/已驳回|refuse/.test(raw)) return '已驳回';
  if (/开发|developing|研发/.test(raw)) return '开发中';
  if (/测试|testing|联调/.test(raw)) return '测试中';
  if (/待验收|waitacceptance/.test(raw)) return '待验收';
  if (/已验收|acceptanced/.test(raw)) return '已验收';
  if (/待交付|waitdeliver/.test(raw)) return '待交付';
  if (/发布|released|上线/.test(raw)) return '已发布';
  return '草稿';
}
function getDemandItemIteration(item) {
  if (item.iterationLabel) return item.iterationLabel;
  const plan = String(item.iterationPlan || item.windowName || '');
  if (plan && plan !== '-' && plan !== '—' && plan !== '未排窗口' && plan !== '待交付') {
    return (typeof getVersionWindowShortName === 'function') ? getVersionWindowShortName(plan) : plan;
  }
  const row = (typeof getScheduleRowByDemandItem === 'function') ? getScheduleRowByDemandItem(item) : null;
  if (row) {
    const win = String(row.windowName || row.iterationPlan || '');
    if (win && win !== '-' && win !== '—' && win !== '未排窗口' && win !== '待交付') {
      return (typeof getVersionWindowShortName === 'function') ? getVersionWindowShortName(win) : win;
    }
  }
  return '未归属';
}
function getDemandItemWindowType(item) {
  return item.windowType || '常规';
}
function getDemandItemRelation(item) {
  const me = getDemandCurrentUser();
  if (item.createdBy === me) return '我创建';
  if (item.followed) return '我关注';
  if (item.owner === me || item.demandOwner === me || item.proposer === me) return '我牵头';
  const role = String(item.systemRole || '');
  if (role === 'main' || /主系统|主办|PO|需求负责人/.test(role)) return '我牵头';
  if (role === 'sub' || /配合|协作|参与/.test(role)) return '我配合';
  return '';
}
const DEMAND_TEAM_DEPT_MAP = {
  '对公团队': '企业金融部',
  '零售团队': '零售银行部',
  '平台团队': '支付清算部',
  '安全团队': '科技管理部',
  '信用卡团队': '零售银行部',
  '资金团队': '企业金融部',
  '营销团队': '零售银行部',
  '数据团队': '科技管理部',
  '测试团队': '科技管理部'
};
function getScheduleRowByDemandItem(item) {
  const code = String(item.reqCode || item.id || '');
  return (typeof scheduleData !== 'undefined' ? scheduleData : []).find((row) => row.id === code);
}
function getDemandItemProposer(item) {
  if (item.proposer && item.proposer !== '-') return item.proposer;
  const row = getScheduleRowByDemandItem(item);
  return demandNonBlankValue(row?.proposer, item.createdBy, item.demandOwner, item.owner, item.BRA, item.nodeType === 'rd' ? '研发团队' : '业务人员');
}
function getDemandItemDepartment(item) {
  if (item.department) return item.department;
  const row = getScheduleRowByDemandItem(item);
  if (row?.department) return row.department;
  if (item.team && DEMAND_TEAM_DEPT_MAP[item.team]) return DEMAND_TEAM_DEPT_MAP[item.team];
  const depts = ['企业金融部', '零售银行部', '支付清算部', '科技管理部'];
  const hash = String(item.id || '').split('').reduce((sum, ch) => sum + ch.charCodeAt(0), 0);
  return depts[hash % depts.length];
}

let demandRolePreset = 'default';
let demandAnalyticsVisible = false;

function getDemandFilteredFlatRows() {
  return DEMAND_TREE_DATA.filter((row) => demandTreeItemMatchesBarFilters(row));
}

function demandNonBlankValue() {
  for (let i = 0; i < arguments.length; i++) {
    const value = arguments[i];
    if (value === undefined || value === null) continue;
    const text = String(value).trim();
    if (text && text !== '-') return text;
  }
  return '';
}

function getDemandItemOwner(item) {
  return demandNonBlankValue(item.BRA, item.demandOwner, item.owner, item.nextOwner, item.assignee, getDemandItemProposer(item), getDemandCurrentUser());
}

function getDemandItemAssignee(item) {
  const direct = demandNonBlankValue(item.assignedTo, item.assignee, item.nextOwner, item.owner);
  if (direct) return direct;
  if (item.nodeType === 'rd') {
    const teamOwner = { '对公团队': '王工', '零售团队': '李工', '平台团队': '周工', '安全团队': '赵工', '测试团队': '测试组', '数据团队': '吴工' };
    return teamOwner[item.team] || '开发负责人';
  }
  return getDemandItemOwner(item);
}

function getDemandItemTester(item) {
  const row = getScheduleRowByDemandItem(item);
  return demandNonBlankValue(item.QD, item.tester, row?.testOwner, row?.tester, item.nodeType === 'rd' ? '测试组' : '待测试排期');
}

function getDemandItemAcceptOwner(item) {
  return demandNonBlankValue(item.RD, item.acceptOwner, item.businessAcceptOwner, getDemandItemProposer(item), '业务验收人');
}

function getDemandItemReviewer(item) {
  return demandNonBlankValue(item.reviewer, item.reviewOwner, item.nodeType === 'rd' ? '技术评审人' : '业务评审委员会');
}

function getDemandItemCreatedDate(item) {
  if (item.createdDate) return item.createdDate;
  const row = getScheduleRowByDemandItem(item);
  if (row?.createdDate) return row.createdDate;
  const hash = String(item.id || '').split('').reduce((sum, ch) => sum + ch.charCodeAt(0), 0);
  const day = String((hash % 28) + 1).padStart(2, '0');
  return `2026-04-${day}`;
}

function getDemandItemDeliveryDate(item) {
  if (item.deliverDate) return item.deliverDate;
  if (item.deliveryDate) return item.deliveryDate;
  const row = getScheduleRowByDemandItem(item);
  if (row?.deliveryDate) return row.deliveryDate;
  if (/交付|上线|关闭|完成/.test(String(item.phase || '') + String(item.status || ''))) {
    return demandNonBlankValue(item.deadline, item.estimateLaunch, item.onlineDate, '待回填');
  }
  return '未交付';
}

function getDemandItemMainSystem(item) {
  if (item.mainSystem) return item.mainSystem;
  const list = item.systems || [];
  return list[0] || '-';
}

function getDemandItemCategoryLabel(item) {
  const category = item.category || '';
  const map = {
    feature: '新增需求',
    feature_1: '拓客促活',
    feature_2: '对抗竞品',
    feature_3: '监管要求',
    experience: '体验优化',
    BUG: 'BUG',
    tecopt: '技术优化',
    performance: '性能',
    safe: '安全',
    DATACG: '数据变更(研发迭代)',
    datachange: '数据变更',
    dataexport: '数据导出',
    other: '其他'
  };
  if (map[category] || category) return map[category] || category;
  if (/BUG|缺陷|异常/.test(String(item.title || '') + String(item.reqCode || ''))) return 'BUG';
  if (/规则|审批|流程|登录|接口|参数|结算/.test(String(item.title || ''))) return '功能优化';
  if (/数据|报表|导出|ETL/.test(String(item.title || ''))) return '数据变更';
  return item.nodeType === 'rd' ? '研发实现' : '业务优化';
}

function getDemandItemSourceLabel(item) {
  const source = item.source || '';
  const map = { user: '用户反馈', market: '市场', operation: '运营', project: '项目', feedback: '反馈', other: '其他' };
  if (map[source] || source) return map[source] || source;
  const dept = getDemandItemDepartment(item);
  return dept && dept !== '-' ? dept : '业务提出';
}

function getDemandItemSeverityLabel(item) {
  const severity = String(item.severity || item.importance || '');
  const map = { '1': '关键', '2': '重要', '3': '一般', '4': '低' };
  if (map[severity] || severity) return map[severity] || severity;
  if (item.priority === 'P0' || /阻塞|超期|风险/.test(String(item.status || '') + String(item.riskType || ''))) return '关键';
  if (item.priority === 'P1') return '重要';
  return '一般';
}

function getDemandItemDeadline(item) {
  return demandNonBlankValue(item.deadline, item.estimateLaunch, item.onlineDate, '待确认');
}

function getDemandItemEstimateLaunch(item) {
  return demandNonBlankValue(item.estimateLaunch, item.onlineDate, item.deadline, '待定');
}

function getDemandItemSchedulePlanDate(item) {
  return demandNonBlankValue(item.schedulePlanDate, item.planDate, item.reviewDate, '待排期');
}

function getDemandItemHangLabel(item) {
  return String(item.hang || '') === '1' || getDemandItemZentaoStatus(item) === '已挂起' ? '是' : '否';
}

function getDemandItemChangeLabel(item) {
  return String(item.isChange || '') === 'changing' || /变更/.test(String(item.phase || '')) ? '是' : '否';
}

function getDemandItemManagerReviewLabel(item) {
  const value = String(item.isManagerReview || item.reviewStatus || '');
  return value && !['0', 'done', 'finished'].includes(value) ? '是' : '否';
}

function getDemandItemFocusLabel(item) {
  return String(item.isNeedFocus || '') === '1' || item.priority === 'P0' ? '是' : '否';
}

function parseWorkbenchRoleValue(raw) {
  const v = String(raw || '').trim().replace(/=+$/, '');
  if (/^(po|sm|pm|biz|lead|dev|qa|pmo)$/.test(v)) return v;
  return '';
}
function detectWorkbenchRole() {
  try {
    const own = parseWorkbenchRoleValue(new URLSearchParams(location.search).get('role'));
    const embedded = document.body.classList.contains('workbench-embed') || (window.self !== window.top);
    if (!embedded && own) return own;
    try {
      const parentRole = parseWorkbenchRoleValue(new URLSearchParams(window.parent.location.search).get('role'));
      if (parentRole) return parentRole;
    } catch (e0) {}
    if (own) return own;
  } catch (e1) {}
  try {
    const stored = parseWorkbenchRoleValue(localStorage.getItem('wb_active_role'));
    if (stored) return stored;
  } catch (e2) {}
  try {
    if (window.frameElement && window.frameElement.dataset && window.frameElement.dataset.role) {
      const fr = parseWorkbenchRoleValue(window.frameElement.dataset.role);
      if (fr) return fr;
    }
  } catch (e3) {}
  try {
    if (window.RoleSwitcher && typeof RoleSwitcher.detectRole === 'function') return RoleSwitcher.detectRole();
  } catch (e4) {}
  // R15 §35: 角色未知不得静默 — 无条件回退 po 前必须可观测 (console.warn 一次/页)。
  try {
    if (!window.__wbRoleDetectWarned) {
      window.__wbRoleDetectWarned = true;
      console.warn('[workbench] detectWorkbenchRole 全部来源未命中, 显式回退 po (URL/embed/localStorage/frame/RoleSwitcher)');
    }
  } catch (e5) {}
  return 'po';
}
function fieldViewPresetForRole(role) {
  if (role === 'biz') return 'biz';
  if (role === 'po' || role === 'lead') return 'lead';
  if (role === 'dev' || role === 'qa' || role === 'pm') return 'dev';
  return 'default';
}
const DEMAND_FIELD_VIEW_STORE_KEY = 'wb-demand-fieldview';
function loadDemandFieldViewOverride(role) {
  try {
    const raw = localStorage.getItem(DEMAND_FIELD_VIEW_STORE_KEY);
    if (!raw) return '';
    const map = JSON.parse(raw) || {};
    return map[role] || '';
  } catch (e) { return ''; }
}
function saveDemandFieldViewOverride(role, preset) {
  try {
    const raw = localStorage.getItem(DEMAND_FIELD_VIEW_STORE_KEY);
    const map = raw ? (JSON.parse(raw) || {}) : {};
    map[role] = preset;
    localStorage.setItem(DEMAND_FIELD_VIEW_STORE_KEY, JSON.stringify(map));
  } catch (e) {}
}
let lastSyncedWorkbenchRole = null;
function syncDemandFieldViewFromRole(opts) {
  opts = opts || {};
  const role = detectWorkbenchRole();
  // 默认跟随当前角色；用户手动切过则沿用他的选择
  const preset = loadDemandFieldViewOverride(role) || fieldViewPresetForRole(role);
  if (!opts.force && lastSyncedWorkbenchRole === role) return;
  lastSyncedWorkbenchRole = role;
  applyDemandRolePreset(preset, { silent: opts.silent !== false, skipRender: !!opts.skipRender });
}
function applyDemandRolePreset(role, opts) {
  opts = opts || {};
  demandRolePreset = role || 'default';
  if (!opts.silent) saveDemandFieldViewOverride(detectWorkbenchRole(), demandRolePreset);
  document.querySelectorAll('.dw-role-preset').forEach((el) => {
    el.classList.toggle('active', el.dataset.demandRole === demandRolePreset);
  });
  if (role === 'lead') {
    saveColumnConfigStore('demand', new Set(['id','title','priority','severity','category','status','assignee','owner','schedulePlan','online','hang','watch','action']));
    currentDemandActionFilter = 'all';
  } else if (role === 'biz') {
    saveColumnConfigStore('demand', new Set(['id','title','priority','category','source','status','assignee','owner','online','delivery','reason','action']));
    currentDemandActionFilter = 'all';
  } else if (role === 'dev') {
    saveColumnConfigStore('demand', new Set(['id','title','priority','status','phase','assignee','tester','acceptOwner','schedulePlan','online','hang','change','action']));
    currentDemandActionFilter = 'all';
  } else {
    saveColumnConfigStore('demand', new Set(columnConfigDefaults.demand));
  }
  if (!opts.skipRender) {
    renderDemandAnalyticsPanel();
    renderDemandListView();
  }
  if (!opts.silent) {
    showToast(role === 'lead' ? '已切换常用视图：PO 列' : role === 'biz' ? '已切换常用视图：业务列' : role === 'dev' ? '已切换常用视图：研发列' : '已切换常用视图：默认列');
  }
}

function toggleDemandAnalyticsPanel() {
  demandAnalyticsVisible = !demandAnalyticsVisible;
  renderDemandAnalyticsPanel();
}

function setDemandScopeFilter(scope) {
  const normalized = scope === 'subBiz' ? 'biz' : (scope || '');
  const el = document.getElementById('demandScopeFilter');
  if (el) el.value = normalized;
  document.querySelectorAll('.dw-scope-seg .dw-scope-chip').forEach((chip) => {
    chip.classList.toggle('active', (chip.dataset.dwScope || '') === normalized);
  });
  refreshDemandWorkspaceHeaderCounts();
  applyDemandFilters();
}

function syncDemandActionChips() {
  const host = document.getElementById('demandActionChips');
  if (!host) return;
  host.innerHTML = DEMAND_ACTION_FILTERS.map((item) => `
    <div class="sv-pill${currentDemandActionFilter === item.key ? ' active' : ''}" onclick="setDemandActionFilter('${item.key}')">${item.label}</div>
  `).join('');
}
function setDemandActionFilter(filterKey) {
  currentDemandActionFilter = filterKey;
  demandPage = 1;
  syncDemandActionChips();
  applyDemandFilters();
}
function renderDemandFilterSummary() {
  const host = document.getElementById('demandFilterSummary');
  if (!host) return;
  host.style.display = 'none';
  renderDemandConditionChips();
}

function renderDemandDimensionSummary() {
  const agile = Array.from(window.demandAgileSelection || []);
  const team = Array.from(window.demandTeamSelection || []);
  const system = Array.from(window.demandSystemSelection || []);
  const pool = Array.from(window.demandPoolSelection || []);
  const zt = Array.from(window.demandZentaoStatusSelection || []);
  const summary = document.getElementById('demandDimensionSummary');
  if (summary) {
    const total = agile.length + team.length + system.length + pool.length + zt.length;
    const advancedOpen = document.getElementById('demandPrimaryAdvancedBar')?.style.display !== 'none'
      || document.getElementById('demandAdvancedFilters')?.style.display !== 'none';
    summary.style.display = 'none';
    summary.innerHTML = total && advancedOpen ? `
      <div class="sv-pill"><i class="fas fa-circle-nodes"></i>需求状态：${zt.length ? zt.map((k) => demandStatusChipLabel(k)).join('/') : '全部'}</div>
      <div class="sv-pill"><i class="fas fa-database"></i>需求池：${pool.length ? pool.map((p) => escapeHtml(typeof getDemandPoolLabel === 'function' ? getDemandPoolLabel(p) : p)).join('/') : '全部'}</div>
      <div class="sv-pill"><i class="fas fa-people-group"></i>敏捷小组：${agile.length ? agile.map((a) => escapeHtml(a)).join('/') : '全部'}</div>
      <div class="sv-pill"><i class="fas fa-sitemap"></i>团队：${team.length}项</div>
      <div class="sv-pill"><i class="fas fa-cubes"></i>系统：${system.length}项</div>
    ` : '';
  }
  const tBtn = document.getElementById('demandTeamPickerBtn');
  const sBtn = document.getElementById('demandSystemPickerBtn');
  const pInput = document.getElementById('demandPoolComboInput');
  const ztBtn = document.getElementById('demandZentaoStatusPickerBtn');
  if (tBtn) tBtn.textContent = team.length ? `已选${team.length}个团队` : '选择团队';
  if (sBtn) sBtn.textContent = system.length ? `已选${system.length}个系统` : '选择系统';
  if (pInput && !pInput.value.trim()) pInput.placeholder = pool.length ? (pool.length === 1 ? (typeof getDemandPoolLabel === 'function' ? getDemandPoolLabel(pool[0]) : pool[0]) : `已选${pool.length}个需求池`) : '全部可访问需求池';
  if (ztBtn) {
    ztBtn.textContent = zt.length
      ? (zt.length === 1 ? demandStatusChipLabel(Array.from(zt)[0]) : `已选${zt.length}个状态`)
      : '全部状态';
  }
  syncDemandStatusTabs();
  if (typeof syncDemandStageTabs === 'function') syncDemandStageTabs();
  syncDemandAgileTabs();
  renderDemandFilterSummary();
}
let demandDimPickerType = 'system';
let demandDimPickerDraft = new Set();
function getDemandDimPickerConfig(type) {
  if (type === 'team') return { title: '选择团队', options: getDemandTeamOptions(), source: window.demandTeamSelection };
  if (type === 'pool') return { title: '选择需求池', options: getDemandPoolOptions(), source: window.demandPoolSelection };
  if (type === 'zentaoStatus') {
    return {
      title: '选择需求状态（禅道）',
      options: (DEMAND_ZT_STATUS_CHIPS || []).map((it) => it.key),
      labels: Object.fromEntries((DEMAND_ZT_STATUS_CHIPS || []).map((it) => [it.key, it.label])),
      source: window.demandZentaoStatusSelection
    };
  }
  return { title: '选择系统', options: getDemandSystemOptions(), source: window.demandSystemSelection };
}
function getDemandPoolOptions() {
  const pools = new Map();
  (demandWorkspaceRows || DEMAND_TREE_DATA || []).forEach((item) => {
    const id = String(item.poolId || item.pool || '').trim();
    if (!id) return;
    const label = String(item.poolName || '').trim() || (typeof getDemandPoolLabel === 'function' ? getDemandPoolLabel(id) : id);
    pools.set(id, label);
  });
  window.demandPoolLabelMap = window.demandPoolLabelMap || {};
  pools.forEach((label, id) => { if (label) window.demandPoolLabelMap[id] = label; });
  return Array.from(pools.entries()).sort((a, b) => a[1].localeCompare(b[1], 'zh-CN')).map((entry) => entry[0]);
}
function openDemandDimPicker(type) {
  demandDimPickerType = type;
  const cfg = getDemandDimPickerConfig(type);
  demandDimPickerDraft = new Set(Array.from(cfg.source || []));
  document.getElementById('demandDimModalTitle').textContent = `${cfg.title}（支持多选）`;
  const input = document.getElementById('demandDimSearchInput');
  if (input) input.value = '';
  renderDemandDimPickerOptions();
  document.getElementById('demandDimModal').classList.add('show');
  document.getElementById('demandDimModalOverlay').classList.add('show');
}
function renderDemandDimPickerOptions() {
  const host = document.getElementById('demandDimPickerOptions');
  if (!host) return;
  const cfg = getDemandDimPickerConfig(demandDimPickerType);
  const q = (document.getElementById('demandDimSearchInput')?.value || '').trim().toLowerCase();
  const options = cfg.options.filter((name) => !q || name.toLowerCase().includes(q));
  host.innerHTML = options.map((name) => {
    const label = (cfg.labels && cfg.labels[name]) || name;
    return `
    <label style="display:flex;align-items:center;gap:8px;padding:6px 4px;cursor:pointer">
      <input type="checkbox" ${demandDimPickerDraft.has(name) ? 'checked' : ''} onchange="toggleDemandDimPickerOption('${String(name).replace(/'/g, "\\'")}')">
      <span style="font-size:13px">${label}</span>
    </label>
  `;
  }).join('') || '<div style="font-size:12px;color:var(--t3);padding:8px">无匹配项</div>';
}
function toggleDemandDimPickerOption(name) {
  if (demandDimPickerDraft.has(name)) demandDimPickerDraft.delete(name);
  else demandDimPickerDraft.add(name);
  renderDemandDimPickerOptions();
}
function clearDemandDimPickerSelection() {
  demandDimPickerDraft = new Set();
  renderDemandDimPickerOptions();
}
function confirmDemandDimPicker() {
  if (demandDimPickerType === 'team') window.demandTeamSelection = new Set(Array.from(demandDimPickerDraft));
  else if (demandDimPickerType === 'pool') window.demandPoolSelection = new Set(Array.from(demandDimPickerDraft));
  else if (demandDimPickerType === 'zentaoStatus') window.demandZentaoStatusSelection = new Set(Array.from(demandDimPickerDraft));
  else window.demandSystemSelection = new Set(Array.from(demandDimPickerDraft));
  closeDemandDimPicker();
  renderDemandDimensionSummary();
  applyDemandFilters();
}
function closeDemandDimPicker() {
  document.getElementById('demandDimModal').classList.remove('show');
  document.getElementById('demandDimModalOverlay').classList.remove('show');
}
function openDemandPoolDropdown() {
  const combo = document.getElementById('demandPoolCombo');
  const dropdown = document.getElementById('demandPoolDropdown');
  if (!combo || !dropdown) return;
  combo.classList.add('open');
  dropdown.hidden = false;
  renderDemandPoolDropdownOptions();
}
function closeDemandPoolDropdown() {
  const combo = document.getElementById('demandPoolCombo');
  const dropdown = document.getElementById('demandPoolDropdown');
  if (combo) combo.classList.remove('open');
  if (dropdown) dropdown.hidden = true;
}
function renderDemandPoolDropdownOptions() {
  const host = document.getElementById('demandPoolDropdownOptions');
  if (!host) return;
  const q = (document.getElementById('demandPoolComboInput')?.value || '').trim().toLowerCase();
  const options = getDemandPoolOptions().filter((id) => {
    const label = typeof getDemandPoolLabel === 'function' ? getDemandPoolLabel(id) : id;
    return !q || String(label).toLowerCase().includes(q) || String(id).includes(q);
  });
  host.innerHTML = options.map((id) => {
    const label = typeof getDemandPoolLabel === 'function' ? getDemandPoolLabel(id) : id;
    return `
      <label class="demand-pool-option">
        <input type="checkbox" ${window.demandPoolSelection && window.demandPoolSelection.has(id) ? 'checked' : ''} onchange="toggleDemandPoolDropdownOption('${String(id).replace(/'/g, "\\'")}')">
        <span>${escapeHtml(label)}</span>
      </label>
    `;
  }).join('') || '<div class="demand-pool-empty">无匹配需求池</div>';
}
function toggleDemandPoolDropdownOption(id) {
  window.demandPoolSelection = window.demandPoolSelection || new Set();
  if (window.demandPoolSelection.has(id)) window.demandPoolSelection.delete(id);
  else window.demandPoolSelection.add(id);
  renderDemandDimensionSummary();
  renderDemandPoolDropdownOptions();
  applyDemandFilters();
}
function clearDemandPoolDropdownSelection() {
  window.demandPoolSelection = new Set();
  const input = document.getElementById('demandPoolComboInput');
  if (input) input.value = '';
  renderDemandDimensionSummary();
  renderDemandPoolDropdownOptions();
  applyDemandFilters();
}
function toggleDemandAdvancedFilters() {
  const panel = document.getElementById('demandAdvancedFilters');
  const open = panel && panel.style.display !== 'none';
  const next = open ? 'none' : 'grid';
  if (panel) panel.style.display = next;
  renderDemandDimensionSummary();
}
function saveCurrentDemandQuery() {
  const name = prompt('请输入查询条件名称（如：我的主系统-P1待排期）');
  if (!name) return;
  const snapshot = {
    name,
    perspective: currentDemandPerspective,
    actionFilter: currentDemandActionFilter,
    scope: document.getElementById('demandScopeFilter')?.value || '',
    keyword: document.getElementById('demandKeywordFilter')?.value || '',
    view: demandCurrentView,
    agiles: Array.from(window.demandAgileSelection || []),
    teams: Array.from(window.demandTeamSelection || []),
    systems: Array.from(window.demandSystemSelection || []),
    pools: Array.from(window.demandPoolSelection || []),
    zentaoStatuses: Array.from(window.demandZentaoStatusSelection || []),
    valueStreamStage: window.demandValueStreamStageSelection || '',
    category: document.getElementById('demandCategoryFilter')?.value || '',
    importance: document.getElementById('demandImportanceFilter')?.value || '',
    priority: document.getElementById('demandPriorityFilter')?.value || '',
    dept: document.getElementById('demandDeptFilter')?.value || '',
    assignee: document.getElementById('demandAssigneeFilter')?.value || '',
    createdBy: document.getElementById('demandCreatedByFilter')?.value || '',
    proposer: document.getElementById('demandProposerFilter')?.value || '',
    demandOwner: document.getElementById('demandOwnerFilter')?.value || '',
    tester: document.getElementById('demandTesterFilter')?.value || '',
    delayed: document.getElementById('demandDelayedFilter')?.value || '',
    suspended: document.getElementById('demandSuspendedFilter')?.value || '',
    changing: document.getElementById('demandChangingFilter')?.value || '',
    approval: document.getElementById('demandApprovalFilter')?.value || '',
    iteration: document.getElementById('demandIterationFilter')?.value || '',
    windowType: document.getElementById('demandWindowFilter')?.value || '',
    createdFrom: document.getElementById('demandCreatedFromFilter')?.value || '',
    createdTo: document.getElementById('demandCreatedToFilter')?.value || '',
    deliveryFrom: document.getElementById('demandDeliveryFromFilter')?.value || '',
    deliveryTo: document.getElementById('demandDeliveryToFilter')?.value || '',
    onlineFrom: document.getElementById('demandOnlineFromFilter')?.value || '',
    onlineTo: document.getElementById('demandOnlineToFilter')?.value || '',
    container: document.getElementById('demandContainerFilter')?.value || ''
  };
  const saved = window.demandSavedQueries || [];
  const deduped = saved.filter((it) => it.name !== name);
  deduped.unshift(snapshot);
  window.demandSavedQueries = deduped.slice(0, 8);
  localStorage.setItem('demand_saved_queries_v1', JSON.stringify(window.demandSavedQueries));
  renderDemandSavedQueries();
  showToast(`已保存条件：${name}`);
}
function renderDemandSavedQueries() {
  const host = document.getElementById('demandSavedQueries');
  if (!host) return;
  const saved = window.demandSavedQueries || [];
  if (!saved.length) {
    host.style.display = 'none';
    host.innerHTML = '';
    return;
  }
  host.style.display = 'flex';
  host.innerHTML = saved.map((it) => `<div class="sv-pill" onclick="applySavedDemandQuery('${escapeJsString(it.name)}')"><i class="fas fa-filter"></i>${escapeHtml(it.name)}</div>`).join('');
}
function applySavedDemandQuery(name) {
  const target = (window.demandSavedQueries || []).find((it) => it.name === name);
  if (!target) return;
  currentDemandPerspective = target.perspective || 'all';
  currentDemandActionFilter = target.actionFilter || 'all';
  const scopeNorm = target.scope === 'subBiz' ? 'biz' : (target.scope || '');
  document.getElementById('demandScopeFilter').value = scopeNorm;
  document.querySelectorAll('.dw-scope-seg .dw-scope-chip').forEach((chip) => {
    chip.classList.toggle('active', (chip.dataset.dwScope || '') === scopeNorm);
  });
  document.getElementById('demandKeywordFilter').value = target.keyword || '';
  // 敏捷小组改为单选标签：多选保存条件取首个，空 = 全部
  const savedAgiles = (target.agiles || []).map(normalizeDemandAgile).filter(Boolean);
  window.demandAgileSelection = savedAgiles.length ? new Set([savedAgiles[0]]) : new Set();
  window.demandTeamSelection = new Set(target.teams || []);
  window.demandSystemSelection = new Set(target.systems || []);
  window.demandPoolSelection = new Set(target.pools || []);
  window.demandZentaoStatusSelection = new Set(target.zentaoStatuses || []);
  setDemandValueStreamStageFilter(target.valueStreamStage || '');
  const fieldMap = {
    demandCategoryFilter: target.category,
    demandImportanceFilter: target.importance,
    demandPriorityFilter: target.priority,
    demandDeptFilter: target.dept,
    demandAssigneeFilter: target.assignee,
    demandCreatedByFilter: target.createdBy,
    demandProposerFilter: target.proposer,
    demandOwnerFilter: target.demandOwner,
    demandTesterFilter: target.tester,
    demandDelayedFilter: target.delayed,
    demandSuspendedFilter: target.suspended,
    demandChangingFilter: target.changing,
    demandApprovalFilter: target.approval,
    demandIterationFilter: target.iteration,
    demandWindowFilter: target.windowType,
    demandCreatedFromFilter: target.createdFrom,
    demandCreatedToFilter: target.createdTo,
    demandDeliveryFromFilter: target.deliveryFrom,
    demandDeliveryToFilter: target.deliveryTo,
    demandOnlineFromFilter: target.onlineFrom,
    demandOnlineToFilter: target.onlineTo,
    demandContainerFilter: target.container
  };
  Object.entries(fieldMap).forEach(([id, value]) => {
    const el = document.getElementById(id);
    if (el) el.value = value || '';
  });
  document.querySelectorAll('.perspective-tab').forEach((tab) => {
    tab.classList.toggle('active', tab.dataset.perspective === currentDemandPerspective);
  });
  syncDemandAgileTabs();
  syncDemandActionChips();
  renderDemandDimensionSummary();
  if (target.view && target.view !== demandCurrentView) switchDemandView(target.view);
  applyDemandFilters();
}
function applyDemandFilters() {
  demandPage = 1;
  renderDemandDimensionSummary();
  renderDemandConditionChips();
  if (typeof triggerDemandWorkspaceRefresh === 'function') {
    triggerDemandWorkspaceRefresh({ page: demandPage, pageSize: demandPageSize });
  } else if (demandCurrentView === 'list') {
    renderDemandListView();
  } else if (demandCurrentView === 'kanban') {
    renderDemandKanbanView();
  } else if (demandCurrentView === 'group') {
    renderDemandGroupView();
  }
  renderDemandAnalyticsPanel();
}

function getDemandNodeMeta(rows) {
  const nodeMap = new Map(rows.map((item) => [item.id, item]));
  const childrenMap = new Map();
  rows.forEach((item) => {
    if (!item.parentId) return;
    if (!childrenMap.has(item.parentId)) childrenMap.set(item.parentId, []);
    childrenMap.get(item.parentId).push(item);
  });
  return { nodeMap, childrenMap };
}

function ensureDefaultDemandExpansion(rows) {
  const { childrenMap } = getDemandNodeMeta(rows);
  const expanded = window.expandedDemandIds || new Set();
  // 仅首次默认展开业务/子业务；之后尊重用户折叠（勿每次 render 强制加回）
  if (!window.__demandExpandInitialized) {
    rows.forEach((row) => {
      if (!childrenMap.has(row.id)) return;
      if (row.nodeType === 'biz' || row.nodeType === 'subBiz') expanded.add(row.id);
    });
    window.__demandExpandInitialized = true;
  }
  window.expandedDemandIds = expanded;
}

function collectNodeWithDescendants(id, childrenMap, collector = []) {
  const children = childrenMap.get(id) || [];
  children.forEach((child) => {
    collector.push(child.id);
    collectNodeWithDescendants(child.id, childrenMap, collector);
  });
  return collector;
}

function switchDemandView(viewType) {
  if (viewType === 'group' || viewType === 'kanban') viewType = 'list';
  demandCurrentView = viewType;
  const dw = document.getElementById('page-demandWorkspace');
  if (dw) {
    dw.querySelectorAll('.view-tab').forEach(tab => tab.classList.remove('active'));
    const next = dw.querySelector(`.view-tab[data-view="${viewType}"]`);
    if (next) next.classList.add('active');
  }
  document.getElementById('demandListView').style.display = viewType === 'list' ? 'block' : 'none';
  document.getElementById('demandKanbanView').style.display = viewType === 'kanban' ? 'block' : 'none';
  document.getElementById('demandGroupView').style.display = viewType === 'group' ? 'block' : 'none';
  if (viewType === 'list') renderDemandListView();
  else if (viewType === 'kanban') renderDemandKanbanView();
  else if (viewType === 'group') renderDemandGroupView();
  renderDemandAnalyticsPanel();
}

function renderDemandHeaderCounts() {
  refreshDemandWorkspaceHeaderCounts();
}

function countDemandBy(rows, getter) {
  const map = new Map();
  rows.forEach((row) => {
    const key = getter(row) || '未归属';
    map.set(key, (map.get(key) || 0) + 1);
  });
  return Array.from(map.entries()).sort((a, b) => b[1] - a[1]).slice(0, 4);
}

function renderDemandHeaderCountsLegacy() {
  const rows = DEMAND_TREE_DATA || [];
  const todayStr = new Date().toISOString().slice(0, 10);
  const isOpen = (row) => !['已关闭', '已取消'].includes(getDemandItemZentaoStatus(row));
  const isRelated = (row) => ['我牵头', '我配合', '我创建', '我关注'].includes(getDemandItemRelation(row));
  const isLead = (row) => getDemandItemRelation(row) === '我牵头';
  const isBlocked = (row) => ['flowBlocked','testBlocked','blocked'].includes(String(row.riskType || '')) || /阻塞/.test(String(row.status || ''));
  const isOverdue = (row) => {
    const d = String(getDemandItemEstimateLaunch(row) || getDemandItemDeadline(row) || '').slice(0, 10);
    return !!d && d !== '-' && d < todayStr;
  };
  const isFocus = (row) => row.followed === true;
  const setCount = (id, count) => {
    const el = document.getElementById(id);
    if (el) el.textContent = String(count);
  };
  // BUG #8 (2026-08-20): 「全部」chip 口径对齐首页 9 阶段卡片 (价值流 API 真源)。
  // 旧实现用 DEMAND_TREE_DATA.rows.length (本页面分页/过滤后行数), 跟首页
  // poValueStreamCounts.all (value-stream API 全生命周期计数) 两个口径, 导致
  // admin 视角首页 762 vs 需求查询 500 互相打架。改用 poValueStreamCounts.all,
  // 与首页紧凑视图 / 详细视图 9 阶段卡片同源同字段; poValueStreamCounts 加载前
  // fallback 到 rows.length (兼容冷启动一帧)。
  const vsAll = (typeof poValueStreamCounts === 'object' && poValueStreamCounts && typeof poValueStreamCounts.all === 'number') ? poValueStreamCounts.all : null;
  // r2：快捷查询数字跟随对象范围；与原 all 统计不同来源，避免四个 chip 永远同值
  const scopeRaw = document.getElementById('demandScopeFilter')?.value || '';
  const scope = scopeRaw === 'subBiz' ? 'biz' : scopeRaw;
  const filterByScope = (row) => !scope
    || (scope === 'biz' && (row.nodeType === 'biz' || row.nodeType === 'subBiz'))
    || (scope === 'rd' && row.nodeType === 'rd');
  const scoped = rows.filter(filterByScope);
  setCount('dwCountMyRelated', scoped.filter(isRelated).length);
  setCount('dwCountMyLead', scoped.filter(isLead).length);
  setCount('dwCountBlocked', scoped.filter(isBlocked).length);
  setCount('dwCountOverdue', scoped.filter(isOverdue).length);
  setCount('dwCountNeedFocus', scoped.filter(isFocus).length);
  setCount('dwCountScopeAll', vsAll !== null ? vsAll : rows.length);
  setCount('dwCountScopeBiz', rows.filter((row) => row.nodeType === 'biz' || row.nodeType === 'subBiz').length);
  setCount('dwCountScopeRd', rows.filter((row) => row.nodeType === 'rd').length);
}

function renderDemandAnalyticsPanel() {
  const host = document.getElementById('demandAnalyticsPanel');
  if (!host) return;
  if (!demandAnalyticsVisible) {
    host.style.display = 'none';
    host.innerHTML = '';
    return;
  }
  const rows = getDemandFilteredFlatRows();
  const rootRows = rows.filter((row) => row.nodeType === 'biz');
  const rdRows = rows.filter((row) => row.nodeType === 'rd');
  const blockerRows = rows.filter((row) => !!getDemandItemBlockerInfo(row));
  const overdueRows = rows.filter((row) => String(row.deadline || '').slice(0, 10) && String(row.deadline || '').slice(0, 10) < new Date().toISOString().slice(0, 10));
  const currentWindowName = (typeof vfPrimaryVersions !== 'undefined' && vfPrimaryVersions[0] && vfPrimaryVersions[0].name)
    ? vfPrimaryVersions[0].name
    : '';
  const currentWindowRows = currentWindowName
    ? rows.filter((row) => getDemandItemIteration(row) === currentWindowName || getDemandItemIteration(row) === (typeof getVersionWindowShortName === 'function' ? getVersionWindowShortName(currentWindowName) : currentWindowName))
    : [];
  const topTeams = countDemandBy(rows, (row) => row.team);
  const topStages = countDemandBy(rows, (row) => getDemandItemValueStage(row));
  const topOwners = countDemandBy(rows, (row) => getDemandItemOwner(row));
  const renderMiniBars = (items) => items.map(([name, count]) => `<span class="dw-analysis-chip"><span>${name}</span><strong>${count}</strong></span>`).join('');
  host.style.display = 'grid';
  host.innerHTML = `
    <div class="dw-analysis-card">
      <div class="dw-analysis-k">查询结果</div>
      <div class="dw-analysis-v">${rows.length}</div>
      <div class="dw-analysis-sub">BR ${rootRows.length} · RD ${rdRows.length}</div>
    </div>
    <div class="dw-analysis-card warn">
      <div class="dw-analysis-k">风险与超期</div>
      <div class="dw-analysis-v">${blockerRows.length}</div>
      <div class="dw-analysis-sub">超期 ${overdueRows.length}</div>
    </div>
    <div class="dw-analysis-card">
      <div class="dw-analysis-k">当前窗口</div>
      <div class="dw-analysis-v">${currentWindowRows.length}</div>
      <div class="dw-analysis-sub">${currentWindowName ? vfH(currentWindowName) : '暂无窗口'}</div>
    </div>
    <div class="dw-analysis-card wide">
      <div class="dw-analysis-k">团队分布</div>
      <div class="dw-analysis-list">${renderMiniBars(topTeams)}</div>
    </div>
    <div class="dw-analysis-card wide">
      <div class="dw-analysis-k">价值流阶段</div>
      <div class="dw-analysis-list">${renderMiniBars(topStages)}</div>
    </div>
    <div class="dw-analysis-card wide">
      <div class="dw-analysis-k">负责人负载</div>
      <div class="dw-analysis-list">${renderMiniBars(topOwners)}</div>
    </div>
  `;
}

function exportDemandWorkspace() {
  const rows = getDemandFilteredFlatRows();
  const headers = ['编号','所属需求池','所属模块','优先级','重要程度','需求类别','需求来源','需求主题','指派给','测试负责人','验收负责人','业务需求负责人','业务评审人','状态','是否变更中','是否系统主管审批','期望完成时间','创建时间','由谁创建','澄清日期','预计上线时间','排定计划日期','开发完成时间','测试完成时间','交付时间','是否挂起','是否需要重点关注','卡点原因'];
  const csvRows = rows.map((row) => [
    row.reqCode || row.id,
    row.pool || '',
    row.module || '',
    row.priority || '',
    getDemandItemSeverityLabel(row),
    getDemandItemCategoryLabel(row),
    getDemandItemSourceLabel(row),
    row.title || '',
    getDemandItemAssignee(row),
    getDemandItemTester(row),
    getDemandItemAcceptOwner(row),
    getDemandItemOwner(row),
    getDemandItemReviewer(row),
    getDemandItemZentaoStatus(row),
    getDemandItemChangeLabel(row),
    getDemandItemManagerReviewLabel(row),
    getDemandItemDeadline(row),
    getDemandItemCreatedDate(row),
    row.createdBy || '',
    row.clarifyDate || '',
    getDemandItemEstimateLaunch(row),
    getDemandItemSchedulePlanDate(row),
    row.developFinish || '',
    row.testFinish || '',
    getDemandItemDeliveryDate(row),
    getDemandItemHangLabel(row),
    getDemandItemFocusLabel(row),
    getDemandItemBlockerInfo(row)?.reason || bottleneckReasonFromItem(row)
  ]);
  const escape = (value) => `"${String(value ?? '').replace(/"/g, '""')}"`;
  const csv = [headers, ...csvRows].map((line) => line.map(escape).join(',')).join('\n');
  try {
    const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `业务需求综合查询_${new Date().toISOString().slice(0,10)}.csv`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    showToast(`已导出 ${rows.length} 条需求`);
  } catch (e) {
    showToast(`已导出当前查询 ${rows.length} 条`);
  }
}

/* 2026-08-31: 需求查询「操作」列关注 / 取消关注 toggle。
   真源 = /workbench/api/watches（AddWatch / RemoveWatch），跟详情面板 bdToggleWatch 一致。
   乐观本地更新 item.followed，再整体 rerender 列表，让星标空/实心立即翻转。 */
async function toggleDemandWatch(rawId, currentFollowed) {
  const oid = (typeof extractDemandId === 'function') ? extractDemandId(rawId)
    : (rawId == null ? null : Number(String(rawId).replace(/[^0-9]/g, '')));
  if (!oid) { (window.showToast || function(){})('无法识别需求编号'); return; }
  const wasWatched = !!currentFollowed;
  const url = wasWatched
    ? '/workbench/api/watches/' + oid
    : '/workbench/api/watches';
  const opts = wasWatched
    ? { method: 'DELETE' }
    : { method: 'POST', body: { objectType: 'demand', objectId: oid } };
  try {
    const res = await (window.apiFetch || apiFetch)(url, opts);
    if (!res || !res.success) {
      (window.showToast || function(){})((res && res.message) || (wasWatched ? '取消关注失败' : '关注失败'));
      return;
    }
    const rows = (typeof DEMAND_TREE_DATA !== 'undefined' && Array.isArray(DEMAND_TREE_DATA)) ? DEMAND_TREE_DATA
      : (typeof demandWorkspaceRows !== 'undefined' && Array.isArray(demandWorkspaceRows) ? demandWorkspaceRows : []);
    const target = rows.find((r) => String(r && r.id) === String(rawId) || (typeof extractDemandId === 'function' && r && extractDemandId(r.id) === oid));
    if (target) target.followed = !wasWatched;
    (window.showToast || function(){})((res && res.message) || (wasWatched ? '已取消关注' : '已关注'));
    if (typeof renderDemandListView === 'function') renderDemandListView();
    if (typeof refreshDemandWorkspaceHeaderCounts === 'function') refreshDemandWorkspaceHeaderCounts();
  } catch (e) {
    (window.showToast || function(){})(wasWatched ? '取消关注失败' : '关注失败');
  }
}
window.toggleDemandWatch = toggleDemandWatch;
window.applyDemandRolePreset = applyDemandRolePreset;
window.toggleDemandAnalyticsPanel = toggleDemandAnalyticsPanel;
window.exportDemandWorkspace = exportDemandWorkspace;
window.openDemandPoolDropdown = openDemandPoolDropdown;
window.closeDemandPoolDropdown = closeDemandPoolDropdown;
window.renderDemandPoolDropdownOptions = renderDemandPoolDropdownOptions;
window.toggleDemandPoolDropdownOption = toggleDemandPoolDropdownOption;
window.clearDemandPoolDropdownSelection = clearDemandPoolDropdownSelection;
