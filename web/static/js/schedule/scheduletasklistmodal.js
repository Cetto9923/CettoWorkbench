(function ($) {
  "use strict";

  var MODAL_IDS = ["taskListModal", "taskListModalOverlay"];
  var currentStoryId = 0;

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function escapeHtml(text) {
    return $("<div>").text(text == null ? "" : String(text)).html();
  }

  function toast(message, type) {
    if (typeof window.showToast === "function") {
      window.showToast(message, type || "error");
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

  function formatHours(value) {
    var num = Number(value || 0);
    if (isNaN(num)) {
      num = 0;
    }
    if (Math.floor(num) === num) {
      return String(num);
    }
    return String(num);
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
        "<td>", escapeHtml(task.id), "</td>",
        "<td>", escapeHtml(task.name || "—"), "</td>",
        "<td>", escapeHtml(task.typeLabel || task.type || "—"), "</td>",
        "<td>", escapeHtml(task.priLabel || ""), "</td>",
        "<td>", escapeHtml(task.statusLabel || task.status || "—"), "</td>",
        "<td>", escapeHtml(task.assignedToName || "—"), "</td>",
        "<td>", escapeHtml(task.finishedByName || "—"), "</td>",
        "<td>", escapeHtml(task.deadline || "—"), "</td>",
        "<td>", escapeHtml(task.finishedDate || "—"), "</td>",
        "<td>", escapeHtml(formatHours(task.estimate)), "</td>",
        "<td>", escapeHtml(formatHours(task.consumed)), "</td>",
        "<td>", escapeHtml(formatHours(task.left)), "</td>",
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
    return scheduleGetJSON("/schedule/stories/" + storyId + "/tasks").then(function (resp) {
      if (!resp || !resp.success) {
        return Promise.reject(new Error((resp && resp.error) || "加载失败"));
      }
      var story = resp.story || {};
      $("#taskListModalTitle").text("相关任务 · RD-" + (story.id || storyId));
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
