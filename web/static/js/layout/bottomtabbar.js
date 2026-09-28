/*
 * 文件: web/static/js/layout/bottomtabbar.js
 * 模块: 布局
 * 职责: 底部工作区页签（规范路径去重 + 状态/URL记忆 + 滚动恢复 + 关闭切换 + /home常驻）
 */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.bottomTabs.v1";
  var SCROLL_PREFIX = "workbench.scroll.";
  var bar = document.getElementById("workbench-tabbar") || document.getElementById("bottom-tabbar");
  if (!bar) return;

  var actionsBar = document.getElementById("workbench-tab-actions");

  function currentPath() {
    return window.location.pathname || "/";
  }

  function currentUrl() {
    return (window.location.pathname || "/") + (window.location.search || "") + (window.location.hash || "");
  }

  function readTabs() {
    try {
      var raw = window.localStorage.getItem(STORAGE_KEY);
      if (!raw) return [];
      var parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) return [];
      return parsed.filter(function (t) {
        return t && typeof t.path === "string" && t.path && typeof t.title === "string";
      });
    } catch (e) {
      return [];
    }
  }

  function writeTabs(tabs) {
    try {
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(tabs));
    } catch (e) {
      /* ignore quota / private mode */
    }
  }

  function getMainScroller() {
    return document.querySelector(".main") || document.scrollingElement || document.documentElement;
  }

  function saveScrollPosition() {
    var path = currentPath();
    var scroller = getMainScroller();
    if (scroller && path) {
      try {
        window.sessionStorage.setItem(SCROLL_PREFIX + path, String(scroller.scrollTop || 0));
      } catch (e) {}
    }
  }

  function restoreScrollPosition() {
    var path = currentPath();
    try {
      var saved = window.sessionStorage.getItem(SCROLL_PREFIX + path);
      if (saved) {
        var top = parseInt(saved, 10);
        if (!isNaN(top) && top > 0) {
          setTimeout(function () {
            var scroller = getMainScroller();
            if (scroller) scroller.scrollTop = top;
          }, 80);
        }
      }
    } catch (e) {}
  }

  function iconClassFromEl(el) {
    if (!el) return "";
    var icon = el.querySelector("i.nav-icon, i.topnav-primary-icon, i.fas, i.bi, i[class*='fa-']");
    if (!icon) return "";
    return (icon.className || "").trim();
  }

  function titleFromEl(el) {
    if (!el) return "";
    var textEl = el.querySelector(".nav-text, .topnav-primary-label");
    if (textEl) return (textEl.textContent || "").trim();
    var title = (el.getAttribute("title") || "").trim();
    if (title) return title;
    return (el.textContent || "").replace(/\s+/g, " ").trim();
  }

  function pathFromEl(el) {
    if (!el) return "";
    var dataPath = (el.getAttribute("data-menu-path") || "").trim();
    if (dataPath) return dataPath;
    var href = (el.getAttribute("href") || "").trim();
    if (!href || href === "#") return "";
    try {
      return new URL(href, window.location.origin).pathname;
    } catch (e) {
      return href.split("?")[0].split("#")[0];
    }
  }

  function findMenuEntry(path) {
    var selectors = [
      "#sidebar a.nav-item.active",
      "#topnav-secondary a.topnav-subitem.is-active",
      "#topnav-primary a.topnav-primary-item--leaf.is-active"
    ];
    var i;
    for (i = 0; i < selectors.length; i++) {
      var active = document.querySelector(selectors[i]);
      if (active && pathFromEl(active) === path) {
        return {
          path: path,
          title: titleFromEl(active),
          icon: iconClassFromEl(active)
        };
      }
    }

    var candidates = document.querySelectorAll(
      "#sidebar a.nav-item[href], #topnav-secondary a.topnav-subitem[href], #topnav-primary a.topnav-primary-item--leaf[href]"
    );
    for (i = 0; i < candidates.length; i++) {
      if (pathFromEl(candidates[i]) === path) {
        return {
          path: path,
          title: titleFromEl(candidates[i]),
          icon: iconClassFromEl(candidates[i])
        };
      }
    }
    if (path === "/home") {
      return { path: "/home", title: "首页", icon: "fas fa-home" };
    }
    return null;
  }

  function upsertTab(tabs, entry) {
    var path = entry.path;
    var url = currentUrl();
    var idx = -1;
    for (var i = 0; i < tabs.length; i++) {
      if (tabs[i].path === path) {
        idx = i;
        break;
      }
    }
    if (idx >= 0) {
      tabs[idx].title = entry.title || tabs[idx].title;
      if (entry.icon) tabs[idx].icon = entry.icon;
      tabs[idx].url = url;
      return tabs;
    }
    tabs.push({
      path: path,
      url: url,
      title: entry.title || entry.path,
      icon: entry.icon || "",
      pinned: path === "/home"
    });
    return tabs;
  }

  function updateCurrentTabUrl() {
    var path = currentPath();
    var url = currentUrl();
    var tabs = readTabs();
    var changed = false;
    for (var i = 0; i < tabs.length; i++) {
      if (tabs[i].path === path) {
        if (tabs[i].url !== url) {
          tabs[i].url = url;
          changed = true;
        }
        break;
      }
    }
    if (changed) {
      writeTabs(tabs);
      render(tabs);
    }
  }

  function syncCurrentTab(newPath, newTitle) {
    if (!newPath) return;
    var tabs = readTabs();
    var changed = false;
    for (var i = 0; i < tabs.length; i++) {
      if (tabs[i].path === newPath || (tabs[i].path.indexOf("/board/") === 0 && newPath.indexOf("/board/") === 0)) {
        tabs[i].path = newPath;
        tabs[i].url = newPath;
        if (newTitle) tabs[i].title = newTitle;
        changed = true;
        break;
      }
    }
    if (!changed) {
      tabs = upsertTab(tabs, { path: newPath, title: newTitle || newPath, icon: "fas fa-table-cells-large" });
      changed = true;
    }
    if (changed) {
      writeTabs(tabs);
      render(tabs);
    }
  }

  function render(tabs) {
    var path = currentPath();
    bar.innerHTML = "";
    var activeEl = null;

    tabs.forEach(function (tab) {
      var isPinned = tab.pinned || tab.path === "/home";
      var isActive = tab.path === path;
      var a = document.createElement("a");
      a.className = "workbench-tab bottom-tab" + (isActive ? " active" : "");
      a.href = tab.url || tab.path;
      a.title = tab.title;
      a.dataset.tabPath = tab.path;

      if (isPinned) {
        var pinIcon = document.createElement("i");
        pinIcon.className = "fas fa-thumbtack workbench-tab-pin bottom-tab-pin";
        pinIcon.setAttribute("aria-hidden", "true");
        a.appendChild(pinIcon);
      } else if (tab.icon) {
        var icon = document.createElement("i");
        icon.className = tab.icon;
        icon.setAttribute("aria-hidden", "true");
        a.appendChild(icon);
      }

      var label = document.createElement("span");
      label.textContent = tab.title;
      a.appendChild(label);

      a.addEventListener("click", function () {
        saveScrollPosition();
      });

      if (!isPinned) {
        var closeBtn = document.createElement("button");
        closeBtn.type = "button";
        closeBtn.className = "workbench-tab-close bottom-tab-close";
        closeBtn.setAttribute("aria-label", "关闭 " + tab.title);
        closeBtn.textContent = "×";
        closeBtn.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          closeTab(tab.path);
        });
        a.appendChild(closeBtn);
      }

      bar.appendChild(a);
      if (isActive) activeEl = a;
    });

    if (activeEl && typeof activeEl.scrollIntoView === "function") {
      try {
        activeEl.scrollIntoView({ inline: "nearest", block: "nearest" });
      } catch (e) {}
    }

    renderActions(tabs);
  }

  function renderActions(tabs) {
    var target = actionsBar || bar;
    if (actionsBar) actionsBar.innerHTML = "";

    var closableCount = tabs.filter(function (t) {
      return t.path !== "/home" && !t.pinned;
    }).length;

    if (closableCount > 0 && tabs.length > 2) {
      var closeOthers = document.createElement("button");
      closeOthers.type = "button";
      closeOthers.className = "workbench-tab-action bottom-tab-action";
      closeOthers.title = "关闭其他未固定页签";
      closeOthers.innerHTML = '<i class="fas fa-rectangle-xmark" aria-hidden="true"></i> <span>关闭其他页签</span>';
      closeOthers.addEventListener("click", function (e) {
        e.preventDefault();
        var current = currentPath();
        var kept = tabs.filter(function (t) {
          return t.path === "/home" || t.pinned || t.path === current;
        });
        writeTabs(kept);
        render(kept);
      });
      target.appendChild(closeOthers);
    }
  }

  function closeTab(path) {
    if (path === "/home") return; // 首页常驻，禁止关闭
    var tabs = readTabs();
    if (tabs.length <= 1) return;

    var idx = -1;
    for (var i = 0; i < tabs.length; i++) {
      if (tabs[i].path === path) {
        idx = i;
        break;
      }
    }
    if (idx < 0) return;

    var wasCurrent = path === currentPath();
    var nextTab = null;
    if (wasCurrent) {
      nextTab = tabs[idx - 1] || tabs[idx + 1];
    }

    tabs.splice(idx, 1);
    writeTabs(tabs);

    if (wasCurrent && nextTab) {
      saveScrollPosition();
      window.location.href = nextTab.url || nextTab.path;
      return;
    }
    render(tabs);
  }

  // 挂钩 history API 动态保存页面内筛选与分页 URL
  var origReplace = window.history && window.history.replaceState;
  if (typeof origReplace === "function") {
    window.history.replaceState = function () {
      var res = origReplace.apply(this, arguments);
      updateCurrentTabUrl();
      return res;
    };
  }

  var origPush = window.history && window.history.pushState;
  if (typeof origPush === "function") {
    window.history.pushState = function () {
      var res = origPush.apply(this, arguments);
      updateCurrentTabUrl();
      return res;
    };
  }

  window.addEventListener("popstate", function () {
    updateCurrentTabUrl();
    restoreScrollPosition();
  });

  window.addEventListener("beforeunload", saveScrollPosition);

  function init() {
    var tabs = readTabs();
    var path = currentPath();
    var entry = findMenuEntry(path);
    if (entry && entry.title) {
      tabs = upsertTab(tabs, entry);
      writeTabs(tabs);
    }
    render(tabs);
    restoreScrollPosition();
  }

  // 暴露给单元测试与回归脚本
  window.WorkbenchTabbar = {
    readTabs: readTabs,
    writeTabs: writeTabs,
    upsertTab: upsertTab,
    closeTab: closeTab,
    updateCurrentTabUrl: updateCurrentTabUrl,
    syncCurrentTab: syncCurrentTab,
    currentPath: currentPath,
    currentUrl: currentUrl
  };

  init();
})();
