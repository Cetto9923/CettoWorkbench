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

  function isScheduleWindowActionDisabled($el) {
    return $el.prop("disabled") || $el.hasClass("is-disabled") || $el.hasClass("action-btn--disabled");
  }

  function setScopeChip($chip) {
    $root.find(".schedule-scope-chip").removeClass("active");
    $chip.addClass("active");
  }

  function setDataTab($tab) {
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
      return;
    }

    $("#scheduleBizListPanel").show();
    $("#scheduleIndependentListPanel").hide();
    $("#scheduleBizPagination").show();
    $("#scheduleIndependentPagination").hide();
  }

  function toggleMoreFilters() {
    $("#scheduleMoreFilters").toggleClass("open");
  }

  function clearFilters() {
    $root.find(".schedule-scope-chip").removeClass("active");
    $root.find('.schedule-scope-chip[data-scope="notClosed"]').addClass("active");
    $("#scheduleMoreFilters").removeClass("open");
    $("#scheduleFilterAgile, #scheduleFilterSystem, #scheduleFilterStatus, #scheduleFilterAnomaly").val("");
    $("#scheduleSearch, #scheduleCreatorFilter, #scheduleSourceFilter, #scheduleDeptFilter").val("");
    $("#scheduleFilterPri, #scheduleFilterWindow, #scheduleFilterDev, #scheduleFilterTest, #scheduleFilterAccept").val("");
    $("#scheduleFilterRole").val("all");
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
    setScopeChip($(this));
  });

  $root.on("click", ".schedule-data-tab", function () {
    setDataTab($(this));
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

  $("#scheduleMoreFiltersBtn").on("click", toggleMoreFilters);
  $("#scheduleClearFilters").on("click", clearFilters);

  var params = new URLSearchParams(window.location.search);
  if (params.get("tab") === "indep") {
    var $indepTab = $root.find('.schedule-data-tab[data-type="independentRD"]');
    if ($indepTab.length) {
      setDataTab($indepTab);
    }
  }
})(jQuery);
