/* PO 工作看板 - 共享状态与工具（V1.3）。零行为搬家。 */
(function (WB) {
  "use strict";

  var mode = location.pathname.indexOf("/board/task") >= 0 ? "task" : "demand";
  // 小组选择：单选某小组，不再提供"全部"。teamgroup=0 仅作为"未选"占位，加载后会落到第一个实际小组。
  var state = { owner: "", teamgroup: loadTeamgroup(), storyFilter: 0, pendingFocus: 0 };
  var teams = [];
  var toastTimer = null;
  var MAX_OWNERS = 6; // 直接展示人数（不含"更多"折叠）

  var TG_KEY = "po.board.teamgroup";
  function loadTeamgroup() { var v = parseInt(localStorage.getItem(TG_KEY), 10); return isNaN(v) ? 0 : v; }
  function saveTeamgroup(id) { localStorage.setItem(TG_KEY, String(id)); }

  function $(id) { return document.getElementById(id); }
  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var priorityBadge = (window.PersonalList && window.PersonalList.priorityBadge) || function (r) {
    var n = parseInt(String(r || "").replace(/^p/i, ""), 10);
    return (isNaN(n) || n < 1 || n > 4) ? '<span class="wb-priority" data-priority="">—</span>' : '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };
  function showToast(msg, type) {
    if (typeof window.showToast === "function" && window.showToast !== showToast) {
      window.showToast(msg, type || "info");
      return;
    }
    var t = $("toast");
    if (!t) { return; }
    t.textContent = msg; t.classList.add("show");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { t.classList.remove("show"); }, 1800);
  }
  if (!window.showToast) window.showToast = showToast;
  function onErr(hostId) {
    var host = $(hostId);
    if (host) host.innerHTML = '<div class="demand-empty">加载失败 · <button type="button" class="action soft" data-retry="1">重试</button></div>';
  }
  function todayStr() {
    var d = new Date(), m = String(d.getMonth() + 1).padStart(2, "0"), day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }
  var NODE_TYPE_KIND = { business: "business", childBusiness: "sub_demand", rd: "story", independentRd: "independent_story", task: "task", unknown: "" };
  var objectTypeBadge = (window.PersonalList && window.PersonalList.objectTypeBadge) || function (kind) {
    var labels = { business: "业务需求", sub_demand: "子需求", story: "研发需求", independent_story: "独立研需", task: "任务" };
    var k = String(kind || "").trim().toLowerCase();
    if (!k || !labels[k]) { return '<span class="wb-type wb-type-unknown">' + (k ? esc(k) : "—") + "</span>"; }
    return '<span class="wb-type wb-type-' + k + '">' + labels[k] + "</span>";
  };
  function nodeTypeOf(node, parentKind) {
    if (node.kind === "story") { return node.independent ? "independentRd" : "rd"; }
    if (node.kind === "sub_demand") { return "childBusiness"; }
    if (node.kind === "demand") { return parentKind ? "childBusiness" : "business"; }
    return "unknown";
  }
  function typeTag(nodeType) {
    if (!NODE_TYPE_KIND.hasOwnProperty(nodeType)) { console.warn("PO board: 未知需求节点类型", nodeType); }
    return objectTypeBadge(NODE_TYPE_KIND[nodeType] || "");
  }
  function priTag(p) { return p ? priorityBadge(p) : ""; }
  function ownerBadge(name, prefix) {
    if (!name) return "";
    var p = prefix ? prefix : "";
    return '<span class="owner-badge"><span class="owner-dot">' + esc(name.charAt(0)) + "</span>" + p + esc(name) + "</span>";
  }

  /* ---------- mode switch ---------- */
  function switchMode(next) {
    mode = next; var demand = next === "demand";
    [["demandTab", demand], ["taskTab", !demand], ["demandBoard", !demand], ["taskBoard", demand],
     ["demandStats", !demand], ["taskStats", demand], ["demandOwners", !demand], ["taskOwners", demand],
     ["demandActions", !demand], ["taskActions", demand]].forEach(function (p) {
      if ($(p[0])) { $(p[0]).classList.toggle(p[0].endsWith("Tab") ? "active" : "hidden", p[1]); }
    });
    if ($("demandRoleBanner")) $("demandRoleBanner").classList.toggle("hidden", !demand);
    if ($("taskRoleBanner")) $("taskRoleBanner").classList.toggle("hidden", demand);
    $("taskFilterTip").classList.toggle("hidden", demand || !state.storyFilter);
    $("ownerTitle").textContent = demand ? "PO / 需求负责人" : "任务负责人";
    if (demand) { if (WB.loadDemand) WB.loadDemand(); } else { ensureTeamgroup(); if (WB.loadTasks) WB.loadTasks(); }
  }
  $("demandTab").addEventListener("click", function () { switchMode("demand"); });
  $("taskTab").addEventListener("click", function () { switchMode("task"); });

  /* ---------- group / owners ---------- */
  function ensureTeamgroup() {
    // 看板必须有真实小组；首次进入无记录时落到第一个。
    if (state.teamgroup === 0 && teams.length) { state.teamgroup = teams[0].id; }
  }
  function renderTeamChips() {
    var host = $("agileChips"); host.innerHTML = "";
    if (!teams.length) { return; }
    if (state.teamgroup === 0) {
      var idx = teams.findIndex(function (t) { return t.id === loadTeamgroup(); });
      state.teamgroup = idx < 0 ? teams[0].id : teams[idx].id;
    }
    var addBtn = function (id, name, active) {
      var b = document.createElement("button");
      b.type = "button"; b.className = "chip" + (active ? " active" : "");
      b.textContent = name; b.dataset.tg = String(id);
      b.addEventListener("click", function () { onGroupPick(id, name); });
      host.appendChild(b);
    };
    teams.forEach(function (t) { addBtn(t.id, t.name, state.teamgroup === t.id); });
  }
  function onGroupPick(id, name) {
    state.teamgroup = Number(id);
    saveTeamgroup(id);
    document.querySelectorAll("#agileChips .chip").forEach(function (x) { x.classList.remove("active"); });
    var sel = document.querySelector('#agileChips .chip[data-tg="' + id + '"]');
    if (sel) { sel.classList.add("active"); }
    $("metricsGroupName").textContent = name;
    if (WB.loadMetrics) WB.loadMetrics();
    if (mode === "task") { if (WB.loadTasks) WB.loadTasks(); } else { if (WB.loadDemand) WB.loadDemand(); }
  }
  // 负责人最多展示 MAX_OWNERS 人，多出的收进"更多"折叠
  function renderOwnerChips(containerId, list, onSelect) {
    var host = $(containerId); host.innerHTML = "";
    if (!list || !list.length) { return; }
    var rest = list.slice(1), shown = rest.slice(0, MAX_OWNERS), hiddenCount = rest.length - shown.length;
    var mkBtn = function (it) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "person" + ((it.value === "" && state.owner === "") || state.owner === it.value ? " active" : "");
      b.dataset.value = it.value;
      // "全部" chip 不带 count 角标，避免被读成"41 个负责人"。
      if (it.display) {
        var count = '<span class="count">' + it.count + "</span>";
        b.innerHTML = '<span class="mini-avatar">' + esc(it.display.charAt(0)) + "</span>" + esc(it.display) + count;
      } else {
        b.textContent = "全部";
      }
      b.addEventListener("click", function () { selectOwner(host, b); onSelect(it.value); });
      return b;
    };
    host.appendChild(mkBtn(list[0]));
    shown.forEach(function (it) { host.appendChild(mkBtn(it)); });
    if (hiddenCount > 0) {
      var more = document.createElement("button");
      more.type = "button"; more.className = "person more"; more.textContent = "更多 " + hiddenCount + " ›";
      more.addEventListener("click", function () {
        host.removeChild(more);
        rest.forEach(function (it) { host.appendChild(mkBtn(it)); });
      });
      host.appendChild(more);
    }
  }
  function selectOwner(host, btn) {
    host.querySelectorAll(".person").forEach(function (x) { x.classList.remove("active"); });
    btn.classList.add("active");
    state.owner = btn.dataset.value;
  }

  /* ---------- state derivation（真实状态派生，不可拖拽） ---------- */

  Object.assign(WB, {
    mode: function () { return mode; },
    setMode: function (m) { mode = m; },
    state: state,
    getTeams: function () { return teams; },
    setTeams: function (t) { teams = t; },
    MAX_OWNERS: MAX_OWNERS,
    $: $,
    esc: esc,
    showToast: showToast,
    onErr: onErr,
    todayStr: todayStr,
    nodeTypeOf: nodeTypeOf,
    typeTag: typeTag,
    priTag: priTag,
    ownerBadge: ownerBadge,
    priorityBadge: priorityBadge,
    objectTypeBadge: objectTypeBadge,
    switchMode: switchMode,
    ensureTeamgroup: ensureTeamgroup,
    renderTeamChips: renderTeamChips,
    onGroupPick: onGroupPick,
    renderOwnerChips: renderOwnerChips,
    selectOwner: selectOwner,
    loadTeamgroup: loadTeamgroup,
    saveTeamgroup: saveTeamgroup
  });
})(window.PoWB = window.PoWB || {});
