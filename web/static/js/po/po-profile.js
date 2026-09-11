// =============================================================================
// 文件: web/static/js/po/po-profile.js
// 模块: PO 工作台 / 个人资料（/profile 与 shared profile modal）
// 职责: 对接 /api/profile 与 /profile/data；
//       支持查看账号信息、编辑邮箱/性别/工作台视图/默认敏捷小组、修改密码；
//       支持页面模式渲染与 openProfileModal 弹窗模式。
// =============================================================================

(function () {
  'use strict';

  var profileCache = null;
  var profileLoading = false;
  var ORG_ONLY_ROLES = { lead: 1, pmo: 1 };

  function esc(s) {
    return String(s == null ? '' : s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function readMeta(name) {
    var el = document.querySelector('meta[name="' + name + '"]');
    return el ? String(el.getAttribute('content') || '').trim() : '';
  }

  function showToast(msg, type) {
    if (typeof window.showToast === 'function') {
      window.showToast(msg, type || 'info');
      return;
    }
    try {
      if (type === 'danger' || type === 'error') {
        console.error('[toast]', msg);
      } else {
        console.info('[toast]', msg);
      }
    } catch (e) { /* ignore */ }
  }

  function setBusy(btn, busy, defaultText, busyText) {
    if (!btn) return;
    btn.disabled = !!busy;
    if (busy) {
      btn.dataset.prevHtml = btn.innerHTML;
      btn.innerHTML = '<i class="fas fa-circle-notch fa-spin"></i> ' + (busyText || '处理中…');
    } else if (btn.dataset.prevHtml) {
      btn.innerHTML = btn.dataset.prevHtml;
    }
  }

  function detectWorkRole() {
    if (window.RoleSwitcher && typeof window.RoleSwitcher.detectRole === 'function') {
      return String(window.RoleSwitcher.detectRole() || '').toLowerCase();
    }
    if (window.currentUser && window.currentUser.role) {
      return String(window.currentUser.role).toLowerCase();
    }
    return String(readMeta('wb-role') || '').toLowerCase();
  }

  async function apiFetch(path, options) {
    options = options || {};
    var headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
      'X-Requested-With': 'XMLHttpRequest',
      'X-CSRF-Token': readMeta('csrf-token')
    };
    if (options.headers) {
      Object.keys(options.headers).forEach(function (k) {
        headers[k] = options.headers[k];
      });
    }
    var fetchOpts = {
      method: options.method || 'GET',
      credentials: 'same-origin',
      headers: headers
    };
    if (options.body && typeof options.body === 'object' && !(options.body instanceof FormData)) {
      fetchOpts.body = JSON.stringify(options.body);
    } else if (options.body != null) {
      fetchOpts.body = options.body;
    }

    var fetchFn = window.appFetch || window.fetch;
    var res = await fetchFn(path, fetchOpts);
    if (res.status === 401) {
      var target = encodeURIComponent(location.pathname + location.search);
      location.href = '/login?redirect=' + target;
      throw new Error('登录已过期，请重新登录');
    }
    var json = await res.json().catch(function () {
      throw new Error('服务端响应格式异常');
    });
    return json;
  }

  function genderOptions(selected) {
    return [
      { v: '', l: '未设置' },
      { v: 'm', l: '男' },
      { v: 'f', l: '女' }
    ].map(function (o) {
      return '<option value="' + o.v + '"' + (selected === o.v ? ' selected' : '') + '>' + o.l + '</option>';
    }).join('');
  }

  function fieldErrorsHtml(errors) {
    if (!Array.isArray(errors) || !errors.length) return '';
    return '<div class="audit-note" id="profileAuditNote">' +
      '<i class="fas fa-circle-exclamation"></i><div>' +
      errors.map(function (e) { return esc((e && e.message) || '校验失败'); }).join('<br>') +
      '</div></div>';
  }

  function preferredSet(p) {
    var raw = Array.isArray(p.preferredRoles) ? p.preferredRoles : [];
    var set = {};
    raw.forEach(function (k) {
      var key = String(k || '').toLowerCase();
      if (key && !ORG_ONLY_ROLES[key]) set[key] = true;
    });
    return set;
  }

  function roleCheckboxesHtml(p) {
    var selectable = Array.isArray(p.allowedRoles) ? p.allowedRoles : [];
    var checked = preferredSet(p);
    if (!selectable.length) {
      return '<div class="profile-hint">暂无可自选工作台角色（团队管理 / PMO 须组织统一配置）</div>';
    }
    var boxes = selectable.map(function (r) {
      var key = String((r && r.key) || '').toLowerCase();
      var label = (r && r.label) || key;
      var isOn = !!checked[key];
      return '<label><input type="checkbox" name="profilePreferredRole" value="' + esc(key) + '"' +
        (isOn ? ' checked' : '') + '> ' + esc(label) + '</label>';
    }).join('');
    return '<div class="profile-check-row" id="profileRoleCheckRow">' + boxes + '</div>' +
      '<div class="profile-hint">可多选；不勾选=顶栏展示全部已授权自选角色。团队管理 / PMO 须组织统一配置，不可在此勾选</div>';
  }

  function roleFieldHtml(p) {
    var currentRole = detectWorkRole();
    var orgLocked = !!ORG_ONLY_ROLES[currentRole];
    var orgNote = '';
    if (orgLocked) {
      var label = currentRole === 'pmo' ? 'PMO' : '团队管理';
      orgNote = '<div class="profile-hint" style="margin-bottom:8px;color:var(--color-primary)">当前视图为「' + esc(label) +
        '」（组织授权），不在自选范围内；下方仍可勾选其他已授权自选角色。</div>';
    }
    return '<div class="form-group col-span-full"><label class="form-label">工作台角色（多选自选视图）</label>' +
      orgNote + roleCheckboxesHtml(p) + '</div>';
  }

  function agileGroupsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    if (!groups.length) {
      return '<span class="profile-empty-text">暂无从禅道团队成员关系识别到已加入的敏捷小组</span>';
    }
    return groups.map(function (g) {
      return '<span class="profile-scope-chip">' + esc(g.name || ('小组#' + g.id)) + '</span>';
    }).join('');
  }

  function mainTeamOptionsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    var selected = Number(p.mainTeamId || 0);
    var opts = ['<option value="0"' + (selected === 0 ? ' selected' : '') + '>未设置</option>'];
    groups.forEach(function (g) {
      var id = Number(g.id || 0);
      opts.push('<option value="' + id + '"' + (selected === id ? ' selected' : '') + '>' +
        esc(g.name || ('小组#' + id)) + '</option>');
    });
    return opts.join('');
  }

  function togglePasswordVisibility(btn, inputId) {
    var inp = document.getElementById(inputId);
    if (!inp) return;
    var isPwd = inp.type === 'password';
    inp.type = isPwd ? 'text' : 'password';
    var icon = btn.querySelector('i');
    if (icon) {
      icon.className = isPwd ? 'fas fa-eye-slash' : 'fas fa-eye';
    }
  }

  function renderBody(host, p, errHtml) {
    var initial = String(p.displayName || p.account || '?').charAt(0);
    var deptText = p.deptName || (p.deptId ? ('部门#' + p.deptId) : '未分配');
    host.innerHTML =
      (errHtml || '') +
      '<div class="profile-card">' +
        '<div class="profile-card-h"><i class="fas fa-id-card"></i>账号基本信息</div>' +
        '<div class="profile-avatar-row">' +
          '<div class="profile-avatar-lg">' + esc(initial) + '</div>' +
          '<div class="profile-avatar-meta">' +
            '<strong>' + esc(p.displayName || p.account || '用户') + '</strong> (' + esc(p.account || '') + ')<br>' +
            '头像取姓名首字 · 可维护邮箱、性别、工作台角色、默认小组与密码' +
          '</div>' +
        '</div>' +
        '<div class="profile-grid">' +
          '<div class="form-group"><label class="form-label">真实姓名</label>' +
            '<input class="form-input" id="profileRealname" value="' + esc(p.displayName || '') + '" readonly disabled>' +
            '<div class="profile-hint">来自禅道账号，不可在此修改</div></div>' +
          '<div class="form-group"><label class="form-label">用户名</label>' +
            '<input class="form-input" id="profileAccount" value="' + esc(p.account || '') + '" readonly disabled>' +
            '<div class="profile-hint">登录账号不可修改</div></div>' +
          '<div class="form-group"><label class="form-label" for="profileEmail">邮箱</label>' +
            '<input class="form-input" id="profileEmail" type="email" value="' + esc(p.email || '') + '" placeholder="name@company.com">' +
            '<div class="profile-hint">可选；填写须为合法邮箱</div></div>' +
          '<div class="form-group"><label class="form-label">手机</label>' +
            '<input class="form-input" id="profileMobile" value="' + esc(p.mobile || '') + '" readonly disabled>' +
            '<div class="profile-hint">来自禅道账号，不可在此修改</div></div>' +
          '<div class="form-group"><label class="form-label" for="profileGender">性别</label>' +
            '<select class="form-input" id="profileGender">' + genderOptions(p.gender || '') + '</select>' +
            '<div class="profile-hint">当前账号性别偏好</div></div>' +
          '<div class="form-group"><label class="form-label">部门</label>' +
            '<input class="form-input" id="profileDept" value="' + esc(deptText) + '" readonly disabled>' +
            '<div class="profile-hint">来自禅道组织架构，不可在此修改</div></div>' +
          roleFieldHtml(p) +
        '</div>' +
      '</div>' +
      '<div class="profile-card">' +
        '<div class="profile-card-h"><i class="fas fa-user-gear"></i>敏捷小组</div>' +
        '<div class="profile-grid">' +
          '<div class="form-group col-span-full"><label class="form-label">参与的敏捷小组</label>' +
            '<div class="profile-scope-row" id="profileAgileCheckRow">' + agileGroupsHtml(p) + '</div>' +
            '<div class="profile-hint">展示当前账号在禅道团队成员关系中已加入的小组</div>' +
          '</div>' +
          '<div class="form-group"><label class="form-label" for="profileMainTeam">默认敏捷小组</label>' +
            '<select class="form-input" id="profileMainTeam">' + mainTeamOptionsHtml(p) + '</select>' +
            '<div class="profile-hint">须选自本人已加入的敏捷小组；对应禅道 zt_user.mainTeam</div></div>' +
        '</div>' +
      '</div>' +
      '<div class="profile-card">' +
        '<div class="profile-card-h"><i class="fas fa-key"></i>修改密码</div>' +
        '<div class="profile-grid">' +
          '<div class="form-group"><label class="form-label" for="profileOldPwd">当前密码</label>' +
            '<div class="pwd-input-wrap">' +
              '<input class="form-input" id="profileOldPwd" type="password" autocomplete="current-password" placeholder="请输入当前密码">' +
              '<button type="button" class="pwd-toggle-btn" data-target="profileOldPwd" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
            '</div></div>' +
          '<div class="form-group"><label class="form-label" for="profileNewPwd">新密码</label>' +
            '<div class="pwd-input-wrap">' +
              '<input class="form-input" id="profileNewPwd" type="password" autocomplete="new-password" placeholder="8-64位，含字母和数字">' +
              '<button type="button" class="pwd-toggle-btn" data-target="profileNewPwd" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
            '</div>' +
            '<div class="profile-hint">8-64 位字符，须同时包含字母和数字</div></div>' +
          '<div class="form-group"><label class="form-label" for="profileNewPwd2">确认新密码</label>' +
            '<div class="pwd-input-wrap">' +
              '<input class="form-input" id="profileNewPwd2" type="password" autocomplete="new-password" placeholder="再次输入新密码">' +
              '<button type="button" class="pwd-toggle-btn" data-target="profileNewPwd2" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
            '</div></div>' +
          '<div class="form-group" style="align-self:end">' +
            '<button type="button" class="action-btn primary" id="profilePwdSaveBtn"><i class="fas fa-key"></i> 更新密码</button>' +
          '</div>' +
        '</div>' +
      '</div>' +
      '<div class="profile-actions">' +
        '<button type="button" class="action-btn primary" id="profileSaveBtn"><i class="fas fa-check"></i> 保存资料</button>' +
      '</div>';

    // 绑定密码可见性切换
    host.querySelectorAll('.pwd-toggle-btn').forEach(function (b) {
      b.addEventListener('click', function () {
        togglePasswordVisibility(b, b.getAttribute('data-target'));
      });
    });

    var saveBtn = document.getElementById('profileSaveBtn');
    if (saveBtn) saveBtn.addEventListener('click', function () { saveProfileForm(); });
    var pwdBtn = document.getElementById('profilePwdSaveBtn');
    if (pwdBtn) pwdBtn.addEventListener('click', changePassword);
  }

  function readPreferredRolesFromDom() {
    var nodes = document.querySelectorAll('input[name="profilePreferredRole"]:checked');
    var out = [];
    nodes.forEach(function (el) {
      var key = String(el.value || '').toLowerCase();
      if (key && !ORG_ONLY_ROLES[key]) out.push(key);
    });
    return out;
  }

  function syncPreferredToSwitcher(roles) {
    var list = Array.isArray(roles) ? roles : [];
    if (window.RoleSwitcher && typeof window.RoleSwitcher.setPreferredRoles === 'function') {
      window.RoleSwitcher.setPreferredRoles(list);
      return;
    }
    try {
      localStorage.setItem('wb_preferred_roles_v1', JSON.stringify({ preferredRoles: list }));
    } catch (e) { /* ignore */ }
  }

  async function loadProfile() {
    var json;
    try {
      json = await apiFetch('/api/profile');
    } catch (e1) {
      json = await apiFetch('/profile/data');
    }
    if (!json || !json.success || !json.data) {
      throw new Error((json && json.message) || '加载个人资料失败');
    }
    profileCache = json.data;
    syncPreferredToSwitcher(profileCache.preferredRoles || []);
    return profileCache;
  }

  window.loadUserProfileDraft = function loadUserProfileDraftFromCache() {
    var p = profileCache || {};
    var account = p.account || (window.currentUser && window.currentUser.account) || '';
    var realname = p.displayName || (window.currentUser && window.currentUser.realname) || account;
    var preferred = Array.isArray(p.preferredRoles) ? p.preferredRoles.slice() : [];
    return {
      realname: realname,
      account: account,
      email: p.email || '',
      gender: p.gender || '',
      mobile: p.mobile || '',
      dept: p.deptName || '',
      workRoles: preferred,
      preferredRoles: preferred,
      workRole: detectWorkRole(),
      agileGroups: Array.isArray(p.agileGroups) ? p.agileGroups : [],
      mainTeamId: Number(p.mainTeamId || 0),
      primaryAgile: String(p.mainTeamId || '')
    };
  };

  window.renderProfilePage = async function renderProfilePageLive() {
    var host = document.getElementById('profilePageBody') || document.getElementById('profileModalBody');
    var loadingEl = document.getElementById('poProfileLoading');
    var errorEl = document.getElementById('poProfileError');
    var errorMsg = document.getElementById('poProfileErrorMsg');
    if (!host) return;

    if (profileLoading) return;
    profileLoading = true;

    if (loadingEl) loadingEl.hidden = false;
    if (errorEl) errorEl.hidden = true;
    host.hidden = true;

    try {
      var data = await loadProfile();
      renderBody(host, data, '');
      if (loadingEl) loadingEl.hidden = true;
      host.hidden = false;
    } catch (e) {
      if (loadingEl) loadingEl.hidden = true;
      if (errorEl) {
        errorEl.hidden = false;
        if (errorMsg) errorMsg.textContent = (e && e.message) || '加载个人资料失败';
      }
      showToast((e && e.message) || '加载个人资料失败', 'danger');
    } finally {
      profileLoading = false;
    }
  };

  window.saveProfileForm = async function saveProfileFormLive() {
    var preferredRoles = readPreferredRolesFromDom();
    var mainTeamEl = document.getElementById('profileMainTeam');
    var mainTeamId = mainTeamEl ? Number(mainTeamEl.value || 0) : Number((profileCache && profileCache.mainTeamId) || 0);
    if (!Number.isFinite(mainTeamId) || mainTeamId < 0) mainTeamId = 0;

    var payload = {
      email: (document.getElementById('profileEmail') || {}).value || '',
      gender: (document.getElementById('profileGender') || {}).value || '',
      preferredRoles: preferredRoles,
      mainTeamId: mainTeamId
    };

    var btn = document.getElementById('profileSaveBtn');
    setBusy(btn, true, '保存资料', '保存中…');

    try {
      var json;
      try {
        json = await apiFetch('/api/profile', { method: 'PUT', body: payload });
      } catch (err) {
        json = await apiFetch('/profile', { method: 'PUT', body: payload });
      }

      if (json && Array.isArray(json.errors) && json.errors.length) {
        var host = document.getElementById('profilePageBody') || document.getElementById('profileModalBody');
        if (host && profileCache) {
          renderBody(host, Object.assign({}, profileCache, {
            email: payload.email,
            gender: payload.gender,
            preferredRoles: payload.preferredRoles,
            mainTeamId: payload.mainTeamId
          }), fieldErrorsHtml(json.errors));
        }
        showToast(json.errors[0].message || '请检查表单输入', 'danger');
        return;
      }
      if (!json || !json.success) {
        showToast((json && json.message) || '保存失败', 'danger');
        return;
      }

      var savedRoles = Array.isArray(json.preferredRoles) ? json.preferredRoles : payload.preferredRoles;
      var savedTeam = json.mainTeamId != null ? Number(json.mainTeamId) : payload.mainTeamId;
      syncPreferredToSwitcher(savedRoles);
      if (profileCache) {
        profileCache.preferredRoles = savedRoles;
        profileCache.mainTeamId = savedTeam;
        profileCache.email = payload.email;
        profileCache.gender = payload.gender;
      }
      showToast(json.message || '个人资料已保存', 'success');

      // 清除错误信息并重新刷新视图以确认持久化
      var note = document.getElementById('profileAuditNote');
      if (note) note.remove();

      if (typeof closeProfileModal === 'function') {
        var modal = document.getElementById('profileModal');
        if (modal && modal.classList.contains('show')) closeProfileModal();
      }
    } catch (e) {
      showToast((e && e.message) || '保存失败，请稍后重试', 'danger');
    } finally {
      setBusy(btn, false);
    }
  };

  async function changePassword() {
    var oldPwd = (document.getElementById('profileOldPwd') || {}).value || '';
    var newPwd = (document.getElementById('profileNewPwd') || {}).value || '';
    var confirm = (document.getElementById('profileNewPwd2') || {}).value || '';

    if (!oldPwd) {
      showToast('请输入当前密码', 'danger');
      return;
    }
    if (!newPwd) {
      showToast('请输入新密码', 'danger');
      return;
    }
    if (newPwd.length < 8 || newPwd.length > 64) {
      showToast('新密码长度需为 8-64 位', 'danger');
      return;
    }
    if (!/[A-Za-z]/.test(newPwd) || !/[0-9]/.test(newPwd)) {
      showToast('新密码须同时包含字母和数字', 'danger');
      return;
    }
    if (newPwd !== confirm) {
      showToast('两次输入的新密码不一致', 'danger');
      return;
    }

    var btn = document.getElementById('profilePwdSaveBtn');
    setBusy(btn, true, '更新密码', '更新中…');

    try {
      var payload = {
        oldPassword: oldPwd,
        newPassword: newPwd,
        confirmPassword: confirm
      };
      var json;
      try {
        json = await apiFetch('/api/profile/password', { method: 'PUT', body: payload });
      } catch (err) {
        json = await apiFetch('/profile/password', { method: 'PUT', body: payload });
      }

      if (json && Array.isArray(json.errors) && json.errors.length) {
        showToast(json.errors[0].message || '请检查密码格式', 'danger');
        return;
      }
      if (!json || !json.success) {
        showToast((json && json.message) || '密码更新失败', 'danger');
        return;
      }

      ['profileOldPwd', 'profileNewPwd', 'profileNewPwd2'].forEach(function (id) {
        var el = document.getElementById(id);
        if (el) el.value = '';
      });
      showToast(json.message || '密码已更新', 'success');
    } catch (e) {
      showToast((e && e.message) || '改密失败，请稍后重试', 'danger');
    } finally {
      setBusy(btn, false);
    }
  }

  // 支持弹窗模式 (openProfileModal / closeProfileModal)
  function onProfileOverlayClick(e) {
    if (e.target !== e.currentTarget) return;
    closeProfileModal();
  }

  window.ensureProfileModalDom = function ensureProfileModalDom() {
    var overlay = document.getElementById('profileModalOverlay');
    var modal = document.getElementById('profileModal');
    if (overlay && modal && document.getElementById('profileModalBody')) {
      return { overlay: overlay, modal: modal };
    }
    if (overlay) overlay.remove();
    if (modal) modal.remove();

    overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.id = 'profileModalOverlay';
    overlay.addEventListener('click', onProfileOverlayClick);

    modal = document.createElement('div');
    modal.className = 'modal profile-modal';
    modal.id = 'profileModal';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'profileModalTitle');
    modal.innerHTML =
      '<div class="modal-hdr">' +
        '<h3 id="profileModalTitle"><i class="fas fa-id-card"></i> 个人资料</h3>' +
        '<button type="button" class="modal-close" aria-label="关闭" onclick="closeProfileModal()"><i class="fas fa-times"></i></button>' +
      '</div>' +
      '<div class="modal-body" id="profileModalBody"></div>';

    document.body.appendChild(overlay);
    document.body.appendChild(modal);
    return { overlay: overlay, modal: modal };
  };

  window.openProfileModal = function openProfileModal(e) {
    if (e && e.preventDefault) e.preventDefault();
    var nodes = window.ensureProfileModalDom();
    window.renderProfilePage();
    requestAnimationFrame(function () {
      nodes.overlay.classList.add('show');
      nodes.modal.classList.add('show');
    });
  };

  window.closeProfileModal = function closeProfileModal() {
    var overlay = document.getElementById('profileModalOverlay');
    var modal = document.getElementById('profileModal');
    if (overlay) overlay.classList.remove('show');
    if (modal) modal.classList.remove('show');
  };

  // 快捷键支持
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') {
      window.closeProfileModal();
    } else if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      var btn = document.getElementById('profileSaveBtn');
      if (btn && !btn.disabled) {
        e.preventDefault();
        saveProfileFormLive();
      }
    }
  });

  // 重试按钮绑定
  function bindRetry() {
    var retryBtn = document.getElementById('poProfileRetryBtn');
    if (retryBtn && retryBtn.dataset.bound !== '1') {
      retryBtn.dataset.bound = '1';
      retryBtn.addEventListener('click', function () {
        window.renderProfilePage();
      });
    }
  }

  // 页面自启动
  function bootstrap() {
    bindRetry();
    if (document.getElementById('profilePageBody')) {
      window.renderProfilePage();
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bootstrap);
  } else {
    bootstrap();
  }
})();
