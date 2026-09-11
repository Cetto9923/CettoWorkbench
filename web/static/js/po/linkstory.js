/*
 * 文件: web/static/js/po/linkstory.js
 * 模块: PO工作台
 * 职责: 关联研发需求弹窗开关、勾选同步，确认后回填提测 link-list。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poLinkstoryModal", "poLinkstoryOverlay"];
  var bound = false;
  var targetCtx = {
    unitId: "",
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
    setAllChecks(false);
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
      $(this).prop("checked", !!ids[id]);
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

  function openModal(opts) {
    opts = opts || {};
    targetCtx.unitId = String(opts.unitId || "");
    targetCtx.$list = opts.$list && opts.$list.length ? opts.$list : null;
    precheckFromList(targetCtx.$list);
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
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
    $("#poLinkstoryCloseBtn, #poLinkstoryOverlay").on("click", function () {
      closeModal();
    });
    $("#poLinkstoryBackBtn").on("click", function () {
      closeModal();
    });
    $("#poLinkstoryLinkBtn").on("click", function () {
      confirmAndClose();
    });
    $("#poLinkstorySearchBtn").on("click", function () {
      showToast("搜索（静态演示，未请求后端）", "info");
    });
    $("#poLinkstoryCheckAll, #poLinkstoryFooterCheck").on("change", function () {
      setAllChecks($(this).prop("checked"));
    });
    $root().on("change", '#poLinkstoryTable tbody input[name="storyId"]', function () {
      syncHeaderCheck();
    });
    $root().on("click", ".po-linkstory-title", function (e) {
      e.preventDefault();
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
