/* ==========================================================================
   文件: web/static/js/schedule/maintaintaskiteration.js
   模块: 排期工作台
   职责: 维护任务弹窗「快速创建迭代」UI 交互与创建并选用
   ========================================================================== */
(function () {
  "use strict";

  var period = "2w";
  var context = {
    productId: 0,
    storyId: 0,
  };

  function $(id) {
    return document.getElementById(id);
  }

  function toast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function shanghaiToday() {
    return new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Shanghai" }).format(new Date());
  }

  function addDays(dateStr, days) {
    var d = new Date(dateStr + "T00:00:00+08:00");
    d.setUTCDate(d.getUTCDate() + days);
    return new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Shanghai" }).format(d);
  }

  function syncPeriodRadios() {
    document.querySelectorAll("#quickIteration input[name='quickIterationPeriod']").forEach(function (input) {
      input.checked = input.value === period;
    });
  }

  function normalizeYMD(value) {
    var raw = String(value || "").trim();
    if (!raw) {
      return "";
    }
    var m = raw.match(/^(\d{4})-(\d{1,2})-(\d{1,2})/);
    if (!m) {
      return "";
    }
    var y = m[1];
    var mo = ("0" + m[2]).slice(-2);
    var d = ("0" + m[3]).slice(-2);
    return y + "-" + mo + "-" + d;
  }

  function syncDateDisplay(input) {
    if (!input) {
      return;
    }
    var wrap = input.closest(".task-iteration-date");
    if (!wrap) {
      return;
    }
    var v = normalizeYMD(input.value);
    wrap.setAttribute("data-date", v || "y-m-d");
    wrap.classList.toggle("is-empty", !v);
  }

  function setDateValue(input, ymd) {
    if (!input) {
      return;
    }
    input.value = normalizeYMD(ymd);
    syncDateDisplay(input);
  }

  function applyPeriodDates() {
    var begin = shanghaiToday();
    var end = addDays(begin, period === "4w" ? 27 : 13);
    setDateValue($("quickIterationBegin"), begin);
    setDateValue($("quickIterationEnd"), end);
  }

  function userItems() {
    var shared = window.ScheduleIntegratedShared;
    var users = (shared && shared.schedulingUsers) || [];
    return users
      .map(function (u) {
        var account = String(u.account || "").trim();
        if (!account) {
          return null;
        }
        var name = String(u.realname || "").trim();
        return {
          value: account,
          label: name || account,
        };
      })
      .filter(Boolean);
  }

  function teamgroupItems() {
    var items = [];
    var seen = {};
    document.querySelectorAll("#taskModalTeamgroupOptions [data-value]").forEach(function (el) {
      var value = String(el.getAttribute("data-value") || "").trim();
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      items.push({
        value: value,
        label: String(el.getAttribute("data-label") || value).trim() || value,
        manager: String(el.getAttribute("data-manager") || "").trim(),
      });
    });
    return items;
  }

  function findItemLabel(items, value) {
    value = String(value || "").trim();
    if (!value) {
      return "";
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].value) === value) {
        return items[i].label || value;
      }
    }
    return "";
  }

  function findItemValueByLabel(items, label) {
    label = String(label || "").trim();
    if (!label || label === "—") {
      return "";
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].label || "").trim() === label) {
        return String(items[i].value);
      }
    }
    return "";
  }

  function findItemByValue(items, value) {
    value = String(value || "").trim();
    if (!value) {
      return null;
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].value) === value) {
        return items[i];
      }
    }
    return null;
  }

  function firstManagerAccount(manager) {
    var raw = String(manager || "").trim();
    if (!raw) {
      return "";
    }
    return raw.split(",")[0].trim();
  }

  function resolveTeamFromStory(storyId) {
    var id = String(storyId || "").trim();
    if (!id) {
      return "";
    }
    var btn = document.querySelector('.js-open-task-modal[data-story-id="' + id + '"]');
    if (!btn) {
      return "";
    }
    var row = btn.closest("tr");
    if (!row) {
      return "";
    }
    var cell = row.querySelector(".schedule-col-group");
    return cell ? String(cell.textContent || "").trim() : "";
  }

  // 解析弹窗打开时默认选中的敏捷团队；仅用于首次默认，切换团队不联动负责人。
  function resolveDefaultTeam(options) {
    options = options || {};
    var items = teamgroupItems();
    var teamId = String(options.teamgroupId || "").trim();
    if (teamId === "0") {
      teamId = "";
    }
    var teamName = String(options.teamgroupName || "").trim();
    if (!teamId) {
      teamId = findItemValueByLabel(items, teamName || resolveTeamFromStory(options.storyId));
    }
    if (!teamName) {
      teamName = findItemLabel(items, teamId) || resolveTeamFromStory(options.storyId);
    }
    var item = findItemByValue(items, teamId);
    return {
      items: items,
      teamId: teamId,
      teamName: teamName,
      manager: item ? firstManagerAccount(item.manager) : "",
    };
  }

  function initOwnerAutocomplete(ownerAccount) {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    var users = userItems();
    var owner = String(ownerAccount || "").trim();
    window.initAutocomplete("quickIterationOwnerInput", "quickIterationOwner", users, {
      labelOnly: true,
      value: owner,
      label: findItemLabel(users, owner),
    });
  }

  function planItems(plans) {
    return (plans || [])
      .map(function (p) {
        var id = String((p && p.id) || "").trim();
        if (!id || id === "0") {
          return null;
        }
        var title = String((p && p.title) || "").trim();
        return {
          value: id,
          label: title || id,
        };
      })
      .filter(Boolean);
  }

  function initPlanAutocomplete(plans) {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("quickIterationPlanInput", "quickIterationPlan", planItems(plans), {
      labelOnly: true,
      value: "",
      placeholder: "请选择产品计划",
    });
  }

  function initTeamAutocomplete(team) {
    team = team || {};
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("quickIterationTeamInput", "quickIterationTeam", team.items || teamgroupItems(), {
      labelOnly: true,
      placeholder: "请选择",
      value: team.teamId || "",
      label: team.teamName || "",
    });
  }

  function resetQuickIterationForm() {
    period = "2w";
    context.productId = 0;
    context.storyId = 0;
    setCreatingState(false);
    var section = $("quickIteration");
    if (section) {
      section.open = false;
    }
    var nameEl = $("quickIterationName");
    if (nameEl) {
      nameEl.value = "";
    }
    if (typeof window.destroyAutocomplete === "function") {
      window.destroyAutocomplete("quickIterationOwnerInput");
      window.destroyAutocomplete("quickIterationPlanInput");
      window.destroyAutocomplete("quickIterationTeamInput");
    } else {
      ["quickIterationOwnerInput", "quickIterationOwner", "quickIterationPlanInput", "quickIterationPlan", "quickIterationTeamInput", "quickIterationTeam"].forEach(function (id) {
        var el = $(id);
        if (el) {
          el.value = "";
        }
      });
    }
    syncPeriodRadios();
    applyPeriodDates();
  }

  function refreshQuickIterationUI(options) {
    options = options || {};
    context.productId = parsePositiveInt(options.productId);
    context.storyId = parsePositiveInt(options.storyId);
    syncPeriodRadios();
    applyPeriodDates();
    var team = resolveDefaultTeam(options);
    initOwnerAutocomplete(team.manager);
    initPlanAutocomplete(options.plans);
    initTeamAutocomplete(team);
  }

  function resolveStoryId() {
    if (context.storyId) {
      return context.storyId;
    }
    if (window.ScheduleMaintainTask && typeof window.ScheduleMaintainTask.getStoryId === "function") {
      return parsePositiveInt(window.ScheduleMaintainTask.getStoryId());
    }
    return 0;
  }

  function resolveProjectId() {
    if (window.ScheduleMaintainTask && typeof window.ScheduleMaintainTask.getProjectId === "function") {
      return parsePositiveInt(window.ScheduleMaintainTask.getProjectId());
    }
    return parsePositiveInt(($("taskModalProjectSelect") || {}).value);
  }

  function resolveProductId() {
    if (context.productId) {
      return context.productId;
    }
    if (window.ScheduleMaintainTask && typeof window.ScheduleMaintainTask.getProductId === "function") {
      return parsePositiveInt(window.ScheduleMaintainTask.getProductId());
    }
    return 0;
  }

  function collectCreatePayload() {
    return {
      name: String(($("quickIterationName") || {}).value || "").trim(),
      begin: normalizeYMD(($("quickIterationBegin") || {}).value),
      end: normalizeYMD(($("quickIterationEnd") || {}).value),
      productId: resolveProductId(),
      planId: parsePositiveInt(($("quickIterationPlan") || {}).value),
      pm: String(($("quickIterationOwner") || {}).value || "").trim(),
      scrumTeam: parsePositiveInt(($("quickIterationTeam") || {}).value),
      storyId: resolveStoryId(),
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
    if (!payload.storyId) {
      return "缺少当前研发需求";
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

  var CREATE_BTN_IDLE_HTML = "创建并选用";
  var CREATE_BTN_BUSY_HTML = '<i class="fas fa-spinner fa-spin" aria-hidden="true"></i>创建中…';

  function setCreatingState(busy) {
    var section = $("quickIteration");
    var createBtn = $("quickIterationCreate");

    if (section) {
      section.classList.toggle("is-creating", !!busy);
    }
    if (createBtn) {
      createBtn.disabled = !!busy;
      createBtn.classList.toggle("is-creating", !!busy);
      createBtn.setAttribute("aria-busy", busy ? "true" : "false");
      createBtn.innerHTML = busy ? CREATE_BTN_BUSY_HTML : CREATE_BTN_IDLE_HTML;
    }

    if (section) {
      section.querySelectorAll("input, button, textarea, select").forEach(function (el) {
        if (el.id === "quickIterationCreate") {
          return;
        }
        if (busy) {
          if (!el.hasAttribute("data-prev-disabled")) {
            el.setAttribute("data-prev-disabled", el.disabled ? "1" : "0");
          }
          el.disabled = true;
        } else if (el.hasAttribute("data-prev-disabled")) {
          el.disabled = el.getAttribute("data-prev-disabled") === "1";
          el.removeAttribute("data-prev-disabled");
        }
      });
    }
  }

  function createAndSelect() {
    var projectId = resolveProjectId();
    var payload = collectCreatePayload();
    var errMsg = validateCreatePayload(payload, projectId);
    if (errMsg) {
      toast(errMsg, "error");
      return;
    }

    var createBtn = $("quickIterationCreate");
    if (createBtn && createBtn.classList.contains("is-creating")) {
      return;
    }

    setCreatingState(true);

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
        var section = $("quickIteration");
        if (section) {
          section.open = false;
        }
        if (window.ScheduleMaintainTask && typeof window.ScheduleMaintainTask.selectNewExecution === "function") {
          window.ScheduleMaintainTask.selectNewExecution(executionId);
        }
      })
      .catch(function () {
        toast("创建失败", "error");
      })
      .finally(function () {
        setCreatingState(false);
      });
  }

  function bindEvents() {
    var section = $("quickIteration");
    if (!section || section.getAttribute("data-bound") === "1") {
      return;
    }
    section.setAttribute("data-bound", "1");

    section.querySelectorAll("input[name='quickIterationPeriod']").forEach(function (input) {
      input.addEventListener("change", function () {
        if (!input.checked) {
          return;
        }
        period = input.value || "2w";
        applyPeriodDates();
      });
    });

    ["quickIterationBegin", "quickIterationEnd"].forEach(function (id) {
      var el = $(id);
      if (!el) {
        return;
      }
      el.addEventListener("change", function () {
        syncDateDisplay(el);
      });
      el.addEventListener("input", function () {
        syncDateDisplay(el);
      });
      syncDateDisplay(el);
    });

    var createBtn = $("quickIterationCreate");
    if (createBtn) {
      createBtn.addEventListener("click", function () {
        createAndSelect();
      });
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    resetQuickIterationForm();
  });

  window.ScheduleQuickIteration = {
    reset: resetQuickIterationForm,
    refresh: refreshQuickIterationUI,
  };
})();
