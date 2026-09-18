(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var tasksApi = window.ScheduleIntegratedTasks;
  var MODAL_IDS = ["taskModal", "taskModalOverlay"];
  var currentStoryId = 0;
  var deletedTaskIds = [];
  var cachedProjects = [];

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

  function escapeHtml(text) {
    if (shared && shared.escapeHtml) {
      return shared.escapeHtml(text);
    }
    return $("<div>").text(text == null ? "" : String(text)).html();
  }

  function toast(message, type) {
    if (typeof window.showToast === "function") {
      window.showToast(message, type || "success");
      return;
    }
    window.alert(message);
  }

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

  function schedulePostJSON(url, body) {
    if (window.scheduleFetch) {
      return window
        .scheduleFetch(url, {
          method: "POST",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify(body || {}),
        })
        .then(function (resp) {
          return resp.json().then(function (data) {
            return { ok: resp.ok, status: resp.status, data: data };
          });
        });
    }
    return $.ajax({
      url: url,
      method: "POST",
      contentType: "application/json",
      dataType: "json",
      data: JSON.stringify(body || {}),
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
    }).then(function (data) {
      return { ok: true, status: 200, data: data };
    });
  }

  function cloneRow(templateId) {
    if (shared && shared.cloneTemplateElement) {
      return shared.cloneTemplateElement(templateId, "tr");
    }
    var tpl = document.getElementById(templateId);
    if (!tpl || !tpl.content) {
      return null;
    }
    return document.importNode(tpl.content, true).querySelector("tr");
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

  function fillModalProjectSelect(projects, selectedId) {
    var $select = $("#taskModalProjectSelect");
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
    var selectedId = selectedExecutionId;
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
    return scheduleGetJSON("/schedule/projects/" + pid + "/executions")
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
    return escapeHtml(raw).replace(/\n/g, "<br>");
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
        html += "<li>" + escapeHtml(file.title || "附件") + "</li>";
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

  function renderExistingRow(task) {
    var row = cloneRow("tplTaskModalRowExisting");
    if (!row) {
      return null;
    }
    var $row = $(row);
    $row.attr("data-task-id", String(task.id || ""));
    $row.attr("data-task-type", task.type || "");
    $row.attr("data-pri", String(normalizeTaskPri(task.pri)));
    $row.attr("data-assigned-to", task.assignedTo || "");
    $row.attr("data-assigned-to-name", task.assignedToName || "");
    $row.attr("data-execution-id", String(task.executionId || ""));
    $row.find(".task-modal-type-cell").text(task.typeLabel || (shared && shared.taskTypeLabel(task.type)) || task.type || "—");
    $row.find(".rd-task-pri").val(taskPriSelectValue(task.pri));
    $row.find(".rd-task-name").val(task.name || "");
    $row.find(".rd-task-hours").val(task.estimate != null ? task.estimate : "");
    $row.find(".rd-task-start").val(task.estStarted || "");
    $row.find(".rd-task-end").val(task.deadline || "");
    return $row;
  }

  function mountTaskRowControls($row) {
    var assignedTo = $.trim($row.attr("data-assigned-to") || "");
    var assignedToName = $.trim($row.attr("data-assigned-to-name") || "");
    bindTaskRowControls($row, assignedTo, assignedToName);
  }

  function syncTaskModalProjectVisibility() {
    var hasTasks = $("#taskModalTableBody .task-modal-row").length > 0;
    $("#taskModalProjectSection").toggle(hasTasks);
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
        mountTaskRowControls($row);
        initTaskModalRowExecution($row, task.executionId || 0);
      }
    });
    syncTaskModalProjectVisibility();
  }

  function addTaskModalRow() {
    var row = cloneRow("tplTaskModalRowNew");
    if (!row) {
      return;
    }
    var $row = $(row);
    if (tasksApi) {
      tasksApi.fillTaskTypeSelect($row.find(".rd-task-type"), "devel");
    }
    $("#taskModalTableBody").append($row);
    mountTaskRowControls($row);
    initTaskModalRowExecution($row, 0);
    syncTaskModalProjectVisibility();
  }

  function resetModalState() {
    deletedTaskIds = [];
    cachedProjects = [];
    $("#taskModalTableBody .task-modal-row").each(function () {
      clearTaskRowControls($(this));
    });
    $("#taskModalTableBody").empty();
    $("#taskModalProjectSelect").empty().append($("<option></option>").val("").text("请选择项目"));
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
    return scheduleGetJSON("/schedule/stories/" + storyId + "/tasks").then(function (resp) {
      if (!resp || !resp.success) {
        return Promise.reject(new Error((resp && resp.error) || "加载失败"));
      }
      if (shared) {
        shared.schedulingUsers = resp.users || [];
      }

      cachedProjects = resp.projects || [];

      var story = resp.story || {};
      $("#taskModalTitle").text("拆任务 · " + (story.id || storyId));
      renderSpec(story);
      renderInfoBar(story);

      fillModalProjectSelect(cachedProjects, resolveDefaultProjectId(resp));
      renderTaskRows(resp.tasks || []);
      return resp;
    });
  }

  function readRowAssignedTo($row) {
    return $.trim($row.find(".rd-node-assignee-value").val() || "");
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
      // 去掉"创建"列后，现有任务默认保留编辑
      tasks.push({
        action: "edit",
        id: taskId,
        projectId: projectId,
        executionId: parsePositiveInt($row.find(".rd-task-execution-select").val()),
        type: $.trim($row.attr("data-task-type") || ""),
        pri: normalizeTaskPri($row.find(".rd-task-pri").val()),
        name: $.trim($row.find(".rd-task-name").val() || ""),
        assignedTo: readRowAssignedTo($row),
        estimate: Number($row.find(".rd-task-hours").val()) || 0,
        estStarted: $.trim($row.find(".rd-task-start").val() || ""),
        deadline: $.trim($row.find(".rd-task-end").val() || ""),
      });
    });

    $("#taskModalTableBody .task-modal-row--new").each(function () {
      var $row = $(this);
      tasks.push({
        action: "new",
        create: true,  // 去掉"创建"列后，新任务默认创建
        projectId: projectId,
        executionId: parsePositiveInt($row.find(".rd-task-execution-select").val()),
        type: $.trim($row.find(".rd-task-type").val() || "devel"),
        pri: normalizeTaskPri($row.find(".rd-task-pri").val()),
        name: $.trim($row.find(".rd-task-name").val() || ""),
        assignedTo: readRowAssignedTo($row),
        estimate: Number($row.find(".rd-task-hours").val()) || 0,
        estStarted: $.trim($row.find(".rd-task-start").val() || ""),
        deadline: $.trim($row.find(".rd-task-end").val() || ""),
      });
    });

    return tasks;
  }

  function saveTaskModal() {
    if (!currentStoryId) {
      return;
    }
    var tasks = collectTasksPayload();
    var missingExecution = tasks.some(function (item) {
      return item.action === "new" && item.create && !item.executionId;
    });
    if (missingExecution) {
      toast("请选择执行", "error");
      return;
    }

    var $btn = $("#taskModalSaveBtn");
    $btn.prop("disabled", true);
    schedulePostJSON("/schedule/stories/" + currentStoryId + "/save-tasks", {
      tasks: tasks,
    })
      .then(function (result) {
        var data = result.data || result;
        if (!data.success) {
          toast(data.message || data.error || "保存失败", "error");
          return;
        }
        toast("任务保存成功", "success");
        closeTaskModal();
        window.location.reload();
      })
      .catch(function () {
        toast("保存失败", "error");
      })
      .finally(function () {
        $btn.prop("disabled", false);
      });
  }

  window.openTaskModal = function (storyId) {
    storyId = parsePositiveInt(storyId);
    if (!storyId) {
      toast("研发需求 ID 无效", "error");
      return;
    }
    currentStoryId = storyId;
    resetModalState();
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    } else {
      $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).addClass("show");
    }
    loadTaskModalData(storyId).catch(function (err) {
      toast((err && err.message) || "加载失败", "error");
      closeTaskModal();
    });
  };

  window.closeTaskModal = function () {
    resetModalState();
    currentStoryId = 0;
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    } else {
      $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).removeClass("show");
    }
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
