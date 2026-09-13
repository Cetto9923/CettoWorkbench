/*
 * 文件: web/static/js/po/linkstory.js
 * 模块: PO工作台
 * 职责: 关联研发需求弹窗：按版本拉取 HTML 片段、搜索（bySearch）、勾选同步；确认后调后端关联禅道并回填提测 link-list；支持解除关联。
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

  function showToast(message, level) { window.showToast(message, level || "info"); }

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
    $list.find("[data-story-id]").each(function () {
      var id = String($(this).attr("data-story-id") || "").trim();
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
    if (targetCtx.buildId) {
      $list.attr("data-tt-build-id", targetCtx.buildId);
    }
    $list.empty();
    items.forEach(function (item) {
      var id = String(item.id || "").trim();
      if (!id) {
        return;
      }
      var title = item.title || "";
      var label = "#" + id + (title ? " " + title : "");
      var $row = $('<div class="po-testtask-link-item">').attr({
        "data-story-id": id,
        "data-story-title": title
      });
      $('<span class="po-testtask-link-item-text">').text(label).appendTo($row);
      $('<button type="button" class="po-testtask-link-unlink">')
        .attr({ title: "解除关联", "aria-label": "解除关联 " + id })
        .html('<i class="fas fa-link-slash" aria-hidden="true"></i>')
        .appendTo($row);
      $list.append($row);
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

  function fieldItems(meta) {
    var items = ((meta && meta.fields) || [])
      .filter(function (f) {
        return f && f.key;
      })
      .map(function (f) {
        return { value: f.key, label: f.label || f.key };
      });
    return items.length
      ? items
      : [
          { value: "title", label: "需求名称" },
          { value: "status", label: "当前状态" }
        ];
  }

  function fieldLabel(items, value) {
    var key = String(value || "");
    for (var i = 0; i < items.length; i++) {
      if (String(items[i].value) === key) {
        return items[i].label || key;
      }
    }
    return key;
  }

  function syncFieldChange(group) {
    var $hidden = $("#poLinkstoryField" + group);
    if (!$hidden.length) {
      return;
    }
    var cur = String($hidden.val() || "");
    var prev = String($hidden.data("prevField") || "");
    if (cur === prev) {
      return;
    }
    $hidden.data("prevField", cur);
    if (!cur) {
      return;
    }
    applyDefaultOperator(group, true);
    $("#poLinkstoryValue" + group + "Init").val("");
    renderValueBox(group, false);
  }

  function initFieldAutocompletes() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    var items = fieldItems(parseSearchMeta());
    [1, 2].forEach(function (group) {
      var inputId = "poLinkstoryField" + group + "Input";
      var hiddenId = "poLinkstoryField" + group;
      var $hidden = $("#" + hiddenId);
      if (!$("#" + inputId).length || !$hidden.length) {
        return;
      }
      var val = String($hidden.val() || (group === 1 ? "title" : "status"));
      $hidden.val(val).data("prevField", val);
      window.initAutocomplete(inputId, hiddenId, items, {
        value: val,
        label: fieldLabel(items, val),
        placeholder: "搜索字段",
        labelOnly: true
      });
    });
  }

  function renderValueBox(group, keepValue) {
    var meta = parseSearchMeta();
    var def = fieldDef(meta, $("#poLinkstoryField" + group).val() || "");
    var $box = $("#poLinkstoryValueBox" + group);
    var prev = String($("#poLinkstoryValue" + group + "Init").val() || "");
    if (keepValue) {
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

  var escapeHtml = window.escapeHtml;
  var escapeAttr = escapeHtml;

  function initSearchControls() {
    if (!$("#poLinkstoryField1").length) {
      return;
    }
    initFieldAutocompletes();
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

    if (targetCtx.$list) {
      targetCtx.$list.attr("data-tt-build-id", targetCtx.buildId);
    }

    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
    loadFragment(fragmentUrl(targetCtx.buildId, 1, 100));
  }

  function postJSON(url, payload) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(url, {
      method: "POST",
      credentials: "same-origin",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-Requested-With": "XMLHttpRequest"
      },
      body: JSON.stringify(payload || {})
    }).then(function (res) {
      return res.text().then(function (text) {
        var data = {};
        try {
          data = text ? JSON.parse(text) : {};
        } catch (ignore) {
          data = {};
        }
        return { ok: res.ok, data: data, status: res.status };
      });
    });
  }

  function setLinkBtnBusy($btn, busy) {
    if (!$btn || !$btn.length) {
      return;
    }
    if (busy) {
      $btn.data("busy", "1").prop("disabled", true);
    } else {
      $btn.removeData("busy").prop("disabled", false);
    }
  }

  function confirmAndClose() {
    var items = selectedStories();
    if (!items.length) {
      showToast("请先勾选要关联的研发需求", "info");
      return;
    }
    if (!targetCtx.buildId) {
      showToast("版本无效", "error");
      return;
    }
    var $btn = $("#poLinkstoryLinkBtn");
    if ($btn.data("busy") === "1" || loading) {
      return;
    }

    var storyIds = items
      .map(function (it) {
        return it.id;
      })
      .join(",");
    setLinkBtnBusy($btn, true);
    loading = true;

    postJSON("/builds/" + encodeURIComponent(targetCtx.buildId) + "/linkstories", {
      stories: storyIds
    })
      .then(function (res) {
        if (!res.ok || !(res.data && res.data.success)) {
          var msg = String((res.data && res.data.message) || "").trim();
          throw new Error(msg || "关联需求失败");
        }
        applyToLinkList(items);
        closeModal();
        showToast(
          String((res.data && res.data.message) || "").trim() ||
            "已关联 " + items.length + " 条研发需求",
          "success"
        );
      })
      .catch(function (err) {
        showToast((err && err.message) || "关联需求失败", "error");
      })
      .then(function () {
        loading = false;
        setLinkBtnBusy($btn, false);
      });
  }

  function unlinkStory($btn) {
    var $row = $btn.closest(".po-testtask-link-item");
    var $list = $btn.closest('[data-tt-fill="link-list"]');
    var storyId = String($row.attr("data-story-id") || "").trim();
    var buildId = String($list.attr("data-tt-build-id") || "").trim();
    if (!storyId) {
      showToast("需求无效", "error");
      return;
    }
    if (!/^\d+$/.test(buildId) || buildId === "0") {
      showToast("版本无效", "error");
      return;
    }
    if ($btn.data("busy") === "1" || loading) {
      return;
    }

    setLinkBtnBusy($btn, true);
    postJSON("/builds/" + encodeURIComponent(buildId) + "/unlinkstories", {
      stories: storyId
    })
      .then(function (res) {
        if (!res.ok || !(res.data && res.data.success)) {
          var msg = String((res.data && res.data.message) || "").trim();
          throw new Error(msg || "解除关联失败");
        }
        $row.remove();
        showToast(
          String((res.data && res.data.message) || "").trim() || "解除关联成功",
          "success"
        );
      })
      .catch(function (err) {
        showToast((err && err.message) || "解除关联失败", "error");
        setLinkBtnBusy($btn, false);
      });
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
    $(document).on("click", ".po-testtask-link-unlink", function () {
      unlinkStory($(this));
    });
    $(document).on("click", "#poLinkstorySearchBtn", function () {
      runSearch();
    });
    $(document).on(
      "click",
      '.ui-autocomplete-dropdown[data-autocomplete-for="poLinkstoryField1Input"] .ui-autocomplete-option,'.concat(
        '.ui-autocomplete-dropdown[data-autocomplete-for="poLinkstoryField2Input"] .ui-autocomplete-option'
      ),
      function () {
        var forId = $(this).closest("[data-autocomplete-for]").attr("data-autocomplete-for") || "";
        var group = forId.indexOf("Field1") >= 0 ? 1 : 2;
        setTimeout(function () {
          syncFieldChange(group);
        }, 0);
      }
    );
    $(document).on("keydown", "#poLinkstoryField1Input, #poLinkstoryField2Input", function (e) {
      if (e.key !== "Enter") {
        return;
      }
      var group = this.id.indexOf("Field1") >= 0 ? 1 : 2;
      setTimeout(function () {
        syncFieldChange(group);
      }, 0);
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
