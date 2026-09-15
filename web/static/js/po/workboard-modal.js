/**
 * =============================================================================
 * 模块: PO 工作看板 - 交互弹窗域
 * 职责: +提问题、+建任务、调整小组成员弹窗的数据与交互逻辑
 * =============================================================================
 */
(function () {
  "use strict";

  // 成员必须来自服务端真实小组数据；未接入时保持空态，禁止展示假成员。
  var currentAgileMembers = [];

  var candidatePool = [];

  var agileTeamRoles = ["研发", "测试", "PO", "SM", "架构", "运维", "美工", "产品经理"];
  var teamDraft = [];
  var teamSearchTerm = "";
  var teamDataRevision = 0;
  var currentTeamgroupID = 0;

  var esc = window.escapeHtml;

  /* ────────── 1. 提问题 Modal ────────── */
  function openKanbanCreateIssueModal() {
    var modal = document.getElementById("kanbanIssueModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (!modal || !overlay) return;

    var nameInput = document.getElementById("kbIssueName");
    var descInput = document.getElementById("kbIssueDesc");
    var assigneeSel = document.getElementById("kbIssueAssignee");
    if (nameInput) nameInput.value = "";
    if (descInput) descInput.value = "";

    if (assigneeSel) {
      assigneeSel.innerHTML = currentAgileMembers.map(function (m) {
        return '<option value="' + esc(m.name) + '">' + esc(m.name) + ' (' + esc(m.account) + ' · ' + esc(m.role) + ')</option>';
      }).join("");
    }

    overlay.classList.add("show");
    modal.classList.add("show");
    setTimeout(function () { if (nameInput) nameInput.focus(); }, 50);
  }

  function closeKanbanIssueModal() {
    var modal = document.getElementById("kanbanIssueModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (modal) modal.classList.remove("show");
    if (overlay) overlay.classList.remove("show");
  }

  function submitKanbanIssueModal() {
    window.showToast("问题登记尚未接入服务端，当前操作不可用", "error");
    return;
  }

  /* ────────── 2. 建任务 Modal ────────── */
  function openKanbanCreateTaskModal() {
    var modal = document.getElementById("kanbanTaskModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (!modal || !overlay) return;

    var nameInput = document.getElementById("kbTaskName");
    var assigneeSel = document.getElementById("kbTaskAssignee");
    var dueInput = document.getElementById("kbTaskDue");

    if (nameInput) nameInput.value = "";
    if (dueInput) {
      var d = new Date();
      d.setDate(d.getDate() + 7);
      dueInput.value = d.toISOString().slice(0, 10);
    }

    if (assigneeSel) {
      assigneeSel.innerHTML = currentAgileMembers.map(function (m) {
        return '<option value="' + esc(m.name) + '">' + esc(m.name) + ' (' + esc(m.account) + ' · ' + esc(m.role) + ')</option>';
      }).join("");
    }

    overlay.classList.add("show");
    modal.classList.add("show");
    setTimeout(function () { if (nameInput) nameInput.focus(); }, 50);
  }

  function closeKanbanTaskModal() {
    var modal = document.getElementById("kanbanTaskModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (modal) modal.classList.remove("show");
    if (overlay) overlay.classList.remove("show");
  }

  function submitKanbanTaskModal() {
    window.showToast("任务创建尚未接入服务端，当前操作不可用", "error");
    return;
  }

  /* ────────── 3. 调整小组成员 Modal ────────── */
  function openKanbanTeamModal() {
    var modal = document.getElementById("kanbanTeamModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (!modal || !overlay) return;

    var teamId = window.PoWB && typeof window.PoWB.getSelectedTeamgroupId === "function"
      ? window.PoWB.getSelectedTeamgroupId() : 0;
    currentTeamgroupID = teamId;
    var titleEl = document.getElementById("kanbanTeamModalTitle");
    var activeGroup = document.querySelector('#agileChips .chip.active');
    var teamName = activeGroup ? activeGroup.textContent.trim() : "当前敏捷小组";
    if (titleEl) titleEl.textContent = "调整小组成员 · " + teamName;

    currentAgileMembers = [];
    candidatePool = [];
    teamDraft = [];
    teamSearchTerm = "";

    renderTeamModal();
    overlay.classList.add("show");
    modal.classList.add("show");
    loadTeamDetail(teamId);
  }

  function jsonFetch(path) {
    if (typeof window.appJson !== "function") return Promise.reject(new Error("页面请求能力未加载"));
    return window.appJson(path, { method: "GET" }).then(function (json) {
      if (!json || json.success !== true) throw new Error((json && json.message) || "小组数据加载失败");
      return json;
    });
  }

  function loadTeamDetail(teamId) {
    var revision = ++teamDataRevision;
    if (!teamId) {
      renderTeamModal();
      window.showToast("当前未选择敏捷小组，请先选择小组", "warning");
      return;
    }
    jsonFetch("/workbench/api/agile-teams/" + encodeURIComponent(teamId)).then(function (json) {
      if (revision !== teamDataRevision) return;
      var data = json.data || {};
      currentAgileMembers = (data.formal || []).map(function (m) {
        return { name: m.name || m.account, account: m.account, role: m.role || "研发", status: m.status || "formal", hours: Number(m.hours) || 0 };
      });
      teamDraft = currentAgileMembers.map(function (m) { return Object.assign({}, m); });
      var titleEl = document.getElementById("kanbanTeamModalTitle");
      if (titleEl) titleEl.textContent = "调整小组成员 · " + (data.name || "当前敏捷小组");
      renderTeamModal();
      searchCandidates("", teamId);
    }).catch(function (err) {
      if (revision !== teamDataRevision) return;
      renderTeamModal();
      window.showToast(err.message || "小组数据加载失败", "error");
    });
  }

  function searchCandidates(query, teamgroupID) {
    var revision = ++teamDataRevision;
    teamgroupID = Number(teamgroupID || currentTeamgroupID || 0);
    if (!query && !teamgroupID) {
      candidatePool = [];
      renderTeamModal();
      return;
    }
    jsonFetch("/workbench/api/agile-teams/candidates?q=" + encodeURIComponent(query) +
      "&teamgroupId=" + encodeURIComponent(teamgroupID)).then(function (json) {
      if (revision !== teamDataRevision) return;
      candidatePool = (json.data || []).map(function (c) {
        return { name: c.name || c.account, account: c.account, role: "研发" };
      });
      renderTeamModal();
      var input = document.getElementById("kbTeamSearch");
      if (input) { input.focus(); input.setSelectionRange(input.value.length, input.value.length); }
    }).catch(function (err) {
      if (revision === teamDataRevision) window.showToast(err.message || "候选人搜索失败", "error");
    });
  }

  function closeKanbanTeamModal() {
    var modal = document.getElementById("kanbanTeamModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (modal) modal.classList.remove("show");
    if (overlay) overlay.classList.remove("show");
  }

  function renderTeamModal() {
    var body = document.getElementById("kbTeamModalBody");
    if (!body) return;

    var existingAccounts = {};
    teamDraft.forEach(function (m) { existingAccounts[m.account] = true; });

    var matchedCandidates = candidatePool.filter(function (c) {
      if (existingAccounts[c.account]) return false;
      // 候选人已经由服务端按姓名、账号或拼音过滤；这里不能再按可见姓名二次过滤。
      return true;
    });

    var candidateEmpty = teamSearchTerm ? "无匹配候选人" : "请输入姓名或账号搜索候选人";
    var candidatesHtml = matchedCandidates.length ? matchedCandidates.map(function (c) {
      return (
        '<button type="button" class="kb-team-candidate" data-account="' + esc(c.account) + '" title="点击加入团队">' +
        '  <span>' + esc(c.name) + '（' + esc(c.account) + '）</span>' +
        '</button>'
      );
    }).join("") : '<div class="kb-team-candidate-empty">' + esc(candidateEmpty) + "</div>";

    var rowsHtml = teamDraft.map(function (m, idx) {
      var isNew = !!m.isNew;
      var roleOptions = agileTeamRoles.map(function (r) {
        return '<option value="' + esc(r) + '"' + (m.role === r ? " selected" : "") + '>' + esc(r) + '</option>';
      }).join("");

      return (
        '<tr class="' + (isNew ? "kb-team-row-new" : "") + '">' +
        '  <td>' +
        '    <div class="kb-team-user">' +
        '      <div class="kb-team-user-avatar">' + esc(m.name.slice(0, 1)) + '</div>' +
        '      <div class="kb-team-user-meta">' +
        '        <div class="kb-team-user-name">' +
        '          ' + esc(m.name) + ' <span class="kb-team-status-tag">正式</span>' +
        (isNew ? ' <em class="kb-team-row-new-badge">+ 新增</em>' : "") +
        '        </div>' +
        '        <div class="kb-team-user-account">' + esc(m.account) + '</div>' +
        '      </div>' +
        '    </div>' +
        '  </td>' +
        '  <td>' +
        '    <select class="kb-select" style="height:28px;width:100%" data-idx="' + idx + '" data-field="role">' +
        roleOptions +
        '    </select>' +
        '  </td>' +
        '  <td>' +
        '    <button type="button" class="kb-team-remove" data-remove-idx="' + idx + '">移除</button>' +
        '  </td>' +
        '</tr>'
      );
    }).join("");

    body.innerHTML =
      '<div class="kb-team-toolbar">' +
      '  <button type="button" class="action-btn kb-team-copy" id="kbTeamCopyBtn">' +
      '    <i class="fas fa-copy"></i>复制项目团队' +
      '  </button>' +
      '  <span class="kb-team-hint">当前身份：PO · 可提交调整；敏捷小组角色仅 PMO 可调整</span>' +
      '</div>' +
      '<div class="kb-team-confirm-tip">成员调整需组织级敏捷教练确认后正式生效</div>' +
      '<div class="kb-team-add-panel">' +
      '  <div class="kb-form-group" style="margin-bottom:6px">' +
      '    <input type="text" class="kb-input" id="kbTeamSearch" placeholder="搜索姓名 / 账号，点击即可加入团队" value="' + esc(teamSearchTerm) + '">' +
      '  </div>' +
      '  <div class="kb-team-candidate-list">' + candidatesHtml + '</div>' +
      '</div>' +
      '<table class="kb-team-table">' +
      '  <thead>' +
      '    <tr>' +
      '      <th style="width:52%">用户</th>' +
      '      <th style="width:30%">敏捷小组角色</th>' +
      '      <th style="width:18%">操作</th>' +
      '    </tr>' +
      '  </thead>' +
      '  <tbody>' + rowsHtml + '</tbody>' +
      '</table>' +
      '<div class="kb-form-group" style="margin-top:14px">' +
      '  <label style="font-size:12px;font-weight:600;color:var(--t1);margin-bottom:4px;display:block">调整说明</label>' +
      '  <textarea class="kb-textarea" id="kbTeamReason" rows="2" placeholder="说明本次成员调整原因（可选）"></textarea>' +
      '</div>';

    var copyBtn = document.getElementById("kbTeamCopyBtn");
    if (copyBtn) {
      copyBtn.addEventListener("click", function () {
        candidatePool.slice(0, 3).forEach(function (cand) {
          if (!teamDraft.some(function (m) { return m.account === cand.account; })) {
            teamDraft.push({ name: cand.name, account: cand.account, role: cand.role, status: "formal", isNew: true });
          }
        });
        window.showToast("已从项目团队同步导入成员", "success");
        renderTeamModal();
      });
    }

    var searchInput = document.getElementById("kbTeamSearch");
    if (searchInput) {
      var composing = false;
      searchInput.addEventListener("compositionstart", function () { composing = true; });
      searchInput.addEventListener("compositionend", function () {
        composing = false;
        teamSearchTerm = searchInput.value.trim();
        searchCandidates(teamSearchTerm);
      });
      searchInput.addEventListener("input", function () {
        if (composing) return;
        teamSearchTerm = searchInput.value.trim();
        searchCandidates(teamSearchTerm, currentTeamgroupID);
      });
    }

    body.querySelectorAll(".kb-team-candidate").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var acc = btn.getAttribute("data-account");
        var cand = candidatePool.find(function (c) { return c.account === acc; });
        if (cand) {
          teamDraft.push({
            name: cand.name,
            account: cand.account,
            role: cand.role,
            status: "formal",
            isNew: true
          });
          renderTeamModal();
        }
      });
    });

    body.querySelectorAll("select[data-field='role']").forEach(function (sel) {
      sel.addEventListener("change", function () {
        var idx = parseInt(sel.getAttribute("data-idx"), 10);
        if (teamDraft[idx]) teamDraft[idx].role = sel.value;
      });
    });

    body.querySelectorAll("button[data-remove-idx]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var idx = parseInt(btn.getAttribute("data-remove-idx"), 10);
        teamDraft.splice(idx, 1);
        renderTeamModal();
      });
    });
  }

  function submitKanbanTeamModal() {
    var teamId = Number(currentTeamgroupID || (window.PoWB && typeof window.PoWB.getSelectedTeamgroupId === "function" ? window.PoWB.getSelectedTeamgroupId() : 0));
    if (!teamId) {
      window.showToast("当前未选择敏捷小组，无法提交调整", "warning");
      return;
    }
    var origMap = {};
    currentAgileMembers.forEach(function (m) { origMap[m.account] = m; });
    var draftMap = {};
    var items = [];
    teamDraft.forEach(function (m) {
      draftMap[m.account] = m;
      if (!origMap[m.account]) {
        items.push({
          account: m.account,
          actionType: "add",
          role: m.role || "研发",
          availableHours: Number(m.hours) || 0
        });
      } else if (m.role !== origMap[m.account].role) {
        items.push({
          account: m.account,
          actionType: "roleChange",
          role: m.role || "研发",
          availableHours: Number(m.hours) || 0
        });
      }
    });
    currentAgileMembers.forEach(function (m) {
      if (!draftMap[m.account]) {
        items.push({
          account: m.account,
          actionType: "remove",
          role: m.role || "研发",
          availableHours: Number(m.hours) || 0
        });
      }
    });

    if (!items.length) {
      window.showToast("尚未产生变更", "info");
      return;
    }

    var reasonInput = document.getElementById("kbTeamAdjustReason");
    var reason = reasonInput ? reasonInput.value.trim() : "";

    if (typeof window.appJson !== "function") {
      window.showToast("页面请求组件未就绪，请刷新重试", "error");
      return;
    }

    window.appJson("/workbench/api/agile-teams/" + encodeURIComponent(teamId) + "/adjustments", {
      method: "POST",
      body: { reason: reason, items: items }
    }).then(function () {
      closeKanbanTeamModal();
      window.showToast("已提交，待组织级敏捷教练确认", "success");
      loadTeamDetail(teamId);
    }).catch(function (err) {
      window.showToast((err && err.message) || "提交调整失败", "error");
    });
  }

  window.openKanbanCreateModal = function (mode) {
    if (mode === "task") openKanbanCreateTaskModal();
    else openKanbanCreateIssueModal();
  };
  window.openKanbanTeamModal = openKanbanTeamModal;

  window.WorkboardModal = {
    openIssueModal: openKanbanCreateIssueModal,
    closeIssueModal: closeKanbanIssueModal,
    submitIssueModal: submitKanbanIssueModal,
    openTaskModal: openKanbanCreateTaskModal,
    closeTaskModal: closeKanbanTaskModal,
    submitTaskModal: submitKanbanTaskModal,
    openTeamModal: openKanbanTeamModal,
    closeTeamModal: closeKanbanTeamModal,
    submitTeamModal: submitKanbanTeamModal
  };

  // 全局键盘 Escape 与遮罩点击关闭交互契约支持
  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape") {
      closeKanbanIssueModal();
      closeKanbanTaskModal();
      closeKanbanTeamModal();
    }
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      var kbOverlay = document.getElementById("kanbanModalOverlay");
      if (kbOverlay) {
        kbOverlay.addEventListener("click", function () {
          closeKanbanIssueModal();
          closeKanbanTaskModal();
          closeKanbanTeamModal();
        });
      }
    });
  } else {
    var kbOverlay = document.getElementById("kanbanModalOverlay");
    if (kbOverlay) {
      kbOverlay.addEventListener("click", function () {
        closeKanbanIssueModal();
        closeKanbanTaskModal();
        closeKanbanTeamModal();
      });
    }
  }
})();
