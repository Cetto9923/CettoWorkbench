/* =============================================================================
   文件: web/static/js/po/todos.js
   模块: PO 个人工作台 - 我的待办交互脚本
   职责: 绑定待办 7 维筛选、时序防竞争、错误与空态隔离、统一重置与分页
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var PL = window.PersonalList || {};
  var priorityBadge = PL.priorityBadge || function (raw) {
    var n = parseInt(String(raw || "").replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };
  var objectTypeBadgeFromKind = PL.objectTypeBadgeFromKind || function (kind) {
    var map = { demand: "business", story: "story", task: "task", bug: "bug", test: "testtask", testtask: "testtask", approval: "approval", todo: "todo", issue: "issue", risk: "risk" };
    var labels = { business: "业务需求", story: "研发需求", task: "任务", bug: "Bug", testtask: "测试单", approval: "审批", todo: "待办", issue: "问题", risk: "风险" };
    var k = map[String(kind || "").toLowerCase()] || "";
    if (!k) { return '<span class="wb-type wb-type-unknown">—</span>'; }
    return '<span class="wb-type wb-type-' + k + '">' + (labels[k] || k) + "</span>";
  };
  // API kind → wb-type cssKind（与 PersonalList.idChipHtml 期望对齐）。
  var chipKindFromApi = function (apiKind) {
    var map = { demand: "business", business: "business", story: "story", task: "task", bug: "bug",
                test: "testtask", testtask: "testtask", approval: "approval", todo: "todo", issue: "issue", risk: "risk" };
    return map[String(apiKind || "").toLowerCase()] || "";
  };
  var primaryActionHtml = (window.PrimaryAction && window.PrimaryAction.primaryActionHtml) || function () {
    return '<span class="home-unavailable" title="等待服务端动作合同落地">—</span>';
  };
  var $ = function (id) { return document.getElementById(id); };

  // 分类芯片直接就是对象类型，因此 objectType 是唯一的对象维度状态，不再另存 tab。
  var state = {
    focus: "pending",
    action: "all",
    stage: "all",
    objectType: "",
    relation: "all",
    responsibility: "all",
    keyword: "",
    page: 1,
    pageSize: 15
  };

  // 支持全量对象类型（汇总所有审批），与"我的已办"同款芯片风格。
  var OBJECT_TYPE_ORDER = ["", "approval", "demand", "story", "task", "bug", "risk", "issue", "todo", "testtask"];
  var OBJECT_TYPE_LABELS = { "": "全部待办", approval: "审批", demand: "业务需求", story: "研发需求", task: "任务", bug: "Bug", risk: "风险", issue: "问题", todo: "待办", testtask: "测试单" };
  var OBJECT_TYPE_ICONS = { approval: "fa-stamp", demand: "fa-lightbulb", story: "fa-diagram-project", task: "fa-list-check", bug: "fa-bug", risk: "fa-triangle-exclamation", issue: "fa-circle-exclamation", todo: "fa-circle-check", testtask: "fa-vial" };

  var VALID_FOCUSES = ["pending", "today", "overdue", "blocked", "p1"];
  var VALID_RELATIONS = ["all", "in_charge", "cooperate"];
  var VALID_ACTIONS = ["all", "todo_review", "todo_schedule", "todo_verify", "todo_deliver", "todo_follow"];
  var VALID_STAGES = ["all", "accept", "clarify", "schedule", "developing", "testing", "waitacceptance", "acceptanced", "publish", "released"];
  var VALID_RESPONSIBILITIES = ["all", "my_action", "my_follow_up"];
  var hasCorrectedPage = false;

  var searchTimer = null;
  var summaryMap = { countPending: "pending", countToday: "today", countOverdue: "overdue", countBlocked: "blocked", countP1: "p1" };
  var controller = null;

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var p = new URLSearchParams();
    if (state.focus !== "pending") { p.set("focus", state.focus); }
    if (state.action !== "all") { p.set("action", state.action); }
    if (state.stage !== "all") { p.set("stage", state.stage); }
    if (state.objectType !== "") { p.set("objectType", state.objectType); }
    if (state.relation !== "all") { p.set("relation", state.relation); }
    if (state.responsibility !== "all") { p.set("responsibility", state.responsibility); }
    if (state.keyword) { p.set("keyword", state.keyword); }
    if (state.page > 1) { p.set("page", String(state.page)); }
    if (state.pageSize !== 15) { p.set("pageSize", String(state.pageSize)); }
    var qs = p.toString();
    var newUrl = window.location.pathname + (qs ? "?" + qs : "");
    window.history.replaceState(null, "", newUrl);
  }

  function initFromUrl() {
    if (!window.location.search) { return; }
    var sp = new URLSearchParams(window.location.search);
    var focus = (sp.get("focus") || "").trim();
    if (VALID_FOCUSES.indexOf(focus) >= 0) {
      state.focus = focus;
      document.querySelectorAll("#todosQuickChips .header-quick-chip").forEach(function (c) {
        var isCurrent = (c.getAttribute("data-focus") || "") === focus;
        c.classList.toggle("active", isCurrent);
        c.setAttribute("aria-pressed", isCurrent ? "true" : "false");
      });
    }

    var rel = (sp.get("relation") || "").trim();
    if (VALID_RELATIONS.indexOf(rel) >= 0) {
      state.relation = rel;
      document.querySelectorAll(".relation-segment button").forEach(function (b) {
        b.classList.toggle("active", (b.getAttribute("data-relation") || "all") === rel);
      });
    }

    var ot = (sp.get("objectType") || "").trim();
    if (ot === "all") { ot = ""; }
    if (OBJECT_TYPE_ORDER.indexOf(ot) >= 0) {
      state.objectType = ot;
    }

    var act = (sp.get("action") || "").trim();
    if (VALID_ACTIONS.indexOf(act) >= 0 && $("todosAction")) {
      state.action = act;
      $("todosAction").value = act;
    }

    var stg = (sp.get("stage") || "").trim();
    if (VALID_STAGES.indexOf(stg) >= 0 && $("todosStage")) {
      state.stage = stg;
      $("todosStage").value = stg;
    }

    var resp = (sp.get("responsibility") || "").trim();
    if (VALID_RESPONSIBILITIES.indexOf(resp) >= 0 && $("todosResponsibility")) {
      state.responsibility = resp;
      $("todosResponsibility").value = resp;
    }

    var kw = (sp.get("keyword") || "").trim();
    if (kw && $("todosKeyword")) {
      state.keyword = kw;
      $("todosKeyword").value = kw;
    }

    var p = parseInt(sp.get("page"), 10);
    if (!isNaN(p) && p >= 1) {
      state.page = p;
    }
    var ps = parseInt(sp.get("pageSize"), 10);
    if (!isNaN(ps) && [15, 30, 50, 100].indexOf(ps) >= 0) {
      state.pageSize = ps;
    } else {
      // URL 未带 pageSize 时，优先使用上次保存值，再退回默认值。
      state.pageSize = window.PersonalList.loadPageSize("po.todos.pageSize", state.pageSize, [15, 30, 50, 100]);
    }
  }

  function buildUrl() {
    var params = new URLSearchParams();
    Object.keys(state).forEach(function (key) {
      if (state[key] !== "") { params.set(key, String(state[key])); }
    });
    return "/todos/items?" + params.toString();
  }

  function stageLabel(value) {
    var labels = {
      draft: "草稿", wait: "待受理", refuse: "已挂起", active: "待澄清",
      clarified: "待排期", developing: "研发中", testing: "测试中",
      waitacceptance: "待验收", acceptanced: "待交付", waitdeliver: "待发布",
      opened: "处理中", doing: "进行中"
    };
    return labels[value] || value || "—";
  }

  function rowHtml(item) {
    var id = esc(item.displayId || item.id);
    var title = esc(item.title || "—");
    var idContent = item.url ? '<a class="table-id-link" href="' + esc(item.url) + '" target="_blank" rel="noopener noreferrer">' + id + "</a>" : id;
    var titleContent = item.url ? '<a class="table-title-link" href="' + esc(item.url) + '" target="_blank" rel="noopener noreferrer">' + title + "</a>" : title;
    var isStory = String(item.kind || "").toLowerCase() === "story";

    // 对象 + ID 单 chip：与通知中心 / 首页 / 已办同款（wb-type / wb-type-{kind}），
    // 走 PersonalList.idChipHtml（OBJECT_TYPE_SHORT_LABELS 缩写 + "#" 分隔符）。
    var idChip = PL.idChipHtml
      ? PL.idChipHtml(chipKindFromApi(item.kind), idContent)
      : objectTypeBadgeFromKind(item.kind);

    return "<tr>" +
      '<td class="todos-col-obj">' + idChip + "</td>" +
      '<td class="todos-col-item" title="' + title + '">' +
        '<div class="todos-item-title">' + titleContent + "</div>" +
      "</td>" +
      '<td class="todos-col-pri">' + priorityBadge(item.priority) + "</td>" +
      '<td class="todos-col-rel"><span class="relation-tag">' + esc(item.relation || "—") + "</span></td>" +
      '<td class="todos-col-stage" title="' + esc(stageLabel(item.reason)) + '">' + esc(stageLabel(item.reason)) + "</td>" +
      '<td class="todos-col-dead">' + esc(item.deadline || "—") + "</td>" +
      '<td class="todos-col-owner" title="' + esc(item.owner || "—") + '">' + esc(item.owner || "—") + "</td>" +
      '<td class="todos-col-opt">' + primaryActionHtml(item, isStory, ["clarify"]) + "</td>" +
      "</tr>";
  }

  function renderCounts(summary) {
    Object.keys(summaryMap).forEach(function (id) {
      var el = $(id);
      if (el) { el.textContent = (summary && summary[summaryMap[id]] != null) ? summary[summaryMap[id]] : "—"; }
    });
  }

  function resetCounts() {
    renderCounts({});
  }

  // 与 done.js 的 renderObjectChips 同款契约：服务端 facets 提供每个对象类型的 SQL 聚合计数，
  // 空 key 芯片展示各类型之和（后端分面口径已排除对象类型维度自身）。
  function renderObjectChips(facets) {
    var host = $("todosObjectChips");
    if (!host) { return; }

    var countMap = {};
    var facetSum = 0;
    (facets || []).forEach(function (f) {
      countMap[f.key] = Number(f.count) || 0;
      facetSum += countMap[f.key];
    });

    var visibleKeys = OBJECT_TYPE_ORDER.filter(function (key) {
      if (key === "") { return true; }
      if (state.objectType && state.objectType === key) { return true; }
      return (countMap[key] || 0) > 0;
    });

    host.innerHTML = visibleKeys.map(function (key) {
      var active = state.objectType === key;
      var count = key === "" ? facetSum : (countMap[key] || 0);
      var icon = OBJECT_TYPE_ICONS[key] ? '<i class="fas ' + OBJECT_TYPE_ICONS[key] + '"></i>' : "";
      return '<button type="button" class="wb-done-tab' + (active ? " active" : "") + '" data-object-type="' + esc(key) + '" aria-pressed="' + (active ? "true" : "false") + '">' +
        icon + esc(OBJECT_TYPE_LABELS[key] || key) + '<span class="wb-done-tab-count">' + count + "</span>" +
        "</button>";
    }).join("");

    host.querySelectorAll(".wb-done-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var key = btn.getAttribute("data-object-type") || "";
        if (state.objectType === key) { return; }
        state.objectType = key;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        refresh();
      });
    });
  }

  function refresh() {
    if (!controller) { return; }
    var url = buildUrl();
    controller.fetch(url, state, function (payload) {
      var items = (payload && payload.items) || [];
      var total = (payload && payload.total) || 0;
      var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

      if (state.page > totalPages && total > 0 && !hasCorrectedPage) {
        hasCorrectedPage = true;
        state.page = totalPages;
        syncUrl();
        refresh();
        return;
      }
      hasCorrectedPage = false;

      var tbody = $("todosTbody");
      if (tbody) { tbody.innerHTML = items.map(rowHtml).join(""); }
      if ($("todosSummary")) { $("todosSummary").textContent = "共 " + total + " 条待办事项"; }
      if (payload && payload.summary) {
        renderCounts(payload.summary);
      }
      renderObjectChips(payload && payload.facets);

      if (window.PersonalList) {
        window.PersonalList.renderPagination({
          container: $("todosPagination"),
          page: state.page,
          pageSize: state.pageSize,
          total: total,
          onPageChange: function (p) {
            state.page = p;
            hasCorrectedPage = false;
            syncUrl();
            refresh();
          },
          onPageSizeChange: function (s) {
            state.pageSize = s;
            state.page = 1;
            hasCorrectedPage = false;
            window.PersonalList.savePageSize("po.todos.pageSize", s);
            syncUrl();
            refresh();
          }
        });
      }
    });
  }

  function initQuickChips() {
    var chips = document.querySelectorAll("#todosQuickChips .header-quick-chip");
    chips.forEach(function (card) {
      card.addEventListener("click", function () {
        var focus = card.getAttribute("data-focus");
        if (!focus) { return; }
        chips.forEach(function (c) {
          c.classList.remove("active");
          c.setAttribute("aria-pressed", "false");
        });
        card.classList.add("active");
        card.setAttribute("aria-pressed", "true");
        state.focus = focus;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        refresh();
      });
    });
  }

  function initToolbar() {
    var kwInput = $("todosKeyword");
    if (kwInput) {
      kwInput.addEventListener("input", function () {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(function () {
          state.keyword = (kwInput.value || "").trim();
          state.page = 1;
          hasCorrectedPage = false;
          syncUrl();
          refresh();
        }, 300);
      });
    }

    var relationBtns = document.querySelectorAll(".relation-segment button");
    relationBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (btn.disabled) { return; }
        var rel = btn.getAttribute("data-relation");
        if (!rel || state.relation === rel) { return; }
        relationBtns.forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.relation = rel;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        refresh();
      });
    });

    ["todosAction:action", "todosStage:stage", "todosResponsibility:responsibility"].forEach(function (pair) {
      var parts = pair.split(":");
      var sel = $(parts[0]);
      if (sel) {
        sel.addEventListener("change", function () {
          state[parts[1]] = sel.value;
          state.page = 1;
          hasCorrectedPage = false;
          syncUrl();
          refresh();
        });
      }
    });

    var resetBtn = $("todosResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.focus = "pending";
        state.action = "all";
        state.stage = "all";
        state.objectType = "";
        state.relation = "all";
        state.responsibility = "all";
        state.keyword = "";
        state.page = 1;

        document.querySelectorAll("#todosQuickChips .header-quick-chip").forEach(function (c) {
          var isPending = c.getAttribute("data-focus") === "pending";
          c.classList.toggle("active", isPending);
          c.setAttribute("aria-pressed", isPending ? "true" : "false");
        });
        relationBtns.forEach(function (b) {
          b.classList.toggle("active", (b.getAttribute("data-relation") || "all") === "all");
        });

        if (kwInput) { kwInput.value = ""; }
        if ($("todosAction")) { $("todosAction").value = "all"; }
        if ($("todosStage")) { $("todosStage").value = "all"; }
        if ($("todosResponsibility")) { $("todosResponsibility").value = "all"; }
        hasCorrectedPage = false;
        syncUrl();
        refresh();
      });
    }

    var retryBtn = $("todosRetryBtn");
    if (retryBtn) {
      retryBtn.addEventListener("click", function () { refresh(); });
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    if (window.PersonalList) {
      controller = window.PersonalList.createController({
        summaryEl: $("todosSummary"),
        emptyEl: $("todosEmpty"),
        errorEl: $("todosError"),
        tbodyEl: $("todosTbody"),
        errorColspan: 8,
        onError: function () { resetCounts(); }
      });
    }

    initQuickChips();
    initToolbar();
    initFromUrl();
    syncUrl();
    refresh();

    var tbody = $("todosTbody");
    if (tbody) {
      tbody.addEventListener("click", function (e) {
        var target = e.target.closest("a.table-title-link");
        if (!target) { return; }
        var row = target.closest("tr");
        if (!row) { return; }
        var idEl = row.querySelector(".todos-item-id");
        if (idEl && window.DemandDetail) {
          var raw = (idEl.textContent || "").trim();
          // Only business demands use the Workbench drawer.  Numeric IDs in
          // this list may be stories, tasks, bugs, or approvals; opening them
          // through /demands/:id/detail turns those valid ZenTao links into a
          // misleading “需求不存在” response.
          if (/^US\d+/i.test(raw)) {
            e.preventDefault();
            window.DemandDetail.open(raw);
          }
        }
      });
    }
  });
})();
