/* ==========================================================================
   文件: web/static/js/layout/header.js
   模块: 布局
   职责: 顶栏主题偏好（当前仅浅色）与外观控件同步。
========================================================================== */
(function () {
  "use strict";

  var STORAGE_KEY = "workbench.theme.preference";
  var currentPreference = "light";

  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    // 当前仅支持浅色；其它历史偏好统一回落为 light
    if (stored === "light") {
      currentPreference = "light";
    }
  } catch (e) {
    // ignore
  }

  function getPreference() {
    return currentPreference;
  }

  function getEffectiveTheme() {
    return "light";
  }

  function applyTheme() {
    if (typeof document === "undefined" || !document.documentElement) {
      return;
    }
    document.documentElement.setAttribute("data-theme", "light");
    if (document.documentElement.dataset) {
      document.documentElement.dataset.theme = "light";
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
    // 仅接受浅色；其它值一律回落
    currentPreference = pref === "light" ? "light" : "light";
    try {
      localStorage.setItem(STORAGE_KEY, currentPreference);
    } catch (e) {
      // ignore
    }
    applyTheme();
    syncControls();
  }

  applyTheme();

  if (typeof window !== "undefined") {
    window.WorkbenchTheme = {
      getPreference: getPreference,
      getEffectiveTheme: getEffectiveTheme,
      setPreference: setPreference,
      syncControls: syncControls,
    };
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
  }

  if (typeof document !== "undefined") {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", bindControls);
    } else {
      bindControls();
    }
  }
})();
