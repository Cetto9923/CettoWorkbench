/* =============================================================================
   文件: web/static/js/kanban/createissue.js
   模块: 工作看板
   职责: 提问题弹窗开关、选项加载与保存（GET/POST /kanban/createissue）。
============================================================================= */
(function () {
  "use strict";

  var OPTIONS_URL = "/kanban/createissue";
  var SAVE_URL = "/kanban/issue";

  var bound = false;
  var loadSeq = 0;
  var submitting = false;

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

  function csrfHeaders() {
    var headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest",
    };
    var meta = document.querySelector('meta[name="csrf-token"]');
    var token = meta ? String(meta.getAttribute("content") || "").trim() : "";
    if (token) headers["X-CSRF-Token"] = token;
    return headers;
  }

  function readFormData() {
    var titleEl = $("kanbanIssueTitle");
    var projectEl = $("kanbanIssueProject");
    var assignedEl = $("kanbanIssueAssignedTo");
    var ownerEl = $("kanbanIssueOwner");
    return {
      title: titleEl ? String(titleEl.value || "").trim() : "",
      project: Number(projectEl ? projectEl.value : 0) || 0,
      assignedTo: assignedEl ? String(assignedEl.value || "").trim() : "",
      owner: ownerEl ? String(ownerEl.value || "").trim() : "",
    };
  }

  function validateFormData(data) {
    if (!data.title) {
      showToast("请填写问题名称", "error");
      return false;
    }
    if (!data.project) {
      showToast("请选择所属项目", "error");
      return false;
    }
    if (!data.assignedTo) {
      showToast("请选择指派给", "error");
      return false;
    }
    if (!data.owner) {
      showToast("请选择提出人", "error");
      return false;
    }
    return true;
  }

  function setSubmitDisabled(disabled) {
    var btn = $("kanbanCreateIssueSaveBtn");
    if (btn) btn.disabled = !!disabled;
  }

  function onSubmit(ev) {
    ev.preventDefault();
    if (submitting) return;
    var data = readFormData();
    if (!validateFormData(data)) return;

    submitting = true;
    setSubmitDisabled(true);
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    fetchFn(SAVE_URL, {
      method: "POST",
      credentials: "same-origin",
      headers: csrfHeaders(),
      body: JSON.stringify(data),
    })
      .then(function (r) {
        return r.text().then(function (text) {
          var parsed = {};
          try {
            parsed = text ? JSON.parse(text) : {};
          } catch (ignore) {
            parsed = {};
          }
          return { ok: r.ok, data: parsed, status: r.status };
        });
      })
      .then(function (wrap) {
        var resp = wrap.data || {};
        if (wrap.ok && resp.success) {
          var msg = resp.message || "问题已创建";
          if (resp.id) msg = msg + "（#" + resp.id + "）";
          showToast(msg, "success");
          closeModal();
          if (window.KanbanIssue && typeof window.KanbanIssue.reload === "function") {
            window.KanbanIssue.reload();
          }
          return;
        }
        if (resp.errors && resp.errors.length) {
          showToast(resp.errors[0].message || "校验失败", "error");
          return;
        }
        showToast(resp.message || "创建问题失败", "error");
      })
      .catch(function () {
        showToast("创建问题失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setSubmitDisabled(false);
      });
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
