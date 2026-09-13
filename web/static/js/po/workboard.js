/* PO 工作看板 - 指标/任务抽屉与初始化。 */
(function (WB) {
  "use strict";
  var $ = WB.$;
  var esc = WB.esc;
  var state = WB.state;
  var showToast = WB.showToast;
  var onErr = WB.onErr;
  var ownerBadge = WB.ownerBadge;
  var ensureTeamgroup = WB.ensureTeamgroup;
  var renderOwnerChips = WB.renderOwnerChips;
  var selectOwner = WB.selectOwner;
  var switchMode = WB.switchMode;
  var MAX_OWNERS = WB.MAX_OWNERS;
  var typeTag = WB.typeTag;
  var objectTypeBadge = WB.objectTypeBadge;
  var draggedTask = null;
  var pendingDoneTask = null;
  var activeTaskFlag = "";

  function getMode() { return WB.mode(); }
  function renderDemandMatrix(tree) { return WB.renderDemandMatrix(tree); }
  function applyDemandFilters() { return WB.applyDemandFilters(); }
  function toggleFlag(flag) { return WB.toggleFlag(flag); }

  function loadMetrics() {
    $("metricsGrid").innerHTML = '<div class="metric-empty">加载中…</div>';
    fetch("/board/group/metrics?teamgroupId=" + state.teamgroup, { method: "GET" })
      .then(function (r) { return r.json(); })
      .then(function (p) {
        if (!p || p.success !== true) { throw new Error("payload"); }
        $("metricsGroupName").textContent = p.groupName || "";
        $("metricsGrid").innerHTML = (p.hasGroup && p.metrics && p.metrics.length) ? "" : '<div class="metric-empty">请选择具体敏捷小组查看效能指标</div>';
        if (p.hasGroup && p.metrics && p.metrics.length) { renderMetrics(p.metrics); }
      })
      .catch(function () { $("metricsGrid").innerHTML = '<div class="metric-empty">效能指标加载失败</div>'; });
  }
  function renderMetrics(metrics) {
    // 看板暂时只展示交付节奏类指标；质量门禁/缺陷/上线延期指标保留在接口与指标中心。
    var visibleMetricKeys = { delivery: true, implement: true, overIteration: true, unscheduled: true };
    metrics = (metrics || []).filter(function (m) { return visibleMetricKeys[m.key]; });
    var details = {
      "交付周期": "端到端流动速度：需求从进入研发到完成交付用了多久。", "实施周期": "研发实施效率：进入实施后到完成研发交付用了多久。",
      "超预迭代周期占比": "交付可预测性：有多少工作超过预期迭代节奏。", "超2周未排期": "需求入口健康度：识别长期未进入时间盒的需求积压。",
      "质量门禁通过率": "内建质量：研发产物能否稳定通过质量门禁。", "缺陷关闭率": "质量收敛：已发现缺陷是否及时形成闭环。",
      "缺陷响应效率": "质量响应速度：缺陷是否得到及时确认和处理。", "上线延期数": "交付兑现：已承诺上线的事项是否按窗口完成。"
    };
    $("metricsGrid").innerHTML = metrics.map(function (m) {
      var cls = "team-metric" + (m.state ? " is-" + m.state : ""), trendHtml = m.trend ? '<span class="metric-trend ' + (m.state || "") + '">' + esc(m.trend) + "</span>" : "";
      return '<div class="' + cls + '" data-metric-name="' + esc(m.name) + '"><div class="metric-top"><span class="metric-name">' + esc(m.name) + '</span><span class="metric-value">' + esc(m.value === "-" ? "—" : m.value) + "</span></div>" +
        '<div class="metric-meta"><span class="metric-target">' + (m.target ? "目标 " + esc(m.target) : "") + "</span>" + trendHtml + "</div></div>";
    }).join("");
    $("metricsGrid").querySelectorAll(".team-metric").forEach(function (el) {
      el.addEventListener("click", function () { var n = el.dataset.metricName || ""; showToast(n + "：" + (details[n] || "")); });
    });
  }

  /* ---------- loaders ---------- */
  function loadDemand() {
    var host = $("demandGroups"); host.innerHTML = '<div class="demand-empty">加载中…</div>';
    var params = new URLSearchParams(); if (state.teamgroup) { params.set("teamgroupId", state.teamgroup); }
    fetch("/board/demand/items?" + params.toString(), { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) { WB.setTeams(payload.teamgroups); WB.renderTeamChips(); }
        renderDemandMatrix(payload.tree || []); renderDemandOwners(payload.tree || []);
        if (state.pendingFocus) { focusStoryRow(state.pendingFocus); state.pendingFocus = 0; }
      })
      .catch(function () { onErr("demandGroups"); });
  }
  function seqOwners(tree) {
    var owners = {}, stack = (tree || []).slice().reverse();
    while (stack.length) {
      var n = stack.pop(); if (n && n.owner) { owners[n.owner] = (owners[n.owner] || 0) + 1; }
      var cs = (n && n.children) || []; for (var i = cs.length - 1; i >= 0; i--) { stack.push(cs[i]); }
    }
    return Object.keys(owners).map(function (acc) { return { value: acc, display: acc, count: owners[acc] }; }).sort(function (a, b) { return b.count - a.count; });
  }
  function renderDemandOwners(tree) {
    // 首位的"全部"chip 不再带 count 角标，避免被读成"41 个负责人"。
    // 总需求数已经在需求树/分页区有体现，这里只要"清空负责人筛选"的能力。
    var opts = [{ value: "", display: "", count: 0 }];
    opts = opts.concat(seqOwners(tree));
    state.owner = "";
    renderOwnerChips("demandOwners", opts, applyDemandFilters);
  }
  function focusStoryRow(storyId) {
    // 研需 displayId 为纯数字；兼容历史 RD- 前缀与 data-rd 直接存数字两种写法。
    var sid = String(storyId || "").replace(/^RD-/i, "");
    var row = document.querySelector('.demand-row[data-rd="' + sid + '"]') ||
      document.querySelector('.demand-row[data-rd="RD-' + sid + '"]');
    if (!row) { return; }
    var grp = row.closest(".biz-group");
    if (grp) { grp.classList.remove("is-collapsed"); }
    row.scrollIntoView({ block: "center", behavior: "smooth" });
    row.style.background = "#fffbea";
    setTimeout(function () { row.style.background = ""; }, 2000);
  }
  function loadTasks() {
    document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) { c.innerHTML = ""; });
    ensureTeamgroup();
    var params = new URLSearchParams();
    // 首次直达任务看板时可能还没有本地小组选择，交给服务端按当前账号选择首个可用小组。
    if (state.teamgroup) { params.set("teamgroupId", state.teamgroup); }
    if (state.storyFilter) { params.set("storyId", state.storyFilter); }
    if (state.owner) { params.set("ownerAccount", state.owner); }
    if (state.focus) { params.set("focus", state.focus); }
    fetch("/board/task/items?" + params.toString(), { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) {
          WB.setTeams(payload.teamgroups);
          if (payload.selectedTeamgroupId) {
            state.teamgroup = Number(payload.selectedTeamgroupId);
            WB.saveTeamgroup(state.teamgroup);
          }
          WB.renderTeamChips();
        }
        if (payload.owners) {
          // 与需求看板一致：首位"全部" chip 不带 count 角标，避免被读成"41 个负责人"。
          var opts = [{ value: "", display: "", count: 0 }];
          payload.owners.forEach(function (o) { opts.push({ value: o.account, display: o.display, count: o.count }); });
          renderOwnerChips("taskOwners", opts, function () { loadTasks(); });
        }
        renderTaskColumns(payload.columns || []);
        applyTaskFilters();
        bindTaskDrag();
        // 任务接口可能在首次进入时才确定默认小组，必须使用服务端返回的小组 ID。
        loadMetrics();
      })
      .catch(function () { onErr("taskBoard"); });
  }
  function renderTaskColumns(cols) {
    var colMap = {}; cols.forEach(function (c) { colMap[c.key] = c.items || []; });
    ["wait", "doing", "done"].forEach(function (key) {
      var colEl = document.querySelector('#taskBoard .task-col[data-col="' + key + '"]'); if (!colEl) { return; }
      var items = colMap[key] || []; colEl.querySelector(".task-col-body").innerHTML = items.map(taskCard).join("");
      colEl.querySelector(".k-count").textContent = items.length;
    });
    updateTaskStats();
  }
  function toggleTaskFlag(flag) {
    activeTaskFlag = activeTaskFlag === flag ? "" : flag;
    document.querySelectorAll("#taskStats .stat").forEach(function (el) {
      el.classList.toggle("active", el.dataset.flag === activeTaskFlag);
    });
    applyTaskFilters();
  }
  function applyTaskFilters() {
    document.querySelectorAll("#taskBoard .task-card").forEach(function (card) {
      var match = !activeTaskFlag || card.classList.contains(activeTaskFlag);
      card.classList.toggle("hidden", !match);
    });
    updateTaskStats();
  }
  function taskCard(task) {
    var cls = "task-card" + (task.blocked ? " blocked" : "") + (task.overdue && task.status !== "done" ? " overdue" : "") + (task.status === "done" ? " done" : "");
    var due = (task.overdue && task.status !== "done") ? '<span class="task-due over">已超期</span>' : (task.deadline ? "截止 " + esc(task.deadline) : "");
    var storyLink = task.storyId ? '<span class="task-link" data-back-rd="' + task.storyId + '" title="返回需求看板并定位研需">所属研需 ' + esc(task.storyTitle || String(task.storyId)) + "</span>" : "";
    var codeText = esc(task.displayId || task.id);
    var titleText = esc(task.title);
    var codeHtml = task.url ? '<a class="code task-code-link" href="' + esc(task.url) + '" target="_blank" rel="noopener noreferrer" title="在禅道打开任务详情">' + codeText + '</a>' : '<span class="code">' + codeText + '</span>';
    var taskDetailUrl = task.id ? ('/workbench/task/' + encodeURIComponent(task.id)) : (task.url || '');
    var titleHtml = taskDetailUrl ? '<a class="task-title task-title-link" href="' + esc(taskDetailUrl) + '" title="' + titleText + '">' + titleText + '</a>' : '<span class="task-title" title="' + titleText + '">' + titleText + '</span>';
    return '<div class="' + cls + '" draggable="true" data-task-id="' + esc(task.id) + '" data-task-status="' + esc(task.status || "wait") + '" data-task-title="' + titleText + '" data-task-owner="' + esc(task.owner || "") + '" data-task-owner-account="' + esc(task.ownerAccount || "") + '" data-owner="' + esc(task.owner || "") + '">' +
      '<div class="task-head">' + typeTag("task") + codeHtml + titleHtml + "</div>" +
      '<div class="task-meta">' + storyLink + (task.type ? "<span>" + esc(task.type) + "</span>" : "") + (task.blocked ? '<span style="color:var(--orange);font-weight:700">阻塞</span>' : "") + "</div>" +
      '<div class="task-footer"><span class="task-due">' + due + '</span><span class="assignee">' + (task.owner ? '<span class="mini-avatar">' + esc(task.owner.charAt(0)) + "</span>" + esc(task.owner) : "") + "</span></div></div>";
  }


  function nowForDatetimeLocal() {
    var d = new Date();
    d.setSeconds(0, 0);
    var pad = function (n) { return String(n).padStart(2, "0"); };
    return d.getFullYear() + "-" + pad(d.getMonth() + 1) + "-" + pad(d.getDate()) + "T" + pad(d.getHours()) + ":" + pad(d.getMinutes());
  }
  function datetimeLocalToServer(v) { return (v || "").replace("T", " ") + ((v || "").length === 16 ? ":00" : ""); }
  function taskFromCard(card) {
    return {
      id: card.dataset.taskId,
      status: card.dataset.taskStatus || "wait",
      title: card.dataset.taskTitle || "",
      owner: card.dataset.taskOwner || "",
      ownerAccount: card.dataset.taskOwnerAccount || ""
    };
  }
  function submitTaskStatus(task, target, extra) {
    extra = extra || {};
    var body = {
      status: target,
      teamgroupId: Number(state.teamgroup || 0),
      finishedBy: extra.finishedBy || "",
      finishedDate: extra.finishedDate || ""
    };
    var appFetch = window.appFetch || window.fetch;
    showToast("正在更新任务状态…");
    return appFetch("/board/tasks/" + encodeURIComponent(task.id) + "/status", {
      method: "PUT",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json", "Accept": "application/json" },
      body: JSON.stringify(body)
    }).then(function (r) {
      return r.json().catch(function () { return {}; }).then(function (p) {
        if (!r.ok || !p || p.success !== true) { throw new Error((p && p.message) || "任务状态更新失败"); }
        showToast("任务状态已更新");
        loadTasks();
      });
    }).catch(function (err) {
      showToast(err && err.message ? err.message : "任务状态更新失败");
      loadTasks();
    });
  }
  function openDoneModal(task) {
    pendingDoneTask = task;
    $("taskDoneTitle").textContent = (task.id ? "#" + task.id + " " : "") + (task.title || "任务");
    $("taskDoneFinishedBy").value = task.ownerAccount || "";
    $("taskDoneFinishedBy").placeholder = task.owner || "默认当前指派人";
    $("taskDoneFinishedDate").value = nowForDatetimeLocal();
    $("taskDoneModal").classList.remove("hidden");
  }
  function closeDoneModal() {
    pendingDoneTask = null;
    if ($("taskDoneModal")) { $("taskDoneModal").classList.add("hidden"); }
  }
  function confirmDoneModal() {
    if (!pendingDoneTask) { closeDoneModal(); return; }
    var task = pendingDoneTask;
    var finishedBy = $("taskDoneFinishedBy").value.trim();
    var finishedDate = datetimeLocalToServer($("taskDoneFinishedDate").value.trim());
    closeDoneModal();
    submitTaskStatus(task, "done", { finishedBy: finishedBy, finishedDate: finishedDate });
  }
  function bindTaskDrag() {
    var board = $("taskBoard");
    if (!board || board.dataset.dragBound === "1") { return; }
    board.dataset.dragBound = "1";
    board.addEventListener("dragstart", function (e) {
      var card = e.target.closest(".task-card");
      if (!card || !board.contains(card)) { return; }
      draggedTask = taskFromCard(card);
      card.classList.add("dragging");
      e.dataTransfer.effectAllowed = "move";
      e.dataTransfer.setData("text/plain", draggedTask.id || "");
    });
    board.addEventListener("dragend", function () {
      document.querySelectorAll("#taskBoard .dragging,#taskBoard .drag-over").forEach(function (el) { el.classList.remove("dragging", "drag-over"); });
      draggedTask = null;
    });
    board.addEventListener("dragover", function (e) {
      var body = e.target.closest(".task-col-body");
      if (!body || !draggedTask) { return; }
      e.preventDefault();
      e.dataTransfer.dropEffect = "move";
      body.classList.add("drag-over");
      var col = body.closest(".task-col"); if (col) { col.classList.add("drag-over"); }
    });
    board.addEventListener("dragleave", function (e) {
      var body = e.target.closest(".task-col-body");
      if (!body || body.contains(e.relatedTarget)) { return; }
      body.classList.remove("drag-over");
      var col = body.closest(".task-col"); if (col) { col.classList.remove("drag-over"); }
    });
    board.addEventListener("drop", function (e) {
      var body = e.target.closest(".task-col-body");
      if (!body || !draggedTask) { return; }
      e.preventDefault();
      document.querySelectorAll("#taskBoard .drag-over").forEach(function (el) { el.classList.remove("drag-over"); });
      var col = body.closest(".task-col");
      var target = col ? col.dataset.col : "";
      var task = draggedTask;
      draggedTask = null;
      if (!target || target === task.status || (target === "doing" && task.status === "pause")) { return; }
      if (target === "done") { openDoneModal(task); return; }
      submitTaskStatus(task, target);
    });
  }
  ["taskDoneCancel", "taskDoneCancelX"].forEach(function (id) { if ($(id)) { $(id).addEventListener("click", closeDoneModal); } });
  if ($("taskDoneConfirm")) { $("taskDoneConfirm").addEventListener("click", confirmDoneModal); }

  function updateTaskStats() {
    var blocked = 0, overdue = 0, visible = 0;
    document.querySelectorAll("#taskBoard .task-card").forEach(function (r) {
      if (r.classList.contains("hidden")) { return; }
      visible++; if (r.classList.contains("blocked")) { blocked++; } if (r.classList.contains("overdue")) { overdue++; }
    });
    $("taskStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("taskStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
    document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) {
      var empty = c.querySelector(".task-filter-empty");
      var hasVisible = Array.prototype.some.call(c.querySelectorAll(".task-card"), function (card) { return !card.classList.contains("hidden"); });
      if (!hasVisible && activeTaskFlag) {
        if (!empty) { c.insertAdjacentHTML("beforeend", '<div class="demand-empty task-filter-empty">当前条件下没有任务</div>'); }
      } else if (empty) {
        empty.remove();
      }
    });
  }

  /* ---------- 任务抽屉（方案1：右侧滑出，不离开需求看板） ---------- */
  var drawerStoryId = 0;
  function openTaskDrawer(storyId, label, storyUrl) {
    drawerStoryId = storyId;
    var drawer = $("taskDrawer"), backdrop = $("taskDrawerBackdrop");
    $("drawerTitle").textContent = label;
    $("drawerTagGroup").innerHTML = objectTypeBadge("story");
    $("drawerContext").innerHTML = "";
    $("drawerSummary").innerHTML = '<div class="drawer-stat"><div class="drawer-stat-label">加载中…</div><div class="drawer-stat-val" style="color:var(--t3)">—</div></div>';
    $("drawerTaskList").innerHTML = '<div class="drawer-loading">正在加载任务…</div>';
    if ($("drawerZtLink")) { $("drawerZtLink").href = storyUrl || "#"; }
    backdrop.classList.remove("hidden");
    drawer.classList.remove("hidden");
    requestAnimationFrame(function () { drawer.classList.add("open"); });
    fetchDrawerTasks(storyId);
  }
  function fetchDrawerTasks(storyId) {
    var url = "/board/task/items?storyId=" + encodeURIComponent(storyId) + "&pageSize=200";
    if (state.teamgroup) { url += "&teamgroupId=" + encodeURIComponent(state.teamgroup); }
    fetch(url, { method: "GET", credentials: "same-origin" })
      .then(function (r) { if (!r.ok) { throw new Error("http " + r.status); } return r.json(); })
      .then(function (p) { if (!p || p.success !== true) { throw new Error("payload"); } renderDrawerTasks(p); })
      .catch(function (err) {
        $("drawerTaskList").innerHTML = '<div class="drawer-empty">任务加载失败，请稍后重试<br><button type="button" class="action soft" style="margin-top:8px" onclick="fetchDrawerTasks(' + drawerStoryId + ')">重试</button></div>';
        $("drawerSummary").innerHTML = "";
        console.warn("task drawer fetch error", err);
      });
  }
  function renderDrawerTasks(payload) {
    var cols = payload.columns || [], summary = payload.summary || {};
    var all = []; cols.forEach(function (c) { (c.items || []).forEach(function (t) { all.push(t); }); });
    var total = all.length, doing = 0, wait = 0, done = 0;
    all.forEach(function (t) { if (t.status === "done") { done++; } else if (t.status === "doing" || t.status === "pause") { doing++; } else { wait++; } });
    $("drawerSummary").innerHTML =
      '<div class="drawer-stat"><div class="drawer-stat-label">关联任务</div><div class="drawer-stat-val">' + total + "</div></div>" +
      '<div class="drawer-stat"><div class="drawer-stat-label">进行中</div><div class="drawer-stat-val" style="color:var(--navy)">' + doing + "</div></div>" +
      '<div class="drawer-stat"><div class="drawer-stat-label">未开始</div><div class="drawer-stat-val" style="color:#64748b">' + wait + "</div></div>" +
      '<div class="drawer-stat"><div class="drawer-stat-label">已完成</div><div class="drawer-stat-val" style="color:var(--green)">' + done + "</div></div>";
    if (!all.length) { $("drawerTaskList").innerHTML = '<div class="drawer-empty">该研发需求下暂无执行任务</div>'; return; }
    // 排序：进行中 > 未开始 > 已完成（各组内按 id 升序）
    var order = { doing: 0, pause: 1, wait: 2, done: 3 };
    all.sort(function (a, b) { return (order[a.status] || 2) - (order[b.status] || 2) || a.id - b.id; });
    $("drawerTaskList").innerHTML = all.map(function (t) { return renderDrawerCard(t); }).join("");
  }
  function drawerTypeClass(typ) {
    if (typ === "devel" || typ === "develop") { return "devel"; }
    if (typ === "test") { return "test"; }
    if (typ === "design") { return "design"; }
    return "other";
  }
  function drawerTypeName(typ) {
    var m = { devel: "开发", develop: "开发", test: "测试", design: "设计", doc: "文档", deploy: "部署" };
    return m[typ] || typ || "任务";
  }
  function drawerStatusName(s) { return { doing: "进行中", pause: "阻塞", wait: "未开始", done: "已完成" }[s] || s; }
  function drawerPriClass(pri) { if (!pri) { return ""; } var n = parseInt(pri); return n <= 1 ? "p1" : n === 2 ? "p2" : "p3"; }
  function renderDrawerCard(t) {
    var cardCls = "drawer-task-card status-" + (t.status || "wait") + (t.blocked ? " is-blocked" : "") + (t.overdue ? " is-overdue" : "");
    var typeC = drawerTypeClass(t.type), typN = drawerTypeName(t.type);
    var priC = drawerPriClass(t.priority), priH = t.priority ? '<span class="drawer-task-pri ' + priC + '">' + esc(t.priority) + "</span>" : "";
    var stN = drawerStatusName(t.status);
    var avatar = t.owner ? '<span class="drawer-task-avatar">' + esc(t.owner.charAt(0)) + "</span>" : "";
    var dueH = "";
    if (t.deadline) { dueH = '<span class="drawer-task-due' + (t.overdue ? " overdue" : "") + '">' + esc(t.deadline) + (t.overdue ? " 已超期" : "") + "</span>"; }
    var linkH = t.url ? '<a href="' + esc(t.url) + '" target="_blank" rel="noopener noreferrer" class="drawer-task-ztlink">在禅道打开 ↗</a>' : "";
    return '<div class="' + cardCls + '">' +
      '<div class="drawer-task-top"><span class="drawer-task-code">' + esc(t.displayId || t.id) + "</span>" +
      '<span class="drawer-task-type ' + typeC + '">' + esc(typN) + "</span>" + priH +
      '<span class="drawer-task-status ' + esc(t.status || "wait") + '">' + esc(stN) + "</span></div>" +
      '<div class="drawer-task-name">' + (t.id ? ('<a class="drawer-task-name-link" href="/workbench/task/' + encodeURIComponent(t.id) + '" title="' + esc(t.title) + '">' + esc(t.title) + '</a>') : esc(t.title)) + "</div>" +
      '<div class="drawer-task-footer"><span class="drawer-task-assignee">' + avatar + esc(t.owner || "未指派") + "</span>" +
      dueH + linkH + "</div></div>";
  }
  function closeTaskDrawer() {
    var drawer = $("taskDrawer"), backdrop = $("taskDrawerBackdrop");
    drawer.classList.remove("open");
    backdrop.classList.add("hidden");
    setTimeout(function () { drawer.classList.add("hidden"); }, 260);
    drawerStoryId = 0;
  }
  ["drawerClose", "drawerCloseBtn", "taskDrawerBackdrop"].forEach(function (id) {
    if ($(id)) $(id).addEventListener("click", closeTaskDrawer);
  });
  document.addEventListener("keydown", function (e) { if (e.key === "Escape" && drawerStoryId) closeTaskDrawer(); });

  document.addEventListener("click", function (e) {
    var retry = e.target.closest("[data-retry]"); if (retry) { getMode() === "demand" ? loadDemand() : loadTasks(); return; }
    var flagBtn = e.target.closest("#demandStats [data-flag]"); if (flagBtn) { toggleFlag(flagBtn.dataset.flag); return; }
    var taskFlagBtn = e.target.closest("#taskStats [data-flag]"); if (taskFlagBtn) { toggleTaskFlag(taskFlagBtn.dataset.flag); return; }
    var open = e.target.closest("[data-open-tasks]");
    if (open) {
      var sid = Number(open.dataset.openTasks), lbl = open.dataset.rdLabel || "研需", storyUrl = open.dataset.storyUrl || "";
      openTaskDrawer(sid, lbl, storyUrl); return;
    }
    var demBtn = e.target.closest("[data-open-demand]");
    if (demBtn && window.DemandDetail && typeof window.DemandDetail.open === "function") {
      e.stopPropagation(); window.DemandDetail.open(demBtn.dataset.openDemand); return;
    }
    var toastBtn = e.target.closest("[data-toast]"); if (toastBtn) { showToast(toastBtn.dataset.toast); return; }
  });

  /* ---------- issue panel & init ---------- */
  function loadIssues() {
    if (window.WorkboardIssue && typeof window.WorkboardIssue.loadIssues === "function") {
      window.WorkboardIssue.loadIssues();
    }
  }

  WB.loadMetrics = loadMetrics;
  WB.loadDemand = loadDemand;
  WB.loadTasks = loadTasks;
  WB.fetchDrawerTasks = fetchDrawerTasks;
  WB.openTaskDrawer = openTaskDrawer;
  WB.closeTaskDrawer = closeTaskDrawer;
  WB.focusStoryRow = focusStoryRow;
  WB.seqOwners = seqOwners;
  WB.renderDemandOwners = renderDemandOwners;
  window.fetchDrawerTasks = function (id) { fetchDrawerTasks(id); };

  try { switchMode(getMode()); }
  catch (e) { console.error("[wb-debug] init", e && e.stack || e); }
})(window.PoWB = window.PoWB || {});
