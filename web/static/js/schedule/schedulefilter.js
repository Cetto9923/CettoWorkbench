(function ($) {
  "use strict";

  var $root = $("#page-schedule");
  if (!$root.length) {
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

  function closeAllScheduleMultiselects(except) {
    $root.find(".schedule-multiselect.open").each(function () {
      if (except && except[0] === this) {
        return;
      }
      var $ms = $(this);
      $ms.removeClass("open");
      $ms.find(".schedule-multiselect-panel").prop("hidden", true);
      $ms.find(".schedule-multiselect-trigger").attr("aria-expanded", "false");
    });
  }

  function getSelectedValues($multiselect) {
    var selected = [];
    $multiselect.find(".schedule-multiselect-checkbox:checked").each(function () {
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
    $multiselect.find(".schedule-multiselect-text").text(formatMultiselectDisplay($multiselect, selected));
  }

  function collectAdvancedFilterValues() {
    var values = {
      groups: "",
      products: "",
      stages: "",
    };
    $root.find(".schedule-multiselect").each(function () {
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
      bizPage: null,
      indepPage: null,
    });
  }

  $root.find(".schedule-multiselect").each(function () {
    syncMultiselectTrigger($(this));
  });

  $root.on("click", ".schedule-multiselect-trigger", function (e) {
    e.stopPropagation();
    var $ms = $(this).closest(".schedule-multiselect");
    var isOpen = $ms.hasClass("open");
    closeAllScheduleMultiselects(isOpen ? $ms : null);
    if (isOpen) {
      $ms.removeClass("open");
      $ms.find(".schedule-multiselect-panel").prop("hidden", true);
      $(this).attr("aria-expanded", "false");
      return;
    }
    $ms.addClass("open");
    $ms.find(".schedule-multiselect-panel").prop("hidden", false);
    $(this).attr("aria-expanded", "true");
  });

  $root.on("click", ".schedule-multiselect-panel", function (e) {
    e.stopPropagation();
  });

  $root.on("change", ".schedule-multiselect-checkbox", function () {
    syncMultiselectTrigger($(this).closest(".schedule-multiselect"));
  });

  $root.on("click", ".schedule-multiselect-apply", function (e) {
    e.preventDefault();
    e.stopPropagation();
    closeAllScheduleMultiselects();
    applyAdvancedFilters();
  });

  $(document).on("click", function () {
    closeAllScheduleMultiselects();
  });

  window.scheduleApplyAdvancedFilters = applyAdvancedFilters;
  window.scheduleCollectAdvancedFilterValues = collectAdvancedFilterValues;
})(jQuery);
