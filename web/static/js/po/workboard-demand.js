/* PO 工作看板 - 需求矩阵渲染与筛选。零行为搬家。 */
(function (WB) {
  "use strict";
  var $ = WB.$;
  var esc = WB.esc;
  var state = WB.state;
  var typeTag = WB.typeTag;
  var priTag = WB.priTag;
  var ownerBadge = WB.ownerBadge;
  var nodeTypeOf = WB.nodeTypeOf;
  var todayStr = WB.todayStr;
  var onErr = WB.onErr;

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
  function renderStageCards(object, nodeType) {
    if (WB.mode() === "demand" && (nodeType === "rd" || nodeType === "childBusiness")) { var e = '<div class="stage-cell"></div>'; return e + e + e + e + e; }
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
  // 子节点行（子需求 / 研发需求）：第一行类型+优先级+标题，第二行编号与交付上下文。
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
      '<div class="node-main"><div class="node-title-line">' + typeTag(nodeType) + priTag(object.priority) + titleTag + "</div>" +
      '<div class="node-meta"><span class="code"># ' + esc(object.displayId) + '</span>' + nodeMeta(object, nodeType) + "</div></div>" +
      "</div>" + renderStageCards(object, nodeType) + "</div>"
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
    var titleTag;
    if (isIndy) {
      // 独立研需来自 zt_story，不能走业务需求详情接口（/demands/:id/detail）。
      // 有禅道链接时按研发需求打开；缺少链接时保持文本展示，避免触发错误请求。
      titleTag = root.url
        ? '<a class="node-title is-link" href="' + esc(root.url) + '" target="_blank" rel="noopener noreferrer" title="' + esc(root.title) + '">' + esc(root.title) + "</a>"
        : '<span class="node-title" title="' + esc(root.title) + '">' + esc(root.title) + "</span>";
    } else {
      titleTag = '<button type="button" class="node-title is-link" data-open-demand="' + esc(root.id) + '" title="' + esc(root.title) + '">' + esc(root.title) + "</button>";
    }
    return '<div class="' + rowCls + '" data-owner="' + esc(root.owner || "") + '" data-flags="' + flags.join(" ") +
      '" data-stage="' + esc(root.stage || "") + '" data-status="' + esc(root.status || "") +
      '" data-deadline="' + esc(root.deadline || "") + '" data-rd="' + (isIndy ? esc(root.displayId) : "") + '" id="bg' + root.id + '">' +
      '<div class="tree-cell ind0">' +
      '<div class="node-main"><div class="node-title-line">' + typeTag(nodeType) + priTag(root.priority) + titleTag + "</div>" +
      '<div class="node-meta"><span class="code"># ' + esc(root.displayId) + '</span>' + metaBits.join(" · ") + "</div></div></div>" + renderStageCards(root, nodeType) + "</div>";
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
      '<div class="tree-cell ind0"><button type="button" class="toggle">▶</button>' +
      '<div class="node-main"><div class="node-title-line">' + typeTag(nodeType) + priTag(root.priority) + '<span class="node-title" title="' + esc(root.title) + '">' + esc(root.title) + '</span></div>' +
      '<div class="node-meta"><span class="code"># ' + esc(root.displayId) + '</span>' + metaBits.join(" · ") + '</div></div></div>' + renderAggregatedCard(root, stories) + '</div>';
  }
  // 需求树矩阵渲染（对齐原型 V1.3）
  function renderDemandMatrix(tree) {
    var host = $("demandGroups");
    if (!tree || !tree.length) { host.innerHTML = ""; $("demandEmpty").style.display = "block"; renderSummaryFromRows(); return; }
    $("demandEmpty").style.display = "none";
    var html = "";
    tree.forEach(function (root) {
      var nodeType = nodeTypeOf(root, null);
      var children = (root.children || []).filter(function (child) { return child.kind !== "story" || child.independent; });
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
        '<button type="button" class="toggle">▼</button>' + '<div class="node-main"><div class="node-title-line">' + typeTag(nodeType) + priTag(root.priority) + '<span class="biz-title" title="' + esc(root.title) + '">' + esc(root.title) + '</span></div>' +
        '<div class="node-meta"><span class="code"># ' + esc(root.displayId) + '</span></div></div>' +
        ownerDot + summaryHtml + dateHtml + '</div></div>' +
        renderCollapsedRow(root, nodeType, stories);
      html += '<section class="biz-group is-collapsed" data-owner="' + esc(root.owner || "") + '" data-flags="' + flags.join(" ") + '" id="bg' + root.id + '">' + headHtml + '<div class="group-body">';
      children.forEach(function (child, ci) {
        var isLastChild = ci === children.length - 1;
        // 展示子业务需求；关联业务需求的研发交付行已在上方过滤。
        html += renderDemandRow(child, 1, isLastChild, root.owner);
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

  Object.assign(WB, {
    renderDemandMatrix: renderDemandMatrix,
    applyDemandFilters: applyDemandFilters,
    renderSummaryFromRows: renderSummaryFromRows,
    toggleFlag: toggleFlag,
    refreshDemandToggleAll: refreshDemandToggleAll
  });
  window.renderDemandMatrix = renderDemandMatrix;
})(window.PoWB = window.PoWB || {});
