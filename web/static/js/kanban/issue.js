/* =============================================================================
   文件: web/static/js/kanban/issue.js
   模块: 工作看板
   职责: 右侧问题栏（按选中提出人 + 未解决/已解决 tab 拉取并渲染）。
============================================================================= */
(function () {
  "use strict";

  var ISSUES_URL = "/kanban/issues";
  var TAB_UNRESOLVED = "unresolved";
  var TAB_RESOLVED = "resolved";

  var root = null;
  var listEl = null;
  var countEl = null;
  var tabsEl = null;
  var getTeamgroupId = function () {
    return "";
  };
  var selectedAccount = "";
  var currentTab = TAB_UNRESOLVED;
  var loadSeq = 0;

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function statusClass(status) {
    var code = String(status || "").toLowerCase();
    if (code === "resolved") return "is-resolved";
    if (code === "closed" || code === "canceled") return "is-closed";
    if (code === "confirmed") return "is-confirmed";
    return "is-unconfirmed";
  }

  function stateClass(status) {
    var code = String(status || "").toLowerCase();
    if (code === "resolved") return "state-resolved";
    if (code === "closed" || code === "canceled") return "state-closed";
    return "state-open";
  }

  function issuesUrl(account, tab) {
    var params = new URLSearchParams();
    var acc = String(account || "").trim();
    if (acc) params.set("account", acc);
    var tg = String(getTeamgroupId() || "").trim();
    if (tg) params.set("teamgroupId", tg);
    params.set("tab", tab || TAB_UNRESOLVED);
    return ISSUES_URL + "?" + params.toString();
  }

  function updateCounts(counts) {
    counts = counts || {};
    var unresolved = Number(counts.unresolved || 0);
    var resolved = Number(counts.resolved || 0);
    if (countEl) countEl.textContent = String(unresolved + resolved);
    if (!tabsEl) return;
    var u = tabsEl.querySelector('[data-issue-count="unresolved"]');
    var r = tabsEl.querySelector('[data-issue-count="resolved"]');
    if (u) u.textContent = String(unresolved);
    if (r) r.textContent = String(resolved);
  }

  function setActiveTab(tab) {
    currentTab = tab === TAB_RESOLVED ? TAB_RESOLVED : TAB_UNRESOLVED;
    if (!tabsEl) return;
    tabsEl.querySelectorAll(".issue-tab[data-issue-tab]").forEach(function (btn) {
      var key = btn.getAttribute("data-issue-tab");
      btn.classList.toggle("active", key === currentTab);
    });
  }

  function showEmpty(msg) {
    if (!listEl) return;
    listEl.innerHTML =
      '<div class="demand-empty demand-empty--visible">' +
      escapeHtml(msg || "暂无问题") +
      "</div>";
  }

  function renderCards(items) {
    if (!listEl) return;
    if (!items || !items.length) {
      var label = currentTab === TAB_RESOLVED ? "已解决" : "未解决";
      showEmpty("当前暂无「" + label + "」问题");
      return;
    }
    listEl.innerHTML = items
      .map(function (it) {
        var id = String((it && it.id) || "—");
        var title = escapeHtml((it && it.title) || "");
        var owner = escapeHtml((it && (it.owner || it.assignedTo || it.createdBy)) || "待指派");
        var sevCode = String((it && it.severity) || "").trim();
        var sevLabel = escapeHtml((it && it.severityLabel) || "");
        var sevBadge = sevLabel
          ? '<span class="wb-severity is-compact" data-severity="' +
            escapeHtml(sevCode) +
            '">' +
            sevLabel +
            "</span>"
          : "";
        var lbl = escapeHtml((it && it.statusLabel) || "");
        var cls = statusClass(it && it.status);
        var state = stateClass(it && it.status);
        var idInner = escapeHtml(id);
        if (it && it.url) {
          idInner =
            '<a class="task-code-link" href="' +
            escapeHtml(it.url) +
            '" target="_blank" rel="noopener noreferrer" title="在禅道打开问题详情">' +
            idInner +
            "</a>";
        }
        var metaLeft = owner + (sevBadge ? " · " + sevBadge : "");
        return (
          '<div class="issue-card ' +
          state +
          '" data-issue-id="' +
          escapeHtml(id) +
          '">' +
          '  <div class="issue-id">' +
          idInner +
          "</div>" +
          '  <div class="issue-title" title="' +
          title +
          '">' +
          title +
          "</div>" +
          '  <div class="issue-meta">' +
          "    <span>" +
          metaLeft +
          "</span>" +
          '    <span class="ki-status ' +
          cls +
          '">' +
          lbl +
          "</span>" +
          "  </div>" +
          "</div>"
        );
      })
      .join("");
  }

  function load() {
    if (!listEl) return;
    var acc = String(selectedAccount || "").trim();
    if (!acc) {
      updateCounts({ unresolved: 0, resolved: 0 });
      showEmpty("暂无成员");
      return;
    }
    var seq = ++loadSeq;
    showEmpty("加载中…");
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    fetchFn(issuesUrl(acc, currentTab), { method: "GET", credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok) throw new Error("http");
        return r.json();
      })
      .then(function (payload) {
        if (seq !== loadSeq) return;
        if (!payload || payload.success !== true) throw new Error("payload");
        updateCounts(payload.counts || {});
        renderCards(payload.items || []);
      })
      .catch(function () {
        if (seq !== loadSeq) return;
        updateCounts({ unresolved: 0, resolved: 0 });
        showEmpty("问题加载失败");
      });
  }

  function bindTabs() {
    if (!tabsEl) return;
    tabsEl.addEventListener("click", function (ev) {
      var btn = ev.target.closest(".issue-tab[data-issue-tab]");
      if (!btn || !tabsEl.contains(btn)) return;
      var tab = btn.getAttribute("data-issue-tab");
      if (!tab || tab === currentTab) return;
      setActiveTab(tab);
      load();
    });
  }

  function init(opts) {
    opts = opts || {};
    root = document.querySelector(".po-board");
    if (!root) return;
    listEl = root.querySelector(".issue-list");
    countEl = root.querySelector(".issue-count");
    tabsEl = root.querySelector(".issue-tabs");
    if (typeof opts.getTeamgroupId === "function") {
      getTeamgroupId = opts.getTeamgroupId;
    }
    setActiveTab(TAB_UNRESOLVED);
    bindTabs();
  }

  function setAccount(account) {
    selectedAccount = String(account || "").trim();
    load();
  }

  window.KanbanIssue = {
    init: init,
    setAccount: setAccount,
    reload: load,
  };
})();
