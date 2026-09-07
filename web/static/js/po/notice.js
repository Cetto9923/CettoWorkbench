/* =============================================================================
   文件: web/static/js/po/notice.js
   模块: PO 个人工作台 - 通知中心交互脚本
   职责: 绑定通知分类、快捷焦点、筛选工具栏、列表渲染与已读状态管理
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var $ = function (id) { return document.getElementById(id); };

  var state = {
    quickView: "all",
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
  var VALID_OBJECT_TYPES = ["all", "demand", "story", "project", "task", "bug", "testtask", "issue", "risk", "approval"];
  var VALID_TIME_RANGES = ["all", "today", "7d", "week", "30d", "month"];
  var VALID_READ_STATES = ["all", "unread", "read"];
  var VALID_ACTION_STATES = ["all", "yes", "no"];
  var hasCorrectedPage = false;

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var p = new URLSearchParams();
    ["quickView", "category", "objectType", "timeRange", "readState", "needAction"].forEach(function (k) {
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

  var OBJECT_TYPE_LABELS = {
    demand: "需求", story: "研需", project: "项目", task: "任务",
    bug: "Bug", testtask: "测试", issue: "问题", risk: "风险", approval: "审批"
  };

  var categoryLabels = {
    all: "全部分类", business: "业务动态", approval: "审批流程", reminder: "时效提醒",
    collaboration: "协作消息", risk: "风险异常", system: "系统消息"
  };

  function buildUrl() {
    var params = new URLSearchParams();
    Object.keys(state).forEach(function (key) {
      if (state[key] !== "") { params.set(key, String(state[key])); }
    });
    return "/notice/items?" + params.toString();
  }

  function formatObjectCell(item) {
    var ot = String(item.objectType || "").trim().toLowerCase();
    var oid = String(item.objectId || "").trim();

    if (ot === "mail") { return "邮件通知"; }
    if (!ot || ot === "—" || !oid || oid === "—" || oid === "0") {
      return '<span class="notice-missing-rel">关联信息缺失</span>';
    }

    var label = (OBJECT_TYPE_LABELS[ot] || ot) + " " + oid;
    if (item.url) {
      return '<a class="table-id-link" href="' + esc(item.url) + '" rel="noopener noreferrer">' + esc(label) + "</a>";
    }
    return esc(label);
  }

  function rowHtml(item) {
    var category = item.category || "business";
    var subject = esc(item.subject || item.data || "—");
    var subText = "";
    if (item.data && item.data !== item.subject && item.data.length > 0) {
      subText = '<div class="notice-subject-sub" title="' + esc(item.data) + '">' + esc(item.data) + "</div>";
    }

    var objectCell = formatObjectCell(item);
    var statusHtml = item.read
      ? '<span class="notice-read-state">已读</span>'
      : '<span class="notice-unread-state"><i class="notice-unread-dot"></i>未读</span>';

    var mainAction = "";
    if (item.needAction && item.url) {
      mainAction = '<a class="table-action-btn primary" href="' + esc(item.url) + '" rel="noopener noreferrer">去处理</a>';
    } else if (item.url) {
      mainAction = '<a class="table-action-btn" href="' + esc(item.url) + '" rel="noopener noreferrer">查看详情</a>';
    } else {
      mainAction = '<span class="notice-no-op">—</span>';
    }

    var readBtn = item.read
      ? ""
      : '<button type="button" class="notice-read-btn" data-notice-id="' + esc(item.id) + '">标为已读</button>';

    return "<tr" + (item.read ? "" : ' class="is-unread"') + ">" +
      '<td class="notice-subject">' +
        '<div class="notice-subject-text" title="' + subject + '">' + subject + "</div>" + subText +
      "</td>" +
      '<td class="notice-object">' + objectCell + "</td>" +
      '<td class="notice-actor">' + esc(item.actor || "—") + "</td>" +
      '<td class="notice-date">' + esc(item.date || "—") + "</td>" +
      '<td class="notice-state">' + statusHtml + "</td>" +
      '<td class="notice-opt">' +
        '<div class="notice-opt-group">' + mainAction + readBtn + "</div>" +
      "</td>" +
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
    if (catAll) { catAll.textContent = payload.total != null ? payload.total : "—"; }
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
        window.PersonalList.renderPagination({
          container: $("noticePagination"),
          page: state.page,
          pageSize: state.pageSize,
          total: total,
          onPageChange: function (p) {
            state.page = p;
            hasCorrectedPage = false;
            syncUrl();
            loadData();
          },
          onPageSizeChange: function (s) {
            state.pageSize = s;
            state.page = 1;
            hasCorrectedPage = false;
            window.PersonalList.savePageSize("po.notice.pageSize", s);
            syncUrl();
            loadData();
          }
        });
      }
    });
  }

  function markSingleRead(id) {
    var fetchFn = window.appFetch || fetch;
    fetchFn("/notice/" + encodeURIComponent(id) + "/read", {
      method: "PUT",
      headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
    })
      .then(function (res) {
        if (!res.ok) { throw new Error("HTTP " + res.status); }
        return res.json().catch(function () { throw new Error("Invalid response format"); });
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error((payload && payload.error) || "mark read failed");
        }
        loadData();
      })
      .catch(function () {
        if (typeof window.showToast === "function") {
          window.showToast("标为已读可能未生效，请刷新重试", "danger");
        }
      });
  }

  function markAllRead() {
    var fetchFn = window.appFetch || fetch;
    fetchFn("/notice/read-all", {
      method: "PUT",
      headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
    })
      .then(function (res) {
        if (!res.ok) { throw new Error("HTTP " + res.status); }
        return res.json().catch(function () { throw new Error("Invalid response format"); });
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error((payload && payload.error) || "mark all read failed");
        }
        if (typeof window.showToast === "function") {
          window.showToast("已将全部通知标为已读", "success");
        }
        loadData();
      })
      .catch(function () {
        if (typeof window.showToast === "function") {
          window.showToast("全部标为已读可能未生效，请刷新重试", "danger");
        }
      });
  }

  function initQuickChips() {
    var chips = document.querySelectorAll("#noticeQuickChips .header-quick-chip");
    chips.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var qv = btn.getAttribute("data-qv") || "all";
        if (state.quickView === qv) { return; }
        chips.forEach(function (b) {
          b.classList.remove("active");
          b.setAttribute("aria-pressed", "false");
        });
        btn.classList.add("active");
        btn.setAttribute("aria-pressed", "true");
        state.quickView = qv;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    });
  }

  function initTabs() {
    var tabs = document.querySelectorAll(".po-notice .category-tab");
    tabs.forEach(function (btn) {
      btn.addEventListener("click", function () {
        var cat = btn.getAttribute("data-category") || "all";
        if (state.category === cat) { return; }
        tabs.forEach(function (b) {
          b.classList.remove("active");
          b.removeAttribute("aria-current");
        });
        btn.classList.add("active");
        btn.setAttribute("aria-current", "page");
        state.category = cat;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
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
          state.page = 1;
          hasCorrectedPage = false;
          syncUrl();
          loadData();
        }, 300);
      });
    }

    var objSel = $("noticeObjectType");
    if (objSel) {
      objSel.addEventListener("change", function () {
        state.objectType = objSel.value;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    }

    var timeSel = $("noticeTimeRange");
    if (timeSel) {
      timeSel.addEventListener("change", function () {
        state.timeRange = timeSel.value;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    }

    var readSel = $("noticeReadState");
    if (readSel) {
      readSel.addEventListener("change", function () {
        state.readState = readSel.value;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    }

    var actionSel = $("noticeNeedAction");
    if (actionSel) {
      actionSel.addEventListener("change", function () {
        state.needAction = actionSel.value;
        state.page = 1;
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    }

    var resetBtn = $("noticeResetBtn");
    if (resetBtn) {
      resetBtn.addEventListener("click", function () {
        state.quickView = "all";
        state.category = "all";
        state.objectType = "all";
        state.timeRange = "all";
        state.readState = "all";
        state.needAction = "all";
        state.keyword = "";
        state.page = 1;

        document.querySelectorAll("#noticeQuickChips .header-quick-chip").forEach(function (b) {
          var isAll = (b.getAttribute("data-qv") || "all") === "all";
          b.classList.toggle("active", isAll);
          b.setAttribute("aria-pressed", isAll ? "true" : "false");
        });
        document.querySelectorAll(".po-notice .category-tab").forEach(function (b) {
          var isAll = (b.getAttribute("data-category") || "all") === "all";
          b.classList.toggle("active", isAll);
          if (isAll) {
            b.setAttribute("aria-current", "page");
          } else {
            b.removeAttribute("aria-current");
          }
        });

        if (kwInput) { kwInput.value = ""; }
        if (objSel) { objSel.value = "all"; }
        if (timeSel) { timeSel.value = "all"; }
        if (readSel) { readSel.value = "all"; }
        if (actionSel) { actionSel.value = "all"; }
        hasCorrectedPage = false;
        syncUrl();
        loadData();
      });
    }

    var markAllBtn = $("noticeMarkAllBtn");
    if (markAllBtn) {
      markAllBtn.addEventListener("click", function () { markAllRead(); });
    }

    var retryBtn = $("noticeRetryBtn");
    if (retryBtn) {
      retryBtn.addEventListener("click", function () { loadData(); });
    }

    var tbody = $("noticeTbody");
    if (tbody) {
      tbody.addEventListener("click", function (e) {
        var btn = e.target.closest(".notice-read-btn");
        if (btn) {
          var id = btn.getAttribute("data-notice-id");
          if (id) { markSingleRead(id); }
        }
      });
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    if (window.PersonalList) {
      controller = window.PersonalList.createController({
        summaryEl: $("noticeSummary"),
        emptyEl: $("noticeEmpty"),
        errorEl: $("noticeError"),
        tbodyEl: $("noticeTbody"),
        errorColspan: 6,
        onError: function () { resetCounts(); }
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
