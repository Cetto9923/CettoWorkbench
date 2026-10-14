/* ==========================================================================
   文件: web/static/js/layout/header.js
   模块: 布局
   职责: 顶栏主题偏好（浅色/深色）与外观控件同步。
========================================================================== */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.theme.preference";
  var currentPreference = "light";

  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    // 暂不支持跟随系统；历史 system 偏好回落为 light
    if (stored === "light" || stored === "dark") {
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
    return p === "dark" ? "dark" : "light";
  }

  function applyTheme(effectiveTheme) {
    if (typeof document === "undefined" || !document.documentElement) {
      return;
    }
    var theme = effectiveTheme || getEffectiveTheme();
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
          quickToggle.setAttribute("aria-label", "切换为浅色模式");
        } else {
          moon.style.display = "inline-block";
          sun.style.display = "none";
          quickToggle.title = "切换为深色模式";
          quickToggle.setAttribute("aria-label", "切换为深色模式");
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
    currentPreference = pref === "dark" ? "dark" : "light";
    try {
      localStorage.setItem(STORAGE_KEY, currentPreference);
    } catch (e) {
      // ignore
    }
    applyTheme(getEffectiveTheme(currentPreference));
    syncControls();
  }

  function toggleQuick() {
    setPreference(getEffectiveTheme() === "dark" ? "light" : "dark");
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

    window.addEventListener("storage", function (e) {
      if (!e || e.key !== STORAGE_KEY) {
        return;
      }
      if (e.newValue === "light" || e.newValue === "dark") {
        currentPreference = e.newValue;
      } else {
        currentPreference = "light";
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
