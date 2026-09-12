/* =============================================================================
   文件: web/static/js/metrics-manage.js
   模块: 指标管理 (metrics) - /metrics/manage
   职责: 指标元数据维护控制台：指标定义、考核目标、关注与风险阈值、计算口径的增删改配。
         采用高保真右侧标准滑出抽屉，支持完整的数据回显、表单校验与持久化。
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

  var CATEGORY_COLOR_CLASS = {
    "需求治理": "cat-demand",
    "交付效率": "cat-delivery",
    "研发质量": "cat-quality",
    "规范执行": "cat-compliance",
    "效能管理": "cat-performance"
  };

  // 14 项基线指标定义全景元数据
  var defaultMetrics = [
    {
      code: "delivery.cycle",
      name: "交付周期",
      category: "交付效率",
      unit: "天",
      dataSource: "FineReport (JbuB) / 禅道 zt_demand.teamGroup",
      period: "月度",
      target: "≤30天",
      warningThreshold: "40天",
      dangerThreshold: "50天",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO",
      description: "业务需求从业务评审通过到完成上线的天数，扣除外部挂起天数（归属看板敏捷小组）",
      formula: "AVG((实际发布时间 - 业务评审通过时间) - 挂起天数)"
    },
    {
      code: "implement.cycle",
      name: "实施周期",
      category: "交付效率",
      unit: "天",
      dataSource: "FineReport (JbuB) / 禅道 zt_demand.teamGroup",
      period: "月度",
      target: "≤20天",
      warningThreshold: "28天",
      dangerThreshold: "35天",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO / SM",
      description: "业务需求从首次需求澄清到完成上线的天数，扣除挂起天数（归属看板敏捷小组）",
      formula: "AVG((实际发布时间 - 首次需求澄清时间) - 挂起天数)"
    },
    {
      code: "story.overIteration",
      name: "超两个迭代周期占比",
      category: "需求治理",
      unit: "%",
      dataSource: "FineReport (hNBB) / 禅道看板",
      period: "月度",
      target: "≤10%",
      warningThreshold: "15%",
      dangerThreshold: "20%",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO",
      description: "实施周期超过42天的工单数占该看板小组当前进行中工单数的比例",
      formula: "实施周期>42天的业务需求数 ÷ 看板进行中需求数 * 100%"
    },
    {
      code: "story.unscheduled",
      name: "超2周未排期单数",
      category: "需求治理",
      unit: "个",
      dataSource: "FineReport (hNBB) / 禅道 zt_demand",
      period: "月度",
      target: "≤3个",
      warningThreshold: "5个",
      dangerThreshold: "8个",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO",
      description: "距离业务评审通过超过2周（14天）仍未在看板进行需求澄清的单数",
      formula: "COUNT(zt_demand WHERE teamGroup=? AND 评审通过超14天未澄清)"
    },
    {
      code: "gate.passRate",
      name: "质量门禁通过率",
      category: "开发质量",
      unit: "%",
      dataSource: "DevOps / 门禁平台",
      period: "月度",
      target: "≥90%",
      warningThreshold: "85%",
      dangerThreshold: "80%",
      goodDirection: "up",
      enabled: true,
      ownerRole: "开发组长",
      description: "通过质量门禁检查的研发需求占需要质量门禁的研发需求比例",
      formula: "质量门禁通过数 ÷ 需质量门禁的研发需求总数 * 100%"
    },
    {
      code: "bug.closeRate",
      name: "缺陷关闭率",
      category: "研发质量",
      unit: "%",
      dataSource: "禅道 zt_bug / 看板团队成员",
      period: "月度",
      target: "≥90%",
      warningThreshold: "85%",
      dangerThreshold: "80%",
      goodDirection: "up",
      enabled: true,
      ownerRole: "测试负责人",
      description: "该看板敏捷小组名下处理的缺陷关闭完成情况",
      formula: "关闭的缺陷数 ÷ 缺陷总数 * 100%"
    },
    {
      code: "bug.responseRate",
      name: "缺陷响应效率",
      category: "研发质量",
      unit: "%",
      dataSource: "禅道 zt_bug",
      period: "月度",
      target: "≥85%",
      warningThreshold: "80%",
      dangerThreshold: "75%",
      goodDirection: "up",
      enabled: true,
      ownerRole: "开发组长",
      description: "致命缺陷1天内（24h）、严重3天内、一般5天内解决响应效率",
      formula: "各等级限期内解决缺陷数 ÷ 对应等级缺陷总数 * 100%"
    },
    {
      code: "story.delayedLaunch",
      name: "上线延期数",
      category: "需求治理",
      unit: "个",
      dataSource: "FineReport (hNBB) / 禅道需求",
      period: "月度",
      target: "0个",
      warningThreshold: "1个",
      dangerThreshold: "2个",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO",
      description: "实际发布时间晚于预计上线时间的业务需求单数",
      formula: "COUNT(releasedDate > estimateLaunch)"
    },
    {
      code: "sp.deviationRate",
      name: "SP偏差率",
      category: "效能管理",
      unit: "%",
      dataSource: "FineReport (Y3VO) / 禅道工时",
      period: "月度",
      target: "≤15%",
      warningThreshold: "25%",
      dangerThreshold: "35%",
      goodDirection: "down",
      enabled: true,
      ownerRole: "SM",
      description: "看板小组业务需求规模（故事点）与实际人力投入偏差率（2026新增）",
      formula: "|实际工时 - 理论工时| ÷ 理论工时 = |实际工时 - (故事点 × 7小时)| ÷ (故事点 × 7小时)"
    },
    {
      code: "story.total",
      name: "研发需求总量",
      category: "需求治理",
      unit: "个",
      dataSource: "禅道 zt_demand / zt_story",
      period: "实时",
      target: "—",
      warningThreshold: "—",
      dangerThreshold: "—",
      goodDirection: "up",
      enabled: true,
      ownerRole: "PO",
      description: "研发需求累计总数，反映团队在承接需求池规模",
      formula: "COUNT(zt_story WHERE deleted='0')"
    },
    {
      code: "story.active",
      name: "进行中研发需求",
      category: "需求治理",
      unit: "个",
      dataSource: "禅道 zt_demand.teamGroup",
      period: "实时",
      target: "≤30个",
      warningThreshold: "30个",
      dangerThreshold: "45个",
      goodDirection: "down",
      enabled: true,
      ownerRole: "PO",
      description: "该看板敏捷小组当前未关闭、未发布的在研需求总数",
      formula: "COUNT(zt_demand WHERE teamGroup=? AND status NOT IN ('closed','released'))"
    },
    {
      code: "story.doneRate",
      name: "研发需求完成率",
      category: "交付效率",
      unit: "%",
      dataSource: "禅道 zt_demand.teamGroup",
      period: "月度",
      target: "≥90%",
      warningThreshold: "70%",
      dangerThreshold: "60%",
      goodDirection: "up",
      enabled: true,
      ownerRole: "PO",
      description: "已关闭/已发布需求占小组总需求池的比例",
      formula: "(closed + released) ÷ total"
    },
    {
      code: "bug.open",
      name: "未关闭缺陷",
      category: "研发质量",
      unit: "个",
      dataSource: "禅道 zt_bug / 看板团队成员",
      period: "实时",
      target: "≤5个",
      warningThreshold: "5个",
      dangerThreshold: "10个",
      goodDirection: "down",
      enabled: true,
      ownerRole: "测试负责人",
      description: "看板小组名下未关闭、未取消的缺陷存量",
      formula: "COUNT(zt_bug WHERE status NOT IN ('closed','cancelled'))"
    },
    {
      code: "norm.completeness",
      name: "业务需求富文本完整性",
      category: "规范执行",
      unit: "%",
      dataSource: "门禁数据待同步",
      period: "月度",
      target: "≥95%",
      warningThreshold: "80%",
      dangerThreshold: "95%",
      goodDirection: "up",
      enabled: true,
      ownerRole: "PMO",
      description: "业务需求富文本正文（非空描述+字段完整）的覆盖率；门禁数据待同步",
      formula: "COUNT(demand WHERE description!='') ÷ COUNT(demand)"
    }
  ];

  var STORAGE_KEY = "crcb_metrics_meta_config_v2";
  var metricsList = [];

  function loadLocalMetrics() {
    try {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        var parsed = JSON.parse(saved);
        if (Array.isArray(parsed) && parsed.length) {
          return parsed;
        }
      }
    } catch (e) {
      console.warn("loadLocalMetrics error:", e);
    }
    return JSON.parse(JSON.stringify(defaultMetrics));
  }

  function saveLocalMetrics() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(metricsList));
    } catch (e) {
      console.warn("saveLocalMetrics error:", e);
    }
  }

  var state = {
    category: "",
    role: "",
    status: "enabled",
    keyword: "",
    page: 1,
    pageSize: 15
  };

  var searchTimer = null;

  function showToast(msg) {
    var toast = $("mToast");
    if (!toast) return;
    toast.textContent = msg;
    toast.hidden = false;
    setTimeout(function () { toast.hidden = true; }, 2200);
  }

  function renderSummary() {
    var total = metricsList.length;
    var enabledCount = metricsList.filter(function (m) { return m.enabled !== false; }).length;
    var warnCount = metricsList.filter(function (m) { return m.warningThreshold && m.warningThreshold !== "—"; }).length;
    var dangerCount = metricsList.filter(function (m) { return m.dangerThreshold && m.dangerThreshold !== "—"; }).length;
    var catMap = {};
    metricsList.forEach(function (m) { if (m.category) { catMap[m.category] = true; } });

    if ($("mSummaryTotal")) { $("mSummaryTotal").textContent = String(total); }
    if ($("mSummaryEnabled")) { $("mSummaryEnabled").textContent = String(enabledCount); }
    if ($("mSummaryWarnThreshold")) { $("mSummaryWarnThreshold").textContent = String(warnCount); }
    if ($("mSummaryDangerThreshold")) { $("mSummaryDangerThreshold").textContent = String(dangerCount); }
    if ($("mSummaryCategories")) { $("mSummaryCategories").textContent = String(Object.keys(catMap).length); }
  }

  function renderRow(m, idx) {
    var categoryCls = CATEGORY_COLOR_CLASS[m.category] || "cat-default";
    var isEnabled = m.enabled !== false;

    var nameHtml = '<strong class="metric-name" style="cursor:pointer; color:var(--po-blue, #2563eb);" data-metric-edit="' + esc(m.code) + '">' + esc(m.name) + '</strong>' +
      '<small class="metric-code">' + esc(m.code) + (m.unit ? ' · ' + esc(m.unit) : '') + '</small>';

    var targetHtml = '<span class="metric-target-val" style="font-weight:700; color:#0f172a; font-family:ui-monospace, monospace;">' + esc(m.target || "—") + '</span>';

    var warnText = esc(m.warningThreshold || "—");
    var dangerText = esc(m.dangerThreshold || "—");
    var thresholdHtml = '<div class="metric-target-cell">' +
      '<span class="metric-target-row" data-tone="warn"><em>关注</em><span>' + warnText + '</span></span>' +
      '<span class="metric-target-row" data-tone="danger"><em>风险</em><span>' + dangerText + '</span></span>' +
      '</div>';

    var dirText = m.goodDirection === "up"
      ? '<span style="color:#059669; font-weight:600;">越大越好 ↑</span>'
      : '<span style="color:#2563eb; font-weight:600;">越小越好 ↓</span>';

    var statusBadge = isEnabled
      ? '<span class="metric-status normal">启用中</span>'
      : '<span class="metric-status unknown">已停用</span>';

    var toggleBtn = isEnabled
      ? '<button type="button" class="action-btn text-danger" data-metric-toggle="' + esc(m.code) + '" title="停用该指标">停用</button>'
      : '<button type="button" class="action-btn text-success" data-metric-toggle="' + esc(m.code) + '" title="启用该指标">启用</button>';

    var editBtn = '<button type="button" class="action-btn primary" data-metric-edit="' + esc(m.code) + '" style="font-weight:600;">配置编辑</button>';

    return '<tr data-code="' + esc(m.code) + '">' +
      '<td class="metric-col-name">' + nameHtml + '</td>' +
      '<td><span class="metric-category ' + categoryCls + '">' + esc(m.category) + '</span></td>' +
      '<td style="font-size:12px; color:#475569; max-width:180px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;" title="' + esc(m.dataSource || "") + '">' + esc(m.dataSource || "—") + '</td>' +
      '<td>' + targetHtml + '</td>' +
      '<td class="metric-col-target">' + thresholdHtml + '</td>' +
      '<td><span class="state-tag">' + esc(m.ownerRole || "—") + '</span></td>' +
      '<td style="font-size:11px;">' + dirText + '</td>' +
      '<td>' + statusBadge + '</td>' +
      '<td class="metric-col-opt" style="white-space:nowrap;">' + editBtn + ' ' + toggleBtn + '</td>' +
      '</tr>';
  }

  function applyFilterAndRender() {
    var kw = (state.keyword || "").toLowerCase();
    var filtered = metricsList.filter(function (m) {
      if (state.category && m.category !== state.category) { return false; }
      if (state.role && m.ownerRole !== state.role) { return false; }
      if (state.status === "enabled" && m.enabled === false) { return false; }
      if (state.status === "disabled" && m.enabled !== false) { return false; }
      if (kw) {
        var str = (m.name + " " + m.code + " " + m.dataSource + " " + m.formula + " " + m.ownerRole).toLowerCase();
        if (str.indexOf(kw) === -1) { return false; }
      }
      return true;
    });

    renderSummary();

    var total = filtered.length;
    var tbody = $("mTbody");
    if (tbody) {
      if (total === 0) {
        tbody.innerHTML = '<tr><td colspan="9" class="state-placeholder">当前筛选条件下暂无指标配置</td></tr>';
      } else {
        tbody.innerHTML = filtered.map(function (m, i) { return renderRow(m, i); }).join("");
      }
    }

    if ($("mSummary")) {
      $("mSummary").textContent = "共 " + total + " 条指标定义 (总数 " + metricsList.length + " 项)";
    }
  }

  // 打开右侧滑出抽屉
  function openDrawer(metric) {
    var modal = $("mDrawerModal");
    if (!modal) return;

    var isEdit = !!metric;
    $("fIsEdit").value = isEdit ? "1" : "0";
    $("mDrawerBadge").textContent = isEdit ? "编辑指标" : "新增指标";
    $("mDrawerTitle").textContent = isEdit ? "配置指标 [" + metric.name + "]" : "新增科技研发质效指标";
    $("mDrawerSub").textContent = isEdit
      ? "修改考核目标、关注与风险阈值红线及取数公式"
      : "创建新的指标元数据定义，用于敏捷团队度量考核";

    $("fCodeInput").value = isEdit ? metric.code : "";
    $("fCodeInput").disabled = isEdit;
    $("fName").value = isEdit ? metric.name : "";
    $("fCategory").value = isEdit ? metric.category : "需求治理";
    $("fOwnerRole").value = isEdit ? (metric.ownerRole || "PO") : "PO";
    $("fUnit").value = isEdit ? (metric.unit || "") : "";
    $("fPeriod").value = isEdit ? (metric.period || "月度") : "月度";

    $("fTarget").value = isEdit ? (metric.target || "") : "";
    $("fWarning").value = isEdit ? (metric.warningThreshold || "") : "";
    $("fDanger").value = isEdit ? (metric.dangerThreshold || "") : "";
    $("fDirection").value = isEdit ? (metric.goodDirection || "down") : "down";
    $("fDataSource").value = isEdit ? (metric.dataSource || "") : "";

    $("fFormula").value = isEdit ? (metric.formula || "") : "";
    $("fDescription").value = isEdit ? (metric.description || "") : "";

    var isEnabled = isEdit ? (metric.enabled !== false) : true;
    $("fEnabledCheck").checked = isEnabled;
    $("fEnabledText").textContent = isEnabled ? "状态：启用中" : "状态：已停用";

    modal.classList.add("show");
    modal.setAttribute("aria-hidden", "false");
    document.body.style.overflow = "hidden";

    setTimeout(function () {
      if (isEdit) {
        $("fTarget").focus();
      } else {
        $("fCodeInput").focus();
      }
    }, 150);
  }

  // 关闭右侧滑出抽屉
  function closeDrawer() {
    var modal = $("mDrawerModal");
    if (!modal) return;
    modal.classList.remove("show");
    modal.setAttribute("aria-hidden", "true");
    document.body.style.overflow = "";
  }

  // 保存表单
  function saveForm() {
    var code = $("fCodeInput").value.trim();
    var name = $("fName").value.trim();
    var target = $("fTarget").value.trim();

    if (!code) {
      alert("请输入指标编码！");
      $("fCodeInput").focus();
      return;
    }
    if (!name) {
      alert("请输入指标名称！");
      $("fName").focus();
      return;
    }
    if (!target) {
      alert("请输入考核达标目标值！");
      $("fTarget").focus();
      return;
    }

    var isEdit = $("fIsEdit").value === "1";
    var targetObj = null;

    if (isEdit) {
      targetObj = metricsList.find(function (m) { return m.code === code; });
      if (!targetObj) {
        alert("未找到原指标对象！");
        return;
      }
    } else {
      var exists = metricsList.some(function (m) { return m.code === code; });
      if (exists) {
        alert("指标编码 [" + code + "] 已存在，请勿重复添加！");
        $("fCodeInput").focus();
        return;
      }
      targetObj = { code: code };
      metricsList.unshift(targetObj);
    }

    targetObj.name = name;
    targetObj.category = $("fCategory").value;
    targetObj.ownerRole = $("fOwnerRole").value;
    targetObj.unit = $("fUnit").value.trim();
    targetObj.period = $("fPeriod").value;

    targetObj.target = target;
    targetObj.warningThreshold = $("fWarning").value.trim();
    targetObj.dangerThreshold = $("fDanger").value.trim();
    targetObj.goodDirection = $("fDirection").value;
    targetObj.dataSource = $("fDataSource").value.trim();

    targetObj.formula = $("fFormula").value.trim();
    targetObj.description = $("fDescription").value.trim();
    targetObj.enabled = $("fEnabledCheck").checked;

    saveLocalMetrics();
    closeDrawer();
    applyFilterAndRender();
    showToast("✓ 指标 [" + name + "] 配置已成功保存并生效！");
  }

  // 一键切换启用/停用
  function toggleMetricStatus(code) {
    var m = metricsList.find(function (item) { return item.code === code; });
    if (m) {
      m.enabled = !m.enabled;
      saveLocalMetrics();
      applyFilterAndRender();
      showToast("指标 [" + m.name + "] 已" + (m.enabled ? "启用" : "停用"));
    }
  }

  function initEvents() {
    $("mCreateBtn").addEventListener("click", function () {
      openDrawer(null);
    });

    $("mDrawerCloseBtn").addEventListener("click", closeDrawer);
    $("mDrawerCancelBtn").addEventListener("click", closeDrawer);
    $("mDrawerBackdrop").addEventListener("click", closeDrawer);

    $("mDrawerSaveBtn").addEventListener("click", saveForm);

    $("fEnabledCheck").addEventListener("change", function () {
      $("fEnabledText").textContent = this.checked ? "状态：启用中" : "状态：已停用";
    });

    $("mCategory").addEventListener("change", function () {
      state.category = this.value;
      applyFilterAndRender();
    });

    $("mRole").addEventListener("change", function () {
      state.role = this.value;
      applyFilterAndRender();
    });

    $("mState").addEventListener("change", function () {
      state.status = this.value;
      applyFilterAndRender();
    });

    $("mKeyword").addEventListener("input", function () {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(function () {
        state.keyword = $("mKeyword").value.trim();
        applyFilterAndRender();
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
      applyFilterAndRender();
    });

    $("mTbody").addEventListener("click", function (e) {
      var editBtn = e.target.closest("[data-metric-edit]");
      if (editBtn) {
        var code = editBtn.getAttribute("data-metric-edit");
        var m = metricsList.find(function (item) { return item.code === code; });
        if (m) { openDrawer(m); }
        return;
      }
      var toggleBtn = e.target.closest("[data-metric-toggle]");
      if (toggleBtn) {
        var c = toggleBtn.getAttribute("data-metric-toggle");
        toggleMetricStatus(c);
        return;
      }
    });

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" || e.keyCode === 27) {
        closeDrawer();
      }
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    metricsList = loadLocalMetrics();
    initEvents();
    applyFilterAndRender();
  });
})();
