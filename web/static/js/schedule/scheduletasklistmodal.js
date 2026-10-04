(function ($) {
  "use strict";

  var MODAL_IDS = ["taskListModal", "taskListModalOverlay"];
  var currentStoryId = 0;

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }


  function toast(message, type) {
    if (typeof window.showToast === "function") {
      window.showToast(message, type || "error");
      return;
    }
    window.alert(message);
  }


  function formatHours(value) {
    var num = Number(value || 0);
    return String(isNaN(num) ? 0 : num);
  }

  function renderInfoBar(story) {
    var parts = [];
    if (story && story.demandName) {
      parts.push("业务需求 " + story.demandName);
    }
    if (story && story.windowName) {
      parts.push("版本窗口 " + story.windowName);
    }
    if (story && story.releaseDate) {
      parts.push("预计上线 " + story.releaseDate);
    }
    $("#taskListModalInfoBar").text(parts.join(" | ") || "—");
  }

  function renderRows(tasks) {
    var $body = $("#taskListModalTableBody");
    $body.empty();
    if (!tasks || !tasks.length) {
      $body.append('<tr><td colspan="13" class="task-list-modal-empty">暂无任务</td></tr>');
      return;
    }

    tasks.forEach(function (task) {
      var progress = parsePositiveInt(task.progress);
      var rowHtml = [
        "<tr>",
        "<td>", window.escapeHtml(task.id), "</td>",
        "<td>", window.escapeHtml(task.name || "—"), "</td>",
        "<td>", window.escapeHtml(task.typeLabel || task.type || "—"), "</td>",
        "<td>", window.escapeHtml(task.priLabel || ""), "</td>",
        "<td>", window.escapeHtml(task.statusLabel || task.status || "—"), "</td>",
        "<td>", window.escapeHtml(task.assignedToName || "—"), "</td>",
        "<td>", window.escapeHtml(task.finishedByName || "—"), "</td>",
        "<td>", window.escapeHtml(task.deadline || "—"), "</td>",
        "<td>", window.escapeHtml(task.finishedDate || "—"), "</td>",
        "<td>", window.escapeHtml(formatHours(task.estimate)), "</td>",
        "<td>", window.escapeHtml(formatHours(task.consumed)), "</td>",
        "<td>", window.escapeHtml(formatHours(task.left)), "</td>",
        '<td><div class="task-list-modal-progress"><span class="task-list-modal-progress-bar" style="width:' + progress + '%;"></span><em>' + progress + "%</em></div></td>",
        "</tr>",
      ].join("");
      $body.append(rowHtml);
    });
  }

  function renderSummary(summary) {
    summary = summary || {};
    $("#taskListModalSummary").text(
      "本页共 " + formatHours(summary.total) +
      " 个任务，未开始 " + formatHours(summary.waitCount) +
      "，进行中 " + formatHours(summary.doingCount) +
      "，总预计 " + formatHours(summary.estimateTotal) +
      " 工时，已消耗 " + formatHours(summary.consumedTotal) +
      " 工时，剩余 " + formatHours(summary.leftTotal) + " 工时。"
    );
  }

  function loadTaskListData(storyId) {
    return window.appJson("/schedule/stories/" + storyId + "/tasks").then(function (resp) {
      if (!resp || !resp.success) {
        return Promise.reject(new Error((resp && resp.error) || "加载失败"));
      }
      var story = resp.story || {};
      $("#taskListModalTitle").text("相关任务 · " + (story.id || storyId));
      renderInfoBar(story);
      renderRows(resp.tasks || []);
      renderSummary(resp.summary || {});
      return resp;
    });
  }

  function resetState() {
    $("#taskListModalTitle").text("相关任务");
    $("#taskListModalInfoBar").text("—");
    $("#taskListModalTableBody").html('<tr><td colspan="13" class="task-list-modal-empty">暂无任务</td></tr>');
    $("#taskListModalSummary").text("本页共 0 个任务，未开始 0，进行中 0，总预计 0 工时，已消耗 0 工时，剩余 0 工时。");
  }

  window.openTaskListModal = function (storyId) {
    storyId = parsePositiveInt(storyId);
    if (!storyId) {
      toast("研发需求 ID 无效");
      return;
    }
    currentStoryId = storyId;
    resetState();
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    } else {
      $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).addClass("show");
    }
    loadTaskListData(storyId).catch(function (err) {
      toast((err && err.message) || "加载失败");
      window.closeTaskListModal();
    });
  };

  window.closeTaskListModal = function () {
    resetState();
    currentStoryId = 0;
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    } else {
      $(MODAL_IDS.map(function (id) { return "#" + id; }).join(",")).removeClass("show");
    }
  };

  $(document).on("click", ".js-open-task-list-modal", function (e) {
    e.preventDefault();
    var storyId = parsePositiveInt($(this).data("story-id") || $(this).attr("data-story-id"));
    if (!storyId) {
      return;
    }
    window.openTaskListModal(storyId);
  });

  $("#taskListModalCloseBtn").on("click", window.closeTaskListModal);
  $("#taskListModalDismissBtn").on("click", window.closeTaskListModal);
  $("#taskListModalOverlay").on("click", window.closeTaskListModal);
})(jQuery);

