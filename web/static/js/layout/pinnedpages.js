/**
 * 文件: web/static/js/layout/pinnedpages.js
 * 模块: 工作台
 * 职责: 顶栏「固定到工作台」——读写服务端 /usermenus，title/path 与 zt_menus 一致。
 */
(function () {
  'use strict';

  var API = '/usermenus';
  var pinnedItems = [];
  var currentMatch = null;

  function appFetch(url, options) {
    if (typeof window.appFetch === 'function') {
      return window.appFetch(url, options);
    }
    var opts = options || {};
    var headers = new Headers(opts.headers || {});
    var csrfEl = document.querySelector('meta[name="csrf-token"]');
    if (csrfEl) {
      headers.set('X-CSRF-Token', csrfEl.getAttribute('content') || '');
    }
    headers.set('X-Requested-With', 'XMLHttpRequest');
    opts.headers = headers;
    return fetch(url, opts);
  }

  function matchItemForPath(path) {
    var best = null;
    var bestLen = -1;
    for (var i = 0; i < pinnedItems.length; i++) {
      var item = pinnedItems[i];
      var p = (item.path || '').trim();
      if (!p) continue;
      if (path === p || path.indexOf(p + '/') === 0) {
        if (p.length > bestLen) {
          best = item;
          bestLen = p.length;
        }
      }
    }
    return best;
  }

  function updateTopbarPinState(btn) {
    var textEl = btn.querySelector('.page-pin-text');
    if (currentMatch) {
      btn.classList.add('is-pinned');
      btn.title = '点击从我的工作台取消固定';
      if (textEl) textEl.textContent = '已在工作台';
      btn.dataset.menuId = String(currentMatch.menuId || '');
    } else {
      btn.classList.remove('is-pinned');
      btn.title = '固定至我的工作台';
      if (textEl) textEl.textContent = '固定到工作台';
      btn.dataset.menuId = '';
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
    document.querySelectorAll('.po-subnav-panel a.nav-item[href]').forEach(function (link) {
      if (link.closest('#sidebarPinnedContainer, .po-pinned-list, #pinnedSection')) return;
      var href = link.getAttribute('href') || '';
      link.classList.toggle('is-workbench-pinned', !!matchItemForPath(href));
    });
  }

  function loadPinnedItems() {
    return appFetch(API, { method: 'GET', headers: { Accept: 'application/json' } })
      .then(function (res) { return res.json().then(function (data) { return { res: res, data: data }; }); })
      .then(function (wrap) {
        if (!wrap.res.ok || !wrap.data || !wrap.data.success) {
          pinnedItems = [];
          return;
        }
        pinnedItems = Array.isArray(wrap.data.items) ? wrap.data.items : [];
      })
      .catch(function () {
        pinnedItems = [];
      });
  }

  function resolveCurrentMenuFromSidebar(path) {
    var nav = document.getElementById('sidebar');
    if (!nav) return null;
    var best = null;
    var bestLen = -1;
    nav.querySelectorAll('.po-subnav-body a.nav-item[href]').forEach(function (link) {
      var href = (link.getAttribute('href') || '').trim();
      if (!href || href === '/home') return;
      if (path === href || path.indexOf(href + '/') === 0) {
        if (href.length > bestLen) {
          var textEl = link.querySelector('.nav-text');
          best = { path: href, title: ((textEl || link).textContent || '').trim() };
          bestLen = href.length;
        }
      }
    });
    return best;
  }

  function pinCurrent(path) {
    return appFetch(API, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ path: path })
    }).then(function (res) {
      return res.json().then(function (data) { return { res: res, data: data }; });
    });
  }

  function unpinCurrent(menuId) {
    return appFetch(API + '/' + encodeURIComponent(menuId), {
      method: 'DELETE',
      headers: { Accept: 'application/json' }
    }).then(function (res) {
      return res.json().then(function (data) { return { res: res, data: data }; });
    });
  }

  function reorderPinned(menuIds) {
    return appFetch(API, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({ menuIds: menuIds })
    }).then(function (res) {
      return res.json().then(function (data) { return { res: res, data: data }; });
    });
  }

  function notifyPinnedChange() {
    window.dispatchEvent(new CustomEvent('workbench:pinned-change', {
      detail: { pinned: pinnedItems }
    }));
  }

  function setPinnedItems(items) {
    pinnedItems = Array.isArray(items) ? items.slice() : [];
  }

  function initTopbarPinToggle() {
    var btn = document.getElementById('pagePinToggle');
    if (!btn) return;

    var currentPath = window.location.pathname;
    if (currentPath === '/home') {
      btn.style.display = 'none';
      return;
    }

    var sidebarHit = resolveCurrentMenuFromSidebar(currentPath);
    if (!sidebarHit) {
      btn.style.display = 'none';
      return;
    }

    btn.style.display = 'inline-flex';
    currentMatch = matchItemForPath(currentPath);
    updateTopbarPinState(btn);

    btn.addEventListener('click', function (e) {
      e.preventDefault();
      if (btn.dataset.busy === '1') return;
      btn.dataset.busy = '1';

      var doingUnpin = !!currentMatch;
      var req = doingUnpin
        ? unpinCurrent(currentMatch.menuId)
        : pinCurrent(currentPath);

      req.then(function (wrap) {
        if (!wrap.res.ok || !wrap.data || !wrap.data.success) {
          var msg = (wrap.data && wrap.data.message) || '操作失败';
          if (typeof window.showToast === 'function') {
            window.showToast(msg, 'error');
          }
          return;
        }
        return loadPinnedItems().then(function () {
          currentMatch = matchItemForPath(currentPath);
          updateTopbarPinState(btn);
          syncPinnedMenuState();
          var title = (wrap.data.item && wrap.data.item.title) || sidebarHit.title || currentPath;
          if (doingUnpin) {
            title = sidebarHit.title || title;
          }
          showPinnedToast(!doingUnpin, title);
          notifyPinnedChange();
        });
      }).catch(function () {
        if (typeof window.showToast === 'function') {
          window.showToast('操作失败，请稍后重试', 'error');
        }
      }).then(function () {
        btn.dataset.busy = '0';
      });
    });
  }

  function boot() {
    loadPinnedItems().then(function () {
      initTopbarPinToggle();
      syncPinnedMenuState();
      notifyPinnedChange();
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }

  window.WorkbenchPinned = {
    getPinnedItems: function () { return pinnedItems.slice(); },
    reload: loadPinnedItems,
    unpin: function (menuId) {
      return unpinCurrent(menuId).then(function (wrap) {
        if (!wrap.res.ok || !wrap.data || !wrap.data.success) {
          return wrap;
        }
        return loadPinnedItems().then(function () {
          currentMatch = matchItemForPath(window.location.pathname);
          var btn = document.getElementById('pagePinToggle');
          if (btn && btn.style.display !== 'none') {
            updateTopbarPinState(btn);
          }
          syncPinnedMenuState();
          notifyPinnedChange();
          return wrap;
        });
      });
    },
    reorder: function (menuIds) {
      return reorderPinned(menuIds).then(function (wrap) {
        if (wrap.res.ok && wrap.data && wrap.data.success && Array.isArray(wrap.data.items)) {
          setPinnedItems(wrap.data.items);
          syncPinnedMenuState();
          notifyPinnedChange();
        }
        return wrap;
      });
    },
    resolvePinnedIcon: resolvePinnedIcon
  };

  function resolvePinnedIcon(item) {
    var raw = String((item && item.icon) || "").trim();
    if (raw && /\bfa-[a-z0-9-]+/.test(raw) && raw !== "fa-circle" && raw !== "fas fa-circle") {
      return /\b(fas|far|fal|fad|fab|fa-solid|fa-regular|fa-light)\b/.test(raw) ? raw : "fas " + raw;
    }
    return "";
  }
})();
