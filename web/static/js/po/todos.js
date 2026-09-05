/* =============================================================================
   文件: web/static/js/po/todos.js
   模块: PO 个人工作台 - 我的待办交互脚本
   职责: 绑定待办 7 维筛选、时序防竞争、错误与空态隔离、统一重置与分页
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var $ = function (id) { return document.getElementById(id); };

  var state = {
    tab: "all",
    focus: "pending",
    action: "all",
    stage: "all",
    objectType: "all",
    relation: "all",
    responsibility: "all",
    keyword: "",
    page: 1,
    pageSize: 15
  };

  var searchTimer = null;
  var summaryMap = { countPending: "pending", countToday: "today", countOverdue: "overdue", countBlocked: "blocked", countP1: "p1" };
  var groupMap = { groupAll: "all", groupDemand: "demand", groupExecution: "execution", groupTesting: "testing" };
  var controller = null;

  var TAB_OBJECT_OPTIONS = {
    all: [
      { value: "all", label: "具体对象：全部" },
      { value: "demand", label: "业务需求" },
      { value: "task", label: "任务" },
      { value: "bug", label: "Bug" }
    ],
    demand: [{ value: "all", label: "具体对象：全部" }, { value: "demand", label: "业务需求" }],
    execution: [{ value: "all", label: "具体对象：全部" }, { value: "task", label: "任务" }],
    testing: [{ value: "all", label: "具体对象：全部" }, { value: "bug", label: "Bug" }]
  };

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
    var pri = ["P1", "P2", "P3", "P4"].indexOf(item.priority) >= 0 ? item.priority.toLowerCase() : "normal";
    var id = esc(item.displayId || item.id);
    var title = esc(item.title || "—");
    var idContent = item.url ? '<a class="table-id-link" href="' + esc(item.url) + '" rel="noopener noreferrer">' + id + "</a>" : id;
    var titleContent = item.url ? '<a class="table-title-link" href="' + esc(item.url) + '" rel="noopener noreferrer">' + title + "</a>" : title;
    var action = esc(item.action || "办理");

    return "<tr>" +
      '<td class="todos-col-item" title="' + title + '">' +
        '<div class="todos-item-title">' + titleContent + "</div>" +
        '<div class="todos-item-id">' + idContent + "</div>" +
      "</td>" +
      '<td class="todos-col-type"><span class="type-tag">' + esc(item.type || "—") + "</span></td>" +
      '<td class="todos-col-pri"><span class="inline-pri ' + pri + '">' + esc(item.priority || "—") + "</span></td>" +
      '<td class="todos-col-rel"><span class="relation-tag">' + esc(item.relation || "—") + "</span></td>" +
      '<td class="todos-col-stage" title="' + esc(stageLabel(item.reason)) + '">' + esc(stageLabel(item.reason)) + "</td>" +
      '<td class="todos-col-dead">' + esc(item.deadline || "—") + "</td>" +
      '<td class="todos-col-owner" title="' + esc(item.owner || "—") + '">' + esc(item.owner || "—") + "</td>" +
      '<td class="todos-col-opt">' + (item.url ? '<a class="table-action-btn primary" href="' + esc(item.url) + '" rel="noopener noreferrer">' + action + "</a>" : "—") + "</td>" +
      "</tr>";
  }

  function renderCounts(summary, groups) {
    Object.keys(summaryMap).forEach(function (id) {
      var el = $(id);
      if (el) { el.textContent = (summary && summary[summaryMap[id]] != null) ? summary[summaryMap[id]] : "—"; }
    });
    Object.keys(groupMap).forEach(function (id) {
      var el = $(id);
      if (el) { el.textContent = (groups && groups[groupMap[id]] != null) ? groups[groupMap[id]] : "—"; }
    });
  }

  function resetCounts() {
    renderCounts({}, {});
  }

  function syncObjectTypeOptions(tab) {
    var select = $("todosObjectType");
    if (!select) { return; }
    var opts = TAB_OBJECT_OPTIONS[tab] || TAB_OBJECT_OPTIONS.all;
    select.innerHTML = opts.map(function (o) {
      return '<option value="' + esc(o.value) + '">' + esc(o.label) + "</option>";
    }).join("");
    state.objectType = "all";
  }

  function refresh() {
    if (!controller) { return; }
    var url = buildUrl();
    controller.fetch(url, state, function (payload) {
      var items = (payload && payload.items) || [];
      var total = (payload && payload.total) || 0;
      var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

      if (state.page > totalPages && total > 0) {
        state.page = totalPages;
        refresh();
        return;
      }

      var tbody = $("todosTbody");
      if (tbody) { tbody.innerHTML = items.map(rowHtml).join(""); }
      if ($("todosSummary")) { $("todosSummary").textContent = "共 " + total + " 条待办事项"; }
      if (payload && payload.summary && payload.groups) {
        renderCounts(payload.summary, payload.groups);
      }

      if (window.PersonalList) {
        window.PersonalList.renderPagination({
          container: $("todosPagination"),
          page: state.page,
          pageSize: state.pageSize,
          total: total,
          onPageChange: function (p) { state.page = p; refresh(); },
          onPageSizeChange: function (s) { state.pageSize = s; state.page = 1; refresh(); }
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
        refresh();
      });
    });
  }

  function initTabs() {
    var tabs = document.querySelectorAll(".po-todos .category-tab");
    tabs.forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (btn.disabled || btn.classList.contains("unsupported")) { return; }
        var tab = btn.getAttribute("data-tab");
        if (!tab || state.tab === tab) { return; }
        tabs.forEach(function (t) {
          t.classList.remove("active");
          t.removeAttribute("aria-current");
        });
        btn.classList.add("active");
        btn.setAttribute("aria-current", "page");
        state.tab = tab;
        state.page = 1;
        state.stage = "all";
        state.action = "all";
        if ($("todosStage")) { $("todosStage").value = "all"; }
        if ($("todosAction")) { $("todosAction").value = "all"; }
        syncObjectTypeOptions(tab);
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
        refresh();
      });
    });

    ["todosAction:action", "todosStage:stage", "todosObjectType:objectType", "todosResponsibility:responsibility"].forEach(function (pair) {
      var parts = pair.split(":");
      var sel = $(parts[0]);
      if (sel) {
        sel.addEventListener("change", function () {
          state[parts[1]] = sel.value;
          state.page = 1;
          refresh();
        });
      }
    });

    var resetBtn = $("todosResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.tab = "all";
        state.focus = "pending";
        state.action = "all";
        state.stage = "all";
        state.objectType = "all";
        state.relation = "all";
        state.responsibility = "all";
        state.keyword = "";
        state.page = 1;

        document.querySelectorAll("#todosQuickChips .header-quick-chip").forEach(function (c) {
          var isPending = c.getAttribute("data-focus") === "pending";
          c.classList.toggle("active", isPending);
          c.setAttribute("aria-pressed", isPending ? "true" : "false");
        });
        document.querySelectorAll(".po-todos .category-tab").forEach(function (t) {
          var isAll = (t.getAttribute("data-tab") || "all") === "all";
          t.classList.toggle("active", isAll);
          if (isAll) {
            t.setAttribute("aria-current", "page");
          } else {
            t.removeAttribute("aria-current");
          }
        });
        relationBtns.forEach(function (b) {
          b.classList.toggle("active", (b.getAttribute("data-relation") || "all") === "all");
        });

        if (kwInput) { kwInput.value = ""; }
        if ($("todosAction")) { $("todosAction").value = "all"; }
        if ($("todosStage")) { $("todosStage").value = "all"; }
        if ($("todosResponsibility")) { $("todosResponsibility").value = "all"; }
        syncObjectTypeOptions("all");
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

    syncObjectTypeOptions("all");
    initQuickChips();
    initTabs();
    initToolbar();
    refresh();
  });
})();
