(function ($) {
  "use strict";

  var PAGE_SIZES = [4, 5, 6, 7, 8, 10];
  var state = {
    items: [],
    status: "all",
    focal: "myPending",
    page: 1,
    pageSize: 6
  };

  var FOCAL_LABELS = {
    today: "今日必推",
    myPending: "待我处理",
    blocked: "阻塞",
    overdue: "超期"
  };

  function demandsUrl(status) {
    return "/demands?status=" + encodeURIComponent(status || "all");
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

  function loadDemands(status) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(demandsUrl(status), { method: "GET" })
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
        return Array.isArray(payload.items) ? payload.items : [];
      })
      .catch(function () {
        if (typeof window.showToast === "function") {
          window.showToast("加载需求列表失败，请稍后重试", "danger");
        }
        return [];
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

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var pri = item.pri || "";
    var idHtml = url
      ? "<a " + zentaoLinkAttrs(url, "row-id-link") + ">" + escapeHtml(id) + "</a>"
      : "<span class=\"row-id-link\">" + escapeHtml(id) + "</span>";
    var action = actionLabel(item);
    var actionHtml = url
      ? "<a " + zentaoLinkAttrs(url, "action-btn primary") + ">" + escapeHtml(action) + "</a>"
      : "<button type=\"button\" class=\"action-btn primary\" disabled>" + escapeHtml(action) + "</button>";
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

  function renderPagination(total) {
    var page = state.page;
    var ps = state.pageSize;
    var totalPages = Math.max(1, Math.ceil(total / Math.max(1, ps)));
    var startIdx = total ? (page - 1) * ps + 1 : 0;
    var endIdx = Math.min(page * ps, total);
    var html = "<div class=\"pagination-container\"><div class=\"pagination\">";
    html += "<span class=\"pagination-summary\">显示 " + startIdx + "-" + endIdx + " / 共 " + total + " 条</span>";
    html += "<div class=\"pagination-controls\">";
    html += "<button type=\"button\" class=\"action-btn js-show-all\">查看全部 " + total + " 条</button>";
    html += "<select class=\"filter-select\" aria-label=\"每页条数\">";
    PAGE_SIZES.forEach(function (n) {
      html += "<option value=\"" + n + "\"" + (n === ps ? " selected" : "") + ">" + n + " 条/页</option>";
    });
    html += "</select>";
    html += "<button type=\"button\" class=\"action-btn small js-page-prev\"" + (page <= 1 ? " disabled" : "") + ">上一页</button>";
    for (var i = 1; i <= totalPages; i++) {
      if (i === 1 || i === totalPages || (i >= page - 1 && i <= page + 1)) {
        html += "<button type=\"button\" class=\"action-btn small js-page-num" + (i === page ? " primary" : "") + "\" data-page=\"" + i + "\">" + i + "</button>";
      } else if (i === page - 2 || i === page + 2) {
        html += "<span class=\"pagination-ellipsis\">...</span>";
      }
    }
    html += "<button type=\"button\" class=\"action-btn small js-page-next\"" + (page >= totalPages ? " disabled" : "") + ">下一页</button>";
    html += "</div></div></div>";
    return html;
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

  function bindPagination(total) {
    var $list = $("#top5List");
    $list.find(".js-show-all").on("click", function () {
      state.pageSize = Math.max(total, 1);
      state.page = 1;
      renderList();
    });
    $list.find(".filter-select").on("change", function () {
      state.pageSize = parseInt($(this).val(), 10) || 6;
      state.page = 1;
      renderList();
    });
    $list.find(".js-page-prev").on("click", function () {
      if (state.page > 1) {
        state.page -= 1;
        renderList();
      }
    });
    $list.find(".js-page-next").on("click", function () {
      var totalPages = Math.max(1, Math.ceil(total / Math.max(1, state.pageSize)));
      if (state.page < totalPages) {
        state.page += 1;
        renderList();
      }
    });
    $list.find(".js-page-num").on("click", function () {
      state.page = parseInt($(this).attr("data-page"), 10) || 1;
      renderList();
    });
  }

  function renderList() {
    var total = state.items.length;
    updateTitle(total);
    if (!total) {
      renderEmpty();
      return;
    }
    var totalPages = Math.max(1, Math.ceil(total / Math.max(1, state.pageSize)));
    if (state.page > totalPages) {
      state.page = totalPages;
    }
    var start = (state.page - 1) * state.pageSize;
    var pageItems = state.items.slice(start, start + state.pageSize);
    var html = "<div class=\"top5-cols\"><span>ID</span><span>标题</span><span>当前阶段</span><span>需求状态</span><span>下一步</span><span>下一责任人</span><span>操作</span></div>";
    html += $.map(pageItems, renderRow).join("");
    html += renderPagination(total);
    $("#top5List").html(html);
    bindZentaoLinks($("#top5List"));
    bindPagination(total);
  }

  function setActiveCard($card) {
    $(".home-vs-mini-card").removeClass("active");
    $card.addClass("active");
  }

  function refreshDemands(status) {
    state.status = status || "all";
    state.page = 1;
    return loadDemands(state.status).then(function (items) {
      state.items = items;
      renderList();
      return items;
    });
  }

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
      updateTitle(state.items.length);
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
