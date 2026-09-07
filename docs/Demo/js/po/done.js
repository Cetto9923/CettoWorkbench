/* =============================================================================
   文件: web/static/js/po/done.js
   模块: PO 个人工作台 - 我的已办
   职责: 对齐老版本视图：4 张 KPI 指标卡、11 个对象 Chip 过滤栏、工具栏、
         九列表格、处理记录抽屉与分页。
   ============================================================================= */

(function () {
  "use strict";

  function esc(str) {
    if (str === null || str === undefined) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  var PL = window.PersonalList || {};
  var objectTypeBadgeFromKind = PL.objectTypeBadgeFromKind || function (kind) {
    var map = { demand: "business", story: "story", task: "task", bug: "bug", issue: "issue", todo: "todo", testtask: "testtask", approval: "approval" };
    var labels = { business: "业务需求", story: "研发需求", task: "任务", bug: "Bug", issue: "问题", todo: "待办", testtask: "测试单", approval: "审批" };
    var k = map[String(kind || "").toLowerCase()] || "";
    if (!k) { return '<span class="wb-type wb-type-unknown">' + esc(kind || "—") + "</span>"; }
    return '<span class="wb-type wb-type-' + k + '">' + labels[k] + "</span>";
  };

  var OBJECT_TYPE_ORDER = [
    "", "demand", "story", "task", "bug", "risk", "issue", "feedback", "release", "build", "todo"
  ];

  var OBJECT_TYPE_LABELS = {
    "": "全部已办",
    "demand": "业务需求",
    "story": "研发需求",
    "task": "任务",
    "bug": "Bug",
    "risk": "风险",
    "issue": "问题",
    "feedback": "反馈",
    "release": "发布",
    "build": "构建",
    "todo": "待办"
  };

  var OBJECT_TYPE_ICONS = {
    "demand": "fa-lightbulb",
    "task": "fa-list-check",
    "story": "fa-diagram-project",
    "bug": "fa-bug",
    "issue": "fa-circle-exclamation",
    "risk": "fa-triangle-exclamation",
    "todo": "fa-circle-check",
    "release": "fa-rocket",
    "build": "fa-cube",
    "feedback": "fa-comments"
  };

  var state = {
    mode: "core",
    timeRange: "all",
    objectType: "",
    action: "",
    result: "",
    project: "",
    keyword: "",
    page: 1,
    pageSize: 20
  };

  var metaData = {
    actions: [],
    results: [],
    projects: []
  };

  var requestSeq = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function fmtTwoLines(value) {
    if (!value) return "--";
    var m = String(value).match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/);
    return m ? m[1] + "<br>" + m[2] : esc(value);
  }

  function tagClass(result) {
    var map = {
      approved: "success", done: "success", resolved: "success", verified: "success",
      rejected: "danger", returned: "warning",
      closed: "gray",
      activated: "blue", submitted: "blue"
    };
    return map[result] || "gray";
  }

  function statusClass(s) {
    var t = String(s || "");
    if (t.indexOf("关闭") >= 0 || t.indexOf("closed") >= 0) return "gray";
    if (t.indexOf("完成") >= 0 || t.indexOf("done") >= 0 || t.indexOf("验收") >= 0 || t.indexOf("发布") >= 0 || t.indexOf("通过") >= 0) return "success";
    if (t.indexOf("待") >= 0 || t.indexOf("wait") >= 0 || t.indexOf("暂停") >= 0) return "warning";
    if (t.indexOf("驳回") >= 0 || t.indexOf("拒绝") >= 0 || t.indexOf("失败") >= 0) return "danger";
    if (t.indexOf("开发") >= 0 || t.indexOf("doing") >= 0 || t.indexOf("测试") >= 0 || t.indexOf("评审") >= 0) return "blue";
    return "gray";
  }

  /* ────────── 1. 元数据加载与下拉渲染 ────────── */
  function loadMeta() {
    fetch("/done/meta", { credentials: "same-origin" })
      .then(function (r) { return r.json(); })
      .then(function (res) {
        if (!res || !res.data) return;
        var d = res.data;
        metaData.actions = d.actionTypes || [];
        metaData.results = d.results || [];
        metaData.projects = d.projects || [];

        renderSelectOptions("doneAction", metaData.actions, "全部操作");
        renderSelectOptions("doneResult", metaData.results, "全部结果");
        renderSelectOptions("doneProject", metaData.projects, "全部项目");
      })
      .catch(function () {});
  }

  function renderSelectOptions(selectId, list, defaultLabel) {
    var el = $(selectId);
    if (!el) return;
    var html = '<option value="">' + esc(defaultLabel) + '</option>';
    (list || []).forEach(function (opt) {
      html += '<option value="' + esc(opt.key) + '">' + esc(opt.label) + '</option>';
    });
    el.innerHTML = html;
  }

  /* ────────── 2. 指标卡与 11 芯片渲染 ────────── */
  function renderSummaryKPIs(sum) {
    sum = sum || {};
    var today = Number(sum.today || 0);
    var week = Number(sum.week || 0);
    var month = Number(sum.month || 0);
    var objects = Number(sum.objects || 0);

    var elToday = $("kpiToday");
    var elWeek = $("kpiWeek");
    var elMonth = $("kpiMonth");
    var elObjects = $("kpiObjects");

    if (elToday) elToday.textContent = today;
    if (elWeek) elWeek.textContent = week;
    if (elMonth) elMonth.textContent = month;
    if (elObjects) elObjects.textContent = objects;

    document.querySelectorAll(".wb-done-kpi").forEach(function (card) {
      var r = card.getAttribute("data-range");
      card.classList.toggle("active", r === state.timeRange);
    });
  }

  function renderObjectChips(facets, total) {
    var host = $("doneObjectChips");
    if (!host) return;

    var countMap = {};
    var facetSum = 0;
    (facets || []).forEach(function (f) {
      countMap[f.key] = Number(f.count) || 0;
      facetSum += countMap[f.key];
    });

    var html = OBJECT_TYPE_ORDER.map(function (key) {
      var active = state.objectType === key;
      var count = key === "" ? facetSum : (countMap[key] || 0);
      var icon = OBJECT_TYPE_ICONS[key] ? '<i class="fas ' + OBJECT_TYPE_ICONS[key] + '"></i>' : "";
      return (
        '<button type="button" class="wb-done-tab' + (active ? " active" : "") + '" data-object-type="' + esc(key) + '" aria-pressed="' + (active ? "true" : "false") + '">' +
        icon + esc(OBJECT_TYPE_LABELS[key] || key) + '<span class="wb-done-tab-count">' + count + '</span>' +
        '</button>'
      );
    }).join("");

    host.innerHTML = html;

    host.querySelectorAll(".wb-done-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        state.objectType = btn.getAttribute("data-object-type") || "";
        state.page = 1;
        loadList();
      });
    });
  }

  /* ────────── 3. 列表加载与 9 列表格渲染 ────────── */
  function loadList() {
    var seq = ++requestSeq;
    var tbody = $("doneTbody");
    var empty = $("doneEmpty");
    var error = $("doneError");
    var pagEl = $("donePagination");

    if (tbody) tbody.innerHTML = '<tr><td colspan="9" class="state-placeholder">正在拉取已办记录…</td></tr>';
    if (empty) empty.hidden = true;
    if (error) error.hidden = true;

    var params = new URLSearchParams({
      mode: state.mode,
      timeRange: state.timeRange,
      objectType: state.objectType,
      action: state.action,
      result: state.result,
      project: state.project,
      keyword: state.keyword,
      page: String(state.page),
      pageSize: String(state.pageSize)
    });

    fetch("/done/items?" + params.toString(), { credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok) throw new Error("HTTP " + r.status);
        return r.json();
      })
      .then(function (res) {
        if (seq !== requestSeq) return;
        var data = res.data || res;
        var items = data.items || [];
        var total = data.total || 0;

        renderSummaryKPIs(data.summary);
        renderObjectChips(data.facets, total);

        if (!items.length) {
          if (tbody) tbody.innerHTML = "";
          if (empty) empty.hidden = false;
          if (pagEl) pagEl.hidden = true;
          return;
        }

        var html = items.map(function (it) {
          var titleCell = "";
          if (it.objectType === "demand" && window.DemandDetail && typeof window.DemandDetail.open === "function") {
            titleCell = '<button type="button" class="wb-done-title-link" data-open-demand="' + it.objectId + '">' + esc(it.objectTitle || it.objectName || "--") + '</button>';
          } else if (it.url) {
            titleCell = '<a class="wb-done-title-link" href="' + esc(it.url) + '" target="_blank" rel="noopener noreferrer">' + esc(it.objectTitle || it.objectName || "--") + '</a>';
          } else {
            titleCell = '<span class="wb-done-title-plain">' + esc(it.objectTitle || it.objectName || "--") + '</span>';
          }

          var changeHtml = (it.beforeStatus && it.afterStatus)
            ? '<span class="done-change-before">' + esc(it.beforeStatus) + '</span><span class="done-arrow">→</span><span class="done-change-after">' + esc(it.afterStatus) + '</span>'
            : '<span class="done-change-na">--</span>';

          var ctxHtml = '<div class="done-ctx-project">' + esc(it.projectName || "--") + '</div>' +
            (it.executionName ? '<div class="done-ctx-exec">' + esc(it.executionName) + '</div>' : "");

          return (
            '<tr>' +
            '<td class="done-time">' + fmtTwoLines(it.handledAt || it.date) + '</td>' +
            '<td class="done-obj">' + objectTypeBadgeFromKind(it.objectType) + '<span class="done-obj-code">' + esc(it.objectId) + '</span></td>' +
            '<td class="done-title">' + titleCell + '</td>' +
            '<td class="done-action"><span class="done-action-name">' + esc(it.actionName || it.action) + '</span></td>' +
            '<td class="done-result"><span class="done-tag ' + tagClass(it.resultCode || it.result) + '">' + esc(it.resultText || it.result || "--") + '</span></td>' +
            '<td class="done-change">' + changeHtml + '</td>' +
            '<td class="done-ctx">' + ctxHtml + '</td>' +
            '<td class="done-status"><span class="done-tag ' + statusClass(it.currentStatus) + '">' + esc(it.currentStatus || "--") + '</span></td>' +
            '<td class="done-op"><button type="button" class="action-btn small wb-done-detail-btn" data-action-id="' + it.id + '">查看记录</button></td>' +
            '</tr>'
          );
        }).join("");

        if (tbody) tbody.innerHTML = html;

        // 绑定行内事件
        tbody.querySelectorAll("[data-open-demand]").forEach(function (btn) {
          btn.addEventListener("click", function () {
            var did = btn.getAttribute("data-open-demand");
            window.DemandDetail.open(did);
          });
        });

        tbody.querySelectorAll("[data-action-id]").forEach(function (btn) {
          btn.addEventListener("click", function () {
            var aid = btn.getAttribute("data-action-id");
            openDetailDrawer(aid);
          });
        });

        // 统一分页
        if (pagEl && window.PersonalList && typeof window.PersonalList.renderPagination === "function") {
          window.PersonalList.renderPagination({
            container: pagEl,
            page: state.page,
            pageSize: state.pageSize,
            total: total,
            onPageChange: function (p) {
              state.page = p;
              loadList();
            },
            onPageSizeChange: function (ps) {
              state.pageSize = ps;
              state.page = 1;
              window.PersonalList.savePageSize("po.done.pageSize", ps);
              loadList();
            }
          });
        }
      })
      .catch(function (err) {
        if (seq !== requestSeq) return;
        if (tbody) tbody.innerHTML = "";
        if (error) error.hidden = false;
      });
  }

  /* ────────── 4. 查看记录详情抽屉 ────────── */
  function openDetailDrawer(actionId) {
    var mask = $("doneDrawerMask");
    var body = $("doneDrawerBody");
    var openObjBtn = $("doneDrawerOpenObj");
    if (!mask || !body) return;

    body.innerHTML = '<div class="state-placeholder">加载记录详情中…</div>';
    mask.hidden = false;

    fetch("/done/detail/" + actionId, { credentials: "same-origin" })
      .then(function (r) { return r.json(); })
      .then(function (res) {
        if (!res || !res.data) {
          body.innerHTML = '<div class="state-placeholder error">加载详情失败</div>';
          return;
        }
        var it = res.data.item || {};
        var ctx = res.data.context || {};
        var tl = res.data.timeline || [];

        if (openObjBtn) {
          if (it.url) {
            openObjBtn.href = it.url;
            openObjBtn.hidden = false;
          } else {
            openObjBtn.hidden = true;
          }
        }

        var changeText = (it.beforeStatus && it.afterStatus) ? (it.beforeStatus + " → " + it.afterStatus) : "--";

        var timelineHtml = tl.length ? tl.map(function (x) {
          return (
            '<div class="done-tl' + (x.isCurrent ? " current" : "") + '">' +
            '  <div class="done-tl-title">' + esc(x.actionName) + '</div>' +
            '  <div class="done-tl-meta">' + esc(x.actorName) + ' · ' + esc(x.occurredAt) + '</div>' +
            '</div>'
          );
        }).join("") : '<div class="done-tl-na">暂无历史时间线</div>';

        body.innerHTML =
          '<section class="done-section">' +
          '  <div class="done-section-title">本次办理摘要</div>' +
          '  <div class="done-summary-box">' +
          '    <div class="done-kv">' +
          '      <span class="done-kv-k">业务对象</span><span class="done-kv-v done-kv-wide">' + esc(it.objectCode) + ' · ' + esc(it.objectTitle || it.objectName || "--") + '</span>' +
          '      <span class="done-kv-k">来源</span><span class="done-kv-v">禅道</span>' +
          '      <span class="done-kv-k">我做了什么</span><span class="done-kv-v done-kv-action">' + esc(it.actionName) + '</span>' +
          '      <span class="done-kv-k">处理结果</span><span class="done-kv-v">' + esc(it.resultText) + '</span>' +
          '      <span class="done-kv-k">办理人</span><span class="done-kv-v">' + esc(it.actorName) + '</span>' +
          '      <span class="done-kv-k">处理时间</span><span class="done-kv-v">' + esc(it.date || it.handledAt) + '</span>' +
          '      <span class="done-kv-k">状态变化</span><span class="done-kv-v">' + esc(changeText) + '</span>' +
          '      <span class="done-kv-k">下一责任人</span><span class="done-kv-v">' + esc(it.nextOwnerName || "--") + '</span>' +
          '    </div>' +
          '  </div>' +
          '</section>' +
          '<section class="done-section">' +
          '  <div class="done-section-title">对象上下文</div>' +
          '  <div class="done-summary-box">' +
          '    <div class="done-kv">' +
          '      <span class="done-kv-k">所属产品</span><span class="done-kv-v">' + esc(ctx.productName || "--") + '</span>' +
          '      <span class="done-kv-k">所属项目</span><span class="done-kv-v">' + esc(ctx.projectName || "--") + '</span>' +
          '      <span class="done-kv-k">执行 / 迭代</span><span class="done-kv-v">' + esc(ctx.executionName || "--") + '</span>' +
          '      <span class="done-kv-k">当前状态</span><span class="done-kv-v">' + esc(ctx.currentStatus || "--") + '</span>' +
          '      <span class="done-kv-k">当前负责人</span><span class="done-kv-v">' + esc(ctx.currentOwner || "--") + '</span>' +
          '    </div>' +
          '  </div>' +
          '</section>' +
          '<section class="done-section">' +
          '  <div class="done-section-title">邻近历史</div>' +
          '  <div class="done-timeline">' + timelineHtml + '</div>' +
          '</section>';
      })
      .catch(function () {
        body.innerHTML = '<div class="state-placeholder error">加载详情失败，请稍后重试</div>';
      });
  }

  function closeDetailDrawer() {
    var mask = $("doneDrawerMask");
    if (mask) mask.hidden = true;
  }

  /* ────────── 5. 事件绑定与初始化 ────────── */
  function initEvents() {
    // 4 张 KPI 卡点击筛选时间段
    document.querySelectorAll(".wb-done-kpi").forEach(function (card) {
      card.addEventListener("click", function () {
        var r = card.getAttribute("data-range");
        if (!r || r === "objects") return;
        state.timeRange = r;
        var sel = $("doneTimeRange");
        if (sel) sel.value = r;
        state.page = 1;
        loadList();
      });
    });

    // 核心已办 / 全部操作 切换
    document.querySelectorAll(".wb-done-seg").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".wb-done-seg").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.mode = btn.getAttribute("data-mode") || "core";
        state.page = 1;
        loadList();
      });
    });

    // 搜索框
    var kwInput = $("doneKeyword");
    if (kwInput) {
      var debounceTimer = null;
      kwInput.addEventListener("input", function () {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(function () {
          state.keyword = kwInput.value.trim();
          state.page = 1;
          loadList();
        }, 300);
      });
    }

    // 时间范围下拉
    var timeSel = $("doneTimeRange");
    if (timeSel) {
      timeSel.addEventListener("change", function () { state.timeRange = timeSel.value; state.page = 1; loadList(); });
    }
    var actSel = $("doneAction");
    if (actSel) {
      actSel.addEventListener("change", function () { state.action = actSel.value; state.page = 1; loadList(); });
    }
    var resSel = $("doneResult");
    if (resSel) {
      resSel.addEventListener("change", function () { state.result = resSel.value; state.page = 1; loadList(); });
    }
    var prjSel = $("doneProject");
    if (prjSel) {
      prjSel.addEventListener("change", function () { state.project = prjSel.value; state.page = 1; loadList(); });
    }

    var resetBtn = $("doneResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.timeRange = "all"; state.objectType = ""; state.action = ""; state.result = "";
        state.project = ""; state.keyword = ""; state.mode = "core"; state.page = 1;
        if (kwInput) kwInput.value = "";
        if (timeSel) timeSel.value = "all";
        if (actSel) actSel.value = "";
        if (resSel) resSel.value = "";
        if (prjSel) prjSel.value = "";
        document.querySelectorAll(".wb-done-seg").forEach(function (b) {
          b.classList.toggle("active", b.getAttribute("data-mode") === "core");
        });
        loadList();
      });
    }

    // 重试按钮
    var retryBtn = $("doneRetryBtn");
    if (retryBtn) retryBtn.addEventListener("click", loadList);

    // 抽屉关闭事件
    var dClose = $("doneDrawerClose");
    var dCloseBtn = $("doneDrawerCloseBtn");
    var dMask = $("doneDrawerMask");
    if (dClose) dClose.addEventListener("click", closeDetailDrawer);
    if (dCloseBtn) dCloseBtn.addEventListener("click", closeDetailDrawer);
    if (dMask) {
      dMask.addEventListener("click", function (e) {
        if (e.target === dMask) closeDetailDrawer();
      });
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    state.pageSize = window.PersonalList.loadPageSize("po.done.pageSize", state.pageSize, [10, 15, 20, 30, 50]);
    initEvents();
    loadMeta();
    loadList();
  });
})();
