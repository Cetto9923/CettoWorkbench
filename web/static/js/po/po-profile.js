// =============================================================================
// 文件: web/static/js/po/po-profile.js
// 模块: PO 工作台 / 个人资料（/profile 与 shared profile modal）
// 职责: 对接 /api/profile 与 /profile/data；
//       支持查看账号信息、编辑邮箱/性别/工作台视图/默认敏捷小组、修改密码；
//       采用双栏无滚动宽幅大弹窗美化设计，操作吸底常驻；
//       支持页面模式渲染与 openProfileModal 弹窗模式；
//       支持 RoleSwitcher 联动顶部导航栏视图 Tab（隐藏未勾选/未授权视图）。
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

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || window.escapeHtml || function (s) { return String(s == null ? '' : s); };

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

  // =============================================================================
  // 工作台角色切换联动器 (RoleSwitcher)
  // 负责顶部导航栏角色 tab 的显隐联动与本地持久化
  // =============================================================================
  window.RoleSwitcher = {
    detectRole: function () {
      var activeTab = document.querySelector('.po-role-tab.active');
      if (activeTab) {
        return (activeTab.getAttribute('data-role') || activeTab.textContent || '').trim().toLowerCase();
      }
      return 'po';
    },

    getOrgRoles: function () {
      var roles = profileCache && profileCache.orgRoles;
      return Array.isArray(roles) ? roles.map(function (r) { return typeof r === 'string' ? r : r.key; }) : [];
    },

    setPreferredRoles: function (roles, account) {
      var list = Array.isArray(roles) ? roles.map(function (r) { return String(r || '').toLowerCase(); }) : [];
      var acc = String(account || (profileCache && profileCache.account) || (window.currentUser && window.currentUser.account) || '').trim();
      var orgRoles = this.getOrgRoles();

      // 有效可见角色 = 用户勾选的自选角色 + 组织授权的角色
      var activeMap = {};
      list.forEach(function (r) {
        if (r && !ORG_ONLY_ROLES[r]) activeMap[r] = true;
      });
      orgRoles.forEach(function (r) {
        if (r) activeMap[r] = true;
      });

      // 默认开放 PO 与 PMO 视图
      activeMap['po'] = true;
      activeMap['pmo'] = true;

      try {
        localStorage.setItem('wb_preferred_roles_v1', JSON.stringify({
          preferredRoles: list,
          effectiveRoles: Object.keys(activeMap)
        }));
      } catch (e) { /* ignore */ }

      // 顶部切换栏已解耦收口，此处保留角色偏好持久化供后续视角/能力加载使用
    },

    init: function () {
      var acc = String((window.currentUser && window.currentUser.account) || '').trim();
      var stored = null;
      try {
        var raw = localStorage.getItem('wb_preferred_roles_v1');
        if (raw) stored = JSON.parse(raw);
      } catch (e) {}

      var prefList = (stored && Array.isArray(stored.preferredRoles)) ? stored.preferredRoles : ['po'];
      this.setPreferredRoles(prefList, acc);
    }
  };

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
    var account = String((p && p.account) || (window.currentUser && window.currentUser.account) || '').trim();
    var checked = preferredSet(p);

    // 1. 自选工作台角色：产品负责人 (PO)，用户可自主勾选或取消
    var selfSelectable = [
      { key: 'po', label: '产品负责人 (PO)', checked: checked['po'] !== false }
    ];

    var boxes = [];
    selfSelectable.forEach(function (r) {
      boxes.push(
        '<label class="role-checkbox-card' + (r.checked ? ' selected' : '') + '">' +
          '<input type="checkbox" name="profilePreferredRole" value="' + esc(r.key) + '"' +
          (r.checked ? ' checked' : '') + '> ' +
          '<span>' + esc(r.label) + '</span>' +
        '</label>'
      );
    });

    // 2. 组织固定授权角色：服务端返回的组织角色只读展示，不可在个人资料自主修改。
    var orgRoles = window.RoleSwitcher ? window.RoleSwitcher.getOrgRoles() : [];
    orgRoles.forEach(function (key) {
      var label = key === 'pmo' ? 'PMO' : (key === 'lead' ? '团队管理' : key.toUpperCase());
      boxes.push(
        '<label class="role-checkbox-card selected is-disabled" title="组织已授权视图，须由 PMO 或管理员统一配置，个人不可修改">' +
          '<input type="checkbox" checked disabled />' +
          '<span>' + esc(label) + '</span>' +
          '<span class="readonly-tag" style="color:var(--color-primary);font-size:11px;font-weight:500;margin-left:auto;">（组织已授权 · 只读）</span>' +
        '</label>'
      );
    });

    return '<div class="role-cards-grid" style="grid-template-columns: 1fr;" id="profileRoleCheckRow">' + boxes.join('') + '</div>' +
      '<span class="field-tip" style="margin-top:6px; display:inline-block; line-height:1.4;">' +
        '提示：产品负责人 (PO) 可由个人自主选择开启或关闭；PMO 与团队管理视图属于组织固定授权，须由 PMO 或管理员在后台统一授权配置，个人不可在此更改。' +
      '</span>';
  }

  function roleFieldHtml(p) {
    var currentRole = detectWorkRole();
    var orgLocked = !!ORG_ONLY_ROLES[currentRole];
    var orgNote = '';
    if (orgLocked) {
      var label = currentRole === 'pmo' ? 'PMO' : '团队管理';
      orgNote = '<div class="field-tip" style="margin-bottom:8px;color:var(--color-primary);font-weight:500;">当前正处于「' + esc(label) +
        '」工作视角（组织固定授权）。</div>';
    }
    return orgNote + roleCheckboxesHtml(p);
  }

  function agileGroupsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    if (!groups.length) {
      return '<span class="profile-empty-text">暂无从禅道团队成员关系识别到已加入的敏捷小组</span>';
    }
    return groups.map(function (g) {
      return '<span class="agile-chip-pill"><i class="fas fa-user-group"></i> ' + esc(g.name || ('小组#' + g.id)) + '</span>';
    }).join('');
  }

  function mainTeamOptionsHtml(p) {
    var groups = Array.isArray(p.agileGroups) ? p.agileGroups : [];
    var selected = Number(p.mainTeamId || 0);
    var opts = ['<option value="0"' + (selected === 0 ? ' selected' : '') + '>未设置默认小组</option>'];
    groups.forEach(function (g) {
      var id = Number(g.id || 0);
      opts.push('<option value="' + id + '"' + (selected === id ? ' selected' : '') + '>' +
        esc(g.name || ('小组#' + id)) + (id ? (' (ID: ' + id + ')') : '') + '</option>');
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

  function syncThemeLabel() {
    var lbl = document.getElementById('profileThemeLabel');
    if (!lbl) return;
    var cur = (document.documentElement.getAttribute('data-theme') || '').toLowerCase();
    if (cur === 'dark') {
      lbl.textContent = '深色';
    } else {
      lbl.textContent = '浅色';
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
      '<div class="modal-content-grid">' +
        '<!-- 左栏：基本账号与身份信息 -->' +
        '<div class="section-panel">' +
          '<div class="section-panel-header">' +
            '<span><i class="fas fa-user-shield"></i> 账号身份与基本信息</span>' +
            '<span style="font-size: 11px; font-weight: 400; color: var(--color-text-muted);">部分字段来自禅道同步</span>' +
          '</div>' +

          '<!-- 用户名片区（清晰展示姓名、工号、部门、首字大头像） -->' +
          '<div class="user-identity-card">' +
            '<div class="user-avatar-big">' + esc(initial) + '</div>' +
            '<div class="user-info-text">' +
              '<div class="user-info-name-line">' +
                '<span class="user-name-title">' + esc(cleanName) + '</span>' +
                '<span class="user-account-badge">' + esc(account) + '</span>' +
                '<span class="user-dept-badge">' + esc(deptText) + '</span>' +
              '</div>' +
              '<div class="user-source-hint">' +
                '<i class="fas fa-check-circle" style="color:var(--color-success)"></i> 禅道同步账号 · 头像取姓名首字' +
              '</div>' +
            '</div>' +
          '</div>' +

          '<!-- 字段表单网格 -->' +
          '<div class="fields-grid-2">' +
            '<div class="field-item">' +
              '<label class="field-label">真实姓名 <span class="readonly-tag">（只读）</span></label>' +
              '<input class="field-control" id="profileRealname" type="text" value="' + esc(cleanName) + '" readonly disabled />' +
              '<span class="field-tip">来自禅道账号，不可在此修改</span>' +
            '</div>' +

            '<div class="field-item">' +
              '<label class="field-label">登录工号 / 用户名 <span class="readonly-tag">（只读）</span></label>' +
              '<input class="field-control" id="profileAccount" type="text" value="' + esc(account) + '" readonly disabled />' +
              '<span class="field-tip">系统登录主键</span>' +
            '</div>' +

            '<div class="field-item">' +
              '<label class="field-label">绑定手机 <span class="readonly-tag">（只读）</span></label>' +
              '<input class="field-control" id="profileMobile" type="text" value="' + esc(p.mobile || '') + '" readonly disabled />' +
              '<span class="field-tip">来自禅道通讯录</span>' +
            '</div>' +

            '<div class="field-item">' +
              '<label class="field-label">所属部门 <span class="readonly-tag">（只读）</span></label>' +
              '<input class="field-control" id="profileDept" type="text" value="' + esc(deptText) + '" readonly disabled />' +
              '<span class="field-tip">来自银行组织架构</span>' +
            '</div>' +

            '<div class="field-item">' +
              '<label class="field-label">个人邮箱 <span style="color:var(--color-primary); font-size:11px;">（可维护）</span></label>' +
              '<input class="field-control" id="profileEmail" type="email" value="' + esc(p.email || '') + '" placeholder="暂未设置邮箱（可在此填写绑定）" />' +
              '<span class="field-tip">用于接收工作台动态与任务提醒</span>' +
            '</div>' +

            '<div class="field-item">' +
              '<label class="field-label">性别偏好 <span style="color:var(--color-primary); font-size:11px;">（可维护）</span></label>' +
              '<select class="field-control" id="profileGender">' + genderOptions(p.gender || '') + '</select>' +
              '<span class="field-tip">当前账号性别偏好</span>' +
            '</div>' +
          '</div>' +
        '</div>' +

        '<!-- 右栏：敏捷团队归属 + 自选角色视图 + 安全改密 -->' +
        '<div style="display:flex; flex-direction:column; gap: 14px;">' +
          '<!-- 敏捷小组卡片 -->' +
          '<div class="section-panel" style="padding: 14px 16px;">' +
            '<div class="section-panel-header">' +
              '<span><i class="fas fa-users-gear"></i> 敏捷小组归属</span>' +
            '</div>' +
            '<div style="display:flex; flex-direction:column; gap: 8px;">' +
              '<label class="field-label">本人已加入的小组</label>' +
              '<div class="agile-chips-wrap" id="profileAgileCheckRow">' + agileGroupsHtml(p) + '</div>' +
            '</div>' +
            '<div class="field-item" style="margin-top: 4px;">' +
              '<label class="field-label" for="profileMainTeam">默认敏捷小组（对应禅道 mainTeam）</label>' +
              '<select class="field-control" id="profileMainTeam">' + mainTeamOptionsHtml(p) + '</select>' +
              '<span class="field-tip">须选自本人已加入的敏捷小组</span>' +
            '</div>' +
          '</div>' +

          '<!-- 工作台角色多选卡片 -->' +
          '<div class="section-panel" style="padding: 14px 16px;">' +
            '<div class="section-panel-header">' +
              '<span><i class="fas fa-layer-group"></i> 工作台角色（自选多视图）</span>' +
            '</div>' +
            roleFieldHtml(p) +
          '</div>' +

          '<!-- 修改密码轻量折叠卡 -->' +
          '<div class="section-panel" style="padding: 12px 16px;">' +
            '<div class="password-toggle-header" id="pwdToggleHeader">' +
              '<span style="font-size: 13px; font-weight: 700; color: var(--color-text-primary); display:flex; align-items:center; gap:6px;">' +
                '<i class="fas fa-key" style="color:var(--color-primary)"></i> 安全修改密码' +
              '</span>' +
              '<span style="font-size: 12px; color: var(--color-primary); cursor: pointer;" id="pwdToggleText">' +
                '展开设置 <i class="fas fa-chevron-down" id="pwdToggleIcon"></i>' +
              '</span>' +
            '</div>' +
            '<div id="pwdExpandBox" class="password-expand-box" style="display: none;">' +
              '<div class="field-item">' +
                '<label class="field-label" for="profileOldPwd">当前原密码</label>' +
                '<div class="pwd-input-wrap">' +
                  '<input class="field-control" type="password" id="profileOldPwd" placeholder="请输入当前密码" autocomplete="current-password" />' +
                  '<button class="pwd-eye-btn" type="button" data-target="profileOldPwd" title="查看密码"><i class="fas fa-eye"></i></button>' +
                '</div>' +
              '</div>' +
              '<div class="pwd-row">' +
                '<div class="field-item">' +
                  '<label class="field-label" for="profileNewPwd">新密码</label>' +
                  '<div class="pwd-input-wrap">' +
                    '<input class="field-control" type="password" id="profileNewPwd" placeholder="8-64位，含字母数字" autocomplete="new-password" />' +
                    '<button class="pwd-eye-btn" type="button" data-target="profileNewPwd" title="查看密码"><i class="fas fa-eye"></i></button>' +
                  '</div>' +
                '</div>' +
                '<div class="field-item">' +
                  '<label class="field-label" for="profileNewPwd2">确认新密码</label>' +
                  '<div class="pwd-input-wrap">' +
                    '<input class="field-control" type="password" id="profileNewPwd2" placeholder="再次输入新密码" autocomplete="new-password" />' +
                    '<button class="pwd-eye-btn" type="button" data-target="profileNewPwd2" title="查看密码"><i class="fas fa-eye"></i></button>' +
                  '</div>' +
                '</div>' +
              '</div>' +
              '<button class="btn-pwd-commit" id="profilePwdSaveBtn" type="button">' +
                '<i class="fas fa-shield"></i> 更新密码' +
              '</button>' +
            '</div>' +
          '</div>' +
        '</div>' +
      '</div>';

    // 如果非弹窗模式（独立页面 /profile），底部追加保存操作栏
    if (!document.getElementById('profileModalFooter')) {
      host.innerHTML +=
        '<div class="modal-bottom-bar" style="margin-top:16px; border-radius:8px; border:1px solid var(--color-border);">' +
          '<div class="modal-bottom-hint"><i class="fas fa-shield-halved"></i> <span>资料直接同步生效至禅道与研发工作台</span></div>' +
          '<div class="modal-bottom-btns">' +
            '<button type="button" class="btn-modal-save" id="profileSaveBtn"><i class="fas fa-check"></i> 保存资料</button>' +
          '</div>' +
        '</div>';
    }

    // 绑定角色卡片点击交互（置灰项不响应点击）
    host.querySelectorAll('.role-checkbox-card').forEach(function (card) {
      if (card.classList.contains('is-disabled')) {
        return;
      }
      var cb = card.querySelector('input[type="checkbox"]');
      if (!cb || cb.disabled) return;
      card.addEventListener('click', function (e) {
        if (e.target !== cb) {
          cb.checked = !cb.checked;
        }
        card.classList.toggle('selected', cb.checked);
      });
      cb.addEventListener('change', function () {
        card.classList.toggle('selected', cb.checked);
      });
    });

    // 绑定密码可见性切换
    host.querySelectorAll('.pwd-eye-btn').forEach(function (b) {
      b.addEventListener('click', function () {
        togglePasswordVisibility(b, b.getAttribute('data-target'));
      });
    });

    // 绑定修改密码折叠展开
    var pwdHeader = host.querySelector('#pwdToggleHeader');
    if (pwdHeader) {
      pwdHeader.addEventListener('click', function () {
        var box = host.querySelector('#pwdExpandBox');
        var text = host.querySelector('#pwdToggleText');
        if (!box) return;
        var isHidden = box.style.display === 'none' || box.hasAttribute('hidden');
        if (isHidden) {
          box.style.display = 'flex';
          box.removeAttribute('hidden');
          if (text) text.innerHTML = '收起 <i class="fas fa-chevron-up"></i>';
        } else {
          box.style.display = 'none';
          box.setAttribute('hidden', '');
          if (text) text.innerHTML = '展开设置 <i class="fas fa-chevron-down"></i>';
        }
      });
    }

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
      if (el.disabled) return;
      var key = String(el.value || '').toLowerCase();
      if (key && !ORG_ONLY_ROLES[key]) out.push(key);
    });
    // 保障至少保留开放的 PO 视图
    if (!out.length) {
      out.push('po');
    }
    return out;
  }

  function syncPreferredToSwitcher(roles, account) {
    var list = Array.isArray(roles) ? roles : [];
    var acc = account || (profileCache && profileCache.account) || '';
    if (window.RoleSwitcher && typeof window.RoleSwitcher.setPreferredRoles === 'function') {
      window.RoleSwitcher.setPreferredRoles(list, acc);
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
    syncPreferredToSwitcher(profileCache.preferredRoles || [], profileCache.account);
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

    // 若已有缓存数据，先行秒开渲染，杜绝空白等待
    if (profileCache) {
      renderBody(host, profileCache, '');
      host.hidden = false;
    } else {
      host.innerHTML =
        '<div class="state-placeholder" style="display:flex;align-items:center;justify-content:center;min-height:300px;gap:10px;color:var(--color-text-secondary);font-size:14px;">' +
          '<i class="fas fa-circle-notch fa-spin"></i> 正在加载个人资料…' +
        '</div>';
      host.hidden = false;
    }

    if (profileLoading) return;
    profileLoading = true;

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
      if (!profileCache) {
        host.innerHTML =
          '<div class="state-placeholder error" style="display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:300px;gap:12px;color:var(--color-danger);font-size:14px;">' +
            '<div><i class="fas fa-circle-exclamation"></i> ' + esc((e && e.message) || '加载个人资料失败') + '</div>' +
            '<button type="button" class="btn-modal-save" style="height:32px;font-size:12px;padding:0 14px;" onclick="window.renderProfilePage()"><i class="fas fa-rotate-right"></i> 重试</button>' +
          '</div>';
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
      if (profileCache) {
        profileCache.preferredRoles = savedRoles;
        profileCache.mainTeamId = savedTeam;
        profileCache.email = payload.email;
        profileCache.gender = payload.gender;
      }
      syncPreferredToSwitcher(savedRoles, (profileCache && profileCache.account) || '');
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
    overlay.className = 'modal-overlay prototype-backdrop';
    overlay.id = 'profileModalOverlay';
    overlay.addEventListener('click', onProfileOverlayClick);

    modal = document.createElement('div');
    modal.className = 'modal profile-modal profile-modal-window';
    modal.id = 'profileModal';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'profileModalTitle');
    modal.innerHTML =
      '<div class="modal-head">' +
        '<div class="modal-head-title">' +
          '<i class="fas fa-id-card"></i>' +
          '<span id="profileModalTitle">个人资料与偏好设置</span>' +
        '</div>' +
        '<div class="modal-head-actions">' +
          '<button class="theme-pill-btn" id="profileThemeToggleBtn" type="button" title="切换深浅主题">' +
            '<i class="fas fa-circle-half-stroke"></i>' +
            '<span id="profileThemeLabel">浅色</span>' +
          '</button>' +
          '<button type="button" class="modal-close-icon" aria-label="关闭" title="关闭" onclick="window.closeProfileModal()"><i class="fas fa-times"></i></button>' +
        '</div>' +
      '</div>' +
      '<div class="modal-body" id="profileModalBody"></div>' +
      '<div class="modal-bottom-bar" id="profileModalFooter">' +
        '<div class="modal-bottom-hint"><i class="fas fa-shield-halved"></i> <span>资料直接同步生效至禅道与研发工作台</span></div>' +
        '<div class="modal-bottom-btns">' +
          '<button type="button" class="btn-modal-cancel" onclick="window.closeProfileModal()">取消</button>' +
          '<button type="button" class="btn-modal-save" id="profileSaveBtn"><i class="fas fa-check"></i> 保存资料</button>' +
        '</div>' +
      '</div>';

    document.body.appendChild(overlay);
    document.body.appendChild(modal);

    // 绑定主题切换按钮事件
    var themeBtn = modal.querySelector('#profileThemeToggleBtn');
    if (themeBtn) {
      themeBtn.addEventListener('click', function () {
        var curTheme = document.documentElement.getAttribute('data-theme') || 'light';
        var nextTheme = curTheme === 'dark' ? 'light' : 'dark';
        if (window.WorkbenchTheme && typeof window.WorkbenchTheme.setPreference === 'function') {
          window.WorkbenchTheme.setPreference(nextTheme);
        } else {
          document.documentElement.setAttribute('data-theme', nextTheme);
        }
        syncThemeLabel();
      });
    }

    return { overlay: overlay, modal: modal };
  };

  window.openProfileModal = function openProfileModal(e) {
    if (e && e.preventDefault) e.preventDefault();
    if (e && e.stopPropagation) e.stopPropagation();

    // 关闭已展开的用户下拉菜单
    document.querySelectorAll('.dropdown.open, .dropdown.show').forEach(function (d) {
      d.classList.remove('open', 'show');
    });

    var nodes = window.ensureProfileModalDom();
    syncThemeLabel();
    window.renderProfilePage();

    nodes.overlay.classList.add('show');
    nodes.modal.classList.add('show');
    return false;
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

  // 页面加载阶段初始化角色 tab 显隐
  if (window.RoleSwitcher && typeof window.RoleSwitcher.init === 'function') {
    window.RoleSwitcher.init();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', function () {
      if (window.RoleSwitcher && typeof window.RoleSwitcher.init === 'function') {
        window.RoleSwitcher.init();
      }
      if (document.getElementById('profilePageBody')) {
        window.renderProfilePage();
      }
    });
  } else {
    if (window.RoleSwitcher && typeof window.RoleSwitcher.init === 'function') {
      window.RoleSwitcher.init();
    }
    if (document.getElementById('profilePageBody')) {
      window.renderProfilePage();
    }
  }
})();
