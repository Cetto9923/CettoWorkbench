/* ==========================================================================
   文件: web/static/js/layout/header.js
   模块: 布局
   职责: 顶栏外观模式（跟随系统 / 浅色 / 深色）与主题色系（经典蓝 / 樱花粉）正交管理
         支持四套主题（蓝浅 / 蓝深 / 粉浅 / 粉深）、localStorage 持久化与首屏零闪屏同步。
========================================================================== */
(function () {
  "use strict";

  var MODE_KEY = "workbench.theme.mode";
  var COLOR_KEY = "workbench.theme.color";
  var LEGACY_PREF_KEY = "workbench.theme.preference";

  var VALID_MODES = { system: true, light: true, dark: true };
  var VALID_COLORS = { blue: true, pink: true };

  var currentMode = "light";
  var currentColor = "blue";

  try {
    var storedMode = localStorage.getItem(MODE_KEY);
    var storedColor = localStorage.getItem(COLOR_KEY);
    var legacyPref = localStorage.getItem(LEGACY_PREF_KEY);

    if (storedMode && VALID_MODES[storedMode]) {
      currentMode = storedMode;
    } else if (legacyPref) {
      if (legacyPref === "pink") {
        currentMode = "light";
        if (!storedColor) currentColor = "pink";
      } else if (VALID_MODES[legacyPref]) {
        currentMode = legacyPref;
      }
    }

    if (storedColor && VALID_COLORS[storedColor]) {
      currentColor = storedColor;
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

  function getEffectiveTheme(mode) {
    var m = mode || currentMode;
    if (m === "system") {
      return getSystemTheme();
    }
    return m === "dark" ? "dark" : "light";
  }

  function getModeLabel(mode, effective) {
    if (mode === "system") {
      return "跟随系统（" + (effective === "dark" ? "深色" : "浅色") + "）";
    }
    return mode === "dark" ? "深色模式" : "浅色模式";
  }

  function getColorLabel(color) {
    return color === "pink" ? "樱花粉" : "经典蓝";
  }

  function applyTheme() {
    if (typeof document === "undefined" || !document.documentElement) {
      return;
    }
    var effective = getEffectiveTheme(currentMode);
    var root = document.documentElement;
    root.setAttribute("data-theme", effective);
    root.setAttribute("data-theme-color", currentColor);
    root.setAttribute("data-theme-pref", currentMode);
    if (root.dataset) {
      root.dataset.theme = effective;
      root.dataset.themeColor = currentColor;
      root.dataset.themePref = currentMode;
    }

    var systemSub = document.getElementById("themeSystemSub");
    if (systemSub) {
      systemSub.textContent = getSystemTheme() === "dark" ? "深色" : "浅色";
    }

    var quickToggle = document.getElementById("themeQuickToggle");
    if (quickToggle) {
      var icon = quickToggle.querySelector(".theme-quick-icon") || quickToggle.querySelector("i");
      if (icon) {
        if (currentMode === "system") {
          icon.className = "fas fa-circle-half-stroke theme-quick-icon";
        } else if (effective === "dark") {
          icon.className = "fas fa-sun theme-quick-icon";
        } else {
          icon.className = "fas fa-moon theme-quick-icon";
        }
      }

      var nextModeText = effective === "dark" ? "浅色模式" : "深色模式";
      var label =
        "当前：" +
        getColorLabel(currentColor) +
        " · " +
        getModeLabel(currentMode, effective) +
        "（点击切换至" +
        nextModeText +
        "）";
      quickToggle.title = label;
      quickToggle.setAttribute("aria-label", label);
    }
  }

  function upgradeCachedDropdownDOM() {
    if (typeof document === "undefined") return;
    var existingColorGroup = document.querySelector(".theme-color-group");
    if (existingColorGroup) return;

    var pickerGroup = document.querySelector(".topbar-actions .theme-picker-group");
    if (!pickerGroup) return;

    var headerEl = pickerGroup.querySelector(".dropdown-header");
    if (headerEl) headerEl.textContent = "外观模式";

    var pinkBtn = pickerGroup.querySelector('[data-theme-value="pink"]');
    if (pinkBtn && pinkBtn.parentNode) {
      pinkBtn.parentNode.removeChild(pinkBtn);
    }

    var modeBtns = pickerGroup.querySelectorAll("[data-theme-value]");
    for (var i = 0; i < modeBtns.length; i++) {
      var v = modeBtns[i].getAttribute("data-theme-value");
      if (v) modeBtns[i].setAttribute("data-theme-mode", v);
    }

    var divider = document.createElement("div");
    divider.className = "dropdown-divider";

    var colorGroup = document.createElement("div");
    colorGroup.className = "theme-picker-group theme-color-group";
    colorGroup.setAttribute("role", "group");
    colorGroup.setAttribute("aria-label", "主题色系");
    colorGroup.innerHTML =
      '<div class="dropdown-header">主题色系</div>' +
      '<button type="button" class="dropdown-item" role="menuitemradio" aria-checked="true" data-theme-color-opt="blue">' +
      '<i class="fas fa-circle theme-blue-swatch"></i><span>经典蓝</span><i class="fas fa-check check-mark"></i>' +
      "</button>" +
      '<button type="button" class="dropdown-item" role="menuitemradio" aria-checked="false" data-theme-color-opt="pink">' +
      '<i class="fas fa-heart theme-pink-swatch"></i><span>樱花粉</span><i class="fas fa-check check-mark"></i>' +
      "</button>";

    if (pickerGroup.parentNode) {
      pickerGroup.parentNode.insertBefore(colorGroup, pickerGroup.nextSibling);
      pickerGroup.parentNode.insertBefore(divider, colorGroup);
    }
  }

  function syncControls() {
    if (typeof document === "undefined" || !document.querySelectorAll) {
      return;
    }
    var modeControls = document.querySelectorAll('[role="menuitemradio"][data-theme-mode]');
    for (var i = 0; i < modeControls.length; i++) {
      var ctrl = modeControls[i];
      var val = ctrl.getAttribute("data-theme-mode");
      var checked = val === currentMode;
      ctrl.setAttribute("aria-checked", checked ? "true" : "false");
      ctrl.classList.toggle("active", checked);
    }

    var colorControls = document.querySelectorAll('[role="menuitemradio"][data-theme-color-opt]');
    for (var j = 0; j < colorControls.length; j++) {
      var cCtrl = colorControls[j];
      var cVal = cCtrl.getAttribute("data-theme-color-opt");
      var cChecked = cVal === currentColor;
      cCtrl.setAttribute("aria-checked", cChecked ? "true" : "false");
      cCtrl.classList.toggle("active", cChecked);
    }
  }

  function persistState() {
    try {
      localStorage.setItem(MODE_KEY, currentMode);
      localStorage.setItem(COLOR_KEY, currentColor);
      localStorage.setItem(LEGACY_PREF_KEY, currentMode);
    } catch (e) {
      // ignore
    }
  }

  function setMode(mode) {
    currentMode = VALID_MODES[mode] ? mode : "system";
    persistState();
    applyTheme();
    syncControls();
  }

  function setColor(color) {
    currentColor = VALID_COLORS[color] ? color : "blue";
    persistState();
    applyTheme();
    syncControls();
  }

  function setPreference(pref) {
    if (pref === "pink") {
      setColor("pink");
      return;
    }
    if (pref === "blue") {
      setColor("blue");
      return;
    }
    setMode(pref);
  }

  /* 顶栏快捷按钮：点击直接在浅色与深色之间切换 */
  function toggleQuick() {
    var effective = getEffectiveTheme(currentMode);
    var next = effective === "dark" ? "light" : "dark";
    setMode(next);
  }

  applyTheme();

  if (typeof window !== "undefined") {
    window.WorkbenchTheme = {
      getPreference: function () {
        return currentMode;
      },
      getMode: function () {
        return currentMode;
      },
      getColor: function () {
        return currentColor;
      },
      getEffectiveTheme: function () {
        return getEffectiveTheme(currentMode);
      },
      setPreference: setPreference,
      setMode: setMode,
      setColor: setColor,
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
          if (currentMode === "system") {
            applyTheme();
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
      if (!e) return;
      if (e.key === MODE_KEY && e.newValue && VALID_MODES[e.newValue]) {
        currentMode = e.newValue;
        applyTheme();
        syncControls();
      } else if (e.key === COLOR_KEY && e.newValue && VALID_COLORS[e.newValue]) {
        currentColor = e.newValue;
        applyTheme();
        syncControls();
      }
    });
  }

  function bindControls() {
    upgradeCachedDropdownDOM();
    syncControls();
    applyTheme();

    var modeControls = document.querySelectorAll('[role="menuitemradio"][data-theme-mode]');
    for (var i = 0; i < modeControls.length; i++) {
      if (modeControls[i]._modeBound) continue;
      modeControls[i]._modeBound = true;
      modeControls[i].addEventListener("click", function (e) {
        e.preventDefault();
        setMode(this.getAttribute("data-theme-mode"));
      });
    }

    var colorControls = document.querySelectorAll('[role="menuitemradio"][data-theme-color-opt]');
    for (var j = 0; j < colorControls.length; j++) {
      if (colorControls[j]._colorBound) continue;
      colorControls[j]._colorBound = true;
      colorControls[j].addEventListener("click", function (e) {
        e.preventDefault();
        setColor(this.getAttribute("data-theme-color-opt"));
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
