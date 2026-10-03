/**
 * 文件: web/static/js/layout/pinned-pages.js
 * 模块: 工作台
 * 职责: 管理用户自定义固定常用功能，支持 localStorage 隔离持久化、顶栏/侧栏即时联动与自定义弹窗。
 */
(function () {
  'use strict';

  var STORAGE_KEY_PREFIX = 'workbench.pinned_pages.';
  var DEFAULT_PINNED = ['schedule', 'query', 'board_demand', 'agileteam', 'issues_risk'];

  var PINNABLE_PAGES = {
    schedule: {
      key: 'schedule',
      title: '需求排期',
      subtitle: '迭代规划与容量分配',
      meaning: '产品经理进行迭代容量规划、按周排期与交付承诺窗口确认。',
      icon: 'fas fa-calendar-check',
      path: '/schedule',
      badge: '需求',
      group: 'plan',
      groupTitle: '规划',
      metric: '排期规划'
    },
    query: {
      key: 'query',
      title: '需求查询',
      subtitle: '全流程跨项目检索',
      meaning: '跨团队、跨敏捷小组的全流程需求状态多维检索与导出。',
      icon: 'fas fa-magnifying-glass',
      path: '/query',
      group: 'plan',
      groupTitle: '规划',
      metric: '全景检索'
    },
    board_demand: {
      key: 'board_demand',
      title: '工作看板',
      subtitle: '敏捷泳道与状态流转',
      meaning: '敏捷卡片协同泳道，可视化跟踪需求、任务与缺陷流转进度。',
      icon: 'fas fa-table-columns',
      path: '/board/demand',
      group: 'collab',
      groupTitle: '协作',
      metric: '流转看板'
    },
    agileteam: {
      key: 'agileteam',
      title: '敏捷小组',
      subtitle: '团队人员与负荷分配',
      meaning: '敏捷小组成员配置、研发工时负荷与团队产能水位管理。',
      icon: 'fas fa-people-group',
      path: '/agileteam',
      group: 'collab',
      groupTitle: '协作',
      metric: '团队产能'
    },
    metrics_manage: {
      key: 'metrics_manage',
      title: '度量大盘',
      subtitle: '交付效能与质量指标',
      meaning: '研发吞吐率、交付周期与质效达标全景多维大盘。',
      icon: 'fas fa-gauge-high',
      path: '/metrics/manage',
      group: 'gov',
      groupTitle: '治理',
      metric: '效能大盘'
    },
    metrics_radar: {
      key: 'metrics_radar',
      title: '度量雷达',
      subtitle: '敏捷成熟度能力评估',
      meaning: '敏捷小组与产品线质量成熟度、工程规范雷达评估。',
      icon: 'fas fa-compass-drafting',
      path: '/metrics/radar',
      group: 'gov',
      groupTitle: '治理',
      metric: '能力雷达'
    },
    issues_risk: {
      key: 'issues_risk',
      title: '问题风险',
      subtitle: '滞留与超期预警排查',
      meaning: '超期未闭环、阶段滞留与阻塞依赖风险全景监控。',
      icon: 'fas fa-triangle-exclamation',
      path: '/issues/risk',
      group: 'gov',
      groupTitle: '治理',
      metric: '风险预警'
    }
  };

  PINNABLE_PAGES.board_task = Object.assign({}, PINNABLE_PAGES.board_demand, {
    key: 'board_task', title: '任务看板', path: '/board/task'
  });

  document.querySelectorAll('.nav-item[href]').forEach(function (link) {
    var path = link.getAttribute('href');
    if (!path || path.charAt(0) !== '/' || path === '/home' || keyFromPathOrKey(path)) return;
    var key = 'page:' + path;
    var title = (link.querySelector('.nav-text') || link).textContent.trim();
    var icon = link.querySelector('.nav-icon');
    PINNABLE_PAGES[key] = { key: key, title: title, path: path,
      icon: icon ? icon.className : 'fas fa-file', group: 'workbench', groupTitle: '工作台' };
  });

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
