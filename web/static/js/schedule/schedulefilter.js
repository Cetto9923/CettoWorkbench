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
      $ms.find(".schedule-ms-search-input").val("");
      $ms.find(".schedule-ms-search-clear").prop("hidden", true);
      $ms.find(".schedule-ms-option").show();
      $ms.find(".schedule-ms-no-match").prop("hidden", true);
    });
  }

  function filterMultiselectOptions($ms) {
    var $input = $ms.find(".schedule-ms-search-input");
    if (!$input.length) {
      return;
    }
    var query = $.trim($input.val() || "").toLowerCase();
    var $options = $ms.find(".schedule-ms-option");
    var $clearBtn = $ms.find(".schedule-ms-search-clear");
    $clearBtn.prop("hidden", !query);

    var visibleCount = 0;
    $options.each(function () {
      var $opt = $(this);
      var label = String($opt.attr("data-label") || $opt.text() || "").toLowerCase();
      var match = !query || label.indexOf(query) !== -1;
      $opt.toggle(match);
      if (match) {
        visibleCount++;
      }
    });
    $ms.find(".schedule-ms-no-match").prop("hidden", visibleCount > 0 || $options.length === 0);
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
      windowType: $.trim($("#scheduleFilterWindowType").val() || ""),
      dev: $.trim($("#scheduleFilterDev").val() || ""),
      test: $.trim($("#scheduleFilterTest").val() || ""),
      accept: $.trim($("#scheduleFilterAccept").val() || ""),
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
      windowType: values.windowType || null,
      dev: values.dev || null,
      test: values.test || null,
      accept: values.accept || null,
      bizPage: null,
      indepPage: null,
    });
  }

  function hasMoreFiltersActive() {
    var values = collectAdvancedFilterValues();
    return !!(values.pri || values.windowType || values.dev || values.test || values.accept);
  }

  function setMoreFiltersOpen(open) {
    var $more = $("#scheduleMoreFilters");
    var $toggle = $("#scheduleMoreFilterToggle");
    if (!$more.length) {
      return;
    }
    $more.toggleClass("open", open).prop("hidden", !open);
    $toggle.toggleClass("active", open).attr("aria-expanded", open ? "true" : "false");
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
    closeAllScheduleMultiselects();
    if (isOpen) {
      return;
    }
    $ms.addClass("open");
    $ms.find(".schedule-ms-panel").prop("hidden", false);
    $(this).attr("aria-expanded", "true");
    var $searchInput = $ms.find(".schedule-ms-search-input");
    if ($searchInput.length) {
      setTimeout(function () {
        $searchInput.trigger("focus");
      }, 50);
    }
  });

  $row.on("click", ".schedule-ms-panel", function (e) {
    e.stopPropagation();
  });

  $row.on("input", ".schedule-ms-search-input", function () {
    filterMultiselectOptions($(this).closest(".schedule-ms"));
  });

  $row.on("click", ".schedule-ms-search-clear", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $ms = $(this).closest(".schedule-ms");
    $ms.find(".schedule-ms-search-input").val("").trigger("focus");
    filterMultiselectOptions($ms);
  });

  $row.on("keydown", ".schedule-ms-search-input", function (e) {
    if (e.key === "Escape") {
      e.preventDefault();
      closeAllScheduleMultiselects();
    } else if (e.key === "Enter") {
      e.preventDefault();
    }
  });

  $row.on("change", ".schedule-ms-checkbox", function () {
    syncMultiselectTrigger($(this).closest(".schedule-ms"));
  });

  $row.on("click", "#scheduleApplyFilters", function (e) {
    e.preventDefault();
    closeAllScheduleMultiselects();
    applyAdvancedFilters();
  });

  $row.on("click", "#scheduleMoreFilterToggle", function (e) {
    e.preventDefault();
    e.stopPropagation();
    closeAllScheduleMultiselects();
    toggleMoreFilters();
  });

  $("#scheduleSearch").on("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      applyAdvancedFilters();
    }
  });

  $("#scheduleMoreFilters").on("change", ".form-select", function () {
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
