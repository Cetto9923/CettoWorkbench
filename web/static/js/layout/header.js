/* ==========================================================================
   文件: web/static/js/layout/header.js
   模块: 布局
   职责: 顶栏主题偏好（跟随系统 / 浅色 / 深色 / 樱花粉）与外观控件同步，支持系统深浅色实时联动。
========================================================================== */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.theme.preference";
  var VALID_PREFS = { system: true, light: true, dark: true, pink: true };
  var currentPreference = "system";

  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    if (stored && VALID_PREFS[stored]) {
      currentPreference = stored;
    }
  } catch (e) {
    // ignore
  }

  function getSystemTheme() {
    try {
      if (
        typeof window !== "undefined" &&
        window.matchMedia &&
        window.matchMedia("(prefers-color-scheme: dark)").matches
      ) {
        return "dark";
      }
    } catch (e) {
      // ignore
    }
    return "light";
  }

  function getPreference() {
    return currentPreference;
  }

  function getEffectiveTheme(pref) {
    var p = pref || currentPreference;
    if (p === "system") {
      return getSystemTheme();
    }
    if (p === "dark" || p === "pink") {
      return p;
    }
    return "light";
  }

  function getThemeLabel(pref, effective) {
    if (pref === "system") {
      return "跟随系统（当前" + (effective === "dark" ? "深色" : "浅色") + "）";
    }
    if (pref === "dark") return "深色模式";
    if (pref === "pink") return "樱花粉主题";
    return "浅色模式";
  }

  function applyTheme(effectiveTheme) {
    if (typeof document === "undefined" || !document.documentElement) {
      return;
    }
    var theme = effectiveTheme || getEffectiveTheme(currentPreference);
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.setAttribute("data-theme-pref", currentPreference);
    if (document.documentElement.dataset) {
      document.documentElement.dataset.theme = theme;
      document.documentElement.dataset.themePref = currentPreference;
    }

    var systemSub = document.getElementById("themeSystemSub");
    if (systemSub) {
      systemSub.textContent = getSystemTheme() === "dark" ? "深色" : "浅色";
    }

    var quickToggle = document.getElementById("themeQuickToggle");
    if (quickToggle) {
      var sysIcon = quickToggle.querySelector(".theme-system-icon");
      var moon = quickToggle.querySelector(".theme-moon-icon");
      var sun = quickToggle.querySelector(".theme-sun-icon");

      if (sysIcon) sysIcon.style.display = "none";
      if (moon) moon.style.display = "none";
      if (sun) sun.style.display = "none";

      if (currentPreference === "system" && sysIcon) {
        sysIcon.style.display = "inline-block";
      } else if (theme === "dark" && sun) {
        sun.style.display = "inline-block";
      } else if (moon) {
        moon.style.display = "inline-block";
      }

      var nextAction = theme === "dark" ? "切换为浅色模式" : "切换为深色模式";
      var label = "当前：" + getThemeLabel(currentPreference, theme) + " · 点击" + nextAction;
      quickToggle.title = label;
      quickToggle.setAttribute("aria-label", label);
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
    currentPreference = VALID_PREFS[pref] ? pref : "system";
    try {
      localStorage.setItem(STORAGE_KEY, currentPreference);
    } catch (e) {
      // ignore
    }
    applyTheme(getEffectiveTheme(currentPreference));
    syncControls();
  }

  function toggleQuick() {
    var effective = getEffectiveTheme(currentPreference);
    setPreference(effective === "dark" ? "light" : "dark");
  }

  applyTheme(getEffectiveTheme(currentPreference));

  if (typeof window !== "undefined") {
    window.WorkbenchTheme = {
      getPreference: getPreference,
      getEffectiveTheme: function () {
        return getEffectiveTheme(currentPreference);
      },
      setPreference: setPreference,
      syncControls: syncControls,
      toggleQuick: toggleQuick,
    };

    try {
      if (window.matchMedia) {
        var mql = window.matchMedia("(prefers-color-scheme: dark)");
        var onSystemThemeChange = function () {
          var systemSub = document.getElementById("themeSystemSub");
          if (systemSub) {
            systemSub.textContent = getSystemTheme() === "dark" ? "深色" : "浅色";
          }
          if (currentPreference === "system") {
            applyTheme(getEffectiveTheme("system"));
            syncControls();
          }
        };
        if (typeof mql.addEventListener === "function") {
          mql.addEventListener("change", onSystemThemeChange);
        } else if (typeof mql.addListener === "function") {
          mql.addListener(onSystemThemeChange);
        }
      }
    } catch (e) {
      // ignore
    }

    window.addEventListener("storage", function (e) {
      if (!e || e.key !== STORAGE_KEY) {
        return;
      }
      if (e.newValue && VALID_PREFS[e.newValue]) {
        currentPreference = e.newValue;
      } else {
        currentPreference = "system";
      }
      applyTheme(getEffectiveTheme(currentPreference));
      syncControls();
    });
  }

  function bindControls() {
    syncControls();
    applyTheme(getEffectiveTheme(currentPreference));
    var controls = document.querySelectorAll('[role="menuitemradio"][data-theme-value]');
    for (var i = 0; i < controls.length; i++) {
      controls[i].addEventListener("click", function (e) {
        e.preventDefault();
        var val = this.getAttribute("data-theme-value");
        setPreference(val);
      });
    }

    var quickToggle = document.getElementById("themeQuickToggle");
    if (quickToggle && !quickToggle._themeBound) {
      quickToggle._themeBound = true;
      quickToggle.addEventListener("click", function (e) {
        e.preventDefault();
        toggleQuick();
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
