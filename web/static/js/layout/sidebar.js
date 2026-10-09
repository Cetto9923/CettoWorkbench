/*
 * 文件: web/static/js/layout/sidebar.js
 * 模块: 布局
 * 职责: 侧边导航栏交互
 *   1) 双轨侧栏：一级窄轨切换二级面板
 *   2) 手风琴展开/折叠（同级互斥）
 *   3) 侧边栏整体收起 + localStorage 持久化
 *   4) 自动展开当前页面所在菜单路径
 */
(function () {
  "use strict";

  var sidebar = document.getElementById("sidebar");
  if (!sidebar) return;
  var isDualSidebar = sidebar.classList.contains("po-dual-sidebar");

  /* ── 工具函数 ── */

  function getSubmenu(toggle) {
    var header = toggle.closest(".nav-item-header");
    if (!header) return null;
    var next = header.nextElementSibling;
    return next && next.classList.contains("js-nav-submenu") ? next : null;
  }

  function openSubmenu(sub) {
    sub.classList.add("open");
    sub.style.maxHeight = sub.scrollHeight + "px";
    syncToggle(sub, true);
  }

  function closeSubmenu(sub) {
    sub.style.maxHeight = sub.scrollHeight + "px";
    sub.offsetHeight; // 强制 reflow
    sub.style.maxHeight = "0";
    sub.classList.remove("open");
    syncToggle(sub, false);
  }

  function syncToggle(sub, expanded) {
    var id = sub.getAttribute("data-parent-id");
    if (!id) return;
    var btn = sidebar.querySelector('.js-nav-toggle[data-menu-id="' + id + '"]');
    if (btn) btn.setAttribute("aria-expanded", expanded ? "true" : "false");
  }

  function closeSiblings(currentSub) {
    var container = currentSub.parentElement;
    if (!container) return;
    if (container.classList.contains("nav-group-collapsible")) {
      container = container.parentElement;
    }
    if (!container) return;
    var openSubs = container.querySelectorAll(
      ":scope > .nav-group > .js-nav-submenu.open," +
        " :scope > .nav-group-collapsible > .js-nav-submenu.open"
    );
    openSubs.forEach(function (s) {
      if (s !== currentSub) closeSubmenu(s);
    });
  }

  var RAIL_STORAGE_KEY = "po_active_rail";
  function getStoredRailGroup() {
    try {
      var match = document.cookie.match(/(?:^|;\s*)po_active_rail=([^;]+)/);
      if (match && match[1]) return decodeURIComponent(match[1]);
      return window.localStorage.getItem(RAIL_STORAGE_KEY) || "";
    } catch (e) {
      return "";
    }
  }

  function setStoredRailGroup(group) {
    if (!group) return;
    try {
      window.localStorage.setItem(RAIL_STORAGE_KEY, group);
      document.cookie =
        "po_active_rail=" + encodeURIComponent(group) + ";path=/;max-age=31536000;SameSite=Lax";
    } catch (e) {}
  }

  function activateRailGroup(group, updateRoute) {
    if (!isDualSidebar || !group) return;
    sidebar.querySelectorAll(".po-rail-btn").forEach(function (btn) {
      var isCurrent = btn.getAttribute("data-rail-group") === group;
      btn.classList.toggle("is-drawer-open", isCurrent);
      btn.setAttribute("aria-expanded", isCurrent ? "true" : "false");
      if (updateRoute) {
        btn.classList.toggle("is-active-route", isCurrent);
      }
    });
    sidebar.querySelectorAll(".po-subnav-panel").forEach(function (panel) {
      panel.classList.toggle("is-active", panel.getAttribute("data-subnav-panel") === group);
    });
    if (updateRoute) {
      setStoredRailGroup(group);
    }
  }

  /* ── 侧栏点击代理 ── */

  sidebar.addEventListener("click", function (e) {
    var railBtn = e.target.closest(".po-rail-btn");
    if (railBtn && sidebar.contains(railBtn)) {
      var group = railBtn.getAttribute("data-rail-group");
      if (group) {
        activateRailGroup(group, true);
        if (document.body.classList.contains("sidebar-collapsed")) {
          applyCollapsed(false);
          try {
            window.localStorage.setItem(STORAGE_KEY, "0");
          } catch (err) {}
        }
      }
      return;
    }

    var navLink = e.target.closest(".po-subnav a.nav-item[href]");
    if (navLink && sidebar.contains(navLink)) {
      var panel = navLink.closest(".po-subnav-panel");
      if (panel) {
        var g = panel.getAttribute("data-subnav-panel");
        if (g) setStoredRailGroup(g);
      }
    }

    var toggle = e.target.closest(".js-nav-toggle");
    if (!toggle || !sidebar.contains(toggle)) return;
    e.preventDefault();

    var sub = getSubmenu(toggle);
    if (!sub) return;

    if (sub.classList.contains("open")) {
      closeSubmenu(sub);
    } else {
      closeSiblings(sub);
      openSubmenu(sub);
    }
  });

  sidebar.addEventListener("transitionend", function (e) {
    var t = e.target;
    if (t.classList && t.classList.contains("js-nav-submenu") && t.classList.contains("open")) {
      t.style.maxHeight = "none";
    }
  });

  /* ── 整体收起 ── */

  var collapseBtn = document.getElementById("sidebarToggle");
  var STORAGE_KEY = "gofw.sidebar.collapsed";

  function applyCollapsed(collapsed) {
    document.body.classList.toggle("sidebar-collapsed", collapsed);
    if (!collapseBtn) return;
    collapseBtn.setAttribute("aria-expanded", collapsed ? "false" : "true");
    var label = collapsed ? "展开子菜单" : "收起子菜单";
    collapseBtn.setAttribute("title", label);
    collapseBtn.setAttribute("aria-label", label);
    var text = collapseBtn.querySelector(".sidebar-collapse-text, .nav-text.sidebar-collapse-text");
    if (text) text.textContent = collapsed ? "展开菜单" : "收起菜单";
  }

  var initialCollapsed = false;
  try {
    initialCollapsed = window.localStorage.getItem(STORAGE_KEY) === "1";
  } catch (err) {
    initialCollapsed = false;
  }
  applyCollapsed(initialCollapsed);

  if (collapseBtn) {
    collapseBtn.addEventListener("click", function () {
      var collapsed = !document.body.classList.contains("sidebar-collapsed");
      applyCollapsed(collapsed);
      try {
        window.localStorage.setItem(STORAGE_KEY, collapsed ? "1" : "0");
      } catch (err) {}
    });
  }

  function syncActiveRoute() {
    if (!isDualSidebar) return;
    var storedGroup = getStoredRailGroup();
    var activePanel = sidebar.querySelector(".po-subnav-panel.is-active");
    var routeGroup = activePanel ? activePanel.getAttribute("data-subnav-panel") : "";
    var group = routeGroup || storedGroup;

    if (!group) {
      var firstBtn = sidebar.querySelector(".po-rail-btn[data-rail-group]");
      if (firstBtn) group = firstBtn.getAttribute("data-rail-group") || "";
    }

    if (group) activateRailGroup(group, !routeGroup);
  }

  if (isDualSidebar) syncActiveRoute();

  window.addEventListener("pageshow", function () {
    syncActiveRoute();
  });

  /* ── 自动展开当前活跃项的祖先路径 ── */
  var activeNodes = sidebar.querySelectorAll(".nav-item.active");
  var activeItem = activeNodes.length === 0 ? null : activeNodes[activeNodes.length - 1];
  if (activeItem) {
    var node = activeItem.parentElement;
    while (node && node !== sidebar) {
      if (node.classList.contains("js-nav-submenu")) {
        node.classList.add("open");
        node.style.maxHeight = "none";
        var parentId = node.getAttribute("data-parent-id");
        var btn = parentId
          ? sidebar.querySelector('.js-nav-toggle[data-menu-id="' + parentId + '"]')
          : null;
        if (btn) {
          btn.setAttribute("aria-expanded", "true");
          var header = btn.closest(".nav-item-header");
          if (header) header.classList.add("has-active-child");
        }
      }
      node = node.parentElement;
    }
  }
})();
