/* =============================================================================
 * 文件: web/static/workbench/po/po-issue-risk.js
 * 模块: PO 工作台 · 问题风险列表
 * 职责: 按 V8 原型渲染类型/关系/闭环/逾期快捷条件、项目状态关键词筛选、当前条件 Chip、高密度列表。
 * 数据: GET /workbench/api/issue-risks（后端 SQL 收敛，禁止前端二次过滤业务口径）
 * ============================================================================= */

var issueRiskFilter = {
  kind: 'issue',
  relation: 'myAction',
  loop: 'open',
  overdue: false,
  project: 0,
  status: '',
  keyword: '',
  page: 1,
  pageSize: 20
};
var issueRiskHighlightId = '';
var issueRiskLastResp = null;
var issueRiskLoading = false;

function issueRiskEsc(value) {
  // R1-B1 单源收敛: 委托 shared/workbench-utils.js。
  return WBUtils.escapeHtml(value);
}

function issueRiskSevLabel(sev) {
  var n = Number(sev) || 0;
  if (n === 1) return '致命';
  if (n === 2) return '严重';
  return '一般';
}

function issueRiskSevClass(sev) {
  var n = Number(sev) || 0;
  if (n === 1) return 'fatal';
  if (n === 2) return 'serious';
  return 'normal';
}

function issueRiskStatusOptions(kind) {
  if (kind === 'risk') {
    return [
      { v: '', l: '状态：全部' },
      { v: 'active', l: '激活' },
      { v: 'tracked', l: '跟踪中' },
      { v: 'hangup', l: '已挂起' },
      { v: 'closed', l: '已关闭' },
      { v: 'canceled', l: '已取消' }
    ];
  }
  return [
    { v: '', l: '状态：全部' },
    { v: 'unconfirmed', l: '未处理' },
    { v: 'active', l: '未处理(激活)' },
    { v: 'wait', l: '未处理(待处理)' },
    { v: 'doing', l: '处理中' },
    { v: 'confirmed', l: '处理中(已确认)' },
    { v: 'resolved', l: '已解决' },
    { v: 'closed', l: '已关闭' },
    { v: 'canceled', l: '已取消' }
  ];
}

function issueRiskBuildQuery() {
  var f = issueRiskFilter;
  var qs = [
    'kind=' + encodeURIComponent(f.kind || 'issue'),
    'relation=' + encodeURIComponent(f.relation || 'myAction'),
    'loop=' + encodeURIComponent(f.loop || 'open'),
    'page=' + (f.page || 1),
    'pageSize=' + (f.pageSize || 20)
  ];
  if (f.overdue) qs.push('overdue=true');
  if (f.project) qs.push('project=' + encodeURIComponent(f.project));
  if (f.status) qs.push('status=' + encodeURIComponent(f.status));
  if (f.keyword) qs.push('keyword=' + encodeURIComponent(f.keyword));
  return qs.join('&');
}

async function fetchIssueRiskList() {
  if (typeof apiFetch !== 'function') return null;
  var path = '/workbench/api/issue-risks?' + issueRiskBuildQuery();
  var json = await apiFetch(path);
  if (!json || !json.success) return null;
  return json;
}

