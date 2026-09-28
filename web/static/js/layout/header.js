/* ==========================================================================
   文件: web/static/js/layout/header.js
   模块: 布局
   职责: 顶栏主题偏好（跟随系统/浅色/深色）与外观控件同步。
========================================================================== */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.theme.preference";
  var currentPreference = "light";

  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "light" || stored === "dark" || stored === "system") {
      currentPreference = stored;
    }
  } catch (e) {
    // ignore
  }

  function getPreference() {
    return currentPreference;
  }

  function getEffectiveTheme(pref) {
    var p = pref || currentPreference;
    if (p === "dark") return "dark";
    if (p === "system") {
      if (typeof window !== "undefined" && window.matchMedia) {
        try {
          if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
            return "dark";
          }
        } catch (e) {}
      }
    }
    return "light";
  }

  function applyTheme(effectiveTheme) {
    if (typeof document === "undefined" || !document.documentElement) {
      return;
    }
    var theme = effectiveTheme || getEffectiveTheme(currentPreference);
    document.documentElement.setAttribute("data-theme", theme);
    if (document.documentElement.dataset) {
      document.documentElement.dataset.theme = theme;
    }
    var quickToggle = document.getElementById("themeQuickToggle");
    if (quickToggle) {
      var moon = quickToggle.querySelector(".theme-moon-icon");
      var sun = quickToggle.querySelector(".theme-sun-icon");
      if (moon && sun) {
        if (theme === "dark") {
          moon.style.display = "none";
          sun.style.display = "inline-block";
          quickToggle.title = "切换为浅色模式";
        } else {
          moon.style.display = "inline-block";
          sun.style.display = "none";
          quickToggle.title = "切换为深色模式";
        }
      }
    }
  }

  function syncControls() {
    if (typeof document === "undefined" || !document.querySelectorAll) {
      return;
    }
    var controls = document.querySelectorAll('[role="menuitemradio"][data-theme-value]');
    for (var i = 0; i < controls.length; i++) {
      var ctrl = controls[i];
      var val = ctrl.getAttribute("data-theme-value");
      if (val === currentPreference) {
        ctrl.setAttribute("aria-checked", "true");
        if (ctrl.classList) {
          ctrl.classList.add("active");
        }
      } else {
        ctrl.setAttribute("aria-checked", "false");
        if (ctrl.classList) {
          ctrl.classList.remove("active");
        }
      }
    }
  }

  function setPreference(pref) {
    if (pref !== "light" && pref !== "dark" && pref !== "system") {
      pref = "light";
    }
    currentPreference = pref;
    try {
      localStorage.setItem(STORAGE_KEY, currentPreference);
    } catch (e) {
      // ignore
    }
    var effectiveTheme = getEffectiveTheme(currentPreference);
    applyTheme(effectiveTheme);
    syncControls();
  }

  applyTheme();

  if (typeof window !== "undefined") {
    window.WorkbenchTheme = {
      getPreference: getPreference,
      getEffectiveTheme: function () {
        return getEffectiveTheme(currentPreference);
      },
      setPreference: setPreference,
      syncControls: syncControls,
      toggleQuick: function () {
        var eff = getEffectiveTheme(currentPreference);
        setPreference(eff === "dark" ? "light" : "dark");
      }
    };
  }

  // 监听系统主题变化
  if (typeof window !== "undefined" && window.matchMedia) {
    try {
      var mql = window.matchMedia("(prefers-color-scheme: dark)");
      var handleMediaChange = function () {
        if (currentPreference === "system") {
          applyTheme(getEffectiveTheme("system"));
        }
      };
      if (mql && mql.addEventListener) {
        mql.addEventListener("change", handleMediaChange);
      } else if (mql && mql.addListener) {
        mql.addListener(handleMediaChange);
      }
    } catch (e) {}
  }

  // 跨标签页同步
  if (typeof window !== "undefined" && window.addEventListener) {
    window.addEventListener("storage", function (e) {
      if (!e || e.key !== STORAGE_KEY) return;
      var val = e.newValue;
      if (val === "light" || val === "dark" || val === "system") {
        currentPreference = val;
      } else {
        currentPreference = "light";
      }
      applyTheme(getEffectiveTheme(currentPreference));
      syncControls();
    });
  }

  function bindControls() {
    syncControls();
    var controls = document.querySelectorAll('[role="menuitemradio"][data-theme-value]');
    for (var i = 0; i < controls.length; i++) {
      controls[i].addEventListener("click", function (e) {
        e.preventDefault();
        var val = this.getAttribute("data-theme-value");
        setPreference(val);
      });
    }

    var quickToggle = document.getElementById("themeQuickToggle");
    if (quickToggle) {
      quickToggle.addEventListener("click", function (e) {
        e.preventDefault();
        if (window.WorkbenchTheme && window.WorkbenchTheme.toggleQuick) {
          window.WorkbenchTheme.toggleQuick();
        }
      });
    }
  }

  if (typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", bindControls);
    } else {
      bindControls();
    }
  }
})();
