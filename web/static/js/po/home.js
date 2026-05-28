(function ($) {
  "use strict";

  var DEFAULT_VISIBLE = 5;
  var DEMANDS_URL = "/po/demands?status=all";

  function escapeHtml(text) {
    return $("<div>").text(text).html();
  }

  function buildRowHtml(item) {
    var rowClass = item.alert ? "js-top5-row top5-row--alert" : "js-top5-row";
    return (
      "<tr class=\"" + rowClass + "\">" +
      "<td class=\"table-id col-id\">" + escapeHtml(item.id || "") + "</td>" +
      "<td class=\"text-strong cell-ellipsis\"><span class=\"priority-tag " + escapeHtml(item.pri || "") + "\">" +
      escapeHtml(item.pri || "") + "</span>" + escapeHtml(item.title || "") + "</td>" +
      "<td><span class=\"table-tag\">" + escapeHtml(item.stage || "") + "</span></td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.blocker || "") + "</td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.next || "") + "</td>" +
      "<td>" + escapeHtml(item.owner || "") + "</td>" +
      "<td class=\"col-action\"><button type=\"button\" class=\"table-row-btn\">处理</button></td>" +
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

  function loadTop5Items() {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(DEMANDS_URL, { method: "GET" })
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
          window.showToast("加载推进事项失败，请稍后重试", "danger");
        }
        return [];
      });
  }

  function initTop5Toggle($rows) {
    var $wrap = $("#top5ToggleWrap");
    var $btn = $("#top5Toggle");
    if (!$wrap.length || !$btn.length || !$rows.length) {
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

    $btn.on("click", function () {
      expanded = !expanded;
      apply();
    });

    apply();
  }

  $(function () {
    loadTop5Items().then(function (items) {
      initTop5Toggle(renderTop5Rows(items));
    });
  });
})(jQuery);
