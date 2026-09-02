(function ($) {
  "use strict";

  var DEFAULT_VISIBLE = 5;

  function demandsUrl(status) {
    return "/demands?status=" + encodeURIComponent(status || "all");
  }

  function escapeHtml(text) {
    return $("<div>").text(text).html();
  }

  function buildActionCell(item) {
    var url = (item.zentaoUrl || "").trim();
    if (!url) {
      return "<td class=\"col-action\"><button type=\"button\" class=\"table-row-btn\" disabled>处理</button></td>";
    }
    return (
      "<td class=\"col-action\">" +
      "<a href=\"" + escapeHtml(url) + "\" class=\"table-row-btn\" target=\"_blank\">处理</a>" +
      "</td>"
    );
  }

  function buildRowHtml(item) {
    var rowClass = item.alert ? "js-top5-row top5-row--alert" : "js-top5-row";
    return (
      "<tr class=\"" + rowClass + "\">" +
      "<td class=\"table-id col-id\">" + escapeHtml(item.id || "") + "</td>" +
      "<td class=\"text-strong cell-ellipsis\"><span class=\"priority-tag " + escapeHtml(item.pri || "") + "\">" +
      escapeHtml(item.pri || "") + "</span>" + escapeHtml(item.title || "") + "</td>" +
      "<td><span class=\"table-tag\">" + escapeHtml(item.valueStream || item.stage || "") + "</span></td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.blocker || "") + "</td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.next || "") + "</td>" +
      "<td>" + escapeHtml(item.owner || "") + "</td>" +
      buildActionCell(item) +
      "</tr>"
    );
  }

  function renderTop5Rows(items) {
    var $tbody = $("#top5TableBody");
    if (!$tbody.length) {
      return $();
    }

    var list = Array.isArray(items) ? items : [];
    var html = $.map(list, buildRowHtml).join("");
    $tbody.html(html);
    return $tbody.find(".js-top5-row");
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

  function updateSectionTitle($card, count) {
    var $title = $("#top5Title");
    if (!$title.length) {
      return;
    }
    var label = $card.find(".vs-mini-name").first().text() || "全部";
    var iconHtml = $title.find("i").first().prop("outerHTML") || "<i class=\"fas fa-star\"></i>";
    $title.html(iconHtml + " 当前应推进事项：今日必推 / " + escapeHtml(label) + " / " + count + " 条");
  }

  function setActiveCard($card) {
    $(".home-vs-mini-card").removeClass("active");
    $card.addClass("active");
  }

  function refreshDemandsTable(status, $card) {
    return loadDemands(status).then(function (items) {
      updateSectionTitle($card, items.length);
      initTop5Toggle(renderTop5Rows(items));
      return items;
    });
  }

  function initTop5Toggle($rows) {
    var $wrap = $("#top5ToggleWrap");
    var $btn = $("#top5Toggle");
    if (!$wrap.length || !$btn.length || !$rows.length) {
      if ($wrap.length) {
        $wrap.prop("hidden", true);
      }
      return;
    }

    if ($rows.length <= DEFAULT_VISIBLE) {
      $wrap.prop("hidden", true);
      return;
    }

    $wrap.prop("hidden", false);
    var expanded = false;

    function apply() {
      $rows.each(function (index) {
        $(this).toggleClass("is-row-hidden", !expanded && index >= DEFAULT_VISIBLE);
      });
      $btn
        .attr("aria-expanded", expanded ? "true" : "false")
        .text(expanded ? "收起 ↑" : "查看全部 " + $rows.length + " 条 →");
    }

    $btn.off("click.top5toggle").on("click.top5toggle", function () {
      expanded = !expanded;
      apply();
    });

    apply();
  }

  function initValueStreamLinkage() {
    var $cards = $(".home-vs-mini-card");
    if (!$cards.length) {
      return;
    }

    $cards.on("click", function () {
      var $card = $(this);
      var status = $card.attr("data-vs-status");
      if (!status) {
        return;
      }
      setActiveCard($card);
      refreshDemandsTable(status, $card);
    });
  }

  $(function () {
    initValueStreamLinkage();

    var $active = $(".home-vs-mini-card.active").first();
    if (!$active.length) {
      $active = $(".home-vs-mini-card").first();
      if ($active.length) {
        setActiveCard($active);
      }
    }

    var initialStatus = $active.attr("data-vs-status") || "all";
    refreshDemandsTable(initialStatus, $active);
  });
})(jQuery);
