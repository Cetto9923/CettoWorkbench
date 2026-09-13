/* =============================================================================
   文件: web/static/js/metrics-radar.js
   模块: 指标雷达 (metrics) - 质效监控大屏
   职责: 从后端 /metrics/api 读取指标元数据，按 5 分类聚合雷达盘 / 健康 / 矩阵；
         未接入外部源的指标显示「暂无数据」且不参与评分、均值、排名。
   ============================================================================= */
(function () {
  "use strict";

  var PL = window.PersonalList || {};
  var esc = window.escapeHtml;
  var $ = function (id) { return document.getElementById(id); };

  var CATEGORY_NAMES = ["需求治理", "交付效率", "研发质量", "规范执行", "效能管理"];

  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };

  var STATUS_LABEL = { normal: "正常达标", warn: "预警关注", danger: "高危风险", unknown: "暂无数据" };

  var allMetrics = [];
  var categoryScores = {};
  var state = { matrixCategory: "", selectedCode: "" };

  function parseNum(value) {
    if (!value || value === "—") return NaN;
    var n = parseFloat(String(value).replace(/[^0-9.\-]/g, ""));
    return isNaN(n) ? NaN : n;
  }

  // 从后端 metadata API 拉取指标目录（唯一 SSOT）。
  function loadMetrics(cb) {
    fetch("/metrics/api?page=1&pageSize=100")
      .then(function (r) { return r.json(); })
      .then(function (payload) {
        allMetrics = (payload && Array.isArray(payload.items)) ? payload.items : [];
        if (typeof cb === "function") cb();
      })
      .catch(function () {
        allMetrics = [];
        if (typeof cb === "function") cb();
      });
  }

  // 按 5 分类聚合：normal/warn/danger 参与评分，unknown（unavailable）不参与。
  function computeCategoryScores(items) {
    var catMap = {};
    CATEGORY_NAMES.forEach(function (c) {
      catMap[c] = { normal: 0, warn: 0, danger: 0, unknown: 0, total: 0 };
    });
    items.forEach(function (m) {
      var bucket = catMap[m.category];
      if (!bucket) return;
      bucket.total++;
      var st = m.status || "unknown";
      if (st === "normal") bucket.normal++;
      else if (st === "warn") bucket.warn++;
      else if (st === "danger") bucket.danger++;
      else bucket.unknown++;
    });

    categoryScores = {};
    var totalNormal = 0, totalWarn = 0, totalDanger = 0, scorableTotal = 0;
    CATEGORY_NAMES.forEach(function (c) {
      var d = catMap[c];
      var scorable = d.normal + d.warn + d.danger;
      var score = scorable > 0 ? Math.round(((d.normal * 1.0 + d.warn * 0.5) / scorable) * 100) : 0;
      categoryScores[c] = {
        score: score,
        severity: scorable === 0 ? "unknown" : (score >= 80 ? "normal" : (score >= 60 ? "warn" : "danger")),
        normal: d.normal, warn: d.warn, danger: d.danger, unknown: d.unknown, total: d.total
      };
      totalNormal += d.normal;
      totalWarn += d.warn;
      totalDanger += d.danger;
      scorableTotal += scorable;
    });

    var overallScore = scorableTotal > 0 ? Math.round((totalNormal * 1.0 + totalWarn * 0.5) / scorableTotal * 100) : 0;

    if ($("kpiScore")) $("kpiScore").textContent = overallScore;
    if ($("kpiScoreRank")) {
      $("kpiScoreRank").textContent = scorableTotal === 0 ? "暂无数据" : (overallScore >= 85 ? "质效优秀 ↑" : (overallScore >= 75 ? "质效良好" : "质效警示 ↓"));
    }
    if ($("kpiNormalCount")) $("kpiNormalCount").textContent = totalNormal;
    if ($("kpiNormalRate")) {
      var rate = scorableTotal > 0 ? Math.round(totalNormal / scorableTotal * 100) : 0;
      $("kpiNormalRate").textContent = "达标率 " + rate + "%";
      if ($("kpiNormalBar")) $("kpiNormalBar").style.width = rate + "%";
    }
    if ($("kpiWarnCount")) $("kpiWarnCount").textContent = totalWarn;
    if ($("kpiDangerCount")) $("kpiDangerCount").textContent = totalDanger;
    if ($("kpiDangerHint")) {
      $("kpiDangerHint").innerHTML = totalDanger > 0
        ? '<i class="fas fa-triangle-exclamation"></i> ' + totalDanger + ' 项高危需治理'
        : '<i class="fas fa-check"></i> 运行平稳 · 暂无阻断';
    }
    if ($("kpiCurrentTarget")) $("kpiCurrentTarget").textContent = "全量指标 (实时快照)";
    if ($("kpiSnapshotTime")) $("kpiSnapshotTime").innerHTML = '<i class="fas fa-database"></i> ' + esc((allMetrics[0] && allMetrics[0].lastCalcTime) || "实时快照");
  }

  // 4. 自适应 SVG 五维雷达盘 (双模式主题响应)
  function renderRadarSvg() {
    var host = $("radarSvgHost");
    if (!host) return;

    var isDark = (document.documentElement.getAttribute("data-theme") || "").toLowerCase() === "dark";

    var sizeW = 240, sizeH = 216, cx = 120, cy = 106, r = 70;
    var angles = [-90, -18, 54, 126, 198];

    function getCoord(score, angleDeg) {
      var rad = angleDeg * Math.PI / 180;
      var dist = (score / 100) * r;
      return { x: cx + dist * Math.cos(rad), y: cy + dist * Math.sin(rad) };
    }

    var defs = '<defs>' +
      '<linearGradient id="radarFillGrad" x1="0%" y1="0%" x2="100%" y2="100%">' +
      (isDark
        ? '<stop offset="0%" stop-color="#3b82f6" stop-opacity="0.40" /><stop offset="100%" stop-color="#2563eb" stop-opacity="0.10" />'
        : '<stop offset="0%" stop-color="#3b82f6" stop-opacity="0.25" /><stop offset="100%" stop-color="#2563eb" stop-opacity="0.05" />'
      ) +
      '</linearGradient>' +
      (isDark ? '<filter id="radarGlow" x="-20%" y="-20%" width="140%" height="140%"><feGaussianBlur stdDeviation="2.5" result="blur" /><feComposite in="SourceGraphic" in2="blur" operator="over" /></filter>' : '') +
      '</defs>';

    var rings = [20, 40, 60, 80, 100];
    var ringStroke = isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(15, 23, 42, 0.12)";
    var axisStroke = isDark ? "rgba(255, 255, 255, 0.1)" : "rgba(15, 23, 42, 0.12)";
    var benchmarkColor = isDark ? "#10b981" : "#059669";
    var polygonStroke = isDark ? "#3b82f6" : "#2563eb";
    var dotColor = isDark ? "#60a5fa" : "#2563eb";
    var dotHalo = isDark ? "rgba(59, 130, 246, 0.35)" : "rgba(37, 99, 235, 0.25)";
    var textColor = isDark ? "#cbd5e1" : "#1e293b";

    var ringPaths = rings.map(function (val) {
      var pts = angles.map(function (deg) {
        var p = getCoord(val, deg);
        return p.x.toFixed(1) + "," + p.y.toFixed(1);
      }).join(" ");
      var stroke = val === 80 ? benchmarkColor : ringStroke;
      var dash = val === 80 ? 'stroke-dasharray="3,3" stroke-width="1.3"' : 'stroke-width="0.8"';
      return '<polygon points="' + pts + '" fill="none" stroke="' + stroke + '" ' + dash + '/>';
    }).join("");

    var axisLines = angles.map(function (deg) {
      var p = getCoord(100, deg);
      return '<line x1="' + cx + '" y1="' + cy + '" x2="' + p.x.toFixed(1) + '" y2="' + p.y.toFixed(1) + '" stroke="' + axisStroke + '" stroke-width="0.8"/>';
    }).join("");

    var scoreCoords = CATEGORY_NAMES.map(function (cat, i) {
      var s = categoryScores[cat] ? categoryScores[cat].score : 0;
      return getCoord(s, angles[i]);
    });
    var scorePts = scoreCoords.map(function (p) { return p.x.toFixed(1) + "," + p.y.toFixed(1); }).join(" ");

    var filterAttr = isDark ? 'filter="url(#radarGlow)"' : '';
    var scorePolygon = '<polygon points="' + scorePts + '" fill="url(#radarFillGrad)" stroke="' + polygonStroke + '" stroke-width="2" ' + filterAttr + '/>';

    var scoreDots = scoreCoords.map(function (p) {
      return '<circle cx="' + p.x.toFixed(1) + '" cy="' + p.y.toFixed(1) + '" r="4" fill="' + dotHalo + '"/>' +
        '<circle cx="' + p.x.toFixed(1) + '" cy="' + p.y.toFixed(1) + '" r="2.5" fill="' + dotColor + '" stroke="#ffffff" stroke-width="1"/>';
    }).join("");

    var labels = CATEGORY_NAMES.map(function (cat, i) {
      var s = categoryScores[cat] ? categoryScores[cat].score : 0;
      var p = getCoord(124, angles[i]);
      var anchor = "middle";
      var offsetY = 3;

      if (angles[i] === -90) {
        anchor = "middle";
        offsetY = -2;
      } else if (angles[i] > -90 && angles[i] < 90) {
        anchor = "start";
        offsetY = 3;
      } else {
        anchor = "end";
        offsetY = 3;
      }

      var scoreColor = isDark
        ? (s >= 80 ? "#34d399" : (s >= 60 ? "#fbbf24" : "#f87171"))
        : (s >= 80 ? "#059669" : (s >= 60 ? "#d97706" : "#dc2626"));

      return '<text x="' + p.x.toFixed(1) + '" y="' + (p.y + offsetY).toFixed(1) + '" font-size="9.5" fill="' + textColor + '" font-weight="600" text-anchor="' + anchor + '">' +
        esc(cat) + ' <tspan fill="' + scoreColor + '" font-weight="700">' + s + '</tspan></text>';
    }).join("");

    host.innerHTML = '<svg viewBox="0 0 ' + sizeW + ' ' + sizeH + '" preserveAspectRatio="xMidYMid meet">' +
      defs + ringPaths + axisLines + scorePolygon + scoreDots + labels +
      '</svg>';
  }

  // 5. 5分类健康条
  function renderCategoryCards() {
    var host = $("radarCategoryCards");
    if (!host) return;

    var html = CATEGORY_NAMES.map(function (cat) {
      var data = categoryScores[cat] || { score: 0, severity: "unknown", normal: 0, warn: 0, danger: 0, unknown: 0, total: 0 };
      var cls = CATEGORY_COLOR_CLASS[cat] || "cat-default";
      var scorable = data.normal + data.warn + data.danger;
      var scoreCls = scorable === 0 ? "score-danger" : (data.score >= 80 ? "score-ok" : (data.score >= 60 ? "score-warn" : "score-danger"));
      var isActive = state.matrixCategory === cat ? "active" : "";

      var pOk = scorable > 0 ? Math.round((data.normal / scorable) * 100) : 0;
      var pWarn = scorable > 0 ? Math.round((data.warn / scorable) * 100) : 0;
      var pDanger = scorable > 0 ? Math.round((data.danger / scorable) * 100) : 0;
      var scoreText = scorable > 0 ? data.score + "分" : "暂无数据";

      return '<div class="radar-cat-card ' + isActive + '" data-cat="' + esc(cat) + '" title="点击筛选右侧' + esc(cat) + '指标">' +
        '<div class="radar-cat-row-top">' +
          '<div class="radar-cat-left">' +
            '<span class="metric-category ' + cls + '" style="height:18px; padding:0 6px; font-size:10px; font-weight:600;">' + esc(cat) + '</span>' +
            '<span class="radar-cat-score ' + scoreCls + '">' + esc(scoreText) + '</span>' +
          '</div>' +
          '<div class="radar-cat-counts-tag">' +
            '<span class="c-ok">' + data.normal + '达标</span>' +
            (data.warn > 0 ? ' · <span class="c-warn">' + data.warn + '关注</span>' : '') +
            (data.danger > 0 ? ' · <span class="c-danger">' + data.danger + '风险</span>' : '') +
            (data.unknown > 0 ? ' · <span class="c-unknown">' + data.unknown + '未接入</span>' : '') +
          '</div>' +
        '</div>' +
        '<div class="radar-cat-track">' +
          '<div class="trk-seg ok" style="width:' + pOk + '%"></div>' +
          (pWarn > 0 ? '<div class="trk-seg warn" style="width:' + pWarn + '%"></div>' : '') +
          (pDanger > 0 ? '<div class="trk-seg danger" style="width:' + pDanger + '%"></div>' : '') +
        '</div>' +
      '</div>';
    }).join("");

    host.innerHTML = html;
  }

  // 6. 核心指标矩阵
  function renderMatrixGrid() {
    var host = $("radarMatrixGrid");
    if (!host) return;

    var filtered = allMetrics;
    if (state.matrixCategory) {
      filtered = allMetrics.filter(function (m) { return m.category === state.matrixCategory; });
    }

    if (filtered.length === 0) {
      host.innerHTML = '<div class="state-placeholder">当前分类下暂无指标</div>';
      return;
    }

    var html = filtered.map(function (m) {
      var isSel = m.code === state.selectedCode;
      var statusCls = m.status || "unknown";
      var statusText = STATUS_LABEL[statusCls] || "—";
      var statusIcon = statusCls === "normal" ? "fa-circle-check" : (statusCls === "warn" ? "fa-triangle-exclamation" : (statusCls === "danger" ? "fa-circle-xmark" : "fa-circle-question"));
      var catCls = CATEGORY_COLOR_CLASS[m.category] || "cat-delivery";

      var vn = parseNum(m.value);
      var isUnavailable = statusCls === "unknown" || isNaN(vn);
      var gaugePct = 0;
      if (!isUnavailable) {
        if (m.unit === "%") {
          gaugePct = Math.min(100, Math.max(0, vn));
        } else if (m.unit === "天") {
          var targetNum = parseNum(m.target) || 30;
          gaugePct = Math.min(100, Math.max(10, Math.round((vn / (targetNum * 1.3)) * 100)));
        } else {
          gaugePct = statusCls === "normal" ? 100 : (statusCls === "warn" ? 60 : 30);
        }
      }

      var valueHtml = isUnavailable ? '<span class="m-val-text unavailable">暂无数据</span>' : '<strong class="m-val-text">' + esc(m.value) + '</strong>';

      return '<div class="radar-m-compact-tile ' + (isSel ? 'selected' : '') + '" data-code="' + esc(m.code) + '">' +
        '<div class="tile-row-top">' +
          '<div class="tile-title-group">' +
            '<span class="cat-dot ' + catCls + '"></span>' +
            '<span class="m-name-text" title="' + esc(m.name) + '">' + esc(m.name) + '</span>' +
          '</div>' +
          '<span class="m-badge-pill ' + statusCls + '">' +
            '<i class="fas ' + statusIcon + '"></i> ' + esc(statusText) +
          '</span>' +
        '</div>' +
        '<div class="tile-row-middle">' +
          valueHtml +
          '<div class="m-mini-gauge">' +
            '<div class="m-gauge-bar ' + statusCls + '" style="width:' + gaugePct + '%"></div>' +
          '</div>' +
        '</div>' +
        '<div class="tile-row-bottom">' +
          '<span class="m-target-text"><i class="fas fa-bullseye" style="font-size:9px; margin-right:3px; opacity:0.7;"></i>目标 ' + esc(m.target || "—") + '</span>' +
          '<span class="m-code-text">' + esc(m.code) + '</span>' +
        '</div>' +
      '</div>';
    }).join("");

    host.innerHTML = html;
    updateDetailDock();
    updateFilterTags();
  }

  function updateFilterTags() {
    var container = $("matrixCategoryFilters");
    if (!container) return;
    var counts = { "": allMetrics.length };
    allMetrics.forEach(function (m) { counts[m.category] = (counts[m.category] || 0) + 1; });
    var tags = [["", "全部"]].concat(CATEGORY_NAMES.map(function (c) { return [c, c]; }));
    container.innerHTML = tags.map(function (pair) {
      var cat = pair[0], label = pair[1];
      var n = counts[cat] || 0;
      var active = state.matrixCategory === cat ? "active" : "";
      return '<button type="button" class="matrix-tag-btn ' + active + '" data-cat="' + esc(cat) + '">' + esc(label) + ' (' + n + ')</button>';
    }).join("");
  }

  // 7. 底部透视控制台 Dock
  function updateDetailDock() {
    var target = allMetrics.find(function (m) { return m.code === state.selectedCode; });
    if (!target && allMetrics.length > 0) {
      target = allMetrics[0];
      state.selectedCode = target.code;
    }
    if (!target) return;

    var catCls = CATEGORY_COLOR_CLASS[target.category] || "cat-delivery";
    var dockCat = $("dockMetricCategory");
    if (dockCat) {
      dockCat.className = "dock-badge " + catCls;
      dockCat.textContent = target.category;
    }
    if ($("dockMetricTitle")) $("dockMetricTitle").textContent = target.name;
    if ($("dockMetricCode")) $("dockMetricCode").textContent = target.code;
    var statusEl = $("dockMetricStatus");
    if (statusEl) {
      statusEl.className = "dock-status-badge " + (target.status || "unknown");
      var icon = target.status === "normal" ? "fa-circle-check" : (target.status === "warn" ? "fa-triangle-exclamation" : "fa-shield-virus");
      statusEl.innerHTML = '<i class="fas ' + icon + '"></i> ' + (STATUS_LABEL[target.status] || "—");
    }
    if ($("dockCurrentVal")) $("dockCurrentVal").textContent = (target.value && target.value !== "—") ? target.value : "暂无数据";
    if ($("dockTargetVal")) {
      $("dockTargetVal").textContent = "目标 " + (target.target || "—") +
        " · 关注 " + (target.warningThreshold || "—") +
        " · 风险 " + (target.dangerThreshold || "—");
    }
    if ($("dockFormula")) $("dockFormula").textContent = target.formula || "标准计算规则";
    if ($("dockSample")) $("dockSample").textContent = target.description || "基于后端指标目录的实时测算";
  }

  function render() {
    computeCategoryScores(allMetrics);
    renderRadarSvg();
    renderCategoryCards();
    renderMatrixGrid();
  }

  function initControls() {
    // 矩阵分类筛选按钮
    var tagContainer = $("matrixCategoryFilters");
    if (tagContainer) {
      tagContainer.addEventListener("click", function (e) {
        var btn = e.target.closest(".matrix-tag-btn");
        if (!btn) return;
        state.matrixCategory = btn.getAttribute("data-cat") || "";
        renderMatrixGrid();
      });
    }

    // 5分类健康卡片点击联动筛选
    var catBox = $("radarCategoryCards");
    if (catBox) {
      catBox.addEventListener("click", function (e) {
        var card = e.target.closest(".radar-cat-card");
        if (!card) return;
        var cat = card.getAttribute("data-cat");
        state.matrixCategory = (state.matrixCategory === cat) ? "" : cat;
        renderCategoryCards();
        renderMatrixGrid();
      });
    }

    // 点击卡片下钻联动
    $("radarMatrixGrid").addEventListener("click", function (e) {
      var tile = e.target.closest(".radar-m-compact-tile");
      if (!tile) return;
      var code = tile.getAttribute("data-code");
      state.selectedCode = code;
      $("radarMatrixGrid").querySelectorAll(".radar-m-compact-tile").forEach(function (t) { t.classList.remove("selected"); });
      tile.classList.add("selected");
      updateDetailDock();
    });

    // 监听全局主题 (Dark / Light) 动态切换，刷新雷达 SVG
    try {
      var themeObserver = new MutationObserver(function (mutations) {
        mutations.forEach(function (mutation) {
          if (mutation.attributeName === "data-theme") {
            renderRadarSvg();
          }
        });
      });
      themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
    } catch (e) {
      console.warn("Theme observer not supported", e);
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    initControls();
    loadMetrics(render);
  });
})();
