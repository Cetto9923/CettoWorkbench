/* =============================================================================
   文件: web/static/js/metrics-manage.js
   模块: 指标管理 (metrics) - /metrics/manage 交互脚本
   职责: 12 条指标定义列表 + 筛选 + 分页 + 单页内详情派生。
         状态/分类/周期/角色/单位全部走 metric 自定义中文 label，
         不输出 raw 英文状态；占位 "—" 由 backend 保证。
   依赖: personal-list.js (escapeHtml / createController / renderPagination /
         loadPageSize / savePageSize)
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
  var $ = function (id) { return document.getElementById(id); };

  // 与 backend service.go ValidCategories + MetricStatusLabel 同步。
  var STATUS_LABEL = { normal: "正常", warn: "关注", danger: "风险", unknown: "暂无数据" };
  var CATEGORY_LABEL = {
    "需求治理": "需求治理",
    "交付效率": "交付效率",
    "研发质量": "研发质量",
    "规范执行": "规范执行",
    "效能管理": "效能管理"
  };
  var ROLE_LABEL = { "PO": "PO", "SM": "SM", "PMO": "PMO", "测试": "测试" };
  // 与 backend CategoryColorClass 同步；CSS 把 .cat-* 着色到 semantic token。
  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };

  var PAGE_SIZE_KEY = "po.metrics.manage.pageSize";
  var PAGE_SIZE_ALLOWED = PL.PAGE_SIZE_OPTIONS || [10, 20, 50, 100];

  var state = {
    category: "",
    status: "",
    keyword: "",
    page: 1,
    pageSize: PL.loadPageSize ? PL.loadPageSize(PAGE_SIZE_KEY, 15, PAGE_SIZE_ALLOWED) : 15
  };

  var controller = null;
  var lastItems = [];
  var searchTimer = null;

  function buildUrl() {
    var p = new URLSearchParams();
    if (state.category) { p.set("category", state.category); }
    if (state.status) { p.set("status", state.status); }
    if (state.keyword) { p.set("keyword", state.keyword); }
    p.set("page", String(state.page));
    p.set("pageSize", String(state.pageSize));
    return "/metrics/api?" + p.toString();
  }

  function renderTargetCell(m) {
    var targetText = esc(m.target || "—");
    var warnText = esc(m.warningThreshold || "—");
    var dangerText = esc(m.dangerThreshold || "—");
    return '<div class="metric-target-cell">' +
      '<span class="metric-target-row" data-tone="t2">' +
      '<em>目标</em><span>' + targetText + "</span></span>" +
      '<span class="metric-target-row" data-tone="warn">' +
      '<em>关注</em><span>' + warnText + "</span></span>" +
      '<span class="metric-target-row" data-tone="danger">' +
      '<em>风险</em><span>' + dangerText + "</span></span>" +
      "</div>";
  }

  function renderRow(m, idx) {
    var categoryCls = CATEGORY_COLOR_CLASS[m.category] || "cat-default";
    var categoryText = CATEGORY_LABEL[m.category] || m.category || "—";
    var statusText = STATUS_LABEL[m.status] || "—";
    var roleText = ROLE_LABEL[m.ownerRole] || m.ownerRole || "—";
    var dataSource = esc(m.dataSource || m.source || "—");
    var period = esc(m.period || "—");
    var nameHtml = '<button type="button" class="table-title-link metric-name" ' +
      'data-metric-detail="' + esc(m.code) + '" ' +
      'title="' + esc(m.description || m.name) + '">' +
      esc(m.name) + "</button>" +
      '<small class="metric-code">' + esc(m.code) + "</small>";
    var statusHtml = '<span class="metric-status ' + esc(m.status || "unknown") + '">' + statusText + "</span>";
    var configTag = '<span class="state-tag">只读</span>';
    var actionBtn = '<button type="button" class="action-btn metric-action-detail" data-metric-detail="' + esc(m.code) + '">详情</button>';
    return "<tr data-row-idx=\"" + idx + "\">" +
      '<td class="metric-col-name">' + nameHtml + "</td>" +
      '<td><span class="metric-category ' + categoryCls + '">' + categoryText + "</span></td>" +
      "<td>" + dataSource + "</td>" +
      "<td>" + period + "</td>" +
      '<td class="metric-col-target">' + renderTargetCell(m) + "</td>" +
      "<td>" + roleText + "</td>" +
      "<td>" + statusHtml + "</td>" +
      "<td>" + configTag + "</td>" +
      '<td class="metric-col-opt">' + actionBtn + "</td>" +
      "</tr>";
  }

  function renderSummary(payload) {
    var s = (payload && payload.summary) || {};
    var totalItems = s.totalItems != null ? s.totalItems : "—";
    var withSnapshot = s.withSnapshot != null ? s.withSnapshot : "—";
    var abnormal = s.abnormal != null ? s.abnormal : "—";
    var catMap = s.categoryCount || {};
    var catCount = 0;
    for (var k in catMap) {
      if (Object.prototype.hasOwnProperty.call(catMap, k) && Number(catMap[k]) > 0) {
        catCount++;
      }
    }
    if ($("mSummaryTotal")) { $("mSummaryTotal").textContent = String(totalItems); }
    if ($("mSummarySnapshot")) { $("mSummarySnapshot").textContent = String(withSnapshot); }
    if ($("mSummaryAbnormal")) { $("mSummaryAbnormal").textContent = String(abnormal); }
    if ($("mSummaryCategories")) { $("mSummaryCategories").textContent = String(catCount); }
  }

  function hideDetail() {
    var detail = $("mDetailView");
    var list = $("mListView");
    if (detail) { detail.hidden = true; }
    if (list) { list.hidden = false; }
  }

  function showDetail(m) {
    var list = $("mListView");
    if (list) { list.hidden = true; }
    var detail = $("mDetailView");
    if (!detail || !m) { return; }
    detail.hidden = false;
    setText("mDetailCode", m.code || "—");
    setText("mDetailName", m.name || "—");
    setText("mDetailCategory", CATEGORY_LABEL[m.category] || m.category || "—");
    setText("mDetailOwnerRole", ROLE_LABEL[m.ownerRole] || m.ownerRole || "—");
    setText("mDetailValue", m.value || "—");
    setText("mDetailTarget", m.target || "—");
    setText("mDetailTargetText", m.target || "—");
    setText("mDetailWarning", m.warningThreshold || "—");
    setText("mDetailDanger", m.dangerThreshold || "—");
    setText("mDetailDirection", m.goodDirection === "up" ? "越大越好" : (m.goodDirection === "down" ? "越小越好" : "—"));
    setText("mDetailDataSource", m.dataSource || m.source || "—");
    setText("mDetailPeriod", m.period || "—");
    setText("mDetailUnit", m.unit || "—");
    setText("mDetailFormula", m.formula || "—");
    setText("mDetailLastCalc", m.lastCalcTime || "—");
    var statusEl = $("mDetailStatus");
    if (statusEl) {
      var cls = m.status || "unknown";
      statusEl.innerHTML = '<span class="metric-status ' + esc(cls) + '">' + esc(STATUS_LABEL[cls] || "—") + "</span>";
    }
    setText("mDetailDescription", m.description || "—");
  }

  function setText(id, value) {
    var el = $(id);
    if (el) { el.textContent = value; }
  }

  function findItem(code) {
    for (var i = 0; i < lastItems.length; i++) {
      if (String(lastItems[i].code) === String(code)) { return lastItems[i]; }
    }
    return null;
  }

  function refresh() {
    if (!controller) { return; }
    controller.fetch(buildUrl(), state, function (payload) {
      lastItems = (payload && Array.isArray(payload.items)) ? payload.items : [];
      renderSummary(payload);
      var total = (payload && typeof payload.total === "number") ? payload.total : 0;
      var tbody = $("mTbody");
      if (tbody) { tbody.innerHTML = lastItems.map(function (m, i) { return renderRow(m, i); }).join(""); }
      if ($("mSummary")) {
        $("mSummary").textContent = "共 " + total + " 条指标";
      }
      hideDetail();
      if (PL.renderPagination) {
        PL.renderPagination({
          container: $("mPager"),
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
            if (PL.savePageSize) { PL.savePageSize(PAGE_SIZE_KEY, next); }
            refresh();
          }
        });
      }
    });
  }

  function initToolbar() {
    var cat = $("mCategory");
    if (cat) {
      cat.addEventListener("change", function () {
        state.category = cat.value;
        state.page = 1;
        refresh();
      });
    }
    var status = $("mStatus");
    if (status) {
      status.addEventListener("change", function () {
        state.status = status.value;
        state.page = 1;
        refresh();
      });
    }
    var kw = $("mKeyword");
    if (kw) {
      kw.addEventListener("input", function () {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(function () {
          state.keyword = (kw.value || "").trim();
          state.page = 1;
          refresh();
        }, 300);
      });
    }
    var resetBtn = $("mReset");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.category = "";
        state.status = "";
        state.keyword = "";
        state.page = 1;
        if (cat) { cat.value = ""; }
        if (status) { status.value = ""; }
        if (kw) { kw.value = ""; }
        refresh();
      });
    }
    var retryBtn = $("mRetryBtn");
    if (retryBtn) {
      retryBtn.addEventListener("click", function () { refresh(); });
    }
    var closeBtn = $("mDetailClose");
    if (closeBtn) {
      closeBtn.addEventListener("click", function () { hideDetail(); });
    }
  }

  function initTableClicks() {
    var tbody = $("mTbody");
    if (!tbody) { return; }
    tbody.addEventListener("click", function (e) {
      var trigger = e.target.closest("[data-metric-detail]");
      if (!trigger) { return; }
      e.preventDefault();
      var code = trigger.getAttribute("data-metric-detail");
      var item = findItem(code);
      if (item) { showDetail(item); }
    });
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" || e.keyCode === 27) {
      var detail = $("mDetailView");
      if (detail && !detail.hidden) { hideDetail(); }
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    if (PL.createController) {
      controller = PL.createController({
        summaryEl: $("mSummary"),
        emptyEl: $("mEmpty"),
        errorEl: $("mError"),
        tbodyEl: $("mTbody")
      });
    }
    initToolbar();
    initTableClicks();
    refresh();
  });
})();
