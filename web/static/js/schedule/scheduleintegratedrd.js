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
    $node.attr("data-product-name", $.trim(story.productName || ""));
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
    $node.attr("data-new", "true");
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

  function initDraftNodePickers($node) {
    var inputId = $node.find(".rd-node-assignee-input").attr("id");
    var hiddenId = $node.find(".rd-node-assignee-value").attr("id");
    shared.initStoryAssigneePicker(inputId, hiddenId, "", "");
  }

  function assigneeDisplayText(assignedToName, assignedTo) {
    return $.trim(assignedToName || "") || $.trim(assignedTo || "") || "待分配";
  }

  function resolveProductName(productId) {
    var id = String(productId || "");
    if (!id) {
      return "—";
    }
    var name = "";
    (shared.involvedProducts || []).some(function (product) {
      if (String(product.id || "") === id) {
        name = $.trim(product.name || "") || id;
        return true;
      }
      return false;
    });
    return name || id;
  }

  function buildProductSelect(selectedId) {
    var $select = $('<select class="rd-node-product form-select rd-node-product-select"></select>');
    shared.fillProductSelect($select, shared.involvedProducts, selectedId);
    return $select;
  }

  function applyProductChange($node, productId) {
    var name = resolveProductName(productId);
    $node.attr("data-product-id", productId || "");
    $node.attr("data-product-name", name === "—" ? "" : name);
    $node.find(".rd-node-role-badge").first().replaceWith(shared.buildRoleBadge(productId, shared.mainSystemId));
    if (!productId) {
      $node.attr("data-projects", JSON.stringify([]));
      return;
    }
    tasksApi.loadProductProjects(productId, function (projects) {
      $node.attr("data-projects", JSON.stringify(projects));
    });
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
    var productId = $node.attr("data-product-id") || "";
    var inputId = "rdStoryOwnerInput" + storyId;
    var hiddenId = "rdStoryOwnerValue" + storyId;

    $node.addClass("rd-node--editing");
    var $header = $node.find(".rd-node-header").first();
    $header.addClass("rd-node-header--story-edit");

    $header.find(".rd-node-product-name").replaceWith(buildProductSelect(productId));
    $header.find(".rd-node-role-badge").replaceWith(shared.buildRoleBadge(productId, shared.mainSystemId));

    $header.find(".rd-node-title-display").replaceWith(
      $('<input type="text" class="rd-node-title-input rd-node-title form-input">')
        .attr("placeholder", "研发需求名称")
        .val(title)
    );

    shared.destroyStoryAssigneePicker(inputId);

    var assigneeWrap = shared.cloneTemplateElement("tplRdStoryAssigneeEdit", ".rd-node-assignee-wrap");
    if (!assigneeWrap) {
      $node.removeClass("rd-node--editing");
      $header.removeClass("rd-node-header--story-edit");
      return;
    }
    $(assigneeWrap).find(".rd-node-assignee-input").attr("id", inputId);
    $(assigneeWrap).find(".rd-node-assignee-value").attr("id", hiddenId);
    $header.find(".rd-node-assignee-display").replaceWith(assigneeWrap);

    var confirmBtn = shared.cloneTemplateElement("tplRdNodeConfirmBtn");
    if (confirmBtn) {
      $header.find(".rd-edit-btn").replaceWith(confirmBtn);
    }

    shared.initStoryAssigneePicker(inputId, hiddenId, assignedTo, assignedToName);
    $node.find(".rd-node-title-input, .rd-node-title").first().trigger("focus");
  }

  function exitStoryEditMode($node) {
    if (!$node.length || !$node.hasClass("rd-node--editing")) {
      return;
    }

    var $header = $node.find(".rd-node-header").first();
    var title = $.trim($node.find(".rd-node-title-input").val() || $node.find(".rd-node-title").val() || "");
    var assignedTo = $.trim($node.find(".rd-node-assignee-value").val() || "");
    var assignedToName = $.trim($node.find(".rd-node-assignee-input").val() || "");
    var productId = $.trim($header.find(".rd-node-product").val() || $node.attr("data-product-id") || "");
    var productName = resolveProductName(productId);

    if (title) {
      $node.attr("data-story-title", title);
    }
    $node.attr("data-assigned-to", assignedTo);
    $node.attr("data-assigned-to-name", assignedToName);
    $node.attr("data-product-id", productId);
    $node.attr("data-product-name", productName === "—" ? "" : productName);

    $header.find(".rd-node-product").replaceWith(
      $('<span class="rd-node-product-name"></span>').text(productName || "—")
    );
    $header.find(".rd-node-role-badge").replaceWith(shared.buildRoleBadge(productId, shared.mainSystemId));

    $header.find(".rd-node-title-input, .rd-node-title").first().replaceWith(
      $('<strong class="rd-node-title-display"></strong>').text(title || "—")
    );

    var assigneeText = assigneeDisplayText(assignedToName, assignedTo);
    $header.find(".rd-node-assignee-wrap").replaceWith(
      $('<span class="rd-node-assignee-display"></span>').text("指派: " + assigneeText)
    );

    mountNodeActions($header, "tplRdNodeActionsEdit");

    $header.removeClass("rd-node-header--story-edit");
    $node.removeClass("rd-node--editing");

    shared.destroyStoryAssigneePicker("rdStoryOwnerInput" + ($node.attr("data-story-id") || "0"));
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
    shared.resetDeletedRecords();
    $("#rdTreeNodes").empty();
    $("#rdTreeEmpty").show();
  }

  function addDraftRdNode() {
    if (!shared.isSchedulingDetailLoaded) {
      if (typeof window.showToast === "function") {
        window.showToast("业需详情加载中，请稍后再试", "error");
      } else {
        window.alert("业需详情加载中，请稍后再试");
      }
      return;
    }
    if (!shared.involvedProducts.length) {
      var detailUrl = $.trim(shared.currentDemandDetailURL || "");
      var confirmed = window.confirm("暂无涉及系统，无法添加研发需求，去澄清添加系统。");
      if (confirmed && detailUrl) {
        window.location.href = detailUrl;
      }
      return;
    }
    var node = buildDraftRdNode();
    if (!node) {
      return;
    }
    var $node = $(node);
    $("#rdTreeNodes").append(node);
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
    var storyId = parseInt(String($node.attr("data-story-id") || ""), 10);
    if (!isNaN(storyId) && storyId > 0) {
      shared.deletedStoryIds.push(storyId);
    }
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
    applyProductChange($(this).closest(".rd-node"), $(this).val());
  });
})(jQuery);
