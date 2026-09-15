/* =============================================================================
   文件: web/static/js/po/home-render.js
   模块: PO 个人工作台 - 首页列表/价值流渲染
   职责: 状态标签、行渲染、价值流计数卡与标题文案
   依赖: personal-list.js, primary-action.js
   ============================================================================= */

(function (root) {
  "use strict";

  var esc = root.escapeHtml;
  var PL = root.PersonalList || {};
  var priorityBadge = PL.priorityBadge;
  var primaryActionHtml = (root.PrimaryAction && root.PrimaryAction.primaryActionHtml) || function () {
    return '<span class="home-unavailable">—</span>';
  };

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

  function canShowReview(item) {
    if (!item || isStoryItem(item)) {
      return false;
    }
    return !!item.canReview;
  }

  function resolveDemandEditUrl(item) {
    if (!item || isStoryItem(item)) {
      return "";
    }
    var editUrl = (item.zentaoEditUrl || "").trim();
    if (/^https?:\/\//i.test(editUrl)) {
      if (editUrl.indexOf("#") === -1 && editUrl.indexOf("demand") !== -1) {
        editUrl += "#app=demandpool";
      }
      return editUrl;
    }
    var viewUrl = (item.zentaoUrl || "").trim();
    if (viewUrl) {
      var derived = viewUrl.replace(/\/demand-view-(\d+)\.html/i, "/demand-edit-$1.html");
      if (derived !== viewUrl) {
        return derived;
      }
      derived = viewUrl.replace(/([?&]f=)view(&|$)/i, "$1edit$2");
      if (derived !== viewUrl) {
        return derived;
      }
      if (editUrl && editUrl.charAt(0) === "/" && /^https?:\/\//i.test(viewUrl)) {
        try {
          var parsed = new URL(viewUrl);
          var hash = editUrl.indexOf("#") === -1 ? "#app=demandpool" : "";
          return parsed.origin + editUrl + hash;
        } catch (e) {}
      }
    }
    return "";
  }

  function reviewActionHtml(item) {
    var editUrl = resolveDemandEditUrl(item);
    var editBtn = editUrl
      ? '<a class="table-action-btn secondary" href="' + esc(editUrl) + '" target="_blank" rel="noopener noreferrer">编辑 ↗</a>'
      : "";
    return (
      '<div class="table-action-group">' +
      '<button type="button" class="table-action-btn primary js-demand-review" data-review-demand-id="' +
      esc(item.id || "") +
      '">评审</button>' +
      editBtn +
      '</div>'
    );
  }

  function renderRow(item) {
    var id = item.id || "";
    var url = (item.zentaoUrl || "").trim();
    var isStory = isStoryItem(item);
    var isDemand = !isStory && (/^US\d+/i.test(id) || /^\d+$/.test(id));
    var dataAttr = isDemand ? ' data-demand-id="' + esc(id) + '"' : '';

    var displayId = isStory ? String(id).replace(/^U/i, "") : id;

    var idHtml = url
      ? '<a class="table-id-link" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer">' + esc(displayId) + '</a>'
      : '<span class="table-id-link">' + esc(displayId) + '</span>';

    var actionHtml = canShowReview(item) ? reviewActionHtml(item) : primaryActionHtml(item, isStory);

    var priTag = priorityBadge(item.pri);
    var inlineFlags = priTag;
    if (item.suspended === true || String(item.suspended || "").toLowerCase() === "true" || String(item.suspended || "") === "1") {
      inlineFlags += '<span class="home-inline-flag suspended">挂起</span>';
    }
    if (item.blocked === true || String(item.blocked || "").toLowerCase() === "true" || String(item.blocked || "") === "1") {
      inlineFlags += '<span class="home-inline-flag blocked">阻塞</span>';
    }

    var typeKind = isStory ? "story" : "business";
    var idChip = PL.idChipHtml ? PL.idChipHtml(typeKind, idHtml) :
      '<span class="wb-type wb-type-' + typeKind + '">' + (isStory ? "研发需求" : "业务需求") + " " + idHtml + "</span>";

    var workbenchHref = String(item.workbenchUrl || "").trim();
    if (!workbenchHref && !isStory) {
      var cleanId = String(id).replace(/^US/i, "");
      workbenchHref = cleanId ? "/demands/" + encodeURIComponent(cleanId) : "";
    }
    var titleLink = workbenchHref
      ? '<a class="table-title-link" href="' + esc(workbenchHref) + '">' + esc(item.title || "—") + '</a>'
      : (url ? '<a class="table-title-story" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer">' + esc(item.title || "—") + '</a>' : esc(item.title || "—"));
    var titleHtml = '<div class="home-title-line">' + inlineFlags + titleLink + '</div>';

    var statusText = getHomeZentaoStatusLabel(item);
    var statusHtml = (PL && PL.statusTagHtml) ? PL.statusTagHtml(statusText) : ('<span class="status-tag">' + esc(statusText) + '</span>');

    return '<tr>' +
      '<td class="c-id">' + idChip + '</td>' +
      '<td class="c-title" title="' + esc(item.title || "") + '">' + titleHtml + '</td>' +
      '<td class="c-stage"><span class="stage-tag">' + esc(item.valueStream || item.stage || "—") + '</span></td>' +
      '<td class="c-zt-status">' + statusHtml + '</td>' +
      '<td class="c-owner">' + esc(dash(item.nextOwner || item.owner)) + '</td>' +
      '<td class="c-actions">' + actionHtml + '</td>' +
      '</tr>';
  }

  // 右上角焦点是一级范围，价值流卡片必须显示该范围内各阶段的数量、条数分布与耗时。
  // 当焦点为全部 (all) 或未提供 stageSummary 时，恢复全景流的服务端基线统计。
  function renderValueStreamSummary(rows, focus) {
    if (!document || !document.querySelectorAll) { return; }
    var isAllFocus = !focus || focus === "all";
    if (isAllFocus || !rows || !rows.length) {
      document.querySelectorAll(".home-vs-mini-card").forEach(function (card) {
        var baseCount = card.getAttribute("data-base-count");
        var baseMeta = card.getAttribute("data-base-meta");
        var baseDuration = card.getAttribute("data-base-duration");
        var count = card.querySelector(".vs-mini-count");
        var breakdown = card.querySelector(".vs-mini-breakdown") || card.querySelector(".vs-mini-meta");
        var dur = card.querySelector(".vs-mini-dur") || card.querySelector(".vs-mini-duration-row .val");
        if (baseCount && count) { count.textContent = baseCount; }
        if (baseMeta && breakdown) { breakdown.textContent = baseMeta; }
        if (baseDuration && dur) {
          dur.innerHTML = (!baseDuration || baseDuration === "—") ? "—" : '<span class="tag">均</span>' + baseDuration;
        }
        var totalNum = parseInt(baseCount, 10);
        card.classList.toggle("empty", !isNaN(totalNum) && totalNum === 0);
        card.setAttribute("title", (card.querySelector(".vs-mini-name") || {}).textContent + " · 共 " + (baseCount || "0") + " 条");
      });
      return;
    }
    var byStatus = {};
    rows.forEach(function (row) { byStatus[row.status] = row; });
    document.querySelectorAll(".home-vs-mini-card").forEach(function (card) {
      var status = card.getAttribute("data-vs-status") || "";
      var row = byStatus[status];
      if (!row) { return; }
      var total = Number(row.count || 0);
      var count = card.querySelector(".vs-mini-count");
      var breakdown = card.querySelector(".vs-mini-breakdown") || card.querySelector(".vs-mini-meta");
      var dur = card.querySelector(".vs-mini-dur") || card.querySelector(".vs-mini-duration-row .val");
      if (count) { count.textContent = String(total); }
      if (breakdown) {
        var dCount = typeof row.demandCount === "number" ? row.demandCount : total;
        var sCount = typeof row.storyCount === "number" ? row.storyCount : 0;
        breakdown.textContent = "业" + dCount + " · 研" + sCount;
      }
      if (dur) {
        var durDays = Number(row.avgDurationDays || 0);
        dur.innerHTML = durDays > 0 ? ('<span class="tag">均</span>' + durDays + "天") : "—";
      }
      card.classList.toggle("empty", total === 0);
      card.setAttribute("title", (card.querySelector(".vs-mini-name") || {}).textContent + " · 共 " + total + " 条");
    });
  }

  function updateTitle($, state, count, displayedCount, pageItemCount) {
    var $title = $("#top5Title");
    if ($title.length) {
      var stage = $(".home-vs-mini-card.active .vs-mini-name").first().text() || "全部";
      $title.text(stage === "全部" ? "全部需求" : stage + "阶段");
    }

    var $caption = $("#homeListCaption");
    if ($caption.length) {
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

  function fillUpdateTime($) {
    var now = new Date();
    var pad = function (n) { return n < 10 ? "0" + n : String(n); };
    $("#lastUpdateTime").text(
      now.getFullYear() + "-" + pad(now.getMonth() + 1) + "-" + pad(now.getDate()) + " " +
      pad(now.getHours()) + ":" + pad(now.getMinutes())
    );
  }

  function filterItems(items) {
    if (!items || !items.length) { return []; }
    return items.slice();
  }

  root.PoHomeRender = {
    renderRow: renderRow,
    renderValueStreamSummary: renderValueStreamSummary,
    updateTitle: updateTitle,
    fillUpdateTime: fillUpdateTime,
    filterItems: filterItems,
    isStoryItem: isStoryItem,
    getHomeZentaoStatusLabel: getHomeZentaoStatusLabel
  };
})(typeof window !== "undefined" ? window : this);
