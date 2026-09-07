/* 任务详情页交互脚本（CRCBWorkbench / workbench 模块，additive）
 * 复用：po-core.js 的 viewInZentao(id,'task') 做「去禅道」；设计令牌由 task-detail.css 提供。
 * 自包含：toast / modal / confirm 不依赖其他 SPA 脚本，避免破坏其他功能。
 */
(function () {
  'use strict';

  var TASK_ID = (function () {
    var urlParams = new URLSearchParams(window.location.search);
    if (urlParams.get('id')) return parseInt(urlParams.get('id'), 10);
    var m = String(location.pathname || '').match(/\/workbench\/task\/(\d+)/);
    return m ? parseInt(m[1], 10) : 88320;
  })();

  var TASK_TYPE_OPTIONS = [
    ['devel', '开发'], ['design', '设计'], ['test', '测试'], ['study', '研究'],
    ['discuss', '讨论'], ['ui', '界面'], ['affair', '事务'], ['misc', '其他'],
    ['request', '需求分析'], ['review', '评审'], ['codereview', '代码评审'],
    ['Oreview', '投产评审'], ['testCase', '用例编写'], ['smokeTesting', '冒烟测试'],
    ['OLtest', '投产验证'], ['OLDebug', '投产调试'], ['Online', '投产上线'],
    ['Train', '培训'], ['meeting', '会议']
  ];

  function $(id) { return document.getElementById(id); }
  function esc(s) {
    // R1-B1 单源收敛: 委托 shared/workbench-utils.js (null → '' 语义不变)。
    if (window.WBUtils && window.WBUtils.escapeHtml) return WBUtils.escapeHtml(s); return String(s == null ? "" : s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  }
  function fmtNum(n) {
    if (n == null || isNaN(n)) return '0';
    return (Math.round(Number(n) * 100) / 100).toString();
  }
  function todayStr() {
    var d = new Date();
    return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0');
  }
  function csrf() {
    var m = document.querySelector('meta[name="csrf-token"]');
    return m ? m.content : '';
  }

  // ── 请求 ──────────────────────────────────────────────
  function apiGet(url) {
    return fetch(url, { headers: { 'Accept': 'application/json' } }).then(function (r) {
      return r.json().then(function (j) { return { ok: r.ok, status: r.status, body: j }; });
    });
  }
  function apiSend(url, method, payload) {
    return fetch(url, {
      method: method,
      headers: { 'Content-Type': 'application/json', 'Accept': 'application/json', 'X-CSRF-Token': csrf() },
      body: JSON.stringify(payload || {})
    }).then(function (r) {
      return r.json().then(function (j) { return { ok: r.ok, status: r.status, body: j }; });
    });
  }

  // ── Toast ─────────────────────────────────────────────
  function toast(msg, type) {
    var c = $('tdToastContainer');
    if (!c) {
      c = document.createElement('div');
      c.id = 'tdToastContainer';
      c.className = 'td-toast-container';
      document.body.appendChild(c);
    }
    var t = document.createElement('div');
    t.className = 'td-toast ' + (type || 'info');
    t.textContent = msg;
    c.appendChild(t);
    setTimeout(function () {
      t.style.opacity = '0';
      t.style.transition = 'opacity .3s';
      setTimeout(function () { if (t.parentNode) t.parentNode.removeChild(t); }, 320);
    }, 2600);
  }

  // ── Modal ─────────────────────────────────────────────
  function openModal(title, bodyHTML, footerHTML) {
    $('tdModalTitle').innerHTML = title;
    $('tdModalBody').innerHTML = bodyHTML;
    $('tdModalFooter').innerHTML = footerHTML || '';
    $('tdModalOverlay').classList.add('show');
    $('tdModal').classList.add('show');
  }
  function closeModal() {
    $('tdModalOverlay').classList.remove('show');
    $('tdModal').classList.remove('show');
  }
  window.tdCloseModal = closeModal;
  window.tdCloseModalOnOverlay = function (e) { if (e.target === $('tdModalOverlay')) closeModal(); };

  function confirmBox(title, bodyHTML, onYes, yesText) {
    $('tdConfirmTitle').innerHTML = title;
    $('tdConfirmBody').innerHTML = bodyHTML;
    $('tdConfirmFooter').innerHTML =
      '<button type="button" class="action-btn" onclick="tdCloseConfirm()">取消</button>' +
      '<button type="button" class="action-btn danger" id="tdConfirmYes">' + (yesText || '确认') + '</button>';
    $('tdConfirmOverlay').classList.add('show');
    $('tdConfirm').classList.add('show');
    $('tdConfirmYes').onclick = function () {
      tdCloseConfirm();
      onYes();
    };
  }
  window.tdCloseConfirm = function () {
    $('tdConfirmOverlay').classList.remove('show');
    $('tdConfirm').classList.remove('show');
  };
  window.tdCloseConfirmOnOverlay = function (e) { if (e.target === $('tdConfirmOverlay')) tdCloseConfirm(); };

  // ui-consistency.mdc §9.3：弹窗必须支持 Escape 关闭（PRD §12）。
  document.addEventListener('keydown', function (e) {
    if (e.key !== 'Escape' && e.keyCode !== 27) return;
    if ($('tdModal') && $('tdModal').classList.contains('show')) closeModal();
    if ($('tdConfirm') && $('tdConfirm').classList.contains('show')) tdCloseConfirm();
  });

  // ── 主操作判定 ───────────────────────────────────────
  function mainAction(status) {
    switch (status) {
      case 'wait': return { label: '开始', action: 'start', cls: 'primary' };
      case 'doing': return { label: '完成任务', action: 'finish', cls: 'primary' };
      case 'pause': return { label: '重新开始', action: 'resume', cls: 'primary' };
      case 'done': return { label: '关闭', action: 'close', cls: 'primary' };
      case 'closed': return { label: '激活', action: 'activate', cls: 'primary' };
      case 'cancel': return { label: '激活', action: 'activate', cls: 'primary' };
    }
    return null;
  }
  function moreItems(status) {
    var items = [];
    if (status === 'doing') items.push({ label: '暂停', action: 'pause' });
    if (status !== 'closed' && status !== 'cancel') items.push({ label: '取消任务', action: 'cancel', danger: true });
    items.push({ label: '编辑任务', action: 'edit' });
    return items;
  }

  function doAction(action, note) {
    apiSend('/workbench/api/tasks/' + TASK_ID + '/action', 'PUT', { action: action, note: note || '' })
      .then(function (res) {
        if (res.ok && res.body && res.body.success) {
          toast('操作成功', 'success');
          load();
        } else {
          toast((res.body && res.body.message) || '操作失败', 'error');
        }
      })
      .catch(function () { toast('网络错误，请重试', 'error'); });
  }

  // ── 渲染 ─────────────────────────────────────────────
  function statusBadgeClass(status) {
    switch (status) {
      case 'doing': return 'is-doing';
      case 'pause': return 'is-pause';
      case 'done': return 'is-success';
      case 'closed': return 'is-closed';
      case 'cancel': return 'is-cancel';
      default: return 'is-muted';
    }
  }

  function render(data) {
    var r = data.task;
    var t = r.task;
    var root = $('task-detail-root');
    var can = r.canOperate;

    var ma = mainAction(t.status);
    var actionsHTML = '';
    actionsHTML += '<button type="button" class="action-btn" onclick="tdGoZentao()"><i class="fas fa-external-link-alt"></i> 去禅道</button>';
    if (can) {
      actionsHTML += '<button type="button" class="action-btn" onclick="tdOpenAssign()">指派</button>';
      actionsHTML += '<button type="button" class="action-btn" onclick="tdOpenEffort()">记录工时</button>';
      if (ma) actionsHTML += '<button type="button" class="action-btn ' + ma.cls + '" onclick="tdMainAction()">' + (ma.action === 'finish' ? '<i class="fas fa-check"></i> ' : '') + ma.label + '</button>';
      var mi = moreItems(t.status);
      if (mi.length) {
        var menu = mi.map(function (it) {
          return '<button data-action="' + it.action + '"' + (it.danger ? ' data-danger="1"' : '') + (can ? '' : ' disabled') + '>' + it.label + '</button>';
        }).join('');
        actionsHTML += '<div class="td-more-wrap"><button type="button" class="action-btn icon" onclick="tdToggleMore(event)"><i class="fas fa-ellipsis-h"></i></button>' +
          '<div class="td-more-menu" id="tdMoreMenu">' + menu + '</div></div>';
      }
    } else {
      actionsHTML += '<span class="td-viewonly" style="color:var(--color-text-secondary);font-size:12px;align-self:center">仅查看（无操作权限）</span>';
    }

    // 摘要条
    var overdueHTML = '';
    if (t.overdue) overdueHTML = '<span class="td-summary-v danger">逾期 ' + t.overdueDays + ' 天</span>';
    else overdueHTML = '<span class="td-summary-v">' + (t.deadline || '—') + '</span>';

    var summaryHTML =
      '<div class="td-summary-item"><div class="td-summary-k">任务进度</div><div class="td-progress-line"><div class="td-bar"><i style="width:' + t.progress + '%"></i></div><span class="td-summary-v">' + t.progress + '%</span></div></div>' +
      '<div class="td-summary-item"><div class="td-summary-k">最初预计</div><div class="td-summary-v">' + fmtNum(t.estimate) + ' h</div></div>' +
      '<div class="td-summary-item"><div class="td-summary-k">已消耗</div><div class="td-summary-v">' + fmtNum(t.consumed) + ' h</div></div>' +
      '<div class="td-summary-item"><div class="td-summary-k">剩余</div><div class="td-summary-v">' + fmtNum(t.left) + ' h</div></div>' +
      '<div class="td-summary-item"><div class="td-summary-k">实际开始</div><div class="td-summary-v">' + (t.realStarted || '—') + '</div></div>' +
      '<div class="td-summary-item"><div class="td-summary-k">截止日期</div>' + overdueHTML + '</div>';

    // 左侧主内容
    var mainHTML = '';

    // 描述
    mainHTML += section('任务描述', '当前任务需要完成什么', '',
      '<div class="td-desc">' + (t.desc ? esc(t.desc) : '<span class="td-empty">暂无任务描述</span>') + '</div>');

    // 关联研发需求
    if (r.relatedStory) {
      var st = r.relatedStory;
      var storyInner =
        '<div class="td-story-head"><span class="td-task-id">' + st.id + '</span>' +
        '<span class="td-story-title" onclick="tdOpenStory(' + st.id + ')">' + esc(st.title) + '</span>' +
        '<span class="status-badge ' + statusBadgeClass(st.status) + '">' + esc(st.statusLabel || st.status) + '</span></div>' +
        '<div class="td-story-grid">' +
        '<div class="td-story-cell"><div class="td-story-label">需求描述</div><div class="td-story-text">' + (st.desc ? esc(st.desc) : '暂无') + '</div></div>' +
        '<div class="td-story-cell"><div class="td-story-label">验收标准</div><div class="td-story-text">' + (st.verify ? esc(st.verify) : '暂无') + '</div></div>' +
        '</div>';
      if (st.attachments && st.attachments.length) {
        storyInner += '<div class="td-story-foot">附件：' + st.attachments.map(function (a) { return esc(a); }).join('、') + '　<a onclick="tdOpenStory(' + st.id + ')">查看研发需求附件 →</a></div>';
      }
      mainHTML += section('相关研发需求', '任务来源与验收上下文', '', '<div class="td-story-box">' + storyInner + '</div>');
    }

    // 需求与发布协同
    if (r.releases && r.releases.length) {
      var rel = r.releases;
      var relHead = '<div class="td-release-head"><b>关联发布</b><span>共 ' + rel.length + ' 条</span><a onclick="tdGoZentao()">在禅道查看完整发布链路 →</a></div>';
      var relRows = rel.map(function (x) {
        return '<tr><td>' + esc(x.name) + '</td><td>' + (x.buildName ? esc(x.buildName) : '—') + '</td><td>' + (x.date || '—') + '</td><td><span class="status-badge is-info">' + esc(x.status) + '</span></td></tr>';
      }).join('');
      var relTable = '<div class="table-wrap"><table class="tbl"><thead><tr><th>发布名称</th><th>构建</th><th>计划/实际日期</th><th>状态</th></tr></thead><tbody>' + relRows + '</tbody></table></div>';
      mainHTML += section('需求与发布协同', '跨系统关联发布（best-effort）', '', relHead + relTable);
    }

    // 相关对象（子任务 / Bug / 来源 Bug）
    var linkHTML = '';
    if (r.children && r.children.length) {
      linkHTML += linkBlock('子任务', r.children, function (x) { return tdGoTask(x.id); });
    }
    if (r.bugs && r.bugs.length) {
      linkHTML += linkBlock('关联 Bug', r.bugs, function (x) { return tdGoZentaoObj('bug', x.id); });
    }
    if (r.sourceBug) {
      linkHTML += linkBlock('来源 Bug', [r.sourceBug], function (x) { return tdGoZentaoObj('bug', x.id); });
    }
    if (linkHTML) mainHTML += section('相关对象', '任务直接关联的研发对象', '', linkHTML);

    // 历史记录 —— 统一 Timeline（第 3 参是 count，第 4 参才是 body；勿把 HTML 放进 count，会被 esc）
    mainHTML += section('历史记录', '来源：禅道 zt_action + zt_history', '',
      '<div id="wb-task-detail-history" data-wb-history-timeline data-object-type="task" data-object-id="' + esc(TASK_ID) + '"></div>');

    // 右侧信息
    var sideHTML = '';
    var infoRows = [
      ['所属执行', t.executionName ? esc(t.executionName) : (t.execution ? ('#' + t.execution) : '—')],
      ['所属项目', t.projectName ? esc(t.projectName) : (t.project ? ('#' + t.project) : '—')],
      ['所属模块', t.moduleName ? esc(t.moduleName) : (t.module ? ('#' + t.module) : '—')],
      ['任务类型', t.typeLabel],
      ['状态', '<span class="status-badge ' + statusBadgeClass(t.status) + '">' + esc(t.statusLabel) + '</span>'],
      ['优先级', t.priority],
      ['指派给', t.assignedName ? (esc(t.assignedName) + (t.assignedTo ? ' (' + esc(t.assignedTo) + ')' : '')) : '—'],
      ['来源 Bug', r.sourceBug ? ('#' + r.sourceBug.id) : '—'],
      ['关键词', t.keywords ? esc(t.keywords) : '—'],
      ['抄送', t.mailto ? esc(t.mailto) : '—']
    ];
    sideHTML += '<div class="td-panel"><div class="td-side-section" style="border-top:0"><div class="td-side-title">基本信息</div>' +
      infoList(infoRows) + '</div>';

    // 任务的一生
    var life = [
      ['创建人', t.openedByName ? (esc(t.openedByName) + (t.openedBy ? ' (' + esc(t.openedBy) + ')' : '')) : '—', t.openedDate],
      ['最后编辑', t.lastEditedName ? esc(t.lastEditedName) : '—', t.lastEditedDate],
      ['完成人', t.finishedName ? esc(t.finishedName) : '—', t.finishedDate],
      ['取消人', t.canceledName ? esc(t.canceledName) : '—', t.canceledDate],
      ['关闭人', t.closedName ? esc(t.closedName) : '—', t.closedDate],
      ['关闭原因', t.closedReason ? esc(t.closedReason) : '—', '']
    ];
    sideHTML += '<div class="td-side-section"><div class="td-side-title">任务的一生</div>' +
      life.map(function (x) {
        return '<div class="td-collab-row"><span>' + esc(x[0]) + '：' + x[1] + '</span><span style="color:var(--color-text-secondary)">' + (x[2] || '') + '</span></div>';
      }).join('') + '</div>';

    // 工时信息
    sideHTML += '<div class="td-side-section"><div class="td-side-title">工时信息</div>' +
      '<div class="td-hours">' +
      '<div class="td-hour"><div class="k">预计</div><div class="v">' + fmtNum(t.estimate) + 'h</div></div>' +
      '<div class="td-hour"><div class="k">消耗</div><div class="v">' + fmtNum(t.consumed) + 'h</div></div>' +
      '<div class="td-hour"><div class="k">剩余</div><div class="v">' + fmtNum(t.left) + 'h</div></div>' +
      '</div>' +
      '<div class="td-date-list">' +
      '<div class="k">预计开始</div><div class="v">—</div>' +
      '<div class="k">实际开始</div><div class="v">' + (t.realStarted || '—') + '</div>' +
      '<div class="k">截止日期</div><div class="v"' + (t.overdue ? ' style="color:var(--color-danger);font-weight:600"' : '') + '>' + (t.deadline || '—') + (t.overdue ? ' · 已逾期' : '') + '</div>' +
      '</div></div>';

    // 协作成员（仅多人任务）
    if (r.collaborators && r.collaborators.length) {
      var collab = r.collaborators.map(function (c) {
        return '<div class="td-collab-row"><span>' + esc(c.name) + (c.account ? ' (' + esc(c.account) + ')' : '') + '</span>' +
          '<span style="color:var(--color-text-secondary)">预计 ' + fmtNum(c.estimate) + 'h · 消耗 ' + fmtNum(c.consumed) + 'h · 剩余 ' + fmtNum(c.left) + 'h</span></div>';
      }).join('');
      sideHTML += '<div class="td-side-section"><div class="td-side-title">协作成员</div>' + collab + '</div>';
    }

    // 组装
    root.innerHTML =
      '<div class="td-panel td-title-panel">' +
        '<div class="td-title-row">' +
          '<button class="td-back" onclick="history.back()" title="返回"><i class="fas fa-arrow-left"></i></button>' +
          '<div class="td-head-main">' +
            '<div class="td-title-line">' +
              '<span class="td-task-id">#' + t.id + '</span>' +
              '<h1 class="td-title">' + esc(t.name) + '</h1>' +
              '<span class="status-badge ' + statusBadgeClass(t.status) + '">' + esc(t.statusLabel) + '</span>' +
              '<span class="status-badge is-info">' + esc(t.priority) + '</span>' +
              (t.overdue ? '<span class="status-badge is-danger">已逾期</span>' : '') +
            '</div>' +
            '<div class="td-meta">' +
              (t.executionName ? '<span>执行：<b>' + esc(t.executionName) + '</b></span>' : '') +
              (r.relatedStory ? '<span>研发需求：<a onclick="tdOpenStory(' + r.relatedStory.id + ')">' + esc(r.relatedStory.title) + '</a></span>' : '') +
              '<span>负责人：<b>' + (t.assignedName ? esc(t.assignedName) : '—') + (t.assignedTo ? ' (' + esc(t.assignedTo) + ')' : '') + '</b></span>' +
              '<span>类型：' + esc(t.typeLabel) + '</span>' +
            '</div>' +
          '</div>' +
          '<div class="td-actions">' + actionsHTML + '</div>' +
        '</div>' +
        summaryHTML +
      '</div>' +
      '<div class="td-grid">' +
        '<div class="td-main-col">' + mainHTML + '</div>' +
        '<div class="td-side-col">' + sideHTML + '</div>' +
      '</div>';

    // 异步 DOM 写入后挂载统一 Timeline（bootAll 时节点尚不存在）
    if (window.WBHistoryTimeline) {
      var histEl = document.getElementById('wb-task-detail-history');
      if (histEl) window.WBHistoryTimeline.mount(histEl);
    }

    // 绑定更多菜单
    var menu = $('tdMoreMenu');
    if (menu) {
      menu.querySelectorAll('button').forEach(function (b) {
        b.addEventListener('click', function () {
          var a = b.getAttribute('data-action');
          menu.classList.remove('show');
          if (a === 'edit') { tdOpenEdit(); return; }
          if (b.getAttribute('data-danger')) {
            confirmBox('确认' + b.textContent, '该操作将' + b.textContent + '任务 #' + TASK_ID + '，是否继续？', function () { doAction(a); },
              '确认' + b.textContent);
          } else {
            doAction(a);
          }
        });
      });
    }
  }

  function section(title, sub, count, body) {
    return '<div class="td-panel td-section"><div class="td-section-head"><span class="td-section-title">' + esc(title) + '</span>' +
      (sub ? '<span class="td-section-sub">' + esc(sub) + '</span>' : '') +
      (count ? '<span class="td-section-count">' + esc(count) + '</span>' : '') + '</div>' + body + '</div>';
  }
  function infoList(rows) {
    return '<div class="td-info-list">' + rows.map(function (r) {
      return '<div class="td-info-k">' + esc(r[0]) + '</div><div class="td-info-v">' + r[1] + '</div>';
    }).join('') + '</div>';
  }
  function linkBlock(title, items, onClick) {
    var rows = items.map(function (x) {
      return '<div class="td-link-row"><span class="td-link-id">#' + x.id + '</span>' +
        '<span class="td-link-title" data-id="' + x.id + '">' + esc(x.title || ('#' + x.id)) + '</span>' +
        '<span class="td-link-state">' + esc(x.status || '') + '</span></div>';
    }).join('');
    var html = '<div class="td-link-list">' + rows + '</div>';
    // 绑定点击
    setTimeout(function () {
      var list = document.querySelectorAll('.td-link-title[data-id]');
      list.forEach(function (el) {
        el.onclick = function () { onClick(parseInt(el.getAttribute('data-id'), 10)); };
      });
    }, 0);
    return section(title, '', items.length + ' 条', html);
  }

  // ── 跳转 ─────────────────────────────────────────────
  window.tdGoZentao = function () { if (window.viewInZentao) viewInZentao(String(TASK_ID), 'task'); };
  window.tdOpenStory = function (id) { if (window.viewInZentao) viewInZentao(String(id), 'story'); };
  // 任务详情页内跳转同样是 /workbench/task/:id 的导航边界，必须消费 po-core 的唯一
  // normalize 边界，而不是把原始入参直接拼进 URL（TASK- 前缀 / NaN / 任意串都会拼出非法路由）。
  // po-core.js 在本页先于 task-detail.js 加载，normalizeTaskId 是脚本顶层声明。
  window.tdGoTask = function (id) {
    var nid = (typeof normalizeTaskId === 'function')
      ? normalizeTaskId(id)
      : (/^\d+$/.test(String(id == null ? '' : id)) ? String(id) : null);
    if (!nid) return;
    location.href = '/workbench/task/' + encodeURIComponent(nid);
  };
  window.tdGoZentaoObj = function (type, id) { if (window.viewInZentao) viewInZentao(String(id), type); };

  // ── 主操作 / 更多 ───────────────────────────────────
  window.tdMainAction = function () {
    var ma = mainAction(window.__tdStatus);
    if (ma) {
      if (ma.action === 'close' || ma.action === 'activate') {
        confirmBox('确认' + ma.label, '是否' + ma.label + '任务 #' + TASK_ID + '？', function () { doAction(ma.action); }, ma.label);
      } else {
        doAction(ma.action);
      }
    }
  };
  window.tdToggleMore = function (e) {
    e.stopPropagation();
    var m = $('tdMoreMenu');
    if (m) m.classList.toggle('show');
  };
  document.addEventListener('click', function () { var m = $('tdMoreMenu'); if (m) m.classList.remove('show'); });

  // ── 记录工时 ─────────────────────────────────────────
  window.tdOpenEffort = function () {
    if (!window.__tdCan) { toast('当前账号无记录工时权限', 'error'); return; }
    openModal('记录工时', '<div class="td-loading"><i class="fas fa-spinner fa-spin"></i> 加载部门/执行选项…</div>', '');
    apiGet('/api/mobile/efforts/today-draft?source=task&objectId=' + TASK_ID).then(function (res) {
      var depts = (res.body && res.body.departments) || [];
      var execs = (res.body && res.body.executions) || [];
      if (!depts.length) {
        openModal('记录工时', '<div class="td-form-error">无法获取可记录工时的部门，可能无记录权限。</div>',
          '<button type="button" class="action-btn" onclick="tdCloseModal()">关闭</button>');
        return;
      }
      var deptOpts = depts.map(function (d) { return '<option value="' + d.value + '">' + esc(d.label) + '</option>'; }).join('');
      var execOpts = '<option value="0">不指定</option>' + execs.map(function (d) { return '<option value="' + d.value + '">' + esc(d.label) + '</option>'; }).join('');
      var body =
        '<div class="td-form-row"><label>工作日期</label><input type="date" id="effDate" value="' + todayStr() + '"></div>' +
        '<div class="td-form-row"><label>工作内容 / 日志 <span style="color:var(--color-danger)">*</span></label><textarea id="effWork" placeholder="本次完成的工作内容"></textarea></div>' +
        '<div class="td-form-inline">' +
          '<div class="td-form-row"><label>消耗工时(h) <span style="color:var(--color-danger)">*</span></label><input type="number" id="effConsumed" min="0.1" step="0.1" value="1"></div>' +
          '<div class="td-form-row"><label>剩余工时(h) <span style="color:var(--color-danger)">*</span></label><input type="number" id="effLeft" min="0" step="0.1" value="0"></div>' +
        '</div>' +
        '<div class="td-form-inline">' +
          '<div class="td-form-row"><label>部门 <span style="color:var(--color-danger)">*</span></label><select id="effDept">' + deptOpts + '</select></div>' +
          '<div class="td-form-row"><label>执行</label><select id="effExec">' + execOpts + '</select></div>' +
        '</div>' +
        '<div class="td-form-error" id="effErr"></div>';
      var footer =
        '<button type="button" class="action-btn" onclick="tdCloseModal()">取消</button>' +
        '<button type="button" class="action-btn primary" onclick="tdSubmitEffort()">提交工时</button>';
      openModal('记录工时', body, footer);
    }).catch(function () {
      openModal('记录工时', '<div class="td-form-error">加载失败，请重试。</div>',
        '<button type="button" class="action-btn" onclick="tdCloseModal()">关闭</button>');
    });
  };
  window.tdSubmitEffort = function () {
    var work = $('effWork').value.trim();
    var consumed = parseFloat($('effConsumed').value);
    var left = parseFloat($('effLeft').value);
    var dept = parseInt($('effDept').value, 10);
    var exec = parseInt($('effExec').value, 10) || 0;
    var date = $('effDate').value;
    var err = $('effErr');
    if (!work) { err.textContent = '请填写工作内容'; return; }
    if (!(consumed > 0)) { err.textContent = '消耗工时必须大于 0'; return; }
    if (isNaN(left) || left < 0) { err.textContent = '剩余工时不能为负'; return; }
    if (!dept) { err.textContent = '请选择部门'; return; }
    var payload = {
      date: date,
      entries: [{
        objectType: 'task', objectId: TASK_ID, work: work,
        consumed: consumed, left: left, departmentId: dept, executionId: exec
      }]
    };
    apiSend('/api/mobile/efforts/batch', 'POST', payload).then(function (res) {
      if (res.ok && res.body && res.body.success) {
        closeModal();
        toast('工时记录成功', 'success');
        load();
      } else {
        err.textContent = (res.body && res.body.message) || '提交失败';
      }
    }).catch(function () { err.textContent = '网络错误，请重试'; });
  };

  // ── 指派 ─────────────────────────────────────────────
  window.tdOpenAssign = function () {
    var body =
      '<div class="td-form-row"><label>当前指派人</label><div class="td-info-v">' + esc(window.__tdAssigned || '—') + '</div></div>' +
      '<div class="td-form-row"><label>新指派人账号 <span style="color:var(--color-danger)">*</span></label><input type="text" id="assignTo" placeholder="输入禅道账号，如 zhouml"></div>' +
      '<div class="td-form-row"><label>备注</label><textarea id="assignNote" placeholder="可选"></textarea></div>' +
      '<div class="td-form-hint">人员候选范围沿用当前项目 / 执行成员；此处按账号指派。</div>' +
      '<div class="td-form-error" id="assignErr"></div>';
    var footer =
      '<button type="button" class="action-btn" onclick="tdCloseModal()">取消</button>' +
      '<button type="button" class="action-btn primary" onclick="tdSubmitAssign()">确认指派</button>';
    openModal('指派任务', body, footer);
  };
  window.tdSubmitAssign = function () {
    var to = $('assignTo').value.trim();
    var err = $('assignErr');
    if (!to) { err.textContent = '请填写新指派人账号'; return; }
    apiSend('/workbench/api/tasks/' + TASK_ID + '/action', 'PUT', { action: 'assign', assignedTo: to, note: $('assignNote').value.trim() })
      .then(function (res) {
        if (res.ok && res.body && res.body.success) {
          closeModal(); toast('任务已指派给 ' + to, 'success'); load();
        } else { err.textContent = (res.body && res.body.message) || '指派失败'; }
      }).catch(function () { err.textContent = '网络错误，请重试'; });
  };

  // ── 编辑任务 ─────────────────────────────────────────
  window.tdOpenEdit = function () {
    var t = window.__tdTask;
    if (!t) return;
    var typeOpts = TASK_TYPE_OPTIONS.map(function (o) {
      return '<option value="' + o[0] + '"' + (o[0] === t.type ? ' selected' : '') + '>' + o[1] + '</option>';
    }).join('');
    var priOpts = [1, 2, 3, 4].map(function (p) {
      return '<option value="' + p + '"' + (('P' + p) === t.priority ? ' selected' : '') + '>P' + p + '</option>';
    }).join('');
    var body =
      '<div class="td-form-row"><label>任务名称</label><input type="text" id="edName" value="' + esc(t.name) + '"></div>' +
      '<div class="td-form-row"><label>描述</label><textarea id="edDesc">' + esc(t.desc) + '</textarea></div>' +
      '<div class="td-form-inline">' +
        '<div class="td-form-row"><label>类型</label><select id="edType">' + typeOpts + '</select></div>' +
        '<div class="td-form-row"><label>优先级</label><select id="edPri">' + priOpts + '</select></div>' +
      '</div>' +
      '<div class="td-form-inline">' +
        '<div class="td-form-row"><label>预计工时(h)</label><input type="number" id="edEst" min="0" step="0.5" value="' + fmtNum(t.estimate) + '"></div>' +
        '<div class="td-form-row"><label>剩余工时(h)</label><input type="number" id="edLeft" min="0" step="0.5" value="' + fmtNum(t.left) + '"></div>' +
      '</div>' +
      '<div class="td-form-row"><label>截止日期</label><input type="date" id="edDeadline" value="' + (t.deadline || '') + '"></div>' +
      '<div class="td-form-inline">' +
        '<div class="td-form-row"><label>关键词</label><input type="text" id="edKw" value="' + esc(t.keywords) + '"></div>' +
        '<div class="td-form-row"><label>抄送</label><input type="text" id="edMail" value="' + esc(t.mailto) + '"></div>' +
      '</div>' +
      '<div class="td-form-error" id="edErr"></div>';
    var footer =
      '<button type="button" class="action-btn" onclick="tdCloseModal()">取消</button>' +
      '<button type="button" class="action-btn primary" onclick="tdSubmitEdit()">保存</button>';
    openModal('编辑任务', body, footer);
  };
  window.tdSubmitEdit = function () {
    var est = parseFloat($('edEst').value);
    var left = parseFloat($('edLeft').value);
    var pri = parseInt($('edPri').value, 10);
    var err = $('edErr');
    if (isNaN(est) || est < 0) { err.textContent = '预计工时不合法'; return; }
    if (isNaN(left) || left < 0) { err.textContent = '剩余工时不合法'; return; }
    var payload = {
      action: 'edit',
      name: $('edName').value.trim(),
      desc: $('edDesc').value,
      type: $('edType').value,
      pri: pri,
      estimate: est,
      left: left,
      deadline: $('edDeadline').value,
      keywords: $('edKw').value.trim(),
      mailto: $('edMail').value.trim()
    };
    apiSend('/workbench/api/tasks/' + TASK_ID + '/action', 'PUT', payload).then(function (res) {
      if (res.ok && res.body && res.body.success) { closeModal(); toast('任务已更新', 'success'); load(); }
      else { err.textContent = (res.body && res.body.message) || '保存失败'; }
    }).catch(function () { err.textContent = '网络错误，请重试'; });
  };

  // ── 加载 ─────────────────────────────────────────────
  function load() {
    apiGet('/workbench/api/tasks/' + TASK_ID + '/detail').then(function (res) {
      if (res.ok && res.body && res.body.success) {
        var r = res.body.task;
        if (!r || !r.task) {
          $('task-detail-root').innerHTML = '<div class="td-empty">任务详情加载失败。</div>';
          return;
        }
        window.__tdStatus = r.task.status;
        window.__tdCan = r.canOperate;
        window.__tdTask = r.task;
        window.__tdAssigned = (r.task.assignedName ? r.task.assignedName : '') + (r.task.assignedTo ? ' (' + r.task.assignedTo + ')' : '');
        try {
          render(res.body);
        } catch (e) {
          console.error('task detail render failed', e);
          $('task-detail-root').innerHTML = '<div class="td-empty">任务详情渲染失败。</div>';
        }
      } else if (res.status === 403) {
        $('task-detail-root').innerHTML = '<div class="td-empty">无权限查看该任务。</div>';
      } else if (res.status === 404) {
        $('task-detail-root').innerHTML = '<div class="td-empty">任务不存在或已删除。</div>';
      } else {
        $('task-detail-root').innerHTML = '<div class="td-empty">任务详情加载失败。</div>';
      }
    }).catch(function () {
      $('task-detail-root').innerHTML = '<div class="td-empty">网络错误，请稍后重试。</div>';
    });
  }

  // 初始化用户信息
  (function initUser() {
    var acc = document.querySelector('meta[name="wb-user-account"]');
    var name = document.querySelector('meta[name="wb-user-name"]');
    var an = $('userAvatar'), un = $('userName');
    var nm = name ? name.content : '';
    if (un) un.textContent = nm || (acc ? acc.content : '—');
    if (an) an.textContent = nm ? nm.trim().charAt(0) : '?';
  })();

  if (TASK_ID) load();
  else $('task-detail-root').innerHTML = '<div class="td-empty">任务 ID 无效。</div>';
})();
