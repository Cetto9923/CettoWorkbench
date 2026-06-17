(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var rdApi = window.ScheduleIntegratedRd;
  var MODAL_IDS = ["scheduleIntegratedModal", "scheduleIntegratedModalOverlay"];

  function extractDemandID($btn) {
    var raw = $btn.data("demand-id");
    if (raw === undefined || raw === null || raw === "") {
      return 0;
    }
    var id = parseInt(String(raw), 10);
    return isNaN(id) || id <= 0 ? 0 : id;
  }

  function extractRowContext($btn) {
    var $row = $btn.closest("tr");
    var id = $.trim($row.find(".schedule-id-badge").first().text()) || "REQ-—";
    var $titleEl = $row.find(".schedule-title-link").first();
    var title = $.trim($titleEl.attr("title") || $titleEl.text()) || "—";
    var owner = $.trim($row.find(".schedule-owner-line .name").first().text()) || "待分配";
    var system = "—";

    if ($row.hasClass("schedule-indep-row") || $row.hasClass("schedule-indep-child-row")) {
      system = $.trim($row.find("td").eq(2).text()) || "—";
    }

    return {
      id: id,
      title: title,
      owner: owner,
      system: system,
    };
  }

  function initSchedulingOwnerPickers(users, data) {
    if (typeof window.initAutocomplete !== "function" || !shared) {
      return;
    }

    var items = shared.toAutocompleteItems(users);
    var placeholder = "输入姓名或工号搜索";

    window.initAutocomplete("scheduleIntRDInput", "scheduleIntRDValue", items, {
      placeholder: placeholder,
      maxShow: 50,
      value: data.rd,
      label: data.rdName,
    });
    window.initAutocomplete("scheduleIntQDInput", "scheduleIntQDValue", items, {
      placeholder: placeholder,
      maxShow: 50,
      value: data.qd,
      label: data.qdName,
    });
    window.initAutocomplete("scheduleIntAccepterInput", "scheduleIntAccepterValue", items, {
      placeholder: placeholder,
      maxShow: 50,
      value: data.accepter,
      label: data.accepterName,
    });
  }

  function resetSchedulingOwnerPickers() {
    if (typeof window.clearAutocomplete !== "function") {
      return;
    }

    window.clearAutocomplete("scheduleIntRDInput");
    window.clearAutocomplete("scheduleIntQDInput");
    window.clearAutocomplete("scheduleIntAccepterInput");
  }

  function fillWindowSelect(windows, selectedID, selectedName, fallbackReleaseDate) {
    var $select = $("#scheduleIntegratedWindowSelect");
    var selected = String(selectedID || "");
    var seen = {};

    $select.empty();
    $("<option></option>").val("").text("请选择版本窗口").appendTo($select);

    (windows || []).forEach(function (window) {
      var id = String(window.id || "");
      var name = $.trim(window.name || "");
      var releaseDate = $.trim(window.releaseDate || "");
      if (!id) {
        return;
      }
      seen[id] = true;
      $("<option></option>")
        .val(id)
        .text(name || id)
        .attr("data-release-date", releaseDate)
        .appendTo($select);
    });

    if (selected && selected !== "0" && !seen[selected]) {
      $("<option></option>")
        .val(selected)
        .text($.trim(selectedName || "") || selected)
        .attr("data-release-date", $.trim(fallbackReleaseDate || ""))
        .appendTo($select);
    }

    if (selected && selected !== "0") {
      $select.val(selected);
    } else {
      $select.val("");
    }
  }

  function syncPlanDateFromWindow() {
    var $select = $("#scheduleIntegratedWindowSelect");
    var $selected = $select.find("option:selected");
    var releaseDate = "";

    if ($select.val()) {
      releaseDate = $.trim($selected.attr("data-release-date") || "");
    }

    setDateInputValue($("#scheduleIntegratedSchedulePlanDate"), releaseDate);
    updateReleaseMeta();
  }

  function updateReleaseMeta() {
    var $select = $("#scheduleIntegratedWindowSelect");
    var windowLabel = "—";
    var planDate = "—";

    if ($select.val()) {
      windowLabel = $.trim($select.find("option:selected").text()) || "—";
      planDate = $.trim($("#scheduleIntegratedSchedulePlanDate").val()) || "—";
    }

    $("#scheduleIntegratedReleaseStrip").text("窗口 " + windowLabel + " ｜ " + planDate);
  }

  function setDateInputValue($input, value) {
    var date = $.trim(value || "");
    $input.val(date);
    if (shared) {
      shared.syncIntegratedDateInputState($input[0]);
    }
  }

  function syncAllIntegratedDateInputs() {
    $("#scheduleIntegratedModalBody input[type='date']").each(function () {
      if (shared) {
        shared.syncIntegratedDateInputState(this);
      }
    });
  }

  function fillStoryItems(stories) {
    var $tbody = $("#storyItemsBody");
    var $summary = $("#storyItemsSummary");
    var list = stories || [];
    var totalSp = 0;

    $tbody.empty();

    if (!list.length) {
      var emptyRow = shared ? shared.cloneTemplateElement("tplStoryItemEmpty", "tr") : null;
      if (emptyRow) {
        $tbody.append(emptyRow);
      }
      if ($summary.length) {
        $summary.text("只读，用于拼接研发需求");
      }
      return;
    }

    list.forEach(function (story, index) {
      var row = shared ? shared.cloneTemplateElement("tplStoryItemRow", "tr") : null;
      if (!row) {
        return;
      }
      var role = story.isMain ? "主系统故事" : "配合故事";
      var estimate = shared ? shared.formatStoryEstimate(story.estimate) : String(story.estimate || "0");
      var estimateNum = Number(estimate);
      if (!isNaN(estimateNum)) {
        totalSp += estimateNum;
      }

      row.querySelector(".story-item-index").textContent = String(index + 1);
      row.querySelector(".story-item-role").textContent = role;
      row.querySelector(".story-item-title").textContent = story.title || "—";
      row.querySelector(".story-item-product").textContent = story.productName || "—";
      row.querySelector(".story-item-estimate").textContent = estimate;
      $tbody.append(row);
    });

    if ($summary.length) {
      $summary.text(
        "只读，用于拼接研发需求 · 合计 " +
          (shared ? shared.formatStoryEstimate(totalSp) : totalSp) +
          " SP"
      );
    }
  }

  function resetStoryItems() {
    fillStoryItems([]);
  }

  function fillSchedulingDetail(data) {
    var mainSystem = $.trim(data.mainSystemName || "") || "—";
    var owner = $.trim(data.braName || "") || $.trim(data.bra || "") || "待分配";
    var involvedProducts = data.involvedProducts || [];
    var mainSystemId = data.mainSystemId || 0;
    var users = data.users || [];

    if (shared) {
      shared.schedulingUsers = users;
      shared.mainSystemId = mainSystemId;
      shared.involvedProducts = involvedProducts;
      shared.productProjectsMap = shared.buildProductProjectsMap(data.productProjects);
      shared.productExecutionsMap = shared.buildProductExecutionsMap(data.projectExecutions);
    }

    $("#scheduleIntegratedReqTitle").text(data.name || "—");
    $("#scheduleIntegratedReqSystem").text(mainSystem);
    $("#scheduleIntegratedReqOwner").text(owner);

    fillWindowSelect(data.windows, data.windowId, data.windowName, data.schedulePlanDate);
    syncPlanDateFromWindow();
    initSchedulingOwnerPickers(users, data);
    setDateInputValue($("#scheduleIntegratedDevelopFinish"), data.developFinish);
    setDateInputValue($("#scheduleIntegratedTestFinish"), data.testFinish);
    setDateInputValue($("#scheduleIntegratedAcceptancedDate"), data.acceptancedDate);

    var systemsHint = shared
      ? shared.formatInvolvedSystemsHint(involvedProducts, mainSystem)
      : mainSystem;
    $("#scheduleIntegratedSystemsHint").text("涉及系统：" + systemsHint);

    fillStoryItems(data.stories);
    if (rdApi) {
      rdApi.render(data.stories, users, involvedProducts, mainSystemId, data.productProjects);
    }
    syncAllIntegratedDateInputs();
  }

  function loadSchedulingDetail(demandID) {
    $.ajax({
      url: "/schedule/demands/" + demandID + "/scheduling",
      method: "GET",
      dataType: "json",
    })
      .done(function (resp) {
        if (!resp || !resp.success) {
          window.alert((resp && resp.error) || "加载业需详情失败");
          return;
        }
        fillSchedulingDetail(resp);
        if (resp.id) {
          $("#scheduleIntegratedModalTitle").text("排期一体化办理 · REQ-" + resp.id);
        }
      })
      .fail(function () {
        window.alert("加载业需详情失败，请稍后重试");
      });
  }

  function resetIntegratedForm() {
    var $body = $("#scheduleIntegratedModalBody");

    $body.find("select").each(function () {
      this.selectedIndex = 0;
    });
    resetSchedulingOwnerPickers();
    $body.find('input[type="date"]').val("").removeClass("has-value");
    $body.find('input[type="checkbox"]').prop("checked", false);
    $("#scheduleIntegratedReleaseStrip").text("窗口 — ｜ —");
    $("#scheduleIntegratedSystemsHint").text("涉及系统：—");
    resetStoryItems();
    if (rdApi) {
      rdApi.reset();
    }
    if (shared) {
      shared.schedulingUsers = [];
      shared.involvedProducts = [];
      shared.mainSystemId = 0;
      shared.productProjectsMap = {};
      shared.productExecutionsMap = {};
      shared.taskRowSeq = 0;
      shared.manualNodeSeq = 0;
    }
  }

  function fillModalHeader(ctx) {
    $("#scheduleIntegratedModalTitle").text("排期一体化办理 · " + ctx.id);
    $("#scheduleIntegratedReqTitle").text(ctx.title);
    $("#scheduleIntegratedReqSystem").text(ctx.system);
    $("#scheduleIntegratedReqOwner").text(ctx.owner);
  }

  function openScheduleIntegratedModal(source) {
    var ctx;
    var demandID = 0;

    if (source && source.jquery) {
      demandID = extractDemandID(source);
      ctx = extractRowContext(source);
    } else if (source && typeof source === "object" && source.id) {
      ctx = {
        id: source.id,
        title: source.title || "—",
        owner: source.owner || "待分配",
        system: source.system || "—",
      };
      demandID = source.demandId || source.demandID || 0;
    } else {
      ctx = {
        id: "REQ-—",
        title: "—",
        owner: "待分配",
        system: "—",
      };
    }

    resetIntegratedForm();
    fillModalHeader(ctx);

    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }

    if (demandID > 0) {
      loadSchedulingDetail(demandID);
    }
  }

  function closeScheduleIntegratedModal() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  window.openScheduleIntegratedModal = openScheduleIntegratedModal;
  window.closeScheduleIntegratedModal = closeScheduleIntegratedModal;

  $("#scheduleIntegratedModalOverlay").on("click", closeScheduleIntegratedModal);
  $("#scheduleIntegratedModalCloseBtn, #scheduleIntegratedModalDismissBtn").on("click", closeScheduleIntegratedModal);

  $(document).on("change input", "#scheduleIntegratedModalBody input[type='date']", function () {
    if (shared) {
      shared.syncIntegratedDateInputState(this);
    }
  });

  $(document).on("change", "#scheduleIntegratedWindowSelect", syncPlanDateFromWindow);
})(jQuery);
