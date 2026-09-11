/*
 * 文件: web/static/js/po/linkstory.js
 * 模块: PO工作台
 * 职责: 关联研发需求弹窗：按版本拉取 HTML 片段、搜索（bySearch）、勾选同步，确认后回填提测 link-list。
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

  function parseSearchMeta() {
    var raw = ($("#poLinkstorySearchMeta").val() || "").trim();
    if (!raw) {
      return { fields: [], options: {} };
    }
    try {
      return JSON.parse(raw);
    } catch (e) {
      return { fields: [], options: {} };
    }
  }

  function fieldDef(meta, key) {
    var fields = (meta && meta.fields) || [];
    for (var i = 0; i < fields.length; i++) {
      if (fields[i].key === key) {
        return fields[i];
      }
    }
    return { key: key, control: "input", operator: "=", optionsKey: "" };
  }

  function renderValueBox(group, keepValue) {
    var meta = parseSearchMeta();
    var fieldKey = $("#poLinkstoryField" + group).val() || "";
    var def = fieldDef(meta, fieldKey);
    var $box = $("#poLinkstoryValueBox" + group);
    var prev = keepValue
      ? String($box.find("[name='value" + group + "']").val() || $("#poLinkstoryValue" + group + "Init").val() || "")
      : String($("#poLinkstoryValue" + group + "Init").val() || "");
    if (!keepValue) {
      prev = String($("#poLinkstoryValue" + group + "Init").val() || "");
    } else {
      var live = $box.find("[name='value" + group + "']").val();
      if (typeof live !== "undefined") {
        prev = String(live || "");
      }
    }

    var control = def.control || "input";
    var html;
    if (control === "select") {
      var opts = ((meta.options || {})[def.optionsKey] || []).slice();
      html = '<select id="poLinkstoryValue' + group + '" name="value' + group + '" class="select" aria-label="第' + group + '组取值">';
      if (!opts.length) {
        html += '<option value=""></option>';
      }
      opts.forEach(function (opt) {
        var selected = String(opt.value) === String(prev) ? " selected" : "";
        html +=
          '<option value="' +
          escapeAttr(opt.value) +
          '"' +
          selected +
          ">" +
          escapeHtml(opt.label) +
          "</option>";
      });
      html += "</select>";
    } else if (control === "date") {
      html =
        '<input id="poLinkstoryValue' +
        group +
        '" name="value' +
        group +
        '" type="date" class="search-input" aria-label="第' +
        group +
        '组取值" autocomplete="off" value="' +
        escapeAttr(prev) +
        '" />';
    } else {
      html =
        '<input id="poLinkstoryValue' +
        group +
        '" name="value' +
        group +
        '" type="text" class="search-input" aria-label="第' +
        group +
        '组取值" autocomplete="off" value="' +
        escapeAttr(prev) +
        '" />';
    }
    $box.html(html);
  }

  function applyDefaultOperator(group, force) {
    var meta = parseSearchMeta();
    var fieldKey = $("#poLinkstoryField" + group).val() || "";
    var def = fieldDef(meta, fieldKey);
    var $op = $("#poLinkstoryOp" + group);
    if (force || !$op.val()) {
      $op.val(def.operator || "=");
    }
  }

  function escapeAttr(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/"/g, "&quot;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function escapeHtml(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function initSearchControls() {
    if (!$("#poLinkstoryField1").length) {
      return;
    }
    applyDefaultOperator(1, false);
    applyDefaultOperator(2, false);
    renderValueBox(1, false);
    renderValueBox(2, false);
  }

  function collectSearchParams() {
    return {
      browseType: "bySearch",
      field1: $("#poLinkstoryField1").val() || "title",
      operator1: $("#poLinkstoryOp1").val() || "include",
      value1: ($("#poLinkstoryValue1").val() || "").trim(),
      andOr: $("#poLinkstoryAndOr").val() || "and",
      field2: $("#poLinkstoryField2").val() || "status",
      operator2: $("#poLinkstoryOp2").val() || "=",
      value2: ($("#poLinkstoryValue2").val() || "").trim()
    };
  }

  function fragmentUrl(buildId, page, pageSize, searchParams) {
    var url = "/builds/" + encodeURIComponent(buildId) + "/linkstory";
    var q = [];
    if (page) {
      q.push("page=" + encodeURIComponent(page));
    }
    if (pageSize) {
      q.push("pageSize=" + encodeURIComponent(pageSize));
    }
    if (searchParams) {
      Object.keys(searchParams).forEach(function (k) {
        q.push(encodeURIComponent(k) + "=" + encodeURIComponent(searchParams[k]));
      });
    }
    return q.length ? url + "?" + q.join("&") : url;
  }

  function currentPageSize() {
    var size = $root().find(".po-linkstory-size-form select[name='pageSize']").val();
    return size || "100";
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
          return res
            .json()
            .catch(function () {
              return {};
            })
            .then(function (data) {
              var msg = String((data && data.message) || "").trim();
              throw new Error(msg || "加载关联需求失败");
            });
        }
        return res.text();
      })
      .then(function (html) {
        $body.html(html);
        initSearchControls();
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

  function runSearch() {
    if (!targetCtx.buildId) {
      return;
    }
    loadFragment(fragmentUrl(targetCtx.buildId, 1, currentPageSize(), collectSearchParams()));
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
      runSearch();
    });
    $(document).on("change", "#poLinkstoryField1, #poLinkstoryField2", function () {
      var group = $(this).data("group");
      applyDefaultOperator(group, true);
      $("#poLinkstoryValue" + group + "Init").val("");
      renderValueBox(group, false);
    });
    $(document).on("change", "#poLinkstoryCheckAll, #poLinkstoryFooterCheck", function () {
      setAllChecks($(this).prop("checked"));
    });
    $(document).on("change", '#poLinkstoryTable tbody input[name="storyId"]', function () {
      syncHeaderCheck();
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
      var suffix = String($root().attr("data-query-suffix") || "");
      var searchParams = null;
      if (suffix.indexOf("browseType=bySearch") !== -1) {
        searchParams = collectSearchParams();
      }
      loadFragment(fragmentUrl(targetCtx.buildId, 1, size, searchParams));
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
    initSearchControls();
  }

  window.openPoLinkstoryModal = openModal;
  window.closePoLinkstoryModal = closeModal;

  $(init);
})(jQuery);
