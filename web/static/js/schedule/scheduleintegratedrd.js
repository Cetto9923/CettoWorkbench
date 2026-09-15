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
    $header.find(".rd-node-id").text(story.id || "—");
    $header.find(".rd-node-role-badge").replaceWith(shared.buildRoleBadge(story.productId, shared.mainSystemId));
    $header.find(".rd-node-product-name").text(story.productName || "—");
    $header.find(".rd-node-title-display").text(story.title || "—");
    $header.find(".rd-node-assignee-display").text("指派: " + assignee);
    mountNodeActions($header, "tplRdNodeActionsEdit");
  }

  function applyStoryDataAttrs($node, story) {
    $node.attr("data-story-title", $.trim(story.title || ""));
    $node.attr("data-module-id", story.moduleId || 0);
    $node.attr("data-plan-id", story.planId || "");
    $node.attr("data-plan-name", $.trim(story.planName || ""));
    $node.attr("data-story-type", $.trim(story.type || "story") || "story");
    $node.attr("data-pri", story.pri || 3);
    $node.attr("data-estimate", story.estimate || 0);
    $node.attr("data-spec", story.spec || "");
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
    shared.fillProductSelect($node.find(".rd-node-product"), shared.involvedProducts, defaultProductId);
    applyDraftDefaults($node, defaultProductId);

    var inputId = "rdNodeOwnerInput" + nodeId;
    var hiddenId = "rdNodeOwnerValue" + nodeId;
    $node.find(".rd-node-assignee-input").attr("id", inputId);
    $node.find(".rd-node-assignee-value").attr("id", hiddenId);
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
    shared.initStoryAssigneePicker(
      inputId,
      hiddenId,
      $node.attr("data-assigned-to") || "",
      $node.attr("data-assigned-to-name") || ""
    );
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
  function storyDefaultFor(productId) {
    return shared.draftStoryDefaults[String(productId || "")] || {};
  }

  function planFor(productId) {
    var windowId = String($("#scheduleIntegratedWindowSelect").val() || "");
    var found = null;
    (shared.windowProductPlans || []).some(function (plan) {
      if (String(plan.windowId || "") === windowId && String(plan.productId || "") === String(productId || "")) {
        found = plan;
        return true;
      }
      return false;
    });
    return found;
  }

  function applyDraftDefaults($node, productId) {
    var defaults = storyDefaultFor(productId);
    var hasProductDefaults = !!defaults.productId;
    var plan = planFor(productId) || defaults;
    var title = $.trim(hasProductDefaults ? defaults.title : shared.draftStoryTitle || "");
    var spec = hasProductDefaults ? String(defaults.spec || "") : "";
    var assignedTo = $.trim(hasProductDefaults ? defaults.assignedTo : shared.draftStoryAssignee || "");
    var assignedToName = $.trim(hasProductDefaults ? defaults.assignedToName : shared.draftStoryAssigneeName || "");
    var pri = defaults.pri || 3;
    var estimate = defaults.estimate || 0;
    var type = $.trim(defaults.type || "story") || "story";
    $node.attr("data-story-title", title);
    $node.attr("data-module-id", "0");
    $node.attr("data-plan-id", plan && plan.planId ? plan.planId : "");
    $node.attr("data-plan-name", plan && plan.planName ? $.trim(plan.planName) : "");
    $node.attr("data-story-type", type);
    $node.attr("data-pri", pri);
    $node.attr("data-estimate", estimate);
    $node.attr("data-spec", spec);
    $node.attr("data-assigned-to", assignedTo);
    $node.attr("data-assigned-to-name", assignedToName);
    shared.fillStoryPlanSelect($node.find(".rd-node-plan-select"), productId, plan && plan.planId ? plan.planId : "");
    shared.fillStoryTypeSelect($node.find(".rd-node-type"), type);
    $node.find(".rd-node-title").val(title);
    $node.find(".rd-node-spec").val(spec);
    $node.find(".rd-node-pri").val(String(pri));
    $node.find(".rd-node-estimate").val(estimate ? estimate : "");
  }

  function refreshDraftPlanLabels() {
    $("#rdTreeNodes .rd-node--draft").each(function () {
      var $node = $(this);
      var plan = planFor($node.attr("data-product-id"));
      var planId = plan && plan.planId ? plan.planId : "";
      shared.fillStoryPlanSelect($node.find(".rd-node-plan-select"), $node.attr("data-product-id"), planId);
      $node.attr("data-plan-id", planId);
      $node.attr("data-plan-name", plan && plan.planName ? $.trim(plan.planName) : "");
    });
  }

  function defaultsFromResponse(data) {
    data = data || {};
    return {
      title: data.name,
      assignee: data.bra,
      assigneeName: data.braName,
      storyDefaults: data.storyDefaults,
      windowProductPlans: data.windowProductPlans,
      productPlans: data.productPlans,
    };
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
    shared.fillStoryPlanSelect($node.find(".rd-node-plan-select"), productId, "");
    $node.attr("data-plan-id", "");
    $node.attr("data-plan-name", "");
    if ($node.hasClass("rd-node--draft")) {
      applyDraftDefaults($node, productId);
      initDraftNodePickers($node);
    }
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

  function renderRdTree(stories, users, involvedProducts, mainSystemId, productProjects, draftDefaults) {
    shared.schedulingUsers = users || shared.schedulingUsers;
    shared.involvedProducts = involvedProducts || [];
    shared.mainSystemId = mainSystemId || 0;
    shared.draftStoryTitle = $.trim((draftDefaults && draftDefaults.title) || "");
    shared.draftStoryAssignee = $.trim((draftDefaults && draftDefaults.assignee) || "");
    shared.draftStoryAssigneeName = $.trim((draftDefaults && draftDefaults.assigneeName) || "");
    shared.draftStoryDefaults = {};
    (draftDefaults && draftDefaults.storyDefaults || []).forEach(function (item) {
      if (item && item.productId) {
        shared.draftStoryDefaults[String(item.productId)] = item;
      }
    });
    shared.windowProductPlans = (draftDefaults && draftDefaults.windowProductPlans) || [];
    shared.productPlans = (draftDefaults && draftDefaults.productPlans) || {};
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
    shared.draftStoryTitle = "";
    shared.draftStoryAssignee = "";
    shared.draftStoryAssigneeName = "";
    shared.draftStoryDefaults = {};
    shared.windowProductPlans = [];
    shared.productPlans = {};
    shared.productProjectsMap = {};
    shared.manualNodeSeq = 0;
    shared.taskRowSeq = 0;
    shared.resetDeletedRecords();
    $("#rdTreeNodes").empty();
    $("#rdTreeEmpty").show();
  }

  function addDraftRdNode() {
    if (!shared.isSchedulingDetailLoaded) {
      window.showToast("业需详情加载中，请稍后再试", "error");
      return;
    }
    if (!shared.involvedProducts.length) {
      var detailUrl = $.trim(shared.currentDemandDetailURL || "");
      var confirmed = window.confirm("暂无涉及系统，无法添加研发需求，去澄清添加系统。");
      if (confirmed && detailUrl) {
        window.open(detailUrl, "_blank");
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
    defaultsFromResponse: defaultsFromResponse,
    refreshPlansForWindow: refreshDraftPlanLabels,
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

  $(document).on("input", "#scheduleIntegratedModalBody .rd-node-spec", function () {
    $(this).closest(".rd-node").attr("data-spec", $(this).val() || "");
  }).on("input", "#scheduleIntegratedModalBody .rd-node-title", function () {
    $(this).closest(".rd-node").attr("data-story-title", $(this).val() || "");
  }).on("change", "#scheduleIntegratedModalBody .rd-node-type", function () {
    $(this).closest(".rd-node").attr("data-story-type", $(this).val() || "story");
  }).on("change", "#scheduleIntegratedModalBody .rd-node-pri", function () {
    $(this).closest(".rd-node").attr("data-pri", $(this).val() || "3");
  }).on("input", "#scheduleIntegratedModalBody .rd-node-estimate", function () {
    $(this).closest(".rd-node").attr("data-estimate", $(this).val() || "0");
  });

  $(document).on("change", "#scheduleIntegratedModalBody .rd-node-plan-select", function () {
    var $select = $(this);
    var $option = $select.find("option:selected");
    $select.closest(".rd-node")
      .attr("data-plan-id", $select.val() || "")
      .attr("data-plan-name", $.trim($option.text() || ""));
  });

  $(document).on("change", "#scheduleIntegratedWindowSelect", refreshDraftPlanLabels);
})(jQuery);
