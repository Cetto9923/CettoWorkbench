/* =============================================================================
   文件: web/static/js/metrics-manage.js
   模块: 指标管理 (metrics) - /metrics/manage
   职责: 指标元数据只读展示。指标定义（21 条）的唯一事实源在后端
         /metrics/api（internal/module/metrics/catalog.go），本页不维护
         第二套指标字典，也不做 localStorage 假持久化。
   ============================================================================= */
(function () {
  "use strict";

  var PL = window.PersonalList || {};
  var esc = window.escapeHtml;
  var $ = function (id) { return document.getElementById(id); };

  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };

  var STATUS_LABEL = { normal: "正常", warn: "关注", danger: "风险", unknown: "暂无数据" };

  var metricsList = [];

  var state = {
    category: "",
    role: "",
    status: "enabled",
    keyword: ""
  };

  var searchTimer = null;

  // 从后端 metadata API 拉取指标目录（唯一 SSOT）。
  function loadMetrics(cb) {
    fetch("/metrics/api?page=1&pageSize=100")
      .then(function (r) { return r.json(); })
      .then(function (payload) {
        metricsList = (payload && Array.isArray(payload.items)) ? payload.items : [];
        if (typeof cb === "function") cb();
      })
      .catch(function () {
        metricsList = [];
        if (typeof cb === "function") cb();
      });
  }

  function renderSummary() {
    var total = metricsList.length;
    var enabledCount = metricsList.filter(function (m) { return m.enabled === true; }).length;
    var targetCount = metricsList.filter(function (m) { return m.target && m.target !== "—"; }).length;
    var dangerCount = metricsList.filter(function (m) { return m.dangerThreshold && m.dangerThreshold !== "—"; }).length;
    var catMap = {};
    metricsList.forEach(function (m) { if (m.category) catMap[m.category] = true; });

    if ($("mSummaryTotal")) $("mSummaryTotal").textContent = String(total);
    if ($("mSummaryEnabled")) $("mSummaryEnabled").textContent = String(enabledCount);
    if ($("mSummaryWarnThreshold")) $("mSummaryWarnThreshold").textContent = String(targetCount);
    if ($("mSummaryDangerThreshold")) $("mSummaryDangerThreshold").textContent = String(dangerCount);
    if ($("mSummaryCategories")) $("mSummaryCategories").textContent = String(Object.keys(catMap).length);
  }

  function renderRow(m) {
    var categoryCls = CATEGORY_COLOR_CLASS[m.category] || "cat-default";
    var status = m.status || "unknown";
    var statusText = STATUS_LABEL[status] || "—";

    var nameHtml = '<strong class="metric-name">' + esc(m.name) + '</strong>' +
      '<small class="metric-code">' + esc(m.code) + (m.unit ? ' · ' + esc(m.unit) : '') + '</small>';

    var targetHtml = '<span class="metric-target-val">' + esc(m.target || "—") + '</span>';

    var warnText = esc(m.warningThreshold || "—");
    var dangerText = esc(m.dangerThreshold || "—");
    var thresholdHtml = '<div class="metric-target-cell">' +
      '<span class="metric-target-row" data-tone="warn"><em>关注</em><span>' + warnText + '</span></span>' +
      '<span class="metric-target-row" data-tone="danger"><em>风险</em><span>' + dangerText + '</span></span>' +
      '</div>';

    var dirText = m.goodDirection === "up"
      ? '<span class="dir-tag dir-up">越大越好 ↑</span>'
      : '<span class="dir-tag dir-down">越小越好 ↓</span>';

    var valueText = (m.value && m.value !== "—") ? esc(m.value) : '<span class="state-tag">暂无数据</span>';
    var statusBadge = '<span class="metric-status ' + status + '">' + esc(statusText) + '</span>';

    return '<tr data-code="' + esc(m.code) + '">' +
      '<td class="metric-col-name">' + nameHtml + '</td>' +
      '<td><span class="metric-category ' + categoryCls + '">' + esc(m.category) + '</span></td>' +
      '<td class="metric-col-source" title="' + esc(m.dataSource || "") + '">' + esc(m.dataSource || "—") + '</td>' +
      '<td class="metric-col-target-val">' + targetHtml + '</td>' +
      '<td class="metric-col-target">' + thresholdHtml + '</td>' +
      '<td><span class="state-tag">' + esc(m.ownerRole || "—") + '</span></td>' +
      '<td class="metric-col-dir">' + dirText + '</td>' +
      '<td class="metric-col-value">' + valueText + '</td>' +
      '<td>' + statusBadge + '</td>' +
      '</tr>';
  }

  function applyFilterAndRender() {
    var kw = (state.keyword || "").toLowerCase();
    var filtered = metricsList.filter(function (m) {
      if (state.category && m.category !== state.category) return false;
      if (state.role && m.ownerRole !== state.role) return false;
      if (state.status === "enabled" && m.enabled === false) return false;
      if (state.status === "disabled" && m.enabled !== false) return false;
      if (kw) {
        var str = (m.name + " " + m.code + " " + m.dataSource + " " + m.formula + " " + m.ownerRole).toLowerCase();
        if (str.indexOf(kw) === -1) return false;
      }
      return true;
    });

    renderSummary();

    var tbody = $("mTbody");
    if (tbody) {
      if (filtered.length === 0) {
        tbody.innerHTML = '<tr><td colspan="9" class="state-placeholder">当前筛选条件下暂无指标</td></tr>';
      } else {
        tbody.innerHTML = filtered.map(renderRow).join("");
      }
    }

    if ($("mSummary")) {
      $("mSummary").textContent = "共 " + filtered.length + " 条指标定义 (总数 " + metricsList.length + " 项)";
    }
  }

  function render() {
    applyFilterAndRender();
  }

  function initEvents() {
    $("mCategory").addEventListener("change", function () {
      state.category = this.value;
      render();
    });

    $("mRole").addEventListener("change", function () {
      state.role = this.value;
      render();
    });

    $("mState").addEventListener("change", function () {
      state.status = this.value;
      render();
    });

    $("mKeyword").addEventListener("input", function () {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(function () {
        state.keyword = $("mKeyword").value.trim();
        render();
      }, 300);
    });

    $("mReset").addEventListener("click", function () {
      state.category = "";
      state.role = "";
      state.status = "enabled";
      state.keyword = "";
      $("mCategory").value = "";
      $("mRole").value = "";
      $("mState").value = "enabled";
      $("mKeyword").value = "";
      render();
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    initEvents();
    loadMetrics(render);
  });
})();
