/* PO 工作看板（需求/任务双视图，V1.3 紧凑效能快照）。
   数据真实；指标带来自 /board/group/metrics（小组级）。树遍历用迭代栈，不使用递归。 */
(function () {
  "use strict";

  var mode = location.pathname.indexOf("/board/task") >= 0 ? "task" : "demand";
  // 小组选择：优先上次记录，其次主小组，最后才全部；teamgroup=0 表示"全部"。
  var state = { owner: "", teamgroup: loadTeamgroup(), storyFilter: 0, pendingFocus: 0 };
  var teams = [];
  var toastTimer = null;
  var MAX_OWNERS = 6; // 直接展示人数（不含"全部"与"更多"）

  var TG_KEY = "po.board.teamgroup";
  function loadTeamgroup() {
    var v = parseInt(localStorage.getItem(TG_KEY), 10);
    return isNaN(v) ? 0 : v;
  }
  function saveTeamgroup(id) { localStorage.setItem(TG_KEY, String(id)); }

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
    toastTimer = setTimeout(function () { t.classList.remove("show"); }, 1800);
  }
  window.showToast = showToast;
  function onErr(hostId) {
    var host = $(hostId);
    if (host) { host.innerHTML = '<div class="demand-empty">加载失败 · <button type="button" class="action soft" data-retry="1">重试</button></div>'; }
  }
  function todayStr() {
    var d = new Date();
    var m = String(d.getMonth() + 1).padStart(2, "0");
    var day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }
  /* 四类需求节点类型（前端 ViewModel）：基于真实父子关系 + 树位置判定，不依赖 ID 前缀/标题。
     business=业务需求根；childBusiness=子业务；rd=研发需求；independentRd=独立研发需求；unknown=未知（不 fallback）。 */
  var NODE_TYPES = {
    business: { label: "业务需求", cls: "type-biz" },
    childBusiness: { label: "子业务", cls: "type-child" },
    rd: { label: "研发需求", cls: "type-rd" },
    independentRd: { label: "独立研发需求", cls: "type-rd type-rd-indep" },
    task: { label: "任务", cls: "type-task" },
    unknown: { label: "未知需求类型", cls: "type-unknown" }
  };
  function nodeTypeOf(node, parentKind) {
    if (node.kind === "story") { return node.independent ? "independentRd" : "rd"; }
    if (node.kind === "sub_demand") { return "childBusiness"; }
    if (node.kind === "demand") { return parentKind ? "childBusiness" : "business"; }
    return "unknown";
  }
  function typeTag(nodeType) {
    var t = NODE_TYPES[nodeType] || NODE_TYPES.unknown;
    if (!NODE_TYPES[nodeType]) { console.warn("PO board: 未知需求节点类型", nodeType); }
    return '<span class="type-tag ' + t.cls + '">' + t.label + "</span>";
  }
  function priTag(p) {
    if (!p) { return ""; }
    return '<span class="priority">' + esc(p) + "</span>";
  }
  function ownerBadge(name, prefix) {
    if (!name) { return ""; }
    var p = prefix ? prefix : "";
    return '<span class="owner-badge"><span class="owner-dot">' + esc(name.charAt(0)) + "</span>" + p + esc(name) + "</span>";
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
    if ($("demandRoleBanner")) { $("demandRoleBanner").classList.toggle("hidden", !demand); }
    if ($("taskRoleBanner")) { $("taskRoleBanner").classList.toggle("hidden", demand); }
    $("taskFilterTip").classList.toggle("hidden", demand || !state.storyFilter);
    $("ownerTitle").textContent = demand ? "PO / 需求负责人" : "任务负责人";
    $("ownerHint").textContent = demand
      ? "仅显示当前小组中实际拥有业务需求 / 独立研发需求的人；开发人员不会混进来"
      : "仅显示当前小组中实际拥有任务的人；没有任务的 PO 不会混进来";
    $("modeNote").textContent = demand
      ? "PO视角：看我负责的需求是否持续向前推进"
      : "研发执行视角：看开发/测试具体正在做什么";
    if (demand) { loadDemand(); } else { ensureTeamgroup(); loadTasks(); }
  }
  $("demandTab").addEventListener("click", function () { switchMode("demand"); });
  $("taskTab").addEventListener("click", function () { switchMode("task"); });

  /* ---------- group / owners ---------- */
  function ensureTeamgroup() {
    // 任务看板必须有真实小组；用户显式选"全部"时不强制改回
    if (state.teamgroup === 0 && teams.length && !state.userPickedAll) { state.teamgroup = teams[0].id; }
  }
  function renderTeamChips() {
    var host = $("agileChips"); host.innerHTML = "";
    if (!teams.length) { return; }
    // 默认具体小组：无记录则取首个小组为当前用户的主小组；用户已选"全部"时保持全部。
    if (state.teamgroup === 0 && !state.userPickedAll) {
      var idx = teams.findIndex(function (t) { return t.id === loadTeamgroup(); });
      state.teamgroup = idx < 0 ? teams[0].id : teams[idx].id;
    }
    // "全部"作为可选项保留，但默认不选中
    var all = document.createElement("button");
    all.type = "button";
    all.className = "chip" + (state.teamgroup === 0 ? " active" : "");
    all.textContent = "全部";
    all.dataset.tg = "0";
    all.addEventListener("click", function () { onGroupPick(0, "全部"); });
    host.appendChild(all);
    teams.forEach(function (team) {
      var b = document.createElement("button");
      b.type = "button";
      b.className = "chip" + (state.teamgroup === team.id ? " active" : "");
      b.textContent = team.name;
      b.dataset.tg = team.id;
      b.addEventListener("click", function () { onGroupPick(team.id, team.name); });
      host.appendChild(b);
    });
  }
  function onGroupPick(id, name) {
    state.teamgroup = Number(id);
    state.userPickedAll = state.teamgroup === 0;
    saveTeamgroup(id);
    document.querySelectorAll("#agileChips .chip").forEach(function (x) { x.classList.remove("active"); });
    var sel = document.querySelector('#agileChips .chip[data-tg="' + id + '"]');
    if (sel) { sel.classList.add("active"); }
    $("metricsGroupName").textContent = id === 0 ? "全部" : name;
    loadMetrics();
    if (mode === "task") { loadTasks(); } else { loadDemand(); }
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
      var count = '<span class="count">' + it.count + "</span>";
      b.innerHTML = it.display ? '<span class="mini-avatar">' + esc(it.display.charAt(0)) + "</span>" + esc(it.display) + count : "全部" + count;
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
  function stageColOf(stage) {
    stage = String(stage || "");
    if (/受理|澄清/.test(stage)) { return 1; }
    if (/排期/.test(stage)) { return 2; }
    if (/研发|提测/.test(stage)) { return 3; }
    if (/联调|验收/.test(stage)) { return 4; }
    if (/交付|评价/.test(stage)) { return 5; }
    return 0;
  }
  function isBlocked(it) { return it.status === "refuse" || it.status === "hang"; }
  function isOverdue(it) {
    if (it.status === "done" || it.status === "released") { return false; }
    return it.deadline && it.deadline < todayStr();
  }
  // 研需阶段卡：查看任务仅 taskTotal>0；无任务时按阶段给"建任务/去排期"真实动作。
  function storyReason(it) {
    if (it.taskTotal > 0) { return "任务 " + it.taskDone + "/" + it.taskTotal + (it.currentOwner ? " · " + it.currentOwner + "负责" : ""); }
    if (/研发|提测/.test(it.stage)) { return "尚未创建研发任务"; }
    if (/排期/.test(it.stage)) { return "未绑定版本窗口"; }
    if (/受理|澄清/.test(it.stage)) { return "科技侧需求梳理尚未完成"; }
    if (/联调|验收/.test(it.stage)) { return "测试完成 · 等待业务验收"; }
    if (/交付|评价/.test(it.stage)) { return "待交付 · 等待发起"; }
    return "尚未进入执行";
  }
  function storyAction(it) {
    if (it.taskTotal > 0) { return { label: "查看任务", open: true }; }
    if (/排期/.test(it.stage)) { return { label: "去排期", open: false }; }
    if (/受理|澄清/.test(it.stage)) { return { label: "去梳理", open: false }; }
    if (/研发|提测/.test(it.stage)) { return { label: "建任务", open: false }; }
    if (/联调|验收/.test(it.stage)) { return { label: "验收", open: false }; }
    return { label: "查看任务", open: true };
  }
  function renderStageCards(object) {
    var html = "";
    var col = stageColOf(object.stage);
    var blocked = isBlocked(object);
    var overdue = isOverdue(object);
    for (var c = 1; c <= 5; c++) {
      if (c !== col) { html += '<div class="stage-cell"></div>'; continue; }
      var cls = "stage-card";
      if (blocked) { cls += " blocked"; }
      else if (overdue) { cls += " overdue"; }
      else { cls += " actionable"; }
      if (object.kind === "story") {
        var pill = '<span class="stage-pill' + (object.status === "done" ? " green" : "") + '">' + esc(object.stage) + "</span>";
        if (object.progress > 0 && object.status !== "done") {
          pill = '<span class="stage-pill">开发中 ' + object.progress + '%</span>';
        }
        var act = storyAction(object);
        var actionBtn = act.open
          ? '<button type="button" class="stage-action" data-open-tasks="' + object.id + '" data-rd-label="' + esc(object.displayId) + '">' + esc(act.label) + "</button>"
          : '<button type="button" class="stage-action" data-toast="' + esc(act.label + "：" + object.displayId) + '">' + esc(act.label) + "</button>";
        var mini = storyReason(object);
        var bar = (object.progress > 0 && object.status !== "done") ? '<div class="progress"><span style="width:' + object.progress + '%"></span></div>' : "";
        html += '<div class="stage-cell"><div class="' + cls + '"><div class="stage-top">' + pill + actionBtn + "</div>" +
          '<div class="stage-mini">' + esc(mini) + "</div>" + bar + "</div></div>";
      } else {
        // 业务需求 / 子业务：只有无研需时自身承载阶段；否则仅由下层最细对象承载（父不重复出卡）
        if (object.storyCount > 0) {
          // 父节点仅汇总，不在此列渲染推进卡；但保持列占位对齐
          if (c === col) { html += '<div class="stage-cell"><span class="stage-mini">' + object.storyCount + " 研发需求推进中</span></div>"; }
          else { html += '<div class="stage-cell"></div>'; }
          continue;
        }
        var nm = isBlocked(object) ? esc(object.stage) + " · 阻塞" : esc(object.stage);
        var reason = "";
        if (object.kind === "sub_demand") { reason = "待拆研发需求"; }
        else if (object.stage.indexOf("排期") >= 0) { reason = "未绑定版本窗口"; }
        else if (object.stage.indexOf("受理") >= 0 || object.stage.indexOf("澄清") >= 0) { reason = "科技侧需求梳理尚未完成"; }
        if (isBlocked(object)) { reason = "审批未通过 · 暂不可推进"; }
        var actLabel = object.actionLabel || "查看";
        var btn = '<button type="button" class="stage-action" data-toast="' + esc(actLabel + "：" + object.displayId) + '">' + esc(actLabel) + "</button>";
        var m = reason ? '<div class="stage-mini">' + esc(reason) + "</div>" : "";
        html += '<div class="stage-cell"><div class="' + cls + '"><div class="stage-top"><span class="stage-name">' + nm + "</span>" + btn + "</div>" + m + "</div></div>";
      }
    }
    return html;
  }
  function nodeMeta(object, nodeType) {
    var bits = [];
    if (nodeType === "rd" || nodeType === "independentRd") {
      if (object.productName) { bits.push("产品：" + esc(object.productName)); }
      if (object.currentOwner) { bits.push(esc(object.currentOwner) + " 负责"); }
    } else if (nodeType === "childBusiness") {
      bits.push("交付单元");
      if (object.owner) { bits.push(ownerBadge(object.owner)); }
    } else {
      if (object.owner) { bits.push(ownerBadge(object.owner)); }
    }
    if (object.deadline) { bits.push("目标上线 " + esc(object.deadline)); }
    return bits.length ? '<span>' + bits.join(" · ") + "</span>" : "";
  }
  // 子节点行（子业务 / 研发需求）：第一行 tag+ID+标题+P，第二行 meta。
  function renderDemandRow(object, depth, isLast, groupOwner) {
    var nodeType = nodeTypeOf(object, depth > 0 ? "childBusiness" : null);
    var branch = depth === 0 ? "" : (isLast ? "└" : "├");
    var isIndy = !!object.independent;
    var rowOwner = isIndy ? object.owner : (groupOwner || object.owner);
    var ind = Math.min(depth, 2);
    var rowCls = "demand-row" + (isOverdue(object) ? " has-overdue" : "") + (isBlocked(object) ? " has-blocked" : "");
    return (
      '<div class="' + rowCls + '" data-owner="' + esc(rowOwner || "") + '" data-stage="' + esc(object.stage || "") +
      '" data-status="' + esc(object.status || "") + '" data-deadline="' + esc(object.deadline || "") +
      '" data-rd="' + (object.kind === "story" ? esc(object.displayId) : "") + '">' +
      '<div class="tree-cell ind' + ind + '">' +
      (depth > 0 ? '<span class="branch">' + branch + "</span>" : "") +
      typeTag(nodeType) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(object.displayId) + "</span>" +
      '<span class="node-title" title="' + esc(object.title) + '">' + esc(object.title) + "</span>" + priTag(object.priority) + "</div>" +
      '<div class="node-meta">' + nodeMeta(object, nodeType) + "</div></div>" +
      "</div>" + renderStageCards(object) + "</div>"
    );
  }
  // 叶行：根节点自身即最细推进对象（无子业务的 BR / 独立研需）。
  function renderLeafRow(object) {
    var nodeType = nodeTypeOf(object, null);
    var isIndy = !!object.independent;
    var rowCls = "demand-row" + (isOverdue(object) ? " has-overdue" : "") + (isBlocked(object) ? " has-blocked" : "");
    var metaBits = [];
    if (object.owner) { metaBits.push("PO " + esc(object.owner)); }
    if (isIndy) {
      metaBits.push("尚未纳入执行");
    } else {
      metaBits.push("研发需求 0");
    }
    return (
      '<div class="' + rowCls + '" data-owner="' + esc(object.owner || "") + '" data-stage="' + esc(object.stage || "") +
      '" data-status="' + esc(object.status || "") + '" data-deadline="' + esc(object.deadline || "") +
      '" data-rd="' + (isIndy ? esc(object.displayId) : "") + '">' +
      '<div class="tree-cell"><span class="branch">└</span>' +
      typeTag(nodeType) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(object.displayId) + "</span>" +
      '<span class="node-title" title="' + esc(object.title) + '">' + esc(object.title) + "</span>" + priTag(object.priority) + "</div>" +
      '<div class="node-meta"><span>' + metaBits.join(" · ") + "</span></div></div></div>" +
      renderStageCards(object) + "</div>"
    );
  }
  // 需求树矩阵渲染（对齐原型 V1.3）
  function renderDemandMatrix(tree) {
    var host = $("demandGroups");
    if (!tree || !tree.length) { host.innerHTML = ""; $("demandEmpty").style.display = "block"; renderSummaryFromRows(); return; }
    $("demandEmpty").style.display = "none";
    var html = "";
    tree.forEach(function (root) {
      var nodeType = nodeTypeOf(root, null);
      var children = root.children || [];
      var hasChild = children.length > 0;
      var flags = [];
      if (isOverdue(root)) { flags.push("overdue"); }
      if (isBlocked(root)) { flags.push("blocked"); }
      var summaries = [];
      if (nodeType === "business") {
        if (root.subDemandCount > 0) { summaries.push(root.subDemandCount + " 子业务"); }
        if (root.storyCount > 0) { summaries.push(root.storyCount + " 研发需求"); }
        if (root.subDemandCount === 0 && root.storyCount === 0) { summaries.push("无子业务"); }
      } else {
        summaries.push("无业务需求来源");
      }
      if (root.stage && root.stage.indexOf("排期") >= 0) { summaries.push("待排期"); }
      if (isBlocked(root)) { summaries.push("阻塞"); }
      if (isOverdue(root)) { summaries.push("超期"); }

      var summaryHtml = summaries.map(function (s) {
        var cls = "summary";
        if (s.indexOf("阻塞") >= 0 || s.indexOf("排期") >= 0) { cls += " warn"; }
        if (s.indexOf("超期") >= 0) { cls += " danger"; }
        return '<span class="' + cls + '">' + esc(s) + "</span>";
      }).join("");

      var ownerDot = root.owner ? '<span class="owner-badge"><span class="owner-dot">' + esc(root.owner.charAt(0)) + "</span>PO " + esc(root.owner) + "</span>" : "";
      var dateHtml = root.deadline ? '<span class="biz-date">目标上线 ' + esc(root.deadline) + "</span>" : "";

      html += '<section class="biz-group" data-owner="' + esc(root.owner || "") + '" data-flags="' + flags.join(" ") + '" id="bg' + root.id + '">';
      html += '<div class="biz-head"><div class="biz-head-main" data-toggle-group="bg' + root.id + '">' +
        '<button type="button" class="toggle">▼</button>' +
        typeTag(nodeType) +
        '<span class="code">' + esc(root.displayId) + "</span>" +
        '<span class="biz-title" title="' + esc(root.title) + '">' + esc(root.title) + "</span>" +
        priTag(root.priority) +
        ownerDot +
        summaryHtml +
        dateHtml +
        "</div></div>";
      html += '<div class="group-body">';
      if (hasChild) {
        children.forEach(function (child, ci) {
          html += renderDemandRow(child, child.kind === "story" ? 2 : 1, ci === children.length - 1, root.owner);
        });
      } else {
        html += renderLeafRow(root);
      }
      html += "</div></section>";
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
    applyDemandOwnerFilter();
  }
  function applyDemandOwnerFilter() {
    document.querySelectorAll("#demandGroups .demand-row, #demandGroups .biz-group")
      .forEach(function (el) { el.classList.toggle("hidden", state.owner && el.dataset.owner !== state.owner); });
    renderSummaryFromRows();
  }
  function renderSummaryFromRows() {
    var rows = document.querySelectorAll("#demandGroups .demand-row");
    var clarify = 0, schedule = 0, blocked = 0, overdue = 0, visible = 0;
    rows.forEach(function (r) {
      if (r.classList.contains("hidden")) { return; }
      visible++;
      var stage = r.dataset.stage || "";
      var status = r.dataset.status || "";
      var deadline = r.dataset.deadline || "";
      if (stage.indexOf("受理") >= 0 || stage.indexOf("澄清") >= 0) { clarify++; }
      if (stage.indexOf("排期") >= 0) { schedule++; }
      if (status === "refuse" || status === "hang") { blocked++; }
      if (deadline && status !== "done" && status !== "released" && deadline < todayStr()) { overdue++; }
    });
    $("demandStats").querySelector('[data-flag="clarify"] strong').textContent = clarify;
    $("demandStats").querySelector('[data-flag="schedule"] strong').textContent = schedule;
    $("demandStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("demandStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
    if (visible === 0) { $("demandEmpty").style.display = "block"; }
  }

  /* ---------- metrics ---------- */
  function loadMetrics() {
    // 全部小组 → 轻量 Empty State（指标属于具体小组）
    if (state.teamgroup === 0) {
      $("metricsGroupName").textContent = "全部";
      $("metricsGrid").innerHTML = '<div class="metric-empty">请选择具体敏捷小组查看效能指标</div>';
      return;
    }
    $("metricsGrid").innerHTML = '<div class="metric-empty">加载中…</div>';
    fetch("/board/group/metrics?teamgroupId=" + state.teamgroup, { method: "GET" })
      .then(function (r) { return r.json(); })
      .then(function (p) {
        if (!p || p.success !== true) { throw new Error("payload"); }
        $("metricsGroupName").textContent = p.groupName || "";
        if (p.hasGroup && p.metrics && p.metrics.length) { renderMetrics(p.metrics); }
        else { $("metricsGrid").innerHTML = '<div class="metric-empty">请选择具体敏捷小组查看效能指标</div>'; }
      })
      .catch(function () { $("metricsGrid").innerHTML = '<div class="metric-empty">效能指标加载失败</div>'; });
  }
  function renderMetrics(metrics) {
    var details = {
      "交付周期": "端到端流动速度：需求从进入研发到完成交付用了多久。",
      "实施周期": "研发实施效率：进入实施后到完成研发交付用了多久。",
      "超预迭代周期占比": "交付可预测性：有多少工作超过预期迭代节奏。",
      "超2周未排期": "需求入口健康度：识别长期未进入时间盒的需求积压。",
      "质量门禁通过率": "内建质量：研发产物能否稳定通过质量门禁。",
      "缺陷关闭率": "质量收敛：已发现缺陷是否及时形成闭环。",
      "缺陷响应效率": "质量响应速度：缺陷是否得到及时确认和处理。",
      "上线延期数": "交付兑现：已承诺上线的事项是否按窗口完成。"
    };
    $("metricsGrid").innerHTML = metrics.map(function (m) {
      var cls = "team-metric" + (m.state ? " is-" + m.state : "");
      var trendHtml = m.trend ? '<span class="metric-trend ' + (m.state || "") + '">' + esc(m.trend) + "</span>" : "";
      return '<div class="' + cls + '" data-metric-name="' + esc(m.name) + '">' +
        '<div class="metric-top"><span class="metric-name">' + esc(m.name) + '</span><span class="metric-value">' + esc(m.value === "-" ? "—" : m.value) + "</span></div>" +
        '<div class="metric-meta"><span class="metric-target">' + (m.target ? "目标 " + esc(m.target) : "") + "</span>" + trendHtml + "</div></div>";
    }).join("");
    $("metricsGrid").querySelectorAll(".team-metric").forEach(function (el) {
      el.addEventListener("click", function () { var n = el.dataset.metricName || ""; showToast(n + "：" + (details[n] || "")); });
    });
  }

  /* ---------- loaders ---------- */
  function loadDemand() {
    var host = $("demandGroups");
    host.innerHTML = '<div class="demand-empty">加载中…</div>';
    var params = new URLSearchParams();
    if (state.teamgroup) { params.set("teamgroupId", state.teamgroup); }
    fetch("/board/demand/items?" + params.toString(), { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) { teams = payload.teamgroups; renderTeamChips(); }
        renderDemandMatrix(payload.tree || []);
        renderDemandOwners(payload.tree || []);
        if (state.pendingFocus) { focusStoryRow(state.pendingFocus); state.pendingFocus = 0; }
        loadMetrics();
      })
      .catch(function () { onErr("demandGroups"); });
  }
  function seqOwners(tree) {
    var owners = {};
    var stack = (tree || []).slice().reverse();
    while (stack.length) {
      var n = stack.pop();
      if (n && n.owner) { owners[n.owner] = (owners[n.owner] || 0) + 1; }
      var cs = (n && n.children) || [];
      for (var i = cs.length - 1; i >= 0; i--) { stack.push(cs[i]); }
    }
    var arr = Object.keys(owners).map(function (acc) { return { value: acc, display: acc, count: owners[acc] }; });
    arr.sort(function (a, b) { return b.count - a.count; });
    return arr;
  }
  function renderDemandOwners(tree) {
    var opts = [{ value: "", display: "", count: (tree || []).length }];
    opts = opts.concat(seqOwners(tree));
    state.owner = "";
    renderOwnerChips("demandOwners", opts, applyDemandOwnerFilter);
  }
  function focusStoryRow(storyId) {
    var row = document.querySelector('.demand-row[data-rd="RD-' + storyId + '"]');
    if (!row) { return; }
    var grp = row.closest(".biz-group");
    if (grp) {
      var body = grp.querySelector(".group-body");
      if (body && body.classList.contains("collapsed")) {
        body.classList.remove("collapsed");
        var tg = grp.querySelector(".toggle");
        if (tg) { tg.textContent = "▼"; }
      }
    }
    row.scrollIntoView({ block: "center", behavior: "smooth" });
    row.style.background = "#fffbea";
    setTimeout(function () { row.style.background = ""; }, 2000);
  }
  function loadTasks() {
    document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) { c.innerHTML = ""; });
    ensureTeamgroup();
    if (!state.teamgroup) { return; }
    var params = new URLSearchParams();
    params.set("teamgroupId", state.teamgroup);
    if (state.storyFilter) { params.set("storyId", state.storyFilter); }
    if (state.owner) { params.set("ownerAccount", state.owner); }
    fetch("/board/task/items?" + params.toString(), { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        if (payload.teamgroups && payload.teamgroups.length) { teams = payload.teamgroups; renderTeamChips(); }
        if (payload.owners) {
          var total = payload.owners.reduce(function (s, o) { return s + o.count; }, 0);
          var opts = [{ value: "", display: "", count: total }];
          payload.owners.forEach(function (o) { opts.push({ value: o.account, display: o.display, count: o.count }); });
          renderOwnerChips("taskOwners", opts, function () { loadTasks(); });
        }
        renderTaskColumns(payload.columns || []);
        loadMetrics();
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
    updateTaskStats();
  }
  function taskCard(task) {
    var cls = "task-card";
    if (task.blocked) { cls += " blocked"; }
    if (task.overdue && task.status !== "done") { cls += " overdue"; }
    if (task.status === "done") { cls += " done"; }
    var due = task.deadline ? "截止 " + esc(task.deadline) : "";
    if (task.overdue && task.status !== "done") { due = '<span class="task-due over">已超期</span>'; }
    var storyLink = "";
    if (task.storyId) {
      storyLink = '<span class="task-link" data-back-rd="' + task.storyId + '" title="返回需求看板并定位研需">所属研需 ' + esc(task.storyTitle || ("RD-" + task.storyId)) + "</span>";
    }
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
  function updateTaskStats() {
    var blocked = 0, overdue = 0, visible = 0;
    document.querySelectorAll("#taskBoard .task-card").forEach(function (r) {
      visible++;
      if (r.classList.contains("blocked")) { blocked++; }
      if (r.classList.contains("overdue")) { overdue++; }
    });
    $("taskStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("taskStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
    if (visible === 0) {
      document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) {
        if (!c.children.length) { c.innerHTML = '<div class="demand-empty" style="display:block">当前条件下没有任务</div>'; }
      });
    }
  }

  /* ---------- demand ↔ task 穿透 ---------- */
  function openStoryTasks(storyId, label) {
    state.storyFilter = storyId;
    state.owner = "";
    var tip = $("taskFilterTip");
    tip.innerHTML = '已从需求看板下钻：仅显示 <strong>' + esc(label) + '</strong> 的任务。点任务卡上的「所属研需」可返回需求看板。';
    var allBtn = document.querySelector('#taskOwners .person[data-value=""]');
    if (allBtn) { allBtn.classList.add("active"); }
    ensureTeamgroup();
    switchMode("task");
    tip.classList.remove("hidden");
  }
  function backToDemandFromTask(storyId) {
    state.storyFilter = 0;
    state.pendingFocus = storyId;
    switchMode("demand");
  }

  document.addEventListener("click", function (e) {
    var retry = e.target.closest("[data-retry]");
    if (retry) { mode === "demand" ? loadDemand() : loadTasks(); return; }
    var open = e.target.closest("[data-open-tasks]");
    if (open) { openStoryTasks(Number(open.dataset.openTasks), open.dataset.rdLabel || "研需"); return; }
    var back = e.target.closest("[data-back-rd]");
    if (back) { backToDemandFromTask(Number(back.dataset.backRd)); return; }
    var toastBtn = e.target.closest("[data-toast]");
    if (toastBtn) { showToast(toastBtn.dataset.toast); return; }
  });

  /* ---------- issue panel ---------- */
  function loadIssues() {
    fetch("/board/issues", { method: "GET" })
      .then(function (r) { if (!r.ok) { throw new Error("http"); } return r.json(); })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("payload"); }
        renderIssues(payload);
      })
      .catch(function () { $("issueList").innerHTML = '<div class="demand-empty">加载失败</div>'; });
  }
  var issueTabFilter = "open";
  function renderIssues(payload) {
    var items = payload.items || [];
    var open = (payload.open || 0), closed = (payload.closed || 0);
    $("issueCount").textContent = items.length || open + closed;
    var tabs = $("issueTabs"); tabs.innerHTML = "";
    var tabDefs = [{ key: "open", label: "未解决 " + open }];
    if (closed > 0) { tabDefs.push({ key: "closed", label: "已关闭 " + closed }); }
    tabDefs.forEach(function (td) {
      var b = document.createElement("button");
      b.type = "button"; b.className = "issue-tab" + (issueTabFilter === td.key ? " active" : "");
      b.textContent = td.label; b.dataset.key = td.key;
      b.addEventListener("click", function () {
        issueTabFilter = td.key;
        tabs.querySelectorAll(".issue-tab").forEach(function (x) { x.classList.remove("active"); });
        b.classList.add("active");
        renderIssueCards(items, td.key);
      });
      tabs.appendChild(b);
    });
    renderIssueCards(items, issueTabFilter);
  }
  function renderIssueCards(items, filter) {
    var list = $("issueList");
    var matched = items.filter(function (it) { return (it.status === "closed" ? "closed" : "open") === filter; });
    if (!matched.length) { list.innerHTML = '<div class="demand-empty">暂无问题</div>'; return; }
    list.innerHTML = matched.map(function (it) {
      var isCls = it.status === "closed";
      return '<div class="issue-card"><div class="issue-id">ISSUE-' + it.id + "</div>" +
        '<div class="issue-title" title="' + esc(it.title) + '">' + esc(it.title) + "</div>" +
        '<div class="issue-meta"><span>' + esc(it.createdBy || "") + (it.priority ? " · P" + esc(it.priority) : "") + "</span>" +
        '<span class="' + (isCls ? "" : "issue-state") + '">' + (isCls ? "已关闭" : "未处理") + "</span></div></div>";
    }).join("");
  }

  /* ---------- init ---------- */
  loadIssues();
  switchMode(mode);
})();