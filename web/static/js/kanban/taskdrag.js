/* =============================================================================
   文件: web/static/js/kanban/taskdrag.js
   模块: 工作看板
   职责: 任务看板未开始↔进行中前端拖拽（仅 DOM，不持久化）。
============================================================================= */
(function () {
  "use strict";

  var taskBoard = document.querySelector(".po-board .task-board");
  if (!taskBoard || taskBoard.dataset.dragBound === "1") return;
  taskBoard.dataset.dragBound = "1";

  var draggedCard = null;

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
    if (!card || !targetKey) return;
    var fromStatus = card.getAttribute("data-task-status") || "";
    if (fromStatus === targetKey) return;
    if (targetKey !== "wait" && targetKey !== "doing") return;
    if (fromStatus !== "wait" && fromStatus !== "doing") return;

    var targetCol = taskBoard.querySelector(
      '.task-col[data-col="' + targetKey + '"]'
    );
    var sourceCol = card.closest(".task-col");
    if (!targetCol) return;
    var targetBody = targetCol.querySelector(".task-col-body");
    if (!targetBody) return;

    card.setAttribute("data-task-status", targetKey);
    targetBody.appendChild(card);
    refreshColMeta(sourceCol);
    refreshColMeta(targetCol);
  }

  taskBoard.addEventListener("dragstart", function (e) {
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
    if (!draggedCard) return;
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
    if (!draggedCard) return;
    var body = e.target.closest(".task-col-body");
    if (!body || !taskBoard.contains(body)) return;
    e.preventDefault();
    clearDragOver();
    var col = body.closest(".task-col");
    var target = col ? col.getAttribute("data-col") : "";
    var card = draggedCard;
    draggedCard = null;
    card.classList.remove("dragging");
    moveCardToColumn(card, target);
  });
})();
