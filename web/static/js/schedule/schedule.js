(function ($) {
  "use strict";

  var $root = $("#page-schedule");
  if (!$root.length) {
    return;
  }

  var listTitles = {
    bizReq: "业务需求列表",
    independentRD: "独立研发需求列表",
  };

  var defaultFilter = "all_open";
  var bizOnlyFilters = ["manager_reviewing"];
  var indepTabDisabledTitles = {
    manager_reviewing: "无主管审批状态",
  };

  function currentDataTabType() {
    var $active = $root.find(".schedule-data-tab.active");
    return ($active.data("type") || "bizReq");
  }

  function isScheduleWindowActionDisabled($el) {
    return $el.prop("disabled") || $el.hasClass("is-disabled") || $el.hasClass("action-btn--disabled");
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

  function currentFilter() {
    return readURLParams().get("filter") || defaultFilter;
  }

  var bizStageValues = ["no_window", "no_story", "no_task", "task_unassigned", "task_assigned"];
  var indepStageValues = ["no_window", "no_task", "task_unassigned", "task_assigned"];

  function sanitizeStagesForTab(raw, type) {
    var allowed = type === "independentRD" ? indepStageValues : bizStageValues;
    if (!raw) {
      return "";
    }
    return String(raw)
      .split(",")
      .map(function (item) {
        return $.trim(item);
      })
      .filter(function (item) {
        return allowed.indexOf(item) >= 0;
      })
      .filter(function (item, index, list) {
        return list.indexOf(item) === index;
      })
      .join(",");
  }

  function isSuspendedActive() {
    return readURLParams().get("suspended") === "1";
  }

  function preserveSuspendedOverride(overrides) {
    if (isSuspendedActive()) {
      overrides.suspended = "1";
    }
    return overrides;
  }

  function isIndepTabBlockedByFilter(filter) {
    return bizOnlyFilters.indexOf(filter) >= 0;
  }

  function syncFilterTabUI() {
    var type = currentDataTabType();
    var filter = currentFilter();
    var $indepTab = $root.find('.schedule-data-tab[data-type="independentRD"]');
    var $bizOnlyControls = $root.find(".schedule-scope-chip--biz-only, #scheduleSuspendedToggle");

    if (type === "independentRD") {
      $bizOnlyControls.hide();
      $indepTab.removeClass("is-disabled").removeAttr("title");
    } else {
      $bizOnlyControls.show();
      if (isIndepTabBlockedByFilter(filter)) {
        $indepTab
          .addClass("is-disabled")
          .attr("title", indepTabDisabledTitles[filter] || "");
      } else {
        $indepTab.removeClass("is-disabled").removeAttr("title");
      }
    }

    updateFilterChipCounts();
  }

  function updateFilterChipCounts() {
    var type = currentDataTabType();
    var attrName = type === "independentRD" ? "indep-count" : "biz-count";
    $root.find(".schedule-scope-chip").each(function () {
      var $chip = $(this);
      var count = $chip.attr("data-" + attrName);
      if (count === undefined || count === null || count === "") {
        count = 0;
      }
      $chip.find(".js-filter-count").text(count);
    });
  }

  function setDataTab($tab, updateURL) {
    $root.find(".schedule-data-tab").removeClass("active");
    $tab.addClass("active");
    var type = $tab.data("type") || "bizReq";
    $("#scheduleListTitle").text(listTitles[type] || listTitles.bizReq);

    var $count = $("#scheduleListCountText");
    var total = type === "independentRD"
      ? ($count.data("independent-total") || 0)
      : ($count.data("biz-total") || 0);
    $count.text("（共 " + total + " 条）");

    if (type === "independentRD") {
      $("#scheduleBizListPanel").hide();
      $("#scheduleIndependentListPanel").show();
      $("#scheduleBizPagination").hide();
      $("#scheduleIndependentPagination").show();
    } else {
      $("#scheduleBizListPanel").show();
      $("#scheduleIndependentListPanel").hide();
      $("#scheduleBizPagination").show();
      $("#scheduleIndependentPagination").hide();
    }

    syncFilterTabUI();

    if (updateURL) {
      var params = readURLParams();
      var overrides = {
        tab: type === "independentRD" ? "indep" : null,
      };
      var stages = sanitizeStagesForTab(params.get("stages"), type);
      overrides.stages = stages || null;
      navigateSchedule(overrides);
    }
  }

  function clearFilters() {
    navigateSchedule({
      filter: defaultFilter,
      suspended: null,
      groups: null,
      products: null,
      stages: null,
      windows: null,
      keyword: null,
      pri: null,
      windowType: null,
      dev: null,
      test: null,
      accept: null,
      bizPage: null,
      indepPage: null,
    });
  }

  function toggleBizChildren(parentId, $toggle) {
    var $children = $root.find('tr.schedule-child-row[data-parent="' + parentId + '"]');
    var collapsed = $toggle.hasClass("is-collapsed");

    if (collapsed) {
      $toggle.removeClass("is-collapsed fa-chevron-right").addClass("fa-chevron-down");
      $children.removeClass("is-hidden");
      return;
    }

    $toggle.addClass("is-collapsed").removeClass("fa-chevron-down").addClass("fa-chevron-right");
    $children.addClass("is-hidden");
  }

  function closeAllScheduleWindowCardMenus() {
    $root.find(".version-card-dropdown.show").removeClass("show");
  }

  function toggleScheduleWindowCardMenu($actions) {
    var $dropdown = $actions.find(".version-card-dropdown");
    var isOpen = $dropdown.hasClass("show");
    closeAllScheduleWindowCardMenus();
    if (!isOpen) {
      $dropdown.addClass("show");
    }
  }

  window.closeAllScheduleWindowCardMenus = closeAllScheduleWindowCardMenus;

  $root.on("click", ".schedule-action-buttons .action-btn", function (e) {
    var label = $.trim($(this).text());
    if (label !== "去排期" && label !== "排期") {
      return;
    }
    e.preventDefault();
    if (typeof window.openScheduleIntegratedModal === "function") {
      window.openScheduleIntegratedModal($(this));
    }
  });

  $root.on("click", ".schedule-scope-chip", function () {
    var filter = $(this).data("filter") || defaultFilter;
    navigateSchedule(preserveSuspendedOverride({
      filter: filter,
      bizPage: null,
      indepPage: null,
    }));
  });

  $root.on("click", "#scheduleSuspendedToggle", function () {
    navigateSchedule({
      suspended: isSuspendedActive() ? null : "1",
      bizPage: null,
      indepPage: null,
    });
  });

  $root.on("click", ".schedule-data-tab", function () {
    var $tab = $(this);
    if ($tab.hasClass("is-disabled")) {
      return;
    }
    setDataTab($tab, true);
  });

  $root.on("click", ".schedule-row-expand", function (e) {
    e.stopPropagation();
    var $btn = $(this);
    var $icon = $btn.find("i");
    var parentId = $btn.data("target") || $btn.closest("tr").data("id");
    if (!parentId || !$icon.length) {
      return;
    }
    toggleBizChildren(parentId, $icon);
  });

  $root.on("click", ".js-open-create-version-window", function () {
    if (typeof window.openScheduleCreateVersionWindowModal === "function") {
      window.openScheduleCreateVersionWindowModal();
    }
  });

  $root.on("click", ".js-open-manage-version-windows", function () {
    if (typeof window.openManageVersionWindowsModal === "function") {
      window.openManageVersionWindowsModal();
    }
  });

  $root.on("click", ".js-schedule-reload", function () {
    window.location.reload();
  });

  $root.on("click", ".js-toggle-window-card-menu", function (e) {
    e.preventDefault();
    e.stopPropagation();
    toggleScheduleWindowCardMenu($(this).closest(".schedule-version-card-actions"));
  });

  $root.on("click", ".js-edit-version-window", function (e) {
    e.preventDefault();
    e.stopPropagation();
    if (isScheduleWindowActionDisabled($(this))) {
      return;
    }
    if (typeof window.openScheduleEditVersionWindowModal === "function") {
      window.openScheduleEditVersionWindowModal($(this).data("window-id"));
    }
  });

  $root.on("click", ".js-delete-version-window", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $btn = $(this);
    if (isScheduleWindowActionDisabled($btn)) {
      return;
    }
    if (typeof window.deleteScheduleVersionWindow === "function") {
      window.deleteScheduleVersionWindow($btn.data("window-id"), $btn.data("window-name"));
    }
  });

  $(document).on("click", ".js-manage-edit-version-window", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $btn = $(this);
    if (isScheduleWindowActionDisabled($btn)) {
      return;
    }
    window.closeShowModals(["manageVersionWindowsModal", "manageVersionWindowsOverlay"]);
    if (typeof window.openScheduleEditVersionWindowModal === "function") {
      window.openScheduleEditVersionWindowModal($btn.data("window-id"));
    }
  });

  $(document).on("click", ".js-manage-delete-version-window", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $btn = $(this);
    if (isScheduleWindowActionDisabled($btn)) {
      return;
    }
    if (typeof window.deleteScheduleVersionWindow === "function") {
      window.deleteScheduleVersionWindow($btn.data("window-id"), $btn.data("window-name"));
    }
  });

  $(document).on("click", function (e) {
    if ($(e.target).closest(".schedule-version-card-actions").length) {
      return;
    }
    closeAllScheduleWindowCardMenus();
  });

  $("#scheduleVersionWindowModalOverlay").on("click", function () {
    if (typeof window.closeScheduleVersionWindowModal === "function") {
      window.closeScheduleVersionWindowModal();
    }
  });
  $("#scheduleVersionWindowModalCloseBtn, #scheduleVersionWindowModalDismissBtn").on("click", function () {
    if (typeof window.closeScheduleVersionWindowModal === "function") {
      window.closeScheduleVersionWindowModal();
    }
  });
  $("#scheduleVersionWindowModalSaveBtn").on("click", function () {
    if (typeof window.saveScheduleVersionWindowModal === "function") {
      window.saveScheduleVersionWindowModal();
    }
  });

  $("#manageVersionWindowsOverlay").on("click", function () {
    window.closeShowModals(["manageVersionWindowsModal", "manageVersionWindowsOverlay"]);
  });
  $("#manageVersionWindowsCloseBtn, #manageVersionWindowsDismissBtn").on("click", function () {
    window.closeShowModals(["manageVersionWindowsModal", "manageVersionWindowsOverlay"]);
  });
  $(".js-open-create-version-window-from-manage").on("click", function () {
    window.closeShowModals(["manageVersionWindowsModal", "manageVersionWindowsOverlay"]);
    if (typeof window.openScheduleCreateVersionWindowModal === "function") {
      window.openScheduleCreateVersionWindowModal();
    }
  });

  $("#scheduleClearFilters").on("click", clearFilters);

  var params = readURLParams();
  if (params.get("tab") === "indep") {
    var $indepTab = $root.find('.schedule-data-tab[data-type="independentRD"]');
    if ($indepTab.length) {
      setDataTab($indepTab, false);
    }
  } else {
    syncFilterTabUI();
  }
})(jQuery);
