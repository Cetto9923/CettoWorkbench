// =============================================================================
// 文件: web/static/js/po/home-team-kanban.js
// 模块: 团队长工作台 - 小组研发工作看板
// 职责: 异步加载指定小组研发任务，提供三列展示、成员筛选、待核查任务隔离与详情下钻。
// =============================================================================

(function () {
  "use strict";

  const state = {
    teamID: 0,
    groupID: 0,
    groupName: "",
    members: [],
    selectedAccount: "all",
    currentView: "confirmed", // confirmed | pending
    isLoading: false,
    kanbanData: null,
    requestSeq: 0,
  };

  function esc(s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  async function loadTasks() {
    const container = document.getElementById("thGroupKanbanContainer");
    if (!container || state.teamID <= 0 || state.groupID <= 0) return;

    const reqSeq = ++state.requestSeq;
    state.isLoading = true;
    container.innerHTML =
      '<div class="state-placeholder th-kanban-placeholder"><i class="fas fa-spinner fa-spin"></i> 正在加载小组研发工作数据…</div>';

    try {
      let url =
        "/team/group/tasks?teamId=" +
        encodeURIComponent(state.teamID) +
        "&groupId=" +
        encodeURIComponent(state.groupID);
      if (state.selectedAccount && state.selectedAccount !== "all") {
        url += "&account=" + encodeURIComponent(state.selectedAccount);
      }

      const resp = await fetch(url, { credentials: "same-origin" });
      if (reqSeq !== state.requestSeq) return;
      if (resp.status === 403) {
        container.innerHTML =
          '<div class="th-empty-card" role="alert"><p class="th-empty-text">您无权访问该小组任务数据</p></div>';
        return;
      }
      if (!resp.ok) {
        throw new Error("HTTP " + resp.status);
      }

      const json = await resp.json();
      if (reqSeq !== state.requestSeq) return;
      if (!json.success || !json.data) {
        throw new Error(json.message || "加载小组任务失败");
      }

      state.kanbanData = json.data;
      render();
    } catch (err) {
      if (reqSeq !== state.requestSeq) return;
      container.innerHTML =
        '<div class="th-empty-card" role="alert"><p class="th-empty-text">' +
        esc(err.message || "加载小组研发任务失败") +
        '</p><button type="button" class="retry-btn" id="thKanbanRetryBtn">重试</button></div>';
      const btn = document.getElementById("thKanbanRetryBtn");
      if (btn) {
        btn.addEventListener("click", loadTasks);
      }
    } finally {
      if (reqSeq === state.requestSeq) {
        state.isLoading = false;
      }
    }
  }

  function render() {
    const container = document.getElementById("thGroupKanbanContainer");
    if (!container || !state.kanbanData) return;

    const data = state.kanbanData;
    const summary = data.summary || {};
    const columns = data.columns || [];
    const pendingTasks = data.pendingReviewTasks || [];

    let html = '<div class="th-kanban-wrap">';

    // 1. 顶部操作栏与统计
    html += '<div class="th-kanban-header">';
    html += '  <div class="th-kanban-title-row">';
    html +=
      '    <span class="th-kanban-title"><i class="fas fa-diagram-project"></i> 研发工作看板</span>';
    html += '    <div class="th-kanban-view-tabs" role="tablist">';
    html +=
      '      <button type="button" class="th-kanban-view-tab' +
      (state.currentView === "confirmed" ? " active" : "") +
      '" data-view="confirmed" role="tab">';
    html +=
      '        正式研发任务 <strong>(' +
      (summary.confirmedTotal || 0) +
      ")</strong>";
    html += "      </button>";
    html +=
      '      <button type="button" class="th-kanban-view-tab' +
      (state.currentView === "pending" ? " active" : "") +
      '" data-view="pending" role="tab">';
    html +=
      '        待核查候选任务 <span class="th-kanban-badge-warn">' +
      (summary.pendingReviewTotal || 0) +
      "</span>";
    html += "      </button>";
    html += "    </div>";
    html += "  </div>";

    // 指标胶囊
    html += '  <div class="th-kanban-stats">';
    html +=
      '    <span class="th-kanban-stat-pill">未开始:<strong>' +
      (summary.confirmedWait || 0) +
      "</strong></span>";
    html +=
      '    <span class="th-kanban-stat-pill">进行中:<strong>' +
      (summary.confirmedDoing || 0) +
      "</strong></span>";
    html +=
      '    <span class="th-kanban-stat-pill">已完成:<strong>' +
      (summary.confirmedDone || 0) +
      "</strong></span>";
    if (summary.confirmedOverdue > 0) {
      html +=
        '    <span class="th-kanban-stat-pill danger">逾期:<strong>' +
        summary.confirmedOverdue +
        "</strong></span>";
    }
    html += "  </div>";
    if (data.timeRangeLabel) {
      html += '  <div class="th-kanban-time-tip">';
      html += '    <i class="fas fa-circle-info"></i> ' + esc(data.timeRangeLabel);
      html += '  </div>';
    }
    html += "</div>";

    // 2. 成员筛选栏
    html += '<div class="th-kanban-people-bar">';
    html += '  <span class="th-kanban-people-label">人员筛选:</span>';
    const allActive = state.selectedAccount === "all" ? " active" : "";
    html +=
      '  <button type="button" class="th-kanban-person-btn' +
      allActive +
      '" data-account="all">全部成员</button>';

    if (state.members && state.members.length > 0) {
      state.members.forEach(function (m) {
        const isActive = state.selectedAccount === m.account ? " active" : "";
        html +=
          '  <button type="button" class="th-kanban-person-btn' +
          isActive +
          '" data-account="' +
          esc(m.account) +
          '">';
        html += esc(m.name || m.account);
        html += "  </button>";
      });
    }
    html += "</div>";

    // 3. 看板主体
    if (state.currentView === "confirmed") {
      html += '<div class="th-kanban-columns">';
      columns.forEach(function (col) {
        html += '  <div class="th-kanban-col">';
        html += '    <div class="th-kanban-col-hdr">';
        html += "      <span>" + esc(col.name) + "</span>";
        html +=
          '      <span class="th-kanban-col-count">' +
          (col.count || 0) +
          "</span>";
        html += "    </div>";
        html += '    <div class="th-kanban-col-body">';
        if (!col.items || col.items.length === 0) {
          html +=
            '      <div class="state-placeholder th-kanban-placeholder-col">暂无' +
            esc(col.name) +
            "任务</div>";
        } else {
          col.items.forEach(function (item) {
            html += renderCardHTML(item);
          });
        }
        html += "    </div>";
        html += "  </div>";
      });
      html += "</div>";
    } else {
      // 待核查候选任务面板
      const pendPag = data.pendingPagination || {};
      html += '<div class="th-pending-panel">';
      html += '  <div class="th-pending-tip">';
      html +=
        '    <i class="fas fa-triangle-exclamation"></i> 提示：以下任务因无明确业务需求归属或属于独立技术改造，未计入正式小组指标。请团队长核查后在禅道中补充需求挂靠。' +
        (pendPag.total > 0 ? '（共 ' + pendPag.total + ' 项候选任务）' : '');
      html += "  </div>";
      if (pendingTasks.length === 0) {
        html +=
          '  <div class="state-placeholder th-kanban-placeholder">当前无待核查候选任务</div>';
      } else {
        html += '  <div class="th-pending-grid">';
        pendingTasks.forEach(function (item) {
          html += renderCardHTML(item);
        });
        html += "  </div>";
      }
      html += "</div>";
    }

    html += "</div>"; // end th-kanban-wrap
    container.innerHTML = html;

    // 绑定交互事件
    bindEvents(container);
  }

  function renderCardHTML(item) {
    let cls = "th-kanban-card";
    if (item.overdue) cls += " is-overdue";
    if (item.isPendingReview) cls += " is-pending";

    let html = '<div class="' + cls + '">';
    html += '  <div class="th-kanban-card-title-row">';
    html +=
      '    <strong class="th-kanban-card-title">' +
      esc(item.title) +
      "</strong>";
    if (item.url) {
      html +=
        '    <a class="th-kanban-card-id" href="' +
        esc(item.url) +
        '" target="_blank" rel="noopener noreferrer" title="在禅道查看任务详情">' +
        esc(item.displayId || "#" + item.id) +
        " ↗</a>";
    } else {
      html +=
        '    <span class="th-kanban-card-id">' +
        esc(item.displayId || "#" + item.id) +
        "</span>";
    }
    html += "  </div>";

    if (item.storyTitle) {
      html +=
        '  <div class="th-kanban-card-story" title="所属研需: ' +
        esc(item.storyTitle) +
        '"><i class="fas fa-book-bookmark"></i> ' +
        esc(item.storyTitle) +
        "</div>";
    }

    html += '  <div class="th-kanban-card-meta">';
    html += "    <span><i class=\"fas fa-user\"></i> " + esc(item.owner || "未指派") + "</span>";

    html += '    <div class="th-kanban-card-tags">';
    if (item.sourceGroupName) {
      html +=
        '      <span class="th-kanban-tag tag-cross" title="来自跨组需求"><i class="fas fa-arrows-split-up-and-left"></i> ' +
        esc(item.sourceGroupName) +
        "</span>";
    }
    if (item.isPendingReview) {
      html +=
        '      <span class="th-kanban-tag tag-pending"><i class="fas fa-clock"></i> 待核查</span>';
    }
    if (item.overdue) {
      html +=
        '      <span class="th-kanban-tag tag-overdue"><i class="fas fa-hourglass-end"></i> 逾期</span>';
    } else if (item.deadline) {
      html += "      <span>" + esc(item.deadline) + "</span>";
    }
    html += "    </div>";
    html += "  </div>";

    html += "</div>";
    return html;
  }

  function bindEvents(container) {
    const viewTabs = container.querySelectorAll(".th-kanban-view-tab");
    viewTabs.forEach(function (tab) {
      tab.addEventListener("click", function () {
        const v = tab.getAttribute("data-view");
        if (v && v !== state.currentView) {
          state.currentView = v;
          render();
        }
      });
    });

    const personBtns = container.querySelectorAll(".th-kanban-person-btn");
    personBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        const acc = btn.getAttribute("data-account");
        if (acc && acc !== state.selectedAccount) {
          state.selectedAccount = acc;
          loadTasks();
        }
      });
    });
  }

  // 供外部调用的入口方法
  window.TeamGroupKanban = {
    loadGroupKanban: function (teamID, groupID, groupName, members) {
      state.teamID = teamID;
      state.groupID = groupID;
      state.groupName = groupName || "";
      state.members = members || [];
      state.selectedAccount = "all";
      state.currentView = "confirmed";
      loadTasks();
    },
    clear: function () {
      const container = document.getElementById("thGroupKanbanContainer");
      if (container) container.innerHTML = "";
    },
  };
})();
