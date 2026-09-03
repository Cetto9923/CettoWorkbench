/* PO 工作看板（需求/任务双视图，V1.3 紧凑效能快照）。数据来自真实 API，禁止 mock。 */
(function () {
  "use strict";

  var mode = location.pathname.indexOf("/board/task") >= 0 ? "task" : "demand";
  var state = { owner: "", flag: "", teamgroup: 0, storyFilter: 0, demandRows: [] };
  var teams = [];
  var toastTimer = null;

  function $(id) { return document.getElementById(id); }
  function esc(v) {
    return String(v == null ? "" : v)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  function showToast(msg) {
    var t = $("toast");
    if (!t) { return; }
    t.textContent = msg; t.classList.add("show");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { t.classList.remove("show"); }, 1500);
  }
  function onErr(hostId, retry) {
    var host = $(hostId);
    if (host) { host.innerHTML = '<div class="demand-empty">加载失败 · <button type="button" class="action soft" data-retry="1">重试</button></div>'; }
  }
  function dash(v) { return String(v || "").trim() || "—"; }
  function statusTone(s) {
    if (s === "blocked" || s === "refuse") { return "orange"; }
    if (s === "overdue") { return "red"; }
    return "";
  }
  function typeTag(kind) {
    if (kind === "demand") { return '<span class="type-tag type-biz">业务需求</span>'; }
    if (kind === "sub_demand") { return '<span class="type-tag type-child">子业务</span>'; }
    if (kind === "story") { return '<span class="type-tag type-rd">研发需求</span>'; }
    if (kind === "task") { return '<span class="type-tag type-task">任务</span>'; }
    return "";
  }
  function priTag(p) {
    if (!p) { return ""; }
    return '<span class="priority">' + esc(p) + "</span>";
  }

  /* ---------- mode switch ---------- */
  function switchMode(next) {
    mode = next;
    var demand = next === "demand";
    $("demandTab").classList.toggle("active", demand);
    $("taskTab").classList.toggle("active", !demand);
    $("demandBoard").classList.toggle("hidden", !demand);
    $("taskBoard").classList.toggle("hidden", demand);
    $("demandStats").classList.toggle("hidden", !demand);
    $("taskStats").classList.toggle("hidden", demand);
    $("demandOwners").classList.toggle("hidden", !demand);
    $("taskOwners").classList.toggle("hidden", demand);
    $("demandActions").classList.toggle("hidden", !demand);
    $("taskActions").classList.toggle("hidden", demand);
    $("demandRoleBanner").classList.toggle("hidden", !demand);
    $("taskRoleBanner").classList.toggle("hidden", demand);
    $("ownerTitle").textContent = demand ? "PO / 需求负责人" : "任务负责人";
    $("ownerHint").textContent = demand
      ? "仅显示实际拥有业务需求 / 独立研发需求的人；开发人员不会混进来"
      : "仅显示实际拥有任务的人；没有任务的 PO 不会混进来";
    $("modeNote").textContent = demand
      ? "PO视角：看我负责的需求是否持续向前推进"
      : "研发执行视角：看开发/测试具体正在做什么";
    if (demand) { loadDemand(); } else { loadTasks(); }
  }
  $("demandTab").addEventListener("click", function () { switchMode("demand"); });
  $("taskTab").addEventListener("click", function () { switchMode("task"); });
  document.addEventListener("click", function (e) {
    var r = e.target.closest("[data-retry]");
    if (r) { mode === "demand" ? loadDemand() : loadTasks(); }
  });

  /* ---------- group / owners chips ---------- */
  function renderTeamChips() {
    var host = $("agileChips"); host.innerHTML = "";
    if (!teams.length) { return; }
    teams.forEach(function (team, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "chip" + (i === 0 || state.teamgroup === team.id ? " active" : "");
      b.textContent = team.name;
      b.dataset.tg = team.id;
      b.addEventListener("click", function () {
        teams.forEach(function (t) { t.btn && t.btn.classList.remove("active"); });
        document.querySelectorAll("#agileChips .chip").forEach(function (x) { x.classList.remove("active"); });
        b.classList.add("active");
        state.teamgroup = Number(b.dataset.tg);
        $("metricsGroupName").textContent = b.textContent;
        // 任务模式随小组刷新；需求模式当前按当前账号（后端未接小组过滤）
        if (mode === "task") { loadTasks(); }
        showToast("已切换到 " + b.textContent);
      });
      host.appendChild(b);
    });
    if (teams.length) { $("metricsGroupName").textContent = teams[0].name; }
  }
  function renderOwnerChips(containerId, list, onSelect) {
    var host = $(containerId); host.innerHTML = "";
    if (!list || !list.length) { return; }
    list.forEach(function (it, i) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "person" + (i === 0 || state.owner === it.value ? " active" : "");
      b.dataset.value = it.value;
      var inner = '<span class="count">' + it.count + "</span>";
      if (it.display && it.display.length) {
        inner = '<span class="mini-avatar">' + esc(it.display.charAt(0)) + "</span>" + esc(it.display) + inner;
      } else {
        inner = "全部" + inner;
      }
      b.innerHTML = inner;
      b.addEventListener("click", function () {
        host.querySelectorAll(".person").forEach(function (x) { x.classList.remove("active"); });
        b.classList.add("active");
        state.owner = b.dataset.value;
        onSelect(state.owner);
      });
      host.appendChild(b);
    });
  }

  /* ---------- demand matrix ---------- */
  var STAGE_COLS = { "clarify": 1, "schedule": 2, "dev": 3, "test": 4, "deliver": 5 };
  function stageColOf(stage) {
    stage = String(stage || "");
    if (stage.indexOf("受理") >= 0 || stage.indexOf("澄清") >= 0) { return 1; }
    if (stage.indexOf("排期") >= 0) { return 2; }
    if (stage.indexOf("研发") >= 0 || stage.indexOf("提测") >= 0) { return 3; }
    if (stage.indexOf("联调") >= 0 || stage.indexOf("验收") >= 0 || stage.indexOf("测试") >= 0) { return 4; }
    if (stage.indexOf("交付") >= 0 || stage.indexOf("评价") >= 0) { return 5; }
    return 0;
  }
  function leafAction(item) {
    if (item.actionLabel) { return item.actionLabel; }
    return "查看";
  }
  function renderStageCards(row, object) {
    var html = "";
    for (var col = 1; col <= 5; col++) {
      if (col === stageColOf(object.stage)) {
        var tone = statusTone(object.status);
        var cls = "stage-card";
        if (object.status === "clarified") { cls += " actionable"; }
        else if (tone === "orange") { cls += " blocked"; }
        else if (tone === "red") { cls += " overdue"; }
        var top = '<span class="stage-name">' + esc(object.stage || object.status) + "</span>";
        var action = object.url
          ? '<a class="stage-action" href="' + esc(object.url) + '" target="_blank" rel="noopener noreferrer">' + esc(leafAction(object)) + "</a>"
          : "";
        var mini = "";
        if (object.storyCount > 0) { mini += object.storyCount + " 研发需求 · "; }
        if (object.taskOpenCount > 0) { mini += object.taskOpenCount + " 开放任务 · "; }
        if (object.storyCount === 0 && object.kind !== "story") { mini += "未拆研发 · 即使无任务也显示 · "; }
        html += '<div class="stage-cell"><div class="' + cls + '"><div class="stage-top">' + top + action +
          "</div><div class=\"stage-mini\">" + esc(mini || "") + "</div></div></div>";
      } else {
        html += '<div class="stage-cell"></div>';
      }
    }
    return html;
  }
  function renderDemandRow(object, depth, isLast) {
    var branch = (depth === 0 ? "" : (isLast ? "└" : "├"));
    var ownerBadge = object.owner
      ? '<span class="owner-badge"><span class="owner-dot">' + esc(object.owner.charAt(0)) + "</span>" + esc(object.owner) + "</span>"
      : "";
    var metaBits = [];
    if (object.deadline) { metaBits.push("目标上线 " + esc(object.deadline)); }
    if (object.subDemandCount > 0) { metaBits.push(object.subDemandCount + " 子业务"); }
    if (object.storyCount > 0) { metaBits.push(object.storyCount + " 研发需求"); }
    return (
      '<div class="demand-row" data-owner="' + esc(object.owner || "") + '" data-stage="' + esc(object.stage || "") + '" data-rd="' + esc(object.displayId || "") + '">' +
      '<div class="tree-cell ind' + depth + '">' +
      (depth > 0 ? '<span class="branch">' + branch + "</span>" : "") +
      typeTag(object.kind) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(object.displayId) + "</span>" +
      '<span class="node-title">' + esc(object.title) + "</span>" + priTag(object.priority) + "</div>" +
      '<div class="node-meta">' + ownerBadge + (metaBits.length ? '<span>' + esc(metaBits.join(" · ")) + "</span>" : "") + "</div></div>" +
      "</div>" +
      renderStageCards(null, object) +
      "</div>"
    );
  }
  function renderDemandMatrix(tree) {
    var host = $("demandGroups");
    if (!tree || !tree.length) { host.innerHTML = ""; $("demandEmpty").style.display = "block"; return; }
    $("demandEmpty").style.display = "none";
    var html = "";
    tree.forEach(function (root, ri) {
      var children = root.children || [];
      var hasChild = children.length > 0;
      // 自身对象作为最细行(无子业务)或作为汇总根
      if (hasChild) {
        // 根 = 业务需求汇总
        var cflags = [];
        if (root.status === "clarified") { cflags.push("schedule"); }
        if (root.status === "refuse" || root.status === "hang") { cflags.push("blocked"); }
        var summaries = [];
        if (children.length) { summaries.push('<span class="summary">' + children.length + " 子业务</span>"); }
        if (root.storyCount) { summaries.push('<span class="summary">' + root.storyCount + " 研发需求</span>"); }
        var rootOwner = root.owner ? '<span class="owner-badge"><span class="owner-dot">' + esc(root.owner.charAt(0)) + "</span>PO " + esc(root.owner) + "</span>" : "";
        html += '<section class="biz-group" data-owner="' + esc(root.owner || "") + '" data-flags="' + cflags.join(" ") + '" id="bg' + ri + '">';
        html += '<div class="biz-head"><div class="biz-head-main" data-toggle-group="bg' + ri + '">' +
          '<button type="button" class="toggle">▼</button>' + typeTag("demand") + '<span class="code">' + esc(root.displayId) + "</span>" +
          '<span class="biz-title" title="' + esc(root.title) + '">' + esc(root.title) + "</span>" + priTag(root.priority) +
          rootOwner + summaries.join("") +
          '<span class="biz-date">' + (root.deadline ? "目标上线 " + esc(root.deadline) : "") + "</span></div></div>";
        html += '<div class="group-body">';
        children.forEach(function (child, ci) {
          html += renderDemandRow(child, 1, ci === children.length - 1);
        });
        html += "</div></section>";
      } else {
        // 无子业务的业务需求/独立研发需求：自身即最细行
        html += '<section class="biz-group" data-owner="' + esc(root.owner || "") + '" data-flags="" id="bg' + ri + '">';
        html += '<div class="group-body">' + renderDemandRow(root, 0, true) + "</div></section>";
      }
    });
    host.innerHTML = html;
    host.querySelectorAll("[data-toggle-group]").forEach(function (h) {
      h.addEventListener("click", function () {
        var group = $(h.dataset.toggleGroup);
        if (!group) { return; }
        var body = group.querySelector(".group-body");
        var toggle = group.querySelector(".toggle");
        var collapsed = body.classList.toggle("collapsed");
        toggle.textContent = collapsed ? "▶" : "▼";
      });
    });
    renderSummaryFromRows();
  }
  function renderSummaryFromRows() {
    var rows = Array.prototype.slice.call(document.querySelectorAll("#demandGroups .demand-row"));
    var total = rows.length, clarify = 0, schedule = 0, blocked = 0, overdue = 0;
    rows.forEach(function (r) {
      var stage = r.dataset.stage || "";
      if (stage.indexOf("澄清") >= 0 || stage.indexOf("受理") >= 0) { clarify++; }
      if (stage.indexOf("排期") >= 0) { schedule++; }
      if (r.className.indexOf("blocked") >= 0) { blocked++; }
      if (r.className.indexOf("overdue") >= 0) { overdue++; }
    });
    $("demandStats").querySelector('[data-flag="clarify"] strong').textContent = clarify;
    $("demandStats").querySelector('[data-flag="schedule"] strong').textContent = schedule;
    $("demandStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("demandStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
    if (total === 0) { $("demandEmpty").style.display = "block"; }
  }

  /* ---------- loaders ---------- */
  function loadDemand() {
    var host = $("demandGroups");
    host.innerHTML = '<div class="demand-empty">加载中…</div>';
    fetch("/board/demand/items", { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) {
          teams = payload.teamgroups;
          renderTeamChips();
        }
        renderDemandMatrix(payload.tree || []);
        // 需求负责人：仅含实际拥有需求对象的人
        var owners = {};
        (payload.tree || []).forEach(function walk(n) {
          if (n && n.owner) { owners[n.owner] = (owners[n.owner] || 0) + 1; }
          (n.children || []).forEach(walk);
        });
        var opts = [{ value: "", display: "全部", count: (payload.tree || []).length }];
        Object.keys(owners).forEach(function (acc) { opts.push({ value: acc, display: acc, count: owners[acc] }); });
        state.owner = "";
        renderOwnerChips("demandOwners", opts, function (owner) {
          document.querySelectorAll("#demandGroups .demand-row, #demandGroups .biz-group")
            .forEach(function (el) { el.classList.toggle("hidden", owner && el.dataset.owner !== owner); });
        });
      })
      .catch(function () { onErr("demandGroups"); });
  }
  function loadTasks() {
    var host = document.querySelector("#taskBoard .task-col-body"); // 容器先清
    document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) { c.innerHTML = ""; });
    var params = new URLSearchParams();
    if (state.teamgroup) { params.set("teamgroupId", state.teamgroup); }
    if (state.storyFilter) { params.set("storyId", state.storyFilter); }
    fetch("/board/task/items?" + params.toString(), { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) {
          teams = payload.teamgroups;
          renderTeamChips();
        }
        if (payload.owners) {
          var opts = [{ value: "", display: "全部", count: payload.owners.reduce(function (s, o) { return s + o.count; }, 0) }];
          payload.owners.forEach(function (o) { opts.push({ value: o.account, display: o.display, count: o.count }); });
          renderOwnerChips("taskOwners", opts, function () { applyTaskOwnerFilter(); });
        }
        renderTaskColumns(payload.columns || []);
        updateTaskStats();
      })
      .catch(function () { onErr("taskBoard"); });
  }
  function renderTaskColumns(cols) {
    var colMap = {};
    cols.forEach(function (c) { colMap[c.key] = c.items || []; });
    ["wait", "doing", "done"].forEach(function (key) {
      var colEl = document.querySelector('#taskBoard .task-col[data-col="' + key + '"]');
      if (!colEl) { return; }
      var body = colEl.querySelector(".task-col-body");
      var items = colMap[key] || [];
      body.innerHTML = items.map(taskCard).join("");
      colEl.querySelector(".k-count").textContent = items.length;
    });
  }
  function taskCard(task) {
    var cls = "task-card";
    if (task.blocked) { cls += " blocked"; }
    if (task.overdue && task.status !== "done") { cls += " overdue"; }
    if (task.status === "done") { cls += " done"; }
    var due = task.deadline ? "截止 " + esc(task.deadline) : "";
    if (task.overdue && task.status !== "done") { due = '<span class="task-due over">已超期</span>'; }
    var storyLink = task.storyTitle
      ? '<span class="task-link" data-back-rd="' + esc(task.displayId || task.storyId) + '">所属研需 ' + esc(task.storyTitle) + "</span>"
      : "";
    return (
      '<div class="' + cls + '" data-owner="' + esc(task.owner || "") + '">' +
      '<div class="task-head">' + typeTag("task") + '<span class="code">' + esc(task.displayId || task.id) + "</span>" +
      '<span class="task-title" title="' + esc(task.title) + '">' + esc(task.title) + "</span></div>" +
      '<div class="task-meta">' + storyLink + (task.type ? "<span>" + esc(task.type) + "</span>" : "") +
      (task.blocked ? '<span style="color:var(--orange);font-weight:700">阻塞</span>' : "") + "</div>" +
      '<div class="task-footer"><span class="task-due">' + due + "</span>" +
      '<span class="assignee">' + (task.owner ? '<span class="mini-avatar">' + esc(task.owner.charAt(0)) + "</span>" + esc(task.owner) : "") + "</span></div>" +
      "</div>"
    );
  }
  function applyTaskOwnerFilter() {
    var rows = document.querySelectorAll("#taskBoard .task-card");
    rows.forEach(function (r) {
      r.classList.toggle("hidden", state.owner && r.dataset.owner !== state.owner);
    });
    updateTaskStats();
  }
  function updateTaskStats() {
    var blocked = 0, overdue = 0;
    document.querySelectorAll("#taskBoard .task-card").forEach(function (r) {
      if (r.classList.contains("hidden")) { return; }
      if (r.classList.contains("blocked")) { blocked++; }
      if (r.classList.contains("overdue")) { overdue++; }
    });
    $("taskStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("taskStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
  }

  /* ---------- init ---------- */
  switchMode(mode);
})();