// Quick iteration creation uses the same native API and searchable picker as scheduling.
(function () {
  'use strict';
  let current = null;
  let period = '2w';
  function previewIteration() {
    const plan = (current.plans || []).find(item => String(item.id) === document.getElementById('quickIterationPlan').value);
    const dateFormat = new Intl.DateTimeFormat('sv-SE', {timeZone: 'Asia/Shanghai'});
    const today = dateFormat.format(new Date());
    const end = new Date(today + 'T00:00:00+08:00');
    end.setUTCDate(end.getUTCDate() + (period === '4w' ? 27 : 13));
    const endDate = dateFormat.format(end);
    document.getElementById('quickIterationName').value = period === 'plan'
      ? (plan ? plan.begin + ' - ' + plan.end : '') : today + ' - ' + endDate;
    document.querySelectorAll('[data-iteration-period]').forEach(button => {
      button.classList.toggle('active', button.dataset.iterationPeriod === period);
      button.setAttribute('aria-pressed', String(button.dataset.iterationPeriod === period));
    });
  }
  document.addEventListener('schedule:task-modal', function (event) {
    current = event.detail;
    period = '2w';
    window.destroyAutocomplete('quickIterationPlanInput');
    const previous = document.getElementById('quickIteration');
    if (previous) previous.remove();
    const section = document.createElement('details');
    section.id = 'quickIteration';
    section.className = 'task-iteration-form';
    section.innerHTML = '<summary>快速创建迭代</summary><div class="task-iteration-fields">' +
      '<div class="task-iteration-field"><label for="quickIterationName">迭代名称</label><input class="input" id="quickIterationName" readonly></div>' +
      '<div class="task-iteration-field"><span>时间盒周期</span><div class="task-iteration-periods" role="group" aria-label="迭代周期"><button type="button" class="btn btn-sm" data-iteration-period="2w">2 周</button>' +
      '<button type="button" class="btn btn-sm" data-iteration-period="4w">4 周</button><button type="button" class="btn btn-sm" data-iteration-period="plan">计划时间</button></div></div>' +
      '<div class="task-iteration-field"><label for="quickIterationPlanInput">关联产品计划（可选）</label><input class="input" id="quickIterationPlanInput" placeholder="搜索当前产品的计划"><input type="hidden" id="quickIterationPlan"></div>' +
      '<div class="task-iteration-field task-iteration-field--actions"><button type="button" class="action-btn action-btn--primary" id="quickIterationCreate">创建并选用</button></div></div>';
    document.querySelector('#taskModal .task-modal-project-section').appendChild(section);
    window.initAutocomplete('quickIterationPlanInput', 'quickIterationPlan', (current.plans || []).map(plan => ({
      value: String(plan.id), label: plan.title + ' · ' + plan.begin + ' ~ ' + plan.end
    })), {labelOnly: true});
    section.querySelectorAll('[data-iteration-period]').forEach(button => button.addEventListener('click', function () {
      period = button.dataset.iterationPeriod;
      previewIteration();
    }));
    document.getElementById('quickIterationPlan').addEventListener('change', previewIteration);
    document.getElementById('quickIterationCreate').addEventListener('click', createIteration);
    previewIteration();
  });
  async function createIteration() {
    const storyID = current.story.id;
    const projectID = Number(document.getElementById('taskModalProjectSelect').value);
    const button = document.getElementById('quickIterationCreate');
    button.disabled = true;
    try {
      const result = await window.appJson('/schedule/stories/' + storyID + '/executions', {method: 'POST', body: {
        projectId: projectID,
        period: period, planId: Number(document.getElementById('quickIterationPlan').value)
      }});
      if (!result.success) throw new Error(result.message || '创建迭代失败');
      if (current.story.id !== storyID || Number(document.getElementById('taskModalProjectSelect').value) !== projectID) {
        window.showToast('迭代已创建，请在对应项目中选用', 'success');
        return;
      }
      const response = await window.appJson('/schedule/projects/' + projectID + '/executions');
      if (!response.success) throw new Error('迭代已创建，加载失败，请刷新核对');
      if (current.story.id !== storyID || Number(document.getElementById('taskModalProjectSelect').value) !== projectID) return;
      const created = (response.executions || []).find(item => item.id === result.executionId);
      if (!created) throw new Error('迭代已创建，请刷新执行列表核对');
      Object.assign(document.getElementById('taskModalProjectSelect').dataset, {quickProject: String(projectID), quickExecution: String(created.id)});
      document.querySelectorAll('#taskModal .rd-task-execution-select').forEach(select => {
        select.add(new Option(created.name, String(created.id)));
        select.disabled = false;
        if (!select.value) select.value = String(created.id);
      });
      document.getElementById('quickIteration').open = false;
      window.showToast('迭代已创建', 'success');
    } catch (error) {
      window.showToast(error.message || '创建迭代失败，请刷新核对', 'error');
    } finally { button.disabled = false; }
  }
})();
