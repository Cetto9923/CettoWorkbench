/* =============================================================================
   文件: web/static/js/kanban/createissue.js
   模块: 工作看板
   职责: 提问题静态弹窗开关与检索下拉（占位选项，不提交）。
============================================================================= */
(function () {
  "use strict";

  var PROJECT_OPTIONS = [
    { value: "101", label: "示例项目 · 信贷系统" },
    { value: "102", label: "示例项目 · 支付中台" },
    { value: "103", label: "示例项目 · 数据平台" },
  ];

  var MEMBER_OPTIONS = [
    { value: "zhangsan", label: "张三" },
    { value: "lisi", label: "李四" },
    { value: "wangwu", label: "王五" },
    { value: "zhaoliu", label: "赵六" },
  ];

  var bound = false;
  var selectsReady = false;

  function $(id) {
    return document.getElementById(id);
  }

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function currentAccount() {
    var people = document.querySelector(".po-board [data-people]");
    return people
      ? String(people.getAttribute("data-current-account") || "").trim()
      : "";
  }

  function memberOptionsWithAccount() {
    var account = currentAccount();
    if (!account) return MEMBER_OPTIONS.slice();
    for (var i = 0; i < MEMBER_OPTIONS.length; i++) {
      if (MEMBER_OPTIONS[i].value === account) {
        return MEMBER_OPTIONS.slice();
      }
    }
    return MEMBER_OPTIONS.concat([{ value: account, label: account }]);
  }

  function findOption(items, value) {
    var v = String(value || "").trim();
    if (!v) return null;
    for (var i = 0; i < items.length; i++) {
      if (items[i].value === v) return items[i];
    }
    return null;
  }

  function initSearchSelects() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    if (selectsReady) {
      return;
    }
    var members = memberOptionsWithAccount();
    var owner =
      findOption(members, currentAccount()) || members[0] || null;
    var sm = members[0] || null;

    window.initAutocomplete(
      "kanbanIssueProjectInput",
      "kanbanIssueProject",
      PROJECT_OPTIONS,
      {
        value: PROJECT_OPTIONS[0] ? PROJECT_OPTIONS[0].value : "",
        label: PROJECT_OPTIONS[0] ? PROJECT_OPTIONS[0].label : "",
        labelOnly: true,
        placeholder: "搜索项目",
      }
    );
    window.initAutocomplete(
      "kanbanIssueAssignedToInput",
      "kanbanIssueAssignedTo",
      members,
      {
        value: sm ? sm.value : "",
        label: sm ? sm.label : "",
        labelOnly: true,
        placeholder: "搜索成员",
      }
    );
    window.initAutocomplete(
      "kanbanIssueOwnerInput",
      "kanbanIssueOwner",
      members,
      {
        value: owner ? owner.value : "",
        label: owner ? owner.label : "",
        labelOnly: true,
        placeholder: "搜索成员",
      }
    );
    selectsReady = true;
  }

  function destroySearchSelects() {
    if (typeof window.destroyAutocomplete !== "function") {
      return;
    }
    window.destroyAutocomplete("kanbanIssueProjectInput");
    window.destroyAutocomplete("kanbanIssueAssignedToInput");
    window.destroyAutocomplete("kanbanIssueOwnerInput");
    selectsReady = false;
  }

  function resetForm() {
    var title = $("kanbanIssueTitle");
    if (title) title.value = "";
    destroySearchSelects();
    initSearchSelects();
  }

  function openModal() {
    var overlay = $("kanbanCreateIssueOverlay");
    if (!overlay) return;
    resetForm();
    overlay.classList.add("show");
    overlay.setAttribute("aria-hidden", "false");
    var title = $("kanbanIssueTitle");
    if (title) {
      setTimeout(function () {
        title.focus();
      }, 0);
    }
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
        return;
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
