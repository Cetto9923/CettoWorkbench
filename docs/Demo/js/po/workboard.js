/* PO 工作看板（需求/任务双视图，V1.3 紧凑效能快照）。
   数据真实；指标带来自 /board/group/metrics（小组级）。树遍历用迭代栈，不使用递归。 */
(function () {
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
  function esc(v) {
    return String(v == null ? "" : v)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }
  var priorityBadge = (window.PersonalList && window.PersonalList.priorityBadge) || function (r) {
    var n = parseInt(String(r || "").replace(/^p/i, ""), 10);
    return (isNaN(n) || n < 1 || n > 4) ? '<span class="wb-priority" data-priority="">—</span>' : '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };
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
    if (host) host.innerHTML = '<div class="demand-empty">加载失败 · <button type="button" class="action soft" data-retry="1">重试</button></div>';
  }
  function todayStr() {
    var d = new Date(), m = String(d.getMonth() + 1).padStart(2, "0"), day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }
  var NODE_TYPE_KIND = { business: "business", childBusiness: "sub_demand", rd: "story", independentRd: "independent_story", task: "task", unknown: "" };
  var objectTypeBadge = (window.PersonalList && window.PersonalList.objectTypeBadge) || function (kind) {
    var labels = { business: "业务需求", sub_demand: "子需求", story: "研发需求", independent_story: "独立研发需求", task: "任务" };
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
    if (demand) { loadDemand(); } else { ensureTeamgroup(); loadTasks(); }
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
    if (it.taskTotal > 0) return "任务 " + it.taskDone + "/" + it.taskTotal + (it.currentOwner ? " · " + it.currentOwner + "负责" : "");
    if (/研发|提测/.test(it.stage)) return "尚未创建研发任务";
    if (/排期/.test(it.stage)) return "未绑定版本窗口";
    if (/受理|澄清/.test(it.stage)) return "科技侧需求梳理尚未完成";
    if (/联调|验收/.test(it.stage)) return "测试完成 · 等待业务验收";
    return /交付|评价/.test(it.stage) ? "待交付 · 等待发起" : "尚未进入执行";
  }
  function storyAction(it) {
    if (it.taskTotal > 0) { return { label: "查看任务", open: true }; }
    var m = { "排期": "去排期", "受理": "去梳理", "澄清": "去梳理", "研发": "建任务", "提测": "建任务", "联调": "验收", "验收": "验收" };
    for (var k in m) { if (it.stage && it.stage.indexOf(k) >= 0) { return { label: m[k], open: false }; } }
    return { label: "查看任务", open: true };
  }
  function renderStageCards(object) {
    var html = "", col = stageColOf(object.stage), blocked = isBlocked(object), overdue = isOverdue(object);
    for (var c = 1; c <= 5; c++) {
      if (c !== col) { html += '<div class="stage-cell"></div>'; continue; }
      var cls = "stage-card" + (blocked ? " blocked" : (overdue ? " overdue" : " actionable"));
      if (object.kind === "story") {
        var pill = '<span class="stage-pill' + (object.status === "done" ? " green" : "") + '">' + esc(object.stage) + "</span>";
        if (object.progress > 0 && object.status !== "done") { pill = '<span class="stage-pill">开发中 ' + object.progress + '%</span>'; }
        var act = storyAction(object);
        var actionBtn = /排期/.test(object.stage || "")
          ? window.ScheduleLink.scheduleStoryAction(object, act.label)
          : (act.open ? window.ScheduleLink.openTasksButton(object, act.label) : window.ScheduleLink.toastButton(act.label, object.displayId));
        var mini = storyReason(object);
        var bar = (object.progress > 0 && object.status !== "done") ? '<div class="progress"><span style="width:' + object.progress + '%"></span></div>' : "";
        html += '<div class="stage-cell"><div class="' + cls + '"><div class="stage-top">' + pill + actionBtn + "</div>" +
          '<div class="stage-mini">' + esc(mini) + "</div>" + bar + "</div></div>";
      } else {
        if (object.storyCount > 0) {
          html += '<div class="stage-cell"><span class="stage-mini">' + object.storyCount + " 研发需求推进中</span></div>";
          continue;
        }
        var nm = isBlocked(object) ? esc(object.stage) + " · 阻塞" : esc(object.stage);
        var reason = object.kind === "sub_demand" ? "待拆研发需求" : (/排期/.test(object.stage) ? "未绑定版本窗口" : (/受理|澄清/.test(object.stage) ? "科技侧需求梳理尚未完成" : ""));
        if (isBlocked(object)) { reason = "审批未通过 · 暂不可推进"; }
        var actLabel = object.actionLabel || "查看";
        var isSched = /排期/.test(object.stage || "") && (object.kind === "demand" || object.kind === "sub_demand");
        var btn = isSched ? window.ScheduleLink.scheduleDemandAction(object, actLabel) : window.ScheduleLink.toastButton(actLabel, object.displayId);
        var m = reason ? '<div class="stage-mini">' + esc(reason) + "</div>" : "";
        html += '<div class="stage-cell"><div class="' + cls + '"><div class="stage-top"><span class="stage-name">' + nm + "</span>" + btn + "</div>" + m + "</div></div>";
      }
    }
    return html;
  }
  function nodeMeta(object, nodeType) {
    var bits = [];
    if (nodeType === "rd" || nodeType === "independentRd") {
      if (object.productName) bits.push("产品：" + esc(object.productName));
      if (object.currentOwner) bits.push(esc(object.currentOwner) + " 负责");
    } else if (nodeType === "childBusiness") {
      bits.push("交付单元");
      if (object.owner) bits.push(ownerBadge(object.owner));
    } else if (object.owner) {
      bits.push(ownerBadge(object.owner));
    }
    if (object.deadline) bits.push("目标上线 " + esc(object.deadline));
    return bits.length ? '<span>' + bits.join(" · ") + "</span>" : "";
  }
  // 子节点行（子需求 / 研发需求）：第一行 tag+ID+P+标题，第二行 meta。
  function renderDemandRow(object, depth, isLast, groupOwner) {
    var nodeType = nodeTypeOf(object, depth > 0 ? "childBusiness" : null);
    var branch = depth === 0 ? "" : (isLast ? "└" : "├");
    var isIndy = !!object.independent, rowOwner = isIndy ? object.owner : (groupOwner || object.owner);
    var ind = Math.min(depth, 2);
    var rowCls = "demand-row" + (isOverdue(object) ? " has-overdue" : "") + (isBlocked(object) ? " has-blocked" : "");
    var isDemand = object.kind === "demand" || object.kind === "sub_demand";
    var titleTag = isDemand
      ? '<button type="button" class="node-title is-link" data-open-demand="' + esc(object.id) + '" title="' + esc(object.title) + '">' + esc(object.title) + "</button>"
      : '<span class="node-title" title="' + esc(object.title) + '">' + esc(object.title) + "</span>";
    return (
      '<div class="' + rowCls + '" data-owner="' + esc(rowOwner || "") + '" data-stage="' + esc(object.stage || "") +
      '" data-status="' + esc(object.status || "") + '" data-deadline="' + esc(object.deadline || "") +
      '" data-rd="' + (object.kind === "story" ? esc(object.displayId) : "") + '">' +
      '<div class="tree-cell ind' + ind + '">' +
      (depth > 0 ? '<span class="branch">' + branch + "</span>" : "") +
      typeTag(nodeType) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(object.displayId) + "</span>" +
      priTag(object.priority) + titleTag + "</div>" +
      '<div class="node-meta">' + nodeMeta(object, nodeType) + "</div></div>" +
      "</div>" + renderStageCards(object) + "</div>"
    );
  }
  // 单独一行：无子需求/无子级的事项，直接单行展示（无折叠按钮、无缩进符号、左侧标签正确）
  function renderStandaloneRow(root) {
    var nodeType = nodeTypeOf(root, null), isIndy = !!root.independent;
    var rowCls = "demand-row is-standalone" + (isOverdue(root) ? " has-overdue" : "") + (isBlocked(root) ? " has-blocked" : "");
    var flags = []; if (isOverdue(root)) flags.push("overdue"); if (isBlocked(root)) flags.push("blocked");
    var metaBits = []; if (root.owner) metaBits.push(ownerBadge(root.owner, "PO "));
    metaBits.push(isIndy ? "<span>尚未纳入执行</span>" : "<span>研发需求 0</span>");
    if (root.deadline) metaBits.push('<span class="biz-date">目标上线 ' + esc(root.deadline) + "</span>");
    var titleTag = '<button type="button" class="node-title is-link" data-open-demand="' + esc(root.id) + '" title="' + esc(root.title) + '">' + esc(root.title) + "</button>";
    return '<div class="' + rowCls + '" data-owner="' + esc(root.owner || "") + '" data-flags="' + flags.join(" ") +
      '" data-stage="' + esc(root.stage || "") + '" data-status="' + esc(root.status || "") +
      '" data-deadline="' + esc(root.deadline || "") + '" data-rd="' + (isIndy ? esc(root.displayId) : "") + '" id="bg' + root.id + '">' +
      '<div class="tree-cell ind0">' + typeTag(nodeType) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(root.displayId) + "</span>" +
      priTag(root.priority) + titleTag + "</div>" +
      '<div class="node-meta">' + metaBits.join(" · ") + "</div></div></div>" + renderStageCards(root) + "</div>";
  }
  function collectStories(root) {
    var s = [];
    (root.children || []).forEach(function (c) {
      if (c.kind === "story") s.push(c);
      (c.children || []).forEach(function (gc) { if (gc.kind === "story") s.push(gc); });
    });
    return s;
  }
  function deriveAggStage(root, stories) {
    if (!stories.length) return stageColOf(root.stage) || 1;
    var max = 1; stories.forEach(function (s) { var c = stageColOf(s.stage); if (c > max) max = c; });
    return stories.every(function (s) { return s.status === "done" || /交付|评价/.test(s.stage); }) ? 5 : max;
  }
  function renderAggregatedCard(root, stories) {
    var col = deriveAggStage(root, stories), html = "", tot = stories.length, dev = 0, sched = 0, clar = 0, done = 0, blk = 0, ovd = 0, prog = 0;
    stories.forEach(function (s) {
      if (s.status === "done") { done++; } else if (/研发|提测/.test(s.stage)) { dev++; } else if (/排期/.test(s.stage)) { sched++; } else if (/受理|澄清/.test(s.stage)) { clar++; }
      if (isBlocked(s)) { blk++; } if (isOverdue(s)) { ovd++; } prog += (s.progress || 0);
    });
    var avg = tot > 0 ? (done === tot ? 100 : Math.round(prog / tot)) : 0;
    var names = ["", "受理澄清中", "排期中", "研发实施中", "联调验收中", "已交付"], title = names[col] || "推进中";
    if (col === 3 && avg > 0) { title += " · " + avg + "%"; }
    var parts = [];
    if (dev) { parts.push(dev + "开发中"); } if (sched) { parts.push(sched + "排期"); } if (clar) { parts.push(clar + "澄清"); } if (done) { parts.push(done + "完成"); }
    var meta = tot ? tot + "研需: " + (parts.join(" · ") || "进行中") : ((root.subDemandCount || 0) + "子需求推进中");
    var risk = (blk ? '<span class="risk-badge blocked">⚠️ ' + blk + '阻塞</span>' : "") + (ovd ? '<span class="risk-badge overdue">超期 ' + ovd + '</span>' : "");
    var bar = (col === 3 && avg > 0) ? '<div class="biz-agg-bar"><div class="biz-agg-fill" style="width:' + avg + '%"></div></div>' : "";
    for (var c = 1; c <= 5; c++) {
      if (c !== col) { html += '<div class="stage-cell"></div>'; continue; }
      html += '<div class="stage-cell"><div class="card-biz-agg"><div class="biz-agg-top"><span class="biz-agg-title">🔥 ' + esc(title) +
        '</span><button type="button" class="stage-action" data-toggle-group="bg' + root.id + '">展开明细</button></div>' +
        bar + '<div class="biz-agg-meta"><span>' + esc(meta) + '</span>' + risk + '</div></div></div>';
    }
    return html;
  }
  function renderCollapsedRow(root, nodeType, stories) {
    var metaBits = []; if (root.owner) { metaBits.push(ownerBadge(root.owner, "PO ")); }
    var sub = root.subDemandCount || 0, story = root.storyCount || 0, counts = [];
    if (sub) { counts.push(sub + "子需求"); } if (story) { counts.push(story + "研需"); }
    if (counts.length) { metaBits.push('<span class="summary">' + counts.join(" · ") + '</span>'); }
    if (root.deadline) { metaBits.push('<span class="biz-date">目标 ' + esc(root.deadline) + '</span>'); }
    return '<div class="biz-collapsed-row" data-toggle-group="bg' + root.id + '">' +
      '<div class="tree-cell ind0"><button type="button" class="toggle">▶</button>' + typeTag(nodeType) +
      '<div class="node-main"><div class="node-title-line"><span class="code">' + esc(root.displayId) + '</span>' +
      priTag(root.priority) + '<span class="node-title" title="' + esc(root.title) + '">' + esc(root.title) + '</span></div>' +
      '<div class="node-meta">' + metaBits.join(" · ") + '</div></div></div>' + renderAggregatedCard(root, stories) + '</div>';
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

      // 无子需求/无子级：单独一条展示，不产生折叠头部和重复行
      if (!hasChild) {
        html += renderStandaloneRow(root);
        return;
      }

      var summaries = [];
      if (nodeType === "business") {
        if (root.subDemandCount > 0) summaries.push(root.subDemandCount + " 子需求");
        if (root.storyCount > 0) summaries.push(root.storyCount + " 研发需求");
      } else { summaries.push("无业务需求来源"); }
      if (root.stage && root.stage.indexOf("排期") >= 0) summaries.push("待排期");
      if (isBlocked(root)) summaries.push("阻塞");
      if (isOverdue(root)) summaries.push("超期");

      var summaryHtml = summaries.map(function (s) {
        var cls = "summary" + (/阻塞|排期/.test(s) ? " warn" : (/超期/.test(s) ? " danger" : ""));
        return '<span class="' + cls + '">' + esc(s) + "</span>";
      }).join("");

      var ownerDot = root.owner ? '<span class="owner-badge"><span class="owner-dot">' + esc(root.owner.charAt(0)) + "</span>PO " + esc(root.owner) + "</span>" : "";
      var dateHtml = root.deadline ? '<span class="biz-date">目标上线 ' + esc(root.deadline) + "</span>" : "";
      var stories = collectStories(root);

      var headHtml = '<div class="biz-head"><div class="biz-head-main" data-toggle-group="bg' + root.id + '">' +
        '<button type="button" class="toggle">▼</button>' + typeTag(nodeType) + '<span class="code">' + esc(root.displayId) + '</span>' +
        priTag(root.priority) + '<span class="biz-title" title="' + esc(root.title) + '">' + esc(root.title) + '</span>' +
        ownerDot + summaryHtml + dateHtml + '</div></div>' +
        renderCollapsedRow(root, nodeType, stories);
      html += '<section class="biz-group is-collapsed" data-owner="' + esc(root.owner || "") + '" data-flags="' + flags.join(" ") + '" id="bg' + root.id + '">' + headHtml + '<div class="group-body">';
      children.forEach(function (child, ci) {
        var isLastChild = ci === children.length - 1;
        if (child.kind === "sub_demand") {
          // Layer 2: 子需求
          var grandChildren = child.children || [];
          var hasGrand = grandChildren.length > 0;
          html += renderDemandRow(child, 1, isLastChild && !hasGrand, root.owner);
          // Layer 3: 研发需求（挂在子需求下）
          if (hasGrand) {
            grandChildren.forEach(function (story, si) {
              var isLastStory = isLastChild && (si === grandChildren.length - 1);
              html += renderDemandRow(story, 2, isLastStory, root.owner);
            });
          }
        } else {
          // 根需求直接关联的研发需求
          html += renderDemandRow(child, 1, isLastChild, root.owner);
        }
      });
      html += "</div></section>";
    });
    host.innerHTML = html;
    host.querySelectorAll("[data-toggle-group]").forEach(function (h) {
      h.addEventListener("click", function (e) {
        if (e.target.closest(".stage-action[data-toast]")) { return; }
        e.stopPropagation();
        var group = $(h.dataset.toggleGroup);
        if (!group) { return; }
        group.classList.toggle("is-collapsed");
        refreshDemandToggleAll();
      });
    });
    var allBtn = $("demandToggleAll");
    if (allBtn) {
      allBtn.onclick = function () {
        var groups = host.querySelectorAll(".biz-group");
        if (!groups.length) { return; }
        var anyCollapsed = host.querySelectorAll(".biz-group.is-collapsed").length > 0;
        groups.forEach(function (g) {
          if (anyCollapsed) { g.classList.remove("is-collapsed"); }
          else { g.classList.add("is-collapsed"); }
        });
        refreshDemandToggleAll();
      };
    }
    refreshDemandToggleAll();
    applyDemandFilters();
  }
  function refreshDemandToggleAll() {
    var btn = $("demandToggleAll");
    if (!btn) { return; }
    var groups = document.querySelectorAll("#demandGroups .biz-group");
    if (!groups.length) { btn.hidden = true; return; }
    btn.hidden = false;
    var anyCollapsed = document.querySelectorAll("#demandGroups .biz-group.is-collapsed").length > 0;
    btn.textContent = anyCollapsed ? "展开研需" : "折叠研需";
  }
  var activeFlag = "";
  function toggleFlag(flag) {
    activeFlag = activeFlag === flag ? "" : flag;
    document.querySelectorAll("#demandStats .stat").forEach(function (x) {
      x.classList.toggle("active", x.dataset.flag === activeFlag);
    });
    applyDemandFilters();
  }
  window.renderDemandMatrix = renderDemandMatrix;
  function rowMatchesFlag(r, flag) {
    if (!flag) { return true; }
    var stage = r.dataset.stage || "";
    var status = r.dataset.status || "";
    var deadline = r.dataset.deadline || "";
    if (flag === "clarify") { return /受理|澄清/.test(stage); }
    if (flag === "schedule") { return /排期/.test(stage); }
    if (flag === "blocked") { return status === "refuse" || status === "hang"; }
    if (flag === "overdue") { return deadline && status !== "done" && status !== "released" && deadline < todayStr(); }
    return true;
  }
  function applyDemandFilters() {
    var owner = state.owner;
    document.querySelectorAll("#demandGroups .biz-group").forEach(function (group) {
      var groupOwner = group.dataset.owner;
      var groupFlags = (group.dataset.flags || "").split(" ");
      var hasVisibleRow = false;
      group.querySelectorAll(".demand-row").forEach(function (row) {
        var ownerOk = !owner || row.dataset.owner === owner;
        var flagOk = rowMatchesFlag(row, activeFlag);
        var match = ownerOk && flagOk;
        row.classList.toggle("hidden", !match);
        if (match) { hasVisibleRow = true; }
      });
      var groupFlagOk = !activeFlag || groupFlags.indexOf(activeFlag) >= 0;
      var groupOwnerOk = !owner || groupOwner === owner;
      var showGroup = (groupOwnerOk && groupFlagOk) || hasVisibleRow;
      group.classList.toggle("hidden", !showGroup);
    });
    document.querySelectorAll("#demandGroups > .demand-row").forEach(function (row) {
      var ownerOk = !owner || row.dataset.owner === owner;
      var flagOk = rowMatchesFlag(row, activeFlag);
      row.classList.toggle("hidden", !(ownerOk && flagOk));
    });
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
        if (payload.teamgroups && payload.teamgroups.length) { teams = payload.teamgroups; renderTeamChips(); }
        renderDemandMatrix(payload.tree || []); renderDemandOwners(payload.tree || []);
        if (state.pendingFocus) { focusStoryRow(state.pendingFocus); state.pendingFocus = 0; }
        loadMetrics();
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
          // 与需求看板一致：首位"全部" chip 不带 count 角标，避免被读成"41 个负责人"。
          var opts = [{ value: "", display: "", count: 0 }];
          payload.owners.forEach(function (o) { opts.push({ value: o.account, display: o.display, count: o.count }); });
          renderOwnerChips("taskOwners", opts, function () { loadTasks(); });
        }
        renderTaskColumns(payload.columns || []);
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
  function taskCard(task) {
    var cls = "task-card" + (task.blocked ? " blocked" : "") + (task.overdue && task.status !== "done" ? " overdue" : "") + (task.status === "done" ? " done" : "");
    var due = (task.overdue && task.status !== "done") ? '<span class="task-due over">已超期</span>' : (task.deadline ? "截止 " + esc(task.deadline) : "");
    var storyLink = task.storyId ? '<span class="task-link" data-back-rd="' + task.storyId + '" title="返回需求看板并定位研需">所属研需 ' + esc(task.storyTitle || String(task.storyId)) + "</span>" : "";
    return '<div class="' + cls + '" data-owner="' + esc(task.owner || "") + '">' +
      '<div class="task-head">' + typeTag("task") + '<span class="code">' + esc(task.displayId || task.id) + '</span><span class="task-title" title="' + esc(task.title) + '">' + esc(task.title) + "</span></div>" +
      '<div class="task-meta">' + storyLink + (task.type ? "<span>" + esc(task.type) + "</span>" : "") + (task.blocked ? '<span style="color:var(--orange);font-weight:700">阻塞</span>' : "") + "</div>" +
      '<div class="task-footer"><span class="task-due">' + due + '</span><span class="assignee">' + (task.owner ? '<span class="mini-avatar">' + esc(task.owner.charAt(0)) + "</span>" + esc(task.owner) : "") + "</span></div></div>";
  }
  function updateTaskStats() {
    var blocked = 0, overdue = 0, visible = 0;
    document.querySelectorAll("#taskBoard .task-card").forEach(function (r) {
      visible++; if (r.classList.contains("blocked")) { blocked++; } if (r.classList.contains("overdue")) { overdue++; }
    });
    $("taskStats").querySelector('[data-flag="blocked"] strong').textContent = blocked;
    $("taskStats").querySelector('[data-flag="overdue"] strong').textContent = overdue;
    if (visible === 0) {
      document.querySelectorAll("#taskBoard .task-col-body").forEach(function (c) {
        if (!c.children.length) { c.innerHTML = '<div class="demand-empty" style="display:block">当前条件下没有任务</div>'; }
      });
    }
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
  window.fetchDrawerTasks = function (id) { fetchDrawerTasks(id); };
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
      '<div class="drawer-task-top"><span class="drawer-task-code">' + esc(t.displayId || "TASK-" + t.id) + "</span>" +
      '<span class="drawer-task-type ' + typeC + '">' + esc(typN) + "</span>" + priH +
      '<span class="drawer-task-status ' + esc(t.status || "wait") + '">' + esc(stN) + "</span></div>" +
      '<div class="drawer-task-name">' + esc(t.title) + "</div>" +
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
    var retry = e.target.closest("[data-retry]"); if (retry) { mode === "demand" ? loadDemand() : loadTasks(); return; }
    var flagBtn = e.target.closest("#demandStats [data-flag]"); if (flagBtn) { toggleFlag(flagBtn.dataset.flag); return; }
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
  loadIssues(); switchMode(mode);
})();
