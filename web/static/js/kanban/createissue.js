/* =============================================================================
   文件: web/static/js/kanban/createissue.js
   模块: 工作看板
   职责: 提问题弹窗开关；所属项目/指派给/提出人走 GET /kanban/createissue。
============================================================================= */
(function () {
  "use strict";

  var OPTIONS_URL = "/kanban/createissue";

  var bound = false;
  var loadSeq = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function activeTeamgroupID() {
    var active = document.querySelector(
      ".po-board [data-teamgroup-chips] .chip.active[data-teamgroup-id]"
    );
    return active
      ? String(active.getAttribute("data-teamgroup-id") || "").trim()
      : "";
  }

  function activeAccount() {
    var group = document.querySelector(
      ".po-board .people-group:not(.is-hidden)"
    );
    if (!group) return "";
    var person = group.querySelector(".person.active[data-account]");
    return person
      ? String(person.getAttribute("data-account") || "").trim()
      : "";
  }

  function normalizeOptions(field) {
    field = field || {};
    var raw = Array.isArray(field.options) ? field.options : [];
    var options = [];
    for (var i = 0; i < raw.length; i++) {
      var value = String((raw[i] && raw[i].value) || "").trim();
      if (!value) continue;
      var label = String((raw[i] && raw[i].label) || "").trim() || value;
      options.push({ value: value, label: label });
    }
    return {
      options: options,
      value: String(field.value || "").trim(),
    };
  }

  function findLabel(options, value) {
    var v = String(value || "").trim();
    if (!v) return "";
    for (var i = 0; i < options.length; i++) {
      if (options[i].value === v) return options[i].label;
    }
    return v;
  }

  function destroySearchSelects() {
    if (typeof window.destroyAutocomplete !== "function") {
      return;
    }
    window.destroyAutocomplete("kanbanIssueProjectInput");
    window.destroyAutocomplete("kanbanIssueAssignedToInput");
    window.destroyAutocomplete("kanbanIssueOwnerInput");
  }

  function initField(inputId, hiddenId, field, placeholder) {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    var normalized = normalizeOptions(field);
    var value = normalized.value;
    var label = findLabel(normalized.options, value);
    window.initAutocomplete(inputId, hiddenId, normalized.options, {
      value: value,
      label: label,
      labelOnly: true,
      placeholder: placeholder || "搜索",
    });
  }

  function initSelects(project, assignedTo, owner) {
    destroySearchSelects();
    initField(
      "kanbanIssueProjectInput",
      "kanbanIssueProject",
      project,
      "搜索项目"
    );
    initField(
      "kanbanIssueAssignedToInput",
      "kanbanIssueAssignedTo",
      assignedTo,
      "搜索成员"
    );
    initField("kanbanIssueOwnerInput", "kanbanIssueOwner", owner, "搜索成员");
  }

  function emptyFields() {
    return {
      project: { options: [], value: "" },
      assignedTo: { options: [], value: "" },
      owner: { options: [], value: "" },
    };
  }

  function optionsUrl(teamgroupId, account) {
    var params = new URLSearchParams();
    if (teamgroupId) params.set("teamgroupId", teamgroupId);
    if (account) params.set("account", account);
    var q = params.toString();
    return q ? OPTIONS_URL + "?" + q : OPTIONS_URL;
  }

  function loadOptions() {
    var teamgroupId = activeTeamgroupID();
    var account = activeAccount();
    var title = $("kanbanIssueTitle");
    if (title) title.value = "";

    if (!teamgroupId) {
      var empty = emptyFields();
      initSelects(empty.project, empty.assignedTo, empty.owner);
      showToast("请先选择敏捷小组", "error");
      return Promise.resolve(false);
    }

    var seq = ++loadSeq;
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    return fetchFn(optionsUrl(teamgroupId, account), {
      method: "GET",
      credentials: "same-origin",
      headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" },
    })
      .then(function (r) {
        if (!r.ok) throw new Error("http");
        return r.json();
      })
      .then(function (payload) {
        if (seq !== loadSeq) return false;
        if (!payload || payload.success !== true) {
          throw new Error((payload && payload.message) || "payload");
        }
        initSelects(
          payload.project || {},
          payload.assignedTo || {},
          payload.owner || {}
        );
        return true;
      })
      .catch(function () {
        if (seq !== loadSeq) return false;
        var empty = emptyFields();
        initSelects(empty.project, empty.assignedTo, empty.owner);
        showToast("加载提问题选项失败", "error");
        return false;
      });
  }

  function openModal() {
    var overlay = $("kanbanCreateIssueOverlay");
    if (!overlay) return;
    overlay.classList.add("show");
    overlay.setAttribute("aria-hidden", "false");
    loadOptions().then(function () {
      var title = $("kanbanIssueTitle");
      if (title) {
        setTimeout(function () {
          title.focus();
        }, 0);
      }
    });
  }

  function closeModal() {
    destroySearchSelects();
    var overlay = $("kanbanCreateIssueOverlay");
    if (!overlay) return;
    overlay.classList.remove("show");
    overlay.setAttribute("aria-hidden", "true");
  }

  function onSubmit(ev) {
    ev.preventDefault();
    showToast("静态预览：提问题尚未接入保存", "info");
  }

  function bindOnce() {
    if (bound) return;
    bound = true;

    document.addEventListener("click", function (ev) {
      var openBtn = ev.target.closest("[data-kanban-create-issue]");
      if (openBtn) {
        ev.preventDefault();
        openModal();
      }
    });

    var closeBtn = $("kanbanCreateIssueCloseBtn");
    var cancelBtn = $("kanbanCreateIssueCancelBtn");
    var overlay = $("kanbanCreateIssueOverlay");
    var form = $("kanbanCreateIssueForm");

    if (closeBtn) closeBtn.addEventListener("click", closeModal);
    if (cancelBtn) cancelBtn.addEventListener("click", closeModal);
    if (overlay) {
      overlay.addEventListener("click", function (e) {
        if (e.target === overlay) closeModal();
      });
    }
    if (form) form.addEventListener("submit", onSubmit);

    document.addEventListener("keydown", function (ev) {
      if (ev.key !== "Escape") return;
      var el = $("kanbanCreateIssueOverlay");
      if (el && el.classList.contains("show")) closeModal();
    });
  }

  bindOnce();

  window.KanbanCreateIssue = {
    open: openModal,
    close: closeModal,
  };
})();