function renderIssueRisk() {
  var host = document.getElementById('issueRiskBody');
  var detail = document.getElementById('issueRiskDetailBody');
  if (detail) detail.style.display = 'none';
  if (host) host.style.display = '';
  if (!host) return;

  var f = issueRiskFilter;
  var kindCounts = (issueRiskLastResp && issueRiskLastResp.kindCounts) || {};
  var relationCounts = (issueRiskLastResp && issueRiskLastResp.relationCounts) || {};
  var loopCounts = (issueRiskLastResp && issueRiskLastResp.loopCounts) || {};
  var overdueCount = (issueRiskLastResp && issueRiskLastResp.overdueCount) || 0;
  var projects = (issueRiskLastResp && issueRiskLastResp.projects) || [];

  var hdrActions = document.querySelector('#page-issueRisk .page-hdr-actions');
  if (hdrActions) {
    hdrActions.innerHTML =
      '<button type="button" class="action-btn primary" onclick="issueRiskRegister()"><i class="fas fa-plus" style="margin-right:4px"></i>登记</button>';
  }

  function qBtn(group, value, label, count) {
    var active = f[group] === value ? ' active' : '';
    return (
      '<button type="button" class="ir-qbtn' +
      active +
      '" data-ir-group="' +
      group +
      '" data-ir-value="' +
      value +
      '" onclick="setIssueRiskQuick(&quot;' +
      group +
      '&quot;,&quot;' +
      value +
      '&quot;)">' +
      label +
      ' <b>' +
      Number(count || 0) +
      '</b></button>'
    );
  }

  var quickHtml =
    '<div class="ir-compact-groups">' +
    '<div class="ir-qgroup">' +
    qBtn('kind', 'issue', '问题', kindCounts.issue) +
    qBtn('kind', 'risk', '风险', kindCounts.risk) +
    '</div><div class="ir-sep"></div><div class="ir-qgroup">' +
    qBtn('relation', 'myAction', '待我处理', relationCounts.myAction) +
    qBtn('relation', 'mySubmit', '我提出的', relationCounts.mySubmit) +
    qBtn('relation', 'allRelated', '全部相关', relationCounts.allRelated) +
    '</div><div class="ir-sep"></div><div class="ir-qgroup">' +
    qBtn('loop', 'open', '未闭环', loopCounts.open) +
    qBtn('loop', 'closed', '已闭环', loopCounts.closed) +
    qBtn('loop', 'all', '全部', loopCounts.all) +
    '</div>' +
    '<button type="button" class="ir-qbtn ir-warn' + (f.overdue ? ' active' : '') + '" onclick="setIssueRiskOverdue(!issueRiskFilter.overdue)">逾期 <b>' +
    Number(overdueCount || 0) +
    '</b></button>' +
    '</div>';

  var statusTabsHost = document.getElementById('issueRiskStatusTabs');
  if (statusTabsHost) statusTabsHost.innerHTML = quickHtml;

  var projectOpts =
    '<option value="0"' + (!f.project ? ' selected' : '') + '>项目：全部</option>' +
    projects
      .map(function (p) {
        return (
          '<option value="' +
          issueRiskEsc(p.id) +
          '"' +
          (String(f.project) === String(p.id) ? ' selected' : '') +
          '>' +
          issueRiskEsc(p.name) +
          '</option>'
        );
      })
      .join('');

  var statusOpts = issueRiskStatusOptions(f.kind)
    .map(function (o) {
      return (
        '<option value="' +
        issueRiskEsc(o.v) +
        '"' +
        (f.status === o.v ? ' selected' : '') +
        '>' +
        issueRiskEsc(o.l) +
        '</option>'
      );
    })
    .join('');

  var filterHtml = WBFilterShell.primary(
    '<div class="ir-filterbar">' +
    '<select class="ir-select" id="issueRiskProject" onchange="setIssueRiskProject(this.value)" aria-label="项目筛选">' +
    projectOpts +
    '</select>' +
    '<select class="ir-select" id="issueRiskStatus" onchange="setIssueRiskStatus(this.value)" aria-label="状态筛选">' +
    statusOpts +
    '</select>' +
    '<input id="issueRiskSearch" type="search" class="ir-search" placeholder="搜编号 / 标题 / 项目 / 人员" value="' +
    issueRiskEsc(f.keyword || '') +
    '" oninput="onIssueRiskSearchInput(this)" aria-label="搜索问题风险" />' +
    '<button type="button" class="action-btn" onclick="toggleIssueRiskMore()">更多筛选</button>' +
    '<button type="button" class="action-btn ir-clear-btn" onclick="resetIssueRiskFilters()"><i class="fas fa-rotate-left" style="margin-right:4px"></i>重置</button>' +
    '</div>') +
    WBFilterShell.more('<div class="ir-filter-more" id="issueRiskFilterMore" style="display:none">' +
    '<span class="subtle">更多筛选（严重程度 / 优先级 / 时间）本期先保留入口，与列表共用关键词与状态下拉。</span>' +
    '</div>');

  host.innerHTML =
    filterHtml +
    WBFilterShell.active('', 'ir-result-meta po-list-result-bar', ' id="issueRiskResultMeta"') +
    WBListShell.render({
      scrollClass: 'ir-table-wrap',
      content: '<div id="issueRiskTableWrap"></div>',
      pagerId: 'issueRiskPagination'
    });

  loadAndRenderIssueRisk();
}

async function loadAndRenderIssueRisk() {
  if (issueRiskLoading) return;
  issueRiskLoading = true;
  try {
    var json = await fetchIssueRiskList();
    if (!json) {
      issueRiskLastResp = { total: 0, items: [], kindCounts: {}, relationCounts: {}, loopCounts: {}, overdueCount: 0, projects: [] };
    } else {
      issueRiskLastResp = json;
      // 同步缓存给看板/首页：仅在全部相关+未闭环时刷新 issueRiskItems
      if (issueRiskFilter.relation === 'allRelated' && issueRiskFilter.loop === 'open' && !issueRiskFilter.overdue) {
        try {
          if (typeof issueRiskItems !== 'undefined' && Array.isArray(issueRiskItems) && typeof replaceArray === 'function') {
            // 不在此处覆盖全量缓存；看板仍走 loadWorkbenchDatasets 的 limit 调用
          }
        } catch (e) {}
      }
    }
    renderIssueRiskChrome();
    renderIssueRiskTable(issueRiskLastResp.items || []);
  } catch (e) {
    console.warn('issue-risk load failed', e);
    var wrap = document.getElementById('issueRiskTableWrap');
    if (wrap) wrap.innerHTML = '<div class="ir-empty"><i class="fas fa-inbox"></i>加载失败，请稍后重试</div>';
  } finally {
    issueRiskLoading = false;
  }
}

