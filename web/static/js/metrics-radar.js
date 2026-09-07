/* =============================================================================
  文件: web/static/js/metrics-radar.js
  模块: 指标管理 (metrics) - /metrics/radar 交互脚本
  职责: 5 分类 KPI 卡渲染 + 异常指标分组表格 + 趋势 sparkline (inline SVG)。
        状态/分类全部走 metric 自定义中文 label，不输出 raw 英文状态；
        占位 "—" 由 backend 保证。
  依赖: personal-list.js (escapeHtml / createController)
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

  // 与 backend service.go MetricStatusLabel / CategoryColorClass 同步。
  var STATUS_LABEL = { normal: "正常", warn: "关注", danger: "风险", unknown: "暂无数据" };
  var CATEGORY_LABEL = {
    "需求治理": "需求治理",
    "交付效率": "交付效率",
    "研发质量": "研发质量",
    "规范执行": "规范执行",
    "效能管理": "效能管理"
  };
  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };
  var FIXED_CATEGORIES = ["需求治理", "交付效率", "研发质量", "规范执行", "效能管理"];

  var controller = null;
  var lastPayload = null;

  // ---------------------------------------------------------------------------
  // Sparkline: 内联 SVG polyline，90×26，无新依赖。
  // 数据约定：values 是 0~1 之间的归一化数组；length<2 时画一条水平基线。
  // toneClass 控制 stroke 颜色（success / warning / danger / muted）。
  // ---------------------------------------------------------------------------
  function sparkline(values, toneClass) {
    var w = 90, h = 26, pad = 2;
    var stroke = toneClass || "tone-muted";
    if (!Array.isArray(values) || values.length < 2) {
      var midY = h / 2;
      return '<svg class="metric-sparkline ' + stroke + '" viewBox="0 0 ' + w + " " + h +
        '" preserveAspectRatio="none" aria-hidden="true">' +
        '<line x1="' + pad + '" y1="' + midY + '" x2="' + (w - pad) + '" y2="' + midY + '"/>' +
        "</svg>";
    }
    var stepX = (w - pad * 2) / (values.length - 1);
    var pts = [];
    for (var i = 0; i < values.length; i++) {
      var v = Number(values[i]);
      if (!isFinite(v)) { v = 0; }
      if (v < 0) { v = 0; }
      if (v > 1) { v = 1; }
      var x = pad + stepX * i;
      var y = pad + (1 - v) * (h - pad * 2);
      pts.push(x.toFixed(2) + "," + y.toFixed(2));
    }
    return '<svg class="metric-sparkline ' + stroke + '" viewBox="0 0 ' + w + " " + h +
      '" preserveAspectRatio="none" aria-hidden="true">' +
      '<polyline points="' + pts.join(" ") + '" fill="none"/>' +
      "</svg>";
  }

  function severityTone(severity) {
    switch (severity) {
      case "normal": return "tone-success";
      case "warn":   return "tone-warning";
      case "danger": return "tone-danger";
      default:       return "tone-muted";
    }
  }

  function statusTone(status) {
    switch (status) {
      case "normal": return "tone-success";
      case "warn":   return "tone-warning";
      case "danger": return "tone-danger";
      default:       return "tone-muted";
    }
  }

  // ---------------------------------------------------------------------------
  // KPI 卡渲染：5 张紧凑卡，每张含分类名 / 总数 / 得分 / 三态统计 / 主要风险
  // ---------------------------------------------------------------------------
  function renderKpiCards(categories) {
    var host = document.querySelector(".metric-radar-summary");
    if (!host) { return; }
    if (!Array.isArray(categories) || categories.length === 0) {
      host.innerHTML = '<span class="state-placeholder">当前条件下暂无指标数据</span>';
      return;
    }
    var html = categories.map(function (c) {
      var cat = c.category || "";
      var cls = CATEGORY_COLOR_CLASS[cat] || "cat-default";
      var sev = c.severity || "unknown";
      var scoreText = c.score || "—";
      var total = c.totalCount != null ? c.totalCount : 0;
      var danger = c.dangerCount != null ? c.dangerCount : 0;
      var warn = c.warnCount != null ? c.warnCount : 0;
      var normal = c.normalCount != null ? c.normalCount : 0;
      var unknown = c.unknownCount != null ? c.unknownCount : 0;
      var sevLabel = STATUS_LABEL[sev] || "—";
      var sevTone = severityTone(sev);
      var top = "";
      if (danger > 0 && c.topRiskCode) {
        top = '<a class="metric-radar-toprisk" href="#" data-radar-jump="' + esc(c.topRiskCode) + '">' +
          esc(c.topRiskCode) + " · " + esc(c.topRiskName || "") + "</a>";
      } else {
        top = '<span class="metric-radar-toprisk muted">—</span>';
      }
      return '<article class="metric-radar-card ' + esc(sevTone) + '" data-radar-category="' + esc(cat) + '">' +
        '<header class="metric-radar-card-head">' +
        '<span class="metric-category ' + esc(cls) + '">' + esc(CATEGORY_LABEL[cat] || cat) + "</span>" +
        '<span class="metric-radar-score" data-severity="' + esc(sev) + '">' +
        '<em>得分</em><strong>' + esc(scoreText) + "</strong></span>" +
        "</header>" +
        '<div class="metric-radar-counts">' +
        '<span class="metric-radar-count tone-success"><em>正常</em><strong>' + normal + "</strong></span>" +
        '<span class="metric-radar-count tone-warning"><em>关注</em><strong>' + warn + "</strong></span>" +
        '<span class="metric-radar-count tone-danger"><em>风险</em><strong>' + danger + "</strong></span>" +
        "</div>" +
        '<footer class="metric-radar-card-foot">' +
        '<span class="metric-radar-status metric-status ' + esc(sev) + '">' + esc(sevLabel) + "</span>" +
        '<span class="metric-radar-total">共 <strong>' + total + "</strong> 项" +
        (unknown > 0 ? ' · 暂无数据 <strong>' + unknown + "</strong>" : "") + "</span>" +
        "</footer>" +
        '<div class="metric-radar-toprisk-row">主要风险：' + top + "</div>" +
        "</article>";
    }).join("");
    host.innerHTML = html;
  }

  // ---------------------------------------------------------------------------
  // 分组表格：每个分类一组，组内显示其 warn / danger 指标
  // ---------------------------------------------------------------------------
  function renderGroups(abnormalItems) {
    var host = $("radarGroups");
    if (!host) { return; }
    if (!Array.isArray(abnormalItems) || abnormalItems.length === 0) {
      host.innerHTML = '<p class="state-placeholder">当前条件下暂无指标数据</p>';
      return;
    }
    // 按分类分组（保持 FIXED_CATEGORIES 顺序）
    var groups = {};
    FIXED_CATEGORIES.forEach(function (c) { groups[c] = []; });
    abnormalItems.forEach(function (m) {
      if (!groups[m.category]) { groups[m.category] = []; }
      groups[m.category].push(m);
    });

    var html = "";
    FIXED_CATEGORIES.forEach(function (cat) {
      var rows = groups[cat];
      if (!rows || rows.length === 0) { return; }
      var catCls = CATEGORY_COLOR_CLASS[cat] || "cat-default";
      var body = rows.map(function (m) { return renderRow(m); }).join("");
      html += '<section class="metric-radar-group" data-radar-group="' + esc(cat) + '">' +
        '<header class="metric-radar-group-head">' +
        '<span class="metric-category ' + esc(catCls) + '">' + esc(CATEGORY_LABEL[cat] || cat) + "</span>" +
        '<span class="metric-radar-group-count">异常 <strong>' + rows.length + "</strong> 项</span>" +
        "</header>" +
        '<div class="table-scroll-container">' +
        '<table class="workspace-table metric-radar-table">' +
        "<thead><tr>" +
        "<th class=\"metric-col-name\">指标名</th>" +
        "<th>当前值</th>" +
        "<th>目标</th>" +
        "<th>状态</th>" +
        "<th>趋势</th>" +
        "<th>环比</th>" +
        "<th class=\"metric-col-opt\">操作</th>" +
        "</tr></thead>" +
        "<tbody>" + body + "</tbody></table></div>" +
        "</section>";
    });
    if (!html) {
      host.innerHTML = '<p class="state-placeholder">当前条件下暂无指标数据</p>';
      return;
    }
    host.innerHTML = html;
  }

  // ---------------------------------------------------------------------------
  // 单行渲染：指标名 / 当前值 / 目标 / 状态 / sparkline / 环比 / 操作
  // ---------------------------------------------------------------------------
  function renderRow(m) {
    var status = m.status || "unknown";
    var statusCls = statusTone(status);
    var statusLabel = STATUS_LABEL[status] || "—";
    // sparkline 数据：从 MetricItem.sparkline 读取（可选）；无值时画基线
    var sparkValues = Array.isArray(m.sparkline) ? m.sparkline : null;
    var spark = sparkline(sparkValues, statusCls);
    // 环比：同上，sparklineTrend 字段（"up"/"down"/"flat"）+ 文本
    var trendText = "—";
    var trendCls = "tone-muted";
    if (m.trendDirection === "up")   { trendCls = "tone-success"; trendText = "↑ " + (m.trendText || "上升"); }
    else if (m.trendDirection === "down") { trendCls = "tone-danger"; trendText = "↓ " + (m.trendText || "下降"); }
    else if (m.trendDirection === "flat") { trendCls = "tone-muted"; trendText = m.trendText || "持平"; }

    var targetHtml = '<span class="metric-target-cell">' +
      '<span class="metric-target-row" data-tone="t2"><em>目标</em><span>' + esc(m.target || "—") + "</span></span>" +
      (m.warningThreshold && m.warningThreshold !== "—" ?
        '<span class="metric-target-row" data-tone="warn"><em>关注</em><span>' + esc(m.warningThreshold) + "</span></span>" : "") +
      (m.dangerThreshold && m.dangerThreshold !== "—" ?
        '<span class="metric-target-row" data-tone="danger"><em>风险</em><span>' + esc(m.dangerThreshold) + "</span></span>" : "") +
      "</span>";

    var actionHtml = '<button type="button" class="action-btn metric-radar-jump" data-metric-jump="' + esc(m.code) + '">详情</button>';

    return "<tr>" +
      '<td class="metric-col-name">' +
      '<strong class="metric-name" title="' + esc(m.description || m.name || "") + '">' + esc(m.name || "—") + "</strong>" +
      '<small class="metric-code">' + esc(m.code || "") + "</small></td>" +
      '<td class="metric-radar-value">' + esc(m.value || "—") + "</td>" +
      "<td>" + targetHtml + "</td>" +
      '<td><span class="metric-status ' + esc(status) + '">' + esc(statusLabel) + "</span></td>" +
      '<td class="metric-radar-spark">' + spark + "</td>" +
      '<td class="metric-radar-trend ' + esc(trendCls) + '">' + esc(trendText) + "</td>" +
      '<td class="metric-col-opt">' + actionHtml + "</td>" +
      "</tr>";
  }

  function renderMeta(payload) {
    var totalAbnormal = payload && Array.isArray(payload.abnormalItems) ? payload.abnormalItems.length : 0;
    var generatedAt = (payload && payload.generatedAt) || "—";
    if ($("radarAbnormalCount")) { $("radarAbnormalCount").textContent = String(totalAbnormal); }
    if ($("radarGeneratedAt"))   { $("radarGeneratedAt").textContent = generatedAt; }
    if ($("radarSummary"))       { $("radarSummary").textContent = "5 分类指标 · 异常 " + totalAbnormal + " 项"; }
  }

  function renderAll(payload) {
    lastPayload = payload || {};
    renderKpiCards(lastPayload.categories || []);
    renderGroups(lastPayload.abnormalItems || []);
    renderMeta(lastPayload);
  }

  // ---------------------------------------------------------------------------
  // 表格内 "详情" / KPI 卡 "主要风险" 点击 → 跳转 /metrics/manage?code=xxx
  // ---------------------------------------------------------------------------
  function bindActions() {
    var groups = $("radarGroups");
    if (groups) {
      groups.addEventListener("click", function (e) {
        var trigger = e.target.closest("[data-metric-jump]");
        if (!trigger) { return; }
        e.preventDefault();
        var code = trigger.getAttribute("data-metric-jump");
        if (code) { window.location.href = "/metrics/manage?code=" + encodeURIComponent(code); }
      });
    }
    var summary = document.querySelector(".metric-radar-summary");
    if (summary) {
      summary.addEventListener("click", function (e) {
        var trigger = e.target.closest("[data-radar-jump]");
        if (!trigger) { return; }
        e.preventDefault();
        var code = trigger.getAttribute("data-radar-jump");
        if (code) { window.location.href = "/metrics/manage?code=" + encodeURIComponent(code); }
      });
    }
  }

  function refresh() {
    if (!controller) { return; }
    controller.fetch("/metrics/radar/data", {}, function (payload) {
      renderAll(payload);
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    if (PL.createController) {
      controller = PL.createController({
        summaryEl: $("radarSummary"),
        emptyEl: null,
        errorEl: $("radarError"),
        tbodyEl: $("radarGroups")
      });
    }
    var retry = $("radarRetryBtn");
    if (retry) { retry.addEventListener("click", function () { refresh(); }); }
    bindActions();
    refresh();
  });
})();
