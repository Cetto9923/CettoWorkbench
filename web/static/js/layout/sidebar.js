/*
 * 文件: web/static/js/layout/sidebar.js
 * 模块: 布局
 * 职责: 侧边导航栏交互
 *   1) 双轨侧栏：一级窄轨切换二级面板
 *   2) 手风琴展开/折叠（同级互斥）
 *   3) 侧边栏整体收起 + localStorage 持久化
 *   4) 自动展开当前页面所在菜单路径
 *   5) 个人面板「常用」列表渲染与编辑（拖拽排序 / 移除）
 *   6) 从「常用」点击进入时，刷新后仍停留在常用面板并高亮对应项
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
  /* 从「常用」点击进入时置位，避免刷新后被服务端真实菜单分组抢走高亮 */
  var PINNED_NAV_KEY = "po_nav_via_pinned";

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

  function getNavViaPinned() {
    try {
      if (window.localStorage.getItem(PINNED_NAV_KEY) === "1") return true;
      return /(?:^|;\s*)po_nav_via_pinned=1(?:;|$)/.test(document.cookie);
    } catch (e) {
      return false;
    }
  }

  function setNavViaPinned(enabled) {
    try {
      if (enabled) {
        window.localStorage.setItem(PINNED_NAV_KEY, "1");
        document.cookie =
          "po_nav_via_pinned=1;path=/;max-age=31536000;SameSite=Lax";
      } else {
        window.localStorage.removeItem(PINNED_NAV_KEY);
        document.cookie = "po_nav_via_pinned=;path=/;max-age=0;SameSite=Lax";
      }
    } catch (e) {}
  }

  function getPinnedRailGroup() {
    var section = document.getElementById("pinnedSection");
    if (!section) return "";
    var panel = section.closest(".po-subnav-panel");
    return panel ? panel.getAttribute("data-subnav-panel") || "" : "";
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
        setNavViaPinned(false);
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
      var fromPinned = !!navLink.closest("#sidebarPinnedContainer, #pinnedSection");
      setNavViaPinned(fromPinned);
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
    var pinnedGroup = getPinnedRailGroup();
    var group = routeGroup || storedGroup;
    var updateRoute = !routeGroup;

    /* 从「常用」进入：优先停留在常用所在面板，并同步一级轨高亮 */
    if (getNavViaPinned() && pinnedGroup && storedGroup === pinnedGroup) {
      group = pinnedGroup;
      updateRoute = true;
    }

    if (!group) {
      var firstBtn = sidebar.querySelector(".po-rail-btn[data-rail-group]");
      if (firstBtn) group = firstBtn.getAttribute("data-rail-group") || "";
    }

    if (group) activateRailGroup(group, updateRoute);
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

  /* ── 个人面板「常用」列表（WorkbenchPinned：展示 / 编辑排序 / 移除） ── */
  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function pinnedIconClass(icon) {
    var raw = String(icon || "").trim();
    if (!raw) return "fas fa-circle";
    if (/\bfa[srlbd]?\b/.test(raw)) return raw;
    return "fas " + raw;
  }

  function pathIsActive(path, currentPath) {
    var p = String(path || "").trim();
    if (!p) return false;
    return currentPath === p || currentPath.indexOf(p + "/") === 0;
  }

  function initPinnedSection() {
    var container = document.getElementById("sidebarPinnedContainer");
    var manageBtn = document.getElementById("btnManagePinned");
    var editTip = document.getElementById("pinnedEditTip");
    if (!container || !window.WorkbenchPinned) return;

    var isEditMode = false;
    var draftItems = [];
    var saving = false;

    function syncDraftFromStore() {
      draftItems = (window.WorkbenchPinned.getPinnedItems() || []).slice();
    }

    function renderPinnedList() {
      if (!isEditMode) {
        syncDraftFromStore();
      }
      var currentPath = window.location.pathname;
      var html = "";
      draftItems.forEach(function (item) {
        var path = String((item && item.path) || "").trim();
        var menuId = String((item && item.menuId) || "").trim();
        if (!path || !menuId) return;
        var title = String((item && item.title) || path).trim();
        if (!isEditMode) {
          var active = pathIsActive(path, currentPath);
          html +=
            '<a href="' +
            escapeHtml(path) +
            '" class="nav-item' +
            (active ? " active" : "") +
            '" data-menu-id="' +
            escapeHtml(menuId) +
            '" title="' +
            escapeHtml(title) +
            '">';
          html += '<i class="nav-icon ' + escapeHtml(pinnedIconClass(item && item.icon)) + '"></i>';
          html += '<span class="nav-text">' + escapeHtml(title) + "</span>";
          html += "</a>";
        } else {
          html +=
            '<div class="nav-item is-editing-item" data-menu-id="' +
            escapeHtml(menuId) +
            '" draggable="true" title="按住拖拽排序">';
          html +=
            '<span class="pinned-drag-handle" title="按住拖拽排序"><i class="fas fa-grip-vertical"></i></span>';
          html += '<span class="nav-text">' + escapeHtml(title) + "</span>";
          html +=
            '<button type="button" class="nav-item-unpin js-unpin-btn" data-menu-id="' +
            escapeHtml(menuId) +
            '" title="移除此项">×</button>';
          html += "</div>";
        }
      });
      container.innerHTML = html;
      if (isEditMode) {
        bindEditModeEvents();
      }
    }

    function syncOrderFromDOM() {
      var byId = {};
      draftItems.forEach(function (item) {
        byId[String(item.menuId)] = item;
      });
      var next = [];
      container.querySelectorAll(".is-editing-item").forEach(function (el) {
        var id = el.getAttribute("data-menu-id");
        if (id && byId[id]) next.push(byId[id]);
      });
      draftItems = next;
    }

    function bindEditModeEvents() {
      container.querySelectorAll(".js-unpin-btn").forEach(function (btn) {
        btn.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          if (saving) return;
          var menuId = btn.getAttribute("data-menu-id");
          if (!menuId || typeof window.WorkbenchPinned.unpin !== "function") return;
          saving = true;
          window.WorkbenchPinned.unpin(menuId)
            .then(function (wrap) {
              if (!wrap.res.ok || !wrap.data || !wrap.data.success) {
                var msg = (wrap.data && wrap.data.message) || "移除失败";
                if (typeof window.showToast === "function") {
                  window.showToast(msg, "error");
                }
                return;
              }
              draftItems = draftItems.filter(function (it) {
                return String(it.menuId) !== String(menuId);
              });
              renderPinnedList();
            })
            .catch(function () {
              if (typeof window.showToast === "function") {
                window.showToast("移除失败，请稍后重试", "error");
              }
            })
            .then(function () {
              saving = false;
            });
        });
      });

      var draggedEl = null;
      container.querySelectorAll(".is-editing-item").forEach(function (item) {
        item.addEventListener("dragstart", function (e) {
          draggedEl = item;
          e.dataTransfer.effectAllowed = "move";
          e.dataTransfer.setData("text/plain", item.getAttribute("data-menu-id") || "");
          setTimeout(function () {
            if (item) item.classList.add("is-dragging");
          }, 0);
        });
        item.addEventListener("dragover", function (e) {
          e.preventDefault();
          e.dataTransfer.dropEffect = "move";
          if (!draggedEl || draggedEl === item) return;
          var rect = item.getBoundingClientRect();
          var midY = rect.top + rect.height / 2;
          if (e.clientY < midY) {
            if (item.previousElementSibling !== draggedEl) {
              container.insertBefore(draggedEl, item);
            }
          } else if (item.nextElementSibling !== draggedEl) {
            container.insertBefore(draggedEl, item.nextSibling);
          }
        });
        item.addEventListener("drop", function (e) {
          e.preventDefault();
          syncOrderFromDOM();
        });
        item.addEventListener("dragend", function () {
          if (draggedEl) {
            draggedEl.classList.remove("is-dragging");
            draggedEl = null;
          }
          syncOrderFromDOM();
        });
      });
    }

    function exitEditMode(opts) {
      var silent = opts && opts.silent;
      isEditMode = false;
      container.classList.remove("is-editing");
      if (editTip) editTip.hidden = true;
      if (manageBtn) {
        manageBtn.classList.remove("is-editing");
        manageBtn.innerHTML = "<span>编辑</span>";
        manageBtn.title = "自定义与排序常用页面";
      }
      syncDraftFromStore();
      renderPinnedList();
      if (!silent && typeof window.showToast === "function") {
        window.showToast("已保存自选常用排序", "success");
      }
    }

    function enterEditMode() {
      isEditMode = true;
      syncDraftFromStore();
      container.classList.add("is-editing");
      if (editTip) editTip.hidden = false;
      if (manageBtn) {
        manageBtn.classList.add("is-editing");
        manageBtn.innerHTML =
          '<i class="fas fa-check" aria-hidden="true"></i><span>完成</span>';
        manageBtn.title = "完成排序与修改";
      }
      renderPinnedList();
    }

    function finishEditMode() {
      if (saving) return;
      syncOrderFromDOM();
      var menuIds = draftItems
        .map(function (it) {
          return Number(it.menuId);
        })
        .filter(function (id) {
          return id > 0;
        });
      if (typeof window.WorkbenchPinned.reorder !== "function") {
        exitEditMode({ silent: true });
        return;
      }
      saving = true;
      window.WorkbenchPinned.reorder(menuIds)
        .then(function (wrap) {
          if (!wrap.res.ok || !wrap.data || !wrap.data.success) {
            var msg = (wrap.data && wrap.data.message) || "保存排序失败";
            if (typeof window.showToast === "function") {
              window.showToast(msg, "error");
            }
            return;
          }
          exitEditMode();
        })
        .catch(function () {
          if (typeof window.showToast === "function") {
            window.showToast("保存排序失败，请稍后重试", "error");
          }
        })
        .then(function () {
          saving = false;
        });
    }

    if (manageBtn) {
      manageBtn.addEventListener("click", function (e) {
        e.preventDefault();
        if (isEditMode) {
          finishEditMode();
        } else {
          enterEditMode();
        }
      });
    }

    renderPinnedList();
    window.addEventListener("workbench:pinned-change", function () {
      if (!isEditMode) {
        renderPinnedList();
      }
    });
    if (typeof window.WorkbenchPinned.reload === "function") {
      window.WorkbenchPinned.reload().then(function () {
        if (!isEditMode) renderPinnedList();
      });
    }
  }

  initPinnedSection();
})();
