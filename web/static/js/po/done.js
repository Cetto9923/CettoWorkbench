/* =============================================================================
   文件: web/static/js/po/done.js
   模块: PO 个人工作台 - 我的已办
   职责: 对齐老版本视图：4 张 KPI 指标卡、11 个对象 Chip 过滤栏、工具栏、
         九列表格、处理记录抽屉与分页。
   ============================================================================= */

(function () {
  "use strict";

  var PAGE_SIZE_OPTIONS = window.PersonalList.PAGE_SIZE_OPTIONS;

  var esc = window.escapeHtml;
  var PL = window.PersonalList || {};
  // 把 ZenTao API kind（demand/story/task/bug/charter/...）映射成 wb-type CSS kind
  // （business/story/...），与 PersonalList.OBJECT_KIND_FROM_API / wb-priority.css 色板对齐。
  var objectChipKindFromApi = function (apiKind) {
    var map = { demand: "business", business: "business", story: "story", independent_story: "independent_story", task: "task", bug: "bug", issue: "issue", todo: "todo", testtask: "testtask", test: "testtask", approval: "approval", charter: "charter", planchange: "approval", buildguideline: "approval", review: "approval", case: "testcase", feedback: "feedback", release: "charter", build: "charter", risk: "risk", project: "project", mail: "mail" };
    var k = map[String(apiKind || "").toLowerCase()] || "";
    return k === "testcase" ? "testtask" : k;
  };
  var OBJECT_TYPE_ORDER = ["", "approval", "demand", "story", "task", "bug", "risk", "issue", "feedback", "release", "build", "todo"];
  var OBJECT_TYPE_LABELS = { "": "全部已办", approval: "审批", demand: "业务需求", story: "研发需求", task: "任务", bug: "Bug", risk: "风险", issue: "问题", feedback: "反馈", release: "发布", build: "构建", todo: "待办", charter: "章程" };
  var OBJECT_TYPE_ICONS = { demand: "fa-lightbulb", approval: "fa-stamp", task: "fa-list-check", story: "fa-diagram-project", bug: "fa-bug", issue: "fa-circle-exclamation", risk: "fa-triangle-exclamation", todo: "fa-circle-check", release: "fa-rocket", build: "fa-cube", feedback: "fa-comments", charter: "fa-file-contract" };
  var state = {
    mode: "core",
    tab: "all",
    timeRange: "all",
    objectType: "",
    action: "",
    result: "",
    project: "",
    keyword: "",
    page: 1,
    pageSize: 20
  };

  var metaData = { actions: [], results: [], projects: [] }; var requestSeq = 0;
  var metaSeq = 0;

  function $(id) { return document.getElementById(id); }
  function fmtDateTime(value) {
    if (!value) return '<span class="done-time-date">--</span>';
    var m = String(value).match(/^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/);
    if (!m) return '<span class="done-time-date">' + esc(value) + "</span>";
    return '<span class="done-time-date">' + m[1] + '</span><span class="done-time-clock">' + m[2] + "</span>";
  }

  function tagClass(result) {
    var map = { approved: "success", done: "success", resolved: "success", verified: "success", rejected: "danger", returned: "warning", closed: "gray", activated: "blue", submitted: "blue" };
    return map[result] || "gray";
  }

  function statusClass(s) {
    var t = String(s || "").toLowerCase();
    if (t.indexOf("关闭") >= 0 || t.indexOf("closed") >= 0) return "gray";
    if (t.indexOf("完成") >= 0 || t.indexOf("done") >= 0 || t.indexOf("验收") >= 0 || t.indexOf("发布") >= 0 || t.indexOf("通过") >= 0) return "success";
    if (t.indexOf("待") >= 0 || t.indexOf("wait") >= 0 || t.indexOf("暂停") >= 0) return "warning";
    if (t.indexOf("驳回") >= 0 || t.indexOf("拒绝") >= 0 || t.indexOf("失败") >= 0) return "danger";
    if (t.indexOf("开发") >= 0 || t.indexOf("doing") >= 0 || t.indexOf("测试") >= 0 || t.indexOf("评审") >= 0 || t.indexOf("active") >= 0) return "blue";
    return "gray";
  }

  function statusLabel(status, objectType) {
    var raw = String(status || "").trim(), key = raw.toLowerCase(), obj = String(objectType || "").toLowerCase();
    var storyLabels = { draft: "草稿", reviewing: "评审中", active: "激活", changing: "变更中", closed: "已关闭" };
    var demandLabels = { draft: "暂存", wait: "待评审", active: "已评审", clarified: "已澄清", developing: "开发中", testing: "测试中", waitacceptance: "待验收", acceptanced: "已验收", waitdeliver: "待交付", delivered: "已交付", released: "已发布", closed: "已关闭", refuse: "已挂起" };
    var charterLabels = { wait: "待审批", doing: "审批中", reviewed: "已审批", reject: "已驳回", closed: "已关闭" };
    var approvalLabels = { wait: "待审批", doing: "审批中", reviewed: "已审批", approve: "已通过", reject: "已驳回", closed: "已关闭" };
    var labels = { draft: "草稿", wait: "待处理", doing: "进行中", done: "已完成", pause: "已暂停", cancel: "已取消", closed: "已关闭", reviewing: "评审中", active: "已评审", changing: "变更中", clarified: "已澄清", developing: "开发中", testing: "测试中", waitacceptance: "待验收", acceptanced: "已验收", waitdeliver: "待交付", delivered: "已交付", released: "已发布", planned: "已排期", refuse: "已驳回", suspended: "已挂起", blocked: "已阻塞", opened: "处理中", resolved: "已解决", verified: "已验证", unconfirmed: "未确认" };
    if (obj === "story" && storyLabels[key]) return storyLabels[key];
    if ((obj === "demand" || !obj) && demandLabels[key]) return demandLabels[key];
    if (obj === "charter" && charterLabels[key]) return charterLabels[key];
    if ((obj === "buildguideline" || obj === "planchange" || obj === "review") && approvalLabels[key]) return approvalLabels[key];
    return labels[key] || raw || "--";
  }
  function actionLabel(action) {
    var raw = String(action || "").trim(), key = raw.toLowerCase();
    var map = { linkstory: "关联需求", unlinkstory: "移除需求关联", opened: "创建", created: "创建", edited: "编辑", closed: "关闭", deleted: "删除", assigned: "指派", started: "开始", finished: "完成", commented: "添加备注", reviewed: "评审", resolved: "解决", activated: "激活", confirmed: "确认" };
    return map[key] || raw || "--";
  }

  function objectCode(item) {
    var id = String((item && item.objectId) || "").trim();
    if (!id) return "--";
    return String((item && item.objectType) || "").toLowerCase() === "demand" ? "US" + id : id;
  }
  function loadMeta(objectType) {
    var seq = ++metaSeq;
    objectType = String(objectType || "").trim();
    function setEnabled(on) { ["doneAction", "doneResult", "doneProject"].forEach(function (id) { var el = $(id); if (el) el.disabled = !on; }); }
    setEnabled(false);
    renderSelectOptions("doneAction", [], "全部操作"); renderSelectOptions("doneResult", [], "全部结果"); renderSelectOptions("doneProject", [], "全部项目");
    var url = "/done/meta" + (objectType ? "?objectType=" + encodeURIComponent(objectType) : "");
    return fetch(url, { credentials: "same-origin" })
      .then(function (r) { if (!r.ok) throw new Error("HTTP " + r.status); return r.json(); })
      .then(function (res) {
        if (seq !== metaSeq) return false;
        if (!res || !res.data) throw new Error("empty meta");
        var d = res.data;
        metaData.actions = d.actionTypes || []; metaData.results = d.results || []; metaData.projects = d.projects || [];
        renderSelectOptions("doneAction", metaData.actions, "全部操作");
        renderSelectOptions("doneResult", metaData.results, "全部结果");
        renderSelectOptions("doneProject", metaData.projects, "全部项目");
        setEnabled(true);
        return true;
      })
      .catch(function () {
        if (seq !== metaSeq) return false;
        setEnabled(false);
        window.showToast("筛选条件加载失败，请稍后重试", "error");
        return false;
      });
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
  function renderSummaryKPIs(sum) {
    sum = sum || {};
    var today = Number(sum.today || 0), week = Number(sum.week || 0), month = Number(sum.month || 0), objects = Number(sum.objects || 0);
    var elToday = $("kpiToday"), elWeek = $("kpiWeek"), elMonth = $("kpiMonth"), elObjects = $("kpiObjects");
    if (elToday) elToday.textContent = today;
    if (elWeek) elWeek.textContent = week;
    if (elMonth) elMonth.textContent = month;
    if (elObjects) elObjects.textContent = objects;

    document.querySelectorAll(".wb-done-kpi").forEach(function (card) {
      var r = card.getAttribute("data-range");
      var isActive = r === state.timeRange;
      card.classList.toggle("active", isActive);
      card.setAttribute("aria-pressed", isActive ? "true" : "false");
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
      var active = key === "approval" ? state.tab === "approval" : state.tab !== "approval" && state.objectType === key;
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
        var key = btn.getAttribute("data-object-type") || "";
        state.tab = key === "approval" ? "approval" : "all";
        state.objectType = key === "approval" ? "" : key;
        state.action = ""; state.result = ""; state.project = "";
        state.page = 1;
        loadMeta(key).then(function () { loadList(); });
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
      tab: state.tab,
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

        var APPROVAL_OBJECT_TYPES = ["charter", "planchange", "buildguideline", "review", "case"];
        var html = items.map(function (it) {
          var displayTitle = it.objectTitle || it.objectName || "";
          if (!displayTitle) {
            // 审批类对象即便 URL 为空也必须保持非空显示，避免 "--"
            if (APPROVAL_OBJECT_TYPES.indexOf(String(it.objectType || "").toLowerCase()) >= 0) {
              var label = OBJECT_TYPE_LABELS[it.objectType] || it.objectType || "";
              var code = objectCode(it);
              displayTitle = (label && code !== "--") ? (label + " " + code) : (label || code || "");
            }
          }
          if (!displayTitle) displayTitle = "--";
          var titleCell = "";
          if (it.objectType === "demand" && window.DemandDetail && typeof window.DemandDetail.open === "function") {
            titleCell = '<button type="button" class="wb-done-title-link" data-open-demand="' + it.objectId + '">' + esc(displayTitle) + '</button>';
          } else if (it.url) {
            titleCell = '<a class="wb-done-title-link" href="' + esc(it.url) + '" target="_blank" rel="noopener noreferrer">' + esc(displayTitle) + '</a>';
          } else {
            titleCell = '<span class="wb-done-title-plain">' + esc(displayTitle) + '</span>';
          }

          var changeHtml = (it.beforeStatus && it.afterStatus)
            ? '<span class="done-change-before">' + esc(statusLabel(it.beforeStatus, it.objectType)) + '</span><span class="done-arrow">→</span><span class="done-change-after">' + esc(statusLabel(it.afterStatus, it.objectType)) + '</span>'
            : '<span class="done-change-na">--</span>';

          var ctxHtml = '<div class="done-ctx-pool">' + esc(it.poolName || "--") + '</div>' +
            '<div class="done-ctx-project">' + esc(it.projectName || "--") + '</div>' +
            (it.executionName ? '<div class="done-ctx-exec">' + esc(it.executionName) + '</div>' : "");

          var rawCode = objectCode(it);
          var idHtml = rawCode === "--"
            ? '<span class="done-obj-id-plain">--</span>'
            : (it.url
                ? '<a class="done-obj-id-link" href="' + esc(it.url) + '" target="_blank" rel="noopener noreferrer">' + esc(rawCode) + '</a>'
                : '<span class="done-obj-id-plain">' + esc(rawCode) + '</span>');
          // 对象 + ID 单 chip：复用 PersonalList.idChipHtml（与首页/通知中心同款），走
          // OBJECT_TYPE_SHORT_LABELS 缩写 + "#" 分隔符；不再走本地 .done-obj-chip-* 别名。
          var chipKind = objectChipKindFromApi(it.objectType);
          var objCell = (PL && PL.idChipHtml)
            ? PL.idChipHtml(chipKind, idHtml)
            : '<span class="done-obj-chip done-obj-chip-' + esc(chipKind || "unknown") + '">' +
                '<span class="done-obj-chip-label">' + esc(it.objectType || "—") + '</span> ' +
                idHtml +
                '</span>';

          var stLabel = statusLabel(it.currentStatus, it.objectType);
          var stHtml = (PL && PL.statusTagHtml) ? PL.statusTagHtml(stLabel) : ('<span class="done-tag ' + statusClass(it.currentStatus) + '">' + esc(stLabel) + '</span>');

          return (
            '<tr>' +
            '<td class="done-obj">' + objCell + '</td>' +
            '<td class="done-title">' + titleCell + '</td>' +
            '<td class="done-action"><span class="done-action-name">' + esc(actionLabel(it.actionName || it.action)) + '</span></td>' +
            '<td class="done-result"><span class="done-tag ' + tagClass(it.resultCode || it.result) + '">' + esc(it.resultText || it.result || "--") + '</span></td>' +
            '<td class="done-change">' + changeHtml + '</td>' +
            '<td class="done-ctx">' + ctxHtml + '</td>' +
            '<td class="done-status">' + stHtml + '</td>' +
            '<td class="done-time">' + fmtDateTime(it.handledAt || it.date) + '</td>' +
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

        var changeText = (it.beforeStatus && it.afterStatus) ? (statusLabel(it.beforeStatus, it.objectType) + " → " + statusLabel(it.afterStatus, it.objectType)) : "未记录状态变化";

        var timelineHtml = tl.length ? tl.map(function (x) {
          return (
            '<div class="done-tl' + (x.isCurrent ? " current" : "") + '">' +
            '  <div class="done-tl-title">' + esc(x.actionName) + '</div>' +
            '  <div class="done-tl-meta">' + esc(x.actorName) + ' · ' + esc(x.occurredAt) + '</div>' +
            '</div>'
          );
        }).join("") : '<div class="done-tl-na">暂无历史时间线</div>';

        body.innerHTML =
          '<section class="done-section"><div class="done-section-title">本次办理摘要</div><div class="done-summary-box"><div class="done-kv">' +
          '<span class="done-kv-k">业务对象</span><span class="done-kv-v done-kv-wide">' + esc(it.objectCode) + ' · ' + esc(it.objectTitle || it.objectName || "--") + '</span>' +
          '<span class="done-kv-k">来源</span><span class="done-kv-v">禅道</span>' +
          '<span class="done-kv-k">我做了什么</span><span class="done-kv-v done-kv-action">' + esc(actionLabel(it.actionName)) + '</span>' +
          '<span class="done-kv-k">处理结果</span><span class="done-kv-v">' + esc(it.resultText) + '</span>' +
          '<span class="done-kv-k">办理人</span><span class="done-kv-v">' + esc(it.actorName) + '</span>' +
          '<span class="done-kv-k">处理时间</span><span class="done-kv-v">' + esc(it.date || it.handledAt) + '</span>' +
          '<span class="done-kv-k">状态变化</span><span class="done-kv-v">' + esc(changeText) + '</span>' +
          '<span class="done-kv-k">下一责任人</span><span class="done-kv-v">' + esc(it.nextOwnerName || "--") + '</span></div></div></section>' +
          '<section class="done-section"><div class="done-section-title">对象上下文</div><div class="done-summary-box"><div class="done-kv">' +
          '<span class="done-kv-k">所属产品</span><span class="done-kv-v">' + esc(ctx.productName || "--") + '</span>' +
          '<span class="done-kv-k">所属需求池</span><span class="done-kv-v">' + esc(ctx.poolName || "--") + '</span>' +
          '<span class="done-kv-k">所属项目</span><span class="done-kv-v">' + esc(ctx.projectName || "--") + '</span>' +
          '<span class="done-kv-k">执行 / 迭代</span><span class="done-kv-v">' + esc(ctx.executionName || "--") + '</span>' +
          '<span class="done-kv-k">当前状态</span><span class="done-kv-v">' + esc(statusLabel(ctx.currentStatus, it.objectType)) + '</span>' +
          '<span class="done-kv-k">当前负责人</span><span class="done-kv-v">' + esc(ctx.currentOwner || "--") + '</span>' +
          '</div></div></section>' +
          '<section class="done-section"><div class="done-section-title">邻近历史</div><div class="done-timeline">' + timelineHtml + '</div></section>';
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
        state.timeRange = (state.timeRange === r ? "all" : r);
        var sel = $("doneTimeRange");
        if (sel) sel.value = state.timeRange;
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
        loadMeta("").then(function () { loadList(); });
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
    state.pageSize = window.PersonalList.loadPageSize("po.done.pageSize", state.pageSize, PAGE_SIZE_OPTIONS);
    initEvents();
    loadMeta();
    loadList();
  });
})();
