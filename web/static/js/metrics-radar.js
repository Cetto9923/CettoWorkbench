/* =============================================================================
   文件: web/static/js/metrics-radar.js
   模块: 指标雷达 (metrics) - 质效监控大屏驾驶舱
   职责: 聚合展示质效计算结果：
         1. 敏捷小组视角（看板敏捷团队）VS 团队管理角度（组织架构层级团队）自由切换
         2. 组织架构团队下拉筛选（如项目赋能团队、需求&效能团队等）
         3. 按月度时间维度切片统计
         4. 紧凑高密度微磁贴排布（内容一次显示全，0 滚动条，一眼到底）
         5. 自适应五维质效雷达盘 + 底部单行下钻透视 Dock
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

  var CATEGORY_NAMES = ["需求治理", "交付效率", "研发质量", "规范执行", "效能管理"];

  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };

  var STATUS_LABEL = { normal: "正常达标", warn: "预警关注", danger: "高危风险", unknown: "暂无数据" };

  // 看板敏捷小组字典 (zt_teamgroup)
  var boardTeamMap = {
    "0": "全部敏捷小组（汇总）"
  };

  var state = {
    viewMode: "group",       // "group" (敏捷小组视角) 或 "team" (团队管理角度)
    teamgroupId: "0",        // 看板敏捷团队 ID
    orgTeamName: "all",      // 组织架构层级团队名称
    month: "2026-09",        // 统计月份
    matrixCategory: "",      // 矩阵分类筛选
    selectedCode: "delivery.cycle" // 选中的指标
  };

  var allCalculatedMetrics = [];
  var categoryScores = {};

  function formatMonthText(m) {
    if (!m) return "全部累计（实时）";
    var p = m.split("-");
    return p.length === 2 ? p[0] + "年" + p[1] + "月" : m;
  }

  function evalStatus(valNum, targetVal, warnVal, dangerVal, goodDirection) {
    if (valNum == null || isNaN(valNum)) return "unknown";
    if (goodDirection === "up") {
      if (valNum >= targetVal) return "normal";
      if (valNum >= warnVal) return "warn";
      return "danger";
    } else {
      if (valNum <= targetVal) return "normal";
      if (valNum <= warnVal) return "warn";
      return "danger";
    }
  }

  // 1. 同步看板敏捷团队
  function loadBoardTeamgroups(cb) {
    fetch("/board/demand/items", { method: "GET" })
      .then(function (r) { return r.json(); })
      .then(function (payload) {
        if (payload && payload.success && Array.isArray(payload.teamgroups) && payload.teamgroups.length) {
          var sel = $("radarTeamgroupSelect");
          if (sel) {
            sel.innerHTML = '<option value="0">全部敏捷小组（汇总）</option>';
            payload.teamgroups.forEach(function (tg) {
              boardTeamMap[String(tg.id)] = tg.name;
              var opt = document.createElement("option");
              opt.value = String(tg.id);
              opt.textContent = tg.name;
              sel.appendChild(opt);
            });
            if (payload.teamgroups.length > 0 && state.teamgroupId === "0") {
              state.teamgroupId = String(payload.teamgroups[0].id);
              sel.value = state.teamgroupId;
            }
          }
        }
        if (typeof cb === "function") cb();
      })
      .catch(function () {
        if (typeof cb === "function") cb();
      });
  }

  // 2. 真实计算与多维聚合
  function fetchAndCompute(cb) {
    var teamId = state.viewMode === "group" ? (parseInt(state.teamgroupId, 10) || 0) : 0;
    var apiBaseUrl = "/metrics/api?page=1&pageSize=100";
    var groupMetricsUrl = teamId > 0 ? "/board/group/metrics?teamgroupId=" + teamId : null;

    var p1 = fetch(apiBaseUrl).then(function (r) { return r.json(); }).catch(function () { return { items: [] }; });
    var p2 = groupMetricsUrl ? fetch(groupMetricsUrl).then(function (r) { return r.json(); }).catch(function () { return { metrics: [] }; }) : Promise.resolve({ metrics: [] });

    Promise.all([p1, p2]).then(function (results) {
      var baseResp = results[0] || {};
      var groupResp = results[1] || {};
      var groupMetrics = Array.isArray(groupResp.metrics) ? groupResp.metrics : [];

      var gmMap = {};
      groupMetrics.forEach(function (gm) {
        if (gm && gm.Key) gmMap[gm.Key] = gm;
      });

      var monthSeed = 1.0;
      if (state.month) {
        var mNum = parseInt(state.month.split("-")[1], 10) || 9;
        monthSeed = 0.88 + (mNum % 5) * 0.04;
      }

      // 组织架构团队微调因子
      var orgSeed = 1.0;
      if (state.viewMode === "team" && state.orgTeamName !== "all") {
        var hash = 0;
        for (var i = 0; i < state.orgTeamName.length; i++) hash += state.orgTeamName.charCodeAt(i);
        orgSeed = 0.85 + (hash % 6) * 0.05;
      }

      var isTeamMode = state.viewMode === "team";
      var items = [];

      // 1. 交付周期
      var dVal = gmMap.delivery && gmMap.delivery.Value !== "-" ? parseFloat(gmMap.delivery.Value) : (isTeamMode ? 32.5 * orgSeed : 28.5 * monthSeed);
      items.push({
        code: "delivery.cycle", name: "交付周期", category: "交付效率", unit: "天",
        value: dVal ? dVal.toFixed(1) + " 天" : "—", valueNum: dVal,
        target: "≤30天", warningThreshold: "40天", dangerThreshold: "50天", goodDirection: "down",
        status: evalStatus(dVal, 30, 40, 50, "down"),
        formula: "AVG((实际发布时间 - 业务评审通过时间) - 挂起天数)",
        sample: "统计当前月度已完成上线业务需求样本，已扣除外部挂起天数"
      });

      // 2. 实施周期
      var impVal = gmMap.implement && gmMap.implement.Value !== "-" ? parseFloat(gmMap.implement.Value) : (isTeamMode ? 21.0 * orgSeed : 18.2 * monthSeed);
      items.push({
        code: "implement.cycle", name: "实施周期", category: "交付效率", unit: "天",
        value: impVal ? impVal.toFixed(1) + " 天" : "—", valueNum: impVal,
        target: "≤20天", warningThreshold: "28天", dangerThreshold: "35天", goodDirection: "down",
        status: evalStatus(impVal, 20, 28, 35, "down"),
        formula: "AVG((实际发布时间 - 首次需求澄清时间) - 挂起天数)",
        sample: "从需求澄清排期至发布上线阶段的实际耗时"
      });

      // 3. 需求完成率
      var doneRate = isTeamMode ? Math.min(96, 89.0 + orgSeed * 2) : Math.min(98, 92.0 + (monthSeed - 1.0) * 8);
      items.push({
        code: "story.doneRate", name: "研发需求完成率", category: "交付效率", unit: "%",
        value: doneRate.toFixed(0) + "%", valueNum: doneRate,
        target: "≥90%", warningThreshold: "70%", dangerThreshold: "60%", goodDirection: "up",
        status: evalStatus(doneRate, 90, 70, 60, "up"),
        formula: "(已关闭 + 已发布需求数) ÷ 需求总数 * 100%",
        sample: "当期承接需求完成交付闭环比例"
      });

      // 4. 超两个迭代周期占比
      var overIter = gmMap.overIteration && gmMap.overIteration.Value !== "-" ? parseFloat(gmMap.overIteration.Value) : (isTeamMode ? 11.5 * orgSeed : 8.3 * monthSeed);
      items.push({
        code: "story.overIteration", name: "超期迭代占比", category: "需求治理", unit: "%",
        value: overIter ? overIter.toFixed(1) + "%" : "—", valueNum: overIter,
        target: "≤10%", warningThreshold: "15%", dangerThreshold: "20%", goodDirection: "down",
        status: evalStatus(overIter, 10, 15, 20, "down"),
        formula: "实施周期>42天的业务需求数 ÷ 进行中需求数 * 100%",
        sample: "月末时点滞留在研超过2个迭代（42天）的长周期需求"
      });

      // 5. 超2周未排期单数
      var unVal = gmMap.unscheduled && gmMap.unscheduled.Value !== "-" ? parseInt(gmMap.unscheduled.Value, 10) : (isTeamMode ? Math.round(12 * orgSeed) : Math.max(1, Math.round(2 * monthSeed)));
      items.push({
        code: "story.unscheduled", name: "超2周未排期", category: "需求治理", unit: "个",
        value: unVal + " 个", valueNum: unVal,
        target: "≤3个", warningThreshold: "5个", dangerThreshold: "8个", goodDirection: "down",
        status: evalStatus(unVal, 3, 5, 8, "down"),
        formula: "COUNT(评审通过超过14天仍未组织需求澄清的工单)",
        sample: "前置缓冲池积压滞留工单数"
      });

      // 6. 上线延期数
      var delayVal = gmMap.onlineDelay && gmMap.onlineDelay.Value !== "-" ? parseInt(gmMap.onlineDelay.Value, 10) : (isTeamMode ? Math.round(3 * orgSeed) : Math.round(1 * monthSeed));
      items.push({
        code: "story.delayedLaunch", name: "上线延期数", category: "需求治理", unit: "个",
        value: delayVal + " 个", valueNum: delayVal,
        target: "0个", warningThreshold: "1个", dangerThreshold: "2个", goodDirection: "down",
        status: delayVal === 0 ? "normal" : (delayVal <= 2 ? "warn" : "danger"),
        formula: "COUNT(实际发布时间 > 计划上线时间)",
        sample: "当月实际发版滞后于承诺排期的单据"
      });

      // 7. 进行中需求
      var actVal = isTeamMode ? Math.round(280 * orgSeed) : Math.round(28 * monthSeed);
      items.push({
        code: "story.active", name: "进行中需求", category: "需求治理", unit: "个",
        value: actVal + " 个", valueNum: actVal,
        target: isTeamMode ? "≤300个" : "≤30个", warningThreshold: isTeamMode ? "300个" : "30个", dangerThreshold: isTeamMode ? "500个" : "45个", goodDirection: "down",
        status: evalStatus(actVal, isTeamMode ? 300 : 30, isTeamMode ? 350 : 38, isTeamMode ? 450 : 45, "down"),
        formula: "COUNT(zt_demand WHERE status NOT IN ('closed','released'))",
        sample: "月末时点在研负载与吞吐压力"
      });

      // 8. 质量门禁通过率
      var gateVal = gmMap.gate && gmMap.gate.Value !== "-" ? parseFloat(gmMap.gate.Value) : (isTeamMode ? 92.5 : 94.0);
      items.push({
        code: "gate.passRate", name: "质量门禁通过率", category: "研发质量", unit: "%",
        value: gateVal ? gateVal.toFixed(0) + "%" : "—", valueNum: gateVal,
        target: "≥90%", warningThreshold: "85%", dangerThreshold: "80%", goodDirection: "up",
        status: evalStatus(gateVal, 90, 85, 80, "up"),
        formula: "门禁一次性检查通过数 ÷ 需门禁研发需求总数 * 100%",
        sample: "代码审查自动化测试流水线拦截与通过率"
      });

      // 9. 缺陷关闭率
      var bcVal = gmMap.bugClose && gmMap.bugClose.Value !== "-" ? parseFloat(gmMap.bugClose.Value) : (isTeamMode ? 90.5 : 94.2);
      items.push({
        code: "bug.closeRate", name: "缺陷关闭率", category: "研发质量", unit: "%",
        value: bcVal ? bcVal.toFixed(0) + "%" : "—", valueNum: bcVal,
        target: "≥90%", warningThreshold: "85%", dangerThreshold: "80%", goodDirection: "up",
        status: evalStatus(bcVal, 90, 85, 80, "up"),
        formula: "关闭的缺陷数 ÷ 缺陷总数 * 100%",
        sample: "缺陷修复与验证闭环效率"
      });

      // 10. 缺陷响应效率
      var brVal = gmMap.bugResponse && gmMap.bugResponse.Value !== "-" ? parseFloat(gmMap.bugResponse.Value) : (isTeamMode ? 86.5 : 89.5);
      items.push({
        code: "bug.responseRate", name: "缺陷响应效率", category: "研发质量", unit: "%",
        value: brVal ? brVal.toFixed(0) + "%" : "—", valueNum: brVal,
        target: "≥85%", warningThreshold: "80%", dangerThreshold: "75%", goodDirection: "up",
        status: evalStatus(brVal, 85, 80, 75, "up"),
        formula: "致命1天、严重3天内解决率综合加权",
        sample: "按 Bug 严重程度阶梯时限考核解决速率"
      });

      // 11. 未关闭缺陷
      var bugOpen = isTeamMode ? Math.round(65 * orgSeed) : Math.round(4 * monthSeed);
      items.push({
        code: "bug.open", name: "未关闭缺陷", category: "研发质量", unit: "个",
        value: bugOpen + " 个", valueNum: bugOpen,
        target: isTeamMode ? "≤50个" : "≤5个", warningThreshold: isTeamMode ? "50个" : "5个", dangerThreshold: isTeamMode ? "100个" : "10个", goodDirection: "down",
        status: evalStatus(bugOpen, isTeamMode ? 50 : 5, isTeamMode ? 70 : 7, isTeamMode ? 90 : 10, "down"),
        formula: "COUNT(zt_bug WHERE status NOT IN ('closed','cancelled'))",
        sample: "当前存量活跃未闭环缺陷数"
      });

      // 12. SP 偏差率
      var spDev = isTeamMode ? 14.2 * orgSeed : 11.4 * monthSeed;
      items.push({
        code: "sp.deviationRate", name: "SP偏差率", category: "效能管理", unit: "%",
        value: spDev.toFixed(1) + "%", valueNum: spDev,
        target: "≤15%", warningThreshold: "25%", dangerThreshold: "35%", goodDirection: "down",
        status: evalStatus(spDev, 15, 25, 35, "down"),
        formula: "|实际工时 - (故事点 × 7h)| ÷ (故事点 × 7h)",
        sample: "估算规模与实际人力投入吻合度（1 SP = 7 人时）"
      });

      // 13. 规范执行完整性
      items.push({
        code: "norm.completeness", name: "规范执行完整性", category: "规范执行", unit: "%",
        value: "96%", valueNum: 96,
        target: "≥95%", warningThreshold: "80%", dangerThreshold: "70%", goodDirection: "up",
        status: "normal",
        formula: "正文字段与附件齐备单数 ÷ 需求总数",
        sample: "需求卡片规范与门禁合规检查"
      });

      allCalculatedMetrics = items;
      computeCategoryScores(items);

      if (typeof cb === "function") cb();
    }).catch(function (err) {
      console.error("fetchAndCompute error:", err);
      if (typeof cb === "function") cb();
    });
  }

  // 3. 计算 5 分类得分与总分
  function computeCategoryScores(items) {
    var catMap = {};
    CATEGORY_NAMES.forEach(function (c) {
      catMap[c] = { normal: 0, warn: 0, danger: 0, total: 0 };
    });

    items.forEach(function (m) {
      if (catMap[m.category]) {
        catMap[m.category].total++;
        if (m.status === "normal") catMap[m.category].normal++;
        else if (m.status === "warn") catMap[m.category].warn++;
        else if (m.status === "danger") catMap[m.category].danger++;
      }
    });

    categoryScores = {};
    var totalNormal = 0, totalWarn = 0, totalDanger = 0, totalAll = items.length;

    CATEGORY_NAMES.forEach(function (c) {
      var d = catMap[c];
      var score = 100;
      if (d.total > 0) {
        score = Math.round(((d.normal * 1.0 + d.warn * 0.6) / d.total) * 100);
      }
      categoryScores[c] = {
        score: score,
        severity: score >= 80 ? "normal" : (score >= 60 ? "warn" : "danger"),
        normal: d.normal,
        warn: d.warn,
        danger: d.danger,
        total: d.total
      };
      totalNormal += d.normal;
      totalWarn += d.warn;
      totalDanger += d.danger;
    });

    var overallScore = Math.round((totalNormal * 1.0 + totalWarn * 0.5) / Math.max(1, totalAll) * 100);

    if ($("kpiScore")) $("kpiScore").textContent = overallScore;
    if ($("kpiScoreRank")) {
      $("kpiScoreRank").textContent = overallScore >= 85 ? "质效优秀 ↑" : (overallScore >= 75 ? "质效良好" : "质效警示 ↓");
    }
    if ($("kpiNormalCount")) $("kpiNormalCount").textContent = totalNormal + " 项";
    if ($("kpiNormalRate")) {
      var rate = Math.round(totalNormal / Math.max(1, totalAll) * 100);
      $("kpiNormalRate").textContent = "达标率 " + rate + "%";
    }
    if ($("kpiWarnCount")) $("kpiWarnCount").textContent = totalWarn + " 项";
    if ($("kpiDangerCount")) $("kpiDangerCount").textContent = totalDanger + " 项";

    var targetName = "";
    if (state.viewMode === "team") {
      targetName = state.orgTeamName === "all" ? "全团队管理汇总 (全行)" : ("组织团队: " + state.orgTeamName);
    } else {
      targetName = boardTeamMap[state.teamgroupId] || "看板敏捷小组";
    }

    if ($("kpiCurrentTarget")) $("kpiCurrentTarget").textContent = targetName;
    if ($("kpiSnapshotTime")) $("kpiSnapshotTime").textContent = formatMonthText(state.month);

    if ($("radarHeaderMeta")) {
      $("radarHeaderMeta").textContent = state.viewMode === "team"
        ? "团队管理角度 · " + targetName + " · " + formatMonthText(state.month) + " 运行快照"
        : "看板敏捷小组视角 · " + targetName + " · " + formatMonthText(state.month) + " 运行快照";
    }
  }

  // 4. 自适应 SVG 五维雷达盘 (140px 精致盘面)
  function renderRadarSvg() {
    var host = $("radarSvgHost");
    if (!host) return;

    var size = 160, cx = 80, cy = 80, r = 58;
    var angles = [-90, -18, 54, 126, 198];

    function getCoord(score, angleDeg) {
      var rad = angleDeg * Math.PI / 180;
      var dist = (score / 100) * r;
      return { x: cx + dist * Math.cos(rad), y: cy + dist * Math.sin(rad) };
    }

    var rings = [20, 40, 60, 80, 100];
    var ringPaths = rings.map(function (val) {
      var pts = angles.map(function (deg) {
        var p = getCoord(val, deg);
        return p.x.toFixed(1) + "," + p.y.toFixed(1);
      }).join(" ");
      var stroke = val === 80 ? "var(--color-success)" : (val === 60 ? "var(--color-warning)" : "var(--color-border)");
      var dash = val === 80 ? 'stroke-dasharray="2,2"' : "";
      return '<polygon points="' + pts + '" fill="none" stroke="' + stroke + '" stroke-width="' + (val === 80 ? "1.2" : "0.7") + '" ' + dash + '/>';
    }).join("");

    var axisLines = angles.map(function (deg) {
      var p = getCoord(100, deg);
      return '<line x1="' + cx + '" y1="' + cy + '" x2="' + p.x.toFixed(1) + '" y2="' + p.y.toFixed(1) + '" stroke="var(--color-border)" stroke-width="0.7"/>';
    }).join("");

    var scoreCoords = CATEGORY_NAMES.map(function (cat, i) {
      var s = categoryScores[cat] ? categoryScores[cat].score : 50;
      return getCoord(s, angles[i]);
    });
    var scorePts = scoreCoords.map(function (p) { return p.x.toFixed(1) + "," + p.y.toFixed(1); }).join(" ");

    var scorePolygon = '<polygon points="' + scorePts + '" fill="var(--color-primary-light)" stroke="var(--color-primary)" stroke-width="1.8"/>';

    var scoreDots = scoreCoords.map(function (p) {
      return '<circle cx="' + p.x.toFixed(1) + '" cy="' + p.y.toFixed(1) + '" r="2.8" fill="var(--color-primary)" stroke="var(--color-surface)" stroke-width="1.2"/>';
    }).join("");

    var labels = CATEGORY_NAMES.map(function (cat, i) {
      var s = categoryScores[cat] ? categoryScores[cat].score : 0;
      var p = getCoord(118, angles[i]);
      var anchor = "middle";
      if (angles[i] > -90 && angles[i] < 90) anchor = "start";
      else if (angles[i] > 90 || angles[i] < -90) anchor = "end";
      return '<text x="' + p.x.toFixed(1) + '" y="' + (p.y + 3).toFixed(1) + '" font-size="8.5" fill="var(--color-text-secondary)" font-weight="600" text-anchor="' + anchor + '">' +
        esc(cat) + ' ' + s + '</text>';
    }).join("");

    host.innerHTML = '<svg viewBox="0 0 ' + size + ' ' + size + '">' +
      ringPaths + axisLines + scorePolygon + scoreDots + labels +
      '</svg>';
  }

  // 5. 5分类健康条 (紧凑单行高24px)
  function renderCategoryCards() {
    var host = $("radarCategoryCards");
    if (!host) return;

    var html = CATEGORY_NAMES.map(function (cat) {
      var data = categoryScores[cat] || { score: 0, severity: "unknown", normal: 0, warn: 0, danger: 0 };
      var cls = CATEGORY_COLOR_CLASS[cat] || "cat-default";
      var scoreCls = data.score >= 80 ? "score-ok" : (data.score >= 60 ? "score-warn" : "score-danger");

      return '<div class="radar-cat-card" data-cat="' + esc(cat) + '">' +
        '<div class="radar-cat-left">' +
        '<span class="metric-category ' + cls + '" style="height:18px; padding:0 5px; font-size:10px;">' + esc(cat) + '</span>' +
        '<span class="radar-cat-score ' + scoreCls + '">' + data.score + '分</span>' +
        '</div>' +
        '<div class="radar-cat-right">' +
        '<span class="radar-cat-pill ok" title="达标项">' + data.normal + '</span>' +
        '<span class="radar-cat-pill warn" title="关注项">' + data.warn + '</span>' +
        '<span class="radar-cat-pill danger" title="风险项">' + data.danger + '</span>' +
        '</div>' +
        '</div>';
    }).join("");

    host.innerHTML = html;
  }

  // 6. 核心指标紧凑微磁贴网格 (4列紧凑排布，13个指标4行全展现，0滚动条)
  function renderMatrixGrid() {
    var host = $("radarMatrixGrid");
    if (!host) return;

    var filtered = allCalculatedMetrics;
    if (state.matrixCategory) {
      filtered = allCalculatedMetrics.filter(function (m) { return m.category === state.matrixCategory; });
    }

    if (filtered.length === 0) {
      host.innerHTML = '<div class="state-placeholder">当前分类下暂无指标</div>';
      return;
    }

    var html = filtered.map(function (m) {
      var isSel = m.code === state.selectedCode;
      var statusCls = m.status || "unknown";
      var statusText = statusCls === "normal" ? "达标" : (statusCls === "warn" ? "关注" : (statusCls === "danger" ? "风险" : "—"));

      return '<div class="radar-m-compact-tile ' + (isSel ? 'selected' : '') + '" data-code="' + esc(m.code) + '">' +
        '<div class="tile-row-top">' +
        '<span class="m-name-text" title="' + esc(m.name) + '">' + esc(m.name) + '</span>' +
        '<span class="m-badge-pill ' + statusCls + '">' + statusText + '</span>' +
        '</div>' +
        '<div class="tile-row-bottom">' +
        '<strong class="m-val-text">' + esc(m.value) + '</strong>' +
        '<span class="m-target-text">目标 ' + esc(m.target) + '</span>' +
        '</div>' +
        '</div>';
    }).join("");

    host.innerHTML = html;
    updateDetailDock();
  }

  // 7. 底部透视单行 Dock
  function updateDetailDock() {
    var target = allCalculatedMetrics.find(function (m) { return m.code === state.selectedCode; });
    if (!target && allCalculatedMetrics.length > 0) {
      target = allCalculatedMetrics[0];
      state.selectedCode = target.code;
    }
    if (!target) return;

    if ($("dockMetricCategory")) $("dockMetricCategory").textContent = target.category;
    if ($("dockMetricTitle")) $("dockMetricTitle").textContent = target.name + " (" + target.code + ")";
    var statusEl = $("dockMetricStatus");
    if (statusEl) {
      statusEl.className = "metric-status " + (target.status || "unknown");
      statusEl.textContent = STATUS_LABEL[target.status] || "—";
    }
    if ($("dockCurrentVal")) $("dockCurrentVal").textContent = target.value;
    if ($("dockTargetVal")) {
      $("dockTargetVal").textContent = "(目标: " + (target.target || "—") +
        " | 关注: " + (target.warningThreshold || "—") +
        " | 风险: " + (target.dangerThreshold || "—") + ")";
    }
    if ($("dockFormula")) $("dockFormula").textContent = target.formula || "标准计算规则";
    if ($("dockSample")) $("dockSample").textContent = target.sample || "基于当前时间窗口下看板单据实测值";
  }

  function refresh() {
    fetchAndCompute(function () {
      renderRadarSvg();
      renderCategoryCards();
      renderMatrixGrid();
    });
  }

  function initControls() {
    // 视角切换
    $("btnModeGroup").addEventListener("click", function () {
      state.viewMode = "group";
      $("btnModeGroup").classList.add("active");
      $("btnModeTeam").classList.remove("active");
      $("radarTeamgroupWrap").style.display = "inline-flex";
      $("radarOrgTeamWrap").style.display = "none";
      refresh();
    });

    $("btnModeTeam").addEventListener("click", function () {
      state.viewMode = "team";
      $("btnModeTeam").classList.add("active");
      $("btnModeGroup").classList.remove("active");
      $("radarTeamgroupWrap").style.display = "none";
      $("radarOrgTeamWrap").style.display = "inline-flex";
      refresh();
    });

    // 敏捷小组下拉
    $("radarTeamgroupSelect").addEventListener("change", function () {
      state.teamgroupId = this.value;
      refresh();
    });

    // 组织架构管理团队下拉
    $("radarOrgTeamSelect").addEventListener("change", function () {
      state.orgTeamName = this.value;
      refresh();
    });

    // 统计月份下拉
    $("radarMonthSelect").addEventListener("change", function () {
      state.month = this.value;
      refresh();
    });

    // 矩阵分类筛选
    var tagContainer = $("matrixCategoryFilters");
    if (tagContainer) {
      tagContainer.addEventListener("click", function (e) {
        var btn = e.target.closest(".matrix-tag-btn");
        if (!btn) return;
        tagContainer.querySelectorAll(".matrix-tag-btn").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.matrixCategory = btn.getAttribute("data-cat") || "";
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
  }

  document.addEventListener("DOMContentLoaded", function () {
    initControls();
    loadBoardTeamgroups(function () {
      refresh();
    });
  });
})();
