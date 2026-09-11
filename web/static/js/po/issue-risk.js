/* =============================================================================
   文件: web/static/js/po/issue-risk.js
   模块: PO 工作台 - 问题风险交互脚本
   职责: 绑定 kind/relation/status/keyword/loop/overdue/project 维度筛选、
         时序防竞争、错误与空态隔离、单页内抽屉详情。
   依赖: personal-list.js
   ============================================================================= */
(function () {
  "use strict";

  var PL = window.PersonalList || {};
  var esc = PL.escapeHtml || function (v) {
    return String(v == null ? "" : v)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  };
  var priorityBadge = PL.priorityBadge || function (raw) {
    var n = parseInt(String(raw || "").replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };
  var objectTypeBadgeFromKind = PL.objectTypeBadgeFromKind || function (kind) {
    return '<span class="wb-type wb-type-' + esc(String(kind || "unknown")) + '">' + esc(String(kind || "—")) + "</span>";
  };
  var $ = function (id) { return document.getElementById(id); };

  var STATUS_OPTIONS = {
    issue: [
      ["", "状态：全部"],
      ["unconfirmed", "未处理"],
      ["active", "激活"],
      ["wait", "待处理"],
      ["doing", "处理中"],
      ["confirmed", "已确认"],
      ["resolved", "已解决"],
      ["closed", "已关闭"]
    ],
    risk: [
      ["", "状态：全部"],
      ["active", "激活"],
      ["tracked", "跟踪中"],
      ["hangup", "已挂起"],
      ["closed", "已关闭"],
      ["canceled", "已取消"]
    ]
  };

  var state = {
    kind: "issue",
    relation: "allRelated",
    status: "",
    keyword: "",
    loop: "all",
    overdue: false,
    project: 0,
    page: 1,
    pageSize: 20
  };

  var searchTimer = null;
  var controller = null;
  var lastItems = [];

  function refreshStatus() {
    var sel = $("irStatus");
    if (!sel) { return; }
    var list = STATUS_OPTIONS[state.kind] || STATUS_OPTIONS.issue;
    sel.innerHTML = list.map(function (x) {
      var picked = x[0] === state.status ? " selected" : "";
      return '<option value="' + esc(x[0]) + '"' + picked + ">" + esc(x[1]) + "</option>";
    }).join("");
  }

  function refreshProjects(projects) {
    var sel = $("irProject");
    if (!sel) { return; }
    var list = projects || [];
    sel.innerHTML = '<option value="">项目：全部</option>' + list.map(function (p) {
      var picked = Number(state.project) === Number(p.id) ? " selected" : "";
      return '<option value="' + esc(p.id) + '"' + picked + ">" + esc(p.name || "—") + "</option>";
    }).join("");
  }

  function syncChips() {
    var chips = document.querySelectorAll("#irQuickChips .header-quick-chip");
    chips.forEach(function (chip) {
      var loopKey = chip.getAttribute("data-loop");
      var overdueKey = chip.getAttribute("data-overdue");
      var active = false;
      if (overdueKey === "true" && state.overdue && state.loop === "open") {
        active = true;
      } else if (loopKey && !state.overdue && loopKey === state.loop) {
        active = true;
      }
      chip.classList.toggle("active", active);
      chip.setAttribute("aria-pressed", active ? "true" : "false");
    });
  }

  function buildUrl() {
    var params = new URLSearchParams();
    params.set("kind", state.kind);
    params.set("relation", state.relation);
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    if (state.status) { params.set("status", state.status); }
    if (state.keyword) { params.set("keyword", state.keyword); }
    if (state.loop && state.loop !== "all") { params.set("loop", state.loop); }
    if (state.overdue) { params.set("overdue", "true"); }
    if (state.project) { params.set("project", String(state.project)); }
    return "/issues/risk/items?" + params.toString();
  }

  function severityClass(severity) {
    if (severity === "致命") { return "danger"; }
    if (severity === "严重") { return "warn"; }
    return "";
  }

  function rowHtml(item, idx) {
    var severity = item.severity || "—";
    var sevClass = severityClass(severity);
    var title = esc(item.title || "—");
    var idText = esc(item.displayId || item.id || "—");
    var url = esc(item.url || "#");
    var titleBtn = '<button type="button" class="table-title-link" data-ir-detail="' + esc(item.kind) + '|' + esc(item.id) + '">' + title + "</button>";
    var idLink = '<a class="table-id-link" href="' + url + '" target="_blank" rel="noopener noreferrer">' + idText + "</a>";
    var overdueTag = item.isOverdue
      ? '<span class="ir-tag danger">逾期 ' + esc(String(item.overdueDays || 1)) + " 天</span>"
      : '<span class="ir-tag">' + esc(String(item.days || 0)) + " 天</span>";
    var actionBtn = '<button type="button" class="action-btn ir-action-view" data-ir-detail="' + esc(item.kind) + '|' + esc(item.id) + '">查看</button>';
    return "<tr data-row-idx=\"" + idx + "\">" +
      '<td>' + idLink + "</td>" +
      '<td class="ir-title">' + titleBtn + "</td>" +
      "<td>" + esc(item.project || "—") + "</td>" +
      '<td><span class="ir-tag ' + sevClass + '">' + esc(severity) + "</span></td>" +
      "<td>" + priorityBadge(item.priority) + "</td>" +
      "<td>" + esc(item.handler || "—") + "</td>" +
      "<td>" + esc(item.submitter || "—") + "</td>" +
      "<td>" + esc(item.planDate || "—") + "</td>" +
      "<td>" + ((PL && PL.statusTagHtml) ? PL.statusTagHtml(item.status) : esc(item.status || "—")) + "</td>" +
      "<td>" + overdueTag + "</td>" +
      "<td>" + actionBtn + "</td>" +
      "</tr>";
  }

  function setText(id, value) {
    var el = $(id);
    if (el) { el.textContent = value; }
  }

  function renderCounts(payload) {
    var loopCounts = (payload && payload.loopCounts) || {};
    setText("irCountAll", loopCounts.all != null ? loopCounts.all : "—");
    setText("irCountOpen", loopCounts.open != null ? loopCounts.open : "—");
    setText("irCountClosed", loopCounts.closed != null ? loopCounts.closed : "—");
    setText("irCountOverdue", payload && payload.overdueCount != null ? payload.overdueCount : "—");
    var kindCounts = (payload && payload.kindCounts) || {};
    setText("issueCount", kindCounts.issue != null ? kindCounts.issue : "—");
    setText("riskCount", kindCounts.risk != null ? kindCounts.risk : "—");
  }

  function refresh(payload) {
    if (!controller) { return; }
    var url = buildUrl();
    controller.fetch(url, state, function (p) {
      lastItems = (p && p.items) || [];
      renderCounts(p);
      refreshProjects((p && p.projects) || []);

      var total = (p && p.total) || 0;
      var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

      var tbody = $("irTbody");
      if (tbody) { tbody.innerHTML = lastItems.map(function (x, i) { return rowHtml(x, i); }).join(""); }

      var summary = $("irSummary");
      if (summary) {
        summary.textContent = "共 " + total + " 条" + (state.kind === "risk" ? "风险" : "问题");
      }

      syncChips();
      hideDetail();

      if (PL.renderPagination) {
        PL.renderPagination({
          container: $("irPager"),
          page: state.page,
          pageSize: state.pageSize,
          total: total,
          onPageChange: function (next) {
            state.page = next;
            refresh();
          },
          onPageSizeChange: function (next) {
            state.pageSize = next;
            state.page = 1;
            PL.savePageSize && PL.savePageSize("po.issueRisk.pageSize", next);
            refresh();
          }
        });
      }
    });
  }

  function hideDetail() {
    var detail = $("irDetailView");
    var list = $("irListView");
    if (detail) { detail.hidden = true; }
    if (list) { list.hidden = false; }
  }

  function showDetail(item) {
    var list = $("irListView");
    if (list) { list.hidden = true; }
    var detail = $("irDetailView");
    if (!detail || !item) { return; }
    detail.hidden = false;
    setText("irDetailId", item.displayId || item.id || "—");
    setText("irDetailTitle", item.title || "—");
    setText("irDetailProject", item.project || "—");
    setText("irDetailSeverity", item.severity || "—");
    var priBox = $("irDetailPriority");
    if (priBox) { priBox.innerHTML = priorityBadge(item.priority); }
    setText("irDetailStatus", item.status || "—");
    setText("irDetailHandler", item.handler || "—");
    setText("irDetailSubmitter", item.submitter || "—");
    setText("irDetailCreated", item.createdDate || "—");
    setText("irDetailPlan", item.planDate || "—");
    var ext = $("irDetailExternal");
    if (ext) { ext.setAttribute("href", item.url || "#"); }
  }

  function findItem(kind, id) {
    for (var i = 0; i < lastItems.length; i++) {
      var it = lastItems[i];
      if (String(it.kind) === String(kind) && String(it.id) === String(id)) { return it; }
    }
    return null;
  }

  function initFromStorage() {
    var ps = PL.loadPageSize && PL.loadPageSize("po.issueRisk.pageSize", 20, [10, 20, 50]);
    if (ps) { state.pageSize = ps; }
  }

  function initQuickChips() {
    var chips = document.querySelectorAll("#irQuickChips .header-quick-chip");
    chips.forEach(function (chip) {
      chip.addEventListener("click", function () {
        var loopKey = chip.getAttribute("data-loop");
        var overdueKey = chip.getAttribute("data-overdue");
        if (overdueKey === "true") {
          state.loop = "open";
          state.overdue = true;
        } else if (loopKey) {
          state.loop = loopKey;
          state.overdue = false;
        }
        state.page = 1;
        refresh();
      });
    });
  }

  function initTabs() {
    document.querySelectorAll(".po-issue-risk .category-tab").forEach(function (tab) {
      tab.addEventListener("click", function () {
        var kind = tab.getAttribute("data-kind");
        if (!kind || state.kind === kind) { return; }
        document.querySelectorAll(".po-issue-risk .category-tab").forEach(function (t) {
          t.classList.toggle("active", t === tab);
        });
        state.kind = kind;
        state.status = "";
        state.page = 1;
        refreshStatus();
        refresh();
      });
    });
  }

  function initToolbar() {
    var relSel = $("irRelation");
    if (relSel) {
      relSel.addEventListener("change", function () {
        state.relation = relSel.value;
        state.page = 1;
        refresh();
      });
    }

    var statusSel = $("irStatus");
    if (statusSel) {
      statusSel.addEventListener("change", function () {
        state.status = statusSel.value;
        state.page = 1;
        refresh();
      });
    }

    var projSel = $("irProject");
    if (projSel) {
      projSel.addEventListener("change", function () {
        state.project = parseInt(projSel.value, 10) || 0;
        state.page = 1;
        refresh();
      });
    }

    var kwInput = $("irKeyword");
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

    var resetBtn = $("irReset");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.relation = "allRelated";
        state.status = "";
        state.keyword = "";
        state.loop = "all";
        state.overdue = false;
        state.project = 0;
        state.page = 1;
        if (relSel) { relSel.value = "allRelated"; }
        if (projSel) { projSel.value = ""; }
        if (kwInput) { kwInput.value = ""; }
        refreshStatus();
        refresh();
      });
    }

    var retryBtn = $("irRetryBtn");
    if (retryBtn) {
      retryBtn.addEventListener("click", function () { refresh(); });
    }

    var closeBtn = $("irDetailClose");
    if (closeBtn) {
      closeBtn.addEventListener("click", function () { hideDetail(); });
    }
  }

  function initTableClicks() {
    var tbody = $("irTbody");
    if (!tbody) { return; }
    tbody.addEventListener("click", function (e) {
      var trigger = e.target.closest("[data-ir-detail]");
      if (!trigger) { return; }
      e.preventDefault();
      var raw = trigger.getAttribute("data-ir-detail") || "";
      var parts = raw.split("|");
      var item = findItem(parts[0], parts[1]);
      if (item) { showDetail(item); }
    });
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" || e.keyCode === 27) {
      var detail = $("irDetailView");
      if (detail && !detail.hidden) { hideDetail(); }
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    if (PL.createController) {
      controller = PL.createController({
        summaryEl: $("irSummary"),
        emptyEl: $("irEmpty"),
        errorEl: $("irError"),
        tbodyEl: $("irTbody")
      });
    }
    initFromStorage();
    initQuickChips();
    initTabs();
    initToolbar();
    initTableClicks();
    refreshStatus();
    refresh();
  });
})();