function renderIssueRiskChrome() {
  var host = document.getElementById('issueRiskBody');
  if (!host || !issueRiskLastResp) return;
  var f = issueRiskFilter;
  var kindCounts = issueRiskLastResp.kindCounts || {};
  var relationCounts = issueRiskLastResp.relationCounts || {};
  var loopCounts = issueRiskLastResp.loopCounts || {};
  var overdueCount = issueRiskLastResp.overdueCount || 0;

  var chromeRoot = document.getElementById('issueRiskStatusTabs') || host;
  chromeRoot.querySelectorAll('.ir-qbtn[data-ir-group]').forEach(function (btn) {
    var g = btn.getAttribute('data-ir-group');
    var v = btn.getAttribute('data-ir-value');
    btn.classList.toggle('active', f[g] === v);
    var count = 0;
    if (g === 'kind') count = kindCounts[v] || 0;
    else if (g === 'relation') count = relationCounts[v] || 0;
    else if (g === 'loop') count = loopCounts[v] || 0;
    var b = btn.querySelector('b');
    if (b) b.textContent = String(count);
  });
  var overdueBtn = chromeRoot.querySelector('.ir-qbtn.ir-warn');
  if (overdueBtn) {
    overdueBtn.classList.toggle('active', !!f.overdue);
    var ob = overdueBtn.querySelector('b');
    if (ob) ob.textContent = String(overdueCount || 0);
  }

  // 刷新项目下拉（保留选中）
  var projectSel = document.getElementById('issueRiskProject');
  if (projectSel && Array.isArray(issueRiskLastResp.projects)) {
    var cur = String(f.project || '0');
    var html =
      '<option value="0">项目：全部</option>' +
      issueRiskLastResp.projects
        .map(function (p) {
          return (
            '<option value="' +
            issueRiskEsc(p.id) +
            '"' +
            (cur === String(p.id) ? ' selected' : '') +
            '>' +
            issueRiskEsc(p.name) +
            '</option>'
          );
        })
        .join('');
    projectSel.innerHTML = html;
  }

  renderIssueRiskChips();
}

function renderIssueRiskChips() {
  var meta = document.getElementById('issueRiskResultMeta');
  if (!meta) return;
  var f = issueRiskFilter;
  var total = (issueRiskLastResp && issueRiskLastResp.total) || 0;
  var kindLabel = f.kind === 'risk' ? '风险' : '问题';
  var relationLabel = { myAction: '待我处理', mySubmit: '我提出的', allRelated: '全部相关' };
  var loopLabel = { open: '未闭环', closed: '已闭环', all: '全部' };

  var chips = [{ label: '类型：' + kindLabel, clear: '', fixed: true }];
  if (f.relation && f.relation !== 'allRelated') {
    chips.push({ label: '我的关系：' + (relationLabel[f.relation] || f.relation), clear: "clearIssueRiskCondition('relation')" });
  }
  if (f.loop && f.loop !== 'all') {
    chips.push({ label: '结果：' + (loopLabel[f.loop] || f.loop), clear: "clearIssueRiskCondition('loop')" });
  }
  if (f.project) {
    var pname = f.project;
    var projects = (issueRiskLastResp && issueRiskLastResp.projects) || [];
    projects.forEach(function (p) {
      if (String(p.id) === String(f.project)) pname = p.name;
    });
    chips.push({ label: '项目：' + pname, clear: "clearIssueRiskCondition('project')" });
  }
  if (f.status) {
    var sl = f.status;
    issueRiskStatusOptions(f.kind).forEach(function (o) {
      if (o.v === f.status) sl = o.l.replace(/^状态：/, '');
    });
    chips.push({ label: '状态：' + sl, clear: "clearIssueRiskCondition('status')" });
  }
  if (f.overdue) {
    chips.push({ label: '异常：逾期', clear: "clearIssueRiskCondition('overdue')" });
  }
  if (f.keyword) {
    chips.push({ label: '关键词：' + f.keyword, clear: "clearIssueRiskCondition('keyword')" });
  }

  meta.innerHTML =
    '<span class="ir-result-total">共 <strong>' +
    total +
    '</strong> 条</span><span class="dw-condition-label">当前条件</span><span class="ir-condition-chips">' +
    chips
      .map(function (c) {
        if (c.fixed || !c.clear) {
          return '<span class="dw-condition-chip ir-chip-fixed">' + issueRiskEsc(c.label) + '</span>';
        }
        return (
          '<button type="button" class="dw-condition-chip" onclick="' +
          c.clear +
          '">' +
          issueRiskEsc(c.label) +
          ' <span class="dw-condition-x" aria-hidden="true">×</span></button>'
        );
      })
      .join('') +
    '<button type="button" class="dw-condition-clear" onclick="clearIssueRiskExtraFilters()">清空附加条件</button></span>' +
    '<button type="button" class="action-btn icon-action-btn ir-col-config" title="自定义列" onclick="openColumnConfigModal(\'issueRisk\')"><i class="fas fa-gear"></i></button>';
}


