/*
 * 文件: web/static/js/sidebar.js
 * 模块: 布局
 * 职责: 侧边导航栏交互
 *   1) 手风琴展开/折叠（同级互斥）
 *   2) 侧边栏整体收起 + localStorage 持久化
 *   3) 自动展开当前页面所在菜单路径
 */
(function () {
  "use strict";

  var sidebar = document.getElementById("sidebar");
  if (!sidebar) return;

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

  /* ── 手风琴点击 ── */

  sidebar.addEventListener("click", function (e) {
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
    var text =
      collapseBtn.querySelector(".sidebar-collapse-text") ||
      collapseBtn.querySelector(".nav-text.sidebar-collapse-text");
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
      } catch (err) {
        // ignore
      }
    });
  }

  /* ── 自动展开当前活跃项的祖先路径 ── */
  /* 父级链接仅在本页 key 与该项一致时加 .active；若仍有多个 .active，取最后一个以取最深命中项。 */

  var activeNodes = sidebar.querySelectorAll(".nav-item.active");
  var activeItem =
    activeNodes.length === 0 ? null : activeNodes[activeNodes.length - 1];
  if (!activeItem) return;

  var node = activeItem.parentElement;
  while (node && node !== sidebar) {
    if (node.classList.contains("js-nav-submenu")) {
      node.classList.add("open");
      node.style.maxHeight = "none";
      var parentId = node.getAttribute("data-parent-id");
      if (parentId) {
        var btn = sidebar.querySelector('.js-nav-toggle[data-menu-id="' + parentId + '"]');
        if (btn) {
          btn.setAttribute("aria-expanded", "true");
          var header = btn.closest(".nav-item-header");
          if (header) header.classList.add("has-active-child");
        }
      }
    }
    node = node.parentElement;
  }
})();
