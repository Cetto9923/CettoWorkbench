// =============================================================================
// 文件: web/static/js/po/home-team-hierarchy.js
// 模块: 团队长工作台 - 团队与小组层级展示
// 职责: 请求 /team/hierarchy 动态加载团队、小组与成员，完成Tab切换与去重展示
// =============================================================================

(function () {
  "use strict";

  const state = {
    hierarchyData: null,
    activeGroupID: 0,
    selectedTeamID: 0,
    isLoading: false,
  };

  function esc(s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function getQueryTeamID() {
    const params = new URLSearchParams(window.location.search);
    const val = params.get("teamId") || params.get("teamgroupId") || "";
    const n = Number(val);
    return !isNaN(n) && n > 0 ? n : 0;
  }

  function setURLTeamID(teamID) {
    const url = new URL(window.location.href);
    if (teamID > 0) {
      url.searchParams.set("teamId", String(teamID));
    } else {
      url.searchParams.delete("teamId");
    }
    window.history.replaceState(null, "", url.pathname + url.search + url.hash);
  }

  async function loadHierarchy(teamID) {
    const container = document.getElementById("teamHierarchyContainer");
    if (!container) return;

    state.isLoading = true;
    container.innerHTML = '<div class="state-placeholder">正在加载敏捷团队与小组层级…</div>';

    try {
      const url = teamID > 0 ? "/team/hierarchy?teamId=" + teamID : "/team/hierarchy";
      const resp = await fetch(url, { credentials: "same-origin" });
      if (resp.status === 403) {
        container.innerHTML =
          '<div class="th-empty-card" role="alert"><div class="th-empty-icon"><i class="fas fa-ban"></i></div><p class="th-empty-text">您无权访问该团队数据</p></div>';
        return;
      }
      if (!resp.ok) {
        throw new Error("HTTP " + resp.status);
      }
      const json = await resp.json();
      if (!json.success || !json.data) {
        throw new Error(json.message || "数据加载失败");
      }

      state.hierarchyData = json.data;
      if (state.hierarchyData.currentTeam) {
        state.selectedTeamID = state.hierarchyData.currentTeam.id;
        setURLTeamID(state.selectedTeamID);
      }

      // 默认选中第一个子小组
      if (state.hierarchyData.subGroups && state.hierarchyData.subGroups.length > 0) {
        const exist = state.hierarchyData.subGroups.find(function (g) {
          return g.id === state.activeGroupID;
        });
        if (!exist) {
          state.activeGroupID = state.hierarchyData.subGroups[0].id;
        }
      } else {
        state.activeGroupID = 0;
      }

      renderHierarchy();
    } catch (err) {
      container.innerHTML =
        '<div class="th-empty-card" role="alert"><div class="th-empty-icon"><i class="fas fa-circle-exclamation"></i></div><p class="th-empty-text">' +
        esc(err.message || "加载团队层级数据失败") +
        '</p><button type="button" class="retry-btn" id="thRetryBtn">重试</button></div>';
      const retryBtn = document.getElementById("thRetryBtn");
      if (retryBtn) {
        retryBtn.addEventListener("click", function () {
          loadHierarchy(teamID);
        });
      }
    } finally {
      state.isLoading = false;
    }
  }

  function renderHierarchy() {
    const container = document.getElementById("teamHierarchyContainer");
    if (!container || !state.hierarchyData) return;

    const data = state.hierarchyData;
    if (!data.currentTeam) {
      container.innerHTML =
        '<div class="th-empty-card"><div class="th-empty-icon"><i class="fas fa-users-slash"></i></div><p class="th-empty-text">当前账号暂无已关联的敏捷团队</p></div>';
      return;
    }

    const team = data.currentTeam;
    const subGroups = data.subGroups || [];
    const authorizedTeams = data.AuthorizedTeams || data.authorizedTeams || [];

    let teamSelectHTML = "";
    if (authorizedTeams.length > 1) {
      const opts = authorizedTeams
        .map(function (t) {
          const sel = t.id === team.id ? "selected" : "";
          return '<option value="' + t.id + '" ' + sel + ">" + esc(t.name) + "</option>";
        })
        .join("");
      teamSelectHTML =
        '<select class="th-team-selector" id="thTeamSelect" aria-label="切换团队">' +
        opts +
        "</select>";
    }

    // 1. 横幅
    let html = '<div class="th-container">';
    html += '<div class="th-banner-card">';
    html += '  <div class="th-banner-left">';
    html += '    <div class="th-team-title-wrap">';
    html += '      <div class="th-team-icon"><i class="fas fa-sitemap"></i></div>';
    html += '      <h2 class="th-team-title">' + esc(team.name) + "</h2>";
    html += teamSelectHTML;
    html += "    </div>";
    html += '    <div class="th-banner-meta">';
    html +=
      '      <span class="th-meta-item">团队长:<strong>' +
      esc(team.leader ? team.leader.name : "—") +
      "</strong></span>";
    html +=
      '      <span class="th-meta-item">产品经理:<strong>' +
      esc(team.po ? team.po.name : "—") +
      "</strong></span>";
    html += "    </div>";
    html += "  </div>";
    html += '  <div class="th-banner-right">';
    html += '    <div class="th-stat-pill">';
    html += '      <span class="th-stat-label">团队总人数 (去重)</span>';
    html += '      <strong class="th-stat-num">' + team.totalUniqueMembers + "</strong>";
    html += "    </div>";
    html += "  </div>";
    html += "</div>";

    // 2. 子小组卡片与 Tab
    html += '<div class="th-group-tabs-card">';
    if (subGroups.length === 0) {
      html +=
        '<div class="th-empty-card"><p class="th-empty-text">该团队暂无配置子敏捷小组</p></div>';
    } else {
      // Tab 栏
      html += '<div class="th-tabs-nav" role="tablist">';
      subGroups.forEach(function (g) {
        const activeCls = g.id === state.activeGroupID ? " active" : "";
        html +=
          '<button type="button" class="th-tab-btn' +
          activeCls +
          '" role="tab" data-group-id="' +
          g.id +
          '">';
        html += '<i class="fas fa-users"></i>';
        html += "<span>" + esc(g.name) + "</span>";
        html += '<span class="th-tab-count">' + g.memberCount + "人</span>";
        html += "</button>";
      });
      html += "</div>";

      // 选中的小组信息与成员列表
      const curGroup = subGroups.find(function (g) {
        return g.id === state.activeGroupID;
      }) || subGroups[0];

      html += '<div class="th-group-summary-card">';
      html +=
        '  <div class="th-summary-group-title"><i class="fas fa-layer-group"></i> ' +
        esc(curGroup.name) +
        "</div>";
      const coach = curGroup.scrumMaster && curGroup.scrumMaster.name ? curGroup.scrumMaster.name : "未配置";
      const po = curGroup.po && curGroup.po.name ? curGroup.po.name : "未配置";
      const coachCls = coach === "未配置" ? ' class="th-role-unconfigured"' : "";
      const poCls = po === "未配置" ? ' class="th-role-unconfigured"' : "";

      html += '  <div class="th-summary-roles">';
      html +=
        '    <span>敏捷教练: <b' + coachCls + '>' +
        esc(coach) +
        "</b></span>";
      html +=
        '    <span>产品负责人: <b' + poCls + '>' +
        esc(po) +
        "</b></span>";
      html += "    <span>组内人数: <b>" + curGroup.memberCount + " 人</b></span>";
      html += "  </div>";
      html += "</div>";

  function getAvatarText(name, account) {
    var raw = String(name || account || "").trim();
    var clean = raw.replace(/\([^)]*\)/g, "").replace(/（[^）]*）/g, "").trim();
    if (!clean) clean = String(account || "").trim();
    if (!clean) return "人";
    return clean.slice(0, 2);
  }

      // 成员网格
      html += '<div class="th-members-grid">';
      if (!curGroup.members || curGroup.members.length === 0) {
        html +=
          '<div class="th-empty-card" style="grid-column: 1 / -1;"><p class="th-empty-text">暂无成员记录</p></div>';
      } else {
        curGroup.members.forEach(function (m) {
          const avatarChar = getAvatarText(m.name, m.account);
          html += '<div class="th-member-card">';
          html += '  <div class="th-member-avatar">' + esc(avatarChar) + "</div>";
          html += '  <div class="th-member-info">';
          html += '    <div class="th-member-name-row">';
          html += '      <strong class="th-member-name">' + esc(m.name) + "</strong>";
          if (m.isMultiGroup) {
            const othersText =
              m.otherGroupNames && m.otherGroupNames.length > 0
                ? " (" + m.otherGroupNames.join(",") + ")"
                : "";
            html +=
              '      <span class="th-cross-badge" title="该成员同时参与其他小组' +
              esc(othersText) +
              '"><i class="fas fa-arrows-split-up-and-left"></i> 跨组' +
              esc(othersText) +
              "</span>";
          }
          html += "    </div>";
          html +=
            '    <div class="th-member-account">工号: ' + esc(m.account) + "</div>";
          if (m.role) {
            html +=
              '    <div class="th-member-role">岗位: ' + esc(m.role) + "</div>";
          }
          html += "  </div>";
          html += "</div>";
        });
      }
      html += "</div>"; // end th-members-grid
    }
    html += "</div>"; // end th-group-tabs-card
    html += "</div>"; // end th-container

    container.innerHTML = html;

    // 事件监听绑定
    const selectEl = document.getElementById("thTeamSelect");
    if (selectEl) {
      selectEl.addEventListener("change", function (e) {
        const newTeamID = Number(e.target.value);
        if (newTeamID > 0) {
          loadHierarchy(newTeamID);
        }
      });
    }

    const tabBtns = container.querySelectorAll(".th-tab-btn");
    tabBtns.forEach(function (btn) {
      btn.addEventListener("click", function () {
        const gid = Number(btn.getAttribute("data-group-id"));
        if (gid > 0 && gid !== state.activeGroupID) {
          state.activeGroupID = gid;
          renderHierarchy();
        }
      });
    });
  }

  // 页面入口挂载
  function init() {
    const container = document.getElementById("teamHierarchyContainer");
    if (!container) return;
    const teamID = getQueryTeamID();
    loadHierarchy(teamID);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
