/* =============================================================================
   文件: web/static/js/po/home.js
   模块: PO 个人工作台 - 首页交互脚本
   职责: 绑定需求价值流下钻、全站统一工具栏筛选、7列行动列表展示与分页保护
   依赖: personal-list.js, home-render.js, jQuery
   ============================================================================= */

(function ($) {
  "use strict";


  var state = {
    status: "all",
    focus: "all",
    page: 1,
    pageSize: 15,
    keyword: "",
    relation: "all",
    objectType: "all",
    priority: "all"
  };

  var rawItems = [];
  var VALID_STATUSES = ["all", "accept", "clarify", "schedule", "developing", "testing", "waitacceptance", "acceptanced", "publish", "released"];
  var currentSeq = 0;
  var summarySeq = 0;
  var hasCorrectedPage = false;

  function demandsUrl(status, page, pageSize) {
    var p = new URLSearchParams();
    p.set("status", status || "all");
    p.set("focus", state.focus);
    p.set("page", String(page || 1));
    p.set("pageSize", String(pageSize || 15));
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
    var qs = params.toString();
    var newUrl = window.location.pathname + (qs ? "?" + qs : "");
    window.history.replaceState(null, "", newUrl);
  }

  function initFromUrl() {
    // 不能在无 query 时提前 return：否则首次直接打开 /home（无任何参数）会跳过
    // 下面的 pageSize 记忆读取，永远使用默认值，而不是用户上次保存的 pageSize。
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
    if (!isNaN(ps) && [10, 15, 20, 30, 50].indexOf(ps) >= 0) {
      state.pageSize = ps;
    } else {
      // URL 未带 pageSize 时，优先使用上次保存值，再退回默认值。
      state.pageSize = window.PersonalList.loadPageSize("po.home.pageSize", state.pageSize, [10, 15, 20, 30, 50]);
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
    if (state.focus === "all") {
      renderValueStreamSummary([]);
      return Promise.resolve();
    }
    var fetchFn = window.appFetch || fetch;
    var seq = ++summarySeq;
    return loadDemands("all", 1, 1).then(function (res) {
      if (seq !== summarySeq) { return; }
      renderValueStreamSummary(res.stageSummary || []);
    }).catch(function () {
      // 列表请求仍照常展示错误；统计卡保留服务端初始值，避免伪造为 0。
    });
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
      $("#top5Empty").removeAttr("hidden");
      $("#homePagination").attr("hidden", true);
      return;
    }

    $("#top5Empty").attr("hidden", true);
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
        if (res.stageSummary && res.stageSummary.length) {
          renderValueStreamSummary(res.stageSummary);
        }
        fillUpdateTime();
      })
      .catch(function (err) {
        if (reqSeq !== currentSeq) { return; }
        hasCorrectedPage = false;
        $("#top5List").attr("aria-busy", "false");
        $("#lastUpdateTime").text("—");
        updateTitle(null);
        $("#top5Tbody").empty();
        $("#top5Empty").attr("hidden", true);
        $("#top5Error").removeAttr("hidden");
        $("#homePagination").attr("hidden", true);
        if (typeof window.showToast === "function") {
          window.showToast("加载需求列表失败，请稍后重试", "danger");
        }
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
        // 工具栏筛选由服务端 SQL 接管：直接拉服务端，避免 Total / 分页与当前页不一致。
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
      refreshDemands(state.status);
    });

    $("#homeRelationSegment button").on("click", function () {
      $("#homeRelationSegment button").removeClass("active");
      $(this).addClass("active");
      state.relation = $(this).data("relation") || "all";
      state.page = 1;
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

  window.refreshPoHomeDemands = function () {
    refreshValueStreamSummary(); return refreshDemands(state.status);
  };

  $(function () {
    initFromUrl();
    $("#homeQuickChips [data-home-focus]").on("click", function (event) {
      event.preventDefault();
      state.focus = $(this).attr("data-home-focus") || "all";
      state.page = 1;
      hasCorrectedPage = false;
      $("#homeQuickChips [data-home-focus]").removeClass("active").attr("aria-pressed", "false");
      $(this).addClass("active").attr("aria-pressed", "true");
      syncUrl();
      refreshValueStreamSummary();
      refreshDemands(state.status);
    });
    $("#homeQuickChips [data-home-focus]").removeClass("active").attr("aria-pressed", "false");
    $('#homeQuickChips [data-home-focus="' + state.focus + '"]').addClass("active").attr("aria-pressed", "true");
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
