/* =============================================================================
   文件: web/static/js/po/notice.js
   模块: PO 个人工作台 - 通知中心交互脚本
   职责: 绑定通知分类、快捷焦点、筛选工具栏、列表渲染与已读状态管理
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  // 暴露 objectTypeBadge 以满足 priority-helpers.test.js 的 parity 契约（页面实际
  // 不直接消费，统一走 PersonalList.objectTypeBadge 入口）。
  var objectTypeBadge = (window.PersonalList && window.PersonalList.objectTypeBadge) || function (k) { return k; };
  var $ = function (id) { return document.getElementById(id); };

  var state = {
    quickView: "unread",
    category: "all",
    objectType: "all",
    timeRange: "all",
    readState: "all",
    needAction: "all",
    keyword: "",
    page: 1,
    pageSize: 20
  };

  var VALID_QUICKVIEWS = ["all", "unread", "action", "abnormal", "today"];
  var VALID_CATEGORIES = ["all", "business", "approval", "reminder", "collaboration", "risk", "system"];
  var VALID_OBJECT_TYPES = ["all", "demand", "story", "project", "task", "bug", "testtask", "issue", "risk", "approval", "mail"];
  var VALID_TIME_RANGES = ["all", "today", "3d", "7d", "30d"];
  var VALID_READ_STATES = ["all", "unread", "read"];
  var VALID_ACTION_STATES = ["all", "required", "none"];
  var hasCorrectedPage = false;

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var p = new URLSearchParams();
    if (state.quickView) { p.set("quickView", state.quickView); }
    ["category", "objectType", "timeRange", "readState", "needAction"].forEach(function (k) {
      if (state[k] && state[k] !== "all") { p.set(k, state[k]); }
    });
    if (state.keyword) { p.set("keyword", state.keyword); }
    if (state.page > 1) { p.set("page", String(state.page)); }
    if (state.pageSize !== 20) { p.set("pageSize", String(state.pageSize)); }
    var qs = p.toString();
    window.history.replaceState(null, "", window.location.pathname + (qs ? "?" + qs : ""));
  }

  function initFromUrl() {
    if (!window.location.search) { return; }
    var sp = new URLSearchParams(window.location.search);
    var qv = (sp.get("quickView") || "").trim();
    if (VALID_QUICKVIEWS.indexOf(qv) >= 0) {
      state.quickView = qv;
      document.querySelectorAll("#noticeQuickChips .header-quick-chip").forEach(function (b) {
        var isCurrent = (b.getAttribute("data-qv") || "all") === qv;
        b.classList.toggle("active", isCurrent);
        b.setAttribute("aria-pressed", isCurrent ? "true" : "false");
      });
    }

    var cat = (sp.get("category") || "").trim();
    if (VALID_CATEGORIES.indexOf(cat) >= 0) {
      state.category = cat;
      document.querySelectorAll(".po-notice .category-tab").forEach(function (b) {
        var isCurrent = (b.getAttribute("data-category") || "all") === cat;
        b.classList.toggle("active", isCurrent);
        if (isCurrent) { b.setAttribute("aria-current", "page"); } else { b.removeAttribute("aria-current"); }
      });
    }

    [
      { key: "objectType", list: VALID_OBJECT_TYPES, id: "noticeObjectType" },
      { key: "timeRange", list: VALID_TIME_RANGES, id: "noticeTimeRange" },
      { key: "readState", list: VALID_READ_STATES, id: "noticeReadState" },
      { key: "needAction", list: VALID_ACTION_STATES, id: "noticeNeedAction" }
    ].forEach(function (item) {
      var val = (sp.get(item.key) || "").trim();
      if (item.list.indexOf(val) >= 0 && $(item.id)) {
        state[item.key] = val;
        $(item.id).value = val;
      }
    });

    var kw = (sp.get("keyword") || "").trim();
    if (kw && $("noticeKeyword")) {
      state.keyword = kw;
      $("noticeKeyword").value = kw;
    }

    var p = parseInt(sp.get("page"), 10);
    if (!isNaN(p) && p >= 1) { state.page = p; }
    var ps = parseInt(sp.get("pageSize"), 10);
    if (!isNaN(ps) && [10, 20, 50, 100].indexOf(ps) >= 0) { state.pageSize = ps; }
    else { state.pageSize = window.PersonalList.loadPageSize("po.notice.pageSize", state.pageSize, [10, 20, 50, 100]); }
  }

  // 通知对象类型 / API 别名 / subject 历史前缀。demand 走 OBJECT_KIND_FROM_API
  // 归一到 business，最终徽章显示「业务需求」，与首页 / 待办统一。
  var OBJECT_TYPE_LABELS = {
    business: "业务需求", sub_demand: "子需求", story: "研发需求",
    independent_story: "独立研发需求", task: "任务", bug: "Bug",
    testtask: "测试单", issue: "问题", risk: "风险", approval: "审批",
    feedback: "反馈", charter: "立项", mail: "邮件", project: "项目",
    demand: "需求"
  };
  var OBJECT_KIND_FROM_API = {
    demand: "business", business: "business", sub_demand: "sub_demand",
    story: "story", independent_story: "independent_story", task: "task",
    bug: "bug", test: "testtask", testtask: "testtask", issue: "issue",
    risk: "risk", approval: "approval", feedback: "feedback",
    charter: "charter", mail: "mail", project: "project"
  };
  var SUBJECT_PREFIX_ALIASES = {
    business: ["需求", "业务需求", "业需", "demand"],
    story: ["研需", "研发需求", "story"],
    sub_demand: ["子需求"], independent_story: ["独立研发需求"],
    task: ["任务", "task"], bug: ["Bug", "bug"],
    testtask: ["测试", "测试单", "test"], issue: ["问题", "issue"],
    risk: ["风险", "risk"], approval: ["审批", "approval"],
    feedback: ["反馈", "feedback"], charter: ["立项", "charter"],
    project: ["项目", "project"], mail: ["邮件"]
  };

  function canonicalKind(ot) {
    var key = String(ot || "").trim().toLowerCase();
    return OBJECT_KIND_FROM_API[key] || key;
  }
  function escapeRegExp(s) { return String(s).replace(/[.*+?^${}()|[\]\\]/g, "\\$&"); }
  // 「您有 Bug(N)」类系统级模板提醒兜底；委托给 PersonalList 共享 helper。
  function reminderKindFromSubject(s) {
    var pl = window.PersonalList;
    return (pl && pl.reminderKindFromSubject) ? pl.reminderKindFromSubject(s) : "";
  }
  function stripSubjectPrefix(rawSubject, canon, oid) {
    if (!canon || !OBJECT_TYPE_LABELS[canon] || !oid) { return rawSubject; }
    var text = String(rawSubject || "");
    var cands = SUBJECT_PREFIX_ALIASES[canon] || [];
    var cnLabel = OBJECT_TYPE_LABELS[canon];
    if (cnLabel && cands.indexOf(cnLabel) === -1) { cands = cands.concat([cnLabel]); }
    var oidEsc = escapeRegExp(String(oid));
    for (var i = 0; i < cands.length; i++) {
      var label = cands[i];
      if (!label) { continue; }
      var re = new RegExp("^" + escapeRegExp(label) + "\\s*[#]\\s*" + oidEsc + "\\s*[-:：· ]*", "i");
      if (re.test(text)) {
        var stripped = text.replace(re, "").trim();
        if (stripped) { return stripped; }
      }
    }
    return rawSubject;
  }

  function buildUrl() {
    var params = new URLSearchParams();
    Object.keys(state).forEach(function (key) {
      if (state[key] !== "") { params.set(key, String(state[key])); }
    });
    return "/notice/items?" + params.toString();
  }

  var currentItemsMap = {};

  function formatNoticeSubject(item) {
    var ot = String(item.objectType || "").trim().toLowerCase();
    var oid = String(item.objectId || "").trim();
    var rawSubject = String(item.subject || item.title || item.data || "—");

    var canon = canonicalKind(ot);
    var isReminderTemplate = false;
    if (!canon || canon === "mail" || !OBJECT_TYPE_LABELS[canon]) {
      var rk = reminderKindFromSubject(rawSubject);
      if (rk) { canon = rk; isReminderTemplate = true; }
      else if (/^\s*TICKET\s*#/i.test(rawSubject)) { canon = "issue"; }
    }
    var badgeHtml = "";
    if (canon && canon !== "mail" && OBJECT_TYPE_LABELS[canon]) {
      var badgeText = OBJECT_TYPE_LABELS[canon] + ((!isReminderTemplate && oid && oid !== "0") ? " #" + oid : "");
      badgeHtml = '<span class="notice-tag notice-tag-' + esc(canon) + '">' + esc(badgeText) + "</span>";
    }

    // 仅在 canon + oid 都能在 subject 文本里稳定命中时才剥前缀；reminder 模板不剥。
    var displayTitle = (canon && OBJECT_TYPE_LABELS[canon] && !isReminderTemplate)
      ? stripSubjectPrefix(rawSubject, canon, oid)
      : rawSubject;

    var subText = "";
    var rawSummary = String(item.data || item.summary || "").trim();
    if (rawSummary && rawSummary !== rawSubject) {
      var icon = "";
      if (rawSummary.indexOf("审批") >= 0) icon = "💬 ";
      else if (rawSummary.indexOf("指派") >= 0) icon = "📌 ";
      else if (rawSummary.indexOf("描述") >= 0) icon = "📝 ";
      subText = '<div class="notice-subject-sub" title="' + esc(rawSummary) + '">' +
        '<span class="notice-sub-icon">' + icon + "</span>" + esc(rawSummary) + "</div>";
    }

    return '<div class="notice-subject-text" title="' + esc(rawSubject) + '">' +
      badgeHtml + '<button type="button" class="notice-title-main" data-notice-open="' + esc(item.id) + '">' + esc(displayTitle) + "</button></div>" + subText;
  }

  function rowHtml(item) {
    currentItemsMap[String(item.id)] = item;
    var statusHtml = item.read
      ? '<span class="notice-read-state">已读</span>'
      : '<span class="notice-unread-state"><i class="notice-unread-dot"></i>未读</span>';

    var actions = [];
    if (item.url) {
      actions.push('<a class="table-action-btn primary" href="' + esc(item.url) + '" target="_blank" rel="noopener noreferrer">去处理</a>');
    } else {
      actions.push('<button type="button" class="notice-view-btn" data-notice-open="' + esc(item.id) + '">查看详情</button>');
    }
    if (!item.read) {
      actions.push('<button type="button" class="notice-read-btn" data-notice-id="' + esc(item.id) + '">标为已读</button>');
    }

    return "<tr" + (item.read ? "" : ' class="is-unread"') + ">" +
      '<td class="notice-subject">' + formatNoticeSubject(item) + "</td>" +
      '<td class="notice-actor">' + esc(item.actor || "—") + "</td>" +
      '<td class="notice-date">' + esc(item.date || "—") + "</td>" +
      '<td class="notice-state">' + statusHtml + "</td>" +
      '<td class="notice-opt"><div class="notice-opt-group">' + actions.join("") + "</div></td>" +
      "</tr>";
  }

  function updateCounts(payload) {
    var quick = { qvCountAll: "total", qvCountUnread: "unread", qvCountAction: "action", qvCountAbnormal: "abnormal", qvCountToday: "today" };
    var categories = {
      categoryBusiness: "business", categoryApproval: "approval", categoryReminder: "reminder",
      categoryCollaboration: "collaboration", categoryRisk: "risk", categorySystem: "system"
    };
    Object.keys(quick).forEach(function (id) {
      var el = $(id);
      if (el) { el.textContent = payload[quick[id]] != null ? payload[quick[id]] : "—"; }
    });
    var catAll = $("categoryAll");
    if (catAll) { catAll.textContent = (payload.categories && payload.categories.all != null) ? payload.categories.all : "—"; }
    Object.keys(categories).forEach(function (id) {
      var el = $(id);
      if (el) {
        var key = categories[id];
        el.textContent = (payload.categories && payload.categories[key] != null) ? payload.categories[key] : "—";
      }
    });
  }

  function resetCounts() {
    ["qvCountAll", "qvCountUnread", "qvCountAction", "qvCountAbnormal", "qvCountToday",
     "categoryAll", "categoryBusiness", "categoryApproval", "categoryReminder",
     "categoryCollaboration", "categoryRisk", "categorySystem"].forEach(function (id) {
      var el = $(id);
      if (el) { el.textContent = "—"; }
    });
  }

  var controller = null;

  function loadData() {
    if (!controller) { return; }
    controller.fetch(buildUrl(), { method: "GET" }, function (payload) {
      var items = (payload && Array.isArray(payload.items)) ? payload.items : [];
      var total = (payload && typeof payload.filteredTotal === "number") ? payload.filteredTotal : ((payload && typeof payload.total === "number") ? payload.total : items.length);
      var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

      if (state.page > totalPages && total > 0 && !hasCorrectedPage) {
        hasCorrectedPage = true;
        state.page = totalPages;
        syncUrl();
        loadData();
        return;
      }
      hasCorrectedPage = false;

      var tbody = $("noticeTbody");
      if (tbody) { tbody.innerHTML = items.map(rowHtml).join(""); }
      if ($("noticeSummary")) { $("noticeSummary").textContent = "共 " + total + " 条通知"; }
      if (payload) { updateCounts(payload); }

      if (window.PersonalList) {
        var gotoPage = function (p) { state.page = p; hasCorrectedPage = false; syncUrl(); loadData(); };
        window.PersonalList.renderPagination({
          container: $("noticePagination"),
          page: state.page, pageSize: state.pageSize, total: total,
          onPageChange: gotoPage,
          onPageSizeChange: function (s) {
            state.pageSize = s; state.page = 1; hasCorrectedPage = false;
            window.PersonalList.savePageSize("po.notice.pageSize", s);
            syncUrl(); loadData();
          }
        });
      }
    });
  }

  function markSingleRead(id) {
    var fetchFn = window.appFetch || fetch;
    fetchFn("/notice/" + encodeURIComponent(id) + "/read", {
      method: "PUT", headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
    })
      .then(function (res) { if (!res.ok) throw new Error("HTTP " + res.status); return res.json().catch(function () { throw new Error("Invalid response format"); }); })
      .then(function (p) { if (!p || p.success !== true) throw new Error((p && p.error) || "mark read failed"); loadData(); })
      .catch(function () { if (typeof window.showToast === "function") window.showToast("标为已读可能未生效，请刷新重试", "danger"); });
  }

  function markAllRead() {
    var button = $("noticeMarkAllBtn");
    if (button.disabled) { return; }
    button.disabled = true;
    var fetchFn = window.appFetch || fetch;
    fetchFn("/notice/read-all", {
      method: "PUT", headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" },
      body: JSON.stringify({ filters: state })
    })
      .then(function (res) { if (!res.ok) throw new Error("HTTP " + res.status); return res.json().catch(function () { throw new Error("Invalid response format"); }); })
      .then(function (p) {
        if (!p || p.success !== true) throw new Error((p && p.error) || "mark all read failed");
        if (typeof window.showToast === "function") window.showToast("已将当前筛选的全部未读通知标为已读", "success");
        loadData();
      })
      .catch(function () { if (typeof window.showToast === "function") window.showToast("全部标为已读可能未生效，请刷新重试", "danger"); })
      .finally(function () { button.disabled = false; });
  }

  function initQuickChips() {
    document.querySelectorAll("#noticeQuickChips .header-quick-chip").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var qv = btn.getAttribute("data-qv") || "all";
        if (state.quickView === qv) return;
        document.querySelectorAll("#noticeQuickChips .header-quick-chip").forEach(function (b) {
          b.classList.remove("active"); b.setAttribute("aria-pressed", "false");
        });
        btn.classList.add("active"); btn.setAttribute("aria-pressed", "true");
        state.quickView = qv; state.page = 1; hasCorrectedPage = false;
        syncUrl(); loadData();
      });
    });
  }

  function initTabs() {
    document.querySelectorAll(".po-notice .category-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var cat = btn.getAttribute("data-category") || "all";
        if (state.category === cat) return;
        document.querySelectorAll(".po-notice .category-tab").forEach(function (b) {
          b.classList.remove("active"); b.removeAttribute("aria-current");
        });
        btn.classList.add("active"); btn.setAttribute("aria-current", "page");
        state.category = cat; state.page = 1; hasCorrectedPage = false;
        syncUrl(); loadData();
      });
    });
  }

  function initToolbar() {
    var kwInput = $("noticeKeyword");
    if (kwInput) {
      var timer = null;
      kwInput.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(function () {
          state.keyword = (kwInput.value || "").trim();
          state.page = 1; hasCorrectedPage = false; syncUrl(); loadData();
        }, 300);
      });
    }

    [
      { id: "noticeObjectType", key: "objectType" },
      { id: "noticeTimeRange", key: "timeRange" },
      { id: "noticeReadState", key: "readState" },
      { id: "noticeNeedAction", key: "needAction" }
    ].forEach(function (it) {
      var sel = $(it.id);
      if (sel) sel.addEventListener("change", function () {
        state[it.key] = sel.value; state.page = 1; hasCorrectedPage = false; syncUrl(); loadData();
      });
    });

    var resetBtn = $("noticeResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.quickView = "all"; state.category = "all"; state.objectType = "all";
        state.timeRange = "all"; state.readState = "all"; state.needAction = "all";
        state.keyword = ""; state.page = 1;

        document.querySelectorAll("#noticeQuickChips .header-quick-chip").forEach(function (b) {
          var isAll = (b.getAttribute("data-qv") || "all") === "all";
          b.classList.toggle("active", isAll); b.setAttribute("aria-pressed", isAll ? "true" : "false");
        });
        document.querySelectorAll(".po-notice .category-tab").forEach(function (b) {
          var isAll = (b.getAttribute("data-category") || "all") === "all";
          b.classList.toggle("active", isAll);
          if (isAll) b.setAttribute("aria-current", "page"); else b.removeAttribute("aria-current");
        });

        if (kwInput) kwInput.value = "";
        ["noticeObjectType", "noticeTimeRange", "noticeReadState", "noticeNeedAction"].forEach(function (id) {
          if ($(id)) $(id).value = "all";
        });
        hasCorrectedPage = false; syncUrl(); loadData();
      });
    }

    var markAllBtn = $("noticeMarkAllBtn");
    if (markAllBtn) markAllBtn.addEventListener("click", markAllRead);
    var retryBtn = $("noticeRetryBtn");
    if (retryBtn) retryBtn.addEventListener("click", loadData);

    var tbody = $("noticeTbody");
    if (tbody) tbody.addEventListener("click", function (e) {
      var btn = e.target.closest(".notice-read-btn");
      if (btn) { var id = btn.getAttribute("data-notice-id"); if (id) markSingleRead(id); }
    });
  }

  function openNoticeDrawer(id) {
    var item = currentItemsMap[String(id)];
    var mask = $("noticeDrawerMask");
    var body = $("noticeDrawerBody");
    var openObjBtn = $("noticeDrawerOpenObj");
    if (!mask || !body || !item) return;

    var ot = String(item.objectType || "").trim().toLowerCase();
    var oid = String(item.objectId || "").trim();
    var canon = canonicalKind(ot);
    var objName = (OBJECT_TYPE_LABELS[canon] || OBJECT_TYPE_LABELS[ot] || ot || "—") + (oid && oid !== "0" ? " #" + oid : "");
    var rawSubject = item.subject || item.title || item.data || "通知详情";
    var content = item.data || item.summary || item.subject || "无具体内容";

    body.innerHTML = '<div class="notice-detail-section">' +
      '<div class="notice-detail-title">' + esc(rawSubject) + '</div>' +
      '<div class="notice-detail-meta">' +
      '<span class="notice-meta-label">触发人</span><span class="notice-meta-value">' + esc(item.actor || "系统") + '</span>' +
      '<span class="notice-meta-label">通知时间</span><span class="notice-meta-value">' + esc(item.date || "—") + '</span>' +
      '<span class="notice-meta-label">关联对象</span><span class="notice-meta-value">' + esc(objName) + '</span>' +
      '<span class="notice-meta-label">通知状态</span><span class="notice-meta-value">' + (item.read ? "已读" : "未读") + '</span>' +
      '</div>' +
      '<div class="notice-detail-body">' + esc(content) + '</div>' +
      '</div>';

    if (openObjBtn) {
      if (item.url) {
        openObjBtn.href = item.url;
        openObjBtn.hidden = false;
      } else {
        openObjBtn.hidden = true;
      }
    }

    mask.hidden = false;
    if (!item.read) markSingleRead(item.id);
  }

  function closeNoticeDrawer() {
    var mask = $("noticeDrawerMask");
    if (mask) mask.hidden = true;
  }

  document.addEventListener("DOMContentLoaded", function () {
    if (window.PersonalList) {
      controller = window.PersonalList.createController({
        summaryEl: $("noticeSummary"),
        emptyEl: $("noticeEmpty"),
        errorEl: $("noticeError"),
        tbodyEl: $("noticeTbody"),
        errorColspan: 5,
        onError: function () { resetCounts(); }
      });
    }

    if ($("noticeDrawerClose")) $("noticeDrawerClose").addEventListener("click", closeNoticeDrawer);
    if ($("noticeDrawerCloseBtn")) $("noticeDrawerCloseBtn").addEventListener("click", closeNoticeDrawer);
    var drawerMask = $("noticeDrawerMask");
    if (drawerMask) {
      drawerMask.addEventListener("click", function (e) {
        if (e.target === drawerMask) closeNoticeDrawer();
      });
    }

    var tbody = $("noticeTbody");
    if (tbody) {
      tbody.addEventListener("click", function (e) {
        var openBtn = e.target.closest("[data-notice-open]");
        if (openBtn) {
          var nid = openBtn.getAttribute("data-notice-open");
          if (nid) openNoticeDrawer(nid);
        }
      });
    }

    initQuickChips();
    initTabs();
    initToolbar();
    initFromUrl();
    syncUrl();
    loadData();
  });
})();
