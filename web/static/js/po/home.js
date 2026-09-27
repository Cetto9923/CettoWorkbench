/* =============================================================================
   文件: web/static/js/po/home.js
   模块: PO 个人工作台 - 首页交互脚本
   职责: 绑定需求价值流下钻、全站统一工具栏筛选、7列行动列表展示与分页保护
   依赖: personal-list.js (priorityBadge(), objectTypeBadge(), idChipHtml()), home-render.js, jQuery
   ============================================================================= */

(function ($) {
  "use strict";

  var PAGE_SIZE_OPTIONS = window.PersonalList.PAGE_SIZE_OPTIONS;

  var state = {
    status: "all",
    focus: "my_action",
    page: 1,
    pageSize: 20,
    keyword: "",
    relation: "all",
    objectType: "all",
    priority: "all"
  };

  var rawItems = [];
  var VALID_STATUSES = ["all", "accept", "clarify", "schedule", "developing", "testing", "waitacceptance", "acceptanced", "publish", "released"];
  var currentSeq = 0;
  var hasCorrectedPage = false;

  function demandsUrl(status, page, pageSize) {
    var p = new URLSearchParams();
    p.set("status", status || "all");
    p.set("focus", state.focus);
    p.set("page", String(page || 1));
    p.set("pageSize", String(pageSize || 20));
    // 透传工具栏筛选到服务端，由 SQL 过滤 + Count + 分页；前端不再二次过滤。
    if (state.keyword) { p.set("keyword", state.keyword); }
    if (state.objectType && state.objectType !== "all") { p.set("objectType", state.objectType); }
    if (state.priority && state.priority !== "all") { p.set("priority", state.priority); }
    if (state.relation && state.relation !== "all") { p.set("relation", state.relation); }
    return "/demands?" + p.toString();
  }

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var params = new URLSearchParams();
    if (state.focus && state.focus !== "all") { params.set("focus", state.focus); }
    if (state.status && state.status !== "all") { params.set("status", state.status); }
    if (state.page > 1) { params.set("page", String(state.page)); }
    if (state.pageSize && state.pageSize !== 15) { params.set("pageSize", String(state.pageSize)); }
    if (state.keyword) { params.set("keyword", state.keyword); }
    if (state.objectType && state.objectType !== "all") { params.set("objectType", state.objectType); }
    if (state.priority && state.priority !== "all") { params.set("priority", state.priority); }
    if (state.relation && state.relation !== "all") { params.set("relation", state.relation); }
    var qs = params.toString();
    var newUrl = window.location.pathname + (qs ? "?" + qs : "");
    window.history.replaceState(null, "", newUrl);
  }

  function initFromUrl() {
    var sp = new URLSearchParams(window.location.search || "");
    var focus = sp.get("focus");
    if (["all", "my_action", "today", "blocked", "overdue", "suspended"].indexOf(focus) >= 0) { state.focus = focus; }
    var st = (sp.get("status") || "").trim();
    if (VALID_STATUSES.indexOf(st) >= 0) {
      state.status = st;
    }
    var p = parseInt(sp.get("page"), 10);
    if (!isNaN(p) && p >= 1) {
      state.page = p;
    }
    var ps = parseInt(sp.get("pageSize"), 10);
    if (!isNaN(ps) && PAGE_SIZE_OPTIONS.indexOf(ps) >= 0) {
      state.pageSize = ps;
    } else {
      state.pageSize = window.PersonalList.loadPageSize("po.home.pageSize", state.pageSize, PAGE_SIZE_OPTIONS);
    }
    var kw = (sp.get("keyword") || "").trim();
    if (kw) { state.keyword = kw; $("#homeKeyword").val(kw); }
    var ot = (sp.get("objectType") || "").trim();
    if (ot && ot !== "all") {
      state.objectType = ot;
      $("#homeObjectTypeSegment button").removeClass("active");
      $('#homeObjectTypeSegment button[data-object-type="' + ot + '"]').addClass("active");
    }
    var pr = (sp.get("priority") || "").trim();
    if (pr && pr !== "all") {
      state.priority = pr;
      $("#homePrioritySegment button").removeClass("active");
      $('#homePrioritySegment button[data-priority="' + pr + '"]').addClass("active");
    }
    var rel = (sp.get("relation") || "").trim();
    if (rel && rel !== "all") {
      state.relation = rel;
      $("#homeRelationSegment button").removeClass("active");
      $('#homeRelationSegment button[data-relation="' + rel + '"]').addClass("active");
    }
  }

  function loadDemands(status, page, pageSize) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(demandsUrl(status, page, pageSize), { method: "GET" })
      .then(function (res) {
        if (!res.ok) { throw new Error("load demands failed (" + res.status + ")"); }
        return res.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error("invalid payload");
        }
        return {
          items: Array.isArray(payload.items) ? payload.items : [],
          total: typeof payload.total === "number" ? payload.total : 0,
          page: typeof payload.page === "number" ? payload.page : 1,
          pageSize: typeof payload.pageSize === "number" ? payload.pageSize : 15,
          stageSummary: Array.isArray(payload.stageSummary) ? payload.stageSummary : []
        };
      });
  }

  function renderValueStreamSummary(rows) {
    if (window.PoHomeRender) {
      window.PoHomeRender.renderValueStreamSummary(rows, state.focus);
    }
  }

  function refreshValueStreamSummary() {
    // 阶段汇总随正式列表请求返回。这里不再额外发起 pageSize=1 的探测请求，
    // 避免同一筛选条件重复执行 count/page 查询。
    return Promise.resolve();
  }

  function updateTitle(count, displayedCount, pageItemCount) {
    if (window.PoHomeRender) {
      window.PoHomeRender.updateTitle($, state, count, displayedCount, pageItemCount);
    }
  }

  function fillUpdateTime() {
    if (window.PoHomeRender) {
      window.PoHomeRender.fillUpdateTime($);
    }
  }

  function filterItems(items) {
    return window.PoHomeRender ? window.PoHomeRender.filterItems(items) : (items || []).slice();
  }

  function renderRow(item) {
    return window.PoHomeRender ? window.PoHomeRender.renderRow(item) : "";
  }

  function renderList(total) {
    var filtered = filterItems(rawItems);
    updateTitle(total, filtered.length, rawItems.length);
    $("#top5Error").attr("hidden", true);

    if (!filtered.length) {
      $("#top5Tbody").empty();
      $("#top5List").attr("hidden", true);
      $("#top5Empty").removeAttr("hidden");
      $("#homePagination").attr("hidden", true);
      return;
    }

    $("#top5Empty").attr("hidden", true);
    $("#top5List").removeAttr("hidden");
    $("#top5Tbody").html(filtered.map(renderRow).join(""));

    if (window.PersonalList) {
      window.PersonalList.renderPagination({
        container: document.getElementById("homePagination"),
        page: state.page,
        pageSize: state.pageSize,
        total: total,
        onPageChange: function (p) {
          state.page = p;
          syncUrl();
          refreshDemands(state.status);
        },
        onPageSizeChange: function (s) {
          state.pageSize = s;
          state.page = 1;
          window.PersonalList.savePageSize("po.home.pageSize", s);
          syncUrl();
          refreshDemands(state.status);
        }
      });
    }
  }

  function setActiveCard($card) {
    $(".home-vs-mini-card").removeClass("active").attr("aria-pressed", "false");
    $card.addClass("active").attr("aria-pressed", "true");
  }

  function refreshDemands(status) {
    currentSeq += 1;
    var reqSeq = currentSeq;
    if (status) { state.status = status; }

    $("#top5List").removeAttr("hidden");
    $("#top5Empty").attr("hidden", true);
    $("#top5Error").attr("hidden", true);
    $("#top5Tbody").html('<tr><td colspan="6" class="state-placeholder">正在加载行动列表…</td></tr>');

    $("#top5List").attr("aria-busy", "true");
    $("#homeListCaption").text("正在获取当前阶段数据…");

    return loadDemands(state.status, state.page, state.pageSize)
      .then(function (res) {
        if (reqSeq !== currentSeq) { return; }
        rawItems = res.items || [];
        var total = res.total;
        var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

        if (state.page > totalPages && total > 0 && !hasCorrectedPage) {
          hasCorrectedPage = true;
          state.page = totalPages;
          syncUrl();
          refreshDemands(state.status);
          return;
        }
        hasCorrectedPage = false;

        $("#top5List").attr("aria-busy", "false");
        renderList(total);
        // “全部”列表没有 stageSummary：显式恢复模板中的全量基线，避免保留上一次焦点查询的旧数字。
        renderValueStreamSummary(res.stageSummary, state.focus);
        fillUpdateTime();
      })
      .catch(function (err) {
        if (reqSeq !== currentSeq) { return; }
        hasCorrectedPage = false;
        $("#top5List").attr("aria-busy", "false");
        $("#lastUpdateTime").text("—");
        updateTitle(null);
        $("#top5Tbody").empty();
        $("#top5List").attr("hidden", true);
        $("#top5Empty").attr("hidden", true);
        $("#top5Error").removeAttr("hidden");
        $("#homePagination").attr("hidden", true);
        window.showToast("加载需求列表失败，请稍后重试", "danger");
      });
  }

  function initToolbar() {
    var searchTimer = null;
    $("#homeKeyword").on("input", function () {
      var val = $(this).val();
      clearTimeout(searchTimer);
      searchTimer = setTimeout(function () {
        state.keyword = val;
        state.page = 1;
        syncUrl();
        refreshDemands(state.status);
      }, 200);
    });

    $("#homeObjectTypeSegment").on("click", "button[data-object-type]", function () {
      var btn = $(this);
      var next = String(btn.attr("data-object-type") || "all");
      if (next === state.objectType) { return; }
      $("#homeObjectTypeSegment button").removeClass("active");
      btn.addClass("active");
      state.objectType = next;
      state.page = 1;
      syncUrl();
      refreshDemands(state.status);
    });

    $("#homePrioritySegment").on("click", "button[data-priority]", function () {
      var btn = $(this);
      var next = String(btn.attr("data-priority") || "all");
      if (next === state.priority) { return; }
      $("#homePrioritySegment button").removeClass("active");
      btn.addClass("active");
      state.priority = next;
      state.page = 1;
      syncUrl();
      refreshDemands(state.status);
    });

    $("#homeRelationSegment button").on("click", function () {
      $("#homeRelationSegment button").removeClass("active");
      $(this).addClass("active");
      state.relation = $(this).data("relation") || "all";
      state.page = 1;
      syncUrl();
      refreshDemands(state.status);
    });

    $("#homeResetBtn").on("click", function () {
      $("#homeKeyword").val("");
      $("#homeObjectTypeSegment button").removeClass("active").first().addClass("active");
      $("#homePrioritySegment button").removeClass("active").first().addClass("active");
      $("#homeRelationSegment button").removeClass("active").first().addClass("active");
      state.keyword = "";
      state.objectType = "all";
      state.priority = "all";
      state.relation = "all";
      state.page = 1;
      syncUrl();
      refreshDemands(state.status);
    });
  }

  function initValueStreamLinkage() {
    $(".home-vs-mini-card").on("click", function () {
      var $card = $(this);
      var status = $card.attr("data-vs-status");
      if (!status) { return; }
      setActiveCard($card);
      state.status = status;
      state.page = 1;
      syncUrl();
      refreshDemands(status);
    });

    // 右侧 PO 聚焦专区的小卡片触发联动切换价值流阶段
    $(".focus-card.vs-trigger").on("click", function () {
      var targetStage = $(this).data("stage-target");
      if (!targetStage) { return; }
      var $card = $('.home-vs-mini-card[data-vs-status="' + targetStage + '"]');
      if ($card.length) {
        setActiveCard($card);
        state.status = targetStage;
        state.page = 1;
        syncUrl();
        refreshDemands(targetStage);
      }
    });
  }

  var FOCUS_META = {
    my_action: { title: "待我处理事项", tag: "当前要办理", subtip: "统计我参与阶段的需求，优先推进需确认与流转的事项" },
    all: { title: "全量事项清单", tag: "全盘流转", subtip: "查看所有由您关联或参与的需求与研发事项" },
    today: { title: "今日必推清单", tag: "今日聚焦", subtip: "今天到期及急需推进的高优事项" },
    blocked: { title: "阻塞事项清单", tag: "风险拦截", subtip: "处于阻塞停滞状态、急需排查解阻的事项" },
    overdue: { title: "超期事项清单", tag: "超期预警", subtip: "已超出目标交付时间未完成的事项（含关联业务需求与研发需求）" },
    suspended: { title: "挂起事项清单", tag: "暂停流转", subtip: "已暂停流转或进入搁置状态的事项" }
  };

  function updateActionHeader(focus) {
    var meta = FOCUS_META[focus] || FOCUS_META.my_action;
    $("#homeActionTitle").text(meta.title);
    $("#homeActionTag").text(meta.tag);
    $("#homeActionSubtip").text(meta.subtip);
  }

  function initVsToggle() {
    var STORAGE_KEY = "workbench.homeVsCollapsed";
    var $btn = $("#homeVsToggleBtn");
    var $section = $("#homeValueStreamSection");
    if (!$btn.length || !$section.length) return;

    function setCollapsed(collapsed) {
      if (collapsed) {
        $section.addClass("is-collapsed");
        $btn.attr("aria-expanded", "false");
        $("#homeVsToggleText").text("展开流水线");
        $("#homeVsToggleIcon").removeClass("fa-chevron-up").addClass("fa-chevron-down");
      } else {
        $section.removeClass("is-collapsed");
        $btn.attr("aria-expanded", "true");
        $("#homeVsToggleText").text("收起流水线");
        $("#homeVsToggleIcon").removeClass("fa-chevron-down").addClass("fa-chevron-up");
      }
    }

    try {
      if (window.localStorage && window.localStorage.getItem(STORAGE_KEY) === "1") {
        setCollapsed(true);
      }
    } catch (e) {}

    $btn.on("click", function () {
      var next = !$section.hasClass("is-collapsed");
      setCollapsed(next);
      try {
        if (window.localStorage) {
          window.localStorage.setItem(STORAGE_KEY, next ? "1" : "0");
        }
      } catch (e) {}
    });
  }

  window.refreshPoHomeDemands = function () {
    refreshValueStreamSummary(); return refreshDemands(state.status);
  };

  $(function () {
    initFromUrl();
    initVsToggle();
    updateActionHeader(state.focus);
    $("#homeQuickChips [data-home-focus]").on("click", function (event) {
      event.preventDefault();
      state.focus = $(this).attr("data-home-focus") || "all";
      state.page = 1;
      hasCorrectedPage = false;
      $("#homeQuickChips [data-home-focus]").removeClass("active").attr("aria-pressed", "false");
      $(this).addClass("active").attr("aria-pressed", "true");
      updateActionHeader(state.focus);
      syncUrl();
      refreshValueStreamSummary();
      refreshDemands(state.status);
    });
    $("#homeQuickChips [data-home-focus]").removeClass("active").attr("aria-pressed", "false");
    $('#homeQuickChips [data-home-focus="' + state.focus + '"]').addClass("active").attr("aria-pressed", "true");

    // 首页本地工作视角切换（当前仅 PO 生效，其余提示规划中）
    $(".po-perspective-tabs").on("click", ".po-perspective-tab", function (e) {
      e.preventDefault();
      var p = $(this).attr("data-perspective");
      if (p === "po") return;
      var text = $(this).contents().filter(function () { return this.nodeType === 3; }).text().trim() || "该工作视角";
      window.showToast(text + "工作视角规划中，敬请期待", "info");
    });

    initToolbar();
    initValueStreamLinkage();

    $("#homeRetryBtn").on("click", function () {
      refreshDemands(state.status);
    });

    // 业需评审按钮：直接打开精简评审抽屉（做减法，内嵌通过/驳回操作）
    $("#top5Tbody").on("click", ".js-demand-review", function (e) {
      e.preventDefault();
      e.stopPropagation();
      var demandId = String($(this).attr("data-review-demand-id") || $(this).attr("data-demand-id") || "").trim();
      if (!demandId) return;
      if (window.DemandDetail && typeof window.DemandDetail.open === "function") {
        window.DemandDetail.open(demandId, { mode: "review" });
      }
    });

    var $targetCard = $('.home-vs-mini-card[data-vs-status="' + state.status + '"]');
    if ($targetCard.length) {
      setActiveCard($targetCard);
    } else {
      var $active = $(".home-vs-mini-card.active").first();
      if (!$active.length) {
        $active = $(".home-vs-mini-card").first();
      }
      if ($active.length) {
        setActiveCard($active);
        state.status = $active.attr("data-vs-status") || "all";
      }
    }
    syncUrl();
    if (state.focus !== "all") {
      refreshValueStreamSummary();
    }
    refreshDemands(state.status);
  });
})(jQuery);
