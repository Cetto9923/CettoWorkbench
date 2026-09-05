/* =============================================================================
   文件: web/static/js/po/done.js
   模块: PO 个人工作台 - 我的已办交互脚本
   职责: 绑定已办分类 Tab、时间范围 Chips、高级筛选、列表渲染与分页
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var $ = function (id) { return document.getElementById(id); };

  var state = {
    tab: "all",
    timeRange: "all",
    objectType: "",
    action: "all",
    keyword: "",
    result: "all",
    page: 1,
    pageSize: 20
  };

  var RANGE_FIELD_MAP = {
    all: "all",
    today: "today",
    "7d": "d7",
    week: "thisWeek",
    "30d": "d30",
    month: "thisMonth",
    quarter: "thisQuarter"
  };

  var SCENE_OBJECT_TYPES = {
    all: [
      { value: "", label: "全部对象" },
      { value: "demand", label: "业务需求" },
      { value: "story", label: "研发需求" },
      { value: "task", label: "任务" },
      { value: "bug", label: "Bug" },
      { value: "testtask", label: "测试单" },
      { value: "issue", label: "问题" },
      { value: "risk", label: "风险" },
      { value: "approval", label: "审批流程" }
    ],
    approval: [{ value: "approval", label: "审批流程" }],
    demand: [{ value: "demand", label: "业务需求" }],
    execution: [{ value: "task", label: "任务" }, { value: "story", label: "研发需求" }],
    quality: [{ value: "bug", label: "Bug" }, { value: "testtask", label: "测试单" }],
    risks: [{ value: "issue", label: "问题" }, { value: "risk", label: "风险" }]
  };

  function buildUrl() {
    var params = new URLSearchParams();
    params.set("tab", state.tab);
    params.set("timeRange", state.timeRange);
    if (state.objectType) { params.set("objectType", state.objectType); }
    if (state.action !== "all") { params.set("action", state.action); }
    if (state.keyword) { params.set("keyword", state.keyword); }
    if (state.result !== "all") { params.set("result", state.result); }
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    return "/done/items?" + params.toString();
  }

  function renderRow(item) {
    var displayId = item.displayId || (item.objectType + "/" + item.objectId);
    var idCell = item.url
      ? '<a class="table-id-link" href="' + esc(item.url) + '" rel="noopener noreferrer" title="查看对象">' + esc(displayId) + "</a>"
      : esc(displayId);
    var titleCell = item.url
      ? '<a class="table-title-link" href="' + esc(item.url) + '" rel="noopener noreferrer" title="查看对象">' + esc(item.objectName || "—") + "</a>"
      : esc(item.objectName || "—");

    var resultLabels = {
      activated: "已激活", done: "处理成功", submitted: "已提交", approved: "已通过",
      rejected: "已驳回", closed: "已关闭", verified: "已验收", resolved: "已解决", returned: "已退回"
    };
    var result = resultLabels[item.result] || "已处理";
    var resultTone = ["approved", "done", "verified", "resolved"].indexOf(item.result) >= 0
      ? " success"
      : (["rejected", "returned"].indexOf(item.result) >= 0 ? " danger" : "");

    return "<tr>" +
      '<td class="done-col-item" title="' + esc(item.objectName || "") + '">' +
        '<div class="done-item-title">' + titleCell + "</div>" +
        '<div class="done-item-id">' + idCell + "</div>" +
      "</td>" +
      '<td class="done-col-type"><span class="type-tag">' + esc(item.objectTypeLabel || item.objectType || "—") + "</span></td>" +
      '<td class="done-col-action">' + esc(item.action || "已处理") + "</td>" +
      '<td class="done-col-result"><span class="result-tag' + resultTone + '">' + esc(result) + "</span></td>" +
      '<td class="done-col-date">' + esc(item.date || "—") + "</td>" +
      '<td class="done-col-opt">' + (item.url ? '<a class="table-action-btn" href="' + esc(item.url) + '" rel="noopener noreferrer">查看记录</a>' : "—") + "</td>" +
      "</tr>";
  }

  function renderSummary(summary) {
    if (!summary) { return; }
    var totalEl = $("countRangeAll");
    if (totalEl) { totalEl.textContent = summary.all != null ? summary.all : 0; }
    var curEl = $("countRangeCurrent");
    if (curEl) {
      var field = RANGE_FIELD_MAP[state.timeRange] || "all";
      curEl.textContent = summary[field] != null ? summary[field] : 0;
    }
  }

  function syncObjectTypeOptions(tab) {
    var select = $("doneObjectType");
    if (!select) { return; }
    var list = SCENE_OBJECT_TYPES[tab] || SCENE_OBJECT_TYPES.all;
    select.innerHTML = list.map(function (opt) {
      return '<option value="' + esc(opt.value) + '">' + esc(opt.label) + "</option>";
    }).join("");
    state.objectType = list[0] ? list[0].value : "";
  }

  var controller = null;

  function loadData() {
    if (!controller) { return; }
    controller.fetch(buildUrl(), { method: "GET" }, function (payload) {
      var items = (payload && Array.isArray(payload.items)) ? payload.items : [];
      var total = (payload && typeof payload.total === "number") ? payload.total : items.length;

      var tbody = $("doneTbody");
      if (tbody) {
        tbody.innerHTML = items.map(renderRow).join("");
      }
      if ($("doneSummary")) { $("doneSummary").textContent = "共 " + total + " 条已办记录"; }
      renderSummary(payload && payload.summary);

      if (window.PersonalList) {
        window.PersonalList.renderPagination({
          container: $("donePagination"),
          page: state.page,
          pageSize: state.pageSize,
          total: total,
          onPageChange: function (p) { state.page = p; loadData(); },
          onPageSizeChange: function (s) { state.pageSize = s; state.page = 1; loadData(); }
        });
      }
    });
  }

  function initTabs() {
    var tabs = document.querySelectorAll(".po-done .category-tab");
    tabs.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var tab = btn.getAttribute("data-tab") || "all";
        if (state.tab === tab) { return; }
        tabs.forEach(function (b) {
          b.classList.remove("active");
          b.removeAttribute("aria-current");
        });
        btn.classList.add("active");
        btn.setAttribute("aria-current", "page");
        state.tab = tab;
        state.page = 1;
        state.action = "all";
        state.result = "all";
        if ($("doneAction")) { $("doneAction").value = "all"; }
        if ($("doneResult")) { $("doneResult").value = "all"; }
        syncObjectTypeOptions(tab);
        loadData();
      });
    });
  }

  function initTimeChips() {
    var chips = document.querySelectorAll("#doneTimeChips .header-quick-chip");
    chips.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var range = btn.getAttribute("data-range") || "all";
        if (state.timeRange === range) { return; }
        chips.forEach(function (b) {
          b.classList.remove("active");
          b.setAttribute("aria-pressed", "false");
        });
        btn.classList.add("active");
        btn.setAttribute("aria-pressed", "true");
        state.timeRange = range;
        state.page = 1;
        loadData();
      });
    });
  }

  function initToolbar() {
    var kwInput = $("doneKeyword");
    if (kwInput) {
      var timer = null;
      kwInput.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(function () {
          state.keyword = (kwInput.value || "").trim();
          state.page = 1;
          loadData();
        }, 300);
      });
    }

    var objSel = $("doneObjectType");
    if (objSel) {
      objSel.addEventListener("change", function () {
        state.objectType = objSel.value;
        state.page = 1;
        loadData();
      });
    }

    var actSel = $("doneAction");
    if (actSel) {
      actSel.addEventListener("change", function () {
        state.action = actSel.value;
        state.page = 1;
        loadData();
      });
    }

    var resSel = $("doneResult");
    if (resSel) {
      resSel.addEventListener("change", function () {
        state.result = resSel.value;
        state.page = 1;
        loadData();
      });
    }

    var resetBtn = $("doneResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.tab = "all";
        state.timeRange = "all";
        state.objectType = "";
        state.action = "all";
        state.keyword = "";
        state.result = "all";
        state.page = 1;

        document.querySelectorAll(".po-done .category-tab").forEach(function (b) {
          var isAll = (b.getAttribute("data-tab") || "all") === "all";
          b.classList.toggle("active", isAll);
          if (isAll) {
            b.setAttribute("aria-current", "page");
          } else {
            b.removeAttribute("aria-current");
          }
        });
        document.querySelectorAll("#doneTimeChips .header-quick-chip").forEach(function (b) {
          var isAll = (b.getAttribute("data-range") || "all") === "all";
          b.classList.toggle("active", isAll);
          b.setAttribute("aria-pressed", isAll ? "true" : "false");
        });

        if (kwInput) { kwInput.value = ""; }
        if (actSel) { actSel.value = "all"; }
        if (resSel) { resSel.value = "all"; }
        syncObjectTypeOptions("all");
        loadData();
      });
    }

    var retryBtn = $("doneRetryBtn");
    if (retryBtn) {
      retryBtn.addEventListener("click", function () { loadData(); });
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    if (window.PersonalList) {
      controller = window.PersonalList.createController({
        summaryEl: $("doneSummary"),
        emptyEl: $("doneEmpty"),
        errorEl: $("doneError"),
        tbodyEl: $("doneTbody"),
        errorColspan: 6,
        onError: function () {
          var totalEl = $("countRangeAll");
          if (totalEl) { totalEl.textContent = "—"; }
          var curEl = $("countRangeCurrent");
          if (curEl) { curEl.textContent = "—"; }
        }
      });
    }

    syncObjectTypeOptions("all");
    initTabs();
    initTimeChips();
    initToolbar();
    loadData();
  });
})();
