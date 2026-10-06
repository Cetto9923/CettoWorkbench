(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var tasksApi = window.ScheduleIntegratedTasks;
  var MODAL_IDS = ["taskModal", "taskModalOverlay"];
  var currentStoryId = 0;
  var deletedTaskIds = [];
  var cachedProjects = [];
  var currentStoryInfo = {};

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function normalizeTaskPri(value) {
    var raw = $.trim(String(value == null ? "" : value));
    if (!raw) {
      return 0;
    }
    var num = parseInt(raw, 10);
    return isNaN(num) || num < 0 || num > 4 ? 3 : num;
  }

  function taskPriSelectValue(value) {
    var num = normalizeTaskPri(value);
    return num <= 0 ? "3" : String(num);
  }


  function toast(message, type) {
    if (typeof window.showToast === "function") {
      window.showToast(message, type || "success");
      return;
    }
    window.alert(message);
  }



  function bindTaskRowControls($row, assignedTo, assignedToName) {
    if (!tasksApi) {
      return;
    }
    var ownerIds = tasksApi.nextTaskOwnerIds($row);
    tasksApi.mountTaskOwnerPicker($row, ownerIds.inputId, ownerIds.hiddenId, assignedTo, assignedToName);
    tasksApi.syncTaskRowDateInputs($row);
  }

  function clearTaskRowControls($row) {
    if (tasksApi) {
      tasksApi.clearTaskOwnerPickerMeta($row);
    }
  }

  function toProjectAutocompleteItems(projects) {
    return (projects || []).filter(function (project) { return !!project.id; }).map(function (project) {
      return {value: String(project.id), label: $.trim(project.name || "") || String(project.id)};
    });
  }

  function fillModalProjectSelect(projects, selectedId) {
    $("#taskModalProjectSelect").removeAttr("data-quick-project data-quick-execution");
    window.initAutocomplete("taskModalProjectInput", "taskModalProjectSelect", toProjectAutocompleteItems(projects), {
      placeholder: "搜索项目", value: String(selectedId || ""), labelOnly: true,
    });
  }

  function getModalProjectId() {
    return parsePositiveInt($("#taskModalProjectSelect").val());
  }

  function fillRowExecutionSelect($row, executions, selectedId, disabled) {
    var $select = $row.find(".rd-task-execution-select").first();
    var selected = String(selectedId || "");
    $select.empty();
    $("<option></option>").val("").text("请选择执行").appendTo($select);
    (executions || []).forEach(function (execution) {
      var id = String(execution.id || "");
      if (!id) {
        return;
      }
      var label = shared && shared.formatExecutionOptionLabel
        ? shared.formatExecutionOptionLabel(execution)
        : execution.name || id;
      $("<option></option>").val(id).text(label).appendTo($select);
    });
    $select.prop("disabled", !!disabled);
    if (selected) {
      $select.val(selected);
    }
  }

  function loadRowExecutions($row, projectId, selectedExecutionId) {
    var pid = parsePositiveInt(projectId);
    var $project = $("#taskModalProjectSelect");
    var selectedId = selectedExecutionId || ($project.attr("data-quick-project") === String(pid)
      ? parsePositiveInt($project.attr("data-quick-execution")) : 0);
    if (tasksApi) {
      var hasPrev = tasksApi.syncExecutionSameButton($row);
      if (hasPrev && !selectedId && tasksApi.isExecutionSameActive($row)) {
        selectedId = parsePositiveInt(tasksApi.readPrevRowExecution($row).executionId);
      }
    }
    if (!pid) {
      fillRowExecutionSelect($row, [], 0, true);
      if (tasksApi) {
        tasksApi.applySameAsPrevExecution($row);
      }
      return $.Deferred().resolve([]).promise();
    }
    fillRowExecutionSelect($row, [], 0, true);
    return window.appJson("/schedule/projects/" + pid + "/executions")
      .then(function (resp) {
        var executions = (resp && resp.executions) || [];
        fillRowExecutionSelect($row, executions, selectedId, false);
        if (tasksApi) {
          tasksApi.applySameAsPrevExecution($row);
        }
        return executions;
      })
      .catch(function () {
        fillRowExecutionSelect($row, [], 0, false);
        if (tasksApi) {
          tasksApi.applySameAsPrevExecution($row);
        }
        return [];
      });
  }

  function sanitizeRichHtml(html) {
    var tmp = document.createElement("div");
    tmp.innerHTML = String(html || "");
    tmp.querySelectorAll("script,style,iframe,object,embed,form,link,meta").forEach(function (el) {
      el.remove();
    });
    tmp.querySelectorAll("*").forEach(function (el) {
      Array.from(el.attributes).forEach(function (attr) {
        var name = attr.name || "";
        var value = String(attr.value || "");
        if (/^on/i.test(name) || (name === "href" && /^javascript:/i.test(value))) {
          el.removeAttribute(name);
        }
      });
    });
    return tmp.innerHTML;
  }

  function formatRichTextContent(raw) {
    raw = String(raw == null ? "" : raw).trim();
    if (!raw) {
      return "";
    }
    if (/<[a-z][\s\S]*>/i.test(raw)) {
      return sanitizeRichHtml(raw);
    }
    return window.escapeHtml(raw).replace(/\n/g, "<br>");
  }

  function renderSpec(story) {
    var html = "";
    if (story.spec) {
      html += '<div class="task-modal-spec-block"><div class="task-modal-spec-label">需求描述</div><div class="task-modal-spec-content">' + formatRichTextContent(story.spec) + "</div></div>";
    }
    if (story.verify) {
      html += '<div class="task-modal-spec-block"><div class="task-modal-spec-label">验收标准</div><div class="task-modal-spec-content">' + formatRichTextContent(story.verify) + "</div></div>";
    }
    if (story.attachments && story.attachments.length) {
      html += '<div class="task-modal-spec-block"><div class="task-modal-spec-label">附件</div><ul class="task-modal-attachments">';
      story.attachments.forEach(function (file) {
        html += "<li>" + window.escapeHtml(file.title || "附件") + "</li>";
      });
      html += "</ul></div>";
    }
    if (!html) {
      html = '<div class="task-modal-spec-empty">暂无描述</div>';
    }
    $("#taskModalSpec").html(html);
  }

  function renderInfoBar(story) {
    var parts = [];
    if (story.demandName) {
      parts.push("业务需求 " + story.demandName);
    }
    if (story.windowName) {
      parts.push("版本窗口 " + story.windowName);
    }
    if (story.releaseDate) {
      parts.push("预计上线 " + story.releaseDate);
    }
    $("#taskModalInfoBar").text(parts.join(" | ") || "—");
  }

  function taskToRowData(task) {
    return {
      id: String(task.id || ""),
      projectId: String(task.projectId || ""),
      projectName: task.projectName || "",
      executionId: String(task.executionId || ""),
      executionName: task.executionName || "",
      type: $.trim(task.type || ""),
      pri: normalizeTaskPri(task.pri),
      name: task.name || "",
      assignedTo: task.assignedTo || "",
      assignedToName: task.assignedToName || "",
      estimate: task.estimate != null ? task.estimate : "",
      estStarted: task.estStarted || "",
      deadline: task.deadline || "",
    };
  }

  // 存量任务默认只读，点铅笔才进入编辑态（与排期弹窗一致）。
  function renderExistingRow(task) {
    var row = shared.cloneTemplateElement("tplRdTaskRowReadonly", "tr");
    if (!row) {
      return null;
    }
    var $row = $(row);
    var taskData = taskToRowData(task);
    $row.attr("data-task-id", taskData.id);
    tasksApi.applyTaskDataAttrs($row, taskData);
    tasksApi.renderTaskReadCells($row, taskData);
    tasksApi.mountTaskActions($row.find(".rd-task-cell-actions"), true);
    return $row;
  }

  function mountTaskRowControls($row) {
    var assignedTo = $.trim($row.attr("data-assigned-to") || "");
    var assignedToName = $.trim($row.attr("data-assigned-to-name") || "");
    bindTaskRowControls($row, assignedTo, assignedToName);
  }

  function normalizeDateValue(value) {
    var raw = $.trim(String(value == null ? "" : value));
    if (!raw) {
      return "";
    }
    return raw.slice(0, 10);
  }

  function syncTaskModalProjectVisibility() {
    var hasTasks = $("#taskModalTableBody .task-modal-row").length > 0;
    $("#taskModalProjectSection").toggle(hasTasks || !!currentStoryInfo.id);
  }

  function initTaskModalRowExecution($row, preferredExecutionId) {
    if (tasksApi) {
      var hasPrev = tasksApi.syncExecutionSameButton($row);
      if (hasPrev) {
        var prev = tasksApi.readPrevRowExecution($row);
        var preferred = String(preferredExecutionId || "");
        if (preferred && prev.executionId && preferred === prev.executionId) {
          tasksApi.setExecutionSameActive($row, true);
        } else if (preferred) {
          tasksApi.setExecutionSameActive($row, false);
        } else {
          tasksApi.setExecutionSameActive($row, true);
          preferredExecutionId = prev.executionId || 0;
        }
      }
    }
    loadRowExecutions($row, getModalProjectId(), preferredExecutionId || 0);
  }

  function renderTaskRows(tasks) {
    var $body = $("#taskModalTableBody");
    $body.find(".task-modal-row").each(function () {
      clearTaskRowControls($(this));
    });
    $body.empty();
    (tasks || []).forEach(function (task) {
      var $row = renderExistingRow(task);
      if ($row) {
        $body.append($row);
      }
    });
    syncTaskModalProjectVisibility();
  }

  function addEmptyTaskModalRow(config, $afterRow) {
    var row = shared.cloneTemplateElement("tplRdTaskRowNew", "tr");
    if (!row) {
      return null;
    }
    var $row = $(row);
    var cfg = config || {};
    var taskType = $.trim(cfg.type || "devel");
    if (tasksApi) {
      tasksApi.fillTaskTypeSelect($row.find(".rd-task-type"), taskType);
    } else {
      $row.find(".rd-task-type").val(taskType);
    }
    if (cfg.name != null) {
      $row.find(".rd-task-name").val(cfg.name);
    }
    $row.find(".rd-task-start").val(normalizeDateValue(cfg.estStarted));
    $row.find(".rd-task-end").val(normalizeDateValue(cfg.deadline));
    if (cfg.assignedTo) {
      $row.attr("data-assigned-to", $.trim(cfg.assignedTo || ""));
      $row.attr("data-assigned-to-name", $.trim(cfg.assignedToName || "") || shared.resolveUserLabel(cfg.assignedTo));
    }
    if (tasksApi) {
      tasksApi.mountTaskActions($row.find(".rd-task-cell-actions"), false);
    }
    if ($afterRow && $afterRow.length) {
      $afterRow.after($row);
    } else {
      $("#taskModalTableBody").append($row);
    }
    mountTaskRowControls($row);
    initTaskModalRowExecution($row, 0);
    syncTaskModalProjectVisibility();
    return $row;
  }

  function buildDefaultTaskSpecs() {
    var title = $.trim(currentStoryInfo.title || "");
    var developFinish = normalizeDateValue($("#scheduleIntegratedDevelopFinish").val());
    var testFinish = normalizeDateValue($("#scheduleIntegratedTestFinish").val());
    var acceptancedDate = normalizeDateValue($("#scheduleIntegratedAcceptancedDate").val());
    var schedulePlanDate =
      normalizeDateValue($("#scheduleIntegratedSchedulePlanDate").val()) ||
      normalizeDateValue(currentStoryInfo.releaseDate);
    var qd = $.trim($("#scheduleIntQDValue").val() || "");
    var qdName = $.trim($("#scheduleIntQDInput").val() || "") || shared.resolveUserLabel(qd);

    return [
      {
        type: "devel",
        name: "【开发】" + title,
        assignedTo: "",
        assignedToName: "",
        estStarted: "",
        deadline: developFinish,
      },
      {
        type: "test",
        name: "【测试】" + title,
        assignedTo: qd,
        assignedToName: qdName,
        estStarted: developFinish,
        deadline: testFinish,
      },
      {
        type: "Online",
        name: "【上线】" + title,
        assignedTo: "",
        assignedToName: "",
        estStarted: schedulePlanDate,
        deadline: acceptancedDate,
      },
    ];
  }

  function addDefaultTaskModalRows() {
    buildDefaultTaskSpecs().forEach(function (spec) {
      addEmptyTaskModalRow(spec);
    });
  }

  function addTaskModalRow() {
    if ($("#taskModalTableBody .task-modal-row").length === 0) {
      addDefaultTaskModalRows();
      return;
    }
    addEmptyTaskModalRow();
  }

  function resetModalState() {
    deletedTaskIds = [];
    cachedProjects = [];
    currentStoryInfo = {};
    $("#taskModalTableBody .task-modal-row").each(function () {
      clearTaskRowControls($(this));
    });
    $("#taskModalTableBody").empty();
    fillModalProjectSelect([], "");
    $("#taskModalProjectSection").hide();
    $("#taskModalSpec").empty();
    $("#taskModalInfoBar").empty();
  }

  function resolveDefaultProjectId(resp) {
    var defaultId = parsePositiveInt(resp && resp.defaultProjectId);
    if (defaultId) {
      return defaultId;
    }
    var tasks = (resp && resp.tasks) || [];
    for (var i = 0; i < tasks.length; i++) {
      var pid = parsePositiveInt(tasks[i].projectId);
      if (pid) {
        return pid;
      }
    }
    var projects = (resp && resp.projects) || [];
    if (projects.length === 1) {
      return parsePositiveInt(projects[0].id);
    }
    return 0;
  }

  function loadTaskModalData(storyId) {
    return window.appJson("/schedule/stories/" + storyId + "/tasks").then(function (resp) {
      if (!resp || !resp.success) {
        return Promise.reject(new Error((resp && resp.error) || "加载失败"));
      }
      if (shared) {
        shared.schedulingUsers = resp.users || [];
      }

      cachedProjects = resp.projects || [];

      var story = resp.story || {};
      currentStoryInfo = {
        id: story.id || storyId,
        title: story.title || "",
        releaseDate: story.releaseDate || "",
        productId: story.productId || 0,
      };
      $("#taskModalTitle").text("拆任务 · " + (story.id || storyId));
      renderSpec(story);
      renderInfoBar(story);

      fillModalProjectSelect(cachedProjects, resolveDefaultProjectId(resp));
      renderTaskRows(resp.tasks || []);
      document.dispatchEvent(new CustomEvent("schedule:task-modal", {detail: resp}));
    });
  }

  // 合并 data-* 与当前输入框：只读行只有 data-*，编辑行以输入框为准。
  function collectRowFormData($row) {
    var data = tasksApi.readTaskDataFromRow($row);
    var $type = $row.find(".rd-task-type");
    var $pri = $row.find(".rd-task-pri");
    var $name = $row.find(".rd-task-name");
    var $hours = $row.find(".rd-task-hours");
    var $start = $row.find(".rd-task-start");
    var $end = $row.find(".rd-task-end");
    var $execution = $row.find(".rd-task-execution-select");

    if ($type.length) {
      data.type = $.trim($type.val() || "");
    }
    if ($pri.length) {
      data.pri = normalizeTaskPri($pri.val());
    }
    if ($name.length) {
      data.name = $.trim($name.val() || "");
    }
    if ($hours.length) {
      data.estimate = $.trim($hours.val() || "");
    }
    if ($start.length) {
      data.estStarted = $.trim($start.val() || "");
    }
    if ($end.length) {
      data.deadline = $.trim($end.val() || "");
    }
    // 执行下拉是异步填充的，未加载完时保留 data-* 里的原执行，避免被空值覆盖。
    if ($execution.length && $.trim($execution.val() || "")) {
      data.executionId = $.trim($execution.val() || "");
    }
    var $ownerValue = $row.find(".rd-node-assignee-value").first();
    if ($ownerValue.length) {
      data.assignedTo = $.trim($ownerValue.val() || "");
      data.assignedToName =
        $.trim($row.find(".rd-node-assignee-input").first().val() || "") ||
        shared.resolveUserLabel(data.assignedTo);
    }
    data.id = $row.attr("data-task-id") || data.id || "";
    data.projectId = String(getModalProjectId() || "");
    return data;
  }

  function enterRowEdit($row) {
    if (
      !$row.length ||
      $row.hasClass("rd-task-row--new") ||
      $row.hasClass("rd-task-row--editing")
    ) {
      return;
    }
    exitRowEdit($("#taskModalTableBody .rd-task-row--editing").not($row));

    var task = tasksApi.readTaskDataFromRow($row);
    $row.addClass("rd-task-row--editing");
    tasksApi.copyEditCellsFromTemplate($row);
    tasksApi.fillTaskTypeSelect($row.find(".rd-task-type"), task.type);
    $row.find(".rd-task-pri").val(taskPriSelectValue(task.pri));
    $row.find(".rd-task-name").val(task.name);
    $row.find(".rd-task-hours").val(task.estimate);
    $row.find(".rd-task-start").val(normalizeDateValue(task.estStarted));
    $row.find(".rd-task-end").val(normalizeDateValue(task.deadline));
    tasksApi.mountTaskActions($row.find(".rd-task-cell-actions"), false);
    tasksApi.syncTaskRowDateInputs($row);

    var ownerIds = tasksApi.nextTaskOwnerIds($row);
    tasksApi.mountTaskOwnerPicker(
      $row,
      ownerIds.inputId,
      ownerIds.hiddenId,
      task.assignedTo,
      task.assignedToName
    );
    initTaskModalRowExecution($row, task.executionId || 0);
  }

  function exitRowEdit($rows) {
    $rows.each(function () {
      var $row = $(this);
      if (!$row.hasClass("rd-task-row--editing")) {
        return;
      }
      var task = collectRowFormData($row);
      tasksApi.clearTaskOwnerPickerMeta($row);
      tasksApi.applyTaskDataAttrs($row, task);
      tasksApi.renderTaskReadCells($row, task);
      tasksApi.mountTaskActions($row.find(".rd-task-cell-actions"), true);
      $row.removeClass("rd-task-row--editing");
    });
  }

  function collectTasksPayload() {
    var tasks = [];
    var projectId = getModalProjectId();
    deletedTaskIds.forEach(function (id) {
      tasks.push({ action: "delete", id: id });
    });

    $("#taskModalTableBody .task-modal-row--existing").each(function () {
      var $row = $(this);
      var taskId = parsePositiveInt($row.attr("data-task-id"));
      if (!taskId) {
        return;
      }
      var data = collectRowFormData($row);
      tasks.push({
        action: "edit",
        id: taskId,
        projectId: projectId,
        executionId: parsePositiveInt(data.executionId),
        type: $.trim(data.type || ""),
        pri: normalizeTaskPri(data.pri),
        name: $.trim(data.name || ""),
        assignedTo: $.trim(data.assignedTo || ""),
        estimate: Number(data.estimate) || 0,
        estStarted: $.trim(data.estStarted || ""),
        deadline: $.trim(data.deadline || ""),
      });
    });

    $("#taskModalTableBody .task-modal-row--new").each(function () {
      var $row = $(this);
      var data = collectRowFormData($row);
      tasks.push({
        action: "new",
        create: true,
        projectId: projectId,
        executionId: parsePositiveInt(data.executionId),
        type: $.trim(data.type || "devel"),
        pri: normalizeTaskPri(data.pri),
        name: $.trim(data.name || ""),
        assignedTo: $.trim(data.assignedTo || ""),
        estimate: Number(data.estimate) || 0,
        estStarted: $.trim(data.estStarted || ""),
        deadline: $.trim(data.deadline || ""),
      });
    });

    return tasks;
  }

  function saveTaskModal() {
    if (!currentStoryId) {
      return;
    }
    // 编辑中的行先落回 data-*，collectTasksPayload 才能读到最新值。
    exitRowEdit($("#taskModalTableBody .rd-task-row--editing"));
    var tasks = collectTasksPayload();
    var missingExecution = tasks.some(function (item) {
      return item.action === "new" && item.create && !item.executionId;
    });
    if (missingExecution) {
      toast("请选择执行", "error");
      return;
    }

    var $btn = $("#taskModalSaveBtn").prop("disabled", true);
    window.appJson("/schedule/stories/" + currentStoryId + "/save-tasks", { method: "POST", body: { tasks: tasks } })
      .then(function (result) {
        if (!result || !result.success) {
          toast((result && (result.message || result.error)) || "保存失败", "error");
          return;
        }
        toast("任务保存成功", "success");
        closeTaskModal();
        var params = new URLSearchParams(window.location.search);
        var winId = params.get("windows") || (currentStoryInfo && currentStoryInfo.windowId);
        if (winId) {
          var targetUrl = "/schedule?windows=" + encodeURIComponent(winId) + "&filter=all_open";
          if (params.get("tab") === "indep") targetUrl += "&tab=indep";
          targetUrl += "&highlight=" + encodeURIComponent(currentStoryId);
          window.location.href = targetUrl;
          return;
        }
        toast("任务保存成功，未获取到目标窗口，保留当前视图", "info");
        window.location.reload();
      })
      .catch(function () { toast("保存失败", "error"); })
      .finally(function () { $btn.prop("disabled", false); });
  }

  window.openTaskModal = function (storyId) {
    storyId = parsePositiveInt(storyId);
    if (!storyId) {
      toast("研发需求 ID 无效", "error");
      return;
    }
    currentStoryId = storyId;
    resetModalState();
    if (typeof window.openShowModals === "function") window.openShowModals(MODAL_IDS);
    else $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).addClass("show");
    loadTaskModalData(storyId).catch(function (err) {
      toast((err && err.message) || "加载失败", "error");
      closeTaskModal();
    });
  };

  window.closeTaskModal = function () {
    resetModalState();
    currentStoryId = 0;
    if (typeof window.closeShowModals === "function") window.closeShowModals(MODAL_IDS);
    else $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).removeClass("show");
  };

  $("#taskModalProjectSelect").on("change", function () {
    var pid = parsePositiveInt($(this).val());
    $("#taskModalTableBody .task-modal-row").each(function () {
      loadRowExecutions($(this), pid, 0);
    });
  });

  $("#taskModalTableBody").on("click", ".rd-task-execution-same", function (e) {
    e.preventDefault();
    e.stopPropagation();
    if (!tasksApi) {
      return;
    }
    var $row = $(this).closest("tr");
    if ($row.find(".rd-task-execution-combo").hasClass("is-first-row")) {
      return;
    }
    var nextActive = $(this).attr("aria-pressed") !== "true";
    tasksApi.setExecutionSameActive($row, nextActive);
    if (nextActive) {
      tasksApi.applySameAsPrevExecution($row);
      $row.nextAll(".task-modal-row").each(function () {
        if (tasksApi.isExecutionSameActive($(this))) {
          tasksApi.applySameAsPrevExecution($(this));
        }
      });
    }
  });

  $("#taskModalTableBody").on("change", ".rd-task-execution-select", function () {
    if (!tasksApi) {
      return;
    }
    var $row = $(this).closest("tr");
    var hasPrev = tasksApi.syncExecutionSameButton($row);
    var prev = tasksApi.readPrevRowExecution($row);
    var current = $.trim($(this).val() || "");
    if (hasPrev && current && current === prev.executionId) {
      tasksApi.setExecutionSameActive($row, true);
    } else if (hasPrev) {
      tasksApi.setExecutionSameActive($row, false);
    }
    $row.nextAll(".task-modal-row").each(function () {
      if (tasksApi.isExecutionSameActive($(this))) {
        tasksApi.applySameAsPrevExecution($(this));
      }
    });
  });

  $("#taskModalTableBody").on("click", ".task-edit-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    enterRowEdit($(this).closest("tr"));
  });

  $("#taskModalTableBody").on("click", ".task-add-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    addEmptyTaskModalRow({}, $(this).closest("tr"));
  });

  $("#taskModalTableBody").on("click", ".task-delete-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $row = $(this).closest("tr");
    var taskId = parsePositiveInt($row.attr("data-task-id"));
    if (taskId) {
      deletedTaskIds.push(taskId);
    }
    clearTaskRowControls($row);
    $row.remove();
    syncTaskModalProjectVisibility();
  });

  // 点到编辑行之外时自动存回只读
  $(document).on("mousedown", function (e) {
    var $editing = $("#taskModalTableBody .rd-task-row--editing");
    if (!$editing.length) {
      return;
    }
    if ($(e.target).closest("select, .ui-autocomplete, .ui-autocomplete-dropdown").length) {
      return;
    }
    if ($(e.target).closest($editing).length) {
      return;
    }
    exitRowEdit($editing);
  });

  $("#taskModalAddBtn").on("click", addTaskModalRow);
  $("#taskModalSaveBtn").on("click", saveTaskModal);
  $("#taskModalOverlay").on("click", closeTaskModal);

  $(document).on("change", "#taskModalTasksWrap input[type='date']", function () {
    if (tasksApi) {
      tasksApi.syncTaskRowDateInputs($(this).closest("tr"));
    }
  });

  $(document).on("click", ".js-open-task-modal", function (e) {
    e.preventDefault();
    var storyId = parsePositiveInt($(this).data("story-id") || $(this).attr("data-story-id"));
    if (!storyId) {
      var $row = $(this).closest("tr");
      var badge = $.trim($row.find(".schedule-id-badge").first().text() || "").replace(/^#/, "");
      badge = badge.replace(/^(REQ|SUB|RD|US)-?/i, "");
      storyId = parsePositiveInt(badge);
    }
    openTaskModal(storyId);
  });
})(jQuery);
