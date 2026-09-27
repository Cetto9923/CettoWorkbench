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

  $(document).on("click", function () {
    closeAllScheduleMultiselects();
  });

  setMoreFiltersOpen(hasMoreFiltersActive());

  window.scheduleApplyAdvancedFilters = applyAdvancedFilters;
  window.scheduleCollectAdvancedFilterValues = collectAdvancedFilterValues;
  window.toggleScheduleMoreFilters = toggleMoreFilters;
})(jQuery);
