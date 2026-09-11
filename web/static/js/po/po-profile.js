// =============================================================================
// 文件: web/static/js/po/po-profile.js
// 模块: PO 工作台 / 个人资料（/profile 与 shared profile modal）
// 职责: 对接 /api/profile 与 /profile/data；
//       支持查看账号信息、编辑邮箱/性别/工作台视图/默认敏捷小组、修改密码；
//       采用宽幅双栏大弹窗设计，无多余滚动，操作吸底常驻；
//       支持页面模式渲染与 openProfileModal 弹窗模式。
// =============================================================================

(function () {
  'use strict';

  if (window.__poProfileBound) {
    return;
  }
  window.__poProfileBound = true;

  var profileCache = null;
  var profileLoading = false;
  var ORG_ONLY_ROLES = { lead: 1, pmo: 1 };
  // 系统当前实际开放的视图白名单：严格以实际开放为准，当前仅开放 PO
  var OPEN_VIEWS_WHITELIST = { po: 1 };

  function esc(s) {
    return String(s == null ? '' : s)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function cleanAccountName(rawName, account) {
    var name = String(rawName || '').trim();
    var acc = String(account || '').trim();
    if (!name) return acc;
    if (acc) {
      name = name.replace(new RegExp('\\s*[(（]' + acc + '[)）]\\s*$', 'i'), '').trim();
    }
    return name || acc;
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
    // 严格按照用户指示：以实际开放的视图为准，当前只开放了 PO，所以只展示 PO
    selectable = selectable.filter(function (r) {
      var key = String((r && r.key) || '').toLowerCase();
      return !!OPEN_VIEWS_WHITELIST[key];
    });
    if (!selectable.length) {
      selectable = [{ key: 'po', label: '产品负责人 (PO)' }];
    }

    var checked = preferredSet(p);
    var boxes = selectable.map(function (r) {
      var key = String((r && r.key) || '').toLowerCase();
      var label = (r && r.label) || (key === 'po' ? '产品负责人 (PO)' : key);
      // 当前系统唯一定位：PO 默认保持勾选
      var isOn = checked[key] !== false;
      return '<label class="role-box-card' + (isOn ? ' is-checked' : '') + '">' +
        '<input type="checkbox" name="profilePreferredRole" value="' + esc(key) + '"' +
        (isOn ? ' checked' : '') + ' onchange="this.parentElement.classList.toggle(\'is-checked\', this.checked)"> ' +
        '<span>' + esc(label) + '</span>' +
        '</label>';
    }).join('');

    return '<div class="profile-check-row" id="profileRoleCheckRow">' + boxes + '</div>' +
      '<div class="profile-hint">当前系统已开放「产品负责人 (PO)」工作台。团队管理 / PMO 须组织统一配置。</div>';
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
    return orgNote + roleCheckboxesHtml(p);
  }

  function agileGroupsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    if (!groups.length) {
      return '<span class="profile-empty-text">暂无从禅道团队成员关系识别到已加入的敏捷小组</span>';
    }
    return groups.map(function (g) {
      return '<span class="profile-scope-chip"><i class="fas fa-users" style="font-size:10px;margin-right:4px;"></i>' + esc(g.name || ('小组#' + g.id)) + '</span>';
    }).join('');
  }

  function mainTeamOptionsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    var selected = Number(p.mainTeamId || 0);
    var opts = ['<option value="0"' + (selected === 0 ? ' selected' : '') + '>未设置默认小组</option>'];
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
    var rawName = String(p.displayName || p.account || '用户').trim();
    var account = String(p.account || '').trim();
    var cleanName = cleanAccountName(rawName, account);
    var initial = cleanName.charAt(0) || account.charAt(0) || '?';
    var deptText = p.deptName || (p.deptId ? ('部门#' + p.deptId) : '未分配');

    host.innerHTML =
      (errHtml || '') +
      '<div class="profile-content-grid">' +
        '<!-- 左栏：基本身份与账号信息 -->' +
        '<div class="profile-col profile-col-main">' +
          '<div class="profile-card">' +
            '<div class="profile-card-h"><i class="fas fa-user-shield"></i>账号身份与基本资料</div>' +
            '<div class="profile-avatar-row">' +
              '<div class="profile-avatar-lg">' + esc(initial) + '</div>' +
              '<div class="profile-avatar-meta">' +
                '<div class="profile-avatar-name-line">' +
                  '<strong>' + esc(cleanName) + '</strong>' +
                  '<span class="profile-account-chip">' + esc(account) + '</span>' +
                  '<span class="profile-dept-badge">' + esc(deptText) + '</span>' +
                '</div>' +
                '<div class="profile-avatar-sub">来自禅道同步账号 · 头像取姓名首字</div>' +
              '</div>' +
            '</div>' +
            '<div class="profile-grid">' +
              '<div class="form-group"><label class="form-label">真实姓名 <span class="readonly-badge">（只读）</span></label>' +
                '<input class="form-input" id="profileRealname" value="' + esc(cleanName) + '" readonly disabled>' +
                '<div class="profile-hint">来自禅道账号，不可在此修改</div></div>' +
              '<div class="form-group"><label class="form-label">用户名 / 工号 <span class="readonly-badge">（只读）</span></label>' +
                '<input class="form-input" id="profileAccount" value="' + esc(account) + '" readonly disabled>' +
                '<div class="profile-hint">系统登录主键</div></div>' +
              '<div class="form-group"><label class="form-label">手机 <span class="readonly-badge">（只读）</span></label>' +
                '<input class="form-input" id="profileMobile" value="' + esc(p.mobile || '') + '" readonly disabled>' +
                '<div class="profile-hint">来自禅道通讯录</div></div>' +
              '<div class="form-group"><label class="form-label">所属部门 <span class="readonly-badge">（只读）</span></label>' +
                '<input class="form-input" id="profileDept" value="' + esc(deptText) + '" readonly disabled>' +
                '<div class="profile-hint">来自银行组织架构</div></div>' +
              '<div class="form-group"><label class="form-label" for="profileEmail">个人邮箱 <span class="editable-badge">（可维护）</span></label>' +
                '<input class="form-input" id="profileEmail" type="email" value="' + esc(p.email || '') + '" placeholder="暂未设置邮箱（可在此填写绑定）">' +
                '<div class="profile-hint">用于接收工作台动态与任务提醒</div></div>' +
              '<div class="form-group"><label class="form-label" for="profileGender">性别偏好 <span class="editable-badge">（可维护）</span></label>' +
                '<select class="form-input" id="profileGender">' + genderOptions(p.gender || '') + '</select>' +
                '<div class="profile-hint">当前账号性别偏好</div></div>' +
            '</div>' +
          '</div>' +
        '</div>' +

        '<!-- 右栏：敏捷团队归属 + 自选角色视图 + 安全改密 -->' +
        '<div class="profile-col profile-col-side">' +
          '<div class="profile-card">' +
            '<div class="profile-card-h"><i class="fas fa-users-gear"></i>敏捷小组归属</div>' +
            '<div class="form-group">' +
              '<label class="form-label">参与的敏捷小组</label>' +
              '<div class="profile-scope-row" id="profileAgileCheckRow">' + agileGroupsHtml(p) + '</div>' +
            '</div>' +
            '<div class="form-group" style="margin-top:8px;">' +
              '<label class="form-label" for="profileMainTeam">默认敏捷小组</label>' +
              '<select class="form-input" id="profileMainTeam">' + mainTeamOptionsHtml(p) + '</select>' +
              '<div class="profile-hint">须选自本人已加入的小组；对应禅道 zt_user.mainTeam</div>' +
            '</div>' +
          '</div>' +

          '<div class="profile-card">' +
            '<div class="profile-card-h"><i class="fas fa-layer-group"></i>工作台角色（自选多视图）</div>' +
            roleFieldHtml(p) +
          '</div>' +

          '<div class="profile-card profile-pwd-card">' +
            '<div class="profile-card-h profile-pwd-hdr">' +
              '<span><i class="fas fa-key"></i>修改密码</span>' +
              '<button type="button" class="profile-card-toggle" data-toggle-target="profilePwdBlock" aria-expanded="false" aria-controls="profilePwdBlock">' +
                '<span>展开设置</span> <i class="fas fa-chevron-down profile-card-toggle-caret" aria-hidden="true"></i>' +
              '</button>' +
            '</div>' +
            '<div id="profilePwdBlock" class="profile-pwd-content" hidden>' +
              '<div class="form-group"><label class="form-label" for="profileOldPwd">当前密码</label>' +
                '<div class="pwd-input-wrap">' +
                  '<input class="form-input" id="profileOldPwd" type="password" autocomplete="current-password" placeholder="请输入当前密码">' +
                  '<button type="button" class="pwd-toggle-btn" data-target="profileOldPwd" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
                '</div></div>' +
              '<div class="pwd-row-2">' +
                '<div class="form-group"><label class="form-label" for="profileNewPwd">新密码</label>' +
                  '<div class="pwd-input-wrap">' +
                    '<input class="form-input" id="profileNewPwd" type="password" autocomplete="new-password" placeholder="8-64位含字母数字">' +
                    '<button type="button" class="pwd-toggle-btn" data-target="profileNewPwd" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
                  '</div></div>' +
                '<div class="form-group"><label class="form-label" for="profileNewPwd2">确认新密码</label>' +
                  '<div class="pwd-input-wrap">' +
                    '<input class="form-input" id="profileNewPwd2" type="password" autocomplete="new-password" placeholder="再次输入新密码">' +
                    '<button type="button" class="pwd-toggle-btn" data-target="profileNewPwd2" aria-label="切换密码可见性"><i class="fas fa-eye"></i></button>' +
                  '</div></div>' +
              '</div>' +
              '<div class="pwd-action-row">' +
                '<button type="button" class="btn-pwd-save" id="profilePwdSaveBtn"><i class="fas fa-shield"></i> 更新密码</button>' +
              '</div>' +
            '</div>' +
          '</div>' +
        '</div>' +
      '</div>';

    // 如果非弹窗模式（独立页面 /profile），底部追加常规保存操作条
    if (!document.getElementById('profileModalFooter')) {
      host.innerHTML +=
        '<div class="profile-actions">' +
          '<button type="button" class="action-btn primary" id="profileSaveBtn"><i class="fas fa-check"></i> 保存资料</button>' +
        '</div>';
    }

    // 绑定密码可见性切换
    host.querySelectorAll('.pwd-toggle-btn').forEach(function (b) {
      b.addEventListener('click', function () {
        togglePasswordVisibility(b, b.getAttribute('data-target'));
      });
    });

    // 绑定修改密码折叠展开
    host.querySelectorAll('[data-toggle-target]').forEach(function (b) {
      b.addEventListener('click', function () {
        var target = document.getElementById(b.getAttribute('data-toggle-target'));
        if (!target) return;
        var expanded = b.getAttribute('aria-expanded') === 'true';
        if (expanded) {
          target.setAttribute('hidden', '');
          b.setAttribute('aria-expanded', 'false');
          b.querySelector('span').textContent = '展开设置';
        } else {
          target.removeAttribute('hidden');
          b.setAttribute('aria-expanded', 'true');
          b.querySelector('span').textContent = '收起设置';
        }
      });
    });

    var saveBtn = document.getElementById('profileSaveBtn');
    if (saveBtn) {
      saveBtn.onclick = function () { window.saveProfileForm(); };
    }
    var pwdBtn = document.getElementById('profilePwdSaveBtn');
    if (pwdBtn) {
      pwdBtn.onclick = function () { changePassword(); };
    }
  }

  function readPreferredRolesFromDom() {
    var nodes = document.querySelectorAll('input[name="profilePreferredRole"]:checked');
    var out = [];
    nodes.forEach(function (el) {
      var key = String(el.value || '').toLowerCase();
      if (key && !ORG_ONLY_ROLES[key]) out.push(key);
    });
    // 保障至少保留开放的 PO 视图
    if (!out.length) {
      out.push('po');
    }
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
    var realname = cleanAccountName(p.displayName || (window.currentUser && window.currentUser.realname) || account, account);
    var preferred = Array.isArray(p.preferredRoles) ? p.preferredRoles.slice() : ['po'];
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
        throw new Error((json && json.message) || '保存失败');
      }

      var savedRoles = (json.data && json.data.preferredRoles) || payload.preferredRoles;
      var savedTeam = Number(json.data && json.data.mainTeamId != null ? json.data.mainTeamId : payload.mainTeamId);
      syncPreferredToSwitcher(savedRoles);
      if (profileCache) {
        profileCache.preferredRoles = savedRoles;
        profileCache.mainTeamId = savedTeam;
        profileCache.email = payload.email;
        profileCache.gender = payload.gender;
      }
      showToast(json.message || '个人资料已保存', 'success');

      var note = document.getElementById('profileAuditNote');
      if (note) note.remove();

      if (typeof window.closeProfileModal === 'function') {
        var modal = document.getElementById('profileModal');
        if (modal && modal.classList.contains('show')) window.closeProfileModal();
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

  // 弹窗模式模态框 (支持 openProfileModal / closeProfileModal)
  function onProfileOverlayClick(e) {
    if (e.target !== e.currentTarget) return;
    window.closeProfileModal();
  }

  window.ensureProfileModalDom = function ensureProfileModalDom() {
    var overlay = document.getElementById('profileModalOverlay');
    var modal = document.getElementById('profileModal');
    if (overlay && modal && document.getElementById('profileModalBody') && document.getElementById('profileModalFooter')) {
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
        '<h3 id="profileModalTitle"><i class="fas fa-id-card"></i> 个人资料与偏好设置</h3>' +
        '<button type="button" class="modal-close" aria-label="关闭" onclick="window.closeProfileModal()"><i class="fas fa-times"></i></button>' +
      '</div>' +
      '<div class="modal-body" id="profileModalBody"></div>' +
      '<div class="modal-footer" id="profileModalFooter">' +
        '<div class="modal-footer-hint"><i class="fas fa-shield-halved"></i> 资料直接同步生效至禅道与研发工作台</div>' +
        '<div class="modal-footer-actions">' +
          '<button type="button" class="btn-modal-cancel" onclick="window.closeProfileModal()">取消</button>' +
          '<button type="button" class="action-btn primary" id="profileSaveBtn"><i class="fas fa-check"></i> 保存资料</button>' +
        '</div>' +
      '</div>';

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
        window.saveProfileForm();
      }
    }
  });

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      if (document.getElementById('profilePageBody')) {
        window.renderProfilePage();
      }
    });
  } else if (document.getElementById('profilePageBody')) {
    window.renderProfilePage();
  }
})();
