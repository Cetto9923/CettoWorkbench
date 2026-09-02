(function ($) {
  "use strict";

  var taskTypeOptions = [
    "devel",
    "test",
    "OLtest",
    "review",
    "affair",
    "request",
    "misc",
    "codereview",
    "design",
    "study",
    "OLDebug",
    "Online",
    "Train",
    "meeting",
    "discuss",
    "ui",
  ];

  var taskTypeLabels = {
    devel: "开发",
    test: "测试",
    OLtest: "上线测试",
    review: "评审",
    affair: "事务",
    request: "需求",
    misc: "其他",
    codereview: "代码评审",
    design: "设计",
    study: "研究",
    OLDebug: "上线调试",
    Online: "上线",
    Train: "培训",
    meeting: "会议",
    discuss: "讨论",
    ui: "界面",
  };

  function cloneTemplate(id) {
    var tpl = document.getElementById(id);
    if (!tpl || !tpl.content) {
      return null;
    }
    return document.importNode(tpl.content, true);
  }

  function cloneTemplateElement(id, selector) {
    var frag = cloneTemplate(id);
    if (!frag) {
      return null;
    }
    if (selector) {
      return frag.querySelector(selector);
    }
    return frag.firstElementChild;
  }

  window.ScheduleIntegratedShared = {
    schedulingUsers: [],
    involvedProducts: [],
    mainSystemId: 0,
    productProjectsMap: {},
    productExecutionsMap: {},
    taskRowSeq: 0,
    manualNodeSeq: 0,
    currentDemandId: 0,
    currentStoryId: 0,
    currentDemandDetailURL: "",
    currentCanEditWindow: true,
    currentWindowPhase: "",
    isSchedulingDetailLoaded: false,
    deletedStoryIds: [],
    deletedTaskIds: [],
    zentaoURL: "",
    taskTypeOptions: taskTypeOptions,
    taskTypeLabels: taskTypeLabels,

    resetDeletedRecords: function () {
      this.deletedStoryIds = [];
      this.deletedTaskIds = [];
    },

    cloneTemplate: cloneTemplate,
    cloneTemplateElement: cloneTemplateElement,

    escapeHtml: function (text) {
      return $("<div>").text(text == null ? "" : String(text)).html();
    },

    formatStoryEstimate: function (value) {
      if (value === undefined || value === null || value === "") {
        return "0";
      }
      var num = Number(value);
      if (isNaN(num)) {
        return "0";
      }
      if (Math.floor(num) === num) {
        return String(num);
      }
      return String(num);
    },

    toAutocompleteItems: function (users) {
      return (users || [])
        .map(function (user) {
          return {
            value: $.trim(user.account || ""),
            label: $.trim(user.realname || "") || $.trim(user.account || ""),
          };
        })
        .filter(function (item) {
          return !!item.value;
        });
    },

    destroyStoryAssigneePicker: function (inputId) {
      if (!inputId || typeof window.destroyAutocomplete !== "function") {
        return;
      }
      window.destroyAutocomplete(inputId);
    },

    initStoryAssigneePicker: function (inputId, hiddenId, assignedTo, assignedToName) {
      if (typeof window.initAutocomplete !== "function" || !inputId || !hiddenId) {
        return;
      }
      this.destroyStoryAssigneePicker(inputId);
      window.initAutocomplete(inputId, hiddenId, this.toAutocompleteItems(this.schedulingUsers), {
        placeholder: "输入姓名或工号搜索",
        value: assignedTo || "",
        label: assignedToName || "",
      });
    },

    taskTypeLabel: function (type) {
      var key = $.trim(type || "");
      return taskTypeLabels[key] || key || "—";
    },

    taskTypeClass: function (type) {
      var key = $.trim(type || "");
      if (key === "devel" || key === "test" || key === "Online") {
        return "rd-task-type-label--" + key;
      }
      return "rd-task-type-label--default";
    },

    syncIntegratedDateInputState: function (input) {
      if (!input) {
        return;
      }
      var $input = $(input);
      var date = $.trim($input.val() || "");
      $input.toggleClass("has-value", date !== "");
    },

    buildRoleBadge: function (productId, mainSystemId) {
      var badge = document.createElement("span");
      badge.className = "rd-node-role-badge";
      var pid = String(productId || "");
      var mainId = String(mainSystemId || "");
      if (pid && pid === mainId) {
        badge.classList.add("rd-node-role-badge--main");
        badge.textContent = "主";
      } else if (pid) {
        badge.classList.add("rd-node-role-badge--cooperate");
        badge.textContent = "配";
      } else {
        badge.classList.add("rd-node-role-badge--none");
        badge.textContent = "—";
      }
      return badge;
    },

    fillProductSelect: function ($select, products, selectedId) {
      var selected = String(selectedId || "");
      $select.empty();
      $("<option></option>").val("").text("请选择系统").appendTo($select);
      (products || []).forEach(function (product) {
        var id = String(product.id || "");
        if (!id) {
          return;
        }
        $("<option></option>").val(id).text(product.name || id).appendTo($select);
      });
      if (selected) {
        $select.val(selected);
      }
    },

    buildProductProjectsMap: function (productProjects) {
      var map = {};
      if (!productProjects) {
        return map;
      }
      Object.keys(productProjects).forEach(function (key) {
        map[String(key)] = productProjects[key] || [];
      });
      return map;
    },

    buildProductExecutionsMap: function (projectExecutions) {
      var map = {};
      if (!projectExecutions) {
        return map;
      }
      Object.keys(projectExecutions).forEach(function (key) {
        map[String(key)] = projectExecutions[key] || [];
      });
      return map;
    },

    executionTypeLabel: function (type) {
      var key = $.trim(type || "");
      if (key === "sprint") {
        return "迭代";
      }
      if (key === "stage") {
        return "阶段";
      }
      if (key === "kanban") {
        return "看板";
      }
      return "";
    },

    formatExecutionOptionLabel: function (execution) {
      var name = $.trim(execution.name || "") || String(execution.id || "");
      var typeLabel = this.executionTypeLabel(execution.type);
      if (typeLabel) {
        return "[" + typeLabel + "] " + name;
      }
      return name;
    },

    formatInvolvedSystemsHint: function (products, fallbackName) {
      var names = (products || [])
        .map(function (item) {
          return $.trim(item.name || "");
        })
        .filter(function (name) {
          return !!name;
        });
      if (names.length) {
        return names.join("、");
      }
      return $.trim(fallbackName || "") || "—";
    },
  };
})(jQuery);
