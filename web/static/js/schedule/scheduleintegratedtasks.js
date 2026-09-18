(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  if (!shared) {
    return;
  }

  var TASK_TYPE_OPTIONS = shared.taskTypeOptions || [];

  function scheduleGetJSON(url) {
    if (window.scheduleFetch) {
      return window
        .scheduleFetch(url, {
          method: "GET",
          headers: { Accept: "application/json" },
        })
        .then(function (resp) {
          if (!resp.ok) {
            return Promise.reject(new Error("request failed"));
          }
          return resp.json();
        });
    }
    return $.ajax({
      url: url,
      method: "GET",
      dataType: "json",
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    });
  }

  function cacheExecutions(projectId, executions) {
    var key = String(projectId || "");
    if (!key) {
      return;
    }
    shared.productExecutionsMap[key] = executions || [];
  }

  function getCachedExecutions(projectId) {
    var key = String(projectId || "");
    if (!key || !shared.productExecutionsMap[key]) {
      return null;
    }
    return shared.productExecutionsMap[key];
  }

  function getStoryProjects($node) {
    var raw = $node.attr("data-projects");
    if (!raw) {
      return [];
    }
    try {
      return JSON.parse(raw);
    } catch (e) {
      return [];
    }
  }

  function getNodeProjects($node) {
    var productId = String($node.attr("data-product-id") || $node.data("product-id") || "");
    if (productId && shared.productProjectsMap[productId]) {
      return shared.productProjectsMap[productId];
    }
    return getStoryProjects($node);
  }

  function loadProductProjects(productId, done) {
    var key = String(productId || "");
    if (!key) {
      done([]);
      return;
    }
    if (shared.productProjectsMap[key] && shared.productProjectsMap[key].length) {
      done(shared.productProjectsMap[key]);
      return;
    }
    scheduleGetJSON("/schedule/products/" + key + "/projects")
      .then(function (resp) {
        var projects = resp && resp.success ? resp.projects || [] : [];
        shared.productProjectsMap[key] = projects;
        done(projects);
      })
      .catch(function () {
        done([]);
      });
  }

  function ensureNodeProjects($node, done) {
    var productId = String($node.attr("data-product-id") || $node.data("product-id") || "");
    var projects = getNodeProjects($node);
    if (projects.length || !productId) {
      done(projects);
      return;
    }
    loadProductProjects(productId, function (loaded) {
      $node.attr("data-projects", JSON.stringify(loaded));
      done(loaded);
    });
  }

  function fillProjectSelect($select, projects, selectedId) {
    var selected = String(selectedId || "");
    $select.empty();
    $("<option></option>").val("").text("请选择项目").appendTo($select);
    (projects || []).forEach(function (project) {
      var id = String(project.id || "");
      if (!id) {
        return;
      }
      $("<option></option>").val(id).text($.trim(project.name || "") || id).appendTo($select);
    });
    if (selected) {
      $select.val(selected);
    }
  }

  function getSectionProjectSelect($scope) {
    var $section = $scope.closest(".rd-node-tasks");
    if (!$section.length) {
      $section = $scope.find(".rd-node-tasks").first();
    }
    return $section.find(".rd-task-project-section .rd-task-project-select").first();
  }

  function syncProjectSectionVisibility($node) {
    var $section = $node.find(".rd-task-project-section").first();
    if (!$section.length) {
      return;
    }
    var hasTasks = $node.find(".rd-task-body .rd-task-row").length > 0;
    $section.toggle(hasTasks);
  }

  function readSectionProject($scope) {
    var $select = getSectionProjectSelect($scope);
    var projectId = $.trim($select.val() || "");
    return {
      projectId: projectId,
      projectName: projectId ? $.trim($select.find("option:selected").text() || "") : "",
    };
  }

  function pickDefaultProjectId($node) {
    var found = "";
    $node.find(".rd-task-body .rd-task-row").each(function () {
      if (found) {
        return;
      }
      var pid = $.trim($(this).attr("data-project-id") || "");
      if (pid) {
        found = pid;
      }
    });
    return found;
  }

  function syncRowsProjectAttrs($node, projectId, projectName) {
    $node.find(".rd-task-body .rd-task-row").each(function () {
      var $row = $(this);
      $row.attr("data-project-id", projectId || "");
      $row.attr("data-project-name", projectName || "");
    });
  }

  function refreshSectionEditableExecutions($node, projectId, clearSelected) {
    $node.find(".rd-task-body .rd-task-row").each(function () {
      var $row = $(this);
      if (!$row.find(".rd-task-execution-select").length) {
        return;
      }
      var selectedExecutionId = "";
      if (!clearSelected) {
        selectedExecutionId =
          $.trim($row.find(".rd-task-execution-select").val() || "") ||
          $.trim($row.attr("data-execution-id") || "");
      }
      loadRowExecutions($row, projectId, selectedExecutionId);
    });
  }

  function initNodeProjectSelect($node, preferredProjectId) {
    var $select = getSectionProjectSelect($node);
    if (!$select.length) {
      return;
    }
    var hasPreferred = arguments.length >= 2;
    ensureNodeProjects($node, function (projects) {
      var selected = hasPreferred ? String(preferredProjectId || "") : pickDefaultProjectId($node);
      if (selected) {
        var exists = (projects || []).some(function (project) {
          return String(project.id || "") === selected;
        });
        if (!exists) {
          selected = "";
        }
      }
      if (!selected && projects && projects.length === 1) {
        selected = String(projects[0].id || "");
      }
      fillProjectSelect($select, projects, selected);
      var project = readSectionProject($node);
      syncRowsProjectAttrs($node, project.projectId, project.projectName);
      if (hasPreferred) {
        refreshSectionEditableExecutions($node, project.projectId, true);
      }
    });
  }

  function fillExecutionSelect($select, executions, selectedId, disabled) {
    var selected = String(selectedId || "");
    $select.empty();
    $("<option></option>").val("").text("请选择执行").appendTo($select);
    (executions || []).forEach(function (execution) {
      var id = String(execution.id || "");
      if (!id) {
        return;
      }
      $("<option></option>")
        .val(id)
        .text(shared.formatExecutionOptionLabel(execution))
        .appendTo($select);
    });
    $select.prop("disabled", !!disabled);
    if (selected) {
      $select.val(selected);
    }
  }

  function isExecutionSameActive($row) {
    var $btn = $row.find(".rd-task-execution-same").first();
    var $combo = $row.find(".rd-task-execution-combo").first();
    if (!$btn.length || $combo.hasClass("is-first-row")) {
      return false;
    }
    return $btn.attr("aria-pressed") !== "false";
  }

  function setExecutionSameActive($row, active) {
    var $btn = $row.find(".rd-task-execution-same").first();
    if (!$btn.length) {
      return;
    }
    $btn.attr("aria-pressed", active ? "true" : "false");
  }

  function syncExecutionSameButton($row) {
    var $combo = $row.find(".rd-task-execution-combo").first();
    var $btn = $row.find(".rd-task-execution-same").first();
    if (!$combo.length || !$btn.length) {
      return false;
    }
    var hasPrev = $row.prevAll(".rd-task-row").length > 0;
    $combo.toggleClass("is-first-row", !hasPrev);
    if (!hasPrev) {
      $btn.attr("aria-pressed", "false");
      return false;
    }
    return true;
  }

  function readPrevRowExecution($row) {
    var $prev = $row.prevAll(".rd-task-row").first();
    if (!$prev.length) {
      return { executionId: "", executionName: "" };
    }
    if ($prev.hasClass("rd-task-row--new") || $prev.hasClass("rd-task-row--editing")) {
      var executionId = $.trim($prev.find(".rd-task-execution-select").val() || "");
      return {
        executionId: executionId,
        executionName: executionId
          ? $.trim($prev.find(".rd-task-execution-select option:selected").text() || "")
          : "",
      };
    }
    return {
      executionId: $.trim($prev.attr("data-execution-id") || ""),
      executionName: $.trim($prev.attr("data-execution-name") || ""),
    };
  }

  function applySameAsPrevExecution($row) {
    if (!isExecutionSameActive($row)) {
      return;
    }
    var prev = readPrevRowExecution($row);
    var $select = $row.find(".rd-task-execution-select").first();
    if (!$select.length) {
      return;
    }
    if (!prev.executionId) {
      $select.val("");
      return;
    }
    if ($select.find('option[value="' + prev.executionId + '"]').length) {
      $select.val(prev.executionId);
      return;
    }
    if (prev.executionName) {
      $("<option></option>").val(prev.executionId).text(prev.executionName).appendTo($select);
      $select.val(prev.executionId);
    }
  }

  function resolveRowExecutionSelectedId($row, preferredId) {
    if (preferredId) {
      return String(preferredId);
    }
    if (isExecutionSameActive($row)) {
      return readPrevRowExecution($row).executionId || "";
    }
    return "";
  }

  function loadRowExecutions($row, projectId, selectedExecutionId) {
    var $executionSelect = $row.find(".rd-task-execution-select").first();
    syncExecutionSameButton($row);
    var selectedId = resolveRowExecutionSelectedId($row, selectedExecutionId);
    if (isExecutionSameActive($row) && !selectedExecutionId) {
      selectedId = readPrevRowExecution($row).executionId || "";
    }
    if (!projectId) {
      fillExecutionSelect($executionSelect, [], "", true);
      applySameAsPrevExecution($row);
      return;
    }

    var cached = getCachedExecutions(projectId);
    if (cached) {
      fillExecutionSelect($executionSelect, cached, selectedId, false);
      applySameAsPrevExecution($row);
      return;
    }

    fillExecutionSelect($executionSelect, [], "", true);
    scheduleGetJSON("/schedule/projects/" + projectId + "/executions")
      .then(function (resp) {
        if (!resp || !resp.success) {
          fillExecutionSelect($executionSelect, [], "", false);
          applySameAsPrevExecution($row);
          return;
        }
        var executions = resp.executions || [];
        cacheExecutions(projectId, executions);
        fillExecutionSelect($executionSelect, executions, selectedId, false);
        applySameAsPrevExecution($row);
      })
      .catch(function () {
        fillExecutionSelect($executionSelect, [], "", false);
        applySameAsPrevExecution($row);
      });
  }

  function fillTaskTypeSelect($select, selectedType) {
    var selected = $.trim(selectedType || "");
    var seen = {};
    var options = TASK_TYPE_OPTIONS.slice();
    if (selected && options.indexOf(selected) === -1) {
      options.unshift(selected);
    }
    $select.empty();
    options.forEach(function (type) {
      if (seen[type]) {
        return;
      }
      seen[type] = true;
      $("<option></option>").val(type).text(shared.taskTypeLabel(type)).appendTo($select);
    });
    if (selected) {
      $select.val(selected);
    } else {
      $select.val("devel");
    }
  }

  function normalizeTaskPri(pri) {
    var raw = $.trim(String(pri == null ? "" : pri));
    if (!raw) {
      return 0;
    }
    var value = parseInt(raw, 10);
    if (isNaN(value) || value < 0 || value > 4) {
      return 3;
    }
    return value;
  }

  function formatTaskPriLabel(pri) {
    var value = normalizeTaskPri(pri);
    return value <= 0 ? "" : "P" + value;
  }

  function taskPriSelectValue(pri) {
    var value = normalizeTaskPri(pri);
    return value <= 0 ? "3" : String(value);
  }

  function mountTaskActions($container, showEdit) {
    var tplId = showEdit ? "tplRdTaskActions" : "tplRdTaskActionsDeleteOnly";
    var actions = shared.cloneTemplateElement(tplId, ".task-actions");
    if (!actions) {
      return;
    }
    $container.empty().append(actions);
  }

  function applyTaskDataAttrs($row, task) {
    $row.attr("data-project-id", task.projectId || "");
    $row.attr("data-project-name", task.projectName || "");
    $row.attr("data-execution-id", task.executionId || "");
    $row.attr("data-execution-name", task.executionName || "");
    $row.attr("data-task-type", task.type || "");
    $row.attr("data-pri", String(normalizeTaskPri(task.pri)));
    $row.attr("data-task-name", task.name || "");
    $row.attr("data-assigned-to", task.assignedTo || "");
    $row.attr("data-assigned-to-name", task.assignedToName || "");
    $row.attr("data-estimate", task.estimate != null ? String(task.estimate) : "");
    $row.attr("data-est-started", task.estStarted || "");
    $row.attr("data-deadline", task.deadline || "");
  }

  function readTaskDataFromRow($row) {
    return {
      id: $row.attr("data-task-id") || "",
      projectId: $row.attr("data-project-id") || "",
      projectName: $row.attr("data-project-name") || "",
      executionId: $row.attr("data-execution-id") || "",
      executionName: $row.attr("data-execution-name") || "",
      type: $row.attr("data-task-type") || "",
      pri: normalizeTaskPri($row.attr("data-pri")),
      name: $row.attr("data-task-name") || "",
      assignedTo: $row.attr("data-assigned-to") || "",
      assignedToName: $row.attr("data-assigned-to-name") || "",
      estimate: $row.attr("data-estimate") || "",
      estStarted: $row.attr("data-est-started") || "",
      deadline: $row.attr("data-deadline") || "",
    };
  }

  function collectTaskFormData($row) {
    var project = readSectionProject($row);
    var $executionSelect = $row.find(".rd-task-execution-select").first();
    var executionId = $.trim($executionSelect.val() || "");
    return {
      id: $row.attr("data-task-id") || "",
      projectId: project.projectId,
      projectName: project.projectName,
      executionId: executionId,
      executionName: executionId ? $.trim($executionSelect.find("option:selected").text() || "") : "",
      type: $.trim($row.find(".rd-task-type").val() || ""),
      pri: normalizeTaskPri($row.find(".rd-task-pri").val()),
      name: $.trim($row.find(".rd-task-name").val() || ""),
      assignedTo: $.trim($row.find(".rd-node-assignee-value").val() || ""),
      assignedToName: $.trim($row.find(".rd-node-assignee-input").val() || ""),
      estimate: $.trim($row.find(".rd-task-hours").val() || ""),
      estStarted: $.trim($row.find(".rd-task-start").val() || ""),
      deadline: $.trim($row.find(".rd-task-end").val() || ""),
    };
  }

  function renderTaskTypeCell($cell, type) {
    var label = shared.taskTypeLabel(type);
    var span = document.createElement("span");
    span.className = "rd-task-type-label " + shared.taskTypeClass(type);
    span.textContent = label || "—";
    $cell.empty().append(span);
  }

  function renderTaskReadCells($row, task) {
    $row.find(".rd-task-cell-execution").text(task.executionName || "—");
    renderTaskTypeCell($row.find(".rd-task-cell-type"), task.type);
    $row.find(".rd-task-cell-pri").text(formatTaskPriLabel(task.pri));
    $row.find(".rd-task-cell-name").text(task.name || "—");
    $row.find(".rd-task-cell-owner").text(task.assignedToName || task.assignedTo || "—");
    $row.find(".rd-task-cell-hours").text(shared.formatStoryEstimate(task.estimate));
    $row.find(".rd-task-cell-start").text(task.estStarted || "—");
    $row.find(".rd-task-cell-end").text(task.deadline || "—");
  }

  function syncTaskRowDateInputs($row) {
    $row.find("input[type='date']").each(function () {
      shared.syncIntegratedDateInputState(this);
    });
  }

  function destroyTaskOwnerPicker($row) {
    var inputId = $.trim($row.attr("data-owner-input-id") || "");
    if (inputId) {
      shared.destroyStoryAssigneePicker(inputId);
    }
  }

  function nextTaskOwnerIds($row) {
    destroyTaskOwnerPicker($row);
    shared.taskRowSeq += 1;
    var inputId = "rdTaskOwnerInput" + shared.taskRowSeq;
    var hiddenId = "rdTaskOwnerValue" + shared.taskRowSeq;
    $row.attr("data-owner-input-id", inputId);
    $row.attr("data-owner-hidden-id", hiddenId);
    return { inputId: inputId, hiddenId: hiddenId };
  }

  function mountTaskOwnerPicker($row, inputId, hiddenId, assignedTo, assignedToName) {
    var assigneeWrap = shared.cloneTemplateElement("tplRdStoryAssigneeEdit", ".rd-node-assignee-wrap");
    if (!assigneeWrap) {
      return;
    }
    $(assigneeWrap).find(".rd-node-assignee-input").attr("id", inputId);
    $(assigneeWrap).find(".rd-node-assignee-value").attr("id", hiddenId);
    $row.find(".rd-task-cell-owner").first().empty().append(assigneeWrap);
    shared.initStoryAssigneePicker(inputId, hiddenId, assignedTo, assignedToName);
  }

  function clearTaskOwnerPickerMeta($row) {
    destroyTaskOwnerPicker($row);
    $row.removeAttr("data-owner-input-id data-owner-hidden-id");
  }

  function copyEditCellsFromTemplate($row) {
    var tplRow = shared.cloneTemplateElement("tplRdTaskRowEdit", "tr");
    if (!tplRow) {
      return;
    }
    var cellClasses = [
      "rd-task-cell-execution",
      "rd-task-cell-type",
      "rd-task-cell-pri",
      "rd-task-cell-name",
      "rd-task-cell-hours",
      "rd-task-cell-start",
      "rd-task-cell-end",
      "rd-task-cell-actions",
    ];
    cellClasses.forEach(function (cls) {
      var src = tplRow.querySelector("." + cls);
      var dst = $row.find("." + cls).first()[0];
      if (src && dst) {
        dst.innerHTML = src.innerHTML;
      }
    });
  }

  function buildExistingTaskRow(task) {
    var type = $.trim(task.type || "");
    var taskData = {
      projectId: String(task.project || ""),
      projectName: task.projectName || "",
      executionId: String(task.execution || ""),
      executionName: task.executionName || "",
      type: type,
      pri: normalizeTaskPri(task.pri),
      name: task.name || "",
      assignedTo: task.assignedTo || "",
      assignedToName: task.assignedToName || "",
      estimate: task.estimate,
      estStarted: task.estStarted || "",
      deadline: task.deadline || "",
    };
    var row = shared.cloneTemplateElement("tplRdTaskRowReadonly", "tr");
    if (!row) {
      return null;
    }
    var $row = $(row);
    $row.attr("data-task-id", task.id || "");
    applyTaskDataAttrs($row, taskData);
    renderTaskReadCells($row, taskData);
    mountTaskActions($row.find(".rd-task-cell-actions"), true);
    return row;
  }

  function buildEmptyTaskRow(storyId) {
    var row = shared.cloneTemplateElement("tplRdTaskRowNew", "tr");
    if (!row) {
      return null;
    }
    row.setAttribute("data-story-id", storyId || "");
    var $row = $(row);
    mountTaskActions($row.find(".rd-task-cell-actions"), false);
    return row;
  }

  function buildTaskSectionBody(taskRows) {
    var section = shared.cloneTemplateElement("tplRdTaskSection", ".rd-node-tasks");
    if (!section) {
      return null;
    }
    var table = section.querySelector(".rd-task-table");
    var thead = shared.cloneTemplateElement("tplRdTaskTableHead", "thead");
    if (table && thead) {
      table.insertBefore(thead, table.querySelector(".rd-task-body"));
    }
    var $body = $(section).find(".rd-task-body").first();
    (taskRows || []).forEach(function (row) {
      if (row) {
        $body.append(row);
      }
    });
    var $projectSection = $(section).find(".rd-task-project-section").first();
    if ($projectSection.length) {
      $projectSection.toggle($body.find(".rd-task-row").length > 0);
    }
    return section;
  }

  function enterTaskEditMode($row) {
    if (!$row.length || $row.hasClass("rd-task-row--new") || $row.hasClass("rd-task-row--editing")) {
      return;
    }

    exitTaskEditMode($("#scheduleIntegratedModalBody .rd-task-row--editing").not($row));

    var rdApi = window.ScheduleIntegratedRd;
    if (rdApi && rdApi.exitStoryEditMode) {
      rdApi.exitStoryEditMode($("#scheduleIntegratedModalBody .rd-node--editing"));
    }

    var task = readTaskDataFromRow($row);
    var $node = $row.closest(".rd-node");

    $row.addClass("rd-task-row--editing");
    copyEditCellsFromTemplate($row);
    fillTaskTypeSelect($row.find(".rd-task-type"), task.type);
    $row.find(".rd-task-pri").val(taskPriSelectValue(task.pri));
    $row.find(".rd-task-name").val(task.name);
    $row.find(".rd-task-hours").val(task.estimate);
    $row.find(".rd-task-start").val(task.estStarted);
    $row.find(".rd-task-end").val(task.deadline);
    mountTaskActions($row.find(".rd-task-cell-actions"), false);

    var ownerIds = nextTaskOwnerIds($row);
    syncTaskRowDateInputs($row);

    ensureNodeProjects($node, function () {
      var project = readSectionProject($node);
      var projectId = project.projectId || task.projectId || "";
      if (projectId && !project.projectId) {
        fillProjectSelect(getSectionProjectSelect($node), getNodeProjects($node), projectId);
        project = readSectionProject($node);
      }
      syncRowsProjectAttrs($node, project.projectId || projectId, project.projectName || task.projectName);
      var hasPrev = syncExecutionSameButton($row);
      if (hasPrev) {
        var prevExec = readPrevRowExecution($row);
        setExecutionSameActive($row, !!(task.executionId && prevExec.executionId === String(task.executionId)));
      }
      loadRowExecutions($row, project.projectId || projectId, task.executionId);
    });
    mountTaskOwnerPicker($row, ownerIds.inputId, ownerIds.hiddenId, task.assignedTo, task.assignedToName);
    $row.find(".rd-task-name").first().trigger("focus");
  }

  function exitTaskEditMode($rows) {
    $rows.each(function () {
      var $row = $(this);
      if (!$row.hasClass("rd-task-row--editing")) {
        return;
      }

      var task = collectTaskFormData($row);
      clearTaskOwnerPickerMeta($row);
      applyTaskDataAttrs($row, task);
      renderTaskReadCells($row, task);
      mountTaskActions($row.find(".rd-task-cell-actions"), true);
      $row.removeClass("rd-task-row--editing");
    });
  }

  function readRowProjectExecution($sourceRow) {
    if (!$sourceRow || !$sourceRow.length) {
      var empty = { projectId: "", executionId: "" };
      return empty;
    }
    var project = readSectionProject($sourceRow);
    if ($sourceRow.hasClass("rd-task-row--new") || $sourceRow.hasClass("rd-task-row--editing")) {
      var form = collectTaskFormData($sourceRow);
      return {
        projectId: project.projectId || form.projectId || "",
        executionId: form.executionId || "",
      };
    }
    var data = readTaskDataFromRow($sourceRow);
    return {
      projectId: project.projectId || data.projectId || "",
      executionId: data.executionId || "",
    };
  }

  function normalizeDateValue(value) {
    var raw = $.trim(String(value == null ? "" : value));
    if (!raw) {
      return "";
    }
    return raw.slice(0, 10);
  }

  function resolveUserLabel(account) {
    var acc = $.trim(account || "");
    if (!acc) {
      return "";
    }
    var label = "";
    (shared.schedulingUsers || []).some(function (user) {
      if ($.trim(user.account || "") === acc) {
        label = $.trim(user.realname || "") || acc;
        return true;
      }
      return false;
    });
    return label || acc;
  }

  function readNodeStoryTitle($node) {
    var $header = $node.find(".rd-node-header").first();
    var fromInput = $.trim($header.find(".rd-node-title-input, .rd-node-title").first().val() || "");
    if (fromInput) {
      return fromInput;
    }
    return $.trim($node.attr("data-story-title") || $header.find(".rd-node-title-display").first().text() || "");
  }

  function readNodeProductId($node) {
    var $header = $node.find(".rd-node-header").first();
    var fromSelect = $.trim($header.find(".rd-node-product").first().val() || "");
    if (fromSelect) {
      return fromSelect;
    }
    return String($node.attr("data-product-id") || "");
  }

  function findProductReleaseOwner(productId) {
    var id = String(productId || "");
    if (!id) {
      return "";
    }
    var account = "";
    (shared.involvedProducts || []).some(function (product) {
      if (String(product.id || "") === id) {
        account = $.trim(product.rd || "");
        return true;
      }
      return false;
    });
    return account;
  }

  function initNewTaskRowFields($row, $node, defaults, config) {
    var sectionProject = readSectionProject($node);
    var projectId = sectionProject.projectId || (defaults && defaults.projectId) || "";
    var executionId = (defaults && defaults.executionId) || "";
    var hasPrev = syncExecutionSameButton($row);
    if (hasPrev) {
      setExecutionSameActive($row, true);
      if (!executionId) {
        executionId = readPrevRowExecution($row).executionId || "";
      }
    } else {
      setExecutionSameActive($row, false);
    }
    if (projectId) {
      if (!sectionProject.projectId) {
        ensureNodeProjects($node, function (projects) {
          fillProjectSelect(getSectionProjectSelect($node), projects, projectId);
          var project = readSectionProject($node);
          syncRowsProjectAttrs($node, project.projectId, project.projectName);
          loadRowExecutions($row, project.projectId || projectId, executionId);
        });
      } else {
        syncRowsProjectAttrs($node, sectionProject.projectId, sectionProject.projectName);
        loadRowExecutions($row, projectId, executionId);
      }
    } else {
      fillExecutionSelect($row.find(".rd-task-execution-select").first(), [], "", true);
      ensureNodeProjects($node, function (projects) {
        fillProjectSelect(getSectionProjectSelect($node), projects, "");
      });
      applySameAsPrevExecution($row);
    }

    var cfg = config || {};
    if (cfg.type) {
      fillTaskTypeSelect($row.find(".rd-task-type"), cfg.type);
    }
    if (cfg.name != null) {
      $row.find(".rd-task-name").val(cfg.name);
    }
    $row.find(".rd-task-start").val(normalizeDateValue(cfg.estStarted));
    $row.find(".rd-task-end").val(normalizeDateValue(cfg.deadline));

    var ownerIds = nextTaskOwnerIds($row);
    mountTaskOwnerPicker(
      $row,
      ownerIds.inputId,
      ownerIds.hiddenId,
      $.trim(cfg.assignedTo || ""),
      $.trim(cfg.assignedToName || "")
    );
    syncTaskRowDateInputs($row);
  }

  function appendEmptyTaskRow($node, $afterRow) {
    var storyId = $node.data("story-id") || "";
    var row = buildEmptyTaskRow(storyId);
    if (!row) {
      return;
    }
    var $row = $(row);
    if ($afterRow && $afterRow.length) {
      $afterRow.after($row);
    } else {
      $node.find(".rd-task-body").first().append($row);
    }
    syncProjectSectionVisibility($node);
    initNewTaskRowFields($row, $node, readRowProjectExecution($afterRow), null);
  }

  function appendDefaultTaskRows($node) {
    var storyId = $node.data("story-id") || "";
    var title = readNodeStoryTitle($node);
    var developFinish = normalizeDateValue($("#scheduleIntegratedDevelopFinish").val());
    var testFinish = normalizeDateValue($("#scheduleIntegratedTestFinish").val());
    var acceptancedDate = normalizeDateValue($("#scheduleIntegratedAcceptancedDate").val());
    var schedulePlanDate = normalizeDateValue($("#scheduleIntegratedSchedulePlanDate").val());
    var rd = $.trim($("#scheduleIntRDValue").val() || "");
    var rdName = $.trim($("#scheduleIntRDInput").val() || "") || resolveUserLabel(rd);
    var qd = $.trim($("#scheduleIntQDValue").val() || "");
    var qdName = $.trim($("#scheduleIntQDInput").val() || "") || resolveUserLabel(qd);
    var releaseOwner = findProductReleaseOwner(readNodeProductId($node));
    var releaseOwnerName = resolveUserLabel(releaseOwner);

    var specs = [
      {
        type: "devel",
        name: "【开发】" + title,
        assignedTo: rd,
        assignedToName: rdName,
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
        assignedTo: releaseOwner,
        assignedToName: releaseOwnerName,
        estStarted: schedulePlanDate,
        deadline: acceptancedDate,
      },
    ];

    specs.forEach(function (spec) {
      var row = buildEmptyTaskRow(storyId);
      if (!row) {
        return;
      }
      var $row = $(row);
      $node.find(".rd-task-body").first().append($row);
      initNewTaskRowFields($row, $node, null, spec);
    });
    syncProjectSectionVisibility($node);
  }

  function isAutocompleteTarget(target) {
    return $(target).closest(".ui-autocomplete, .ui-autocomplete-dropdown").length > 0;
  }

  function isSelectTarget(target) {
    return $(target).closest("select").length > 0;
  }

  window.ScheduleIntegratedTasks = {
    buildExistingTaskRow: buildExistingTaskRow,
    buildTaskSectionBody: buildTaskSectionBody,
    appendEmptyTaskRow: appendEmptyTaskRow,
    appendDefaultTaskRows: appendDefaultTaskRows,
    loadRowExecutions: loadRowExecutions,
    loadProductProjects: loadProductProjects,
    initNodeProjectSelect: initNodeProjectSelect,
    syncExecutionSameButton: syncExecutionSameButton,
    setExecutionSameActive: setExecutionSameActive,
    isExecutionSameActive: isExecutionSameActive,
    applySameAsPrevExecution: applySameAsPrevExecution,
    readPrevRowExecution: readPrevRowExecution,
    exitEditMode: exitTaskEditMode,
    mountTaskOwnerPicker: mountTaskOwnerPicker,
    syncTaskRowDateInputs: syncTaskRowDateInputs,
    clearTaskOwnerPickerMeta: clearTaskOwnerPickerMeta,
    nextTaskOwnerIds: nextTaskOwnerIds,
    destroyTaskOwnerPicker: destroyTaskOwnerPicker,
    fillTaskTypeSelect: fillTaskTypeSelect,
  };

  $(document).on("click", "#scheduleIntegratedModalBody .rd-add-task", function (e) {
    e.preventDefault();
    appendDefaultTaskRows($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .task-add-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $currentRow = $(this).closest("tr");
    appendEmptyTaskRow($(this).closest(".rd-node"), $currentRow);
  });

  $(document).on("click", "#scheduleIntegratedModalBody .task-delete-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $row = $(this).closest("tr");
    var $node = $row.closest(".rd-node");
    var storyId = parseInt(String($node.attr("data-story-id") || ""), 10);
    var taskId = parseInt(String($row.attr("data-task-id") || ""), 10);
    if (!isNaN(taskId) && taskId > 0) {
      shared.deletedTaskIds.push({
        storyId: !isNaN(storyId) && storyId > 0 ? storyId : 0,
        taskId: taskId,
      });
    }
    clearTaskOwnerPickerMeta($row);
    $row.remove();
    syncProjectSectionVisibility($node);
  });

  $(document).on("click", "#scheduleIntegratedModalBody .task-edit-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    enterTaskEditMode($(this).closest("tr"));
  });

  $(document).on("mousedown", function (e) {
    var $editing = $("#scheduleIntegratedModalBody .rd-task-row--editing");
    if (!$editing.length || isAutocompleteTarget(e.target) || isSelectTarget(e.target)) {
      return;
    }
    if ($(e.target).closest($editing).length) {
      return;
    }
    exitTaskEditMode($editing);
  });

  $(document).on("change", "#scheduleIntegratedModalBody .rd-task-project-section .rd-task-project-select", function () {
    var $node = $(this).closest(".rd-node");
    var project = readSectionProject($node);
    syncRowsProjectAttrs($node, project.projectId, project.projectName);
    refreshSectionEditableExecutions($node, project.projectId, true);
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-task-execution-same", function (e) {
    e.preventDefault();
    e.stopPropagation();
    var $row = $(this).closest("tr");
    if ($row.find(".rd-task-execution-combo").hasClass("is-first-row")) {
      return;
    }
    var nextActive = $(this).attr("aria-pressed") !== "true";
    setExecutionSameActive($row, nextActive);
    if (nextActive) {
      applySameAsPrevExecution($row);
      $row.nextAll(".rd-task-row").each(function () {
        if (isExecutionSameActive($(this))) {
          applySameAsPrevExecution($(this));
        }
      });
    }
  });

  $(document).on("change", "#scheduleIntegratedModalBody .rd-task-execution-select", function () {
    var $row = $(this).closest("tr");
    var hasPrev = syncExecutionSameButton($row);
    var prev = readPrevRowExecution($row);
    var current = $.trim($(this).val() || "");
    if (hasPrev && current && current === prev.executionId) {
      setExecutionSameActive($row, true);
    } else if (hasPrev) {
      setExecutionSameActive($row, false);
    }
    $row.nextAll(".rd-task-row").each(function () {
      if (isExecutionSameActive($(this))) {
        applySameAsPrevExecution($(this));
      }
    });
  });
})(jQuery);
