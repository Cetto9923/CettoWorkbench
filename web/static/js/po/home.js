/* =============================================================================
   文件: web/static/js/po/home.js
   模块: PO 个人工作台 - 首页交互脚本
   职责: 绑定需求价值流下钻、全站统一工具栏筛选、8列行动列表展示与分页保护
   依赖: personal-list.js, jQuery
   ============================================================================= */

(function ($) {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };
  var PL = window.PersonalList || {};
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
  var hasCorrectedPage = false;

  function demandsUrl(status, page, pageSize) {
    var p = new URLSearchParams();
    p.set("status", status || "all");
    p.set("focus", state.focus);
    p.set("page", String(page || 1));
    p.set("pageSize", String(pageSize || 15));
    return "/demands?" + p.toString();
  }

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var params = new URLSearchParams();
    if (state.focus !== "all") { params.set("focus", state.focus); }
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
    if (["all", "today", "blocked", "overdue", "suspended"].indexOf(focus) >= 0) { state.focus = focus; }
    var st = (sp.get("status") || "").trim();
    if (VALID_STATUSES.indexOf(st) >= 0) {
      state.status = st;
    }
    var p = parseInt(sp.get("page"), 10);
    if (!isNaN(p) && p >= 1) {
      state.page = p;
    }
    var ps = parseInt(sp.get("pageSize"), 10);
    if (!isNaN(ps) && [15, 30, 50, 100].indexOf(ps) >= 0) {
      state.pageSize = ps;
    } else {
      // URL 未带 pageSize 时，优先使用上次保存值，再退回默认值。
      state.pageSize = window.PersonalList.loadPageSize("po.home.pageSize", state.pageSize, [15, 30, 50, 100]);
    }
  }

  function dash(value) {
    var text = (value || "").trim();
    return text || "—";
  }

  var ZENTAO_STATUS_LABELS = {
    draft: "暂存", wait: "待评审", active: "已评审", clarified: "已澄清",
    changed: "已变更", developing: "开发中", testing: "测试中", waitacceptance: "待验收",
    acceptanced: "已验收", waitdeliver: "待交付", delivered: "已交付", released: "已发布",
    closed: "已关闭", suspended: "已挂起", refuse: "已驳回"
  };

  var STORY_STATUS_LABELS = {
    draft: "草稿", reviewing: "评审中", active: "激活", changing: "变更中", closed: "已关闭"
  };

  function getZentaoStatusLabel(key) {
    var k = String(key || "").trim().toLowerCase();
    return ZENTAO_STATUS_LABELS[k] || key || "—";
  }

  function getStoryZentaoStatusLabel(key) {
    var k = String(key || "").trim().toLowerCase();
    return STORY_STATUS_LABELS[k] || key || "—";
  }

  function isStoryItem(item) {
    return (item && String(item.kind || "") === "story") ||
      Number(item && item.storyId) > 0 ||
      /^U\d+$/i.test(String((item && item.id) || ""));
  }

  function getHomeZentaoStatusLabel(item) {
    var raw = String((item && item.zentaoStatus) || "").trim();
    if (raw) {
      return isStoryItem(item) ? getStoryZentaoStatusLabel(raw) : getZentaoStatusLabel(raw);
    }
    var label = String((item && (item.zentaoStatusLabel || item.statusLabel)) || "").trim();
    if (label && !/^待(受理|澄清|排期)$/.test(label)) {
      return label;
    }
    return "—";
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
          pageSize: typeof payload.pageSize === "number" ? payload.pageSize : 15
        };
      });
  }

  function updateTitle(count, displayedCount, pageItemCount) {
    var $title = $("#top5Title");
    if ($title.length) {
      var stage = $(".home-vs-mini-card.active .vs-mini-name").first().text() || "全部";
      $title.text(stage === "全部" ? "全部需求" : stage + "阶段");
    }

    var $caption = $("#homeListCaption");
    if ($caption.length) {
      // 是否命中筛选：与"本页未筛选前的条数"比较，而非与跨页总数比较；
      // 否则任何多页阶段（本页条数天然小于总数）都会被误报成"已筛选"。
      var isFiltered = displayedCount != null && typeof pageItemCount === "number" && displayedCount !== pageItemCount;
      if (count == null) {
        $caption.text("当前列表暂不可用，请重试");
      } else if (isFiltered) {
        $caption.text("已筛选 " + displayedCount + " 条 · 阶段总数 " + count + " 条 · 点击详情进入禅道");
      } else {
        $caption.text("共 " + count + " 条关联需求 · 按阶段查看，点击详情进入禅道");
      }
    }

    var $shown = $("#homeShowingCount");
    var $total = $("#homeTotalCount");
    if (!$shown.length && !$total.length) { return; }
    if (count == null) {
      if ($shown.length) { $shown.text("—"); }
      if ($total.length) { $total.text("—"); }
      return;
    }
    var first = count > 0 ? ((state.page - 1) * state.pageSize) + 1 : 0;
    var pageBound = count > 0 ? Math.min(state.page * state.pageSize, count) : 0;
    var displayed = typeof displayedCount === "number" ? displayedCount : pageBound;
    if (displayed > 0) {
      var last = Math.min(first - 1 + displayed, pageBound);
      if ($shown.length) { $shown.text(first + "-" + last); }
    } else {
      if ($shown.length) { $shown.text("0"); }
    }
    if ($total.length) { $total.text(String(count)); }
  }

  function fillUpdateTime() {
    var now = new Date();
    var pad = function (n) { return n < 10 ? "0" + n : String(n); };
    $("#lastUpdateTime").text(
      now.getFullYear() + "-" + pad(now.getMonth() + 1) + "-" + pad(now.getDate()) + " " +
      pad(now.getHours()) + ":" + pad(now.getMinutes())
    );
  }

  // 阶段 → 主要动作渲染：消费 window.PrimaryAction（来自 primary-action.js）。
  var primaryActionHtml = (window.PrimaryAction && window.PrimaryAction.primaryActionHtml) || function (item) {
    return '<span class="home-unavailable">—</span>';
  };

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var pri = (item.pri || "").trim().toUpperCase();
    var isStory = isStoryItem(item);
    var isDemand = !isStory && (/^US\d+/i.test(id) || /^\d+$/.test(id));
    // data-demand-id 只挂在"按钮 / 抽屉入口"上；标题 <a> 不再挂，避免被 demand-detail 全局委托吞掉。
    var dataAttr = isDemand ? ' data-demand-id="' + esc(id) + '"' : '';

    // 研发需求只显示纯 ID（去掉服务端的 "U" 前缀），保持禅道侧链与渲染一致。
    // 业务需求保留 "US{id}"；检测仍依赖原始 id，不影响 isStoryItem / 搜索逻辑。
    var displayId = isStory ? String(id).replace(/^U/i, "") : id;

    // ID 列：禅道原始详情（新页 / 保留办理上下文），无数据降级为纯文本。
    var idHtml = url
      ? '<a class="table-id-link" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer">' + esc(displayId) + '</a>'
      : '<span class="table-id-link">' + esc(displayId) + '</span>';

    // 操作列：消费 Stage 5 的 primaryAction；当前不存在时显示 "—" 占位。
    var actionHtml = primaryActionHtml(item, isStory);

    var priTag = priorityBadge(item.pri);

    var typeTag = isStory
      ? objectTypeBadge("story")
      : objectTypeBadge("business");

    // 标题：当前页打开工作台详情；href 优先使用 server workbenchUrl，否则退化到 /demands/:id。
    var workbenchHref = String(item.workbenchUrl || "").trim();
    if (!workbenchHref) {
      var cleanId = String(id).replace(/^US/i, "");
      workbenchHref = cleanId ? "/demands/" + encodeURIComponent(cleanId) : "";
    }
    var titleHtml = workbenchHref
      ? '<a class="table-title-link" href="' + esc(workbenchHref) + '">' + esc(item.title || "—") + '</a>'
      : esc(item.title || "—");

    var statusText = getHomeZentaoStatusLabel(item);
    var dotClass = (statusText === "开发中" || statusText === "测试中" || statusText === "待验收") ? "active" :
      (statusText === "已挂起" || statusText === "已驳回") ? "danger" : "default";

    return '<tr>' +
      '<td class="c-id">' + idHtml + '</td>' +
      '<td class="c-title" title="' + esc(item.title || "") + '">' + titleHtml + '</td>' +
      '<td class="c-type">' + typeTag + '</td>' +
      '<td class="c-pri">' + priTag + '</td>' +
      '<td class="c-stage"><span class="stage-tag">' + esc(item.valueStream || item.stage || "—") + '</span></td>' +
      '<td class="c-zt-status"><span class="status-tag"><i class="status-dot ' + dotClass + '"></i> ' + esc(statusText) + '</span></td>' +
      '<td class="c-owner">' + esc(dash(item.nextOwner || item.owner)) + '</td>' +
      '<td class="c-actions">' + actionHtml + '</td>' +
      '</tr>';
  }

  function filterItems(items) {
    if (!items || !items.length) { return []; }
    var kw = (state.keyword || "").toLowerCase().trim();
    var objType = state.objectType || "all";
    var pri = state.priority || "all";

    return items.filter(function (item) {
      if (kw) {
        var idMatch = String(item.id || "").toLowerCase().indexOf(kw) >= 0;
        var titleMatch = String(item.title || "").toLowerCase().indexOf(kw) >= 0;
        var ownerMatch = String(item.nextOwner || item.owner || "").toLowerCase().indexOf(kw) >= 0;
        if (!idMatch && !titleMatch && !ownerMatch) { return false; }
      }
      if (objType !== "all") {
        var isStory = isStoryItem(item);
        if (objType === "demand" && isStory) { return false; }
        if (objType === "story" && !isStory) { return false; }
      }
      if (pri !== "all") {
        var p = String(item.pri || "").toLowerCase();
        if (pri === "p1" && p !== "p1") { return false; }
        if (pri === "p2" && p !== "p2") { return false; }
        if (pri === "p3" && p !== "p3" && p !== "p4") { return false; }
      }
      return true;
    });
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
    $("#top5Tbody").html('<tr><td colspan="8" class="state-placeholder">正在加载行动列表…</td></tr>');

    $("#top5List").attr("aria-busy", "true");
    $("#homeRefreshBtn").prop("disabled", true);
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
        $("#homeRefreshBtn").prop("disabled", false);
        renderList(total);
        fillUpdateTime();
      })
      .catch(function (err) {
        if (reqSeq !== currentSeq) { return; }
        hasCorrectedPage = false;
        $("#top5List").attr("aria-busy", "false");
        $("#homeRefreshBtn").prop("disabled", false);
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
        renderList(rawItems.length);
      }, 200);
    });

    $("#homeObjectType").on("change", function () {
      state.objectType = $(this).val();
      renderList(rawItems.length);
    });

    $("#homePriority").on("change", function () {
      state.priority = $(this).val();
      renderList(rawItems.length);
    });

    $("#homeRelationSegment button").on("click", function () {
      $("#homeRelationSegment button").removeClass("active");
      $(this).addClass("active");
      state.relation = $(this).data("relation") || "all";
      renderList(rawItems.length);
    });

    $("#homeResetBtn").on("click", function () {
      $("#homeKeyword").val("");
      $("#homeObjectType").val("all");
      $("#homePriority").val("all");
      $("#homeRelationSegment button").removeClass("active").first().addClass("active");
      state.keyword = "";
      state.objectType = "all";
      state.priority = "all";
      state.relation = "all";
      renderList(rawItems.length);
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
      refreshDemands(state.status);
    });
    $("#homeQuickChips [data-home-focus]").removeClass("active").attr("aria-pressed", "false");
    $('#homeQuickChips [data-home-focus="' + state.focus + '"]').addClass("active").attr("aria-pressed", "true");
    initToolbar();
    initValueStreamLinkage();

    $("#homeRetryBtn, #homeRefreshBtn").on("click", function () {
      refreshDemands(state.status);
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
    refreshDemands(state.status);
  });
})(jQuery);
