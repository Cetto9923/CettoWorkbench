/**
 * 文件: web/static/js/layout/pinned-pages.js
 * 模块: 工作台
 * 职责: 管理用户自定义固定常用功能，支持 localStorage 隔离持久化、顶栏/侧栏即时联动与自定义弹窗。
 */
(function () {
  'use strict';

  var STORAGE_KEY_PREFIX = 'workbench.pinned_pages.';
  var DEFAULT_PINNED = ['schedule', 'query', 'board_demand', 'agileteam', 'issues_risk'];

  var PINNABLE_PAGES = {};

  /* 历史固定 key 与默认排序保留不变，避免用户已保存的 localStorage 失效；
     其标题、图标、链接改为从渲染好的侧栏读取，菜单改名或改链接后自动跟随。 */
  var LEGACY_KEYS = {
    '/schedule': 'schedule',
    '/query': 'query',
    '/board/demand': 'board_demand',
    '/board/task': 'board_task',
    '/agileteam': 'agileteam',
    '/metrics/manage': 'metrics_manage',
    '/metrics/radar': 'metrics_radar',
    '/issues/risk': 'issues_risk'
  };

  // 侧栏二级菜单由服务端按 zt_menus 渲染，这里把它读成可固定的页面清单。
  // 顶栏模式下没有侧栏容器，此时用历史清单兜底，保证常用功能仍可固定。
  function collectPinnablePages() {
    var nav = document.getElementById('sidebar') || document.body;
    if (!nav || typeof nav.querySelectorAll !== 'function') {
      Object.keys(LEGACY_KEYS).forEach(function (path) {
        PINNABLE_PAGES[LEGACY_KEYS[path]] = {
          key: LEGACY_KEYS[path], title: LEGACY_KEYS[path], path: path,
          icon: 'fas fa-file', group: '', groupTitle: ''
        };
      });
      return;
    }
    nav.querySelectorAll('.po-subnav-panel').forEach(function (panel) {
      var titleEl = panel.querySelector('.po-subnav-title');
      var groupTitle = (titleEl && titleEl.textContent) || '';
      var group = (panel.getAttribute && panel.getAttribute('data-subnav-panel') || '').trim();
      var links = panel.querySelectorAll ? panel.querySelectorAll('.po-subnav-body a.nav-item[href]') : [];
      Array.prototype.forEach.call(links, function (link) {
        var path = (link.getAttribute('href') || '').trim();
        if (!path || path.charAt(0) !== '/' || path === '/home') return;
        if (link.closest && link.closest('#sidebarPinnedContainer, .po-pinned-list')) return;
        var textEl = link.querySelector('.nav-text');
        var iconEl = link.querySelector('.nav-icon');
        var title = ((textEl || link).textContent || '').trim();
        if (!title) return;
        var key = LEGACY_KEYS[path] || ('page:' + path);
        PINNABLE_PAGES[key] = {
          key: key,
          title: title,
          path: path,
          icon: (iconEl && iconEl.className) || 'fas fa-file',
          group: group,
          groupTitle: groupTitle.trim()
        };
      });
    });
    // 未在侧栏出现的旧地址（如仅顶栏可见的页面）仍按历史 key 兜底，保证已固定的项不丢。
    Object.keys(LEGACY_KEYS).forEach(function (path) {
      var key = LEGACY_KEYS[path];
      if (!PINNABLE_PAGES[key]) {
        PINNABLE_PAGES[key] = {
          key: key, title: key, path: path,
          icon: 'fas fa-file', group: '', groupTitle: ''
        };
      }
    });
  }

  collectPinnablePages();

  function getCurrentAccount() {
    var userEl = document.querySelector('[data-user-account]');
    if (userEl && userEl.dataset && userEl.dataset.userAccount) {
      return userEl.dataset.userAccount;
    }
    var avatar = document.querySelector('.user-avatar, .user-name, [title*="("]');
    if (avatar && avatar.title) {
      var match = avatar.title.match(/\(([^)]+)\)/);
      if (match && match[1]) return match[1];
    }
    return 'default';
  }

  function getStorageKey() {
    return STORAGE_KEY_PREFIX + getCurrentAccount();
  }

  function getPinnedKeys() {
    try {
      var val = localStorage.getItem(getStorageKey());
      if (!val) {
        return DEFAULT_PINNED.slice();
      }
      var parsed = JSON.parse(val);
      if (Array.isArray(parsed)) {
        return parsed.filter(function (k) { return !!PINNABLE_PAGES[k]; });
      }
    } catch (e) {
      console.warn('Failed to parse pinned pages:', e);
    }
    return DEFAULT_PINNED.slice();
  }

  function savePinnedKeys(keys) {
    try {
      var unique = [];
      for (var i = 0; i < keys.length; i++) {
        var k = keys[i];
        if (PINNABLE_PAGES[k] && unique.indexOf(k) === -1) {
          unique.push(k);
        }
      }
      localStorage.setItem(getStorageKey(), JSON.stringify(unique));
      dispatchChangeEvent();
      return true;
    } catch (e) {
      console.warn('Failed to save pinned pages:', e);
      return false;
    }
  }

  function keyFromPathOrKey(val) {
    if (!val) return null;
    if (PINNABLE_PAGES[val]) return val;
    if (val === '/pmo') val = '/agileteam';
    if (val === '/issue-risk') val = '/issues/risk';
    for (var k in PINNABLE_PAGES) {
      if (PINNABLE_PAGES[k].path === val) return k;
    }
    var parent = Object.keys(PINNABLE_PAGES).filter(function (key) {
      return val.indexOf(PINNABLE_PAGES[key].path + '/') === 0;
    }).sort(function (a, b) { return PINNABLE_PAGES[b].path.length - PINNABLE_PAGES[a].path.length; });
    return parent[0] || null;
  }

  function isPinned(keyOrPath) {
    var key = keyFromPathOrKey(keyOrPath);
    if (!key) return false;
    return getPinnedKeys().indexOf(key) !== -1;
  }

  function pin(keyOrPath) {
    var key = keyFromPathOrKey(keyOrPath);
    if (!key) return false;
    var keys = getPinnedKeys();
    if (keys.indexOf(key) === -1) {
      keys.push(key);
      savePinnedKeys(keys);
    }
    return true;
  }

  function unpin(keyOrPath) {
    var key = keyFromPathOrKey(keyOrPath);
    if (!key) return false;
    var keys = getPinnedKeys();
    var idx = keys.indexOf(key);
    if (idx !== -1) {
      keys.splice(idx, 1);
      savePinnedKeys(keys);
    }
    return true;
  }

  function togglePin(keyOrPath) {
    var key = keyFromPathOrKey(keyOrPath);
    if (!key) return false;
    if (isPinned(key)) {
      unpin(key);
      return false;
    } else {
      pin(key);
      return true;
    }
  }

  function dispatchChangeEvent() {
    var ev = new CustomEvent('workbench:pinned-change', {
      detail: { pinned: getPinnedKeys() }
    });
    window.dispatchEvent(ev);
  }

  function initTopbarPinToggle() {
    var btn = document.getElementById('pagePinToggle');
    if (!btn) return;

    var currentPath = window.location.pathname;
    var key = keyFromPathOrKey(currentPath);

    if (!key || currentPath === '/home') {
      btn.style.display = 'none';
      return;
    }

    btn.style.display = 'inline-flex';
    updateTopbarPinState(btn, key);

    btn.addEventListener('click', function (e) {
      e.preventDefault();
      var pinnedNow = togglePin(key);
      updateTopbarPinState(btn, key);
      showPinnedToast(pinnedNow, PINNABLE_PAGES[key].title);
    });

    window.addEventListener('workbench:pinned-change', function () {
      updateTopbarPinState(btn, key);
    });
  }

  function updateTopbarPinState(btn, key) {
    var pinned = isPinned(key);
    var textEl = btn.querySelector('.page-pin-text');
    if (pinned) {
      btn.classList.add('is-pinned');
      btn.title = '点击从我的工作台取消固定';
      if (textEl) textEl.textContent = '已在工作台';
    } else {
      btn.classList.remove('is-pinned');
      btn.title = '固定至我的工作台';
      if (textEl) textEl.textContent = '固定到工作台';
    }
  }

  function showPinnedToast(isPinned, title) {
    var msg = isPinned
      ? '已将「' + title + '」固定到我的工作台'
      : '已将「' + title + '」从我的工作台移除';
    if (typeof window.showToast === 'function') {
      window.showToast(msg, 'info');
    }
  }

  function syncPinnedMenuState() {
    document.querySelectorAll('.po-subnav-panel:not([data-subnav-panel="personal"]) a.nav-item[href]').forEach(function (link) {
      link.classList.toggle('is-workbench-pinned', isPinned(link.getAttribute('href')));
    });
  }

  window.addEventListener('workbench:pinned-change', syncPinnedMenuState);
  window.addEventListener('pageshow', syncPinnedMenuState);
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', syncPinnedMenuState);
    document.addEventListener('DOMContentLoaded', initTopbarPinToggle);
  } else {
    syncPinnedMenuState();
    initTopbarPinToggle();
  }

  window.WorkbenchPinned = {
    PINNABLE_PAGES: PINNABLE_PAGES,
    getPinnedKeys: getPinnedKeys,
    savePinnedKeys: savePinnedKeys,
    isPinned: isPinned,
    pin: pin,
    unpin: unpin,
    togglePin: togglePin,
    keyFromPathOrKey: keyFromPathOrKey
  };
})();
