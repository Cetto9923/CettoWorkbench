(function ($) {
  "use strict";

  // 与 components/pager 选项一致
  var PAGE_SIZES = [10, 20, 50, 100];
  var state = {
    items: [],
    total: 0,
    status: "all",
    focal: "myPending",
    page: 1,
    pageSize: 10
  };

  var FOCAL_LABELS = {
    today: "今日必推",
    myPending: "待我处理",
    blocked: "阻塞",
    overdue: "超期"
  };

  function demandsUrl(status, page, pageSize) {
    return (
      "/demands?status=" +
      encodeURIComponent(status || "all") +
      "&page=" +
      encodeURIComponent(String(page || 1)) +
      "&pageSize=" +
      encodeURIComponent(String(pageSize || 10))
    );
  }

  function escapeHtml(text) {
    return $("<div>").text(text == null ? "" : String(text)).html();
  }

  function actionLabel(item) {
    return (item.next || "").trim() || "跟进";
  }

  function dash(value) {
    var text = (value || "").trim();
    return text || "—";
  }

  // 禅道业需 status → 中文（与原型 po-core.js ZENTAO_STATUS_LABELS 对齐）
  var ZENTAO_STATUS_LABELS = {
    draft: "暂存",
    wait: "待评审",
    active: "已评审",
    clarified: "已澄清",
    changed: "已变更",
    developing: "开发中",
    testing: "测试中",
    waitacceptance: "待验收",
    acceptanced: "已验收",
    waitdeliver: "待交付",
    delivered: "已交付",
    released: "已发布",
    closed: "已关闭",
    suspended: "已挂起",
    refuse: "已驳回"
  };

  // 禅道研需 zt_story.status → 中文（与业需同名码语义不同，禁止共用）
  var STORY_STATUS_LABELS = {
    draft: "草稿",
    reviewing: "评审中",
    active: "激活",
    changing: "变更中",
    closed: "已关闭"
  };

  function getZentaoStatusLabel(key) {
    var k = String(key || "").trim().toLowerCase();
    if (!k) {
      return "";
    }
    return ZENTAO_STATUS_LABELS[k] || key;
  }

  function getStoryZentaoStatusLabel(key) {
    var k = String(key || "").trim().toLowerCase();
    if (!k) {
      return "";
    }
    return STORY_STATUS_LABELS[k] || key;
  }

  // 对齐原型 getHomeZentaoStatusLabel：业需/研需分表映射，禁止用动作态冒充禅道状态
  function getHomeZentaoStatusLabel(item) {
    var raw = String((item && item.zentaoStatus) || "").trim();
    if (raw) {
      var isStory =
        (item && String(item.kind || "") === "story") ||
        Number(item && item.storyId) > 0 ||
        /^U\d+$/i.test(String((item && item.id) || ""));
      if (isStory) {
        return getStoryZentaoStatusLabel(raw);
      }
      return getZentaoStatusLabel(raw);
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
        if (!res.ok) {
          throw new Error("load demands failed");
        }
        return res.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error("invalid payload");
        }
        return {
          items: Array.isArray(payload.items) ? payload.items : [],
          total: Number(payload.total) || 0,
          page: Number(payload.page) || page || 1,
          pageSize: Number(payload.pageSize) || pageSize || 10
        };
      })
      .catch(function () {
        if (typeof window.showToast === "function") {
          window.showToast("加载需求列表失败，请稍后重试", "danger");
        }
        return { items: [], total: 0, page: 1, pageSize: pageSize || 10 };
      });
  }

  function updateTitle(count) {
    var $title = $("#top5Title");
    if (!$title.length) {
      return;
    }
    var focal = FOCAL_LABELS[state.focal] || "待我处理";
    var stage = $(".home-vs-mini-card.active .vs-mini-name").first().text() || "全部";
    var stagePart = stage && stage !== "全部" ? " · " + escapeHtml(stage) : "";
    $title.html(
      "<i class=\"fas fa-list-check\"></i> 统一行动列表 · " +
        escapeHtml(focal) +
        stagePart +
        "（" +
        count +
        "）"
    );
  }

  function renderEmpty() {
    $("#top5List").html(
      "<div class=\"empty-state\">" +
        "<div style=\"font-weight:700;color:var(--po-t2);margin-bottom:6px\">当前焦点暂无事项</div>" +
        "<div style=\"font-size:12px\">可切换顶部焦点或价值流阶段查看其他队列</div>" +
        "</div>"
    );
  }

  function zentaoLinkAttrs(url, extraClass) {
    var cls = ("js-zentao-link " + (extraClass || "")).trim();
    return (
      "href=\"" +
      escapeHtml(url) +
      "\" class=\"" +
      cls +
      "\" target=\"_blank\" rel=\"noopener noreferrer\""
    );
  }

  function canShowReview(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return !!item.canReview;
  }

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var pri = item.pri || "";
    var idHtml = url
      ? "<a " + zentaoLinkAttrs(url, "row-id-link") + ">" + escapeHtml(id) + "</a>"
      : "<span class=\"row-id-link\">" + escapeHtml(id) + "</span>";
    var action = actionLabel(item);
    var actionHtml = "";
    if (canShowReview(item)) {
      actionHtml =
        "<button type=\"button\" class=\"action-btn primary js-demand-review\" data-demand-id=\"" +
        escapeHtml(item.id || "") +
        "\">评审</button>";
    }
    var titleInner =
      (pri ? "<span class=\"inline-pri " + escapeHtml(pri) + "\">" + escapeHtml(pri) + "</span>" : "") +
      escapeHtml(item.title || "");
    var titleHtml = url
      ? "<a " + zentaoLinkAttrs(url, "row-title-link") + ">" + titleInner + "</a>"
      : titleInner;

    return (
      "<div class=\"top5-row\">" +
      idHtml +
      "<div class=\"row-title\" title=\"" + escapeHtml(item.title || "") + "\">" +
      titleHtml +
      "</div>" +
      "<div class=\"row-stage\"><span class=\"stage-tag\">" + escapeHtml(item.valueStream || item.stage || "—") + "</span></div>" +
      "<div class=\"row-zt-status\"><span class=\"status-tag st-progress\" title=\"" +
      escapeHtml(item.zentaoStatus || "") +
      "\">" +
      escapeHtml(getHomeZentaoStatusLabel(item)) +
      "</span></div>" +
      "<div class=\"row-next\">" + escapeHtml(action) + "</div>" +
      "<div class=\"row-owner\">" + escapeHtml(dash(item.nextOwner || item.owner)) + "</div>" +
      "<div class=\"row-actions\">" + actionHtml + "</div>" +
      "</div>"
    );
  }

  // 对齐 internal/pkg/pagination.buildPages(current, total, around=2)
  function buildPages(currentPage, totalPages, around) {
    if (totalPages <= 0) {
      return [];
    }
    var start = Math.max(1, currentPage - around);
    var end = Math.min(totalPages, currentPage + around);
    var pages = [];
    for (var i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  }

  // DOM 对齐 web/templates/components/pager.html；翻页走后端查询
  function renderPagination(total) {
    var $pager = $("#homePager");
    if (!total) {
      $pager.attr("hidden", true).empty();
      return;
    }
    var page = state.page;
    var ps = state.pageSize;
    var totalPages = Math.max(1, Math.ceil(total / Math.max(1, ps)));
    var pages = buildPages(page, totalPages, 2);
    var hasPrev = page > 1;
    var hasNext = totalPages > 0 && page < totalPages;
    var html = "<div class=\"pagination-container\" aria-label=\"分页\"><ul class=\"pagination\">";

    if (hasPrev) {
      html +=
        "<li class=\"page-item\"><a class=\"page-link\" href=\"#\" data-page=\"" +
        (page - 1) +
        "\" aria-label=\"上一页\"><i class=\"bi bi-chevron-left\" aria-hidden=\"true\"></i></a></li>";
    } else {
      html +=
        "<li class=\"page-item disabled\"><span class=\"page-link\" aria-hidden=\"true\"><i class=\"bi bi-chevron-left\"></i></span></li>";
    }

    if (totalPages > 1 && pages.length > 0) {
      if (pages[0] > 1) {
        html += "<li class=\"page-item\"><a class=\"page-link\" href=\"#\" data-page=\"1\">1</a></li>";
      }
      if (pages[0] > 2) {
        html += "<li class=\"page-item disabled\"><span class=\"page-link\">•••</span></li>";
      }
      pages.forEach(function (p) {
        if (p === page) {
          html +=
            "<li class=\"page-item active\"><span class=\"page-link\" aria-current=\"page\">" +
            p +
            "</span></li>";
        } else {
          html +=
            "<li class=\"page-item\"><a class=\"page-link\" href=\"#\" data-page=\"" +
            p +
            "\">" +
            p +
            "</a></li>";
        }
      });
      if (pages[pages.length - 1] < totalPages - 1) {
        html += "<li class=\"page-item disabled\"><span class=\"page-link\">•••</span></li>";
      }
      if (pages[pages.length - 1] < totalPages) {
        html +=
          "<li class=\"page-item\"><a class=\"page-link\" href=\"#\" data-page=\"" +
          totalPages +
          "\">" +
          totalPages +
          "</a></li>";
      }
    } else {
      html +=
        "<li class=\"page-item active\"><span class=\"page-link\" aria-current=\"page\">" +
        page +
        "</span></li>";
    }

    if (hasNext) {
      html +=
        "<li class=\"page-item\"><a class=\"page-link\" href=\"#\" data-page=\"" +
        (page + 1) +
        "\" aria-label=\"下一页\"><i class=\"bi bi-chevron-right\" aria-hidden=\"true\"></i></a></li>";
    } else {
      html +=
        "<li class=\"page-item disabled\"><span class=\"page-link\" aria-hidden=\"true\"><i class=\"bi bi-chevron-right\"></i></span></li>";
    }

    html += "</ul><div class=\"pagination-options\">";
    html += "<span class=\"pagination-total\">共 " + total + " 条</span>";
    html += "<form method=\"GET\" action=\"#\" class=\"pagination-size-form\">";
    html += "<select name=\"pageSize\" class=\"select\" aria-label=\"每页条数\">";
    PAGE_SIZES.forEach(function (n) {
      html +=
        "<option value=\"" +
        n +
        "\"" +
        (n === ps ? " selected" : "") +
        ">" +
        n +
        " 条/页</option>";
    });
    html += "</select></form>";
    html += "<form method=\"GET\" action=\"#\" class=\"pagination-jump-form\">";
    html += "<input type=\"hidden\" name=\"pageSize\" value=\"" + ps + "\">";
    html += "<span>跳至</span>";
    html +=
      "<input type=\"number\" min=\"1\" max=\"" +
      totalPages +
      "\" name=\"page\" class=\"input\" value=\"" +
      page +
      "\">";
    html += "<span>页</span>";
    html += "<button type=\"submit\" class=\"btn btn-neutral btn-sm\">前往</button>";
    html += "</form></div></div>";

    $pager.html(html).removeAttr("hidden");
    bindPagination(totalPages);
  }

  function bindZentaoLinks($list) {
    $list.find("a.js-zentao-link").on("click", function (e) {
      var href = (this.getAttribute("href") || "").trim();
      if (!href) {
        return;
      }
      e.preventDefault();
      window.open(href, "_blank", "noopener,noreferrer");
    });
  }

  function bindReviewButtons($list) {
    $list.find(".js-demand-review").on("click", function () {
      var demandId = String($(this).attr("data-demand-id") || "").trim();
      var item = null;
      for (var i = 0; i < state.items.length; i++) {
        if (String(state.items[i].id || "") === demandId) {
          item = state.items[i];
          break;
        }
      }
      if (!item || typeof window.openPoDemandReviewModal !== "function") {
        return;
      }
      window.openPoDemandReviewModal(item);
    });
  }

  function bindPagination(totalPages) {
    var $pager = $("#homePager");
    $pager.find("a.page-link[data-page]").on("click", function (e) {
      e.preventDefault();
      var next = parseInt($(this).attr("data-page"), 10) || 1;
      if (next < 1 || next > totalPages || next === state.page) {
        return;
      }
      state.page = next;
      reloadDemands();
    });
    $pager.find(".pagination-size-form").on("submit", function (e) {
      e.preventDefault();
    });
    $pager.find(".pagination-size-form select[name='pageSize']").on("change", function () {
      state.pageSize = parseInt($(this).val(), 10) || 10;
      state.page = 1;
      reloadDemands();
    });
    $pager.find(".pagination-jump-form").on("submit", function (e) {
      e.preventDefault();
      var raw = $(this).find("input[name='page']").val();
      var next = parseInt(raw, 10) || 1;
      if (next < 1) {
        next = 1;
      }
      if (next > totalPages) {
        next = totalPages;
      }
      state.page = next;
      reloadDemands();
    });
  }

  function renderList() {
    var total = state.total;
    updateTitle(total);
    if (!total) {
      renderEmpty();
      renderPagination(0);
      return;
    }
    var html =
      "<div class=\"top5-cols\"><span>ID</span><span>标题</span><span>当前阶段</span><span>需求状态</span><span>下一步</span><span>下一责任人</span><span>操作</span></div>";
    html += $.map(state.items, renderRow).join("");
    $("#top5List").html(html);
    bindZentaoLinks($("#top5List"));
    bindReviewButtons($("#top5List"));
    renderPagination(total);
  }

  function setActiveCard($card) {
    $(".home-vs-mini-card").removeClass("active");
    $card.addClass("active");
  }

  function refreshDemands(status) {
    state.status = status || "all";
    state.page = 1;
    return reloadDemands();
  }

  function reloadDemands() {
    return loadDemands(state.status, state.page, state.pageSize).then(function (payload) {
      state.items = payload.items;
      state.total = payload.total;
      state.page = payload.page;
      state.pageSize = payload.pageSize;
      renderList();
      return payload.items;
    });
  }

  window.refreshPoHomeDemands = reloadDemands;

  function initValueStreamLinkage() {
    $(".home-vs-mini-card").on("click", function () {
      var $card = $(this);
      var status = $card.attr("data-vs-status");
      if (!status) {
        return;
      }
      setActiveCard($card);
      refreshDemands(status);
    });
  }

  function initFocalChips() {
    $(".home-hl-kpi").on("click", function () {
      var $btn = $(this);
      $(".home-hl-kpi").removeClass("active");
      $btn.addClass("active");
      state.focal = $btn.attr("data-focal") || "myPending";
      updateTitle(state.total);
    });
  }

  function fillUpdateTime() {
    var now = new Date();
    var pad = function (n) {
      return n < 10 ? "0" + n : String(n);
    };
    $("#lastUpdateTime").text(
      now.getFullYear() +
        "-" +
        pad(now.getMonth() + 1) +
        "-" +
        pad(now.getDate()) +
        " " +
        pad(now.getHours()) +
        ":" +
        pad(now.getMinutes())
    );
  }

  $(function () {
    initValueStreamLinkage();
    initFocalChips();
    fillUpdateTime();

    var $active = $(".home-vs-mini-card.active").first();
    if (!$active.length) {
      $active = $(".home-vs-mini-card").first();
      if ($active.length) {
        setActiveCard($active);
      }
    }
    refreshDemands($active.attr("data-vs-status") || "all");
  });
})(jQuery);
