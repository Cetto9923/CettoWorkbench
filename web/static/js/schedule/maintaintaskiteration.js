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
    var team = resolveDefaultTeam(options);
    initOwnerAutocomplete(team.manager);
    initPlanAutocomplete();
    initTeamAutocomplete(team);
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
