(function () {
  "use strict";

  const state = {
    scopes: [],
    deptId: positiveID(new URLSearchParams(window.location.search).get("deptId")),
    teamId: 0,
    subteamgroupId: positiveID(new URLSearchParams(window.location.search).get("subteamgroupId")),
    requestNo: 0,
  };
  const params = new URLSearchParams(window.location.search);
  const originalScope = params.get("scope") === "dept" ? "dept" : "team";
  if (originalScope === "dept") state.deptId = positiveID(params.get("scopeId") || params.get("teamgroupId"));
  else state.teamId = positiveID(params.get("scopeId") || params.get("teamgroupId"));

  function positiveID(raw) {
    const value = String(raw || "");
    return /^\d+$/.test(value) && Number(value) > 0 ? Number(value) : 0;
  }

  function esc(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch];
    });
  }

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
        rows.innerHTML = '<div class="state-placeholder">未来 30 天内没有已关联的版本窗口</div>';
        if (status) status.textContent = "0 个窗口";
        return;
      }
      rows.innerHTML = json.data.map(function (windowItem) {
        const id = positiveID(windowItem.id);
        const groupID = positiveID(windowItem.teamgroupId);
        const stage = new URLSearchParams(window.location.search).get("stage") || "";
        const href = "/schedule?windows=" + encodeURIComponent(String(id)) + "&groups=" + encodeURIComponent(String(groupID)) + (stage ? "&stage=" + encodeURIComponent(stage) : "");
        return '<article class="team-home-version-row">' +
          '<div class="team-home-version-main"><a href="' + esc(href) + '"><strong>' + esc(windowItem.name) + '</strong></a>' +
          '<span>' + esc(windowItem.teamgroup || "未命名敏捷小组") + '</span></div>' +
          '<time datetime="' + esc(windowItem.releaseDate) + '">' + esc(windowItem.releaseDate) + '</time>' +
          '<span class="team-home-version-count">' + Number(windowItem.workItemCount || 0) + ' 个工作项</span>' +
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
        return '<div class="team-home-stage-item" data-stage="' + esc(stage.status) + '">' +
          '<span>' + esc(stage.label) + '</span><strong>' + Number(stage.count || 0) + '</strong>' +
          '<small>业务 ' + Number(stage.demandCount || 0) + ' · 研发 ' + Number(stage.storyCount || 0) + '</small></div>';
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
      if (blocked) blocked.textContent = String(Number(data.blocked || 0));
      if (overdue) overdue.textContent = String(Number(data.overdue || 0));
      if (soon) soon.textContent = String(Number(data.soon || 0));
      if (status) status.textContent = "更新于 " + esc(data.updatedAt || "刚刚");
      const due = Array.isArray(data.due) ? data.due : [];
      if (rows) rows.innerHTML = due.length ? due.map(function (item) {
        return '<article class="team-home-due-row' + (item.overdue ? ' is-overdue' : '') + '">' +
          '<div class="team-home-due-main"><strong>' + esc(item.title || "未命名事项") + '</strong>' +
          '<span>' + esc(item.kind === "task" ? "任务" : "需求") + ' · ' + esc(item.owner || "未分配") + '</span></div>' +
          '<time datetime="' + esc(item.deadline || "") + '">' + esc(item.deadline || "无期限") + '</time></article>';
      }).join("") : '<div class="state-placeholder">未来 3 个工作日内没有临期或逾期事项</div>';
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

  load();
})();
