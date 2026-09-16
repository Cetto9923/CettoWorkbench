/* =============================================================================
   文件: web/static/js/kanban/taskdrag.js
   模块: 工作看板
   职责: 任务看板未开始↔进行中拖拽，成功后 PUT /kanban/tasks/:id 持久化。
============================================================================= */
(function () {
  "use strict";

  var root = document.querySelector(".po-board");
  var taskBoard = root ? root.querySelector(".task-board") : null;
  if (!taskBoard || taskBoard.dataset.dragBound === "1") return;
  taskBoard.dataset.dragBound = "1";

  var draggedCard = null;
  var pending = false;

  function toast(msg, type) {
    if (typeof window.showToast === "function") {
      window.showToast(msg, type || "danger");
    }
  }

  function reloadTasks() {
    if (root) {
      root.dispatchEvent(new CustomEvent("kanban:tasks-reload"));
    }
  }

  function clearDragOver() {
    taskBoard.querySelectorAll(".drag-over").forEach(function (el) {
      el.classList.remove("drag-over");
    });
  }

  function refreshColMeta(colEl) {
    if (!colEl) return;
    var body = colEl.querySelector(".task-col-body");
    var count = colEl.querySelector(".k-count");
    if (!body) return;
    var cards = body.querySelectorAll(".task-card");
    if (count) count.textContent = String(cards.length);
    var empty = body.querySelector(".demand-empty");
    if (cards.length === 0) {
      if (!empty) {
        body.innerHTML =
          '<div class="demand-empty demand-empty--visible">暂无任务</div>';
      }
    } else if (empty) {
      empty.remove();
    }
  }

  function moveCardToColumn(card, targetKey) {
    if (!card || !targetKey) return false;
    var fromStatus = card.getAttribute("data-task-status") || "";
    if (fromStatus === targetKey) return false;
    if (targetKey !== "wait" && targetKey !== "doing") return false;
    if (fromStatus !== "wait" && fromStatus !== "doing") return false;

    var targetCol = taskBoard.querySelector(
      '.task-col[data-col="' + targetKey + '"]'
    );
    var sourceCol = card.closest(".task-col");
    if (!targetCol) return false;
    var targetBody = targetCol.querySelector(".task-col-body");
    if (!targetBody) return false;

    card.setAttribute("data-task-status", targetKey);
    targetBody.appendChild(card);
    refreshColMeta(sourceCol);
    refreshColMeta(targetCol);
    return true;
  }

  function submitStatus(taskId, status) {
    var fetchFn =
      typeof window.appFetch === "function" ? window.appFetch : fetch;
    return fetchFn("/kanban/tasks/" + encodeURIComponent(taskId), {
      method: "PUT",
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({ status: status }),
    }).then(function (r) {
      return r
        .json()
        .catch(function () {
          return {};
        })
        .then(function (p) {
          if (!r.ok || !p || p.success !== true) {
            throw new Error((p && p.message) || "任务状态更新失败");
          }
          return p;
        });
    });
  }

  taskBoard.addEventListener("dragstart", function (e) {
    if (pending) {
      e.preventDefault();
      return;
    }
    var card = e.target.closest(".task-card.is-draggable");
    if (!card || !taskBoard.contains(card)) return;
    var status = card.getAttribute("data-task-status") || "";
    if (status !== "wait" && status !== "doing") {
      e.preventDefault();
      return;
    }
    draggedCard = card;
    card.classList.add("dragging");
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = "move";
      e.dataTransfer.setData(
        "text/plain",
        card.getAttribute("data-task-id") || ""
      );
    }
  });

  taskBoard.addEventListener("dragend", function () {
    if (draggedCard) draggedCard.classList.remove("dragging");
    clearDragOver();
    draggedCard = null;
  });

  taskBoard.addEventListener("dragover", function (e) {
    if (!draggedCard || pending) return;
    var body = e.target.closest(".task-col-body");
    if (!body || !taskBoard.contains(body)) return;
    var col = body.closest(".task-col");
    var key = col ? col.getAttribute("data-col") : "";
    if (key !== "wait" && key !== "doing") return;
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = "move";
    clearDragOver();
    body.classList.add("drag-over");
    if (col) col.classList.add("drag-over");
  });

  taskBoard.addEventListener("dragleave", function (e) {
    var body = e.target.closest(".task-col-body");
    if (!body || !taskBoard.contains(body)) return;
    if (body.contains(e.relatedTarget)) return;
    body.classList.remove("drag-over");
    var col = body.closest(".task-col");
    if (col) col.classList.remove("drag-over");
  });

  taskBoard.addEventListener("drop", function (e) {
    if (!draggedCard || pending) return;
    var body = e.target.closest(".task-col-body");
    if (!body || !taskBoard.contains(body)) return;
    e.preventDefault();
    clearDragOver();
    var col = body.closest(".task-col");
    var target = col ? col.getAttribute("data-col") : "";
    var card = draggedCard;
    draggedCard = null;
    card.classList.remove("dragging");

    var fromStatus = card.getAttribute("data-task-status") || "";
    var taskId = card.getAttribute("data-task-id") || "";
    if (!taskId || !target || target === fromStatus) return;
    if (!moveCardToColumn(card, target)) return;

    pending = true;
    toast("正在更新任务状态…", "success");
    submitStatus(taskId, target)
      .then(function () {
        toast("任务状态已更新", "success");
      })
      .catch(function (err) {
        toast(err && err.message ? err.message : "任务状态更新失败", "danger");
        reloadTasks();
      })
      .then(function () {
        pending = false;
      });
  });
})();