function setIssueRiskQuick(group, value) {
  if (group === 'kind') {
    issueRiskFilter.kind = value === 'risk' ? 'risk' : 'issue';
    issueRiskFilter.status = '';
  } else if (group === 'relation') {
    issueRiskFilter.relation = value;
  } else if (group === 'loop') {
    issueRiskFilter.loop = value;
    if (value !== 'open') issueRiskFilter.overdue = false;
  }
  issueRiskFilter.page = 1;
  renderIssueRisk();
}

function setIssueRiskOverdue(on) {
  issueRiskFilter.overdue = !!on;
  if (issueRiskFilter.overdue) issueRiskFilter.loop = 'open';
  issueRiskFilter.page = 1;
  loadAndRenderIssueRisk();
}

function setIssueRiskProject(value) {
  issueRiskFilter.project = Number(value) || 0;
  issueRiskFilter.page = 1;
  loadAndRenderIssueRisk();
}

function setIssueRiskStatus(value) {
  issueRiskFilter.status = value || '';
  issueRiskFilter.page = 1;
  loadAndRenderIssueRisk();
}

function onIssueRiskSearchInput(el) {
  issueRiskFilter.keyword = el && el.value ? String(el.value).trim() : '';
  issueRiskFilter.page = 1;
  if (window._irSearchTimer) clearTimeout(window._irSearchTimer);
  window._irSearchTimer = setTimeout(function () {
    loadAndRenderIssueRisk();
  }, 280);
}

function toggleIssueRiskMore() {
  var el = document.getElementById('issueRiskFilterMore');
  if (!el) return;
  el.style.display = el.style.display === 'none' ? 'block' : 'none';
}

function resetIssueRiskFilters() {
  issueRiskFilter.project = 0;
  issueRiskFilter.status = '';
  issueRiskFilter.keyword = '';
  issueRiskFilter.overdue = false;
  issueRiskFilter.page = 1;
  var search = document.getElementById('issueRiskSearch');
  if (search) search.value = '';
  loadAndRenderIssueRisk();
}

function clearIssueRiskExtraFilters() {
  resetIssueRiskFilters();
}

function clearIssueRiskCondition(type) {
  issueRiskFilter.page = 1;
  if (type === 'relation') issueRiskFilter.relation = 'allRelated';
  else if (type === 'loop') issueRiskFilter.loop = 'all';
  else if (type === 'project') issueRiskFilter.project = 0;
  else if (type === 'status') issueRiskFilter.status = '';
  else if (type === 'overdue') issueRiskFilter.overdue = false;
  else if (type === 'keyword') {
    issueRiskFilter.keyword = '';
    var search = document.getElementById('issueRiskSearch');
    if (search) search.value = '';
  }
  if (type === 'relation' || type === 'loop') renderIssueRisk();
  else loadAndRenderIssueRisk();
}

function issueRiskRegister() {
  if (issueRiskFilter.kind === 'risk') {
    if (typeof showToast === 'function') showToast('风险登记本期暂未开放，请在禅道登记');
    return;
  }
  if (typeof openKanbanCreateModal === 'function') openKanbanCreateModal('issue');
  else if (typeof showToast === 'function') showToast('登记入口未就绪');
}

// 兼容旧闭包 / 看板侧调用
function applyIssueRiskFilters() {
  loadAndRenderIssueRisk();
}
function setIssueRiskKindFilter(key) {
  setIssueRiskQuick('kind', key);
}
function setIssueRiskStatusFilter() {
  /* 旧状态 Tab 已移除 */
}
function clearLegacyIssueRiskTeamStub() {
  /* 已去除承建团队筛选 */
}
function clearLegacyIssueRiskAgileStub() {}
