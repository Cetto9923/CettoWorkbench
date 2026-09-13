/**
 * PMO：调整成员与角色（提交调整单，不直写正式名单）
 *
 * 交互原则：
 * 1. 当前正式成员就是“成员名单”，不再让用户逐行选择“调整类型”。
 * 2. 添加人员 = 自动生成 add；点击移除 = 自动生成 remove；修改角色/工时 = 自动生成 roleChange。
 * 3. 待移除成员保留在列表中并可撤销，避免误操作后失去上下文。
 * 4. 仅提交真正发生变化的成员，兼容现有 adjustments API 合同。
 *
 * 成员角色：前端默认选项（研发/测试/…）+ 当前名单已有角色；落库字段为 zt_team.role（自由文本）。
 * 人员检索：统一 WbPersonPicker（姓名下显示 zt_dept.name）。
 */
(function () {
  function S() { return window.__at || {}; }
  var esc = window.escapeHtml;
  function apiFetch(p, o) { return S().apiFetch(p, o); }
  function isLeadView() { return S().isLeadView(); }
  function person(n, a) { return S().person(n, a); }
  function st() { return S().state || {}; }

  // 角色下拉默认项：前端写死；后端无独立角色字典，存 zt_team.role。
  var DEFAULT_TEAM_ROLES = ["研发", "测试", "产品负责人", "敏捷教练", "项目经理", "架构师"];

  var draft = [];
  var teamId = 0;
  var directoryByAccount = {};

  function numberEq(a, b) {
    return Math.abs((Number(a) || 0) - (Number(b) || 0)) < 0.0001;
  }

  function deptOf(account, fallback) {
    var hit = directoryByAccount[String(account || "").trim()];
    var dept = String((fallback != null ? fallback : (hit && hit.deptName)) || "").trim();
    return dept;
  }

  function calcFormalAction(m) {
    if (!m || m.origin !== "formal") return (m && m.actionType) || "keep";
    if (m.actionType === "remove") return "remove";
    return (String(m.role || "") !== String(m.originalRole || "") || !numberEq(m.hours, m.originalHours))
      ? "roleChange"
      : "keep";
  }

  function changeCounts() {
    var out = { add: 0, remove: 0, roleChange: 0 };
    draft.forEach(function (m) {
      if (out[m.actionType] != null) out[m.actionType]++;
    });
    return out;
  }

  function roleSuggestions() {
    var roles = DEFAULT_TEAM_ROLES.slice();
    draft.forEach(function (m) {
      var r = String(m.role || "").trim();
      if (r && roles.indexOf(r) < 0) roles.push(r);
    });
    return roles;
  }

  function roleSelectHtml(selected, disabled, index) {
    var roles = roleSuggestions();
    var cur = String(selected || "").trim();
    var opts = roles.map(function (r) {
      return '<option value="' + esc(r) + '"' + (r === cur ? " selected" : "") + ">" + esc(r) + "</option>";
    }).join("");
    if (cur && roles.indexOf(cur) < 0) {
      opts = '<option value="' + esc(cur) + '" selected>' + esc(cur) + "</option>" + opts;
    }
    return '<select class="at-field at-member-role-select" ' +
      (disabled ? "disabled " : "") +
      'onchange="atDraftField(' + index + ',\'role\',this.value)">' + opts + "</select>";
  }

  function mountCandidatePicker() {
    var host = document.getElementById("atMemberPickerHost");
    if (!host) return;
    window.destroyUserPicker("atCandidateInput");
    host.innerHTML = '<input id="atCandidateInput" class="at-field" placeholder="搜索姓名或工号" autocomplete="off"><input id="atCandidateAccount" type="hidden">';
    var input = document.getElementById("atCandidateInput");
    var hidden = document.getElementById("atCandidateAccount");
    var timer, revision = 0;
    window.initUserPicker(input.id, hidden.id, []);
    input.addEventListener("input", function () {
      var query = input.value.trim(), current = ++revision;
      clearTimeout(timer);
      timer = setTimeout(function () {
        apiFetch(S().API + "/candidates?q=" + encodeURIComponent(query)).then(function (json) {
          if (current !== revision || !input.isConnected) return;
          var items = (json.data || []).map(function (u) {
            directoryByAccount[u.account] = u;
            return {value:u.account, label:u.name + "(" + u.account + ")", pinyin:u.pinyin || ""};
          });
          window.initUserPicker(input.id, hidden.id, items);
          if (document.activeElement === input) input.dispatchEvent(new Event("focus"));
        }).catch(function (err) { if (current === revision) window.showToast(err.message); });
      }, 250);
    });
    hidden.addEventListener("change", function () {
      var account = hidden.value;
      if (!account) return;
      revision++;
      clearTimeout(timer);
      var person = directoryByAccount[account] || {};
      window.atAddCand(account, person.name || account, "");
      window.clearAutocomplete(input.id);
    });
  }

  window.atOpenMemberEdit = function (id) {
    if (isLeadView()) return;
    teamId = Number(id) || 0;
    var d = st().lastDetail || {};
    draft = [];
    (d.formal || []).forEach(function (m) {
      draft.push({
        account: m.account,
        name: m.name,
        deptName: "",
        role: m.role || "",
        hours: Number(m.hours) || 0,
        originalRole: m.role || "",
        originalHours: Number(m.hours) || 0,
        actionType: "keep",
        origin: "formal"
      });
    });
    ensureMask();
    var teamName = document.getElementById("atAdjustTeamName");
    var count = document.getElementById("atAdjustMemberCount");
    var reason = document.getElementById("atAdjustReason");
    if (teamName) teamName.textContent = d.name || "当前敏捷小组";
    if (count) count.textContent = "当前正式成员 " + draft.length + " 人";
    if (reason) reason.value = "";
    mountCandidatePicker();
    renderDraft();
    document.getElementById("atAdjustMask").classList.add("open");
  };

  window.atCloseMemberEdit = function () {
    var mask = document.getElementById("atAdjustMask");
    if (mask) mask.classList.remove("open");
  };

  function ensureMask() {
    if (document.getElementById("atAdjustMask")) return;
    var el = document.createElement("div");
    el.id = "atAdjustMask";
    el.className = "at-drawer-mask";
    el.onclick = function (e) { if (e.target === el) window.atCloseMemberEdit(); };
    el.innerHTML =
      '<div class="at-modal at-member-edit-modal" role="dialog" aria-label="调整成员与角色">' +
        '<div class="at-modal-head">' +
          '<div><h3>调整成员与角色</h3>' +
          '<div class="at-modal-sub">维护正式成员名单与成员角色；提交后进入组织级敏捷教练确认流程</div></div>' +
          '<button type="button" class="close" onclick="atCloseMemberEdit()">×</button>' +
        '</div>' +
        '<div class="at-modal-body">' +
          '<div class="at-member-edit-teamline">' +
            '<strong id="atAdjustTeamName">当前敏捷小组</strong>' +
            '<span class="at-member-count" id="atAdjustMemberCount"></span>' +
            '<span class="at-member-edit-tip">系统会根据操作自动识别新增、移除和角色调整</span>' +
          '</div>' +
          '<div class="at-member-addbar">' +
            '<div class="at-member-search-wrap">' +
              '<div id="atMemberPickerHost" class="at-member-picker-host"></div>' +
            '</div>' +
            '<span class="at-member-add-hint">选择后加入名单（姓名下显示部门）</span>' +
          '</div>' +
          '<div class="at-section-head at-member-edit-section-head">' +
            '<h4>成员名单</h4>' +
            '<span class="at-section-note">直接修改成员角色 / 工时，或移除成员</span>' +
            '<div class="at-member-change-legend">' +
              '<span><i class="add"></i>新增</span><span><i class="remove"></i>待移除</span><span><i class="change"></i>角色 / 工时调整</span>' +
            '</div>' +
          '</div>' +
          '<div id="atDraftTable"></div>' +
          '<div id="atAdjustSummary" class="at-adjust-summary"></div>' +
          '<textarea id="atAdjustReason" class="textarea at-adjust-reason" placeholder="调整说明（可选），例如：项目分工调整、人员轮换等"></textarea>' +
        '</div>' +
        '<div class="at-modal-foot">' +
          '<span class="at-member-confirm-tip">正式名单在确认通过后生效</span>' +
          '<span style="flex:1"></span>' +
          '<button type="button" class="at-btn" onclick="atCloseMemberEdit()">取消</button>' +
          '<button type="button" class="at-btn primary" onclick="atSubmitMemberEdit()">提交成员调整</button>' +
        '</div>' +
      '</div>';
    document.body.appendChild(el);
  }

  window.atAddCand = function (account, name, deptName) {
    var existing = draft.find(function (x) { return x.account === account; });
    if (existing) {
      if (existing.actionType === "remove") {
        existing.actionType = calcFormalAction(Object.assign({}, existing, { actionType: "keep" }));
        window.showToast("已撤销该成员的移除操作");
        renderDraft();
      } else {
        window.showToast("该成员已在当前成员名单中");
      }
      return;
    }
    draft.push({
      account: account,
      name: name,
      deptName: deptName || deptOf(account),
      role: "研发",
      hours: 7,
      originalRole: "",
      originalHours: 0,
      actionType: "add",
      origin: "add"
    });
    renderDraft();
  };

  window.atRemoveDraftMember = function (i) {
    var m = draft[i];
    if (!m) return;
    if (m.origin === "add") draft.splice(i, 1);
    else m.actionType = "remove";
    renderDraft();
  };

  window.atUndoDraftMember = function (i) {
    var m = draft[i];
    if (!m) return;
    if (m.origin === "add") draft.splice(i, 1);
    else {
      m.actionType = "keep";
      m.actionType = calcFormalAction(m);
    }
    renderDraft();
  };

  window.atDraftField = function (i, field, val) {
    var m = draft[i];
    if (!m || m.actionType === "remove") return;
    if (field === "hours") m.hours = Number(val) || 0;
    else m.role = String(val || "");
    if (m.origin === "formal") m.actionType = calcFormalAction(m);
    renderChangeCell(i);
    renderSummary();
  };

  function renderDraft() {
    var host = document.getElementById("atDraftTable");
    if (!host) return;
    if (!draft.length) {
      host.innerHTML = '<div class="at-empty at-member-edit-empty">当前没有正式成员，可通过上方搜索添加。</div>';
      renderSummary();
      return;
    }
    host.innerHTML =
      '<div class="at-member-table at-member-edit-table"><table class="at-table"><thead><tr>' +
        '<th style="width:28%">成员</th>' +
        '<th style="width:24%">成员角色</th>' +
        '<th style="width:18%">可用工时/天</th>' +
        '<th style="width:18%">本次变化</th>' +
        '<th style="width:12%">操作</th>' +
      "</tr></thead><tbody>" +
      draft.map(function (m, i) {
        var removing = m.actionType === "remove";
        var rowClass = m.actionType === "add" ? "at-member-row-add"
          : m.actionType === "remove" ? "at-member-row-remove"
          : m.actionType === "roleChange" ? "at-member-row-change" : "";
        var change = changeBadge(m.actionType);
        var action = removing
          ? '<button type="button" class="at-link" onclick="atUndoDraftMember(' + i + ')">撤销移除</button>'
          : m.origin === "add"
            ? '<button type="button" class="at-link" onclick="atUndoDraftMember(' + i + ')">撤销新增</button>'
            : '<button type="button" class="at-link at-member-remove-link" onclick="atRemoveDraftMember(' + i + ')">移除</button>';
        var sub = deptOf(m.account, m.deptName) || "—";
        return '<tr class="' + rowClass + '" data-member-index="' + i + '">' +
          '<td><div class="at-member-edit-person"><strong>' + esc(person(m.name, m.account)) + "</strong><span>" + esc(sub) + "</span></div></td>" +
          "<td>" + roleSelectHtml(m.role, removing, i) + "</td>" +
          '<td><input class="at-field at-member-hours-input" type="number" min="0" step="0.5" value="' + esc(m.hours) + '"' +
            (removing ? " disabled" : "") + ' onchange="atDraftField(' + i + ',\'hours\',this.value)" /></td>' +
          '<td class="at-member-change-cell">' + change + "</td>" +
          "<td>" + action + "</td>" +
        "</tr>";
      }).join("") +
      "</tbody></table></div>";
    renderSummary();
  }

  function changeBadge(actionType) {
    if (actionType === "add") return '<span class="at-change-badge add">新增成员</span>';
    if (actionType === "remove") return '<span class="at-change-badge remove">待移除</span>';
    if (actionType === "roleChange") return '<span class="at-change-badge change">角色 / 工时调整</span>';
    return '<span class="at-change-badge none">无变化</span>';
  }

  function renderChangeCell(i) {
    var row = document.querySelector('#atDraftTable tr[data-member-index="' + i + '"]');
    var m = draft[i];
    if (!row || !m) return;
    row.classList.remove("at-member-row-add", "at-member-row-remove", "at-member-row-change");
    if (m.actionType === "add") row.classList.add("at-member-row-add");
    if (m.actionType === "remove") row.classList.add("at-member-row-remove");
    if (m.actionType === "roleChange") row.classList.add("at-member-row-change");
    var cell = row.querySelector(".at-member-change-cell");
    if (cell) cell.innerHTML = changeBadge(m.actionType);
  }

  function renderSummary() {
    var host = document.getElementById("atAdjustSummary");
    if (!host) return;
    var c = changeCounts();
    var changed = c.add + c.remove + c.roleChange;
    host.innerHTML =
      "<strong>本次调整</strong>" +
      "<span>新增 <b>" + c.add + "</b> 人</span>" +
      "<span>移除 <b>" + c.remove + "</b> 人</span>" +
      "<span>角色 / 工时调整 <b>" + c.roleChange + "</b> 人</span>" +
      '<span class="at-adjust-summary-spacer"></span>' +
      '<span class="at-adjust-summary-note">' + (changed ? "仅提交发生变化的成员" : "当前尚未产生变更") + "</span>";
  }

  window.atSubmitMemberEdit = async function () {
    if (isLeadView() || !teamId) return;
    var items = [];
    draft.forEach(function (m) {
      if (m.actionType === "keep") return;
      items.push({
        account: m.account,
        actionType: m.actionType,
        role: m.role,
        availableHours: Number(m.hours) || 0
      });
    });
    if (!items.length) {
      window.showToast("请先添加、移除成员，或修改成员角色 / 工时");
      return;
    }
    var reason = (document.getElementById("atAdjustReason") || {}).value || "";
    try {
      await apiFetch(S().API + "/" + teamId + "/adjustments", {
        method: "POST",
        body: { reason: reason, items: items }
      });
      window.showToast("成员调整已提交，待组织级敏捷教练确认");
      window.atCloseMemberEdit();
      if (typeof window.atLoadDetail === "function") window.atLoadDetail(teamId);
    } catch (e) {
      window.showToast(e.message || "提交失败");
    }
  };
})();
