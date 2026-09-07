/**
 * 文件: web/static/js/layout/topnav.js
 * 模块: 布局
 * 职责: 顶栏一级/二级联动显示；切换一级时默认进入该组首个二级链接；同步 --topbar-stack-height。
 */
(function () {
  function syncTopbarStackHeight() {
    var bar = document.querySelector("header.topbar");
    if (!bar) {
      return;
    }
    var h = Math.round(bar.getBoundingClientRect().height);
    document.documentElement.style.setProperty("--topbar-stack-height", h + "px");
  }

  function setSecondaryVisible(visible) {
    var strip = document.getElementById("topnav-secondary");
    if (!strip) {
      syncTopbarStackHeight();
      return;
    }
    if (visible) {
      strip.removeAttribute("hidden");
    } else {
      strip.setAttribute("hidden", "");
    }
    syncTopbarStackHeight();
  }

  function showPanel(panelId) {
    var strip = document.getElementById("topnav-secondary");
    if (!strip) {
      return;
    }
    var panels = strip.querySelectorAll(".topnav-panel");
    for (var i = 0; i < panels.length; i++) {
      var p = panels[i];
      var pid = p.getAttribute("data-topnav-panel");
      if (pid === panelId) {
        p.removeAttribute("hidden");
        var subs = p.querySelectorAll(".topnav-subitem");
        var hasActive = false;
        for (var j = 0; j < subs.length; j++) {
          if (subs[j].classList.contains("is-active")) {
            hasActive = true;
            break;
          }
        }
        if (!hasActive && subs.length > 0) {
          subs[0].classList.add("is-active");
        }
      } else {
        p.setAttribute("hidden", "");
      }
    }
  }

  function goFirstSecondaryIfChanged(panelId) {
    var strip = document.getElementById("topnav-secondary");
    if (!strip) {
      return;
    }
    var panel = strip.querySelector(
      '.topnav-panel[data-topnav-panel="' + panelId + '"]'
    );
    if (!panel) {
      return;
    }
    var first = panel.querySelector("a.topnav-subitem");
    if (!first) {
      return;
    }
    var raw = first.getAttribute("href");
    if (!raw || raw === "#" || raw.indexOf("javascript:") === 0) {
      return;
    }
    var target = first.href;
    try {
      var cur = new URL(window.location.href);
      var next = new URL(target);
      if (cur.pathname === next.pathname && cur.search === next.search) {
        return;
      }
    } catch (e) {
      // ignore parse errors, still assign href below
    }
    window.location.href = target;
  }

  function clearPrimaryActive() {
    var items = document.querySelectorAll(".topnav-primary-item.is-active");
    for (var i = 0; i < items.length; i++) {
      items[i].classList.remove("is-active");
    }
  }

  function init() {
    var primary = document.getElementById("topnav-primary");
    var strip = document.getElementById("topnav-secondary");
    if (primary && strip) {
      primary.addEventListener("click", function (ev) {
        var t = ev.target;
        if (!t || !t.closest) {
          return;
        }
        if (t.closest(".topnav-dropdown")) {
          return;
        }
        var panelBtn = t.closest(".topnav-primary-item[data-topnav-panel]");
        if (panelBtn) {
          ev.preventDefault();
          var panelId = panelBtn.getAttribute("data-topnav-panel");
          var wasActive = panelBtn.classList.contains("is-active");
          clearPrimaryActive();
          panelBtn.classList.add("is-active");
          showPanel(panelId);
          setSecondaryVisible(true);
          if (!wasActive) {
            goFirstSecondaryIfChanged(panelId);
          }
          return;
        }
        var leaf = t.closest(".topnav-primary-item--leaf");
        if (leaf) {
          if (leaf.tagName === "A" && leaf.getAttribute("href") === "#") {
            ev.preventDefault();
          }
          clearPrimaryActive();
          leaf.classList.add("is-active");
          setSecondaryVisible(false);
          var panels = strip.querySelectorAll(".topnav-panel");
          for (var i = 0; i < panels.length; i++) {
            panels[i].setAttribute("hidden", "");
          }
        }
      });

      strip.addEventListener("click", function (ev) {
        var t = ev.target;
        if (!t || !t.closest) {
          return;
        }
        var sub = t.closest(".topnav-subitem");
        if (!sub) {
          return;
        }
        var panel = sub.closest(".topnav-panel");
        if (!panel) {
          return;
        }
        var subs = panel.querySelectorAll(".topnav-subitem");
        for (var i = 0; i < subs.length; i++) {
          subs[i].classList.remove("is-active");
        }
        sub.classList.add("is-active");
      });
    }

    var bar = document.querySelector("header.topbar");
    if (bar && typeof ResizeObserver !== "undefined") {
      var ro = new ResizeObserver(function () {
        syncTopbarStackHeight();
      });
      ro.observe(bar);
    }
    window.addEventListener("resize", syncTopbarStackHeight);
    syncTopbarStackHeight();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
