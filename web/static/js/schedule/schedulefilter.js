(function ($) {
  "use strict";

  var $root = $("#page-schedule");
  if (!$root.length) {
    return;
  }

  var $row = $("#scheduleMultiselectRow");
  if (!$row.length) {
    return;
  }

  var ownerPickerItems = null;
  var ownerPickerLoading = null;
  var ownerPickersReady = false;
  var suppressOwnerChange = false;
  var priPickerReady = false;
  var suppressPriChange = false;

  var OWNER_PICKERS = [
    { inputId: "scheduleFilterTestInput", hiddenId: "scheduleFilterTestValue", placeholder: "测试负责人" },
    { inputId: "scheduleFilterAcceptInput", hiddenId: "scheduleFilterAcceptValue", placeholder: "验收负责人" },
  ];

  var PRI_OPTIONS = [
    { value: "0", label: "未设置优先级" },
    { value: "1", label: "P1" },
    { value: "2", label: "P2" },
    { value: "3", label: "P3" },
    { value: "4", label: "P4" },
  ];

  function readURLParams() {
    return new URLSearchParams(window.location.search);
  }

  function buildScheduleURL(overrides) {
    var params = readURLParams();
    Object.keys(overrides || {}).forEach(function (key) {
      var value = overrides[key];
      if (value === null || value === undefined || value === "") {
        params.delete(key);
        return;
      }
      params.set(key, String(value));
    });
    var query = params.toString();
    return window.location.pathname + (query ? "?" + query : "");
  }

  function navigateSchedule(overrides) {
    window.location.href = buildScheduleURL(overrides);
  }

  function closeAllScheduleMultiselects() {
    $row.find(".schedule-ms.open").each(function () {
      var $ms = $(this);
      $ms.removeClass("open");
      $ms.find(".schedule-ms-panel").prop("hidden", true);
      $ms.find(".schedule-ms-trigger").attr("aria-expanded", "false");
    });
  }

  function closeAllFilterDropdowns() {
    closeAllScheduleMultiselects();
    if (typeof window.closeAllDropdowns === "function") {
      window.closeAllDropdowns();
    }
  }

  function getSelectedValues($multiselect) {
    var selected = [];
    $multiselect.find(".schedule-ms-checkbox:checked").each(function () {
      selected.push({
        value: String($(this).val()),
        label: String($(this).data("label") || $(this).val()),
      });
    });
    return selected;
  }

  function getFormMultiselectIds($ms) {
    var ids = [];
    $ms.find(".form-multiselect-checkbox:checked").each(function () {
      var value = String($(this).val() || "").trim();
      if (value) {
        ids.push(value);
      }
    });
    return ids;
  }

  function formatMultiselectDisplay($multiselect, selected) {
    var label = $multiselect.data("filter-label") || "筛选";
    if (!selected.length) {
      return label;
    }
    if (selected.length <= 2) {
      return label + ": " + selected.map(function (item) {
        return item.label;
      }).join(", ");
    }
    return label + ": 已选 " + selected.length + " 项";
  }

  function syncMultiselectTrigger($multiselect) {
    var selected = getSelectedValues($multiselect);
    $multiselect.find(".schedule-ms-text").text(formatMultiselectDisplay($multiselect, selected));
  }

  function collectAdvancedFilterValues() {
    var values = {
      groups: "",
      products: "",
      stages: "",
      windows: "",
      keyword: $.trim($("#scheduleSearch").val() || ""),
      pri: $.trim($("#scheduleFilterPri").val() || ""),
      test: $.trim($("#scheduleFilterTestValue").val() || ""),
      accept: $.trim($("#scheduleFilterAcceptValue").val() || ""),
    };
    $row.find(".schedule-ms").each(function () {
      var $ms = $(this);
      var key = $ms.data("filter-key");
      if (!key) {
        return;
      }
      var ids = getSelectedValues($ms).map(function (item) {
        return item.value;
      });
      values[key] = ids.join(",");
    });
    $row.find(".schedule-filter-ms").each(function () {
      var $ms = $(this);
      var key = $ms.data("filter-key");
      if (!key) {
        return;
      }
      values[key] = getFormMultiselectIds($ms).join(",");
    });
    return values;
  }

  function applyAdvancedFilters() {
    var values = collectAdvancedFilterValues();
    navigateSchedule({
      groups: values.groups || null,
      products: values.products || null,
      stages: values.stages || null,
      windows: values.windows || null,
      keyword: values.keyword || null,
      pri: values.pri || null,
      test: values.test || null,
      accept: values.accept || null,
      bizPage: null,
      indepPage: null,
    });
  }

  function hasMoreFiltersActive() {
    var values = collectAdvancedFilterValues();
    return !!(values.pri || values.test || values.accept);
  }

  function toAutocompleteItems(users) {
    return (users || [])
      .map(function (user) {
        return {
          value: $.trim(user.account || ""),
          label: $.trim(user.realname || "") || $.trim(user.account || ""),
        };
      })
      .filter(function (item) {
        return !!item.value;
      });
  }

  function findOptionLabel(items, value) {
    value = $.trim(value || "");
    if (!value) {
      return "";
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (items[i].value === value) {
        return items[i].label || value;
      }
    }
    return value;
  }

  function findOwnerLabel(items, account) {
    return findOptionLabel(items, account);
  }

  function insideUsersURL() {
    return $(".schedule-owner-picker").first().attr("data-inside-users-url") || "/users";
  }

  function initPriPicker() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    suppressPriChange = true;
    var selected = $.trim($("#scheduleFilterPri").attr("data-selected-value") || $("#scheduleFilterPri").val() || "");
    window.initAutocomplete("scheduleFilterPriInput", "scheduleFilterPri", PRI_OPTIONS, {
      placeholder: "优先级",
      value: selected,
      label: findOptionLabel(PRI_OPTIONS, selected),
      labelOnly: true,
    });
    priPickerReady = true;
    suppressPriChange = false;
  }

  function initOwnerPickers(items) {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    suppressOwnerChange = true;
    OWNER_PICKERS.forEach(function (picker) {
      var selected = $.trim($("#" + picker.hiddenId).attr("data-selected-value") || $("#" + picker.hiddenId).val() || "");
      window.initAutocomplete(picker.inputId, picker.hiddenId, items, {
        placeholder: picker.placeholder,
        value: selected,
        label: findOwnerLabel(items, selected),
      });
    });
    ownerPickersReady = true;
    suppressOwnerChange = false;
  }

  function loadOwnerPickers() {
    if (ownerPickerItems) {
      initOwnerPickers(ownerPickerItems);
      return $.Deferred().resolve(ownerPickerItems).promise();
    }
    if (ownerPickerLoading) {
      return ownerPickerLoading;
    }
    var req = window.appFetch
      ? window.appFetch(insideUsersURL(), { headers: { Accept: "application/json" } })
      : fetch(insideUsersURL(), { headers: { Accept: "application/json" } });
    ownerPickerLoading = Promise.resolve(req)
      .then(function (res) {
        return res.json();
      })
      .then(function (json) {
        var items = toAutocompleteItems((json && json.items) || []);
        ownerPickerItems = items;
        initOwnerPickers(items);
        return items;
      })
      .catch(function () {
        ownerPickerItems = [];
        initOwnerPickers([]);
        if (typeof window.showToast === "function") {
          window.showToast("加载负责人列表失败", "error");
        }
        return [];
      })
      .finally(function () {
        ownerPickerLoading = null;
      });
    return ownerPickerLoading;
  }

  function setMoreFiltersOpen(open) {
    var $more = $("#scheduleMoreFilters");
    var $toggle = $("#scheduleMoreFilterToggle");
    if (!$more.length) {
      return;
    }
    $more.toggleClass("open", open).prop("hidden", !open);
    $toggle.toggleClass("active", open).attr("aria-expanded", open ? "true" : "false");
    if (open) {
      initPriPicker();
      loadOwnerPickers();
    }
  }

  function toggleMoreFilters() {
    var $more = $("#scheduleMoreFilters");
    setMoreFiltersOpen(!$more.hasClass("open"));
  }

  $row.find(".schedule-ms").each(function () {
    syncMultiselectTrigger($(this));
  });

  $row.on("click", ".schedule-ms-trigger", function (e) {
    e.stopPropagation();
    var $ms = $(this).closest(".schedule-ms");
    var isOpen = $ms.hasClass("open");
    closeAllFilterDropdowns();
    if (isOpen) {
      return;
    }
    $ms.addClass("open");
    $ms.find(".schedule-ms-panel").prop("hidden", false);
    $(this).attr("aria-expanded", "true");
  });

  $row.on("click", ".schedule-ms-panel", function (e) {
    e.stopPropagation();
  });

  $row.on("change", ".schedule-ms-checkbox", function () {
    syncMultiselectTrigger($(this).closest(".schedule-ms"));
  });

  $row.on("focusin", ".schedule-filter-ms .form-multiselect-input", function () {
    closeAllScheduleMultiselects();
  });

  $row.on("click", "#scheduleApplyFilters", function (e) {
    e.preventDefault();
    closeAllFilterDropdowns();
    applyAdvancedFilters();
  });

  $row.on("click", "#scheduleMoreFilterToggle", function (e) {
    e.preventDefault();
    e.stopPropagation();
    closeAllFilterDropdowns();
    toggleMoreFilters();
  });

  $("#scheduleSearch").on("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      applyAdvancedFilters();
    }
  });

  $("#scheduleMoreFilters").on("change", ".schedule-pri-filter-value", function () {
    if (suppressPriChange || !priPickerReady) {
      return;
    }
    applyAdvancedFilters();
  });

  $("#scheduleMoreFilters").on("change", ".schedule-owner-filter-value", function () {
    if (suppressOwnerChange || !ownerPickersReady) {
      return;
    }
    applyAdvancedFilters();
  });

  // D8: 版本窗口概览卡点击联动筛选
  $(document).on("click", ".schedule-version-card", function (e) {
    if ($(e.target).closest(".schedule-version-card-actions, a, button").length) {
      return;
    }
    e.preventDefault();
    var windowId = String($(this).data("window-id") || "").trim();
    if (!windowId) {
      return;
    }
    var currentParams = readURLParams();
    var currentWindows = (currentParams.get("windows") || "").split(",").filter(Boolean);
    var newWindows;
    if (currentWindows.indexOf(windowId) !== -1) {
      newWindows = currentWindows.filter(function (id) { return id !== windowId; });
    } else {
      newWindows = [windowId];
    }
    navigateSchedule({
      windows: newWindows.length ? newWindows.join(",") : null,
      bizPage: null,
      indepPage: null,
    });
  });

  setMoreFiltersOpen(hasMoreFiltersActive());

  function esc(s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  // 活跃筛选标签渲染与移除
  function renderActiveFilterTags() {
    var $bar = $("#scheduleActiveFilterBar"), $container = $("#scheduleActiveFilterTags");
    if (!$bar.length || !$container.length) { return; }
    var tags = [], params = readURLParams();

    [{ k: "groups", p: "小组" }, { k: "products", p: "系统" }, { k: "windows", p: "窗口" }].forEach(function (m) {
      (params.get(m.k) || "").split(",").filter(Boolean).forEach(function (id) {
        var $cb = $row.find('.schedule-filter-ms[data-filter-key="' + m.k + '"] .form-multiselect-checkbox[value="' + id + '"]');
        var name = $cb.length ? $.trim($cb.closest("label").find(".form-multiselect-name").text()) : (m.p + "#" + id);
        tags.push({ key: m.k, value: id, label: m.p + ": " + name });
      });
    });

    var STAGE_NAME_MAP = { unscheduled: "未排期", wait_confirm: "待确定", planned: "已排期", changing: "变更中", closed: "已关闭" };
    (params.get("stages") || "").split(",").filter(Boolean).forEach(function (val) {
      var $cb = $row.find('.schedule-ms[data-filter-key="stages"] .schedule-ms-checkbox[value="' + val + '"]');
      var name = ($cb.length && $.trim($cb.data("label") || $cb.closest("label").text())) || STAGE_NAME_MAP[val] || val;
      tags.push({ key: "stages", value: val, label: "阶段: " + name });
    });

    var kw = $.trim(params.get("keyword") || "");
    if (kw) { tags.push({ key: "keyword", value: kw, label: '关键词: "' + kw + '"' }); }
    var pri = $.trim(params.get("pri") || "");
    if (pri) { tags.push({ key: "pri", value: pri, label: "优先级: " + (findOptionLabel(PRI_OPTIONS, pri) || ("P" + pri)) }); }
    var testUser = $.trim(params.get("test") || "");
    if (testUser) { tags.push({ key: "test", value: testUser, label: "测试负责人: " + (findOwnerLabel(ownerPickerItems || [], testUser) || testUser) }); }
    var acceptUser = $.trim(params.get("accept") || "");
    if (acceptUser) { tags.push({ key: "accept", value: acceptUser, label: "验收负责人: " + (findOwnerLabel(ownerPickerItems || [], acceptUser) || acceptUser) }); }
    if (params.get("suspended") === "1") { tags.push({ key: "suspended", value: "1", label: "仅看挂起" }); }

    if (!tags.length) { $container.empty(); $bar.prop("hidden", true); return; }
    $container.html(tags.map(function (t) {
      return '<span class="schedule-filter-tag" data-filter-key="' + esc(t.key) + '" data-filter-val="' + esc(t.value) + '">' +
        '<span class="schedule-filter-tag-text">' + esc(t.label) + '</span>' +
        '<button type="button" class="schedule-filter-tag-del" aria-label="移除筛选">×</button>' +
      '</span>';
    }).join(""));
    $bar.prop("hidden", false);
  }

  $(document).on("click", ".schedule-filter-tag-del", function (e) {
    e.preventDefault();
    var $tag = $(this).closest(".schedule-filter-tag"), key = $tag.data("filter-key"), val = String($tag.data("filter-val")), currentParams = readURLParams();
    if (key === "groups" || key === "products" || key === "stages" || key === "windows") {
      var items = (currentParams.get(key) || "").split(",").filter(function (x) { return x && x !== val; });
      var ov = { bizPage: null, indepPage: null };
      ov[key] = items.length ? items.join(",") : null;
      navigateSchedule(ov);
      return;
    }
    var singleMap = { keyword: "#scheduleSearch", pri: "#scheduleFilterPri", test: "#scheduleFilterTestValue", accept: "#scheduleFilterAcceptValue" };
    if (singleMap[key]) { $(singleMap[key]).val(""); }
    var singleOv = { bizPage: null, indepPage: null };
    singleOv[key] = null;
    navigateSchedule(singleOv);
  });

  $(document).on("click", "#scheduleActiveFilterClearAll", function (e) {
    e.preventDefault();
    navigateSchedule({ groups: null, products: null, stages: null, windows: null, keyword: null, pri: null, test: null, accept: null, suspended: null, bizPage: null, indepPage: null });
  });

  // 点击外部与按 Esc 键关闭所有筛选下拉
  $(document).on("click", function (e) {
    if (!$(e.target).closest(".schedule-ms, .schedule-filter-ms, .dropdown").length) {
      closeAllFilterDropdowns();
    }
  });

  $(document).on("keydown", function (e) {
    if (e.key === "Escape" || e.keyCode === 27) {
      closeAllFilterDropdowns();
    }
  });

  renderActiveFilterTags();

  window.scheduleApplyAdvancedFilters = applyAdvancedFilters;
  window.scheduleCollectAdvancedFilterValues = collectAdvancedFilterValues;
  window.toggleScheduleMoreFilters = toggleMoreFilters;
  window.scheduleRenderActiveFilterTags = renderActiveFilterTags;
})(jQuery);
