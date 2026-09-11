/*
 * 文件: web/static/js/layout/bottomtabbar.js
 * 模块: 布局
 * 职责: 底部工作页签（访问累积 + localStorage 持久化 + 关闭）
 */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.bottomTabs.v1";
  var bar = document.getElementById("bottom-tabbar");
  if (!bar) return;

  function currentPath() {
    return window.location.pathname || "/";
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
      "#topnav-primary a.topnav-primary-item--leaf.is-active",
    ];
    var i;
    for (i = 0; i < selectors.length; i++) {
      var active = document.querySelector(selectors[i]);
      if (active && pathFromEl(active) === path) {
        return {
          path: path,
          title: titleFromEl(active),
          icon: iconClassFromEl(active),
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
          icon: iconClassFromEl(candidates[i]),
        };
      }
    }
    if (path === "/profile") {
      return null;
    }
    return null;
  }

  function upsertTab(tabs, entry) {
    var idx = -1;
    var i;
    for (i = 0; i < tabs.length; i++) {
      if (tabs[i].path === entry.path) {
        idx = i;
        break;
      }
    }
    if (idx >= 0) {
      tabs[idx].title = entry.title || tabs[idx].title;
      if (entry.icon) tabs[idx].icon = entry.icon;
      return tabs;
    }
    tabs.push({
      path: entry.path,
      title: entry.title || entry.path,
      icon: entry.icon || "",
    });
    return tabs;
  }

  function render(tabs) {
    var path = currentPath();
    bar.innerHTML = "";
    tabs.forEach(function (tab) {
      var a = document.createElement("a");
      a.className = "bottom-tab" + (tab.path === path ? " active" : "");
      a.href = tab.path;
      a.title = tab.title;

      if (tab.icon) {
        var icon = document.createElement("i");
        icon.className = tab.icon;
        icon.setAttribute("aria-hidden", "true");
        a.appendChild(icon);
      }

      var label = document.createElement("span");
      label.textContent = tab.title;
      a.appendChild(label);

      var closeBtn = document.createElement("button");
      closeBtn.type = "button";
      closeBtn.className = "bottom-tab-close";
      closeBtn.setAttribute("aria-label", "关闭 " + tab.title);
      closeBtn.textContent = "×";
      closeBtn.addEventListener("click", function (e) {
        e.preventDefault();
        e.stopPropagation();
        closeTab(tab.path);
      });
      a.appendChild(closeBtn);

      bar.appendChild(a);
    });
  }

  function closeTab(path) {
    var tabs = readTabs();
    if (tabs.length <= 1) return;

    var idx = -1;
    var i;
    for (i = 0; i < tabs.length; i++) {
      if (tabs[i].path === path) {
        idx = i;
        break;
      }
    }
    if (idx < 0) return;

    var wasCurrent = path === currentPath();
    var nextPath = null;
    if (wasCurrent) {
      var neighbor = tabs[idx - 1] || tabs[idx + 1];
      nextPath = neighbor ? neighbor.path : null;
    }

    tabs.splice(idx, 1);
    writeTabs(tabs);

    if (wasCurrent && nextPath) {
      window.location.href = nextPath;
      return;
    }
    render(tabs);
  }

  function init() {
    var tabs = readTabs();
    var path = currentPath();
    var entry = findMenuEntry(path);
    if (entry && entry.title) {
      tabs = upsertTab(tabs, entry);
      writeTabs(tabs);
    }
    render(tabs);
  }

  init();
})();
