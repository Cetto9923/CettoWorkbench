/**
 * 文件: web/static/js/permission/list.js
 * 模块: 权限配置
 * 职责: 工作台内嵌权限配置 — 4 分类 + 8 角色 Tag OR/跨维 AND + Drawer + 角色页面 ACL
 * 模式: embedded（PMO 组织管理）优先；兼容旧独立页（若仍加载）
 */
(function () {
  const ROLE_LABELS = {
    po: "PO",
    sm: "SM",
    pm: "PM",
    biz: "业务",
    dev: "开发",
    qa: "测试",
    lead: "团队管理",
    pmo: "PMO",
  };
  const GROUP_LABELS = {
    biz: "业务",
    lead: "团队管理",
    pmo: "PMO",
    normal: "普通用户",
  };
  const GRANT_SOURCE_LABELS = {
    user_role_grant: "手工单人授权",
    profile_preferred: "个人资料自选",
    role_group_mapping: "组织角色映射",
  };
  const GROUP_TIP = {
    biz: "业务角色用户",
    lead: "组织自动 + 手工授权",
    pmo: "PMO / 管理员统一授权",
    normal: "全部有效普通账号",
  };
  const state = {
    keyword: "",
    group: "",
    roles: [],
    deptId: 0,
    page: 1,
  };

  function apiHeaders() {
    return {
      Accept: "application/json",
      "X-Requested-With": "XMLHttpRequest",
    };
  }

  function selectedRoles() {
    return state.roles.slice();
  }

  function syncActiveFilters() {
    const host = document.getElementById("activeFilters");
    if (!host) return;
    const roles = selectedRoles();
    host.innerHTML = '<span class="af-label">已筛选：</span>';
    if (!roles.length) {
      host.innerHTML += '<span class="af-tag placeholder">未选择角色</span>';
      return;
    }
    roles.forEach(function (r) {
      host.innerHTML +=
        '<span class="af-tag" data-af-role="' +
        r +
        '">角色：' +
        (ROLE_LABELS[r] || r) +
        ' <button type="button" onclick="permissionRemoveRole(\'' +
        r +
        "')\">×</button></span>";
    });
  }

  function buildQuery() {
    const q = new URLSearchParams();
    q.set("page", String(state.page || 1));
    q.set("pageSize", "20");
    if (state.keyword) q.set("keyword", state.keyword);
    if (state.group) q.set("group", state.group);
    if (state.deptId > 0) q.set("deptId", String(state.deptId));
    state.roles.forEach(function (r) {
      q.append("roles", r);
    });
    return q.toString();
  }

  function renderSummary(groupCounts) {
    const host = document.getElementById("permissionSummary");
    if (!host) return;
    host.innerHTML = "";
    (groupCounts || []).forEach(function (g) {
      const card = document.createElement("div");
      card.className =
        "sum-card" +
        (state.group === g.code ? " active" : "") +
        (g.code === "biz" ? " business" : g.code === "lead" ? " lead" : g.code === "pmo" ? " pmo" : "");
      card.setAttribute("data-group", g.code);
      card.onclick = function () {
        permissionSelectGroup(g.code, card);
      };
      card.innerHTML =
        '<div class="sum-label">' +
        (g.label || g.code) +
        '</div><div class="sum-value">' +
        (g.count || 0) +
        '</div><div class="sum-tip">' +
        (GROUP_TIP[g.code] || "") +
        "</div>";
      host.appendChild(card);
    });
  }

  function renderRoleChips(roleTagCounts) {
    const host = document.getElementById("roleChips");
    if (!host) return;
    const selected = {};
    state.roles.forEach(function (r) {
      selected[r] = true;
    });
    let html = "";
    (roleTagCounts || []).forEach(function (r) {
      html +=
        '<button type="button" class="chip' +
        (selected[r.code] ? " selected" : "") +
        '" data-role="' +
        r.code +
        '" onclick="permissionToggleRole(this)"><span class="label">' +
        (r.label || r.code) +
        '</span><span class="count">' +
        (r.count || 0) +
        "</span></button>";
    });
    html += '<button type="button" class="chip clear" onclick="permissionClearRoles()">清空角色</button>';
    host.innerHTML = html;
  }

  function renderUsers(users) {
    const body = document.getElementById("userRows");
    if (!body) return;
    if (!(users || []).length) {
      body.innerHTML = '<tr><td colspan="6" class="empty">无匹配用户</td></tr>';
      return;
    }
    let html = "";
    users.forEach(function (u) {
      const avatar = (u.realname || u.account || "?").slice(0, 1);
      const tags = (u.roles || [])
        .map(function (r) {
          return '<span class="tag blue">' + (ROLE_LABELS[r] || r) + "</span>";
        })
        .join("");
      html +=
        '<tr data-account="' +
        u.account +
        '"><td><div class="person"><div class="mini-avatar">' +
        avatar +
        '</div><div><div class="name">' +
        (u.realname || "") +
        '</div><div class="sub">' +
        (u.account || "") +
        "</div></div></div></td><td>" +
        (u.deptName || "") +
        '</td><td><div class="tags">' +
        tags +
        '</div></td><td><span class="tag green">' +
        (GROUP_LABELS[u.group] || u.group || "") +
        "</span></td><td>" +
        (GRANT_SOURCE_LABELS[u.grantSource] || u.grantSource || "") +
        '</td><td><button type="button" class="link" onclick="permissionOpenDetail(\'' +
        u.account +
        "')\">权限详情</button></td></tr>";
    });
    body.innerHTML = html;
  }

  function loadList() {
    const loading = document.getElementById("permissionListLoading");
    if (loading) loading.style.display = "block";
    fetch("/admin/permissions?" + buildQuery(), { headers: apiHeaders() })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        if (loading) loading.style.display = "none";
        if (!json.success) {
          renderUsers([]);
          return;
        }
        const data = json.data || {};
        if (json.pageInventory) {
          window.__PERMISSION_PAGES__ = json.pageInventory;
        }
        renderSummary(data.groupCounts);
        renderRoleChips(data.roleTagCounts);
        renderUsers(data.users);
        syncActiveFilters();
      })
      .catch(function () {
        if (loading) loading.style.display = "none";
      });
  }

  window.permissionReload = loadList;

  window.permissionSelectGroup = function (group, el) {
    state.group = state.group === group ? "" : group || "";
    document.querySelectorAll(".permission-summary .sum-card").forEach(function (c) {
      c.classList.toggle("active", c.getAttribute("data-group") === state.group);
    });
    state.page = 1;
    loadList();
  };

  window.permissionToggleRole = function (btn) {
    if (!btn || !btn.getAttribute("data-role")) return;
    const role = btn.getAttribute("data-role");
    const idx = state.roles.indexOf(role);
    if (idx >= 0) state.roles.splice(idx, 1);
    else state.roles.push(role);
    btn.classList.toggle("selected");
    state.page = 1;
    loadList();
  };

  window.permissionRemoveRole = function (role) {
    state.roles = state.roles.filter(function (r) {
      return r !== role;
    });
    state.page = 1;
    loadList();
  };

  window.permissionClearRoles = function () {
    state.roles = [];
    state.page = 1;
    loadList();
  };

  window.permissionSearch = function () {
    const kw = document.getElementById("permKeyword");
    const dept = document.getElementById("permDeptId");
    state.keyword = kw ? String(kw.value || "").trim() : "";
    state.deptId = dept && dept.value ? parseInt(dept.value, 10) || 0 : 0;
    state.page = 1;
    loadList();
  };

  window.permissionReset = function () {
    state.keyword = "";
    state.group = "";
    state.roles = [];
    state.deptId = 0;
    state.page = 1;
    const kw = document.getElementById("permKeyword");
    const dept = document.getElementById("permDeptId");
    if (kw) kw.value = "";
    if (dept) dept.value = "";
    loadList();
  };

  window.permissionSwitchTab = function (name) {
    document.querySelectorAll(".permission-tabs button").forEach(function (b) {
      b.classList.toggle("active", b.getAttribute("data-tab") === name);
    });
    document.querySelectorAll(".permission-tab-panel").forEach(function (p) {
      p.classList.toggle("active", p.id === "tab-" + name);
    });
    if (name === "role-pages" && typeof window.permissionLoadRolePages === "function") {
      window.permissionLoadRolePages("pmo");
    }
    if (name === "audit" && typeof window.permissionLoadAuditLogs === "function") {
      window.permissionLoadAuditLogs();
    }
  };

  window.permissionMount = function () {
    loadList();
  };

  document.addEventListener("DOMContentLoaded", function () {
    if (document.getElementById("permissionEmbedRoot")) return;
  });
})();
