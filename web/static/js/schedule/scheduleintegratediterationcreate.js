/* ==========================================================================
   文件: web/static/js/schedule/scheduleintegratediterationcreate.js
   模块: 排期工作台
   职责: 一体化排期「快速创建迭代」创建并选用（复用 /schedule/:projectId/executions）
   ========================================================================== */
(function ($) {
  "use strict";

  var CREATE_BTN_IDLE_HTML = "创建并选用";
  var CREATE_BTN_BUSY_HTML = '<i class="fas fa-spinner fa-spin" aria-hidden="true"></i>创建中…';

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function toast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    } else {
      window.alert(message);
    }
  }

  function setCreatingState($form, busy) {
    if (!$form || !$form.length) {
      return;
    }
    var createBtn = $form.find(".rd-quick-iteration-create")[0];
    $form.toggleClass("is-creating", !!busy);
    if (createBtn) {
      createBtn.disabled = !!busy;
      createBtn.classList.toggle("is-creating", !!busy);
      createBtn.setAttribute("aria-busy", busy ? "true" : "false");
      createBtn.innerHTML = busy ? CREATE_BTN_BUSY_HTML : CREATE_BTN_IDLE_HTML;
    }
    $form.find("input, button, textarea, select").each(function () {
      if ($(this).hasClass("rd-quick-iteration-create")) {
        return;
      }
      if (busy) {
        if (!this.hasAttribute("data-prev-disabled")) {
          this.setAttribute("data-prev-disabled", this.disabled ? "1" : "0");
        }
        this.disabled = true;
      } else if (this.hasAttribute("data-prev-disabled")) {
        this.disabled = this.getAttribute("data-prev-disabled") === "1";
        this.removeAttribute("data-prev-disabled");
      }
    });
  }

  function resolveNodeStoryId($node) {
    // 未保存（草稿）研需：不传 storyId，后端跳过迭代-需求关联
    if ($node.hasClass("rd-node--draft") || $node.attr("data-new") === "true") {
      return 0;
    }
    return parsePositiveInt($node.attr("data-story-id"));
  }

  function resolveNodeProjectId($node) {
    var tasksApi = window.ScheduleIntegratedTasks;
    if (tasksApi && typeof tasksApi.readSectionProject === "function") {
      return parsePositiveInt(tasksApi.readSectionProject($node).projectId);
    }
    return parsePositiveInt($node.find(".rd-task-project-select").val());
  }

  function resolveNodeProductId($node) {
    var iter = window.ScheduleIntegratedIteration;
    if (iter && typeof iter.resolveNodeProductId === "function") {
      return parsePositiveInt(iter.resolveNodeProductId($node));
    }
    return parsePositiveInt($node.attr("data-product-id") || $node.find(".rd-node-product").val());
  }

  function normalizeYMD(value) {
    var iter = window.ScheduleIntegratedIteration;
    if (iter && typeof iter.normalizeYMD === "function") {
      return iter.normalizeYMD(value);
    }
    var raw = String(value || "").trim();
    var m = raw.match(/^(\d{4})-(\d{1,2})-(\d{1,2})/);
    if (!m) {
      return "";
    }
    return m[1] + "-" + ("0" + m[2]).slice(-2) + "-" + ("0" + m[3]).slice(-2);
  }

  function ensureFieldIds($form) {
    var iter = window.ScheduleIntegratedIteration;
    if (iter && typeof iter.ensureFieldIds === "function") {
      return iter.ensureFieldIds($form);
    }
    return null;
  }

  function collectCreatePayload($form, $node, ids) {
    return {
      name: String($form.find(".rd-quick-iteration-name").val() || "").trim(),
      begin: normalizeYMD($form.find(".rd-quick-iteration-begin").val()),
      end: normalizeYMD($form.find(".rd-quick-iteration-end").val()),
      productId: resolveNodeProductId($node),
      planId: parsePositiveInt($("#" + ids.planHidden).val()),
      pm: String($("#" + ids.ownerHidden).val() || "").trim(),
      scrumTeam: parsePositiveInt($("#" + ids.teamHidden).val()),
      storyId: resolveNodeStoryId($node),
    };
  }

  function validateCreatePayload(payload, projectId) {
    if (!projectId) {
      return "请先选择项目";
    }
    if (!payload.name) {
      return "请填写迭代名称";
    }
    if (!payload.pm) {
      return "请选择执行负责人";
    }
    if (!payload.begin || !payload.end) {
      return "请选择计划起止日期";
    }
    if (payload.begin > payload.end) {
      return "结束日期不能早于开始日期";
    }
    if (!payload.productId) {
      return "缺少需求所属产品";
    }
    return "";
  }

  function postJSON(url, body) {
    if (typeof window.scheduleFetch === "function") {
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
    return fetch(url, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-Requested-With": "XMLHttpRequest",
      },
      body: JSON.stringify(body || {}),
    }).then(function (resp) {
      return resp.json().then(function (data) {
        return { ok: resp.ok, status: resp.status, data: data };
      });
    });
  }

  function createAndSelect($form, $node) {
    var ids = ensureFieldIds($form);
    if (!ids) {
      toast("表单未初始化", "error");
      return;
    }
    var projectId = resolveNodeProjectId($node);
    var payload = collectCreatePayload($form, $node, ids);
    var errMsg = validateCreatePayload(payload, projectId);
    if (errMsg) {
      toast(errMsg, "error");
      return;
    }
    if ($form.hasClass("is-creating")) {
      return;
    }

    setCreatingState($form, true);
    postJSON("/schedule/" + encodeURIComponent(projectId) + "/executions", payload)
      .then(function (result) {
        var data = (result && result.data) || {};
        if (!data.success) {
          toast(data.message || data.error || "创建失败", "error");
          return;
        }
        var execution = data.execution || {};
        var executionId = parsePositiveInt(execution.id);
        if (!executionId) {
          toast("创建成功但未返回执行 ID", "error");
          return;
        }
        toast(data.message || "创建成功", "success");
        $form.prop("open", false);
        var tasksApi = window.ScheduleIntegratedTasks;
        if (tasksApi && typeof tasksApi.selectNewExecution === "function") {
          tasksApi.selectNewExecution($node, executionId, execution.name || payload.name);
        }
      })
      .catch(function () {
        toast("创建失败", "error");
      })
      .finally(function () {
        setCreatingState($form, false);
      });
  }

  $(document).on("click", "#scheduleIntegratedModalBody .rd-quick-iteration-create", function (e) {
    e.preventDefault();
    var $form = $(this).closest(".rd-quick-iteration");
    var $node = $(this).closest(".rd-node");
    if (!$form.length || !$node.length) {
      return;
    }
    createAndSelect($form, $node);
  });
})(jQuery);
