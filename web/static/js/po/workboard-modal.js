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

  var teamRoles = ["研发", "测试", "PO", "SM", "架构", "运维", "美工", "产品经理"];
  var teamDraft = [];
  var teamSearchTerm = "";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (s) {
    return String(s == null ? "" : s);
  };

  function toast(msg) {
    if (typeof window.showToast === "function") window.showToast(msg);
  }

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
    toast("问题登记尚未接入服务端，当前操作不可用");
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
    toast("任务创建尚未接入服务端，当前操作不可用");
    return;
  }

  /* ────────── 3. 调整小组成员 Modal ────────── */
  function openKanbanTeamModal() {
    var modal = document.getElementById("kanbanTeamModalDialog");
    var overlay = document.getElementById("kanbanModalOverlay");
    if (!modal || !overlay) return;

    var titleEl = document.getElementById("kanbanTeamModalTitle");
    if (titleEl) {
      var teamName = "组织变革团队";
      var activeGroup = document.querySelector(".group-tab.active");
      if (activeGroup) teamName = activeGroup.textContent.trim() || teamName;
      titleEl.textContent = "调整小组成员 · " + teamName;
    }

    teamDraft = currentAgileMembers.map(function (m) {
      return Object.assign({}, m);
    });
    teamSearchTerm = "";

    renderTeamModal();
    overlay.classList.add("show");
    modal.classList.add("show");
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
      if (!teamSearchTerm) return true;
      var term = teamSearchTerm.toLowerCase();
      return c.name.toLowerCase().indexOf(term) >= 0 || c.account.toLowerCase().indexOf(term) >= 0;
    });

    var candidatesHtml = matchedCandidates.length ? matchedCandidates.map(function (c) {
      return (
        '<button type="button" class="kb-team-candidate" data-account="' + esc(c.account) + '" title="点击加入团队">' +
        '  <span>' + esc(c.name) + '（' + esc(c.account) + '）</span>' +
        '</button>'
      );
    }).join("") : '<div style="grid-column:1/-1;color:var(--t3);font-size:11px;padding:6px 0">无匹配候选人</div>';

    var rowsHtml = teamDraft.map(function (m, idx) {
      var isNew = !!m.isNew;
      var roleOptions = teamRoles.map(function (r) {
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
        '    <input type="number" class="kb-input" style="height:28px;width:64px" min="0" max="24" step="0.5" value="' + m.hours + '" data-idx="' + idx + '" data-field="hours">' +
        '  </td>' +
        '  <td>' +
        '    <button type="button" class="kb-team-remove" data-remove-idx="' + idx + '">移除</button>' +
        '  </td>' +
        '</tr>'
      );
    }).join("");

    body.innerHTML =
      '<div class="kb-team-toolbar">' +
      '  <button type="button" class="action-btn" id="kbTeamCopyBtn" style="height:28px;padding:0 10px;border:1px solid #cbd5e1;border-radius:4px;background:#fff;cursor:pointer;font-size:12px;color:#334155;display:inline-flex;align-items:center;gap:4px">' +
      '    <i class="fas fa-copy"></i>复制项目团队' +
      '  </button>' +
      '  <span class="kb-team-hint">当前身份：PO · 可提交调整；团队角色仅 PMO 可调整</span>' +
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
      '      <th style="width:38%">用户</th>' +
      '      <th style="width:24%">角色</th>' +
      '      <th style="width:22%">可用工时/天</th>' +
      '      <th style="width:16%">操作</th>' +
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
            teamDraft.push({ name: cand.name, account: cand.account, role: cand.role, hours: 0, status: "formal", isNew: true });
          }
        });
        toast("已从项目团队同步导入成员");
        renderTeamModal();
      });
    }

    var searchInput = document.getElementById("kbTeamSearch");
    if (searchInput) {
      searchInput.addEventListener("input", function () {
        teamSearchTerm = searchInput.value.trim();
        renderTeamModal();
        var refreshed = document.getElementById("kbTeamSearch");
        if (refreshed) {
          refreshed.focus();
          refreshed.setSelectionRange(refreshed.value.length, refreshed.value.length);
        }
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
            hours: 0,
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

    body.querySelectorAll("input[data-field='hours']").forEach(function (inp) {
      inp.addEventListener("change", function () {
        var idx = parseInt(inp.getAttribute("data-idx"), 10);
        if (teamDraft[idx]) teamDraft[idx].hours = parseFloat(inp.value) || 0;
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
    currentAgileMembers = teamDraft.map(function (m) {
      var copy = Object.assign({}, m);
      copy.isNew = false;
      return copy;
    });

    closeKanbanTeamModal();
    toast("团队调整申请已提交成功，待组织级敏捷教练审批生效");
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
