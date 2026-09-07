// ═══ 版本跟进重构 JavaScript ═══

// 当前选中的版本和视角
var currentVfIteration = '';
var currentVfStage = '';
var currentVfTab = 'overview';
var vfHistoryMenuOpen = false;
var vfFilterPanelOpen = false;
var vfHistorySearchQuery = '';
var vfListPaginationState = { page: 1, pageSize: 15, total: 0 };
window.currentVfExceptionFilter = window.currentVfExceptionFilter || 'all';
window.currentVfFilters = window.currentVfFilters || {system:'',agile:'',relation:'',keyword:'',status:'',priority:'',owner:'',releaseWindow:'',dateFrom:'',dateTo:'',myScope:'',project:'',execution:''};

function mergeVfFilters() {
  const d = { system:'', agile:'', relation:'', keyword:'', status:'', priority:'', owner:'', releaseWindow:'', dateFrom:'', dateTo:'', myScope:'' };
  window.currentVfFilters = Object.assign(d, window.currentVfFilters || {});
}

function getVfMemberDisplayName() {
  if (!currentUser) return '';
  return String(currentUser.realname || currentUser.account || currentUser.name || currentUser.username || '').trim();
}

function vfNameRoughMatch(cell, me) {
  if (!me || cell == null || String(cell).trim() === '') return false;
  const c = String(cell).trim();
  const m = String(me).trim();
  if (c === m) return true;
  const c0 = c.split(/[（(]/)[0].trim();
  const m0 = m.split(/[（(]/)[0].trim();
  if (c0 && m0 && c0 === m0) return true;
  return c.includes(m0) || m.includes(c0);
}

function rowMatchesVfMyWork(row, me) {
  if (vfNameRoughMatch(row.dev, me) || vfNameRoughMatch(row.test, me) || vfNameRoughMatch(row.accept, me) || vfNameRoughMatch(row.acceptOwner, me)) return true;
  if (row.relation === '我牵头' && vfNameRoughMatch(row.acceptOwner, me)) return true;
  return false;
}

function setVfMyScope(scope) {
  if (scope === 'mine' && !getVfMemberDisplayName()) {
    showToast('请先登录后再使用「跟我的」');
    return;
  }
  mergeVfFilters();
  window.currentVfFilters.myScope = scope === 'mine' ? 'mine' : '';
  resetVfListPage();
  renderVersionFollow();
}

function setVfRelationChip(value) {
  mergeVfFilters();
  window.currentVfFilters.relation = window.currentVfFilters.relation === value ? '' : (value || '');
  resetVfListPage();
  renderVersionFollow();
}

function toggleVfLayer(layer) {
  if (typeof window.vfShowRdLayer !== 'boolean') window.vfShowRdLayer = false;
  if (typeof window.vfShowTaskLayer !== 'boolean') window.vfShowTaskLayer = false;
  if (layer === 'rd') {
    window.vfShowRdLayer = !window.vfShowRdLayer;
    /* 收起研需时同步收起任务（PRD / v10.2 原型） */
    if (!window.vfShowRdLayer) window.vfShowTaskLayer = false;
  }
  if (layer === 'task') {
    window.vfShowTaskLayer = !window.vfShowTaskLayer;
    /* 展开任务必须先展开研需 */
    if (window.vfShowTaskLayer) window.vfShowRdLayer = true;
  }
  /* 全部展开/收起后，行级展开状态失效，避免开关与箭头互相打架 */
  window.vfExpandedRows = {};
  renderVersionFollow();
}

/** 版本跟进子层真源：优先 DTO 上的 item.devRequirements，其次才回落 scheduleData。 */
function resolveVfDemandTree(item) {
  const fromItem = Array.isArray(item && item.devRequirements) ? item.devRequirements : [];
  if (fromItem.length) {
    return { rds: fromItem, directTasks: [] };
  }
  const biz = (typeof scheduleData !== 'undefined' ? scheduleData : []).find((s) => s && item && s.id === item.id);
  if (biz && typeof ensureIntegratedRdDemoTasks === 'function') ensureIntegratedRdDemoTasks(biz);
  const rds = (biz && Array.isArray(biz.devRequirements)) ? biz.devRequirements : [];
  const directTasks = (!rds.length && biz && Array.isArray(biz.createdTasks)) ? biz.createdTasks : [];
  return { rds: rds, directTasks: directTasks };
}

function toggleVfRow(key) {
  window.vfExpandedRows = window.vfExpandedRows || {};
  window.vfExpandedRows[key] = !window.vfExpandedRows[key];
  renderVfTabBody();
}

function resetVfListPage() {
  vfListPaginationState.page = 1;
}

function vfHistoryQuickSearchInput(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  vfHistorySearchQuery = (ev.target && ev.target.value) ? ev.target.value : '';
  vfHistoryMenuOpen = true;
  renderVfVersionSwitch();
}

function vfCapacityMetaKey() {
  return String(currentVfIteration || (vfPrimaryVersions[0] && vfPrimaryVersions[0].id) || '');
}

function collectVfOwnerOptions() {
  const set = new Set();
  getVfBaseRows().forEach((r) => {
    [r.dev, r.test, r.accept, r.acceptOwner].forEach((x) => {
      if (x && String(x) !== '待分配') set.add(String(x));
    });
  });
  return Array.from(set).sort((a, b) => String(a).localeCompare(String(b), 'zh-CN'));
}

let vfPrimaryVersions = [];
let vfHistoryVersions = [];
let vfVersions = [];

function getCurrentVfVersion() {
  const empty = { id: '', name: '暂无版本窗口', cardName: '-', range: '-', status: '-', eta: '' };
  return vfVersions.find(v => v.id === currentVfIteration) || vfPrimaryVersions[0] || vfVersions[0] || empty;
}

// V1.2.1：历史窗口内所有写操作必须隐藏/禁用（已发布窗只读真源，源自 isVersionWindowHistory）。
function isReadOnlyVersionFollow() {
  const v = getCurrentVfVersion();
  if (!v) return false;
  if (typeof v.isHistory === 'boolean') return v.isHistory;
  const status = String(v.status || '').toLowerCase();
  return status === 'released' || status === 'closed';
}

// V1.2.1：版本跟进 → 验收交付页深链；demandId/windowId/intent 写入 AD state 并切页。
function navigateToAcceptanceDelivery(demandId, windowId, intent) {
  const dId = Number(demandId) || 0;
  const wId = Number(windowId) || 0;
  if (!dId) {
    showToast('需求 ID 无效，无法跳转验收交付');
    return;
  }
  const intentStage = {
    accept: 'accept',
    deliver: 'deliver',
    release: 'release',
    submit: 'submit',
    verify: 'verify'
  }[String(intent || '').toLowerCase()] || '';
  const ctx = { demandId: dId, windowId: wId, intent: intentStage, ts: Date.now() };
  try { window.__adDeepLinkContext = ctx; } catch (e) {}
  if (typeof window.navTo === 'function') {
    try { window.navTo('acceptanceDelivery', { demandId: dId, windowId: wId, intent: intentStage }); return; }
    catch (e) { /* 退回 location */ }
  }
  // 兜底：直接拼 location.hash
  try {
    const u = new URL(window.location.href);
    const rawHash = u.hash && u.hash.indexOf('#/acceptanceDelivery') === 0 ? u.hash : '#/acceptanceDelivery';
    const parts = rawHash.split('?');
    const params = new URLSearchParams(parts[1] || '');
    params.set('demandId', String(dId));
    if (wId) params.set('windowId', String(wId)); else params.delete('windowId');
    if (intentStage) params.set('intent', intentStage); else params.delete('intent');
    u.hash = parts[0] + '?' + params.toString();
    window.location.href = u.toString();
  } catch (e) {
    showToast('已记录深链上下文，请切换到验收交付页');
  }
}

function vfRowMatchesIteration(row, iterId) {
  const want = String(iterId == null ? '' : iterId).trim();
  if (!want) return true;
  return String(row && row.iterationId || '') === want;
}

function isVfBlocked(row) {
  return String(row.reason || '').includes('阻塞') ||
    String(row.currentBlock || '').includes('阻塞') ||
    String(row.bugStatus || '').includes('严重') ||
    (Number(row.severe) || 0) > 0;
}

function isVfOverdue(row) {
  return String(row.reason || '').includes('超期') ||
    String(row.devStatus || '').includes('超期') ||
    String(row.testStatus || '').includes('超期');
}

/* ═══ 版本跟进：日期口径（统一使用真实运行日期）═══ */
function vfToday() {
  return new Date().toISOString().slice(0, 10);
}
function vfNormDate(v) {
  const m = String(v == null ? '' : v).match(/\d{4}-\d{2}-\d{2}/);
  return m ? m[0] : '';
}
function vfShortDate(v) {
  const d = vfNormDate(v);
  return d ? d.slice(5) : '';
}
function vfShiftDate(v, delta) {
  const d = vfNormDate(v);
  if (!d) return '';
  const t = new Date(d + 'T00:00:00');
  t.setDate(t.getDate() + Number(delta || 0));
  return t.toISOString().slice(0, 10);
}
function vfDaysFromToday(v) {
  const d = vfNormDate(v);
  if (!d) return null;
  return Math.round((new Date(d + 'T00:00:00') - new Date(vfToday() + 'T00:00:00')) / 86400000);
}
function vfIsPast(v) {
  const d = vfNormDate(v);
  return !!d && d < vfToday();
}

/* 四个关键时间：数据来自业务需求的开发完成 / 测试完成 / 验收完成 / 预计上线 */
/* 注意：状态值里「未验收 / 未交付」都含「验收 / 交付」，判定必须用带前缀的整词，不能用 includes('验收') */
const vfMilestoneDefs = [
  { key: 'dev', label: '开发完成', plan: (r) => r.devPlan, done: (r) => /开发完成/.test(r.devStatus || '') || /测试中|测试完成/.test(r.testStatus || '') || /待验收|已验收/.test(r.acceptanceStatus || '') || /待交付|已交付/.test(r.deliveryStatus || '') },
  { key: 'test', label: '测试完成', plan: (r) => r.testPlan, done: (r) => /测试完成/.test(r.testStatus || '') || /已验收/.test(r.acceptanceStatus || '') || /已交付/.test(r.deliveryStatus || '') },
  { key: 'accept', label: '验收完成', plan: (r) => r.acceptPlan, done: (r) => /已验收/.test(r.acceptanceStatus || '') || /已交付/.test(r.deliveryStatus || '') },
  { key: 'online', label: '上线', plan: (r) => r.targetDate, done: (r) => /已交付/.test(r.deliveryStatus || '') || /验证通过/.test(r.prodVerify || '') }
];
/* 关键时间已过但该环节没完成 = 里程碑滑期 */
function vfRowSlip(row) {
  for (let i = 0; i < vfMilestoneDefs.length; i++) {
    const d = vfMilestoneDefs[i];
    const plan = vfNormDate(d.plan(row));
    if (plan && vfIsPast(plan) && !d.done(row)) {
      return { label: d.label, plan: plan, days: Math.abs(vfDaysFromToday(plan)) };
    }
  }
  return null;
}
/* 就绪度：阻塞 > 有风险 > 可按期；已交付/已验证一律算完成 */
function getVfReadiness(row) {
  if (String(row.deliveryStatus || '').includes('已交付') || String(row.prodVerify || '').includes('验证通过')) return 'ready';
  if (isVfBlocked(row)) return 'blocked';
  if (isVfOverdue(row) || vfRowSlip(row)) return 'risk';
  return 'ready';
}
function getVfReadinessCounts(rows) {
  const c = { total: rows.length, ready: 0, risk: 0, blocked: 0 };
  rows.forEach((r) => { c[getVfReadiness(r)] += 1; });
  return c;
}
/* 窗口级关键时间对照：计划日取窗口内最迟计划，完成/超期按行统计 */
function getVfWindowMilestones(rows) {
  return vfMilestoneDefs.map((d) => {
    /* 分母取窗口内全部需求：只算「填了关键时间」的行会让未排期需求凭空消失 */
    const withPlan = rows.filter((r) => vfNormDate(d.plan(r)));
    return {
      label: d.label,
      planDate: withPlan.map((r) => vfNormDate(d.plan(r))).sort().pop() || '',
      total: rows.length,
      done: rows.filter((r) => d.done(r)).length,
      late: withPlan.filter((r) => !d.done(r) && vfIsPast(d.plan(r))).length
    };
  }).filter((m) => m.total > 0);
}
function vfTitleOf(id) {
  const s = (typeof scheduleData !== 'undefined' ? scheduleData : []).find((x) => x.id === id);
  return s ? s.title : String(id || '');
}
async function vfUrge(id, owner) {
  // 兼容旧调用：无 urgeType 时默认按开发催办打开弹窗。
  return openRemind('dev', id, { owner: owner || '', reason: '请尽快处理当前责任事项' });
}

const VF_URGE_LABELS = {
  'dev': '催开发',
  'submit': '催提测',
  'test': '催测试',
  'acceptance': '催验收',
  'deliver': '催发起交付',
  'release': '催发布',
  'initial-review': '催需求审批',
  'change-review': '催变更审批',
  'manager-review': '催主管审批'
};

function vfUrgeTypeFromStages(stages) {
  const list = Array.isArray(stages) ? stages : [];
  if (list.indexOf('change-review') >= 0) return 'change-review';
  if (list.indexOf('manager-review') >= 0) return 'manager-review';
  if (list.indexOf('initial-review') >= 0) return 'initial-review';
  return '';
}

// V1.2.1：审批催办旁「查看审批」只读入口。基于已聚合的 pendingApprovalStages/Reviewers 渲染轻量 Drawer。
// 不调用任何 urge / write 接口；不做审批写操作；不重做 v10.2 大 UI。
function openApprovalView(demandId, stage) {
  var dId = Number(demandId) || 0;
  if (!dId) return;
  var stg = String(stage || '');
  var item = (window.vfLastItems || []).find(function (it) { return Number(it && it.demandId) === dId; }) ||
             (window.vfLastItems || []).find(function (it) { return String(it && it.id) === String(demandId); });
  var reviewers = (item && item.pendingApprovalReviewers && item.pendingApprovalReviewers[stg]) || [];
  var stageLabel = ({
    'initial-review': '需求评审',
    'change-review': '需求变更审批',
    'manager-review': '主管部门审批'
  })[stg] || stg || '审批';
  ensureVfApprovalModal();
  var body = document.getElementById('vfApprovalBody');
  var title = document.getElementById('vfApprovalTitle');
  if (title) title.textContent = stageLabel + ' · ' + vfH(item && (item.id || item.code) || demandId);
  /* V1.2.4：按 v10.2 原型 — 三种审批类型分别生成 section + 评审人列表 + 原审批对象按钮 */
  var approvalListHtml = (Array.isArray(reviewers) && reviewers.length)
    ? reviewers.map(function (acc) { return '<div class="vf-approval-person"><div><b>' + vfH(acc) + '</b><br><span>审批人</span></div><div class="wait">待审批</div></div>'; }).join('')
    : '<div class="vf-approval-person"><div><b>待审批人员</b><br><span>由后端按场景聚合</span></div><div>—</div></div>';
  var sections = '';
  if (stg === 'initial-review') {
    sections =
      '<div class="vf-section"><div class="vf-section-title">需求评审进度</div><div class="vf-kv"><div>提交时间</div><div>08-24 10:20</div><div>当前状态</div><div>评审中</div><div>计划完成</div><div>08-25</div></div></div>' +
      '<div class="vf-section"><div class="vf-section-title">评审人</div><div class="vf-approval-list">' + approvalListHtml + '</div></div>';
  } else if (stg === 'manager-review') {
    sections =
      '<div class="vf-section"><div class="vf-section-title">主管部门审批 · 按产品</div><div class="vf-approval-list">' + approvalListHtml + '</div></div>' +
      '<div class="vf-section"><div class="vf-section-title">流程约束</div><div class="vf-kv"><div>控制转研发</div><div>是</div><div>影响</div><div>手机银行审批未通过前，不允许继续转研发</div><div>催办对象</div><div>仅未通过审批人</div></div></div>';
  } else if (stg === 'change-review') {
    sections =
      '<div class="vf-section"><div class="vf-section-title">需求变更审批</div><div class="vf-kv"><div>变更编号</div><div>CHG-' + vfH(new Date().toISOString().slice(2,10).replace(/-/g, '')) + '-01</div><div>变更申请人</div><div>—</div><div>变更原因</div><div>业务规则调整</div><div>提交时间</div><div>—</div><div>当前状态</div><div>变更评审中</div><div>版本影响</div><div style="color:var(--red,#b91c1c)">当前开发暂停，等待变更评审结论</div></div></div>' +
      '<div class="vf-section"><div class="vf-section-title">变更评审人</div><div class="vf-approval-list">' + approvalListHtml + '</div></div>';
  } else {
    sections = '<div class="vf-section"><div class="vf-section-title">审批详情</div><div class="vf-approval-list">' + approvalListHtml + '</div></div>';
  }
  if (body) body.innerHTML =
    '<div class="vf-section"><div class="vf-section-title">当前事项</div><div class="vf-kv"><div>需求</div><div>' + vfH(item && (item.id || item.code) || demandId) + '</div><div>审批类型</div><div>' + vfH(stageLabel) + '</div></div></div>' +
    sections +
    '<button type="button" class="vf-section-btn" onclick="toast(\'原型：查看原审批对象 ' + vfH(item && (item.id || item.code) || demandId) + '\')">查看原审批对象</button>' +
    '<div class="hint" style="margin-top:8px">本抽屉只读，不触发催办与审批操作。审批写操作在禅道完成。</div>';
  var overlay = document.getElementById('vfApprovalModal');
  if (overlay) overlay.classList.add('show');
}

function ensureVfApprovalModal() {
  if (document.getElementById('vfApprovalModal')) return;
  var wrap = document.createElement('div');
  wrap.id = 'vfApprovalModal';
  wrap.className = 'vf-modal-overlay';
  wrap.innerHTML =
    '<div class="vf-modal-mask" onclick="closeVfApprovalView()"></div>' +
    '<div class="vf-modal" role="dialog" aria-modal="true">' +
      '<div class="vf-modal-hdr"><h3 id="vfApprovalTitle">查看审批</h3><button type="button" class="vf-modal-close" onclick="closeVfApprovalView()">×</button></div>' +
      '<div class="vf-modal-body" id="vfApprovalBody"></div>' +
      '<div class="vf-modal-ftr">' +
        '<button type="button" class="action-btn" onclick="closeVfApprovalView()">关闭</button>' +
      '</div>' +
    '</div>';
  document.body.appendChild(wrap);
}

function closeVfApprovalView() {
  var m = document.getElementById('vfApprovalModal');
  if (m) m.classList.remove('show');
}

function vfApprovalBadgeHtml(stages) {
  const list = Array.isArray(stages) ? stages : [];
  if (!list.length) return '';
  const labelMap = {
    'initial-review': '需求审批',
    'manager-review': '主管审批',
    'change-review': '变更审批'
  };
  return list.map((s) => `<span class="status-tag st-warning vf-col-pendingApproval">${vfH(labelMap[s] || s)}</span>`).join(' ');
}

function vfEnsureRemindModal() {
  if (document.getElementById('vfRemindModal')) return;
  const overlay = document.createElement('div');
  overlay.id = 'vfRemindModalOverlay';
  overlay.className = 'batch-modal-overlay';
  overlay.onclick = function () { closeRemindModal(); };
  const modal = document.createElement('div');
  modal.id = 'vfRemindModal';
  modal.className = 'batch-modal';
  modal.setAttribute('role', 'dialog');
  modal.setAttribute('aria-modal', 'true');
  modal.innerHTML =
    '<div class="batch-modal-hdr"><h3 id="vfRemindTitle">催办</h3>' +
    '<button type="button" class="modal-close" aria-label="关闭" onclick="closeRemindModal()"><i class="fas fa-xmark"></i></button></div>' +
    '<div class="batch-modal-body" id="vfRemindBody"></div>' +
    '<div class="batch-modal-footer" id="vfRemindFooter"></div>';
  document.body.appendChild(overlay);
  document.body.appendChild(modal);
}

function closeRemindModal() {
  const overlay = document.getElementById('vfRemindModalOverlay');
  const modal = document.getElementById('vfRemindModal');
  if (overlay) overlay.classList.remove('show');
  if (modal) modal.classList.remove('show');
  if (window._vfRemindPicker && typeof window._vfRemindPicker.destroy === 'function') {
    try { window._vfRemindPicker.destroy(); } catch (e) { /* ignore */ }
  }
  window._vfRemindPicker = null;
  window._vfRemindCtx = null;
}

function vfRemindPickedAccounts() {
  const picker = window._vfRemindPicker;
  if (!picker || typeof picker.getValue !== 'function') return [];
  const v = picker.getValue();
  if (Array.isArray(v)) return v.map(function (x) { return String(x || '').trim(); }).filter(Boolean);
  const one = String(v || '').trim();
  return one ? [one] : [];
}

function validateRemind() {
  const ctx = window._vfRemindCtx || {};
  const inapp = document.getElementById('chInapp');
  const err = document.getElementById('channelError');
  const recipErr = document.getElementById('recipientError');
  const btn = document.getElementById('confirmRemind');
  const channelOk = !!(inapp && inapp.checked);
  if (err) err.style.display = channelOk ? 'none' : 'block';
  const picked = vfRemindPickedAccounts();
  const needsManual = ctx.urgeType === 'deliver' || ctx.urgeType === 'release' || ctx.requireRecipient === true;
  const recipOk = !needsManual || picked.length > 0;
  if (recipErr) recipErr.style.display = recipOk ? 'none' : 'block';
  const ok = channelOk && recipOk;
  if (btn) btn.disabled = !ok;
  return ok;
}

function vfMountRemindRecipientPicker(preferredAccount, preferredLabel) {
  const host = document.getElementById('vfRemindRecipientHost');
  if (!host) return;
  if (window._vfRemindPicker && typeof window._vfRemindPicker.destroy === 'function') {
    try { window._vfRemindPicker.destroy(); } catch (e) { /* ignore */ }
  }
  window._vfRemindPicker = null;
  const initial = String(preferredAccount || '').trim();
  const preferred = initial ? [initial] : [];
  if (typeof window.WbPersonPicker === 'undefined' || typeof window.WbPersonPicker.mount !== 'function') {
    host.innerHTML = '<select id="vfRemindRecipientSelect" class="form-select" style="width:100%">' +
      (initial ? ('<option value="' + vfH(initial) + '" selected>' + vfH(preferredLabel || initial) + '</option>') : '<option value="">请选择催办对象</option>') +
      '</select>';
    const sel = document.getElementById('vfRemindRecipientSelect');
    window._vfRemindPicker = {
      getValue: function () { return sel ? String(sel.value || '').trim() : ''; },
      destroy: function () { }
    };
    if (sel) sel.addEventListener('change', validateRemind);
    validateRemind();
    return;
  }
  host.innerHTML = '';
  // 空态文案固定「请选择…」；勿把「待确认」等系统占位当 emptyLabel，避免误导。
  var emptyLabel = '请选择催办对象';
  if (initial && preferredLabel && preferredLabel !== '待确认' && preferredLabel !== '待分配') {
    emptyLabel = preferredLabel;
  }
  window._vfRemindPicker = window.WbPersonPicker.mount({
    el: host,
    mode: 'single',
    value: initial,
    preferred: preferred,
    placeholder: '搜索姓名 / 拼音 / 工号',
    emptyLabel: emptyLabel,
    onChange: function () { validateRemind(); }
  });
  validateRemind();
}

function openRemind(urgeType, id, opts) {
  opts = opts || {};
  const cur = (typeof getCurrentVfVersion === 'function') ? getCurrentVfVersion() : null;
  // 后端 UrgeDemandReq.WindowID 是 uint；空串/字符串 id 会导致 ShouldBindJSON「参数解析失败」。
  const windowId = Number(opts.windowId || (cur && cur.id) || 0) || 0;
  if (cur && cur.isHistory === true) {
    showToast('历史窗口为只读，不能催办');
    return;
  }
  if (!windowId) {
    showToast('缺少版本窗口，无法催办');
    return;
  }
  const s = String(id || '').trim();
  if (!/^(US|REQ|DEMAND)[-#]?\d+$/i.test(s) && !/^\d+$/.test(s)) {
    showToast(id + ' 催办：当前仅支持业务需求粒度，请到禅道原页面操作');
    return;
  }
  const demandId = /^\d+$/.test(s) ? s : extractDemandId(s);
  if (!demandId) {
    showToast('无法识别需求编号：' + id);
    return;
  }
  const type = String(urgeType || 'dev');
  const label = VF_URGE_LABELS[type] || '催办';
  const ownerAccount = String(opts.ownerAccount || opts.recommendedAccount || '').trim();
  const ownerLabel = String(opts.owner || opts.recommendedLabel || ownerAccount || '').trim();
  const reason = opts.reason || '请尽快处理';
  const reviewerCount = Number(opts.reviewerCount || 0);
  const isApproval = type === 'initial-review' || type === 'manager-review' || type === 'change-review';
  const needsManual = type === 'deliver' || type === 'release';
  vfEnsureRemindModal();
  window._vfRemindCtx = {
    urgeType: type,
    demandId: demandId,
    windowId: windowId,
    displayId: s,
    label: label,
    requireRecipient: needsManual || (!ownerAccount && !isApproval && opts.requireRecipient === true)
  };
  document.getElementById('vfRemindTitle').textContent = label;
  let ownerHint = '默认可由系统按场景推荐；也可搜索修改催办对象';
  if (type === 'acceptance') {
    ownerHint = '默认推荐验收人，无验收人则推荐需求提出人；可搜索修改';
  } else if (isApproval) {
    ownerHint = reviewerCount > 0
      ? ('未选手动对象时，将向系统计算的 ' + reviewerCount + ' 名审批人发送提醒')
      : '未选手动对象时，由服务端按未完成审批节点解析收件人；也可搜索指定';
  } else if (needsManual) {
    ownerHint = '该场景无系统默认责任人，请搜索选择催办对象';
  } else if (ownerAccount) {
    ownerHint = '已预填系统推荐责任人，可搜索修改';
  }
  // V1.2.4：按 v10.2 原型分组：当前事项 / 当前审批状态（仅审批催办）/ 催办对象 / 催办原因 / 通知渠道 / 消息预览
  const ownerState = isApproval
    ? '<div class="vf-section"><div class="vf-section-title">当前审批状态</div><div class="form-control readonly">审批中 / 存在待处理节点</div></div>'
    : '';
  const messagePreview = '【' + label + '】' + s + '：' + reason + '。' +
    (isApproval ? '请及时处理当前审批节点，以免影响后续版本推进。' : '请确认当前进展及是否存在影响版本交付的风险。');
  document.getElementById('vfRemindBody').innerHTML =
    '<div class="vf-section"><div class="vf-section-title">当前事项</div><div class="form-control readonly">' + vfH(s) + ' · ' + vfH(label) + '</div></div>' +
    ownerState +
    '<div class="vf-section"><div class="vf-section-title">催办对象</div>' +
    '<div id="vfRemindRecipientHost" class="vf-remind-recipient"></div>' +
    '<div class="hint">' + vfH(ownerHint) + '</div>' +
    '<div class="error" id="recipientError" style="display:none">请搜索并选择催办对象</div></div>' +
    '<div class="vf-section"><div class="vf-section-title">催办原因</div><div class="form-control readonly">' + vfH(reason) + '</div></div>' +
    '<div class="vf-section"><div class="vf-section-title">通知渠道</div><div class="checks">' +
    '<label><input type="checkbox" id="chYanxun" disabled> 燕讯（暂未接入）</label>' +
    '<label><input type="checkbox" id="chInapp" checked onchange="validateRemind()"> 站内通知</label>' +
    '</div><div class="error" id="channelError" style="display:none">至少选择站内通知</div></div>' +
    '<div class="vf-section"><div class="vf-section-title">消息预览</div><div class="form-control readonly">' + vfH(messagePreview) + '</div></div>' +
    '<div id="vfRemindChannelErrors" class="error" style="display:none"></div>';
  document.getElementById('vfRemindFooter').innerHTML =
    '<button type="button" class="btn" onclick="closeRemindModal()">取消</button>' +
    '<button type="button" class="btn primary write-action" id="confirmRemind" onclick="submitRemind()">确认催办</button>';
  document.getElementById('vfRemindModalOverlay').classList.add('show');
  document.getElementById('vfRemindModal').classList.add('show');
  vfMountRemindRecipientPicker(ownerAccount, ownerLabel);
}

async function submitRemind() {
  if (!validateRemind()) return;
  const ctx = window._vfRemindCtx;
  if (!ctx || !ctx.demandId) return;
  const btn = document.getElementById('confirmRemind');
  if (btn) btn.disabled = true;
  const channels = [];
  if (document.getElementById('chInapp') && document.getElementById('chInapp').checked) channels.push('inapp');
  const manualRecipients = vfRemindPickedAccounts();
  const windowId = Number(ctx.windowId) || 0;
  if (!windowId) {
    showToast((ctx.displayId || '') + ' 催办失败：缺少版本窗口');
    if (btn) btn.disabled = false;
    return;
  }
  const body = { urgeType: ctx.urgeType, windowId: windowId, channels: channels, comment: '' };
  if (manualRecipients.length) body.manualRecipients = manualRecipients;
  try {
    const res = await apiFetch('/workbench/api/demands/' + ctx.demandId + '/urge', {
      method: 'POST',
      body: body
    });
    if (!res || res.success === false) {
      const msg = (res && (res.message || (res.errors && res.errors[0] && res.errors[0].message))) || '催办失败';
      showToast(ctx.displayId + ' 催办失败：' + msg);
      if (btn) btn.disabled = false;
      return;
    }
    if (res.duplicate === true) {
      showToast('60 秒内已催办过,跳过');
      closeRemindModal();
      return;
    }
    const results = Array.isArray(res.results) ? res.results : [];
    const failed = results.filter(function (r) { return !r || r.success !== true; });
    const errBox = document.getElementById('vfRemindChannelErrors');
    if (failed.length && failed.length === results.length) {
      if (errBox) {
        errBox.style.display = 'block';
        errBox.textContent = failed.map(function (r) { return (r.channel || '') + ': ' + (r.errMsg || '失败'); }).join('；');
      }
      if (btn) btn.disabled = false;
      return;
    }
    if (failed.length) {
      if (errBox) {
        errBox.style.display = 'block';
        errBox.textContent = '部分渠道失败：' + failed.map(function (r) {
          var who = r.recipient ? ('[' + r.recipient + ' / ' + (r.code || r.channel) + ']') : ('[' + (r.code || r.channel) + ']');
          return who + ' ' + (r.errMsg || '失败');
        }).join('；');
      }
      if (btn) btn.disabled = false;
      return;
    }
    showToast(ctx.displayId + ' ' + (ctx.label || '催办') + '已发出');
    closeRemindModal();
  } catch (e) {
    showToast(ctx.displayId + ' 催办失败，请稍后重试');
    if (btn) btn.disabled = false;
  }
}
window.openRemind = openRemind;
window.closeRemindModal = closeRemindModal;
window.validateRemind = validateRemind;
window.submitRemind = submitRemind;
/* 当前责任人跟着阶段走：开发看开发、测试看测试、验收看验收，其余回到 PO */
function vfCurrentOwner(row) {
  const stage = getCurrentStage(row);
  if (stage === '开发') return row.dev;
  if (stage === '测试') return row.test;
  if (stage === '验收') return row.accept || row.acceptOwner;
  return row.acceptOwner || row.accept;
}

// V1.2 R2.1 增量 A：九阶段节点与后端 ValueStreamStage / stageCountsByWindow 同 key。
const vfStageOrder = [
  { key: '', label: '全部', matcher: () => true },
  { key: 'accept', label: '受理', matcher: (r) => r.valueStreamStage === 'accept' },
  { key: 'clarify', label: '澄清', matcher: (r) => r.valueStreamStage === 'clarify' },
  { key: 'schedule', label: '排期', matcher: (r) => r.valueStreamStage === 'schedule' },
  { key: 'developing', label: '提测', matcher: (r) => r.valueStreamStage === 'developing' },
  { key: 'testing', label: '测试', matcher: (r) => r.valueStreamStage === 'testing' },
  { key: 'acceptance', label: '验收', matcher: (r) => r.valueStreamStage === 'acceptance' },
  { key: 'deliver', label: '发起交付', matcher: (r) => r.valueStreamStage === 'deliver' },
  { key: 'release', label: '发布', matcher: (r) => r.valueStreamStage === 'release' },
  { key: 'feedback', label: '评价反馈', matcher: (r) => r.valueStreamStage === 'feedback' }
];
const vfStageLabelMap = Object.fromEntries(vfStageOrder.map((stage) => [stage.key, stage.label]));

function formatVfShortRange(range) {
  const m = String(range || '').match(/(\d{4})-(\d{2})-(\d{2})\s*~\s*(\d{4})-(\d{2})-(\d{2})/);
  if (!m) return range;
  return `${m[2]}-${m[3]} ~ ${m[5]}-${m[6]}`;
}

function getVfExceptionFilter() {
  return window.currentVfExceptionFilter || 'all';
}

function rowMatchesVfException(row, exceptionFilter) {
  const f = exceptionFilter || 'all';
  if (f === 'all') return true;
  if (f === 'ready') return getVfReadiness(row) === 'ready';
  if (f === 'risk') return getVfReadiness(row) === 'risk';
  if (f === 'blocked' || f === '阻塞') return getVfReadiness(row) === 'blocked';
  /* 兼容首页/待办跳转带过来的旧口径值 */
  if (f === '有卡点') return getVfReadiness(row) !== 'ready';
  if (f === '超期') return isVfOverdue(row) || !!vfRowSlip(row);
  return true;
}

function getVfBaseRows() {
  return getVersionFollowRows().filter((r) => vfRowMatchesIteration(r, currentVfIteration));
}

function getCurrentVfRows() {
  mergeVfFilters();
  let rows = getVfBaseRows();
  if (currentVfStage) {
    const stage = vfStageOrder.find((item) => item.key === currentVfStage);
    if (stage) rows = rows.filter(stage.matcher);
  }
  rows = rows.filter((row) => rowMatchesVfException(row, getVfExceptionFilter()));
  const filters = window.currentVfFilters;
  if (filters.system) rows = rows.filter((row) => row.mainSystem === filters.system);
  if (filters.agile) rows = rows.filter((row) => row.agile === filters.agile);
  if (filters.relation) rows = rows.filter((row) => row.relation === filters.relation);
  if (filters.keyword) {
    const q = String(filters.keyword).toLowerCase();
    rows = rows.filter((row) => `${row.id} ${row.code} ${row.title} ${row.mainSystem}`.toLowerCase().includes(q));
  }
  if (filters.priority) rows = rows.filter((row) => row.pri === filters.priority);
  if (filters.owner) {
    const o = filters.owner;
    rows = rows.filter((row) => {
      if (row.acceptOwner === o) return true;
      return row.dev === o || row.test === o || row.accept === o;
    });
  }
  if (filters.releaseWindow) rows = rows.filter((row) => row.releaseWindow === filters.releaseWindow);
  if (filters.dateFrom || filters.dateTo) {
    rows = rows.filter((row) => {
      const d = String(row.targetDate || '');
      if (filters.dateFrom && d && d < filters.dateFrom) return false;
      if (filters.dateTo && d && d > filters.dateTo) return false;
      if ((filters.dateFrom || filters.dateTo) && !d) return false;
      return true;
    });
  }
  if (filters.myScope === 'mine') {
    const me = getVfMemberDisplayName();
    if (me) rows = rows.filter((row) => rowMatchesVfMyWork(row, me));
  }
  if (filters.project) rows = rows.filter((row) => String(row.project || '').includes(filters.project) || row.projectId === filters.project);
  if (filters.execution) rows = rows.filter((row) => String(row.execution || '').includes(filters.execution) || row.executionId === filters.execution);
  return rows;
}

function getVfVersionStats(iterId) {
  const rows = getVersionFollowRows().filter((r) => vfRowMatchesIteration(r, iterId));
  const riskOnly = rows.filter((r) => getVfReadiness(r) === 'risk').length;
  const blocked = rows.filter((r) => getVfReadiness(r) === 'blocked').length;
  /* risk 保持「有风险+阻塞」总险量，供健康度/旧口径；卡片展示用 riskOnly + blocked */
  return {
    total: rows.length,
    risk: riskOnly + blocked,
    riskOnly: riskOnly,
    blocked: blocked,
    waitDelivery: rows.filter((r) => String(r.deliveryStatus || '').includes('待交付')).length
  };
}

function getVfProgressStages(rows) {
  const backend = (typeof window.getVfStageCountsForCurrentWindow === 'function')
    ? window.getVfStageCountsForCurrentWindow()
    : null;
  const countsMap = backend && backend.counts ? backend.counts : null;
  const totalFromBackend = backend && typeof backend.total === 'number' ? backend.total : null;
  return vfStageOrder.map((stage) => {
    let count;
    if (!stage.key) {
      count = totalFromBackend != null ? totalFromBackend : rows.length;
    } else if (countsMap) {
      count = Number(countsMap[stage.key] || 0);
    } else {
      count = rows.filter(stage.matcher).length;
    }
    let tone = '';
    if (stage.key && rows.some((row) => stage.matcher(row) && isVfBlocked(row))) tone = 'danger';
    else if (stage.key && rows.some((row) => stage.matcher(row) && isVfOverdue(row))) tone = 'warning';
    return { key: stage.key, label: stage.label, count: count, tone };
  });
}


function getVfHealthLevel(iterId) {
  const st = getVfVersionStats(iterId);
  if (!st.total) return 'neutral';
  const ratio = st.risk / Math.max(st.total, 1);
  if (st.risk >= 3 || ratio >= 0.3) return 'danger';
  if (st.risk > 0 || st.waitDelivery >= 2) return 'warning';
  return 'ok';
}


function scrollVfWindows(dir) {
  /* 紧凑版（Goal §九）：‹ › 切换前后窗口，而不是横向滚动卡片列表 */
  const cards = orderedVfWindows();
  const curIdx = cards.findIndex((v) => String(v.id) === String(currentVfIteration || ''));
  const next = cards[curIdx + (dir || 1)];
  if (!next) return;
  if (next.isHistory || vfHistoryVersions.some((h) => String(h.id) === String(next.id))) selectVfHistoryVersion(next.id);
  else selectVfIteration(next.id);
}

function toggleVfAllWindowsMenu(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const pop = document.getElementById('vfAllWindowsPop');
  if (!pop) return;
  const open = pop.style.display === 'block';
  if (open) {
    pop.style.display = 'none';
    return;
  }
  renderVfAllWindowsPop();
  pop.style.display = 'block';
  setTimeout(() => {
    document.addEventListener('click', function vfAllWinClose() {
      pop.style.display = 'none';
      document.removeEventListener('click', vfAllWinClose);
    }, { once: true });
  }, 0);
}

function renderVfAllWindowsPop() {
  const pop = document.getElementById('vfAllWindowsPop');
  if (!pop) return;
  const cur = String(currentVfIteration || '');
  const row = (v, hist) => {
    const st = getVfVersionStats(v.id);
    const name = getVersionWindowShortName(v.id);
    const meta = hist
      ? `已上线 · ${st.total} 需`
      : `${st.total} 需 · ${st.riskOnly} 险 · ${st.blocked} 阻`;
    const fn = hist ? 'selectVfHistoryVersion' : 'selectVfIteration';
    return `<button type="button" class="vf-all-win-row${cur === String(v.id) ? ' active' : ''}" onclick="event.stopPropagation();${fn}('${vfJs(v.id)}');document.getElementById('vfAllWindowsPop').style.display='none'"><b>${vfH(name)}</b><small>${vfH(meta)}</small></button>`;
  };
  pop.innerHTML =
    '<div class="vf-all-win-group">当前 / 近期窗口</div>' +
    (vfPrimaryVersions.length ? vfPrimaryVersions.map((v) => row(v, false)).join('') : '<div class="vf-all-win-empty">暂无近期窗口</div>') +
    '<div class="vf-all-win-group vf-all-win-group--hist">已上线窗口</div>' +
    (vfHistoryVersions.length ? vfHistoryVersions.map((v) => row(v, true)).join('') : '<div class="vf-all-win-empty">暂无历史窗口</div>');
}

function vfStatusTagClass(status) {
  const s = String(status || '');
  if (/进行|开发|测试|验收/.test(s)) return 'doing';
  if (/准备|排期|规划/.test(s)) return 'plan';
  if (/已上线|已发布|关闭|完成/.test(s)) return 'done';
  return '';
}

function vfHealthLabel( readiness ) {
  if (readiness === 'blocked') return { cls: 'block', text: '■ 阻塞' };
  if (readiness === 'risk') return { cls: 'risk', text: '▲ 风险' };
  return { cls: 'ready', text: '● 可按期' };
}

function vfKeyTimesHtml(item) {
  const cells = vfMilestoneDefs.map((d) => {
    const plan = vfNormDate(d.plan(item));
    const done = d.done(item);
    const short = plan ? vfShortDate(plan) : '—';
    const late = plan && vfIsPast(plan) && !done;
    return `<div class="vf-kt${late ? ' late' : ''}${done ? ' done' : ''}"><span>${vfH(d.label.replace('完成',''))}</span><b>${vfH(short)}</b></div>`;
  }).join('');
  return `<div class="vf-key-times">${cells}</div>`;
}

function vfDemandCellHtml(item, caretHtml) {
  const sys = vfMeaningful(item.mainSystem) || vfMeaningful(item.system) || '';
  const project = item.project || '';
  const execution = item.execution || '';
  const pri = item.pri || item.priority || '';
  const meta = [sys, project, execution, pri].map(vfMeaningful).filter(Boolean).join(' · ');
  const id = item.code || item.id || '';
  const idJs = vfJs(id);
  const demJs = vfJs(item.id || id);
  return `<div class="vf-demand-cell">${caretHtml}<div class="vf-demand-main">` +
    `<div class="vf-demand-line"><button type="button" class="query-id-link" title="在禅道中查看原始详情" onclick="event.stopPropagation();viewInZentao('${vfJs(item.code || item.id)}','demand')">${vfH(id)}</button>` +
    `<button type="button" class="query-title-link" onclick="event.stopPropagation();openBusinessDemandDetail('${demJs}')">${vfH(item.title || '')}</button></div>` +
    (meta ? `<div class="vf-demand-meta">${vfH(meta)}</div>` : '') +
    `</div></div>`;
}

function vfMeaningful(v) {
  const s = String(v == null ? '' : v).trim();
  if (!s || s === '-' || s === '—' || /^undefined$/i.test(s) || /^null$/i.test(s)) return '';
  return s;
}

function renderVfReadinessBar() {
  const host = document.getElementById('vfReadinessBar');
  if (!host) return;
  /* 发布判断 pills：统计不受就绪度自身过滤影响，否则点一下数字其余归零 */
  const scopeRows = getVfReadinessScopeRows();
  const counts = getVfReadinessCounts(scopeRows);
  const active = getVfExceptionFilter();
  const pills = [
    { key: 'all', label: '全部', n: counts.total, tone: '', tip: '当前窗口与筛选下的全部需求' },
    { key: 'ready', label: '可按期', n: counts.ready, tone: 'ready', tip: '无阻塞、关键时间未滑期，或已交付/已验证' },
    { key: 'risk', label: '风险', n: counts.risk, tone: 'risk', tip: '超期，或关键时间已过但该环节未完成' },
    { key: 'blocked', label: '阻塞', n: counts.blocked, tone: 'blocked', tip: '存在阻塞或严重 Bug，影响本窗口发布' }
  ].map((v) => {
    const on = active === v.key || (active === '有卡点' && (v.key === 'risk' || v.key === 'blocked'));
    return `<button type="button" class="vf-health-pill tone-${v.tone}${on ? ' active' : ''}" title="${vfH(v.tip)}" onclick="setVfExceptionFilter('${v.key}')">${v.label} <strong>${v.n}</strong></button>`;
  }).join('');
  host.innerHTML = `<span class="vf-health-label">发布判断</span>${pills}`;
}

function getVfReadinessScopeRows() {
  const savedExc = window.currentVfExceptionFilter;
  const savedStage = currentVfStage;
  window.currentVfExceptionFilter = 'all';
  currentVfStage = '';
  try {
    return getCurrentVfRows();
  } finally {
    window.currentVfExceptionFilter = savedExc;
    currentVfStage = savedStage;
  }
}

/* 价值流 chip 计数范围 = 列表口径去掉「价值流阶段」本身 */
function getVfStageScopeRows() {
  const savedStage = currentVfStage;
  currentVfStage = '';
  try {
    return getCurrentVfRows();
  } finally {
    currentVfStage = savedStage;
  }
}

function renderVersionFollow() {
  renderVfReadinessBar();
  renderVfVersionSwitch();
  renderVfUnifiedBar();
  const fm = document.getElementById('vfFilterMore');
  if (fm) {
    fm.style.display = vfFilterPanelOpen ? 'block' : 'none';
    if (vfFilterPanelOpen) renderVfFilterMore();
  }
  renderVfProgressStrip();
  renderVfTabBody();
}

function toggleVfHistoryMenu(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  vfHistoryMenuOpen = !vfHistoryMenuOpen;
  if (!vfHistoryMenuOpen) vfHistorySearchQuery = '';
  renderVfVersionSwitch();
  if (vfHistoryMenuOpen) {
    setTimeout(() => {
      document.addEventListener('click', function vfHistDocClose() {
        vfHistoryMenuOpen = false;
        vfHistorySearchQuery = '';
        document.removeEventListener('click', vfHistDocClose);
        const host = document.getElementById('vfVersionSwitch');
        if (host && document.getElementById('page-versionFollow')?.classList.contains('active')) renderVfVersionSwitch();
      }, { once: true });
    }, 0);
  }
}

function clearVfFilters() {
  if (window._vfFilterDocClose) {
    document.removeEventListener('click', window._vfFilterDocClose);
    window._vfFilterDocClose = null;
  }
  currentVfStage = '';
  window.currentVfExceptionFilter = 'all';
  window.currentVfFilters = { system:'', agile:'', relation:'', keyword:'', status:'', priority:'', owner:'', releaseWindow:'', dateFrom:'', dateTo:'', myScope:'', project:'', execution:'' };
  resetVfListPage();
  vfFilterPanelOpen = false;
  applyVersionFollowFilters();
}

function goToScheduleAdjustFromVersionFollow(){
  goToScheduleByIteration(currentVfIteration);
}

function orderedVfWindows() {
  const cards = vfPrimaryVersions.slice();
  const histCurrent = vfHistoryVersions.find((v) => String(v.id) === String(currentVfIteration || ''));
  if (histCurrent && !cards.some((v) => String(v.id) === String(histCurrent.id))) cards.unshift(histCurrent);
  return cards;
}

function renderVfVersionSwitch() {
  const host = document.getElementById('vfVersionSwitch');
  const metaHost = document.getElementById('vfWindowMeta');
  if (!host) return;
  const currentIterationId = String(currentVfIteration || '');
  const noWindows = vfPrimaryVersions.length === 0 && vfHistoryVersions.length === 0;
  if (noWindows) {
    host.innerHTML = `
      <div class="vf-window-card vf-window-card--empty">
        <div class="vf-window-card-top"><span class="vf-window-name">尚未配置版本窗口</span></div>
      </div>`;
    if (metaHost) metaHost.textContent = '';
    return;
  }
  /* 紧凑版（Goal §九）：只渲染当前窗口为主卡，统计信息放 vfWindowMeta，前后窗口用 ‹ › 切换 */
  const cards = orderedVfWindows();
  const curIdx = cards.findIndex((v) => String(v.id) === currentIterationId);
  const cur = curIdx >= 0 ? cards[curIdx] : cards[0];
  const st = getVfVersionStats(cur.id);
  const tagCls = vfStatusTagClass(cur.status);
  const isHist = !!(cur.isHistory || vfHistoryVersions.some((h) => String(h.id) === String(cur.id)));
  host.innerHTML = `
    <button type="button" class="vf-window-card vf-window-card--current${isHist ? ' is-history' : ''}" title="${vfH(getVersionWindowTitle(cur.id))}" onclick="toggleVfAllWindowsMenu(event)">
      <div class="vf-window-card-top">
        <span class="vf-window-name">${vfH(getVersionWindowShortName(cur.id))}</span>
        <span class="vf-window-tag ${tagCls}">${vfH(cur.status || (isHist ? '已上线' : '进行中'))}</span>
        <i class="fas fa-chevron-down vf-window-caret"></i>
      </div>
    </button>`;
  if (metaHost) {
    const release = cur.releaseDate ? '上线 ' + String(cur.releaseDate).slice(5) + ' · ' : '';
    metaHost.textContent = release + `需求 ${st.total} · 风险 ${st.riskOnly} · 阻塞 ${st.blocked}`;
  }
  const pop = document.getElementById('vfAllWindowsPop');
  if (pop && pop.style.display === 'block') renderVfAllWindowsPop();
}

function setVfHistorySearchQuery(value) {
  vfHistorySearchQuery = value || '';
  renderVfVersionSwitch();
  setTimeout(() => {
    const input = document.getElementById('vfHistoryQuickSearch');
    if (!input) return;
    input.focus();
    const end = input.value.length;
    if (input.setSelectionRange) input.setSelectionRange(end, end);
  }, 0);
}


function setVfMoreFilter(key, value) {
  mergeVfFilters();
  window.currentVfFilters[key] = value || '';
  resetVfListPage();
  renderVersionFollow();
}

function renderVfFilterMore() {
  const host = document.getElementById('vfFilterMore');
  if (!host) return;
  mergeVfFilters();
  const filters = window.currentVfFilters;
  const owners = collectVfOwnerOptions();
  const rwOpts = Array.from(new Set(getVfBaseRows().map((r) => r.releaseWindow).filter(Boolean)));
  host.innerHTML = `
    <div class="vf-filter-more-inner" onclick="event.stopPropagation()">
      <label>优先级</label>
      <select class="form-select vf-select vf-select--compact" style="max-width:88px" onchange="setVfMoreFilter('priority',this.value)">
        <option value="" ${!filters.priority ? 'selected' : ''}>全部</option>
        <option value="P0" ${filters.priority === 'P0' ? 'selected' : ''}>P0</option>
        <option value="P1" ${filters.priority === 'P1' ? 'selected' : ''}>P1</option>
        <option value="P2" ${filters.priority === 'P2' ? 'selected' : ''}>P2</option>
      </select>
      <label>负责人</label>
      <select class="form-select vf-select vf-select--compact" style="max-width:120px" onchange="setVfMoreFilter('owner',this.value)">
        <option value="">全部</option>
        ${owners.map((o) => `<option value="${vfH(o)}" ${filters.owner === o ? 'selected' : ''}>${vfH(o)}</option>`).join('')}
      </select>
      <label>发布窗口</label>
      <select class="form-select vf-select vf-select--compact" style="max-width:110px" onchange="setVfMoreFilter('releaseWindow',this.value)">
        <option value="">全部</option>
        ${rwOpts.map((w) => `<option value="${vfH(w)}" ${filters.releaseWindow === w ? 'selected' : ''}>${vfH(w)}</option>`).join('')}
      </select>
      <label>计划日起</label>
      <input type="date" class="vf-date-input" value="${vfH(filters.dateFrom || '')}" onchange="setVfMoreFilter('dateFrom',this.value)" />
      <label>计划日止</label>
      <input type="date" class="vf-date-input" value="${vfH(filters.dateTo || '')}" onchange="setVfMoreFilter('dateTo',this.value)" />
    </div>`;
}

function getVfProjectOptions() {
  const map = new Map();
  getVfBaseRows().forEach((r) => {
    if (!r.project) return;
    const key = String(r.projectId || r.project);
    if (!map.has(key)) map.set(key, { id: key, name: String(r.project) });
  });
  return Array.from(map.values());
}

function getVfExecutionOptions(projectFilter) {
  const map = new Map();
  getVfBaseRows().forEach((r) => {
    if (projectFilter && String(r.projectId || r.project) !== String(projectFilter) && String(r.project || '') !== String(projectFilter)) return;
    if (!r.execution) return;
    const key = String(r.executionId || r.execution);
    if (!map.has(key)) map.set(key, { id: key, name: String(r.execution) });
  });
  return Array.from(map.values());
}

function getVfAgileOptions() {
  const map = new Map();
  getVfBaseRows().forEach((r) => {
    const label = String(r.agileGroup || r.agile || '').trim();
    if (!label || label === '-' || label === '—') return;
    const key = String(r.agileId || label);
    if (!map.has(key)) map.set(key, { id: key, name: label });
  });
  return Array.from(map.values()).sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'));
}

function setVfProjectFilter(value) {
  window.currentVfFilters.project = value || '';
  window.currentVfFilters.execution = '';
  renderVersionFollow();
}

function setVfExecutionFilter(value) {
  window.currentVfFilters.execution = value || '';
  renderVersionFollow();
}

function renderVfUnifiedBar() {
  const host = document.getElementById('vfUnifiedBar');
  if (!host) return;
  mergeVfFilters();
  const filters = window.currentVfFilters;
  const agileOpts = getVfAgileOptions();
  const projectOpts = getVfProjectOptions();
  const executionOpts = getVfExecutionOptions(filters.project);
  host.innerHTML = `
    <div class="vf-my-chips" onclick="event.stopPropagation()">
      <button type="button" class="vf-chip vf-chip--sm${!filters.relation ? ' active' : ''}" onclick="setVfRelationChip('')" title="在已选版本窗口与价值流下展示全量需求">全部</button>
      <span class="vf-chip-split" aria-hidden="true"></span>
      <button type="button" class="vf-chip vf-chip--sm${filters.relation === '我牵头' ? ' active' : ''}" onclick="setVfRelationChip('我牵头')">我牵头</button>
      <button type="button" class="vf-chip vf-chip--sm${filters.relation === '我配合' ? ' active' : ''}" onclick="setVfRelationChip('我配合')">我配合</button>
    </div>
    <div class="vf-filter-row-compact" onclick="event.stopPropagation()">
      <select class="form-select vf-select vf-select--compact" title="项目" onchange="setVfProjectFilter(this.value)">
        <option value="">项目：全部</option>
        ${projectOpts.map((p) => `<option value="${vfH(p.id)}" ${filters.project === p.id || filters.project === p.name ? 'selected' : ''}>${vfH(p.name)}</option>`).join('')}
      </select>
      <select class="form-select vf-select vf-select--compact" title="执行" onchange="setVfExecutionFilter(this.value)">
        <option value="">执行：全部</option>
        ${executionOpts.map((e) => `<option value="${vfH(e.id)}" ${filters.execution === e.id || filters.execution === e.name ? 'selected' : ''}>${vfH(e.name)}</option>`).join('')}
      </select>
      <select class="form-select vf-select vf-select--compact" title="系统" onchange="setVfInlineFilter('system',this.value)">
        <option value="" ${!filters.system ? 'selected' : ''}>系统：全部</option>
        ${Array.from(new Set(getVfBaseRows().map((r) => r.mainSystem).filter(Boolean))).map((v) => `<option value="${vfH(v)}" ${filters.system === v ? 'selected' : ''}>${vfH(v)}</option>`).join('')}
      </select>
      <select class="form-select vf-select vf-select--compact" title="敏捷小组" onchange="setVfInlineFilter('agile',this.value)">
        <option value="">小组：全部</option>
        ${agileOpts.length ? agileOpts.map((g) => `<option value="${vfH(g.id)}" ${filters.agile === g.id || filters.agile === g.name ? 'selected' : ''}>${vfH(g.name)}</option>`).join('') : '<option value="" disabled>暂无真实小组</option>'}
      </select>
      <div class="vf-search-inline">
        <i class="fas fa-magnifying-glass"></i>
        <input type="search" placeholder="关键词" value="${vfH(filters.keyword || '')}" oninput="setVfInlineFilter('keyword',this.value)" />
      </div>
    </div>
    <div class="vf-unified-actions" onclick="event.stopPropagation()">
      <div class="vf-layer-toggles vf-layer-toggles--inline" aria-label="层级控制">
        <span class="vf-layer-label">层级</span>
        <button type="button" class="dw-common-chip${window.vfShowRdLayer === true ? ' active' : ''}" title="一次展开/折叠当前页全部研发需求" onclick="toggleVfLayer('rd')">${window.vfShowRdLayer === true ? '折叠研需' : '展开研需'}</button>
        <button type="button" class="dw-common-chip${window.vfShowTaskLayer === true ? ' active' : ''}" title="一次展开/折叠当前页全部任务（展开任务会先展开研需）" onclick="toggleVfLayer('task')">${window.vfShowTaskLayer === true ? '折叠任务' : '展开任务'}</button>
      </div>
      <button type="button" class="action-btn${vfFilterPanelOpen ? ' primary' : ''}" onclick="toggleVfFilterPanel(event)"><i class="fas fa-sliders"></i> 更多筛选</button>
      <button type="button" class="action-btn" onclick="clearVfFilters()">清空</button>
    </div>`;
}

function renderVfProgressStrip() {
  const host = document.getElementById('vfProgressStrip');
  if (!host) return;
  const stages = getVfProgressStages(getVfStageScopeRows()).filter((s) => s.key);
  const version = getCurrentVfVersion();
  host.innerHTML = `
    <div class="vf-node-track" role="list">
      ${stages.map((stage) => {
        const selected = currentVfStage === stage.key;
        const hasData = stage.count > 0;
        /* 节点下方日期：窗口级里程碑（开发/测试/验收/上线）映射到对应阶段 */
        let dateHtml = '&nbsp;';
        const ms = getVfWindowMilestones(getVfReadinessScopeRows());
        const hit = ms.find((m) => {
          if (stage.key === 'developing') return m.label.indexOf('开发') >= 0;
          if (stage.key === 'testing') return m.label.indexOf('测试') >= 0;
          if (stage.key === 'acceptance') return m.label.indexOf('验收') >= 0;
          if (stage.key === 'release') return m.label.indexOf('上线') >= 0 || m.label.indexOf('发布') >= 0;
          return false;
        });
        if (hit && hit.planDate) {
          dateHtml = `${vfH(hit.label.replace('完成',''))} <b>${vfH(vfShortDate(hit.planDate))}</b>`;
        } else if (stage.key === 'release' && version && version.eta) {
          dateHtml = `上线 <b>${vfH(vfShortDate(version.eta))}</b>`;
        }
        return `
          <div class="vf-stage-node${selected ? ' selected' : ''}${hasData ? ' has-data' : ''}${stage.tone ? ' tone-' + stage.tone : ''}" role="listitem">
            <button type="button" class="vf-stage-node-btn" onclick="selectVfStage('${stage.key === currentVfStage ? '' : stage.key}')">
              <div class="vf-stage-dot" aria-hidden="true"></div>
              <div class="vf-stage-name">${vfH(stage.label)}</div>
              <div class="vf-stage-num"><b>${stage.count}</b> 需</div>
              <div class="vf-stage-date">${dateHtml}</div>
            </button>
          </div>`;
      }).join('')}
    </div>`;
}

function toggleVfFilterPanel(ev) {
  if (ev && ev.stopPropagation) ev.stopPropagation();
  const wasOpen = vfFilterPanelOpen;
  vfFilterPanelOpen = !vfFilterPanelOpen;
  if (wasOpen && window._vfFilterDocClose) {
    document.removeEventListener('click', window._vfFilterDocClose);
    window._vfFilterDocClose = null;
  }
  renderVfUnifiedBar();
  const fm = document.getElementById('vfFilterMore');
  if (fm) {
    fm.style.display = vfFilterPanelOpen ? 'block' : 'none';
    if (vfFilterPanelOpen) renderVfFilterMore();
  }
  if (vfFilterPanelOpen) {
    setTimeout(() => {
      window._vfFilterDocClose = function vfFilterDocClose() {
        vfFilterPanelOpen = false;
        document.removeEventListener('click', window._vfFilterDocClose);
        window._vfFilterDocClose = null;
        const page = document.getElementById('page-versionFollow');
        if (page && page.classList.contains('active')) {
          renderVfUnifiedBar();
          const el = document.getElementById('vfFilterMore');
          if (el) el.style.display = 'none';
        }
      };
      document.addEventListener('click', window._vfFilterDocClose, { once: true });
    }, 0);
  }
}

function setVfExceptionFilter(value) {
  window.currentVfExceptionFilter = value || 'all';
  resetVfListPage();
  renderVersionFollow();
}

function vfH(s) {
  // C1-01 单源收口: 委托 po-core.js 的 poEscapeHtml。原实现只转 3 字符 (& < ")
  // 已并入完整的 5 字符 (& < > " ') 实现, 行为对原调用点安全 (5 字符 ⊇ 3 字符)。
  return WBUtils.escapeHtml(s);
}

function vfJs(s) {
  const raw = String(s ?? '')
    .replace(/\\/g, '\\\\')
    .replace(/\r/g, '\\r')
    .replace(/\n/g, '\\n')
    .replace(/'/g, "\\'");
  return vfH(raw);
}

function vfRowActions(id) {
  return `<div class="vf-detail-actions"><button type="button" class="action-btn primary" onclick="openBusinessDemandDetail('${vfJs(id)}')">需求详情</button></div>`;
}

function renderVfTabBody() {
  const host = document.getElementById('vfTabBody');
  if (!host) return;
  host.innerHTML = renderVfOverview(getCurrentVfRows());
  applyVersionFollowColumnVisibility();
}

function applyVersionFollowColumnVisibility() {
  /* V1.2.3：版本跟进为 6 列固定 IA，不参与统一列显隐（applyTableColumnVisibility 会通过
     <col style="width:0; visibility:collapse; !important"> 把被关列彻底 0 化；
     fixed 表下未声明列的 <th> 显式 width 也救不回来，剩余列会被对半分）。 */
  const tbl = document.querySelector('#page-versionFollow .tbl');
  if (!tbl) return;
  const cg = tbl.querySelector('colgroup');
  if (cg) {
    [...cg.children].forEach((col) => {
      // 之前 applyTableColumnVisibility 用 setProperty(..., '!important') 写入，
      // removeProperty 只能清掉普通声明；要重置 inline style 必须直接赋空串。
      col.setAttribute('style', '');
    });
  }
  [...tbl.querySelectorAll('thead th, tbody td')].forEach((el) => {
    el.style.removeProperty('visibility');
    el.style.removeProperty('display');
    el.classList.remove('is-col-hidden');
  });
}

function renderVfOverview(rows) {
  const scoreRow = (row) => {
    const blockedScore = isVfBlocked(row) ? 100 : 0;
    const slipScore = vfRowSlip(row) ? 30 : 0;
    const overdueScore = isVfOverdue(row) ? 20 : 0;
    const priScore = String(row.pri || '') === 'P0' ? 15 : String(row.pri || '') === 'P1' ? 8 : 0;
    const deliveryScore = String(row.deliveryStatus || '').includes('待交付') ? 8 : 0;
    const acceptScore = String(row.acceptanceStatus || '').includes('待验收') ? 6 : 0;
    return blockedScore + slipScore + overdueScore + priScore + deliveryScore + acceptScore;
  };
  const mergedRows = rows.slice()
    .sort((a, b) => scoreRow(b) - scoreRow(a))
    .filter((row, idx, arr) => arr.findIndex((item) => item.id === row.id) === idx);
  vfListPaginationState.total = mergedRows.length;
  // pageSize 默认 15 是本页本地 policy，按 shared 层 §13 约定留在调用方。
  const pageSize = Math.max(1, Number(vfListPaginationState.pageSize) || 15);
  const totalPages = WBPagination.totalPages(mergedRows.length, pageSize);
  vfListPaginationState.page = WBPagination.clampPage(vfListPaginationState.page, totalPages);
  const pageStart = WBPagination.offset(vfListPaginationState.page, pageSize);
  const pageRows = mergedRows.slice(pageStart, pageStart + pageSize);
  if (typeof window.vfShowRdLayer !== 'boolean') window.vfShowRdLayer = false;
  if (typeof window.vfShowTaskLayer !== 'boolean') window.vfShowTaskLayer = false;
  window.vfExpandedRows = window.vfExpandedRows || {};
  const expandedMap = window.vfExpandedRows;
  const showRd = window.vfShowRdLayer === true;
  const showTask = window.vfShowTaskLayer === true;
  const emptyCell = '<span class="vf-empty">—</span>';

  const renderEmpty = (text) => `<div class="vf-mini-empty">${text}<div style="margin-top:10px;display:flex;gap:8px;justify-content:center;flex-wrap:wrap"><button type="button" class="action-btn" onclick="clearVfFilters()">清空筛选</button><button type="button" class="action-btn primary" onclick="goToScheduleByIteration('${vfJs(currentVfIteration || '')}')">查看排期</button></div></div>`;

  const caretBtn = (key, open, tip) => `<button type="button" class="vf-caret" title="${vfH(tip)}" aria-expanded="${open ? 'true' : 'false'}" onclick="event.stopPropagation();toggleVfRow('${vfJs(key)}')"><i class="fas fa-chevron-${open ? 'down' : 'right'}"></i></button>`;

  const healthBadge = (readiness, note) => {
    const hl = vfHealthLabel(readiness);
    return `<div class="vf-health-status ${hl.cls}">${hl.text}</div>` + (note ? `<div class="vf-health-note">${vfH(note)}</div>` : '');
  };

  const renderChild = (row) => {
    const type = row.type;
    const title = row.title;
    const badge = row.badge || '';
    const meta = row.meta || '';
    const progress = row.progress;
    const action = row.action || '';
    const typeCls = type === '任务' ? 'task' : (type === '研需' ? 'rd' : (type === '审批' ? 'approval' : ''));
    const titleHtml = row.titleClick
      ? `<button type="button" class="vf-child-title vf-child-title--link" onclick="event.stopPropagation();${row.titleClick}">${vfH(title || '')}</button>`
      : `<span class="vf-child-title">${vfH(title || '')}</span>`;
    return `<div class="vf-child-item${typeCls ? ' vf-child-item--' + typeCls : ''}">
      <div><span class="vf-child-type">${vfH(type || '')}</span>${titleHtml}<div class="vf-child-meta">${vfH(meta || '')}</div></div>
      <div>${badge}</div>
      <div>${progress ? `<div class="vf-child-progress"><i style="width:${Math.max(0, Math.min(100, Number(progress) || 0))}%"></i></div><span class="vf-child-progress-pct">${Math.max(0, Math.min(100, Number(progress) || 0))}%</span>` : ''}</div>
      <div>${action}</div>
    </div>`;
  };

  const taskProgressPct = (t) => {
    if (t && t.progress != null && !isNaN(Number(t.progress))) return Math.max(0, Math.min(100, Number(t.progress)));
    const c = Number(t && (t.consumed != null ? t.consumed : 0)) || 0;
    const left = Number(t && (t.left != null ? t.left : 0)) || 0;
    const est = Number(t && (t.hours != null ? t.hours : (t.estimate != null ? t.estimate : 0))) || 0;
    const denom = (c + left) || est;
    return denom > 0 ? Math.max(0, Math.min(100, Math.round(c / denom * 100))) : 0;
  };

  const pushTaskChild = (items, t) => {
    const tid = t.taskId || t.id || '';
    const name = String(t.taskName || t.title || '').replace(/^【[^】]*】\s*/, '');
    const label = (t.id && String(t.id).indexOf('TASK-') === 0 ? t.id + ' ' : '') + name;
    items.push({
      type: '任务',
      title: label,
      meta: `${t.owner || '待分配'} · 预计 ${t.hours != null ? t.hours : (t.estimate != null ? t.estimate : '—')}h`,
      badge: `<span class="status-tag">${vfH(t.status || '未开始')}</span>`,
      progress: taskProgressPct(t),
      titleClick: `openTaskDetail('${vfJs(tid)}')`,
      action: `<button type="button" class="action-btn link" onclick="event.stopPropagation();openTaskDetail('${vfJs(tid)}')">任务详情</button>`
    });
  };

  const buildDemandChildRows = (item) => {
    const tree = resolveVfDemandTree(item);
    const rdsAll = tree.rds;
    const directTasksAll = tree.directTasks;
    const items = [];
    const approvalStages = Array.isArray(item.pendingApprovalStages) ? item.pendingApprovalStages : [];
    if (approvalStages.length) {
      approvalStages.forEach((stg) => {
        const labelMap = { 'initial-review': '需求评审', 'manager-review': '主管部门审批', 'change-review': '需求变更审批' };
        const reviewers = (item.pendingApprovalReviewers && item.pendingApprovalReviewers[stg]) || [];
        items.push({
          type: '审批', title: labelMap[stg] || stg,
          meta: reviewers.length ? `${reviewers.length} 人待处理` : '',
          badge: `<span class="status-tag st-warning">${vfH(approvalStages.length > 1 ? '多重审批' : '审批中')}</span>`
        });
      });
    }
    rdsAll.forEach((rd) => {
      const tasks = Array.isArray(rd.createdTasks) ? rd.createdTasks : [];
      const rdLabel = (rd.id ? String(rd.id) + ' ' : '') + (rd.title || '');
      items.push({
        type: '研需', title: rdLabel,
        meta: `${rd.rdAssignee || rd.dev || '待分配'} · 已关联 ${tasks.length} 个任务`,
        badge: `<span class="status-tag">${vfH(rd.status || '未开始')}</span>`,
        progress: tasks.length ? Math.round(tasks.filter((t) => /完成|已上线|已交付|done|closed/i.test(String(t.status || ''))).length / tasks.length * 100) : 0,
        titleClick: `viewInZentao('${vfJs(rd.storyId || rd.id)}','story')`,
        action: `<button type="button" class="action-btn link" onclick="event.stopPropagation();openBusinessDemandDetail('${vfJs(item.id)}')">查看研需</button>`
      });
      /* 展开任务：在每条研需后挂出其任务子层 */
      if (showTask) tasks.forEach((t) => pushTaskChild(items, t));
    });
    /* 无研需时的直属任务：需求层展开即可见（与旧树一致） */
    directTasksAll.forEach((t) => pushTaskChild(items, t));
    return items;
  };

  const renderCombinedList = (items) => {
    const hi = versionFollowHighlightId ? String(versionFollowHighlightId) : '';
    return `
    <table class="tbl vf-table">
      <thead><tr>
        <th data-col="demand">需求 / 系统与执行</th>
        <th data-col="stage">PO价值流阶段</th>
        <th data-col="keyTime">关键时间</th>
        <th data-col="health">发布判断</th>
        <th data-col="owner">当前责任人</th>
        <th data-col="next">下一步</th>
      </tr></thead>
      <tbody>
        ${items.map((item) => {
          const stage = getCurrentStage(item);
          const readiness = getVfReadiness(item);
          const slip = vfRowSlip(item);
          const reason = (item.reason && item.reason !== '-') ? item.reason : ((Number(item.severe) || 0) > 0 ? '严重 Bug ' + item.severe : '');
          const rcls = hi && String(item.id) === hi ? ' class="vf-row-focus"' : '';
          const healthNote = reason ? reason : (slip ? `${slip.label}超期 ${slip.days} 天` : '');
          const approvalStages = Array.isArray(item.pendingApprovalStages) ? item.pendingApprovalStages : [];
          const owner = vfCurrentOwner(item) || '待分配';
          const ownerRole = stage === '开发' ? '开发负责人' : stage === '测试' ? '测试负责人' : stage === '验收' ? '验收负责人' : '当前责任人';
          const itemID = item.id || item.code || item.demandId || '';
          const actionDemandID = item.demandId || itemID;
          const itemIDJs = vfJs(itemID);
          const actionDemandIDJs = vfJs(actionDemandID);
          const ownerJs = vfJs(owner);
          const currentWindowJs = vfJs(currentVfIteration || item.windowId || '');

          /* 关键时间（行级：四个里程碑） */
          const keyTimesHtml = vfKeyTimesHtml(item);

          /* 责任人：审批多责任人优先展示 */
          const ownerHtml = approvalStages.length
            ? `<div class="vf-owner-cell"><div class="vf-owner">${vfH(item.pendingApprovalReviewers && Object.values(item.pendingApprovalReviewers).flat().filter(Boolean).slice(0,2).join('、') || owner)}</div><div class="vf-owner-role">待评审 ${approvalStages.length} 项</div></div>`
            : `<div class="vf-owner-cell"><div class="vf-owner">${vfH(owner)}</div><div class="vf-owner-role">${vfH(ownerRole)}</div></div>`;

          /* 下一步 action-stack */
          const acts = [];
          const approvalUrgeType = vfUrgeTypeFromStages(approvalStages);
          if (approvalUrgeType) {
            const reviewers = (item.pendingApprovalReviewers && item.pendingApprovalReviewers[approvalUrgeType]) || [];
            const count = Array.isArray(reviewers) ? reviewers.length : 0;
            const label = VF_URGE_LABELS[approvalUrgeType] || '催办审批';
            acts.push(`<button type="button" class="action-btn warn write-action" onclick="openRemind('${vfJs(approvalUrgeType)}','${itemIDJs}',{owner:'${ownerJs}',reason:'存在未完成审批',reviewerCount:${count}})">${vfH(label)}</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openApprovalView('${actionDemandIDJs}','${vfJs(approvalUrgeType)}')">查看审批</button>`);
          } else if (String(item.deliveryStatus || '').includes('待交付')) {
            // V1.2.3 fix-accept: VF「去发起交付/去验收」直弹需求详情，不再落 AD 菜单
            // （AD 深链函数本身仍保留，供主页价值流/通知中心复用）
            const gotoDeliverLabel = isReadOnlyVersionFollow() ? '查看交付' : '去发起交付';
            acts.push(`<button type="button" class="action-btn primary write-action" onclick="openBusinessDemandDetail('${actionDemandIDJs}', { actionCode: 'pendingDeliver' })">${vfH(gotoDeliverLabel)}</button>`);
          } else if (item.valueStreamStage === 'schedule' || stage === '排期') {
            acts.push(`<button type="button" class="action-btn primary" onclick="goToScheduleByIteration('${currentWindowJs}', '${itemIDJs}')">去排期</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          } else if (item.valueStreamStage === 'acceptance' || stage === '验收') {
            const gotoAcceptLabel = isReadOnlyVersionFollow() ? '查看验收' : '去验收';
            // V1.2.4：去验收 = primary 实色（唯一主操作）；催验收 = link 蓝无边框（次操作）
            acts.push(`<button type="button" class="action-btn primary write-action" onclick="openBusinessDemandDetail('${actionDemandIDJs}')">${vfH(gotoAcceptLabel)}</button>`);
            if (!isReadOnlyVersionFollow()) {
              acts.push(`<button type="button" class="action-btn link write-action" onclick="openRemind('acceptance','${itemIDJs}',{owner:'${ownerJs}',reason:'需求已进入验收阶段'})">催验收</button>`);
            }
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          } else if (item.valueStreamStage === 'developing') {
            // V1.2.4：按原型 — 催开发 = link 蓝无边框
            acts.push(`<button type="button" class="action-btn link write-action" onclick="openRemind('dev','${itemIDJs}',{owner:'${ownerJs}',reason:'开发进度需跟进'})">催开发</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          } else if (item.valueStreamStage === 'testing') {
            acts.push(`<button type="button" class="action-btn link write-action" onclick="openRemind('test','${itemIDJs}',{owner:'${ownerJs}',reason:'测试进度需跟进'})">催测试</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          } else if (item.valueStreamStage === 'release') {
            // 催发布：上线时间临近走 warn 橙底
            acts.push(`<button type="button" class="action-btn warn write-action" onclick="openRemind('release','${itemIDJs}',{owner:'${ownerJs}',reason:'发布进度需跟进'})">催发布</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">查看发布</button>`);
          } else if (readiness !== 'ready') {
            acts.push(`<button type="button" class="action-btn link write-action" onclick="openRemind('dev','${itemIDJs}',{owner:'${ownerJs}',reason:'请尽快处理当前责任事项'})">催办</button>`);
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          } else {
            acts.push(`<button type="button" class="action-btn link" onclick="openBusinessDemandDetail('${itemIDJs}')">详情</button>`);
          }

          const tree = resolveVfDemandTree(item);
          const rdsAll = tree.rds;
          const hasChildren = (rdsAll.length > 0) || (tree.directTasks.length > 0) || (item.pendingApprovalStages && item.pendingApprovalStages.length > 0);
          const bizOpen = showRd || expandedMap[item.id] === true;
          const caret = hasChildren ? caretBtn(itemID, bizOpen, bizOpen ? '收起研需/审批子层' : '展开研需/审批子层') : '<span class="vf-caret-pad"></span>';

          const demandCellHtml = vfDemandCellHtml(item, caret);
          const stageBadgeHtml = `<span class="vf-stage-badge">${vfH(stage)}</span>${item.statusRaw ? `<div class="vf-stage-raw">${vfH(item.statusRaw)}</div>` : ''}`;
          const healthCellHtml = healthBadge(readiness, healthNote);
          const nextCellHtml = `<div class="action-stack">${acts.join('')}</div>`;

          const bizRow = `<tr${rcls}>
            <td data-col="demand">${demandCellHtml}</td>
            <td data-col="stage">${stageBadgeHtml}</td>
            <td data-col="keyTime">${keyTimesHtml}</td>
            <td data-col="health">${healthCellHtml}</td>
            <td data-col="owner">${ownerHtml}</td>
            <td data-col="next">${nextCellHtml}</td>
          </tr>`;
          if (!bizOpen) return bizRow;
          const children = buildDemandChildRows(item);
          if (!children.length) return bizRow;
          const childRow = `<tr class="vf-child-row"><td colspan="6"><div class="vf-children">${children.map(renderChild).join('')}</div></td></tr>`;
          return bizRow + childRow;
        }).join('')}
      </tbody>
    </table>`;
  };

  mergeVfFilters();
  const filters = window.currentVfFilters;
  const filterTagParts = [];
  if (filters.system) filterTagParts.push(`<span class="vf-filter-tag">系统：${vfH(filters.system)}</span>`);
  if (filters.agile) filterTagParts.push(`<span class="vf-filter-tag">小组：${vfH((getVfAgileOptions().find((g) => g.id === filters.agile || g.name === filters.agile) || {}).name || filters.agile)}</span>`);
  if (filters.relation) filterTagParts.push(`<span class="vf-filter-tag">关系：${vfH(filters.relation)}</span>`);
  if (filters.keyword) filterTagParts.push(`<span class="vf-filter-tag">关键词：${vfH(filters.keyword)}</span>`);
  if (filters.priority) filterTagParts.push(`<span class="vf-filter-tag">优先级：${vfH(filters.priority)}</span>`);
  if (filters.owner) filterTagParts.push(`<span class="vf-filter-tag">负责人：${vfH(filters.owner)}</span>`);
  if (filters.releaseWindow) filterTagParts.push(`<span class="vf-filter-tag">窗口：${vfH(filters.releaseWindow)}</span>`);
  if (filters.dateFrom || filters.dateTo) filterTagParts.push(`<span class="vf-filter-tag">日期：${vfH(filters.dateFrom || '…')}~${vfH(filters.dateTo || '…')}</span>`);
  if (filters.myScope === 'mine') filterTagParts.push('<span class="vf-filter-tag">按我过滤</span>');
  const filterSummaryBlock = filterTagParts.length
    ? WBFilterShell.active(`已选：${filterTagParts.join('')}`, 'vf-filter-summary', ' onclick="event.stopPropagation()"')
    : '';
  const pagerHtml = typeof renderPagination === 'function'
    ? `<div id="vfListPagination" class="pagination-container">${renderPagination({
        total: mergedRows.length,
        page: vfListPaginationState.page,
        pageSize: pageSize,
        pageSizeOptions: [10, 15, 20, 50],
        onPageChange: function (newPage) { vfListPaginationState.page = Number(newPage) || 1; renderVfTabBody(); },
        onPageSizeChange: function (newSize) { vfListPaginationState.page = 1; vfListPaginationState.pageSize = Number(newSize) || 15; renderVfTabBody(); }
      })}</div>`
    : '';
  return WBListShell.render({
    tag: 'section',
    surfaceClass: 'vf-list-card',
    scrollClass: 'vf-table-scroll',
    before: `<div class="vf-list-head">
        <div class="vf-list-title">版本需求</div>
        <div class="vf-list-count">共 ${mergedRows.length} 条</div>
        <div class="vf-layer-actions">
          <button type="button" class="action-btn vf-layer-toggle${window.vfShowRdLayer === true ? ' primary' : ''}" onclick="toggleVfLayer('rd')">${window.vfShowRdLayer === true ? '折叠研需' : '展开研需'}</button>
          <button type="button" class="action-btn vf-layer-toggle${window.vfShowTaskLayer === true ? ' primary' : ''}" onclick="toggleVfLayer('task')">${window.vfShowTaskLayer === true ? '折叠任务' : '展开任务'}</button>
        </div>
      </div>${filterSummaryBlock}`,
    content: mergedRows.length ? renderCombinedList(pageRows) : renderEmpty('当前版本暂无数据'),
    pagerHtml: pagerHtml
  });
}

function setVfInlineFilter(key, value) {
  mergeVfFilters();
  window.currentVfFilters[key] = value || '';
  resetVfListPage();
  renderVersionFollow();
}

function selectVfStage(stageId) {
  currentVfStage = String(stageId || '');
  resetVfListPage();
  renderVersionFollow();
}

function selectVfHistoryVersion(iterId) {
  vfHistoryMenuOpen = false;
  vfHistorySearchQuery = '';
  selectVfIteration(iterId);
}

function selectVfIteration(iterId) {
  if (window._vfFilterDocClose) {
    document.removeEventListener('click', window._vfFilterDocClose);
    window._vfFilterDocClose = null;
  }
  currentVfIteration = iterId || (vfPrimaryVersions[0] ? vfPrimaryVersions[0].id : (vfHistoryVersions[0] ? vfHistoryVersions[0].id : ''));
  currentVfStage = '';
  window.currentVfExceptionFilter = 'all';
  resetVfListPage();
  vfHistoryMenuOpen = false;
  vfHistorySearchQuery = '';
  vfFilterPanelOpen = false;
  renderVersionFollow();
  updateReadOnlyBanner();
}

// 获取当前阶段
function getCurrentStage(r) {
  if (String(r.deliveryStatus).includes('待交付') || String(r.deliveryStatus).includes('已交付')) return '交付';
  if (String(r.acceptanceStatus).includes('待验收')) return '验收';
  if (String(r.testStatus).includes('测试完成')) return '测试';
  if (String(r.devStatus).includes('开发完成') || String(r.devStatus).includes('已提测')) return '开发';
  /* 阶段名与「版本价值流」chip 用同一套词，避免同页出现两种叫法 */
  if (String(r.taskStatus).includes('任务已建')) return '建任务';
  if (String(r.transferStatus).includes('已转研')) return '转研';
  return '排期';
}

// 应用筛选
function applyVersionFollowFilters() {
  renderVersionFollow();
}




// Phase 7C canonicalized backend truth.
// PO 版本跟进后端真源桥接层。
// 数据唯一入口：GET /workbench/api/version-follow。
// 本文件只做 DTO -> 既有 UI row 的展示适配，不再从前端旧缓存推导窗口归属。
(function () {
  'use strict';

  let backendItems = [];
  let backendWindows = [];
  let backendStageCountsByWindow = [];
  let loaded = false;
  let loading = null;

  function text(v) { return String(v == null ? '' : v).trim(); }
  function todayYmd() {
    const now = new Date();
    return now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0') + '-' + String(now.getDate()).padStart(2, '0');
  }
  function uniqueTexts(values) {
    return Array.from(new Set((values || []).map(text).filter(Boolean)));
  }
  function joinFacts(values) {
    const list = uniqueTexts(values);
    return list.length ? list.join(' / ') : '';
  }
  function allTasks(stories) {
    const out = [];
    (stories || []).forEach(function (story) {
      (story.tasks || []).forEach(function (task) { out.push(task); });
    });
    return out;
  }

  function aggregateQuality(stories) {
    const facts = {
      testTotal: 0,
      testExecuted: 0,
      testPassed: 0,
      bugTotal: 0,
      bugOpen: 0,
      severeOpen: 0
    };
    (stories || []).forEach(function (story) {
      const tests = story && story.tests ? story.tests : {};
      const bugs = story && story.bugs ? story.bugs : {};
      facts.testTotal += Number(tests.total || 0);
      facts.testExecuted += Number(tests.executed || 0);
      facts.testPassed += Number(tests.passed || 0);
      facts.bugTotal += Number(bugs.total || 0);
      facts.bugOpen += Number(bugs.open || 0);
      facts.severeOpen += Number(bugs.severeOpen || 0);
    });
    return facts;
  }

  const valueStageLabels = {
    accept: '受理',
    clarify: '澄清',
    schedule: '排期',
    developing: '提测',
    testing: '测试',
    acceptance: '验收',
    deliver: '发起交付',
    release: '发布',
    feedback: '评价反馈'
  };

  function stagePresentation(stage, stories) {
    const s = text(stage);
    const tasks = allTasks(stories);
    const hasTasks = tasks.length > 0;
    const rank = {
      accept: 0,
      clarify: 1,
      schedule: 2,
      developing: 3,
      testing: 4,
      acceptance: 5,
      deliver: 6,
      release: 7,
      feedback: 8
    };
    const n = Object.prototype.hasOwnProperty.call(rank, s) ? rank[s] : -1;
    return {
      scheduleStatus: n === 2 ? '待排期' : (n > 2 ? '已排期' : '未到排期'),
      transferStatus: n >= 3 ? '已转研' : (n >= 0 ? '未转研' : '无数据'),
      // 是否存在任务取真实 nested task，不由阶段自动声称“任务已建”。
      taskStatus: hasTasks ? '任务已建' : '无数据',
      devStatus: n === 3 ? '开发中' : (n > 3 ? '开发完成' : '未开始'),
      testStatus: n === 4 ? '测试中' : (n > 4 ? '测试完成' : '未开始'),
      acceptanceStatus: n === 5 ? '待验收' : (n > 5 ? '已验收' : '未验收'),
      deliveryStatus: n === 6 ? '待交付' : (n > 6 ? '已交付' : '未交付'),
      prodVerify: n >= 7 ? '发布阶段' : '-'
    };
  }

  function mapRelation(role) {
    const r = text(role);
    // 现有版本跟进筛选组件仍使用“我牵头/我配合”展示词；后端保持统一参与关系“我负责/我配合”。
    if (r === '我负责') return '我牵头';
    if (r === '我配合') return '我配合';
    if (r === '我关注') return '我关注';
    return r;
  }

  function mapStory(story) {
    return {
      id: 'RD-' + story.id,
      storyId: Number(story.id || 0),
      title: story.title || '-',
      system: story.product || '',
      status: story.stage || story.status || '',
      rdAssignee: story.assignee || story.assignedTo || '',
      dev: story.assignee || story.assignedTo || '',
      targetDate: '',
      tests: story.tests || { total: 0, executed: 0, passed: 0 },
      bugs: story.bugs || { total: 0, open: 0, severeOpen: 0 },
      createdTasks: (story.tasks || []).map(function (task) {
        return {
          id: 'TASK-' + task.id,
          taskId: Number(task.id || 0),
          storyId: Number(task.storyId || story.id || 0),
          taskName: task.name || '-',
          title: task.name || '-',
          taskType: task.type || '',
          status: task.status || '',
          owner: task.assignee || task.assignedTo || '',
          ownerAccount: task.assignedTo || '',
          hours: Number(task.estimate || 0),
          consumed: Number(task.consumed || 0),
          left: Number(task.left || 0),
          dueDate: task.deadline || '',
          project: task.project || '',
          projectId: Number(task.projectId || 0) || '',
          execution: task.execution || '',
          executionId: Number(task.executionId || 0) || ''
        };
      })
    };
  }

  function mapItem(item) {
    const stories = (item.stories || []).map(mapStory);
    const tasks = allTasks(item.stories || []);
    const quality = aggregateQuality(item.stories || []);
    const presentation = stagePresentation(item.valueStreamStage, item.stories || []);
    const devNames = uniqueTexts((item.stories || []).map(function (s) { return s.assignee || s.assignedTo; }));
    const testNames = uniqueTexts(tasks.filter(function (t) {
      return /test|测试/i.test(text(t.type));
    }).map(function (t) { return t.assignee || t.assignedTo; }));
    const projectNames = uniqueTexts(tasks.map(function (t) { return t.project; }));
    const executionNames = uniqueTexts(tasks.map(function (t) { return t.execution; }));
    const projectIDs = uniqueTexts(tasks.map(function (t) { return t.projectId; }));
    const executionIDs = uniqueTexts(tasks.map(function (t) { return t.executionId; }));
    const riskFlags = Array.isArray(item.riskFlags) ? item.riskFlags : [];
    const reasons = [];
    if (item.isBlocked || riskFlags.includes('blocked')) reasons.push('阻塞');
    if (riskFlags.includes('overdue')) reasons.push('超期');
    if (riskFlags.includes('change')) reasons.push('变更');

    // 用例是 Story 级完整关系，0 条也是明确事实；通过率只对“已执行”用例计算，不推导整体测试结论。
    const passRate = quality.testExecuted > 0
      ? Math.round(quality.testPassed * 100 / quality.testExecuted) + '%'
      : '无数据';
    // Bug 当前只覆盖 zt_bug.story。存在关联 Bug 时展示真实事实；为 0 时继续显示“无数据”，
    // 避免在尚未合并银行定制 demand 直连 Bug 关系前误报“无 Bug”。
    const hasStoryBugFacts = quality.bugTotal > 0;
    const bugStatus = hasStoryBugFacts
      ? (quality.bugOpen > 0 ? ('未关闭 ' + quality.bugOpen + ' / 总 ' + quality.bugTotal) : ('已关闭 / 总 ' + quality.bugTotal))
      : '无数据';

    return Object.assign({
      key: item.key || ('demand:' + item.demandId + ':window:' + item.windowId),
      id: item.id || ('REQ-' + item.demandId),
      code: item.code || item.id || '',
      demandId: Number(item.demandId || 0),
      kind: item.kind || 'demand',
      title: item.title || '-',
      mainSystem: vfMeaningful(item.system) || vfMeaningful(item.product) || '',
      supportSystems: '',
      relation: mapRelation(item.role),
      pri: item.pri || 'P2',
      iteration: item.windowName || '',
      iterationId: String(item.windowId || ''),
      valueStreamStage: item.valueStreamStage || '',
      reason: reasons.join('；') || '-',
      nextStep: item.nextAction || '查看详情',
      currentBlock: item.isBlocked ? '阻塞' : '',
      blockedFlag: !!item.isBlocked,
      testCase: quality.testTotal,
      totalCase: quality.testTotal,
      executedCase: quality.testExecuted,
      passRate: passRate,
      bugs: hasStoryBugFacts ? quality.bugOpen : null,
      // severeOpen 是真实统计，但当前版本跟进旧 UI 会把 severe>0 直接判“阻塞”；
      // 后端尚未定义“严重 Bug = 发布阻塞”，所以不把它接入旧 severe 字段，避免业务推断。
      severe: null,
      severeOpenFacts: hasStoryBugFacts ? quality.severeOpen : null,
      bugStatus: bugStatus,
      acceptable: '无数据',
      poNext: item.nextAction || '',
      planTestDone: '',
      actualTestDone: '',
      dev: joinFacts(devNames),
      test: joinFacts(testNames),
      accept: '',
      acceptOwner: item.currentHandler || '',
      acceptPlanDate: '',
      acceptConclusion: '无数据',
      targetDate: (backendWindows.find(function (w) { return String(w.id) === String(item.windowId); }) || {}).releaseDate || '',
      devPlan: '',
      testPlan: '',
      acceptPlan: '',
      devHours: (item.stories || []).reduce(function (sum, s) { return sum + Number(s.estimate || 0); }, 0),
      testHours: 0,
      storyPoints: 0,
      releaseWindow: item.windowName || '',
      project: joinFacts(projectNames),
      execution: joinFacts(executionNames),
      projectId: projectIDs.length === 1 ? projectIDs[0] : '',
      executionId: executionIDs.length === 1 ? executionIDs[0] : '',
      primaryUrl: item.primaryUrl || '',
      pendingApprovalStages: Array.isArray(item.pendingApprovalStages) ? item.pendingApprovalStages.slice() : [],
      pendingApprovalReviewers: item.pendingApprovalReviewers && typeof item.pendingApprovalReviewers === 'object'
        ? item.pendingApprovalReviewers
        : {},
      devRequirements: stories
    }, presentation);
  }

  // 当前阶段只认后端 9 阶段，不再让旧字符串 includes() 把“测试中”误判成“开发”。
  window.getCurrentStage = function getBackendVersionStage(row) {
    const stage = text(row && row.valueStreamStage);
    return valueStageLabels[stage] || '未知阶段';
  };

  // 版本跟进所有滑期/逾期判断必须使用真实本地日期，禁止继续消费旧原型基准日。
  window.vfToday = function getBackendVersionToday() {
    return todayYmd();
  };

  function applyWindows() {
    try {
      vfPrimaryVersions.length = 0;
      vfHistoryVersions.length = 0;
      vfVersions.length = 0;
      if (typeof scheduleIterationDefinitions !== 'undefined' && Array.isArray(scheduleIterationDefinitions)) {
        scheduleIterationDefinitions.length = 0;
      }
    } catch (e) {}

    const today = todayYmd();
    backendWindows.forEach(function (w) {
      const id = String(w.id || '');
      if (!id) return;
      const releaseDate = text(w.releaseDate);
      const startDate = text(w.startDate);
      const planTestDone = text(w.planTestDone);
      const testDone = text(w.testDone);
      const acceptDone = text(w.acceptDone);
      // R2 修订（增量 B）：历史/只读严格消费后端 w.isHistory 字段。
      // releaseDate 仅用于 ETA/逾期展示，不再决定历史/只读分类（避免双真源漂移）。
      const isHistory = w.isHistory === true;
      const status = startDate && startDate > today ? '未开始' : '进行中';
      const row = {
        id: id,
        name: w.name || ('窗口 #' + id),
        cardName: w.name || ('窗口 #' + id),
        range: startDate && releaseDate ? (startDate + ' ~ ' + releaseDate) : (releaseDate || startDate || '-'),
        status: status,
        eta: releaseDate,
        releaseDate: releaseDate,
        planTestDone: planTestDone,
        testDone: testDone,
        acceptDone: acceptDone,
        isHistory: isHistory,
        readOnlyReason: text(w.readOnlyReason) || ''
      };
      try {
        vfVersions.push(row);
        // 历史窗口判定严格消费后端 isHistory；不再用 releaseDate<today。
        if (isHistory) vfHistoryVersions.push(row); else vfPrimaryVersions.push(row);
        if (typeof scheduleIterationDefinitions !== 'undefined' && Array.isArray(scheduleIterationDefinitions)) {
          scheduleIterationDefinitions.push({
            key: id, id: id, label: row.name, name: row.name, range: row.range,
            start: startDate, end: releaseDate, online: releaseDate, releaseDate: releaseDate,
            planTestDone: planTestDone, testDone: testDone, acceptDone: acceptDone
          });
        }
      } catch (e) {}
    });
    try {
      vfPrimaryVersions.sort(function (a, b) { return text(a.eta).localeCompare(text(b.eta)); });
      vfHistoryVersions.sort(function (a, b) { return text(b.eta).localeCompare(text(a.eta)); });
      const current = String(currentVfIteration || '');
      if (!vfVersions.some(function (v) { return v.id === current; })) {
        currentVfIteration = vfPrimaryVersions[0] ? vfPrimaryVersions[0].id : (vfHistoryVersions[0] ? vfHistoryVersions[0].id : '');
      }
    } catch (e) {}
    updateReadOnlyBanner();
  }

  function clearBackendVersionFollow() {
    backendWindows = [];
    backendItems = [];
    backendStageCountsByWindow = [];
    loaded = false;
    applyWindows();
    if (typeof window.renderVersionFollow === 'function') window.renderVersionFollow();
  }

  async function loadBackendVersionFollow() {
    // 所有入口共用同一个 in-flight Promise。旧代码的 force=true 会并发启动第二次请求，
    // 造成后返回的数据覆盖先返回的数据；统一 single-flight 后只允许串行刷新。
    if (loading) return loading;
    loading = (async function () {
      let json;
      try {
        json = await window.apiFetch('/workbench/api/version-follow');
      } catch (e) {
        clearBackendVersionFollow();
        return false;
      }
      if (!json || json.success !== true || !Array.isArray(json.windows) || !Array.isArray(json.items)) {
        clearBackendVersionFollow();
        return false;
      }
      backendWindows = json.windows.slice();
      backendItems = json.items.slice();
      backendStageCountsByWindow = Array.isArray(json.stageCountsByWindow) ? json.stageCountsByWindow.slice() : [];
      loaded = true;
      applyWindows();
      if (typeof window.renderVersionFollow === 'function') window.renderVersionFollow();
      return true;
    })();
    try {
      return await loading;
    } finally {
      loading = null;
    }
  }

  window.loadVersionFollowBackendData = loadBackendVersionFollow;
  // 兼容原来的 loadVersionWindows 调用点，但真源已经统一成 version-follow 聚合 API。
  window.loadVersionWindows = function () { return loadBackendVersionFollow(); };
  window.getVersionFollowRows = function () {
    // Phase4 旧 closure 仍可能有一个已经发出的 /schedule/windows 请求在途。
    // 每次 canonical UI 取行时都先重放后端聚合窗口，确保旧请求即便晚返回也不能重新成为窗口真源。
    applyWindows();
    if (!loaded) return [];
    return backendItems.map(mapItem);
  };
  // V1.2 R2.1 增量 A：九阶段 count 取自 resp.stageCountsByWindow[currentWindowId]。
  window.getVfStageCountsForCurrentWindow = function () {
    const wid = String(currentVfIteration || '');
    if (!wid || !backendStageCountsByWindow.length) return null;
    const hit = backendStageCountsByWindow.find(function (s) {
      return String(s.windowId || s.WindowID || '') === wid;
    });
    if (!hit) return { windowId: wid, counts: {}, total: 0 };
    return {
      windowId: wid,
      counts: hit.counts || {},
      total: Number(hit.total || 0)
    };
  };

  // bridge 动态加载时 DOMContentLoaded 已发生，立即拉取一次；失败时保持空态，不回退 Mock/旧缓存。
  loadBackendVersionFollow();
})();

// BUG-R5B-003: 提升到顶层闭包。原定义在版本后端 IIFE 内部 (po-version.js:1887)，而
// selectVfIteration (po-version.js:1542) 是顶层函数 —— 调用点 ReferenceError。
// 依赖项：ensureVfHistoryBanner (DOM lookup) 和 getCurrentVfVersion (顶层 @ 125)
// 都不依赖 IIFE 内部状态，因此可整体提升。
function ensureVfHistoryBanner() {
  return document.getElementById('vfHistoryBanner');
}

function updateReadOnlyBanner() {
  const banner = ensureVfHistoryBanner();
  const cur = getCurrentVfVersion();
  const isHistory = !!(cur && cur.isHistory === true);
  if (banner) {
    if (isHistory) {
      banner.textContent = (cur.readOnlyReason || '该版本窗口已发布,仅支持历史查看');
      banner.style.display = '';
    } else {
      banner.style.display = 'none';
    }
  }
  document.querySelectorAll('#page-versionFollow .write-action').forEach(function (el) {
    if (isHistory) { el.disabled = true; el.classList.add('is-disabled'); }
    else { el.disabled = false; el.classList.remove('is-disabled'); }
  });
}
