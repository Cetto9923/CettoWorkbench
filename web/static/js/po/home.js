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

  function dash(value) {
    var text = (value || "").trim();
    return text || "—";
  }

  function extractNumericId(raw) {
    var s = String(raw == null ? "" : raw).trim().replace(/^#/, "");
    if (!s) return "";
    var m = s.match(/^(?:US|REQ|SUB)-?(\d+)$/i);
    if (m) return m[1];
    m = s.match(/^(?:RD|U)-?(\d+)$/i);
    if (m) return m[1];
    m = s.match(/^(\d+)$/);
    return m ? m[1] : "";
  }

  function isStoryItem(item) {
    var kind = String((item && item.kind) || "").toLowerCase();
    if (kind === "story" || kind === "independent_story") {
      return true;
    }
    if (kind === "demand" || kind === "business" || kind === "sub_demand") {
      return false;
    }
    return (
      Number(item && item.storyId) > 0 ||
      /^U\d+$/i.test(String((item && item.id) || "").replace(/^#/, ""))
    );
  }

  function objectTypeKind(item) {
    return isStoryItem(item) ? "story" : "business";
  }

  // 对齐原型 PersonalList.idChipHtml：左段类型缩写 + 右段 #ID
  function idChipHtml(kind, idHtml) {
    var isStory = kind === "story";
    var label = isStory ? "研需" : "业需";
    var cls = isStory ? "wb-type-story" : "wb-type-business";
    var safeId = typeof idHtml === "string" ? idHtml.trim() : "";
    if (!safeId) {
      return '<span class="wb-type ' + cls + '"><span class="wb-type-tag">' + label + "</span></span>";
    }
    if (safeId.charAt(0) !== "#" && safeId.indexOf(">#") < 0 && !/^#/.test(safeId)) {
      if (/^<([a-zA-Z0-9]+)\b([^>]*)>([\s\S]*)<\/\1>$/i.test(safeId)) {
        safeId = safeId.replace(
          /^<([a-zA-Z0-9]+)\b([^>]*)>([\s\S]*)<\/\1>$/i,
          "<$1$2>#$3</$1>"
        );
      } else {
        safeId = "#" + safeId;
      }
    }
    return (
      '<span class="wb-type ' +
      cls +
      '"><span class="wb-type-tag">' +
      label +
      '</span><span class="wb-type-id">' +
      safeId +
      "</span></span>"
    );
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
        "<div class=\"empty-state-title\">当前焦点暂无事项</div>" +
        "<div class=\"empty-state-hint\">可切换顶部焦点或价值流阶段查看其他队列</div>" +
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

  function canShowCancelReview(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return !!item.canCancelReview;
  }

  function canShowSubmitReview(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return !!item.canSubmitReview;
  }

  function canShowEdit(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return !!item.canEdit;
  }

  // 提测阶段占位按钮：业需且禅道状态 developing，或价值流标签为「提测」
  function canShowSubmitTest(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    var zt = String(item.zentaoStatus || "").trim().toLowerCase();
    if (zt === "developing") {
      return true;
    }
    return String(item.valueStream || item.stage || "").trim() === "提测";
  }

  // 澄清阶段：业需价值流为「澄清」时展示跳转禅道澄清页
  function canShowClarify(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return String(item.valueStream || item.stage || "").trim() === "澄清";
  }

  // 评价反馈阶段：业需价值流为「评价反馈」时展示跳转禅道评价页
  function canShowAppraise(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return String(item.valueStream || item.stage || "").trim() === "评价反馈";
  }

  // 联调测试阶段：业需价值流为「联调测试」且有测试单链接时展示
  function canShowTesttask(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return String(item.valueStream || item.stage || "").trim() === "联调测试";
  }

  // 发起交付阶段：业需价值流为「发起交付」时展示静态按钮（暂不接业务）
  function canShowDeliver(item) {
    if (!item || String(item.kind || "") === "story") {
      return false;
    }
    return String(item.valueStream || item.stage || "").trim() === "发起交付";
  }

  // 受理阶段操作按钮（仅展示，提交/撤销/编辑暂不接业务）
  function acceptActionButtons(item) {
    var parts = [];
    var demandId = escapeHtml(item.id || "");
    if (canShowReview(item)) {
      parts.push(
        "<button type=\"button\" class=\"table-action-btn primary js-demand-review\" data-demand-id=\"" +
          demandId +
          "\">评审</button>"
      );
    }
    if (canShowCancelReview(item)) {
      parts.push(
        "<button type=\"button\" class=\"table-action-btn secondary js-cancel-review\" data-demand-id=\"" +
          demandId +
          "\">撤销评审</button>"
      );
    }
    if (canShowSubmitReview(item)) {
      parts.push(
        "<button type=\"button\" class=\"table-action-btn primary js-submit-review\" data-demand-id=\"" +
          demandId +
          "\">提交评审</button>"
      );
    }
    if (canShowEdit(item)) {
      parts.push(
        "<button type=\"button\" class=\"table-action-btn secondary js-edit-demand\" data-demand-id=\"" +
          demandId +
          "\">编辑</button>"
      );
    }
    return parts;
  }

  // 对齐原型 PersonalList.statusTagHtml：胶囊 + 语义色 + 小圆点
  function statusTagHtml(text) {
    var raw = String(text == null ? "" : text).trim();
    if (!raw || raw === "—" || raw === "--") {
      return '<span class="wb-status-tag wb-status-neutral">—</span>';
    }
    var lower = raw.toLowerCase();
    var semantic = "neutral";
    if (
      lower.indexOf("关闭") >= 0 ||
      lower.indexOf("closed") >= 0 ||
      lower.indexOf("暂存") >= 0 ||
      lower.indexOf("草稿") >= 0 ||
      lower.indexOf("draft") >= 0 ||
      lower.indexOf("取消") >= 0
    ) {
      semantic = "neutral";
    } else if (
      lower.indexOf("澄清") >= 0 ||
      lower.indexOf("完成") >= 0 ||
      lower.indexOf("done") >= 0 ||
      lower.indexOf("验收") >= 0 ||
      lower.indexOf("发布") >= 0 ||
      lower.indexOf("通过") >= 0 ||
      lower.indexOf("正常") >= 0 ||
      lower.indexOf("已解决") >= 0 ||
      lower.indexOf("已闭环") >= 0 ||
      lower.indexOf("激活") >= 0
    ) {
      semantic = "success";
    } else if (
      lower.indexOf("待") >= 0 ||
      lower.indexOf("wait") >= 0 ||
      lower.indexOf("排期") >= 0 ||
      lower.indexOf("评审中") >= 0 ||
      lower.indexOf("审批中") >= 0 ||
      lower.indexOf("预警") >= 0 ||
      lower.indexOf("关注") >= 0
    ) {
      semantic = "warning";
    } else if (
      lower.indexOf("挂起") >= 0 ||
      lower.indexOf("驳回") >= 0 ||
      lower.indexOf("阻塞") >= 0 ||
      lower.indexOf("超期") >= 0 ||
      lower.indexOf("逾期") >= 0 ||
      lower.indexOf("失败") >= 0 ||
      lower.indexOf("风险") >= 0 ||
      lower.indexOf("异常") >= 0 ||
      lower.indexOf("延期") >= 0
    ) {
      semantic = "danger";
    } else if (
      lower.indexOf("开发") >= 0 ||
      lower.indexOf("doing") >= 0 ||
      lower.indexOf("测试") >= 0 ||
      lower.indexOf("处理") >= 0 ||
      lower.indexOf("进行") >= 0 ||
      lower.indexOf("评审") >= 0 ||
      lower.indexOf("active") >= 0
    ) {
      semantic = "processing";
    }
    return (
      '<span class="wb-status-tag wb-status-' +
      semantic +
      '"><i class="wb-status-dot" aria-hidden="true"></i>' +
      escapeHtml(raw) +
      "</span>"
    );
  }

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var isStory = isStoryItem(item);
    var numId = extractNumericId(id) || String(id).replace(/^#/, "");
    var displayId = isStory ? numId : numId ? "US" + numId : "";
    var idInner = url
      ? "<a " + zentaoLinkAttrs(url, "row-id-link") + ">" + escapeHtml(displayId) + "</a>"
      : "<span class=\"row-id-link\">" + escapeHtml(displayId) + "</span>";
    var idChip = idChipHtml(objectTypeKind(item), idInner);
    var actionParts = acceptActionButtons(item);
    if (canShowClarify(item)) {
      var clarifyUrl = String(item.clarifyUrl || "").trim();
      if (clarifyUrl) {
        actionParts.push(
          "<a " + zentaoLinkAttrs(clarifyUrl, "table-action-btn primary") + ">澄清</a>"
        );
      }
    }
    if (canShowAppraise(item)) {
      var appraiseUrl = String(item.appraiseUrl || "").trim();
      if (appraiseUrl) {
        actionParts.push(
          "<a " + zentaoLinkAttrs(appraiseUrl, "table-action-btn primary") + ">评价</a>"
        );
      }
    }
    if (canShowTesttask(item)) {
      var testtaskUrl = String(item.testtaskUrl || "").trim();
      if (testtaskUrl) {
        actionParts.push(
          "<a " + zentaoLinkAttrs(testtaskUrl, "table-action-btn primary") + ">测试单</a>"
        );
      }
    }
    if (!actionParts.length && canShowSubmitTest(item)) {
      actionParts.push(
        "<button type=\"button\" class=\"table-action-btn primary js-submit-test\" data-demand-id=\"" +
          escapeHtml(item.id || "") +
          "\">提测</button>"
      );
    }
    if (canShowDeliver(item)) {
      actionParts.push(
        "<button type=\"button\" class=\"table-action-btn primary js-initiate-deliver\" data-demand-id=\"" +
          escapeHtml(item.id || "") +
          "\">发起交付</button>"
      );
    }
    var actionHtml = actionParts.length
      ? "<div class=\"table-action-group\">" + actionParts.join("") + "</div>"
      : "";
    var priNumMatch = String(item.pri || "").match(/(\d+)/);
    var priNum = priNumMatch ? priNumMatch[1] : "";
    var titleInner =
      (item.pri
        ? '<span class="wb-priority" data-priority="' +
          escapeHtml(priNum) +
          '">' +
          escapeHtml(item.pri) +
          "</span> "
        : "") +
      escapeHtml(item.title || "");
    var titleHtml = url
      ? "<a " + zentaoLinkAttrs(url, "row-title-link") + ">" + titleInner + "</a>"
      : titleInner;

    return (
      "<div class=\"top5-row\">" +
      "<div class=\"c-id\">" + idChip + "</div>" +
      "<div class=\"row-title c-title\" title=\"" + escapeHtml(item.title || "") + "\">" +
      titleHtml +
      "</div>" +
      "<div class=\"row-stage c-stage\"><span class=\"stage-tag\">" + escapeHtml(item.valueStream || item.stage || "—") + "</span></div>" +
      "<div class=\"row-zt-status c-zt-status\" title=\"" +
      escapeHtml(item.zentaoStatus || "") +
      "\">" +
      statusTagHtml(getHomeZentaoStatusLabel(item)) +
      "</div>" +
      "<div class=\"row-owner c-owner\">" + escapeHtml(dash(item.nextOwner || item.owner)) + "</div>" +
      "<div class=\"row-actions c-actions\">" + actionHtml + "</div>" +
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

  function findListItemByDemandId(demandId) {
    for (var i = 0; i < state.items.length; i++) {
      if (String(state.items[i].id || "") === demandId) {
        return state.items[i];
      }
    }
    return null;
  }

  function bindReviewButtons($list) {
    $list.find(".js-demand-review").on("click", function () {
      var demandId = String($(this).attr("data-demand-id") || "").trim();
      var item = findListItemByDemandId(demandId);
      if (!item || typeof window.openPoDemandReviewDrawer !== "function") {
        return;
      }
      window.openPoDemandReviewDrawer(item);
    });
  }

  function bindSubmitTestButtons($list) {
    $list.find(".js-submit-test").on("click", function () {
      var demandId = String($(this).attr("data-demand-id") || "").trim();
      var item = findListItemByDemandId(demandId);
      if (!item || typeof window.openPoSubmitTestModal !== "function") {
        return;
      }
      window.openPoSubmitTestModal(item);
    });
  }

  function bindDeliverButtons($list) {
    $list.find(".js-initiate-deliver").on("click", function () {
      var demandId = String($(this).attr("data-demand-id") || "").trim();
      var item = findListItemByDemandId(demandId);
      if (typeof window.openPoDeliverModal !== "function") {
        return;
      }
      window.openPoDeliverModal(item || demandId);
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
      "<div class=\"top5-cols\"><span>对象#ID</span><span>事项标题</span><span>当前阶段</span><span>状态</span><span>当前负责人</span><span>操作</span></div>";
    html += $.map(state.items, renderRow).join("");
    $("#top5List").html(html);
    bindZentaoLinks($("#top5List"));
    bindReviewButtons($("#top5List"));
    bindSubmitTestButtons($("#top5List"));
    bindDeliverButtons($("#top5List"));
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

    $(".focus-card.vs-trigger").on("click", function () {
      var targetStage = $(this).attr("data-stage-target");
      if (!targetStage) {
        return;
      }
      var $card = $('.home-vs-mini-card[data-vs-status="' + targetStage + '"]');
      if ($card.length) {
        setActiveCard($card);
        refreshDemands(targetStage);
      }
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

  $(function () {
    initValueStreamLinkage();
    initFocalChips();

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
