(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var tasksApi = window.ScheduleIntegratedTasks;
  if (!shared || !tasksApi) {
    return;
  }

  function mountNodeActions($header, tplId) {
    var actions = shared.cloneTemplateElement(tplId, ".rd-node-actions");
    if (!actions) {
      return;
    }
    $header.find(".rd-node-actions").replaceWith(actions);
  }

  function fillStoryDisplayHeader($header, story) {
    var assignee = $.trim(story.assignedToName || "") || $.trim(story.assignedTo || "") || "待分配";
    $header.find(".rd-node-id").text("RD-" + (story.id || "—"));
    $header.find(".rd-node-role-badge").replaceWith(shared.buildRoleBadge(story.productId, shared.mainSystemId));
    $header.find(".rd-node-product-name").text(story.productName || "—");
    $header.find(".rd-node-title-display").text(story.title || "—");
    $header.find(".rd-node-assignee-display").text("指派: " + assignee);
    mountNodeActions($header, "tplRdNodeActionsEdit");
  }

  function applyStoryDataAttrs($node, story) {
    $node.attr("data-story-title", $.trim(story.title || ""));
    $node.attr("data-assigned-to", $.trim(story.assignedTo || ""));
    $node.attr("data-assigned-to-name", $.trim(story.assignedToName || ""));
  }

  function buildStoryNode(story) {
    var node = shared.cloneTemplateElement("tplRdNode", ".rd-node");
    if (!node) {
      return null;
    }
    var $node = $(node);
    $node.attr("data-story-id", story.id || "");
    fillStoryDisplayHeader($node.find(".rd-node-header").first(), story);

    var taskRows = (story.tasks || [])
      .map(function (task) {
        return tasksApi.buildExistingTaskRow(task);
      })
      .filter(function (row) {
        return !!row;
      });
    var section = tasksApi.buildTaskSectionBody(taskRows);
    if (section) {
      node.appendChild(section);
    }
    return node;
  }

  function buildDraftRdNode() {
    shared.manualNodeSeq += 1;
    var nodeId = shared.manualNodeSeq;
    var defaultProductId = shared.mainSystemId || "";

    var node = shared.cloneTemplateElement("tplRdNodeDraft", ".rd-node");
    if (!node) {
      return null;
    }
    var $node = $(node);
    $node.attr("data-node-id", nodeId);
    $node.attr("data-product-id", defaultProductId || "");

    var $header = $node.find(".rd-node-header").first();
    $header.find(".rd-node-role-badge").replaceWith(shared.buildRoleBadge(defaultProductId, shared.mainSystemId));
    shared.fillProductSelect($header.find(".rd-node-product"), shared.involvedProducts, defaultProductId);

    var inputId = "rdNodeOwnerInput" + nodeId;
    var hiddenId = "rdNodeOwnerValue" + nodeId;
    $header.find(".rd-node-assignee-input").attr("id", inputId);
    $header.find(".rd-node-assignee-value").attr("id", hiddenId);
    mountNodeActions($header, "tplRdNodeActionsDelete");

    var section = tasksApi.buildTaskSectionBody([]);
    if (section) {
      node.appendChild(section);
    }
    return node;
  }

  function destroyStoryAssigneePicker(inputId) {
    if (!inputId || typeof window.destroyAutocomplete !== "function") {
      return;
    }
    window.destroyAutocomplete(inputId);
  }

  function initStoryAssigneePicker(inputId, hiddenId, assignedTo, assignedToName) {
    if (typeof window.initAutocomplete !== "function" || !inputId || !hiddenId) {
      return;
    }
    destroyStoryAssigneePicker(inputId);
    window.initAutocomplete(inputId, hiddenId, shared.toAutocompleteItems(shared.schedulingUsers), {
      placeholder: "输入姓名或工号搜索",
      maxShow: 50,
      value: assignedTo || "",
      label: assignedToName || "",
    });
  }

  function initDraftNodePickers($node) {
    var inputId = $node.find(".rd-node-assignee-input").attr("id");
    var hiddenId = $node.find(".rd-node-assignee-value").attr("id");
    initStoryAssigneePicker(inputId, hiddenId, "", "");
  }

  function assigneeDisplayText(assignedToName, assignedTo) {
    return $.trim(assignedToName || "") || $.trim(assignedTo || "") || "待分配";
  }

  function closeOtherInlineEdits($exceptNode) {
    $("#scheduleIntegratedModalBody .rd-node--editing").each(function () {
      var $node = $(this);
      if (!$exceptNode || !$exceptNode.length || $node[0] !== $exceptNode[0]) {
        exitStoryEditMode($node);
      }
    });
    if (tasksApi.exitEditMode) {
      tasksApi.exitEditMode($("#scheduleIntegratedModalBody .rd-task-row--editing"));
    }
  }

  function enterStoryEditMode($node) {
    if (!$node.length || $node.hasClass("rd-node--draft") || $node.hasClass("rd-node--editing")) {
      return;
    }

    closeOtherInlineEdits($node);

    var storyId = $node.attr("data-story-id") || "0";
    var title = $node.attr("data-story-title") || "";
    var assignedTo = $node.attr("data-assigned-to") || "";
    var assignedToName = $node.attr("data-assigned-to-name") || "";
    var inputId = "rdStoryOwnerInput" + storyId;
    var hiddenId = "rdStoryOwnerValue" + storyId;

    $node.addClass("rd-node--editing");
    var $header = $node.find(".rd-node-header").first();
    $header.addClass("rd-node-header--edit");

    $header.find(".rd-node-title-display").replaceWith(
      $('<input type="text" class="rd-node-title-input rd-node-title form-input">')
        .attr("placeholder", "研发需求名称")
        .val(title)
    );

    destroyStoryAssigneePicker(inputId);

    var assigneeWrap = shared.cloneTemplateElement("tplRdStoryAssigneeEdit", ".rd-node-assignee-wrap");
    if (!assigneeWrap) {
      $node.removeClass("rd-node--editing");
      $header.removeClass("rd-node-header--edit");
      return;
    }
    $(assigneeWrap).find(".rd-node-assignee-input").attr("id", inputId);
    $(assigneeWrap).find(".rd-node-assignee-value").attr("id", hiddenId);
    $header.find(".rd-node-assignee-display").replaceWith(assigneeWrap);

    var confirmBtn = shared.cloneTemplateElement("tplRdNodeConfirmBtn");
    if (confirmBtn) {
      $header.find(".rd-edit-btn").replaceWith(confirmBtn);
    }

    initStoryAssigneePicker(inputId, hiddenId, assignedTo, assignedToName);
    $node.find(".rd-node-title-input, .rd-node-title").first().trigger("focus");
  }

  function exitStoryEditMode($node) {
    if (!$node.length || !$node.hasClass("rd-node--editing")) {
      return;
    }

    var title = $.trim($node.find(".rd-node-title-input").val() || $node.find(".rd-node-title").val() || "");
    var assignedTo = $.trim($node.find(".rd-node-assignee-value").val() || "");
    var assignedToName = $.trim($node.find(".rd-node-assignee-input").val() || "");

    if (title) {
      $node.attr("data-story-title", title);
    }
    $node.attr("data-assigned-to", assignedTo);
    $node.attr("data-assigned-to-name", assignedToName);

    var $header = $node.find(".rd-node-header").first();
    $header.find(".rd-node-title-input, .rd-node-title").first().replaceWith(
      $('<strong class="rd-node-title-display"></strong>').text(title || "—")
    );

    var assigneeText = assigneeDisplayText(assignedToName, assignedTo);
    $header.find(".rd-node-assignee-wrap").replaceWith(
      $('<span class="rd-node-assignee-display"></span>').text("指派: " + assigneeText)
    );

    mountNodeActions($header, "tplRdNodeActionsEdit");

    $header.removeClass("rd-node-header--edit");
    $node.removeClass("rd-node--editing");

    destroyStoryAssigneePicker("rdStoryOwnerInput" + ($node.attr("data-story-id") || "0"));
  }

  function updateTreeEmptyState() {
    var hasNodes = $("#rdTreeNodes .rd-node").length > 0;
    $("#rdTreeEmpty").toggle(!hasNodes);
  }

  function renderRdTree(stories, users, involvedProducts, mainSystemId, productProjects) {
    shared.schedulingUsers = users || shared.schedulingUsers;
    shared.involvedProducts = involvedProducts || [];
    shared.mainSystemId = mainSystemId || 0;
    shared.productProjectsMap = shared.buildProductProjectsMap(productProjects);

    var $container = $("#rdTreeNodes");
    $container.empty();

    (stories || []).forEach(function (story) {
      var node = buildStoryNode(story);
      if (!node) {
        return;
      }
      var $node = $(node);
      $node.attr("data-projects", JSON.stringify(story.projects || []));
      $node.attr("data-product-id", story.productId || "");
      applyStoryDataAttrs($node, story);
      $container.append(node);
    });

    updateTreeEmptyState();
  }

  function resetRdTree() {
    shared.productProjectsMap = {};
    shared.manualNodeSeq = 0;
    shared.taskRowSeq = 0;
    $("#rdTreeNodes").empty();
    $("#rdTreeEmpty").show();
  }

  function addDraftRdNode() {
    if (!shared.involvedProducts.length) {
      window.alert("暂无涉及系统，无法添加研发需求");
      return;
    }
    var node = buildDraftRdNode();
    if (!node) {
      return;
    }
    var $node = $(node);
    $("#rdTaskTree .rd-add-node").before(node);
    initDraftNodePickers($node);
    var defaultProductId = $node.attr("data-product-id") || "";
    if (defaultProductId) {
      tasksApi.loadProductProjects(defaultProductId, function (projects) {
        $node.attr("data-projects", JSON.stringify(projects));
      });
    }
    updateTreeEmptyState();
  }

  function removeRdNode($node) {
    $node.remove();
    updateTreeEmptyState();
  }

  function toggleRdNode($node) {
    if ($node.hasClass("rd-node--editing")) {
      return;
    }
    var $tasks = $node.find(".rd-node-tasks").first();
    var $toggle = $node.find(".rd-toggle").first();
    var $header = $node.find(".rd-node-header").first();
    var expanded = $tasks.is(":visible");

    if (expanded) {
      $tasks.hide();
      $toggle.text("▶");
      $header.addClass("rd-node-header--collapsed");
      return;
    }

    $tasks.show();
    $toggle.text("▼");
    $header.removeClass("rd-node-header--collapsed");
  }

  window.ScheduleIntegratedRd = {
    render: renderRdTree,
    reset: resetRdTree,
    exitStoryEditMode: exitStoryEditMode,
  };

  $(document).on("click", "#scheduleIntegratedModalBody .rd-toggle", function (e) {
    e.preventDefault();
    e.stopPropagation();
    toggleRdNode($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-add-node", function (e) {
    e.preventDefault();
    addDraftRdNode();
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-delete-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    removeRdNode($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-edit-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    enterStoryEditMode($(this).closest(".rd-node"));
  });

  $(document).on("click", "#scheduleIntegratedModalBody .rd-confirm-btn", function (e) {
    e.preventDefault();
    e.stopPropagation();
    exitStoryEditMode($(this).closest(".rd-node"));
  });

  $(document).on("change", "#scheduleIntegratedModalBody .rd-node-product", function () {
    var $node = $(this).closest(".rd-node");
    var productId = $(this).val();
    $node.find(".rd-node-role-badge").first().replaceWith(shared.buildRoleBadge(productId, shared.mainSystemId));
    tasksApi.loadProductProjects(productId, function (projects) {
      $node.attr("data-product-id", productId || "");
      $node.attr("data-projects", JSON.stringify(projects));
    });
  });
})(jQuery);
