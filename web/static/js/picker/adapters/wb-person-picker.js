/* =============================================================================
 * 文件: web/static/workbench/shared/picker/adapters/wb-person-picker.js
 * 职责: 人员单选 adapter。value 真源为 account；preferred 只排序/置顶，不限制 allowed。
 * ============================================================================= */
(function (global) {
  'use strict';

  var optionsCache = { key: '', value: null };

  function text(v) { return String(v == null ? '' : v).trim(); }
  function esc(s) {
    return WBUtils.escapeHtml(s);
  }
  function userLabel(u) {
    var account = text(u && u.account);
    var name = text(u && (u.realname || u.name || u.label || account));
    if (!name || name === account) return account;
    var re = new RegExp('[（(]\\s*' + account.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\s*[）)]\\s*$');
    if (re.test(name)) return name;
    return name + '（' + account + '）';
  }
  function normalizeUsers(users) {
    return (Array.isArray(users) ? users : []).map(function (u) {
      var account = text(u && (u.account || u.value));
      if (!account) return null;
      return {
        account: account,
        realname: text(u && (u.realname || u.name || u.label || account)) || account,
        pinyin: text(u && u.pinyin),
        deptId: Number(u && (u.deptId || u.deptID)) || 0,
        deptName: text(u && u.deptName)
      };
    }).filter(Boolean);
  }
  function preferredSet(preferred) {
    var out = new Set();
    (Array.isArray(preferred) ? preferred : []).forEach(function (v) {
      if (typeof v === 'string') out.add(text(v));
      else {
        if (v && v.account) out.add(text(v.account));
        if (v && v.value) out.add(text(v.value));
      }
    });
    return out;
  }
  function enrichOptionSearchText(opt) {
    if (!opt || opt.searchText) return opt;
    if (global.WbPickerSearch && typeof global.WbPickerSearch.optionSearchText === 'function') {
      opt.searchText = global.WbPickerSearch.optionSearchText(opt);
    }
    return opt;
  }
  function enrichGroupSearchText(groups) {
    (groups || []).forEach(function (g) {
      if (!g) return;
      if (Array.isArray(g.children)) g.children.forEach(enrichOptionSearchText);
    });
    return groups;
  }
  function buildOptions(users, preferred) {
    var pref = preferredSet(preferred);
    var rows = normalizeUsers(users).slice().sort(function (a, b) {
      var ap = pref.has(a.account) ? 0 : 1;
      var bp = pref.has(b.account) ? 0 : 1;
      if (ap !== bp) return ap - bp;
      return a.account.localeCompare(b.account);
    });
    var pinned = [];
    var byDept = {};
    rows.forEach(function (u) {
      var opt = enrichOptionSearchText({
        id: u.account,
        value: u.account,
        label: userLabel(u),
        subLabel: u.deptName || '',
        avatar: true,
        avatarName: u.realname || u.account,
        meta: {
          account: u.account,
          realname: u.realname,
          pinyin: u.pinyin,
          deptName: u.deptName
        }
      });
      if (pref.has(u.account)) pinned.push(opt);
      var key = u.deptName || '未分配部门';
      if (!byDept[key]) byDept[key] = [];
      byDept[key].push(opt);
    });
    var groups = [];
    if (pinned.length) groups.push({ id: '__preferred', label: '常用 / 当前小组', selectable: false, children: pinned });
    Object.keys(byDept).sort().forEach(function (dept) {
      groups.push({ id: 'dept:' + dept, label: dept, selectable: false, children: byDept[dept] });
    });
    return groups;
  }
  function buildOptionsCached(users, preferred) {
    var normalized = normalizeUsers(users);
    var prefKey = (Array.isArray(preferred) ? preferred : []).map(function (v) {
      return typeof v === 'string' ? v : text(v && (v.account || v.value));
    }).join(',');
    var key = normalized.length + '|' + prefKey + '|' + (normalized[0] && normalized[0].account) + '|' + (normalized[normalized.length - 1] && normalized[normalized.length - 1].account);
    if (optionsCache.key === key && optionsCache.value) return optionsCache.value;
    var groups = enrichGroupSearchText(buildOptions(normalized, preferred));
    optionsCache.key = key;
    optionsCache.value = groups;
    return groups;
  }

  function applyDisabledAccounts(options, disabledSet) {
    if (!disabledSet || !disabledSet.size) return options;
    function walk(list) {
      (list || []).forEach(function (it) {
        if (!it) return;
        if (it.children && it.children.length) { walk(it.children); return; }
        if (disabledSet.has(it.value)) it.disabled = true;
      });
    }
    walk(options);
    return options;
  }
  function selectOptions(select) {
    return Array.prototype.slice.call(select ? select.options : []).map(function (o) {
      var value = text(o.value);
      if (!value) return null;
      var label = text(o.textContent || value);
      var m = label.match(/^(.*?)（([^）]+)）$/);
      return {
        account: value,
        realname: m ? text(m[1]) : label,
        deptName: text(select && select.getAttribute('data-dept-name'))
      };
    }).filter(Boolean);
  }
  function loadUsers(opts, select) {
    if (Array.isArray(opts.users) && opts.users.length) return Promise.resolve(normalizeUsers(opts.users));
    if (select && select.getAttribute('data-wb-directory-source') !== 'all' && select.options && select.options.length > 1) {
      return Promise.resolve(normalizeUsers(selectOptions(select)));
    }
    if (global.WbDirectory && typeof global.WbDirectory.users === 'function') return global.WbDirectory.users().then(normalizeUsers);
    return Promise.resolve([]);
  }
  function resolveInitialAccount(value, users) {
    var raw = text(value);
    if (!raw || /待分配|待定/.test(raw)) return '';
    var exact = users.filter(function (u) { return u.account === raw; });
    if (exact.length === 1) return exact[0].account;
    var byName = users.filter(function (u) { return u.realname === raw; });
    return byName.length === 1 ? byName[0].account : raw;
  }
  function ensureSelectValue(select, value, users) {
    if (!select) return;
    var v = text(value);
    var label = v;
    var hit = (users || []).find(function (u) { return u.account === v; });
    if (hit) label = userLabel(hit);
    if (v && !Array.prototype.slice.call(select.options).some(function (o) { return o.value === v; })) {
      var opt = document.createElement('option');
      opt.value = v;
      opt.textContent = label;
      select.appendChild(opt);
    }
    select.value = v;
  }
  function mount(opts) {
    opts = opts || {};
    var el = typeof opts.el === 'string' ? document.querySelector(opts.el) : opts.el;
    if (!el) throw new Error('WbPersonPicker.mount requires el');
    var select = el.tagName === 'SELECT' ? el : null;
    var host = select ? document.createElement('div') : el;
    if (select) {
      if (select.__wbPersonPicker) return select.__wbPersonPicker;
      host.className = 'wb-person-picker-host';
      select.classList.add('wb-person-picker-source');
      select.setAttribute('aria-hidden', 'true');
      select.tabIndex = -1;
      select.parentNode.insertBefore(host, select.nextSibling);
    }
    var mode = opts.mode === 'multi' ? 'multi' : 'single';
    var disabledSet = new Set(Array.isArray(opts.disabledAccounts) ? opts.disabledAccounts : []);
    var usersRef = [];
    var shellPicker = null;
    var shellValue = text(opts.value != null ? opts.value : (select && select.value));
    if (global.WbPicker && typeof global.WbPicker.mount === 'function') {
      shellPicker = global.WbPicker.mount({
        el: host,
        mode: mode,
        layout: 'grouped',
        options: [],
        value: mode === 'multi' ? [] : shellValue,
        placeholder: opts.placeholder || '搜索姓名 / 拼音 / account',
        emptyLabel: opts.emptyLabel || '加载人员...',
        disabled: true,
        onChange: function (value) {
          if (select) {
            ensureSelectValue(select, value, usersRef);
            select.dispatchEvent(new Event('change', { bubbles: true }));
          }
          if (typeof opts.onChange === 'function') opts.onChange(value);
        }
      });
    } else {
      host.innerHTML = '<div class="wb-picker-empty">加载人员...</div>';
    }
    var api = {
      getValue: function () { return shellPicker ? shellPicker.getValue() : (select ? text(select.value) : ''); },
      setValue: function (value) {
        if (shellPicker) shellPicker.setValue(value);
        else if (select) select.value = text(value);
      },
      destroy: function () {
        if (shellPicker && shellPicker.destroy) shellPicker.destroy();
        else host.innerHTML = '';
        if (select) select.classList.remove('wb-person-picker-source');
      }
    };
    loadUsers(opts, select).then(function (users) {
      usersRef = users;
      var current = resolveInitialAccount(opts.value != null ? opts.value : (select && select.value), users);
      ensureSelectValue(select, current, users);
      var options = buildOptionsCached(users, opts.preferred);
      applyDisabledAccounts(options, disabledSet);
      if (!shellPicker) {
        host.innerHTML = '<div class="wb-picker-empty">' + esc('人员控件未加载') + '</div>';
        return;
      }
      shellPicker.setOptions(options);
      shellPicker.setDisabled(!!(opts.disabled || (select && select.disabled)));
      if (mode === 'multi') shellPicker.setValue(Array.isArray(opts.value) ? opts.value : []);
      else shellPicker.setValue(current);
      api.getValue = function () { return shellPicker.getValue(); };
      api.setValue = function (value) {
        if (mode === 'multi') shellPicker.setValue(Array.isArray(value) ? value : []);
        else {
          var v = text(value);
          shellPicker.setValue(v);
          ensureSelectValue(select, v, usersRef);
        }
      };
      api.setDisabled = shellPicker.setDisabled;
      api.destroy = function () {
        shellPicker.destroy();
        if (select) select.classList.remove('wb-person-picker-source');
      };
      if (select) select.__wbPersonPicker = api;
    }).catch(function (err) {
      if (shellPicker && shellPicker.destroy) shellPicker.destroy();
      shellPicker = null;
      host.innerHTML = '<div class="wb-picker-empty">' + esc((err && err.message) || '人员加载失败') + '</div>';
    });
    if (select) select.__wbPersonPicker = api;
    return api;
  }
  function upgrade(root) {
    root = root || document;
    root.querySelectorAll('select[data-wb-person-picker]').forEach(function (select) {
      if (select.__wbPersonPicker) return;
      mount({
        el: select,
        preferred: (select.getAttribute('data-preferred-accounts') || '').split(',').map(text).filter(Boolean),
        disabled: select.disabled
      });
    });
  }
  function observe() {
    if (!document.body || observe.started) return;
    observe.started = true;
    upgrade(document);
    new MutationObserver(function (mutations) {
      mutations.forEach(function (m) {
        Array.prototype.slice.call(m.addedNodes || []).forEach(function (node) {
          if (node && node.nodeType === 1) upgrade(node);
        });
      });
    }).observe(document.body, { childList: true, subtree: true });
  }
  function prefetchUsers() {
    if (global.WbDirectory && typeof global.WbDirectory.users === 'function') {
      return global.WbDirectory.users().catch(function () { return []; });
    }
    return Promise.resolve([]);
  }

  global.WbPersonPicker = { mount: mount, upgrade: upgrade, prefetchUsers: prefetchUsers };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', observe);
  else observe();
})(typeof window !== 'undefined' ? window : globalThis);
