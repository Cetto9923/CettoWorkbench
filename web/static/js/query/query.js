/* =============================================================================
   文件: web/static/js/query/query.js
   模块: PO 工作台 - 需求查询交互脚本
   职责: 业务需求 / 研发需求 双标签、SQL filter 透传、PersonalList 接管时序保护
         与三态（加载 / 空 / 错误）；操作列支持 DemandDetail.open 与禅道外链。
   依赖: personal-list.js
   ============================================================================= */

(function () {
  "use strict";

  var PL = window.PersonalList || {};
  var PAGE_SIZE_OPTIONS = PL.PAGE_SIZE_OPTIONS || [10, 20, 50, 100];
  var esc = PL.escapeHtml || function (v) { return String(v == null ? "" : v); };
  var priorityBadge = PL.priorityBadge || function (raw) {
    var n = parseInt(String(raw || "").replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };
  var objectTypeBadge = PL.objectTypeBadge || function (kind) {
    var labels = { business: "业务需求", story: "研发需求" };
    var k = String(kind || "").trim().toLowerCase();
    var label = labels[k] || k || "—";
    return '<span class="wb-type wb-type-' + (labels[k] ? k : "unknown") + '">' + label + "</span>";
  };

  var TAB_KIND_MAP = { biz: "business", rd: "story" };

  var ZENTAO_STATUS_LABELS = {
    draft: "暂存", wait: "待评审", active: "已评审", clarified: "已澄清",
    changed: "已变更", developing: "开发中", testing: "测试中", waitacceptance: "待验收",
    acceptanced: "已验收", waitdeliver: "待交付", delivered: "已交付", released: "已发布",
    closed: "已关闭", suspended: "已挂起", refuse: "已驳回"
  };

  var STORY_STATUS_LABELS = {
    draft: "草稿", reviewing: "评审中", active: "激活", changing: "变更中", closed: "已关闭"
  };

  function deriveStatusLabel(row, kind) {
    var raw = String((row && row.status) || "").trim().toLowerCase();
    if (!raw) { return "—"; }
    var labels = (kind === "rd" || kind === "story") ? STORY_STATUS_LABELS : ZENTAO_STATUS_LABELS;
    return labels[raw] || raw;
  }

  var state = {
    tab: "biz",
    keyword: "",
    status: "",
    priority: "",
    owner: "",
    system: "",
    page: 1,
    pageSize: 15
  };

  var controller = null;

  function $(id) { return document.getElementById(id); }

  function buildQuery() {
    var q = new URLSearchParams();
    q.set("tab", state.tab);
    if (state.keyword) { q.set("keyword", state.keyword); }
    if (state.status) { q.set("status", state.status); }
    if (state.priority) { q.set("priority", state.priority); }
    if (state.owner) { q.set("owner", state.owner); }
    if (state.system) { q.set("system", state.system); }
    q.set("page", String(state.page || 1));
    q.set("pageSize", String(state.pageSize || 15));
    return q;
  }

  function renderRow(row, kind) {
    var isStory = (kind === "rd");
    var rawId = row && row.id;
    var displayCode = isStory ? String(rawId) : ("US" + rawId);
    var zentaoUrl = esc(row.url || "");

    // ID 列：业务需求 US{id} 链 /demands/{id}（当前页），研发需求纯数字 + target=_blank 跳禅道。
    var idHtml;
    if (isStory) {
      if (zentaoUrl) {
        idHtml = '<a class="table-id-link" href="' + zentaoUrl + '" target="_blank" rel="noopener noreferrer">' + esc(displayCode) + '</a>';
      } else {
        idHtml = '<span class="table-id-link">' + esc(displayCode) + '</span>';
      }
    } else {
      var workbenchHref = rawId ? "/demands/" + encodeURIComponent(String(rawId)) : "";
      idHtml = workbenchHref
        ? '<a class="table-id-link" href="' + esc(workbenchHref) + '">' + esc(displayCode) + '</a>'
        : '<span class="table-id-link">' + esc(displayCode) + '</span>';
    }

    // 标题列：业务需求可点击触发 DemandDetail.open 抽屉；研发需求走禅道详情。
    var titleLink;
    if (isStory) {
      titleLink = zentaoUrl
        ? '<a class="table-title-link" href="' + zentaoUrl + '" target="_blank" rel="noopener noreferrer">' + esc(row.title || "—") + '</a>'
        : '<span class="table-title-link">' + esc(row.title || "—") + '</span>';
    } else {
      titleLink = '<a class="table-title-link" data-demand-id="' + esc(rawId) + '" href="javascript:void(0)">' + esc(row.title || "—") + '</a>';
    }

    var typeTag = objectTypeBadge(TAB_KIND_MAP[kind] || "");
    var ownerText = esc(row.owner || "—");
    var systemText = esc(row.system || "—");
    var deadline = esc(row.deadline || "—");
    var stageHtml = '<span class="stage-tag">' + esc(row.stage || "—") + '</span>';
    var statusText = deriveStatusLabel(row, kind);
    var statusHtml = (PL && PL.statusTagHtml) ? PL.statusTagHtml(statusText) : ('<span class="status-tag">' + esc(statusText) + '</span>');

    // 操作列：业务需求按钮触发 DemandDetail.open 抽屉，研发需求走禅道详情新窗口。
    var actionHtml;
    if (isStory) {
      actionHtml = zentaoUrl
        ? '<a class="action-btn table-action-btn secondary" href="' + zentaoUrl + '" target="_blank" rel="noopener noreferrer">详情</a>'
        : '<span class="action-btn table-action-btn secondary action-btn-disabled">无外链</span>';
    } else {
      actionHtml = '<button type="button" class="action-btn table-action-btn secondary" data-demand-id="' + esc(rawId) + '">查看</button>';
    }

    return '<tr>' +
      '<td class="query-col-id">' + idHtml + '</td>' +
      '<td class="query-col-title" title="' + esc(row.title || "") + '">' + titleLink + '</td>' +
      '<td class="query-col-type">' + typeTag + '</td>' +
      '<td class="query-col-pri">' + priorityBadge(row.priority) + '</td>' +
      '<td class="query-col-stage">' + stageHtml + '</td>' +
      '<td class="query-col-status">' + statusHtml + '</td>' +
      '<td class="query-col-owner">' + ownerText + '</td>' +
      '<td class="query-col-system">' + systemText + '</td>' +
      '<td class="query-col-dead">' + deadline + '</td>' +
      '<td class="query-col-action">' + actionHtml + '</td>' +
      '</tr>';
  }

  function attachRowEvents() {
    var tbody = $("queryTbody");
    if (!tbody || tbody.__queryBound) { return; }
    tbody.__queryBound = true;
    tbody.addEventListener("click", function (event) {
      // 操作列按钮（业务需求）
      var btn = event.target.closest("button.action-btn[data-demand-id]");
      if (btn) {
        var demandId = btn.getAttribute("data-demand-id");
        if (window.DemandDetail && typeof window.DemandDetail.open === "function") {
          window.DemandDetail.open(demandId);
        }
        event.preventDefault();
        return;
      }
      // 标题列链接（业务需求，链接占位为 javascript:void(0)）
      var titleA = event.target.closest("a.table-title-link[data-demand-id]");
      if (titleA) {
        var id = titleA.getAttribute("data-demand-id");
        if (window.DemandDetail && typeof window.DemandDetail.open === "function") {
          window.DemandDetail.open(id);
        }
        event.preventDefault();
      }
    });
  }

  function renderItems(payload) {
    var kind = (payload && payload.kind) || state.tab;
    var rows = (payload && (payload.rows || payload.items)) || [];
    var total = (payload && typeof payload.total === "number") ? payload.total : rows.length;
    var tbody = $("queryTbody");
    if (!tbody) { return; }

    if (total === 0 || rows.length === 0) {
      tbody.innerHTML = '<tr><td colspan="10" class="state-placeholder">当前条件下暂无需求</td></tr>';
      var pager = $("queryPager");
      if (pager) { pager.hidden = true; pager.innerHTML = ""; }
      return;
    }
    tbody.innerHTML = rows.map(function (row) { return renderRow(row, kind); }).join("");
    PL.renderPagination({
      container: $("queryPager"),
      page: state.page,
      pageSize: state.pageSize,
      total: total,
      onPageChange: function (p) { state.page = p; load(); },
      onPageSizeChange: function (s) {
        state.pageSize = s;
        state.page = 1;
        PL.savePageSize("po.query.pageSize", s);
        load();
      }
    });
  }

  function load() {
    var url = "/query/items?" + buildQuery().toString();
    if (!controller) {
      controller = PL.createController({
        summaryEl: $("querySummary"),
        emptyEl: $("queryEmpty"),
        errorEl: $("queryError"),
        tbodyEl: $("queryTbody"),
        onSuccess: function (payload) { renderItems(payload); },
        onError: function () { /* 错误文案由 createController 写入 #queryError */ }
      });
    }
    controller.fetch(url, state, function () {});
  }

  function initFromDom() {
    var sp = new URLSearchParams(window.location.search || "");
    var tab = sp.get("tab");
    if (tab === "biz" || tab === "rd") {
      state.tab = tab;
      document.querySelectorAll(".po-query .category-tab").forEach(function (t) {
        var isCurrent = t.getAttribute("data-tab") === tab;
        t.classList.toggle("active", isCurrent);
        if (isCurrent) { t.setAttribute("aria-current", "page"); } else { t.removeAttribute("aria-current"); }
      });
    }
    state.keyword = (sp.get("keyword") || "").trim();
    state.status = (sp.get("status") || "").trim();
    state.priority = (sp.get("priority") || "").trim();
    state.owner = (sp.get("owner") || "").trim();
    state.system = (sp.get("system") || "").trim();
    var p = parseInt(sp.get("page"), 10);
    if (!isNaN(p) && p >= 1) { state.page = p; }
    var ps = parseInt(sp.get("pageSize"), 10);
    if (!isNaN(ps) && PAGE_SIZE_OPTIONS.indexOf(ps) >= 0) {
      state.pageSize = ps;
    } else {
      state.pageSize = PL.loadPageSize("po.query.pageSize", state.pageSize, PAGE_SIZE_OPTIONS);
    }
    if ($("queryKeyword")) { $("queryKeyword").value = state.keyword; }
    if ($("queryStatus")) { $("queryStatus").value = state.status; }
    if ($("queryPriority")) { $("queryPriority").value = state.priority; }
    if ($("queryOwner")) { $("queryOwner").value = state.owner; }
    if ($("querySystem")) { $("querySystem").value = state.system; }
  }

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var p = new URLSearchParams();
    if (state.tab !== "biz") { p.set("tab", state.tab); }
    if (state.keyword) { p.set("keyword", state.keyword); }
    if (state.status) { p.set("status", state.status); }
    if (state.priority) { p.set("priority", state.priority); }
    if (state.owner) { p.set("owner", state.owner); }
    if (state.system) { p.set("system", state.system); }
    if (state.page > 1) { p.set("page", String(state.page)); }
    if (state.pageSize !== 15) { p.set("pageSize", String(state.pageSize)); }
    var qs = p.toString();
    var newUrl = window.location.pathname + (qs ? "?" + qs : "");
    window.history.replaceState(null, "", newUrl);
  }

  function bindTabs() {
    document.querySelectorAll(".po-query .category-tab").forEach(function (tab) {
      tab.addEventListener("click", function () {
        var next = tab.getAttribute("data-tab") || "biz";
        if (next === state.tab) { return; }
        state.tab = next;
        state.page = 1;
        document.querySelectorAll(".po-query .category-tab").forEach(function (t) {
          var isCurrent = t === tab;
          t.classList.toggle("active", isCurrent);
          if (isCurrent) { t.setAttribute("aria-current", "page"); } else { t.removeAttribute("aria-current"); }
        });
        syncUrl();
        load();
      });
    });
  }

  function bindToolbar() {
    var searchTimer = null;
    function debouncedLoad() {
      clearTimeout(searchTimer);
      searchTimer = setTimeout(function () {
        state.page = 1;
        syncUrl();
        load();
      }, 250);
    }
    if ($("queryKeyword")) {
      $("queryKeyword").addEventListener("input", function () {
        state.keyword = this.value.trim();
        debouncedLoad();
      });
    }
    if ($("queryStatus")) {
      $("queryStatus").addEventListener("change", function () {
        state.status = this.value;
        state.page = 1;
        syncUrl();
        load();
      });
    }
    if ($("queryPriority")) {
      $("queryPriority").addEventListener("change", function () {
        state.priority = this.value;
        state.page = 1;
        syncUrl();
        load();
      });
    }
    if ($("queryOwner")) {
      $("queryOwner").addEventListener("input", function () {
        state.owner = this.value.trim();
        debouncedLoad();
      });
    }
    if ($("querySystem")) {
      $("querySystem").addEventListener("input", function () {
        state.system = this.value.trim();
        debouncedLoad();
      });
    }
    if ($("queryResetBtn")) {
      $("queryResetBtn").addEventListener("click", function () {
        state.keyword = "";
        state.status = "";
        state.priority = "";
        state.owner = "";
        state.system = "";
        state.page = 1;
        if ($("queryKeyword")) { $("queryKeyword").value = ""; }
        if ($("queryStatus")) { $("queryStatus").value = ""; }
        if ($("queryPriority")) { $("queryPriority").value = ""; }
        if ($("queryOwner")) { $("queryOwner").value = ""; }
        if ($("querySystem")) { $("querySystem").value = ""; }
        syncUrl();
        load();
      });
    }
    if ($("queryRetryBtn")) {
      $("queryRetryBtn").addEventListener("click", function () { load(); });
    }
  }

  function init() {
    initFromDom();
    bindTabs();
    bindToolbar();
    attachRowEvents();
    syncUrl();
    load();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
