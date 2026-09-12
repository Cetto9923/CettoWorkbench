/**
 * 文件: web/static/js/permission/drawer.js
 * 模块: 权限配置
 * 职责: Drawer 详情 / grant·revoke / 角色页面 ACL / 操作记录
 */
(function () {
  const ROLE_LABELS = {
    po: "PO", sm: "SM", pm: "PM", biz: "业务",
    dev: "开发", qa: "测试", lead: "团队管理", pmo: "PMO",
  };
  const SOURCE_LABELS = {
    user_grant: "手工单人授权",
    user_role_grant: "手工单人授权",
    profile_preferred: "个人资料自选",
    org_auto: "组织角色映射",
    role_group_mapping: "组织角色映射",
    mapping: "组织角色映射",
    fallback: "组织角色映射（兼容）",
    profile: "个人资料自选",
  };
  const MIN_ACCESS = {
    "pmo.permission": "write",
    "pmo.permission.user-grant": "write",
    "pmo.permission.role-page-acl": "write",
    "pmo.permission.audit": "read",
  };
  const LEVEL_RANK = { none: 0, read: 1, write: 2 };
  const ACCESS_LABELS = { none: "无权限", read: "只读", write: "可写" };
  function accessLabel(lv) {
    return ACCESS_LABELS[lv] || lv || "—";
  }
  function sourceLabel(raw) {
    const key = String(raw || "").trim();
    return SOURCE_LABELS[key] || key || "—";
  }
  function sourcesText(list) {
    if (!list || !list.length) return "—";
    return list
      .map(function (s) {
        return sourceLabel(s);
      })
      .join(" / ");
  }
  function apiHeaders() {
    return { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" };
  }
  window.permissionOpenDetail = function (account) {
    const mask = document.getElementById("permissionDrawer");
    const body = document.getElementById("drawerBody");
    const title = document.getElementById("drawerTitle");
    const acc = document.getElementById("drawerAccount");
    if (!mask || !body) return;
    mask.classList.add("show");
    body.innerHTML = '<div class="loading">加载中…</div>';
    if (acc) acc.textContent = account || "";
    fetch("/admin/permissions/detail?account=" + encodeURIComponent(account), { headers: apiHeaders() })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        if (!json.success) {
          body.innerHTML = '<div class="info">' + (json.message || "加载失败") + "</div>";
          return;
        }
        const d = json.data || {};
        if (title) title.textContent = (d.realname || account) + " · 权限详情";
        if (acc) acc.textContent = d.account || account;
        let html = "";
        html += '<div class="detail-section"><div class="detail-title">当前有效角色</div>';
        (d.effectiveRoles || []).forEach(function (r) {
          const canManage = r.code === "lead" || r.code === "pmo";
          html +=
            '<div class="role-line"><strong>' +
            (r.label || ROLE_LABELS[r.code] || r.code) +
            '</strong><div><span class="tag blue">有效</span> ' +
            sourcesText(r.sources) +
            "</div>";
          if (canManage) {
            html +=
              '<span class="info" style="margin:0;font-size:11px">见下方授权区</span>';
          } else {
            html +=
              '<button type="button" class="link" disabled title="普通角色请在个人资料中自选">个人资料自选</button>';
          }
          html += "</div>";
        });
        html += "</div>";
        html +=
          '<div class="detail-section"><div class="detail-title">用户自选视图</div><div class="info">' +
          ((d.profilePreferred || []).map(function (k) {
            return ROLE_LABELS[k] || k;
          }).join("、 ") || "未初始化（走兼容 fallback）") +
          "</div></div>";

        html += '<div class="detail-section"><div class="detail-title">受控授权记录（可逐个撤销）</div>';
        if (!(d.grants || []).length) {
          html +=
            '<div class="info">暂无手工单人授权。若列表「授权来源」为组织角色映射，' +
            "请到 <a class=\"link\" href=\"/admin/workbench-roles\">维护角色映射</a> " +
            "调整禅道用户组 ↔ 角色，或把用户移出对应用户组；此处无法直接「撤销映射」。</div>";
        } else {
          (d.grants || []).forEach(function (g) {
            html +=
              '<div class="role-line"><strong>' +
              (ROLE_LABELS[g.roleCode] || g.roleCode) +
              "</strong><div>部门=" +
              (g.scopeDeptId || "不限") +
              " · 含下级=" +
              (g.scopeIncludeChildren ? "是" : "否") +
              " · 小组=" +
              (g.scopeTeamId || "无") +
              '</div><button type="button" class="link" onclick="permissionRevoke(' +
              g.id +
              ')">撤销</button></div>';
          });
        }
        html += "</div>";

        html +=
          '<div class="detail-section"><div class="detail-title">快捷授权（手工任命）</div>' +
          '<div class="info" style="margin-bottom:8px">团队管理必须指定部门或敏捷小组范围；PMO 为全局权限。</div>' +
          '<div class="grant-form">' +
          '<div class="row"><label>部门 ID <input type="number" id="permGrantDeptId" min="0" placeholder="如 12"></label>' +
          '<label>敏捷小组 ID <input type="number" id="permGrantTeamId" min="0" placeholder="可选"></label>' +
          '<label><input type="checkbox" id="permGrantIncludeChildren"> 含下级部门</label></div>' +
          '<div class="row">' +
          '<button type="button" class="btn primary" onclick="permissionGrant(\'' +
          d.account +
          "','lead')\">任命团队管理</button> " +
          '<button type="button" class="btn primary" onclick="permissionGrant(\'' +
          d.account +
          "','pmo')\">任命 PMO</button></div></div></div>";
        body.innerHTML = html;
      })
      .catch(function () {
        body.innerHTML = '<div class="info">网络错误</div>';
      });
  };

  window.permissionCloseDetail = function () {
    const mask = document.getElementById("permissionDrawer");
    if (mask) mask.classList.remove("show");
  };

  window.permissionGrant = function (account, roleCode) {
    let scopeDeptId = 0;
    let scopeTeamId = 0;
    let scopeIncludeChildren = false;
    if (roleCode === "lead") {
      const deptEl = document.getElementById("permGrantDeptId");
      const teamEl = document.getElementById("permGrantTeamId");
      const icEl = document.getElementById("permGrantIncludeChildren");
      scopeDeptId = deptEl && deptEl.value ? parseInt(deptEl.value, 10) || 0 : 0;
      scopeTeamId = teamEl && teamEl.value ? parseInt(teamEl.value, 10) || 0 : 0;
      scopeIncludeChildren = !!(icEl && icEl.checked);
      if (scopeDeptId <= 0 && scopeTeamId <= 0) {
        const msg = "任命团队管理须填写部门 ID 或敏捷小组 ID（不可无限范围）";
        if (typeof showToast === "function") showToast(msg, "error");
        else alert(msg);
        return;
      }
    }
    fetch("/admin/permissions/grants", {
      method: "POST",
      headers: Object.assign({ "Content-Type": "application/json" }, apiHeaders()),
      body: JSON.stringify({
        account: account,
        roleCode: roleCode,
        scopeDeptId: scopeDeptId,
        scopeIncludeChildren: scopeIncludeChildren,
        scopeTeamId: scopeTeamId,
        remark: "",
      }),
    })
      .then(function (r) {
        return r.json().then(function (j) {
          return { status: r.status, json: j };
        });
      })
      .then(function (res) {
        if (res.json.success) {
          if (typeof showToast === "function") showToast(res.json.message || "授权成功", "success");
          permissionOpenDetail(account);
          if (typeof window.permissionReload === "function") window.permissionReload();
        } else {
          const msg =
            (res.json.errors && res.json.errors[0] && res.json.errors[0].message) ||
            res.json.message ||
            "授权失败";
          if (typeof showToast === "function") showToast(msg, "error");
          else alert(msg);
        }
      });
  };

  window.permissionRevoke = function (id) {
    if (!confirm("确认撤销该授权？")) return;
    fetch("/admin/permissions/grants/" + id, {
      method: "DELETE",
      headers: apiHeaders(),
    })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        if (json.success) {
          if (typeof showToast === "function") showToast(json.message || "已撤销", "success");
          const acc = document.getElementById("drawerAccount");
          if (acc && acc.textContent) permissionOpenDetail(acc.textContent);
          if (typeof window.permissionReload === "function") window.permissionReload();
        } else if (typeof showToast === "function") {
          showToast(json.message || "撤销失败", "error");
        }
      });
  };

  function loadRolePages(roleCode) {
    const host = document.getElementById("rolePagesBody");
    if (!host) return;
    host.innerHTML = '<div class="loading">加载中…</div>';
    const roleSelect = document.getElementById("rolePageRole");
    if (roleSelect && roleCode) roleSelect.value = roleCode;
    const role = (roleSelect && roleSelect.value) || roleCode || "po";
    const ensureInventory = fetch("/admin/permissions?page=1&pageSize=1", { headers: apiHeaders() })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        if (json.pageInventory) window.__PERMISSION_PAGES__ = json.pageInventory;
      });
    ensureInventory
      .then(function () {
        return fetch("/admin/permissions/role-pages?roleCode=" + encodeURIComponent(role), {
          headers: apiHeaders(),
        });
      })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        const pages = window.__PERMISSION_PAGES__ || [];
        const aclMap = {};
        (json.data || []).forEach(function (row) {
          aclMap[row.pageKey] = row.accessLevel;
        });
        let html =
          '<table class="acl-matrix"><thead><tr><th>页面</th><th>默认</th><th>当前</th></tr></thead><tbody>';
        pages
          .filter(function (p) {
            return p.roleCode === role;
          })
          .forEach(function (p) {
            const min = MIN_ACCESS[p.code];
            const current = aclMap[p.code] || p.defaultAccess;
            const options = ["none", "read", "write"].filter(function (lv) {
              if (!min) return true;
              return LEVEL_RANK[lv] >= LEVEL_RANK[min];
            });
            const title = p.label || p.code;
            html +=
              "<tr><td><div class=\"page-name\">" +
              title +
              '</div><div class="page-code">' +
              p.code +
              "</div></td><td>" +
              accessLabel(p.defaultAccess) +
              '</td><td><select data-page-key="' +
              p.code +
              '"' +
              (min ? ' title="系统保底最低：' + accessLabel(min) + '"' : "") +
              ">";
            options.forEach(function (lv) {
              html +=
                '<option value="' +
                lv +
                '"' +
                (lv === current ? " selected" : "") +
                ">" +
                accessLabel(lv) +
                "</option>";
            });
            html += "</select></td></tr>";
          });
        html += "</tbody></table>";
        html +=
          '<div style="margin-top:12px"><button type="button" class="btn primary" onclick="permissionSaveRolePages()">保存</button></div>';
        host.innerHTML = html;
      });
  }

  window.permissionLoadRolePages = loadRolePages;

  window.permissionSaveRolePages = function () {
    const roleSelect = document.getElementById("rolePageRole");
    const role = (roleSelect && roleSelect.value) || "po";
    const selects = document.querySelectorAll("#rolePagesBody select[data-page-key]");
    const pages = [];
    selects.forEach(function (sel) {
      pages.push({ pageKey: sel.getAttribute("data-page-key"), accessLevel: sel.value });
    });
    fetch("/admin/permissions/role-pages/" + encodeURIComponent(role), {
      method: "PUT",
      headers: Object.assign({ "Content-Type": "application/json" }, apiHeaders()),
      body: JSON.stringify({ pages: pages }),
    })
      .then(function (r) {
        return r.json().then(function (j) {
          return { status: r.status, json: j };
        });
      })
      .then(function (res) {
        if (res.json.success) {
          if (typeof showToast === "function") showToast(res.json.message || "已保存", "success");
          loadRolePages(role);
        } else {
          const msg =
            (res.json.errors && res.json.errors[0] && res.json.errors[0].message) ||
            res.json.message ||
            "保存失败";
          if (typeof showToast === "function") showToast(msg, "error");
          else alert(msg);
        }
      });
  };

  function loadAuditLogs() {
    const host = document.getElementById("auditLogsBody");
    if (!host) return;
    host.innerHTML = '<div class="loading">加载中…</div>';
    fetch("/admin/permissions/audit-logs?limit=50", { headers: apiHeaders() })
      .then(function (r) {
        return r.json();
      })
      .then(function (json) {
        const rows = json.data || [];
        if (!rows.length) {
          host.innerHTML = '<div class="info">暂无操作记录</div>';
          return;
        }
        let html =
          "<table><thead><tr><th>时间</th><th>操作人</th><th>动作</th><th>目标</th><th>角色</th><th>备注</th></tr></thead><tbody>";
        rows.forEach(function (r) {
          html +=
            "<tr><td>" +
            (r.createdAt || "") +
            "</td><td>" +
            (r.operatorAccount || "") +
            "</td><td>" +
            (r.actionType || "") +
            "</td><td>" +
            (r.targetName || r.targetType || "") +
            "</td><td>" +
            (r.roleCode || "") +
            "</td><td>" +
            (r.remark || "") +
            "</td></tr>";
        });
        html += "</tbody></table>";
        host.innerHTML = html;
      });
  }

  window.permissionLoadAuditLogs = loadAuditLogs;

})();
