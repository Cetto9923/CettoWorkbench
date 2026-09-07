(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var rdApi = window.ScheduleIntegratedRd;
  var tasksApi = window.ScheduleIntegratedTasks;
  var MODAL_IDS = ["scheduleIntegratedModal", "scheduleIntegratedModalOverlay"];
  var SAVE_BTN_DEFAULT_TEXT = "确认并同步";
  var SAVE_BTN_LOADING_TEXT = "保存中...";

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function parseEstimateValue(value) {
    var raw = $.trim(value == null ? "" : String(value));
    if (!raw) {
      return 0;
    }
    var num = Number(raw);
    return isNaN(num) ? 0 : num;
  }

  function parseTaskPri(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num < 0 || num > 4 ? 3 : num;
  }

  function toast(message, type) {
    if (typeof window.showToast === "function") {
      window.showToast(message, type || "success");
      return;
    }
    window.alert(message);
  }

  function flushInlineEdits() {
    if (rdApi && rdApi.exitStoryEditMode) {
      $("#scheduleIntegratedModalBody .rd-node--editing").each(function () {
        rdApi.exitStoryEditMode($(this));
      });
    }
    if (tasksApi && tasksApi.exitEditMode) {
      tasksApi.exitEditMode($("#scheduleIntegratedModalBody .rd-task-row--editing"));
    }
  }

  function readStoryTitle($node) {
    var $header = $node.find(".rd-node-header").first();
    var fromInput = $.trim($header.find(".rd-node-title-input, .rd-node-title").first().val() || "");
    if (fromInput) {
      return fromInput;
    }
    return $.trim($node.attr("data-story-title") || $header.find(".rd-node-title-display").first().text() || "");
  }

  function readStoryProductId($node) {
    var $header = $node.find(".rd-node-header").first();
    var fromSelect = $.trim($header.find(".rd-node-product").first().val() || "");
    if (fromSelect) {
      return parsePositiveInt(fromSelect);
    }
    return parsePositiveInt($node.attr("data-product-id"));
  }

  function readStoryAssignedTo($node) {
    var $header = $node.find(".rd-node-header").first();
    var fromHidden = $.trim($header.find(".rd-node-assignee-value").first().val() || "");
    if (fromHidden) {
      return fromHidden;
    }
    return $.trim($node.attr("data-assigned-to") || "");
  }

  function isDraftStoryEmpty($node) {
    var storyId = parsePositiveInt($node.attr("data-story-id"));
    if (storyId > 0) {
      return false;
    }
    return !readStoryTitle($node) && !readStoryProductId($node);
  }

  function collectTaskFromRow($row) {
    var $row = $($row);
    var taskId = parsePositiveInt($row.attr("data-task-id"));
    var isNewRow = $row.hasClass("rd-task-row--new") || taskId === 0;
    var isEditing = $row.hasClass("rd-task-row--editing") || $row.hasClass("rd-task-row--new");

    var executionId = 0;
    var type = "";
    var pri = 3;
    var name = "";
    var assignedTo = "";
    var estimate = 0;
    var estStarted = "";
    var deadline = "";

    if (isEditing) {
      executionId = parsePositiveInt($row.find(".rd-task-execution-select").first().val() || $row.attr("data-execution-id"));
      type = $.trim($row.find(".rd-task-type").first().val() || $row.attr("data-task-type") || "");
      pri = parseTaskPri($row.find(".rd-task-pri").first().val() || $row.attr("data-pri") || "3");
      name = $.trim($row.find(".rd-task-name").first().val() || $row.attr("data-task-name") || "");
      assignedTo = $.trim($row.find(".rd-node-assignee-value").first().val() || $row.attr("data-assigned-to") || "");
      estimate = parseEstimateValue($row.find(".rd-task-hours").first().val() || $row.attr("data-estimate"));
      estStarted = $.trim($row.find(".rd-task-start").first().val() || $row.attr("data-est-started") || "");
      deadline = $.trim($row.find(".rd-task-end").first().val() || $row.attr("data-deadline") || "");
    } else {
      executionId = parsePositiveInt($row.attr("data-execution-id"));
      type = $.trim($row.attr("data-task-type") || "");
      pri = parseTaskPri($row.attr("data-pri") || "3");
      name = $.trim($row.attr("data-task-name") || "");
      assignedTo = $.trim($row.attr("data-assigned-to") || "");
      estimate = parseEstimateValue($row.attr("data-estimate"));
      estStarted = $.trim($row.attr("data-est-started") || "");
      deadline = $.trim($row.attr("data-deadline") || "");
    }

    var task = {
      action: isNewRow ? "new" : "edit",
      executionId: executionId,
      type: type,
      pri: pri,
      name: name,
      assignedTo: assignedTo,
      estimate: estimate,
      estStarted: estStarted,
      deadline: deadline,
    };
    if (!isNewRow) {
      task.id = taskId;
    }
    return task;
  }

  function isNewTaskRowEmpty(task) {
    return !$.trim(task.name || "") && !task.executionId && !task.estimate && !$.trim(task.deadline || "");
  }

  function collectDeletedTasksForStory(storyId) {
    var tasks = [];
    (shared.deletedTaskIds || []).forEach(function (entry) {
      if (!entry || entry.storyId !== storyId || !entry.taskId) {
        return;
      }
      tasks.push({
        action: "delete",
        id: entry.taskId,
      });
    });
    return tasks;
  }

  function collectTasksFromNode($node, storyId) {
    var tasks = [];
    var seenDeleteIds = {};

    $node.find(".rd-task-body tr").each(function () {
      var task = collectTaskFromRow(this);
      if (task.action === "new" && isNewTaskRowEmpty(task)) {
        return;
      }
      tasks.push(task);
    });

    collectDeletedTasksForStory(storyId).forEach(function (task) {
      if (seenDeleteIds[task.id]) {
        return;
      }
      seenDeleteIds[task.id] = true;
      var exists = tasks.some(function (item) {
        return item.id === task.id;
      });
      if (!exists) {
        tasks.push(task);
      }
    });

    return tasks;
  }

  function collectStoryFromNode($node) {
    var storyId = parsePositiveInt($node.attr("data-story-id"));
    var isNew = $node.attr("data-new") === "true" || storyId === 0;

    if (isNew && isDraftStoryEmpty($node)) {
      return null;
    }

    var story = {
      productId: readStoryProductId($node),
      title: readStoryTitle($node),
      assignedTo: readStoryAssignedTo($node),
      estimate: parseEstimateValue($node.attr("data-estimate")),
      spec: $.trim($node.attr("data-spec") || ""),
      tasks: [],
    };

    if (isNew) {
      story.action = "new";
    } else {
      story.action = "edit";
      story.id = storyId;
    }

    story.tasks = collectTasksFromNode($node, storyId);
    return story;
  }

  function collectSchedulingData() {
    var data = {
      windowId: parsePositiveInt($("#scheduleIntegratedWindowSelect").val()),
      rd: $.trim($("#scheduleIntRDValue").val() || ""),
      qd: $.trim($("#scheduleIntQDValue").val() || ""),
      accepter: $.trim($("#scheduleIntAccepterValue").val() || ""),
      developFinish: $.trim($("#scheduleIntegratedDevelopFinish").val() || ""),
      testFinish: $.trim($("#scheduleIntegratedTestFinish").val() || ""),
      acceptancedDate: $.trim($("#scheduleIntegratedAcceptancedDate").val() || ""),
      stories: [],
    };

    $("#rdTreeNodes .rd-node").each(function () {
      var story = collectStoryFromNode($(this));
      if (story) {
        data.stories.push(story);
      }
    });

    var existingStoryIds = {};
    data.stories.forEach(function (story) {
      if (story.id) {
        existingStoryIds[story.id] = true;
      }
    });

    (shared.deletedStoryIds || []).forEach(function (storyId) {
      if (!storyId || existingStoryIds[storyId]) {
        return;
      }
      data.stories.push({
        action: "delete",
        id: storyId,
        tasks: [],
      });
    });

    return data;
  }

  function validateSchedulingData(data) {
    if (!data.windowId) {
      return "请选择版本窗口";
    }

    for (var i = 0; i < data.stories.length; i++) {
      var story = data.stories[i];
      if (story.action === "new") {
        if (!$.trim(story.title || "")) {
          return "新建的研发需求必须填写标题";
        }
        if (!story.productId) {
          return "新建的研发需求必须选择系统";
        }
      }
      if (story.action === "delete") {
        continue;
      }
      for (var j = 0; j < (story.tasks || []).length; j++) {
        var task = story.tasks[j];
        if (task.action === "delete") {
          continue;
        }
        if (!$.trim(task.name || "")) {
          return "任务名称不能为空";
        }
        if (!task.executionId) {
          return "请选择执行";
        }
        if (!task.estimate || task.estimate <= 0) {
          return "预估不能为空";
        }
        if (!$.trim(task.deadline || "")) {
          return "截止时间不能为空";
        }
      }
    }

    return "";
  }

  function setSaveButtonLoading(loading) {
    var $btn = $("#scheduleIntegratedSaveBtn");
    if (!$btn.length) {
      return;
    }
    if (loading) {
      if (!$btn.data("default-text")) {
        $btn.data("default-text", $.trim($btn.text()) || SAVE_BTN_DEFAULT_TEXT);
      }
      $btn.prop("disabled", true).text(SAVE_BTN_LOADING_TEXT);
      return;
    }
    var defaultText = $btn.data("default-text") || SAVE_BTN_DEFAULT_TEXT;
    $btn.prop("disabled", false).text(defaultText);
  }

  function saveScheduling() {
    // 先判来源：独立研发需求（story）走 story 路径，业需（demand）走 demand 路径
    var isStorySource = shared && shared.currentStoryId > 0;
    if (!isStorySource && (!shared || !shared.currentDemandId)) {
      toast("业需 ID 无效，请关闭弹窗后重试", "error");
      return;
    }

    flushInlineEdits();

    var data = collectSchedulingData();
    var validationError = validateSchedulingData(data);
    if (validationError) {
      toast(validationError, "error");
      return;
    }

    setSaveButtonLoading(true);

    var fetchFn = window.scheduleFetch;
    if (typeof fetchFn !== "function") {
      setSaveButtonLoading(false);
      toast("保存功能未加载，请刷新页面后重试", "error");
      return;
    }

    // 按来源分流：story 路径打 /schedule/stories/:id/save-scheduling，demand 路径保持原样
    var saveUrl = isStorySource
      ? "/schedule/stories/" + shared.currentStoryId + "/save-scheduling"
      : "/schedule/demands/" + shared.currentDemandId + "/save-scheduling";

    fetchFn(saveUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    })
      .then(function (resp) {
        return resp.json();
      })
      .then(function (result) {
        if (result && result.code === "PRODUCT_ACCESS_NOTICE") {
          showProductNotice(result.products || []);
          return;
        }
        if (result && result.success) {
          toast("保存成功", "success");
          closeScheduleIntegratedModal();
          window.location.reload();
          return;
        }
        toast((result && result.message) || "保存失败", "error");
      })
      .catch(function (err) {
        toast("保存失败: " + (err && err.message ? err.message : "请稍后重试"), "error");
      })
      .finally(function () {
        setSaveButtonLoading(false);
      });
  }

  function showProductNotice(products) {
    products = products || [];
    if (!products.length) { return; }
    // 找 id 最大的产品，用其后端生成的 viewUrl（完整 GET 风格链接，不再前端拼接）
    var pick = products[0];
    for (var i = 1; i < products.length; i++) {
      if ((products[i].id || 0) > (pick.id || 0)) { pick = products[i]; }
    }
    var names = [];
    for (var j = 0; j < products.length; j++) {
      var n = $.trim(products[j].name || "");
      if (n) { names.push(n); }
    }
    $("#scheduleProductNoticeMessage").text("您不是 " + names.join("、") + " 的负责人，请去禅道维护");
    $("#scheduleProductNoticeOkBtn").off("click").on("click", function () {
      var viewUrl = $.trim(pick.viewUrl || "");
      if (viewUrl) { window.open(viewUrl, "_blank"); }
      closeProductNotice();
    });
    $("#scheduleProductNoticeCancelBtn").off("click").on("click", closeProductNotice);
    $("#scheduleProductNoticeCloseBtn").off("click").on("click", closeProductNotice);
    if (typeof window.openShowModals === "function") {
      window.openShowModals(["scheduleProductNoticeModal", "scheduleProductNoticeOverlay"]);
    }
  }

  function closeProductNotice() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(["scheduleProductNoticeModal", "scheduleProductNoticeOverlay"]);
    }
  }

  window.showProductNotice = showProductNotice;
  window.closeProductNotice = closeProductNotice;

  function extractDemandID($btn) {
    var raw = $btn.data("demand-id");
    if (raw === undefined || raw === null || raw === "") {
      return 0;
    }
    var id = parseInt(String(raw), 10);
    return isNaN(id) || id <= 0 ? 0 : id;
  }

  function extractStoryID($btn) {
    var raw = $btn.data("story-id");
    if (raw === undefined || raw === null || raw === "") {
      return 0;
    }
    var id = parseInt(String(raw), 10);
    return isNaN(id) || id <= 0 ? 0 : id;
  }

  function isIndependentScheduleSource(source) {
    if (source && source.jquery) {
      var $row = source.closest("tr");
      return $row.hasClass("schedule-indep-row") || $row.hasClass("schedule-indep-child-row");
    }
    if (source && typeof source === "object" && source.isIndependent) {
      return true;
    }
    return false;
  }

  function setRdTaskSectionVisible(visible) {
    $("#scheduleIntegratedRdSection").toggle(!!visible);
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
      value: data.rd,
      label: data.rdName,
    });
    window.initAutocomplete("scheduleIntQDInput", "scheduleIntQDValue", items, {
      placeholder: placeholder,
      value: data.qd,
      label: data.qdName,
    });
    window.initAutocomplete("scheduleIntAccepterInput", "scheduleIntAccepterValue", items, {
      placeholder: placeholder,
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

  function applyWindowEditability(canEdit, phase) {
    var editable = canEdit !== false;
    var $select = $("#scheduleIntegratedWindowSelect");
    $select.prop("disabled", !editable);
    if (editable) {
      $select.removeAttr("title");
    } else {
      $select.attr("title", "终排业务需求不能修改版本窗口");
    }
    if (shared) {
      shared.currentCanEditWindow = editable;
      shared.currentWindowPhase = $.trim(phase || "");
    }
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
        $summary.text("只读 · 用户故事条目");
      }
      return;
    }

    list.forEach(function (item, index) {
      var row = shared ? shared.cloneTemplateElement("tplStoryItemRow", "tr") : null;
      if (!row) {
        return;
      }
      var effectivePoint = Number(item.effectivePoint || 0);
      if (!isNaN(effectivePoint)) {
        totalSp += effectivePoint;
      }

      row.querySelector(".story-item-index").textContent = String(index + 1);
      row.querySelector(".story-item-role").textContent = item.role || "—";
      row.querySelector(".story-item-title").textContent = item.gv || "—";
      row.querySelector(".story-item-product").textContent = item.productName || "—";
      row.querySelector(".story-item-estimate").textContent = item.pointLabel || "—";
      $tbody.append(row);
    });

    if ($summary.length) {
      $summary.text(
        "只读 · 合计 " +
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
      shared.zentaoURL = $.trim(data.zentaoUrl || "");
      shared.isSchedulingDetailLoaded = true;
    }

    $("#scheduleIntegratedReqTitle").text(data.name || "—");
    $("#scheduleIntegratedReqSystem").text(mainSystem);
    $("#scheduleIntegratedReqOwner").text(owner);

    fillWindowSelect(data.windows, data.windowId, data.windowName, data.schedulePlanDate);
    applyWindowEditability(data.canEditWindow, data.windowPhase);
    syncPlanDateFromWindow();
    initSchedulingOwnerPickers(users, data);
    setDateInputValue($("#scheduleIntegratedDevelopFinish"), data.developFinish);
    setDateInputValue($("#scheduleIntegratedTestFinish"), data.testFinish);
    setDateInputValue($("#scheduleIntegratedAcceptancedDate"), data.acceptancedDate);

    var systemsHint = shared
      ? shared.formatInvolvedSystemsHint(involvedProducts, mainSystem)
      : mainSystem;
    $("#scheduleIntegratedSystemsHint").text("涉及系统：" + systemsHint);

    fillStoryItems(data.userStories || []);
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
          if (shared) {
            shared.currentDemandId = parsePositiveInt(resp.id);
            shared.currentStoryId = 0;
          }
        }
      })
      .fail(function () {
        window.alert("加载业需详情失败，请稍后重试");
      });
  }

  function loadStorySchedulingDetail(storyID) {
    $.ajax({
      url: "/schedule/stories/" + storyID + "/scheduling",
      method: "GET",
      dataType: "json",
    })
      .done(function (resp) {
        if (!resp || !resp.success) {
          window.alert((resp && resp.error) || "加载研发需求详情失败");
          return;
        }
        fillSchedulingDetail(resp);
        if (resp.id) {
          $("#scheduleIntegratedModalTitle").text("排期一体化办理 · RD-" + resp.id);
          if (shared) {
            shared.currentStoryId = parsePositiveInt(resp.id);
            shared.currentDemandId = 0;
          }
        }
      })
      .fail(function () {
        window.alert("加载研发需求详情失败，请稍后重试");
      });
  }

  function resetIntegratedForm() {
    var $body = $("#scheduleIntegratedModalBody");

    $body.find("select").each(function () {
      this.selectedIndex = 0;
      $(this).prop("disabled", false).removeAttr("title");
    });
    resetSchedulingOwnerPickers();
    $body.find('input[type="date"]').val("").removeClass("has-value");
    $body.find('input[type="checkbox"]').prop("checked", false);
    $("#scheduleIntegratedReleaseStrip").text("窗口 — ｜ —");
    $("#scheduleIntegratedSystemsHint").text("涉及系统：—");
    resetStoryItems();
    setRdTaskSectionVisible(true);
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
      shared.currentDemandId = 0;
      shared.currentStoryId = 0;
      shared.currentDemandDetailURL = "";
      shared.currentCanEditWindow = true;
      shared.currentWindowPhase = "";
      shared.isSchedulingDetailLoaded = false;
      shared.resetDeletedRecords();
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
    var storyID = 0;
    var fromIndependent = isIndependentScheduleSource(source);

    if (source && source.jquery) {
      demandID = extractDemandID(source);
      storyID = extractStoryID(source);
      ctx = extractRowContext(source);
      ctx.detailUrl = $.trim(source.attr("href") || "");
    } else if (source && typeof source === "object" && source.id) {
      ctx = {
        id: source.id,
        title: source.title || "—",
        owner: source.owner || "待分配",
        system: source.system || "—",
        detailUrl: source.detailUrl || "",
      };
      demandID = source.demandId || source.demandID || 0;
    } else {
      ctx = {
        id: "REQ-—",
        title: "—",
        owner: "待分配",
        system: "—",
        detailUrl: "",
      };
    }

    resetIntegratedForm();
    fillModalHeader(ctx);
    setRdTaskSectionVisible(!fromIndependent);

    if (shared) {
      shared.currentDemandId = fromIndependent ? 0 : demandID;
      shared.currentStoryId = fromIndependent ? storyID : 0;
      shared.currentDemandDetailURL = fromIndependent ? "" : $.trim(ctx.detailUrl || "");
    }

    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }

    if (fromIndependent && storyID > 0) {
      loadStorySchedulingDetail(storyID);
    } else if (demandID > 0) {
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

  $("#scheduleIntegratedSaveBtn").on("click", function () {
    saveScheduling();
  });
})(jQuery);
