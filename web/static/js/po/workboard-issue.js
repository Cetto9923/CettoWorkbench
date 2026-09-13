/**
 * 模块: PO 工作看板 - 问题三态与详情抽屉
 * 职责: 问题按「未解决 / 已解决」筛选、卡片渲染与禅道审计记录展示。
 */
(function () {
  "use strict";

  var allIssues = [];
  var currentTab = "未解决";
  var currentDrawerIssue = null;

  var esc = window.escapeHtml;

  function bucketIssue(it) {
    var code = String((it && it.status) || "").trim().toLowerCase();
    if (code === "closed" || code === "cancel" || code === "canceled" || code === "已关闭" || code === "已取消") {
      return "已关闭";
    }
    if (code === "resolved" || code === "done" || code === "已解决") {
      return "已解决";
    }
    return "未解决";
  }

  function statusLabel(bucket, raw) {
    if (bucket === "已解决") return "已解决";
    if (bucket === "已关闭") return "已关闭";
    if (raw === "confirmed") return "已确认";
    if (raw === "unconfirmed") return "未确认";
    return "未解决";
  }

  function statusClass(bucket) {
    if (bucket === "已解决") return "is-resolved";
    if (bucket === "已关闭") return "is-closed";
    return "is-open";
  }

  function loadIssues() {
    fetch("/board/issues", { method: "GET" })
      .then(function (r) {
        if (!r.ok) throw new Error("http " + r.status);
        return r.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) throw new Error("payload error");
        allIssues = (payload.items || []).filter(function (item) { return bucketIssue(item) !== "已关闭"; });
        renderIssuePanel();
      })
      .catch(function () {
        var list = document.getElementById("issueList");
        if (list) list.innerHTML = '<div class="demand-empty" style="display:block">暂无问题或加载失败</div>';
      });
  }

  function renderIssuePanel() {
    var counts = { "未解决": 0, "已解决": 0 };
    allIssues.forEach(function (it) {
      var b = bucketIssue(it);
      counts[b] = (counts[b] || 0) + 1;
    });

    var countEl = document.getElementById("issueCount");
    if (countEl) countEl.textContent = allIssues.length;

    var tabsEl = document.getElementById("issueTabs");
    if (tabsEl) {
      tabsEl.innerHTML = "";
      var tabs = [
        { key: "未解决", label: "未解决", count: counts["未解决"] },
        { key: "已解决", label: "已解决", count: counts["已解决"] }
      ];
      tabs.forEach(function (t) {
        var btn = document.createElement("button");
        btn.type = "button";
        btn.className = "issue-tab" + (currentTab === t.key ? " active" : "");
        btn.innerHTML = esc(t.label) + ' <strong>' + t.count + '</strong>';
        btn.addEventListener("click", function () {
          currentTab = t.key;
          tabsEl.querySelectorAll(".issue-tab").forEach(function (x) { x.classList.remove("active"); });
          btn.classList.add("active");
          renderIssueCards();
        });
        tabsEl.appendChild(btn);
      });
    }

    renderIssueCards();
  }

  function renderIssueCards() {
    var listEl = document.getElementById("issueList");
    if (!listEl) return;
    var matched = allIssues.filter(function (it) {
      return bucketIssue(it) === currentTab;
    });

    if (!matched.length) {
      listEl.innerHTML = '<div class="demand-empty" style="display:block">当前暂无「' + esc(currentTab) + '」问题</div>';
      return;
    }

    listEl.innerHTML = matched.map(function (it) {
      var b = bucketIssue(it);
      var displayId = String(it.id || "—");
      var lbl = statusLabel(b, it.status);
      var cls = statusClass(b);
      var handler = it.assignedTo || it.handler || it.createdBy || "待指派";
      var pri = it.priority ? "P" + it.priority : "P2";

      return (
        '<div class="issue-card state-' + (b === "已解决" ? "resolved" : b === "已关闭" ? "closed" : "open") +
        '" data-issue-id="' + esc(it.id) + '">' +
        '  <div class="issue-id">' + esc(displayId) + '</div>' +
        '  <div class="issue-title" title="' + esc(it.title) + '">' + esc(it.title) + '</div>' +
        '  <div class="issue-meta">' +
        '    <span>' + esc(handler) + ' · ' + esc(pri) + '</span>' +
        '    <span class="ki-status ' + cls + '">' + esc(lbl) + '</span>' +
        '  </div>' +
        '</div>'
      );
    }).join("");

    listEl.querySelectorAll(".issue-card").forEach(function (card) {
      card.addEventListener("click", function () {
        var id = card.getAttribute("data-issue-id");
        openIssueDetailDrawer(id);
      });
    });
  }

  function actionLabel(action) {
    var labels = {
      opened: "创建问题", edited: "编辑问题", assigned: "指派问题", issueconfirmed: "确认问题",
      resolved: "解决问题", closed: "关闭问题", activated: "重新激活问题", canceled: "取消问题",
      comment: "添加备注"
    };
    return labels[String(action || "").toLowerCase()] || String(action || "操作记录");
  }

  function actionRow(item) {
    var note = [item.extra, item.comment].filter(Boolean).join(" · ");
    return '<li class="issue-action-row">' +
      '<div class="issue-action-time">' + esc(item.date || "—") + '</div>' +
      '<div class="issue-action-main"><strong>' + esc(actionLabel(item.action)) + '</strong>' +
      '<span>' + esc(item.actorName || item.actor || "系统") + '</span>' +
      (note ? '<p>' + esc(note) + '</p>' : "") + '</div></li>';
  }

  function loadIssueActionHistory(issueID, afterID, append) {
    var section = document.querySelector('[data-issue-action-history="' + String(issueID) + '"]');
    if (!section) return;
    fetch("/board/issues/" + encodeURIComponent(issueID) + "/actions?afterId=" + encodeURIComponent(afterID || 0), { method: "GET" })
      .then(function (r) {
        if (!r.ok) throw new Error("http " + r.status);
        return r.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) throw new Error("payload error");
        if (!document.querySelector('[data-issue-action-history="' + String(issueID) + '"]')) return;
        var rows = payload.items || [];
        var list = section.querySelector(".issue-action-list");
        if (!append) list.innerHTML = "";
        if (!rows.length && !append) {
          list.innerHTML = '<li class="issue-action-empty">禅道未记录该问题的操作日志</li>';
        } else if (rows.length) {
          list.insertAdjacentHTML("beforeend", rows.map(actionRow).join(""));
        }
        var more = section.querySelector(".issue-action-more");
        if (payload.nextAfterId) {
          more.hidden = false;
          more.dataset.afterId = payload.nextAfterId;
        } else {
          more.hidden = true;
        }
      })
      .catch(function () {
        if (!append) section.querySelector(".issue-action-list").innerHTML = '<li class="issue-action-empty">操作日志加载失败，请重试</li>';
      });
  }

  function openIssueDetailDrawer(idOrItem) {
    var issue = typeof idOrItem === "object" ? idOrItem : allIssues.find(function (x) {
      return String(x.id) === String(idOrItem);
    });
    if (!issue) {
      window.showToast("未找到该问题详情");
      return;
    }
    currentDrawerIssue = issue;

    var b = bucketIssue(issue);
    var displayId = String(issue.id || "—");
    var lbl = statusLabel(b, issue.status);
    var cls = statusClass(b);

    var drawer = document.getElementById("issueDrawer");
    var backdrop = document.getElementById("issueDrawerBackdrop");
    if (!drawer || !backdrop) return;

    var idEl = document.getElementById("issueDrawerId");
    var titleEl = document.getElementById("issueDrawerTitle");
    var bodyEl = document.getElementById("issueDrawerBody");
    var footerEl = document.getElementById("issueDrawerFooter");

    if (idEl) idEl.textContent = displayId;
    if (titleEl) titleEl.textContent = issue.title || "问题详情";

    if (bodyEl) {
      bodyEl.innerHTML =
        '<div class="issue-detail-section">' +
        '  <div class="issue-detail-section-title">基本信息</div>' +
        '  <div class="issue-detail-grid">' +
        '    <div class="issue-detail-field"><div class="k">当前状态</div><div class="v"><span class="ki-status ' + cls + '">' + esc(lbl) + '</span></div></div>' +
        '    <div class="issue-detail-field"><div class="k">优先级</div><div class="v">P' + esc(issue.priority || "2") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">严重程度</div><div class="v">' + esc(issue.severity || "2级") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">指派给</div><div class="v">' + esc(issue.assignedTo || issue.handler || "待指派") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">提出人</div><div class="v">' + esc(issue.createdBy || "系统/PO") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">计划解决日期</div><div class="v">' + esc(issue.deadline || issue.planDate || "未设定") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">所属项目</div><div class="v">' + esc(issue.project || "核心业务系统研发") + '</div></div>' +
        '    <div class="issue-detail-field"><div class="k">所属执行</div><div class="v">' + esc(issue.execution || "敏捷迭代2026-09") + '</div></div>' +
        '  </div>' +
        '</div>' +
        '<div class="issue-detail-section">' +
        '  <div class="issue-detail-section-title">问题描述</div>' +
        '  <div class="issue-detail-desc">' + esc(issue.desc || issue.description || issue.title || "暂无详细描述信息") + '</div>' +
        '</div>' +
        '<div class="issue-detail-section">' +
        '  <div class="issue-detail-section-title">禅道操作记录</div>' +
        '  <div class="issue-action-history" data-issue-action-history="' + esc(issue.id) + '">' +
        '    <ol class="issue-action-list"><li class="issue-action-empty">正在加载从创建开始的操作记录…</li></ol>' +
        '    <button type="button" class="issue-action-more" hidden>加载更多</button>' +
        '  </div>' +
        '</div>';
      var moreButton = bodyEl.querySelector(".issue-action-more");
      if (moreButton) moreButton.addEventListener("click", function () {
        loadIssueActionHistory(issue.id, Number(moreButton.dataset.afterId || 0), true);
      });
      loadIssueActionHistory(issue.id, 0, false);
    }

    if (footerEl) {
      var actionButtons = [];
      if (b === "未解决") {
        actionButtons.push({ key: "resolve", label: "解决问题", cls: "action primary" });
      } else if (b === "已解决") {
        actionButtons.push({ key: "activate", label: "重新激活", cls: "action" });
        actionButtons.push({ key: "close", label: "关闭问题", cls: "action primary" });
      }
      actionButtons.push({ key: "closeDrawer", label: "关闭", cls: "action soft" });

      footerEl.innerHTML = actionButtons.map(function (btn) {
        return '<button type="button" class="' + btn.cls + '" data-action="' + btn.key + '">' + esc(btn.label) + '</button>';
      }).join("");

      footerEl.querySelectorAll("button[data-action]").forEach(function (btn) {
        btn.addEventListener("click", function () {
          var act = btn.getAttribute("data-action");
          handleIssueAction(act);
        });
      });
    }

    backdrop.classList.remove("hidden");
    drawer.classList.remove("hidden");
    setTimeout(function () {
      drawer.classList.add("open");
    }, 10);
  }

  function closeIssueDetailDrawer() {
    var drawer = document.getElementById("issueDrawer");
    var backdrop = document.getElementById("issueDrawerBackdrop");
    if (drawer) drawer.classList.remove("open");
    setTimeout(function () {
      if (drawer) drawer.classList.add("hidden");
      if (backdrop) backdrop.classList.add("hidden");
      currentDrawerIssue = null;
    }, 220);
  }

  function handleIssueAction(actionKey) {
    if (actionKey === "closeDrawer") {
      closeIssueDetailDrawer();
      return;
    }
    if (!currentDrawerIssue || !["resolve", "close", "activate"].includes(actionKey)) return;
    var issueID = currentDrawerIssue.id;
    var request = window.appFetch || window.fetch;
    request("/board/issues/" + encodeURIComponent(issueID) + "/transition", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ action: actionKey })
    }).then(function (response) {
      return response.json().catch(function () { return {}; }).then(function (payload) {
        if (!response.ok || !payload.success) throw new Error(payload.message || "禅道问题操作未完成");
        window.showToast("禅道已完成问题操作，正在刷新状态");
        closeIssueDetailDrawer();
        loadIssues();
      });
    }).catch(function (err) {
      window.showToast((err && err.message) || "禅道问题操作未完成，状态未变更", "error");
    });
  }

  function init() {
    var closeBtn = document.getElementById("issueDrawerClose");
    if (closeBtn) closeBtn.addEventListener("click", closeIssueDetailDrawer);
    var backdrop = document.getElementById("issueDrawerBackdrop");
    if (backdrop) backdrop.addEventListener("click", closeIssueDetailDrawer);

    loadIssues();
  }

  window.WorkboardIssue = {
    init: init,
    loadIssues: loadIssues,
    bucketIssue: bucketIssue,
    openIssueDetailDrawer: openIssueDetailDrawer,
    closeIssueDetailDrawer: closeIssueDetailDrawer,
    addIssueMock: function (issue) {
      allIssues.unshift(issue);
      currentTab = "未解决";
      renderIssuePanel();
    }
  };

  document.addEventListener("DOMContentLoaded", function () {
    init();
  });
})();
