/*
 * 文件: web/static/js/po/linkstory.js
 * 模块: PO工作台
 * 职责: 关联研发需求弹窗：按版本拉取 HTML 片段、勾选同步，确认后回填提测 link-list。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poLinkstoryModal", "poLinkstoryOverlay"];
  var bound = false;
  var loading = false;
  var targetCtx = {
    unitId: "",
    buildId: "",
    $list: null
  };

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function $root() {
    return $("#poLinkstoryRoot");
  }

  function rowChecks() {
    return $root().find('#poLinkstoryTable tbody input[name="storyId"]');
  }

  function syncHeaderCheck() {
    var $rows = rowChecks();
    var total = $rows.length;
    var checked = $rows.filter(":checked").length;
    var $all = $("#poLinkstoryCheckAll");
    var $footer = $("#poLinkstoryFooterCheck");
    var allOn = total > 0 && checked === total;
    $all.prop("checked", allOn);
    $footer.prop("checked", allOn);
    $all.prop("indeterminate", checked > 0 && checked < total);
    syncLinkBtn(checked > 0);
  }

  function syncLinkBtn(visible) {
    var $btn = $("#poLinkstoryLinkBtn");
    if (!$btn.length) {
      return;
    }
    $btn.prop("hidden", !visible);
  }

  function setAllChecks(on) {
    rowChecks().prop("checked", !!on);
    $("#poLinkstoryCheckAll").prop("checked", !!on).prop("indeterminate", false);
    $("#poLinkstoryFooterCheck").prop("checked", !!on);
    syncLinkBtn(!!on && rowChecks().length > 0);
  }

  function selectedStories() {
    var items = [];
    rowChecks().filter(":checked").each(function () {
      var $tr = $(this).closest("tr");
      var id = String($(this).val() || "").trim();
      var title = $.trim($tr.find(".po-linkstory-title").text() || "");
      if (id) {
        items.push({ id: id, title: title });
      }
    });
    return items;
  }

  function precheckFromList($list) {
    if (!$list || !$list.length) {
      syncHeaderCheck();
      return;
    }
    var ids = {};
    $list.find('input[type="checkbox"]').each(function () {
      var id = String($(this).val() || "").trim();
      if (id) {
        ids[id] = true;
      }
    });
    rowChecks().each(function () {
      var id = String($(this).val() || "").trim();
      if (ids[id]) {
        $(this).prop("checked", true);
      }
    });
    syncHeaderCheck();
  }

  function applyToLinkList(items) {
    var $list = targetCtx.$list;
    if (!$list || !$list.length) {
      return;
    }
    $list.empty();
    items.forEach(function (item) {
      var label = "#" + item.id + (item.title ? " " + item.title : "");
      var $label = $("<label>");
      $("<input>")
        .attr({ type: "checkbox", name: "linkedStoryId", value: item.id, checked: true })
        .appendTo($label);
      $label.append(document.createTextNode(" " + label));
      $list.append($label);
    });
  }

  function closeModal() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function fragmentUrl(buildId, page, pageSize) {
    var url = "/builds/" + encodeURIComponent(buildId) + "/linkstory";
    var q = [];
    if (page) {
      q.push("page=" + encodeURIComponent(page));
    }
    if (pageSize) {
      q.push("pageSize=" + encodeURIComponent(pageSize));
    }
    return q.length ? url + "?" + q.join("&") : url;
  }

  function loadFragment(url) {
    if (loading) {
      return Promise.resolve(false);
    }
    loading = true;
    var $body = $("#poLinkstoryModalBody");
    $body.css("opacity", "0.55");
    return fetch(url, {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "text/html" }
    })
      .then(function (res) {
        if (!res.ok) {
          return res.json().catch(function () {
            return {};
          }).then(function (data) {
            var msg = String((data && data.message) || "").trim();
            throw new Error(msg || "加载关联需求失败");
          });
        }
        return res.text();
      })
      .then(function (html) {
        $body.html(html);
        precheckFromList(targetCtx.$list);
        return true;
      })
      .catch(function (err) {
        showToast((err && err.message) || "加载关联需求失败", "error");
        return false;
      })
      .then(function (ok) {
        loading = false;
        $body.css("opacity", "");
        return ok;
      });
  }

  function openModal(opts) {
    opts = opts || {};
    targetCtx.unitId = String(opts.unitId || "");
    targetCtx.buildId = String(opts.buildId || "").trim();
    targetCtx.$list = opts.$list && opts.$list.length ? opts.$list : null;

    if (!/^\d+$/.test(targetCtx.buildId) || targetCtx.buildId === "0") {
      showToast("请先选择或创建有效版本", "info");
      return;
    }

    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
    loadFragment(fragmentUrl(targetCtx.buildId, 1, 100));
  }

  function confirmAndClose() {
    var items = selectedStories();
    if (!items.length) {
      showToast("请先勾选要关联的研发需求", "info");
      return;
    }
    applyToLinkList(items);
    closeModal();
    showToast("已关联 " + items.length + " 条研发需求", "success");
  }

  function bindEvents() {
    $(document).on("click", "#poLinkstoryCloseBtn, #poLinkstoryOverlay", function () {
      closeModal();
    });
    $(document).on("click", "#poLinkstoryBackBtn", function () {
      closeModal();
    });
    $(document).on("click", "#poLinkstoryLinkBtn", function () {
      confirmAndClose();
    });
    $(document).on("click", "#poLinkstorySearchBtn", function () {
      showToast("搜索（尚未接入）", "info");
    });
    $(document).on("change", "#poLinkstoryCheckAll, #poLinkstoryFooterCheck", function () {
      setAllChecks($(this).prop("checked"));
    });
    $(document).on("change", '#poLinkstoryTable tbody input[name="storyId"]', function () {
      syncHeaderCheck();
    });
    $(document).on("click", ".po-linkstory-title", function (e) {
      e.preventDefault();
    });
    $(document).on("click", "#poLinkstoryRoot a.po-linkstory-page", function (e) {
      e.preventDefault();
      var href = $(this).attr("href");
      if (href) {
        loadFragment(href);
      }
    });
    $(document).on("change", "#poLinkstoryRoot .po-linkstory-size-form select[name='pageSize']", function () {
      var size = $(this).val() || "100";
      if (!targetCtx.buildId) {
        return;
      }
      loadFragment(fragmentUrl(targetCtx.buildId, 1, size));
    });
  }

  function init() {
    if (bound) {
      return;
    }
    if (!$("#poLinkstoryModal").length) {
      return;
    }
    bound = true;
    bindEvents();
  }

  window.openPoLinkstoryModal = openModal;
  window.closePoLinkstoryModal = closeModal;

  $(init);
})(jQuery);
