(function ($) {
  "use strict";

  var MODAL_IDS = ["scheduleIntegratedModal", "scheduleIntegratedModalOverlay"];
  var INPUT_STYLE =
    "width:100%;padding:4px;border:1px solid #d1d5db;border-radius:3px;font-size:11px;box-sizing:border-box;";
  var CELL_STYLE = "padding:6px 8px; border-bottom:1px solid #f0f0f0;";

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

  function resetIntegratedForm() {
    var $body = $("#scheduleIntegratedModalBody");

    $body.find("select").each(function () {
      this.selectedIndex = 0;
    });
    $body.find('input[type="date"]').val("");
    $body.find('input[type="checkbox"]').prop("checked", false);
    $body.find(".schedule-rush-pill").removeClass("is-on");
    $("#scheduleIntegratedReleaseStrip").text("窗口 — ｜ —");
    $("#scheduleIntegratedSystemsHint").text("涉及系统：—");
  }

  function fillModalHeader(ctx) {
    $("#scheduleIntegratedModalTitle").text("排期一体化办理 · " + ctx.id);
    $("#scheduleIntegratedReqTitle").text(ctx.title);
    $("#scheduleIntegratedReqSystem").text(ctx.system);
    $("#scheduleIntegratedReqOwner").text(ctx.owner);
  }

  function openScheduleIntegratedModal(source) {
    var ctx;

    if (source && source.jquery) {
      ctx = extractRowContext(source);
    } else if (source && typeof source === "object" && source.id) {
      ctx = {
        id: source.id,
        title: source.title || "—",
        owner: source.owner || "待分配",
        system: source.system || "—",
      };
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
  }

  function closeScheduleIntegratedModal() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function toggleRdNode($node) {
    var $tasks = $node.find(".rd-node-tasks").first();
    var $toggle = $node.find(".rd-toggle").first();
    var expanded = $tasks.is(":visible");

    if (expanded) {
      $tasks.hide();
      $toggle.text("▶");
      $node.find(".rd-node-header").first().css("border-radius", "6px");
      return;
    }

    $tasks.show();
    $toggle.text("▼");
    $node.find(".rd-node-header").first().css("border-radius", "6px 6px 0 0");
  }

  function buildEmptyTaskRow() {
    var selectStyle = INPUT_STYLE + "padding:3px 4px;";
    return (
      "<tr>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<select class="rd-task-type" style="' +
      selectStyle +
      '">' +
      '<option value="开发">开发</option>' +
      '<option value="测试">测试</option>' +
      '<option value="上线">上线</option>' +
      "</select>" +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<input type="text" class="rd-task-name" placeholder="任务名称" style="' +
      INPUT_STYLE +
      '">' +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<input type="text" class="rd-task-owner" placeholder="负责人" style="' +
      INPUT_STYLE +
      '">' +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<input type="number" class="rd-task-hours" min="0" step="0.5" placeholder="h" style="' +
      INPUT_STYLE +
      '">' +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<input type="date" class="rd-task-start" style="' +
      INPUT_STYLE +
      '">' +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      '">' +
      '<input type="date" class="rd-task-end" style="' +
      INPUT_STYLE +
      '">' +
      "</td>" +
      '<td style="' +
      CELL_STYLE +
      ' text-align:center;">' +
      '<button type="button" class="rd-remove-task" style="border:none;background:none;color:#999;cursor:pointer;font-size:12px;">×</button>' +
      "</td>" +
      "</tr>"
    );
  }

  function buildEmptyRdNode() {
    return (
      '<div class="rd-node" style="border:1px solid #e5e7eb; border-radius:6px; margin-bottom:12px;">' +
      '<div class="rd-node-header" style="display:flex; align-items:center; gap:8px; padding:10px 14px; background:#f8fafc; cursor:pointer; border-radius:6px 6px 0 0;">' +
      '<span class="rd-toggle" style="font-size:12px;">▼</span>' +
      '<span style="background:#64748b;color:#fff;padding:1px 6px;border-radius:3px;font-size:11px;">新</span>' +
      '<input type="text" class="rd-node-system" placeholder="系统" style="width:72px;padding:3px 6px;border:1px solid #d1d5db;border-radius:3px;font-size:11px;">' +
      '<input type="text" class="rd-node-title" placeholder="研发需求名称" style="flex:1;min-width:120px;padding:3px 8px;border:1px solid #d1d5db;border-radius:3px;font-size:12px;font-weight:600;">' +
      '<input type="text" class="rd-node-meta" placeholder="指派 / 团队" style="width:180px;margin-left:auto;padding:3px 8px;border:1px solid #d1d5db;border-radius:3px;font-size:11px;color:#666;">' +
      '<button type="button" class="rd-remove-node" style="border:1px solid #d1d5db;background:#fff;padding:3px 8px;border-radius:4px;font-size:11px;cursor:pointer;color:#666;" title="删除此研发需求">×</button>' +
      "</div>" +
      '<div class="rd-node-tasks" style="padding:0 14px 10px;">' +
      '<table style="width:100%; border-collapse:collapse; font-size:12px; margin-top:8px;">' +
      "<thead>" +
      '<tr style="background:#f9fafb;">' +
      '<th style="width:70px; padding:6px 8px; border-bottom:1px solid #e5e7eb; text-align:left;">类型</th>' +
      '<th style="padding:6px 8px; border-bottom:1px solid #e5e7eb; text-align:left;">任务名称</th>' +
      '<th style="width:90px; padding:6px 8px; border-bottom:1px solid #e5e7eb;">负责人</th>' +
      '<th style="width:60px; padding:6px 8px; border-bottom:1px solid #e5e7eb;">预估(h)</th>' +
      '<th style="width:90px; padding:6px 8px; border-bottom:1px solid #e5e7eb;">开始</th>' +
      '<th style="width:90px; padding:6px 8px; border-bottom:1px solid #e5e7eb;">截止</th>' +
      '<th style="width:40px; padding:6px 8px; border-bottom:1px solid #e5e7eb;">操作</th>' +
      "</tr>" +
      "</thead>" +
      '<tbody class="rd-task-body"></tbody>' +
      "</table>" +
      '<button type="button" class="rd-add-task" style="margin-top:6px; border:1px dashed #d1d5db; background:#fff; padding:4px 12px; border-radius:4px; font-size:11px; color:#2563eb; cursor:pointer;">+ 添加任务</button>' +
      "</div>" +
      "</div>"
    );
  }

  function addTaskRow($node) {
    $node.find(".rd-task-body").first().append(buildEmptyTaskRow());
  }

  function removeTaskRow($row) {
    $row.remove();
  }

  function addRdNode() {
    $("#rdTaskTree .rd-add-node").before(buildEmptyRdNode());
  }

  function removeRdNode($node) {
    $node.remove();
  }

  window.openScheduleIntegratedModal = openScheduleIntegratedModal;
  window.closeScheduleIntegratedModal = closeScheduleIntegratedModal;

  $("#scheduleIntegratedModalOverlay").on("click", closeScheduleIntegratedModal);
  $("#scheduleIntegratedModalCloseBtn, #scheduleIntegratedModalDismissBtn").on("click", closeScheduleIntegratedModal);

  $(document).on("click", "#scheduleIntegratedModalBody .rd-node-header", function (e) {
    if ($(e.target).closest("button, input, select, textarea").length) {
      return;
    }
    toggleRdNode($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-toggle", function (e) {
    e.stopPropagation();
    toggleRdNode($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-add-task", function (e) {
    e.preventDefault();
    addTaskRow($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-remove-task", function (e) {
    e.preventDefault();
    e.stopPropagation();
    removeTaskRow($(this).closest("tr"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-add-node", function (e) {
    e.preventDefault();
    addRdNode();
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-remove-node", function (e) {
    e.preventDefault();
    e.stopPropagation();
    removeRdNode($(this).closest(".rd-node"));
  });

  $(document).on("change", "#scheduleIntegratedModalBody .schedule-rush-pill input[type='checkbox']", function () {
    $(this).closest(".schedule-rush-pill").toggleClass("is-on", this.checked);
  });
})(jQuery);
