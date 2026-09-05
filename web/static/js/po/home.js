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
    items: [],
    status: "all",
    page: 1,
    pageSize: 15
  };

  var currentSeq = 0;

  function demandsUrl(status) {
    return "/demands?status=" + encodeURIComponent(status || "all");
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

  function loadDemands(status) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(demandsUrl(status), { method: "GET" })
      .then(function (res) {
        if (!res.ok) { throw new Error("load demands failed (" + res.status + ")"); }
        return res.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error("invalid payload");
        }
        return Array.isArray(payload.items) ? payload.items : [];
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

  function renderList() {
    var total = state.items.length;
    updateTitle(total);
    $("#top5Error").attr("hidden", true);

    if (!total) {
      $("#top5Tbody").empty();
      $("#top5Empty").removeAttr("hidden");
      $("#homePagination").attr("hidden", true);
      return;
    }

    $("#top5Empty").attr("hidden", true);
    var totalPages = Math.max(1, Math.ceil(total / state.pageSize));
    if (state.page > totalPages) {
      state.page = totalPages;
    }

    var start = (state.page - 1) * state.pageSize;
    var pageItems = state.items.slice(start, start + state.pageSize);

    $("#top5Tbody").html(pageItems.map(renderRow).join(""));

    if (window.PersonalList) {
      window.PersonalList.renderPagination({
        container: document.getElementById("homePagination"),
        page: state.page,
        pageSize: state.pageSize,
        total: total,
        onPageChange: function (p) { state.page = p; renderList(); },
        onPageSizeChange: function (s) { state.pageSize = s; state.page = 1; renderList(); }
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
    state.status = status || "all";
    state.page = 1;

    $("#top5Empty").attr("hidden", true);
    $("#top5Error").attr("hidden", true);
    $("#top5Tbody").html('<tr><td colspan="7" class="state-placeholder">正在加载行动列表…</td></tr>');

    return loadDemands(state.status)
      .then(function (items) {
        if (reqSeq !== currentSeq) { return items; }
        state.items = items;
        renderList();
        fillUpdateTime();
        return items;
      })
      .catch(function (err) {
        if (reqSeq !== currentSeq) { return []; }
        state.items = [];
        updateTitle(0);
        $("#top5Tbody").empty();
        $("#top5Empty").attr("hidden", true);
        $("#top5Error").removeAttr("hidden");
        $("#homePagination").attr("hidden", true);
        if (typeof window.showToast === "function") {
          window.showToast("加载需求列表失败，请稍后重试", "danger");
        }
        return [];
      });
  }

  function initValueStreamLinkage() {
    $(".home-vs-mini-card").on("click", function () {
      var $card = $(this);
      var status = $card.attr("data-vs-status");
      if (!status) { return; }
      setActiveCard($card);
      refreshDemands(status);
    });
  }

  function initFocusChips() {
    $("#homeFocusChips .header-quick-chip").on("click", function () {
      var $chip = $(this);
      var focus = $chip.attr("data-focus");
      if (focus === "all" || focus === "pending") {
        $("#homeFocusChips .header-quick-chip").removeClass("active").attr("aria-pressed", "false");
        $chip.addClass("active").attr("aria-pressed", "true");
        refreshDemands(state.status);
      } else {
        if (typeof window.showToast === "function") {
          window.showToast("该焦点筛选数据源待业务规则冻结 (WAIT DECISION)", "info");
        }
      }
    });
  }

  $(function () {
    initValueStreamLinkage();
    initFocusChips();

    $("#homeRetryBtn").on("click", function () {
      refreshDemands(state.status);
    });

    var $active = $(".home-vs-mini-card.active").first();
    if (!$active.length) {
      $active = $(".home-vs-mini-card").first();
      if ($active.length) { setActiveCard($active); }
    }
    refreshDemands($active.attr("data-vs-status") || "all");
  });
})(jQuery);
