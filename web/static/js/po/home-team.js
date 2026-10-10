// =============================================================================
// 文件: web/static/js/po/home-team.js
// 模块: PO工作台 - 团队视角
// 职责: 加载并渲染敏捷团队视角的数据，包括范围联动、价值流积压、版本窗口、风险雷达与临期事项
// =============================================================================
(function () {
  "use strict";

  const state = {
    scopes: [],
    deptId: positiveID(new URLSearchParams(window.location.search).get("deptId")),
    teamId: 0,
    subteamgroupId: positiveID(new URLSearchParams(window.location.search).get("subteamgroupId")),
    requestNo: 0,
    matrixOpen: false,
    dashboardData: null,
  };
  const params = new URLSearchParams(window.location.search);
  const originalScope = params.get("scope") === "dept" ? "dept" : "team";
  if (originalScope === "dept") state.deptId = positiveID(params.get("scopeId") || params.get("teamgroupId"));
  else state.teamId = positiveID(params.get("scopeId") || params.get("teamgroupId"));

  function positiveID(raw) {
    const value = String(raw || "");
    return /^\d+$/.test(value) && Number(value) > 0 ? Number(value) : 0;
  }

  var esc = window.escapeHtml;

  function queryForScope() {
    const query = new URLSearchParams();
    if (state.teamId) {
      query.set("scope", "team");
      query.set("scopeId", String(state.teamId));
      if (state.subteamgroupId) query.set("subteamgroupId", String(state.subteamgroupId));
    } else {
      query.set("scope", state.deptId ? "dept" : "team");
      if (state.deptId) query.set("scopeId", String(state.deptId));
    }
    return query;
  }

  function syncURL() {
    const url = new URL(window.location.href);
    url.searchParams.set("view", "team");
    const scope = queryForScope();
    url.searchParams.set("scope", scope.get("scope"));
    if (scope.has("scopeId")) url.searchParams.set("scopeId", scope.get("scopeId"));
    else url.searchParams.delete("scopeId");
    if (state.deptId) url.searchParams.set("deptId", String(state.deptId));
    else url.searchParams.delete("deptId");
    if (state.teamId) url.searchParams.set("teamgroupId", String(state.teamId));
    else url.searchParams.delete("teamgroupId");
    if (state.subteamgroupId) url.searchParams.set("subteamgroupId", String(state.subteamgroupId));
    else url.searchParams.delete("subteamgroupId");
    window.history.replaceState(null, "", url.pathname + url.search + url.hash);
  }

  function showError(message) {
    const alert = document.getElementById("teamHomeError");
    const text = document.getElementById("teamHomeErrorText");
    if (text) text.textContent = message || "团队数据暂不可用";
    if (alert) alert.hidden = false;
  }

  function endpointQuery() {
    return queryForScope().toString();
  }

  function setScopeOptions() {
    const depts = state.scopes.filter(function (item) { return item.kind === "dept"; });
    const teams = state.scopes.filter(function (item) { return item.kind === "team"; });
    const deptSelect = document.getElementById("teamHomeDeptSelect");
    const teamSelect = document.getElementById("teamHomeTeamSelect");
    const subgroupSelect = document.getElementById("teamHomeSubteamSelect");
    if (!deptSelect || !teamSelect || !subgroupSelect) return;

    deptSelect.innerHTML = '<option value="">全部可见部门</option>' + depts.map(function (item) {
      return '<option value="' + positiveID(item.id) + '">' + esc(item.name) + "</option>";
    }).join("");
    deptSelect.value = depts.some(function (item) { return positiveID(item.id) === state.deptId; }) ? String(state.deptId) : "";

    const deptTeams = state.deptId
      ? teams.filter(function (item) { return positiveID(item.deptId) === state.deptId; })
      : teams.filter(function (item) { return positiveID(item.parent) === 0; });
    const roots = deptTeams.filter(function (item) {
      return !teams.some(function (candidate) { return positiveID(candidate.id) === positiveID(item.parent); });
    });
    const validTeam = roots.some(function (item) { return positiveID(item.id) === state.teamId; });
    if (!validTeam) {
      state.teamId = 0;
      state.subteamgroupId = 0;
    }
    teamSelect.disabled = roots.length === 0;
    teamSelect.innerHTML = '<option value="">全部组织团队</option>' + roots.map(function (item) {
      return '<option value="' + positiveID(item.id) + '">' + esc(item.name) + "</option>";
    }).join("");
    teamSelect.value = state.teamId ? String(state.teamId) : "";

    const children = state.teamId ? teams.filter(function (item) {
      return positiveID(item.parent) === state.teamId;
    }) : [];
    const validSubgroup = children.some(function (item) { return positiveID(item.id) === state.subteamgroupId; });
    if (!validSubgroup) state.subteamgroupId = 0;
    subgroupSelect.disabled = children.length === 0;
    subgroupSelect.innerHTML = '<option value="">全部敏捷小组</option>' + children.map(function (item) {
      return '<option value="' + positiveID(item.id) + '">' + esc(item.name) + "</option>";
    }).join("");
    subgroupSelect.value = state.subteamgroupId ? String(state.subteamgroupId) : "";
  }

  async function loadScopes() {
    const response = await fetch("/home/team/scopes", {
      credentials: "include",
      headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
    });
    const json = await response.json().catch(function () { return null; });
    if (!response.ok || !json || json.success !== true || !Array.isArray(json.data)) {
      throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
    }
    state.scopes = json.data;
    setScopeOptions();
  }

  async function loadIssueRiskCounts(requestNo) {
    const issueCount = document.getElementById("teamHomeIssueCount");
    const riskCount = document.getElementById("teamHomeRiskCount");
    const status = document.getElementById("teamHomeRiskStatus");
    const link = document.getElementById("teamHomeRiskLink");
    const queryBase = queryForScope();
    const riskQuery = new URLSearchParams(queryBase);
    if (state.teamId) {
      riskQuery.delete("scope");
      riskQuery.delete("scopeId");
      riskQuery.delete("subteamgroupId");
      riskQuery.set("teamgroupId", String(state.subteamgroupId || state.teamId));
    } else if (state.deptId) {
      riskQuery.delete("scope");
      riskQuery.delete("scopeId");
      riskQuery.set("team", String(state.deptId));
    }
    riskQuery.set("kind", "issue");
    riskQuery.set("loop", "open");
    if (link) link.href = "/issues/risk?" + riskQuery.toString();
    if (issueCount) issueCount.textContent = "…";
    if (riskCount) riskCount.textContent = "…";
    if (status) status.textContent = "正在加载…";

    const fetchCount = async function (kind) {
      const query = new URLSearchParams(queryBase);
      if (state.teamId) {
        query.delete("scope");
        query.delete("scopeId");
        query.delete("subteamgroupId");
        query.set("teamgroupId", String(state.subteamgroupId || state.teamId));
      } else if (state.deptId) {
        query.delete("scope");
        query.delete("scopeId");
        query.set("team", String(state.deptId));
      }
      query.set("kind", kind);
      query.set("loop", "open");
      query.set("page", "1");
      query.set("pageSize", "1");
      const response = await fetch("/issues/risk/items?" + query.toString(), {
        credentials: "include",
        headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
      });
      const json = await response.json().catch(function () { return null; });
      if (!response.ok || !json || json.success !== true || !Number.isFinite(Number(json.total))) {
        throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
      }
      return Number(json.total);
    };
    try {
      const counts = await Promise.all([fetchCount("issue"), fetchCount("risk")]);
      if (requestNo !== state.requestNo) return;
      if (issueCount) issueCount.textContent = String(counts[0]);
      if (riskCount) riskCount.textContent = String(counts[1]);
      if (status) status.textContent = "未关闭问题与风险";
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      if (issueCount) issueCount.textContent = "—";
      if (riskCount) riskCount.textContent = "—";
      if (status) status.textContent = "问题风险数据暂不可用";
    }
  }

  async function loadTeamVersionWindows(requestNo) {
    const rows = document.getElementById("teamHomeVersionRows");
    const status = document.getElementById("teamHomeVersionStatus");
    if (!rows) return;
    rows.innerHTML = '<div class="state-placeholder">正在加载授权范围内的版本窗口…</div>';
    if (status) status.textContent = "正在加载…";
    try {
      const response = await fetch("/home/team/version-windows?" + endpointQuery(), {
        credentials: "include",
        headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
      });
      const json = await response.json().catch(function () { return null; });
      if (!response.ok || !json || json.success !== true || !Array.isArray(json.data)) {
        throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
      }
      if (requestNo !== state.requestNo) return;
      if (!json.data.length) {
        rows.innerHTML = '<div class="version-empty-state"><i class="fas fa-calendar-alt" aria-hidden="true"></i><span>未来 30 天内没有已关联的版本窗口</span></div>';
        if (status) status.textContent = "0 个窗口";
        return;
      }
      rows.innerHTML = json.data.map(function (windowItem) {
        const id = positiveID(windowItem.id);
        const groupID = positiveID(windowItem.teamgroupId);
        const stage = new URLSearchParams(window.location.search).get("stage") || "";
        const href = "/schedule?windows=" + encodeURIComponent(String(id)) + "&groups=" + encodeURIComponent(String(groupID)) + (stage ? "&stage=" + encodeURIComponent(stage) : "");
        return '<article class="team-home-version-row team-version-item">' +
          '<div class="team-version-date"><time datetime="' + esc(windowItem.releaseDate) + '">' + esc(windowItem.releaseDate) + '</time></div>' +
          '<div class="team-home-version-main team-version-info">' +
            '<a class="team-version-title" href="' + esc(href) + '"><strong>' + esc(windowItem.name) + '</strong></a>' +
            '<span class="team-version-group">' + esc(windowItem.teamgroup || "未命名敏捷小组") + '</span>' +
          '</div>' +
          '<span class="team-home-version-count team-version-count-badge">' + Number(windowItem.workItemCount || 0) + ' 个工作项</span>' +
        '</article>';
      }).join("");
      if (status) status.textContent = json.data.length + " 个窗口";
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      rows.innerHTML = '<div class="state-placeholder error">版本窗口数据暂不可用</div>';
      if (status) status.textContent = "加载失败";
    }
  }

  async function loadTeamValueStream(requestNo) {
    const rows = document.getElementById("teamHomeValueStreamRows");
    const status = document.getElementById("teamHomeValueStreamStatus");
    if (!rows) return;
    rows.innerHTML = '<div class="state-placeholder">正在加载授权范围内的价值流…</div>';
    if (status) status.textContent = "正在加载…";
    try {
      const response = await fetch("/home/team/value-stream?" + endpointQuery(), {
        credentials: "include",
        headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
      });
      const json = await response.json().catch(function () { return null; });
      if (!response.ok || !json || json.success !== true || !Array.isArray(json.data)) {
        throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
      }
      if (requestNo !== state.requestNo) return;
      rows.innerHTML = json.data.map(function (stage) {
        const count = Number(stage.count || 0);
        const isZero = count === 0 ? " is-zero" : "";
        return '<div class="team-home-stage-item team-vs-card stage-' + esc(stage.status) + '" data-stage="' + esc(stage.status) + '" title="' + esc(stage.label) + '">' +
          '<div class="team-vs-top">' +
            '<span class="team-vs-name">' + esc(stage.label) + '</span>' +
            '<strong class="team-vs-count' + isZero + '">' + count + '</strong>' +
          '</div>' +
          '<div class="team-vs-breakdown">' +
            '<span class="track-pill biz">业务 ' + Number(stage.demandCount || 0) + '</span>' +
            '<span class="track-pill dev">研发 ' + Number(stage.storyCount || 0) + '</span>' +
          '</div>' +
        '</div>';
      }).join("");
      if (!json.data.length) rows.innerHTML = '<div class="state-placeholder">当前授权范围内暂无价值流数据</div>';
      if (status) status.textContent = json.data.length + " 个阶段";
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      rows.innerHTML = '<div class="state-placeholder error">价值流数据暂不可用</div>';
      if (status) status.textContent = "加载失败";
    }
  }


  async function loadTeamDashboard(requestNo) {
    const rows = document.getElementById("teamHomeDueRows");
    const status = document.getElementById("teamHomeDueStatus");
    const blocked = document.getElementById("teamHomeBlockedCount");
    const overdue = document.getElementById("teamHomeOverdueCount");
    const soon = document.getElementById("teamHomeSoonCount");
    const blockedLink = document.getElementById("teamHomeBlockedLink");
    const query = queryForScope();
    if (blockedLink) blockedLink.href = "/home?view=demand&focus=blocked&" + query.toString();
    if (rows) rows.innerHTML = '<div class="state-placeholder">正在加载临期与逾期事项…</div>';
    if (status) status.textContent = "正在加载…";
    try {
      const response = await fetch("/home/team/dashboard?" + query.toString(), {
        credentials: "include",
        headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
      });
      const json = await response.json().catch(function () { return null; });
      if (!response.ok || !json || json.success !== true || !json.data) {
        throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
      }
      if (requestNo !== state.requestNo) return;
      const data = json.data;
      state.dashboardData = data;


      if (data.projects && data.projects.available) {
        const pc = document.getElementById("teamProjectsCount");
        if (pc) pc.innerHTML = data.projects.totalActive + '<small>个在建</small>';
        const ps = document.getElementById("teamProjectsStats");
        if (ps) ps.innerHTML = '<span class="chip red">' + data.projects.overdueCount + ' 延期风险</span>' +
          '<span class="chip orange">' + data.projects.soonCount + ' 临期节点</span>' +
          '<span class="chip green">' + data.projects.normalCount + ' 正常推进</span>';
      }
      if (data.tickets && data.tickets.available) {
        const tc = document.getElementById("teamTicketsCount");
        if (tc) tc.innerHTML = data.tickets.activeCount + '<small>单在途</small>';
        const ts = document.getElementById("teamTicketsStats");
        if (ts) ts.innerHTML = '<span class="chip red">' + data.tickets.overdueCount + ' 在途延期</span>' +
          '<span class="chip orange">' + data.tickets.unscheduledCount + ' 待排期</span>' +
          '<span class="chip green">本月上线 ' + data.tickets.launchedMonth + '</span>';
      }
      if (data.weekly && data.weekly.available) {
        const wp = document.getElementById("teamWeeklyPending");
        if (wp) wp.innerHTML = (data.weekly.missingCount + data.weekly.reviewCount) + '<small>份待办</small>';
        const ws = document.getElementById("teamWeeklyStats");
        if (ws) ws.innerHTML = '<span class="chip red">' + data.weekly.missingCount + ' 份未提交</span>' +
          '<span class="chip orange">' + data.weekly.reviewCount + ' 份待确认</span>' +
          '<span class="chip green">' + data.weekly.confirmedCount + ' 份已归档</span>';
      }

      if (blocked) blocked.textContent = String(Number(data.blocked || 0));
      if (overdue) overdue.textContent = String(Number(data.overdue || 0));
      if (soon) soon.textContent = String(Number(data.soon || 0));
      if (status) status.textContent = "更新于 " + esc(window.formatDateTime ? window.formatDateTime(data.updatedAt) : (data.updatedAt || "刚刚"));
      const due = Array.isArray(data.due) ? data.due : [];
      if (rows) rows.innerHTML = due.length ? due.map(function (item) {
        const isTask = item.kind === "task";
        const kindLabel = isTask ? "任务" : "需求";
        const kindClass = isTask ? "task" : "demand";
        return '<article class="team-home-due-row due-item-row' + (item.overdue ? ' is-overdue' : '') + '">' +
          '<div class="team-home-due-main due-item-main">' +
            '<strong class="due-item-title" title="' + esc(item.title || "未命名事项") + '">' + esc(item.title || "未命名事项") + '</strong>' +
            '<div class="due-item-meta">' +
              '<span class="meta-tag ' + kindClass + '">' + kindLabel + '</span>' +
              '<span><i class="fas fa-user-circle" aria-hidden="true"></i> ' + esc(item.owner || "未分配") + '</span>' +
            '</div>' +
          '</div>' +
          '<time class="due-item-time" datetime="' + esc(item.deadline || "") + '">' + esc(item.deadline || "无期限") + '</time>' +
        '</article>';
      }).join("") : '<div class="version-empty-state"><i class="fas fa-check-circle" aria-hidden="true"></i><span>未来 3 个工作日内没有临期或逾期事项</span></div>';
      const updated = document.getElementById("teamHomeUpdatedAt");
      if (updated) updated.textContent = data.throughDate ? "统计至 " + data.throughDate : "";
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      if (blocked) blocked.textContent = "—";
      if (overdue) overdue.textContent = "—";
      if (soon) soon.textContent = "—";
      if (rows) rows.innerHTML = '<div class="state-placeholder error">临期与逾期数据暂不可用</div>';
      if (status) status.textContent = "加载失败";
    }
  }

  async function load() {
    const requestNo = ++state.requestNo;
    const error = document.getElementById("teamHomeError");
    if (error) error.hidden = true;
    try {
      if (!state.scopes.length) await loadScopes();
      else setScopeOptions();
      syncURL();
      loadIssueRiskCounts(requestNo);
      loadTeamVersionWindows(requestNo);
      loadTeamValueStream(requestNo);
      loadTeamDashboard(requestNo);
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      showError(error.message);
    }
  }

  const deptSelect = document.getElementById("teamHomeDeptSelect");
  const teamSelect = document.getElementById("teamHomeTeamSelect");
  const subgroupSelect = document.getElementById("teamHomeSubteamSelect");
  if (deptSelect) deptSelect.addEventListener("change", function () {
    state.deptId = positiveID(deptSelect.value);
    state.teamId = 0;
    state.subteamgroupId = 0;
    setScopeOptions();
    load();
  });
  if (teamSelect) teamSelect.addEventListener("change", function () {
    state.teamId = positiveID(teamSelect.value);
    state.subteamgroupId = 0;
    setScopeOptions();
    load();
  });
  if (subgroupSelect) subgroupSelect.addEventListener("change", function () {
    state.subteamgroupId = positiveID(subgroupSelect.value);
    load();
  });
  const retry = document.getElementById("teamHomeRetry");
  if (retry) retry.addEventListener("click", load);

  function showTeamToast(msg) {
    if (typeof document === "undefined" || !document.createElement) return;
    const toast = document.createElement("div");
    toast.className = "team-toast";
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(function () { toast.remove(); }, 2500);
  }

  if (typeof document !== "undefined" && typeof document.addEventListener === "function") {
    document.addEventListener("click", function (e) {
      const target = e.target;
      if (!target || typeof target.closest !== "function") return;

      const toggleMatrix = target.closest("#teamToggleMatrixBtn, #teamCloseMatrixBtn");
      if (toggleMatrix) {
        state.matrixOpen = !state.matrixOpen;
        const matrixEl = document.getElementById("fullScreenMatrix");
        const chevron = document.getElementById("teamMatrixChevron");
        if (matrixEl) {
          if (state.matrixOpen) {
            matrixEl.removeAttribute("hidden");
            matrixEl.scrollIntoView({ behavior: "smooth", block: "start" });
          } else {
            matrixEl.setAttribute("hidden", "");
          }
        }
        if (chevron) {
          chevron.className = state.matrixOpen ? "fas fa-chevron-up" : "fas fa-chevron-down";
        }
        return;
      }

      const screenBtn = target.closest("[data-open-screen]");
      if (screenBtn) {
        const screenId = screenBtn.getAttribute("data-open-screen");
        if (screenId === "rep_weekly_list") {
          window.location.href = "/follow";
          return;
        }
        const title = screenBtn.getAttribute("title") || screenBtn.textContent.trim();
        showTeamToast("正在调起组织大屏: " + title + " (已直达)");
        return;
      }

      if (target.closest("#teamBatchUrgeBtn")) {
        showTeamToast("已向 2 位未按时提交周报的项目负责人发送催报提醒通知！");
        return;
      }

      const urgeSingle = target.closest('[data-action="urgeSingle"]');
      if (urgeSingle) {
        const owner = urgeSingle.getAttribute("data-owner") || "项目负责人";
        showTeamToast("已向负责人 [" + owner + "] 发送周报催办提醒！");
        return;
      }


    });
  }

  load();
})();
