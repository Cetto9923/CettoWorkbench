/* =============================================================================
   文件: web/static/js/po/home.js
   模块: PO 个人工作台 - 首页交互脚本
   职责: 绑定需求价值流下钻筛选、统一行动列表展示、分页与更新时序保护
   依赖: personal-list.js, jQuery
   ============================================================================= */

(function ($) {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };

  var state = {
    status: "all",
    page: 1,
    pageSize: 15
  };

  var VALID_STATUSES = ["all", "accept", "clarify", "schedule", "developing", "testing", "waitacceptance", "acceptanced", "publish", "released"];
  var currentSeq = 0;
  var hasCorrectedPage = false;

  function demandsUrl(status, page, pageSize) {
    var p = new URLSearchParams();
    p.set("status", status || "all");
    p.set("page", String(page || 1));
    p.set("pageSize", String(pageSize || 15));
    return "/demands?" + p.toString();
  }

  function syncUrl() {
    if (!window.history || !window.history.replaceState) { return; }
    var params = new URLSearchParams();
    if (state.status && state.status !== "all") { params.set("status", state.status); }
    if (state.page > 1) { params.set("page", String(state.page)); }
    if (state.pageSize && state.pageSize !== 15) { params.set("pageSize", String(state.pageSize)); }
    var qs = params.toString();
    var newUrl = window.location.pathname + (qs ? "?" + qs : "");
    window.history.replaceState(null, "", newUrl);
  }

  function initFromUrl() {
    if (!window.location.search) { return; }
    var sp = new URLSearchParams(window.location.search);
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
    }
  }

  function actionLabel(item) {
    return (item.next || "").trim() || "查看";
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

  function getHomeZentaoStatusLabel(item) {
    var raw = String((item && item.zentaoStatus) || "").trim();
    if (raw) {
      var isStory = (item && String(item.kind || "") === "story") ||
        Number(item && item.storyId) > 0 ||
        /^U\d+$/i.test(String((item && item.id) || ""));
      return isStory ? getStoryZentaoStatusLabel(raw) : getZentaoStatusLabel(raw);
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

  function updateTitle(count) {
    var $title = $("#top5Title");
    if (!$title.length) { return; }
    var stage = $(".home-vs-mini-card.active .vs-mini-name").first().text() || "全部";
    $title.html(
      '<i class="fas fa-list-check"></i>统一行动列表 · ' +
      esc(stage === "全部" ? "全部生命周期" : stage) +
      "（" + count + "）"
    );
  }

  function fillUpdateTime() {
    var now = new Date();
    var pad = function (n) { return n < 10 ? "0" + n : String(n); };
    $("#lastUpdateTime").text(
      now.getFullYear() + "-" + pad(now.getMonth() + 1) + "-" + pad(now.getDate()) + " " +
      pad(now.getHours()) + ":" + pad(now.getMinutes())
    );
  }

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var pri = item.pri || "";
    var idHtml = url
      ? "<a class=\"table-id-link\" href=\"" + esc(url) + "\" target=\"_blank\" rel=\"noopener noreferrer\">" + esc(id) + "</a>"
      : '<span class="table-id-link">' + esc(id) + "</span>";

    var action = actionLabel(item);
    var actionHtml = url
      ? "<a class=\"table-action-btn\" href=\"" + esc(url) + "\" target=\"_blank\" rel=\"noopener noreferrer\">" + esc(action) + "</a>"
      : '<span class="table-action-btn" style="opacity:.4;cursor:default">' + esc(action) + "</span>";

    var priTag = pri ? '<span class="inline-pri ' + esc(pri.toLowerCase()) + '">' + esc(pri) + "</span> " : "";
    var titleHtml = url
      ? "<a class=\"table-title-link\" href=\"" + esc(url) + "\" target=\"_blank\" rel=\"noopener noreferrer\">" + esc(item.title || "—") + "</a>"
      : esc(item.title || "—");

    return "<tr>" +
      '<td class="c-id">' + idHtml + "</td>" +
      '<td class="c-title" title="' + esc(item.title || "") + '">' + priTag + titleHtml + "</td>" +
      '<td class="c-stage"><span class="stage-tag">' + esc(item.valueStream || item.stage || "—") + "</span></td>" +
      '<td class="c-zt-status"><span class="status-tag">' + esc(getHomeZentaoStatusLabel(item)) + "</span></td>" +
      '<td class="c-next">' + esc(action) + "</td>" +
      '<td class="c-owner">' + esc(dash(item.nextOwner || item.owner)) + "</td>" +
      '<td class="c-actions">' + actionHtml + "</td>" +
      "</tr>";
  }

  function renderList(items, total) {
    updateTitle(total);
    $("#top5Error").attr("hidden", true);

    if (!total || !items.length) {
      $("#top5Tbody").empty();
      $("#top5Empty").removeAttr("hidden");
      $("#homePagination").attr("hidden", true);
      return;
    }

    $("#top5Empty").attr("hidden", true);
    $("#top5Tbody").html(items.map(renderRow).join(""));

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
    $("#top5Tbody").html('<tr><td colspan="7" class="state-placeholder">正在加载行动列表…</td></tr>');

    return loadDemands(state.status, state.page, state.pageSize)
      .then(function (res) {
        if (reqSeq !== currentSeq) { return; }
        var total = res.total;
        var totalPages = Math.max(1, Math.ceil(total / state.pageSize));

        // 越界安全：page > totalPages 时最多纠偏一次
        if (state.page > totalPages && total > 0 && !hasCorrectedPage) {
          hasCorrectedPage = true;
          state.page = totalPages;
          syncUrl();
          refreshDemands(state.status);
          return;
        }
        hasCorrectedPage = false;

        renderList(res.items, total);
        fillUpdateTime();
      })
      .catch(function (err) {
        if (reqSeq !== currentSeq) { return; }
        hasCorrectedPage = false;
        updateTitle(0);
        $("#top5Tbody").empty();
        $("#top5Empty").attr("hidden", true);
        $("#top5Error").removeAttr("hidden");
        $("#homePagination").attr("hidden", true);
        if (typeof window.showToast === "function") {
          window.showToast("加载需求列表失败，请稍后重试", "danger");
        }
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
  }

  function initFocusChips() {
    $("#homeFocusChips .header-quick-chip").on("click", function () {
      var $chip = $(this);
      var focus = $chip.attr("data-focus");
      if (focus === "all") {
        $("#homeFocusChips .header-quick-chip").removeClass("active").attr("aria-pressed", "false");
        $chip.addClass("active").attr("aria-pressed", "true");
        state.status = "all";
        state.page = 1;
        var $card = $('.home-vs-mini-card[data-vs-status="all"]');
        if ($card.length) { setActiveCard($card); }
        syncUrl();
        refreshDemands(state.status);
      } else {
        if (typeof window.showToast === "function") {
          window.showToast("该焦点筛选数据源待业务规则冻结 (WAIT DECISION)", "info");
        }
      }
    });
  }

  $(function () {
    initFromUrl();
    initValueStreamLinkage();
    initFocusChips();

    $("#homeRetryBtn").on("click", function () {
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
