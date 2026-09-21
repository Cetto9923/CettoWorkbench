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
      document.cookie = "po_active_rail=" + encodeURIComponent(group) + ";path=/;max-age=31536000;SameSite=Lax";
    } catch (e) {}
  }

  /* ── 侧栏点击代理 ── */

  sidebar.addEventListener("click", function (e) {
    var railBtn = e.target.closest(".po-rail-btn");
    if (railBtn && sidebar.contains(railBtn)) {
      var group = railBtn.getAttribute("data-rail-group");
      if (group) {
        activateRailGroup(group, true);
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

  function applyCollapsed(collapsed) {
    document.body.classList.toggle("sidebar-collapsed", collapsed);
    if (!collapseBtn) return;
    collapseBtn.setAttribute("aria-expanded", collapsed ? "false" : "true");
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

  function syncActiveRoute(path) {
    if (!isDualSidebar) return;
    path = path || window.location.pathname || "/";
    var storedGroup = getStoredRailGroup();
    var isPinned = window.WorkbenchPinned && typeof window.WorkbenchPinned.isPinned === "function" && window.WorkbenchPinned.isPinned(path);
    var isPersonal = path === "/home" || path === "/todos" || path === "/done" || path === "/notice" || path === "/follow";

    var group = storedGroup;
    if (group === "personal" && !isPersonal && !isPinned) group = "";
    if (!group) {
      if (isPersonal || isPinned) {
        group = "personal";
      } else {
        var matchingLink = null;
        sidebar.querySelectorAll(".po-subnav a.nav-item[href]").forEach(function (a) {
          if (a.closest("#sidebarPinnedContainer, .po-pinned-list")) return;
          var href = a.getAttribute("href");
          if (href && (href === path || path.indexOf(href) === 0)) {
            if (!matchingLink || href.length > matchingLink.getAttribute("href").length) matchingLink = a;
          }
        });
        if (matchingLink) {
          var p = matchingLink.closest(".po-subnav-panel");
          if (p) group = p.getAttribute("data-subnav-panel");
        }
      }
    }

    if (group) activateRailGroup(group, true);

    sidebar.querySelectorAll(".po-subnav a.nav-item").forEach(function (a) {
      var href = a.getAttribute("href");
      var isMatch = href && (href === path || path.indexOf(href) === 0);
      var inPinned = !!a.closest("#sidebarPinnedContainer, .po-pinned-list");
      if (group === "personal") {
        a.classList.toggle("active", (inPinned || !!a.closest('.po-subnav-panel[data-subnav-panel="personal"]')) && isMatch);
      } else {
        a.classList.toggle("active", !inPinned && !!a.closest('.po-subnav-panel[data-subnav-panel="' + group + '"]') && isMatch);
      }
    });
  }

  if (isDualSidebar) syncActiveRoute();

  window.addEventListener("popstate", function () { syncActiveRoute(); });
  window.addEventListener("pageshow", function () { syncActiveRoute(); });

  var activeNodes = sidebar.querySelectorAll(".nav-item.active");
  var activeItem = activeNodes.length === 0 ? null : activeNodes[activeNodes.length - 1];
  if (activeItem) {
    var node = activeItem.parentElement;
    while (node && node !== sidebar) {
      if (node.classList.contains("js-nav-submenu")) {
        node.classList.add("open");
        node.style.maxHeight = "none";
        var parentId = node.getAttribute("data-parent-id");
        var btn = parentId ? sidebar.querySelector('.js-nav-toggle[data-menu-id="' + parentId + '"]') : null;
        if (btn) {
          btn.setAttribute("aria-expanded", "true");
          var header = btn.closest(".nav-item-header");
          if (header) header.classList.add("has-active-child");
        }
      }
      node = node.parentElement;
    }
  }


  /* ── 我的工作台自选常用功能逻辑 ── */
  function initPinnedWorkbench() {
    var container = document.getElementById("sidebarPinnedContainer");
    var modal = document.getElementById("workbenchPinnedModal");
    var openBtn = document.getElementById("btnOpenPinnedModal");
    var closeBtn = document.getElementById("btnClosePinnedModal");
    var cancelBtn = document.getElementById("btnCancelPinnedModal");
    var saveBtn = document.getElementById("btnSavePinnedModal");
    var tipEl = document.getElementById("pinnedModalCountTip");
    var manageBtn = document.getElementById("btnManagePinned");
    var editTip = document.getElementById("pinnedEditTip");

    if (!window.WorkbenchPinned) return;

    var isEditMode = false;
    var currentOrder = window.WorkbenchPinned.getPinnedKeys().slice();

    function renderPinnedList() {
      if (!container) return;
      var keys = window.WorkbenchPinned.getPinnedKeys();
      if (!isEditMode) {
        currentOrder = keys.slice();
      }
      if (openBtn) openBtn.hidden = !isEditMode && currentOrder.length > 0;

      if (currentOrder.length === 0) {
        container.innerHTML = '';
        return;
      }

      var currentPath = window.location.pathname;
      var html = "";
      currentOrder.forEach(function (key) {
        var meta = window.WorkbenchPinned.PINNABLE_PAGES[key];
        if (!meta) return;
        var isActive = currentPath === meta.path;
        var colorClass = "icon-" + (meta.group || "plan");

        if (!isEditMode) {
          // 常规浏览模式：纯净标准链接，绝对不渲染删除按钮
          html += '<a href="' + meta.path + '" class="nav-item' + (isActive ? ' active' : '') + '" data-pinned-key="' + key + '" title="' + meta.title + '：' + (meta.meaning || meta.subtitle || '') + '">';
          html += '<i class="nav-icon ' + meta.icon + ' ' + colorClass + '"></i>';
          html += '<span class="nav-text">' + meta.title + '</span>';
          html += '</a>';
        } else {
          // 编辑管理模式：支持拖拽排序，展示抓手与删除按钮
          html += '<div class="nav-item is-editing-item" data-pinned-key="' + key + '" draggable="true" title="按住拖拽排序">';
          html += '<span class="pinned-drag-handle" title="按住拖拽排序"><i class="fas fa-grip-vertical"></i></span>';
          html += '<i class="nav-icon ' + meta.icon + ' ' + colorClass + '"></i>';
          html += '<span class="nav-text">' + meta.title + '</span>';
          html += '<button type="button" class="nav-item-unpin js-unpin-btn" data-key="' + key + '" title="移除此项">×</button>';
          html += '</div>';
        }
      });
      container.innerHTML = html;

      if (isEditMode) {
        bindEditModeEvents();
      }
    }

    function bindEditModeEvents() {
      // 1. 删除按钮事件
      container.querySelectorAll(".js-unpin-btn").forEach(function (btn) {
        btn.addEventListener("click", function (e) {
          e.preventDefault();
          e.stopPropagation();
          var k = btn.getAttribute("data-key");
          if (k) {
            var idx = currentOrder.indexOf(k);
            if (idx !== -1) {
              currentOrder.splice(idx, 1);
              window.WorkbenchPinned.savePinnedKeys(currentOrder);
              renderPinnedList();
            }
          }
        });
      });

      // 2. 原生 HTML5 实时拖拽重排与保序
      function syncOrderFromDOM() {
        var newOrder = [];
        container.querySelectorAll(".is-editing-item").forEach(function (el) {
          var k = el.getAttribute("data-pinned-key");
          if (k) newOrder.push(k);
        });
        currentOrder = newOrder;
        window.WorkbenchPinned.savePinnedKeys(currentOrder);
      }

      var draggedEl = null;
      container.querySelectorAll(".is-editing-item").forEach(function (item) {
        item.addEventListener("dragstart", function (e) {
          draggedEl = item;
          e.dataTransfer.effectAllowed = "move";
          e.dataTransfer.setData("text/plain", item.getAttribute("data-pinned-key") || "");
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
          } else {
            if (item.nextElementSibling !== draggedEl) {
              container.insertBefore(draggedEl, item.nextSibling);
            }
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

    function toggleEditMode() {
      isEditMode = !isEditMode;
      container.classList.toggle("is-editing", isEditMode);
      if (editTip) {
        editTip.style.display = isEditMode ? "block" : "none";
      }
      if (manageBtn) {
        manageBtn.classList.toggle("is-editing", isEditMode);
        if (isEditMode) {
          manageBtn.innerHTML = '<i class="fas fa-check" aria-hidden="true"></i><span>完成</span>';
          manageBtn.title = "完成排序与修改";
        } else {
          manageBtn.innerHTML = '<span>编辑</span>';
          manageBtn.title = "自定义与排序常用页面";
          window.WorkbenchPinned.savePinnedKeys(currentOrder);
          if (typeof window.showToast === "function") {
            window.showToast("已保存自选常用排序与配置", "success");
          }
        }
      }
      renderPinnedList();
    }

    if (manageBtn) {
      manageBtn.addEventListener("click", function (e) {
        e.preventDefault();
        toggleEditMode();
      });
    }

    renderPinnedList();
    window.addEventListener("workbench:pinned-change", function () {
      if (!isEditMode) {
        currentOrder = window.WorkbenchPinned.getPinnedKeys().slice();
        renderPinnedList();
      }
    });

    // 模态弹窗开启入口
    var openModalHandler = function (e) {
      e.preventDefault();
      var keys = currentOrder;
      modal.querySelectorAll(".po-pinned-modal-card").forEach(function (card) {
        var k = card.getAttribute("data-key");
        var chk = card.querySelector("input[type='checkbox']");
        if (chk) chk.checked = (keys.indexOf(k) !== -1);
        card.classList.toggle("is-selected", chk && chk.checked);
      });
      updateModalCount();
      modal.style.display = "flex";
    };

    if (openBtn && modal) openBtn.addEventListener("click", openModalHandler);

    function updateModalCount() {
      if (!modal || !tipEl) return;
      var checked = modal.querySelectorAll(".po-pinned-modal-card input[type='checkbox']:checked").length;
      tipEl.textContent = "已固定 " + checked + " 个功能页面";
    }

    if (modal) {
      modal.querySelectorAll(".po-pinned-modal-card").forEach(function (card) {
        var chk = card.querySelector("input[type='checkbox']");
        if (chk) {
          chk.addEventListener("change", function () {
            card.classList.toggle("is-selected", chk.checked);
            updateModalCount();
          });
        }
      });

      var closeModal = function () { modal.style.display = "none"; };
      if (closeBtn) closeBtn.addEventListener("click", closeModal);
      if (cancelBtn) cancelBtn.addEventListener("click", closeModal);
      modal.addEventListener("click", function (e) {
        if (e.target === modal) closeModal();
      });
      document.addEventListener("keydown", function (e) {
        if (e.key === "Escape" && modal.style.display === "flex") {
          closeModal();
        }
      });
      if (saveBtn) {
        saveBtn.addEventListener("click", function () {
          var selectedKeys = [];
          modal.querySelectorAll(".po-pinned-modal-card input[type='checkbox']:checked").forEach(function (chk) {
            selectedKeys.push(chk.value);
          });
          var updatedOrder = currentOrder.filter(function (k) { return selectedKeys.indexOf(k) !== -1; });
          selectedKeys.forEach(function (k) {
            if (updatedOrder.indexOf(k) === -1) updatedOrder.push(k);
          });
          currentOrder = updatedOrder;
          window.WorkbenchPinned.savePinnedKeys(currentOrder);
          closeModal();
          renderPinnedList();
        });
      }
    }
  }

  initPinnedWorkbench();
})();
