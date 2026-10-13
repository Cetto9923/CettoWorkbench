/* ==========================================================================
   文件: web/static/js/schedule/maintaintaskiteration.js
   模块: 排期工作台
   职责: 维护任务弹窗「快速创建迭代」UI 交互（本轮仅前端，不接创建接口）
   ========================================================================== */
(function () {
  "use strict";

  var period = "2w";

  function $(id) {
    return document.getElementById(id);
  }

  function toast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
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
          label: name ? name + "（" + account + "）" : account,
        };
      })
      .filter(Boolean);
  }

  function teamgroupItems() {
    var items = [];
    var seen = {};
    document.querySelectorAll("#scheduleWindowTeamgroupOptions [data-value]").forEach(function (el) {
      var value = String(el.getAttribute("data-value") || "").trim();
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      items.push({
        value: value,
        label: String(el.getAttribute("data-label") || value).trim() || value,
      });
    });
    if (items.length) {
      return items;
    }
    document.querySelectorAll('#scheduleMultiselectRow [data-filter-key="groups"] .form-multiselect-checkbox').forEach(function (el) {
      var value = String(el.value || "").trim();
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      var nameEl = el.closest("label") ? el.closest("label").querySelector(".form-multiselect-name") : null;
      items.push({
        value: value,
        label: nameEl ? String(nameEl.textContent || "").trim() || value : value,
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

  function initOwnerAutocomplete() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("quickIterationOwnerInput", "quickIterationOwner", userItems(), {
      labelOnly: true,
      value: "",
    });
  }

  function initPlanAutocomplete() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("quickIterationPlanInput", "quickIterationPlan", [], {
      labelOnly: true,
      value: "",
      placeholder: "请选择产品计划",
    });
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

  function initTeamAutocomplete(options) {
    options = options || {};
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
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
    window.initAutocomplete("quickIterationTeamInput", "quickIterationTeam", items, {
      labelOnly: true,
      placeholder: "请选择",
      value: teamId,
      label: teamName,
    });
  }

  function resetQuickIterationForm() {
    period = "2w";
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
    syncPeriodRadios();
    applyPeriodDates();
    initOwnerAutocomplete();
    initPlanAutocomplete();
    initTeamAutocomplete(options);
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
        toast("功能开发中", "info");
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
