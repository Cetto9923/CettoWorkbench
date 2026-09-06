const fs = require('fs');
const path = require('path');
const vm = require('vm');

const themeJsPath = path.join(__dirname, '../../web/static/js/layout/theme.js');
const themeJsContent = fs.readFileSync(themeJsPath, 'utf8');

const baseHtmlPath = path.join(__dirname, '../../web/templates/layout/base.html');
const baseHtmlContent = fs.readFileSync(baseHtmlPath, 'utf8');

const headerHtmlPath = path.join(__dirname, '../../web/templates/layout/header.html');
const headerHtmlContent = fs.readFileSync(headerHtmlPath, 'utf8');

function assert(condition, message) {
  if (!condition) {
    console.error("FAIL: " + message);
    process.exit(1);
  }
}

// 1. First Paint / Structural Test
function checkStructural() {
  const themeJsScript = '<script src="/static/js/layout/theme.js"></script>';
  const variablesCss = '<link rel="stylesheet" href="/static/css/layout/variables.css">';
  const appCss = '<link rel="stylesheet" href="/static/css/layout/app.css">';

  assert(baseHtmlContent.includes(themeJsScript), "theme.js script tag must be present in base.html");
  assert(!baseHtmlContent.includes('<script src="/static/js/layout/theme.js" defer>'), "theme.js must not use defer");
  assert(!baseHtmlContent.includes('<script src="/static/js/layout/theme.js" async>'), "theme.js must not use async");

  const themeIndex = baseHtmlContent.indexOf(themeJsScript);
  const varsIndex = baseHtmlContent.indexOf(variablesCss);
  const appIndex = baseHtmlContent.indexOf(appCss);

  assert(themeIndex !== -1 && varsIndex !== -1 && themeIndex < varsIndex, "theme.js must be loaded before variables.css");
  assert(appIndex !== -1 && themeIndex < appIndex, "theme.js must be loaded before app.css");

  console.log("PASS: First Paint Structural Test");
}

// 2. Template Branching Test (PO Workbench, Sidebar layout, Top navigation layout)
function checkTemplateBranching() {
  assert(headerHtmlContent.includes('{{ if or (eq .CurrentPath "/home")'), "header.html must have PO path branch check");

  const poSplit = headerHtmlContent.split('{{ if or (eq .CurrentPath "/home")')[1];
  assert(poSplit, "PO branch split must exist");

  const poChunk = poSplit.split('<div class="topbar-brand">')[0];
  const nonPoChunk = poSplit.split('<div class="topbar-brand">')[1];

  // PO branch check
  assert(poChunk && poChunk.includes('{{ template "layout/themepicker" }}'),
    "PO Workbench header must include layout/themepicker");

  // Non-PO branch check (both sidebar and top navigation layouts)
  assert(nonPoChunk && nonPoChunk.includes('{{ template "layout/themepicker" }}'),
    "Non-PO layout must include layout/themepicker in topbar-actions");

  // Ensure topbar-actions is not restricted to sidebar layout only
  assert(!nonPoChunk.includes('{{ if eq .LayoutNav "sidebar" }}\n  <div class="topbar-actions">'),
    "topbar-actions must be accessible for both top navigation and sidebar layouts");

  console.log("PASS: Template Branching Test (PO, Sidebar, Top Nav)");
}

checkStructural();
checkTemplateBranching();

function createEnvironment(initialStorage = {}, prefersDark = false, options = {}) {
  const storage = { ...initialStorage };
  const mediaQueryListeners = [];
  let isDark = prefersDark;
  const DOMContentLoadedListeners = [];
  const storageListeners = [];
  const registeredEvents = [];

  const mockLocalStorage = {
    getItem: (key) => {
      if (options.throwOnGet) {
        throw new Error("localStorage.getItem security error");
      }
      return storage[key] !== undefined ? storage[key] : null;
    },
    setItem: (key, val) => {
      if (options.throwOnSet) {
        throw new Error("localStorage.setItem quota exceeded");
      }
      storage[key] = String(val);
      // Native behavior: setItem does NOT dispatch storage event to current window/tab!
    }
  };

  const window = {
    localStorage: mockLocalStorage,
    addEventListener: (event, fn) => {
      if (event === 'storage') storageListeners.push(fn);
    },
    triggerThemeChange: (newDark) => {
      isDark = newDark;
      mediaQueryListeners.forEach(fn => fn());
    },
    dispatchCrossTabStorageEvent: (key, newValue) => {
      storageListeners.forEach(fn => fn({ key, newValue }));
    }
  };

  if (!options.noMatchMedia) {
    window.matchMedia = (query) => {
      return {
        matches: query === '(prefers-color-scheme: dark)' ? isDark : false,
        addEventListener: (event, fn) => {
          if (event === 'change') mediaQueryListeners.push(fn);
        },
        addListener: (fn) => {
          mediaQueryListeners.push(fn);
        }
      };
    };
  }

  const documentElement = {
    dataset: {},
    setAttribute: function(attr, val) {
      this[attr] = val;
      if (attr === 'data-theme') {
        this.dataset.theme = val;
      }
    }
  };

  const createControl = (themeVal) => {
    return {
      role: 'menuitemradio',
      attrs: {
        'role': 'menuitemradio',
        'data-theme-value': themeVal,
        'aria-checked': 'false'
      },
      classList: {
        classes: new Set(),
        add: function(c) { this.classes.add(c); },
        remove: function(c) { this.classes.delete(c); },
        contains: function(c) { return this.classes.has(c); }
      },
      getAttribute: function(a) { return this.attrs[a] !== undefined ? this.attrs[a] : null; },
      setAttribute: function(a, v) { this.attrs[a] = String(v); },
      listeners: {},
      addEventListener: function(event, fn) {
        if (!this.listeners[event]) this.listeners[event] = [];
        this.listeners[event].push(fn);
        registeredEvents.push({ event, themeVal });
      },
      click: function() {
        if (this.listeners['click']) {
          this.listeners['click'].forEach(fn => fn.call(this, { preventDefault: () => {} }));
        }
      }
    };
  };

  const elements = [
    createControl('system'),
    createControl('light'),
    createControl('dark')
  ];

  const document = {
    documentElement: documentElement,
    readyState: 'complete',
    addEventListener: (event, fn) => {
      if (event === 'DOMContentLoaded') DOMContentLoadedListeners.push(fn);
    },
    triggerDOMContentLoaded: () => {
      DOMContentLoadedListeners.forEach(fn => fn());
    },
    querySelectorAll: (selector) => {
      if (selector === '[role="menuitemradio"][data-theme-value]') {
        return elements;
      }
      return [];
    }
  };

  const context = {
    window,
    document,
    localStorage: window.localStorage,
    console: console
  };
  vm.createContext(context);
  vm.runInContext(themeJsContent, context);

  return {
    context,
    window,
    document,
    elements,
    storage,
    registeredEvents
  };
}

// CASE 1: 无 storage + OS light => pref light / effective light
(function() {
  const env = createEnvironment({}, false);
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 1: Pref should be light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 1: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 1: Document theme should be light");
  console.log("PASS: CASE 1");
})();

// CASE 2: 无 storage + OS dark => 仍 light / light
(function() {
  const env = createEnvironment({}, true);
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 2: Pref should be light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 2: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 2: Document theme should be light");
  console.log("PASS: CASE 2");
})();

// CASE 3: stored light + OS dark => light
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'light' }, true);
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 3: Pref should be light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 3: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 3: Document theme should be light");
  console.log("PASS: CASE 3");
})();

// CASE 4: stored dark + OS light => dark
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'dark' }, false);
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "CASE 4: Pref should be dark");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "CASE 4: Effective should be dark");
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 4: Document theme should be dark");
  console.log("PASS: CASE 4");
})();

// CASE 5: stored system + OS dark => system / dark
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, true);
  assert(env.window.WorkbenchTheme.getPreference() === 'system', "CASE 5: Pref should be system");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "CASE 5: Effective should be dark");
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 5: Document theme should be dark");
  console.log("PASS: CASE 5");
})();

// CASE 6: stored system + OS light => system / light
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, false);
  assert(env.window.WorkbenchTheme.getPreference() === 'system', "CASE 6: Pref should be system");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 6: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 6: Document theme should be light");
  console.log("PASS: CASE 6");
})();

// CASE 7: invalid stored value => light / light
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'invalid-val' }, true);
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 7: Pref should fallback to light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 7: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 7: Document theme should be light");
  console.log("PASS: CASE 7");
})();

// CASE 8: localStorage.getItem throws => no uncaught error => light / light
(function() {
  let env;
  try {
    env = createEnvironment({}, true, { throwOnGet: true });
  } catch (e) {
    assert(false, "CASE 8: Should not throw uncaught error");
  }
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 8: Pref should be light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 8: Effective should be light");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 8: Document theme should be light");
  console.log("PASS: CASE 8");
})();

// CASE 9: setPreference dark => memory dark => storage dark => effective dark => data-theme dark
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'light' }, false);
  env.window.WorkbenchTheme.setPreference('dark');
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "CASE 9: In-memory pref dark");
  assert(env.storage['workbench.theme.preference'] === 'dark', "CASE 9: Storage updated dark");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "CASE 9: Effective dark");
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 9: data-theme dark");
  console.log("PASS: CASE 9");
})();

// CASE 10: system + OS light->dark => effective follows OS
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, false);
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 10: Initial light");
  env.window.triggerThemeChange(true);
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 10: Switched to dark on OS change");
  env.window.triggerThemeChange(false);
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 10: Switched to light on OS change");
  console.log("PASS: CASE 10");
})();

// CASE 11: explicit light/dark + OS change => effective unchanged
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'light' }, false);
  env.window.triggerThemeChange(true);
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 11: Explicit light should remain light");

  env.window.WorkbenchTheme.setPreference('dark');
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 11: Explicit dark");
  env.window.triggerThemeChange(false);
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 11: Explicit dark should remain dark");
  console.log("PASS: CASE 11");
})();

// CASE 12: 任何路径：data-theme never equals system
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, true);
  assert(env.document.documentElement.dataset.theme !== 'system', "CASE 12: Initial not system");
  env.window.WorkbenchTheme.setPreference('system');
  assert(env.document.documentElement.dataset.theme !== 'system', "CASE 12: Set system not system");
  env.window.triggerThemeChange(false);
  assert(env.document.documentElement.dataset.theme !== 'system', "CASE 12: OS change not system");
  console.log("PASS: CASE 12");
})();

// CASE 13: localStorage.setItem throws => no uncaught error => memory/effective/picker dark
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'light' }, false, { throwOnSet: true });
  try {
    env.window.WorkbenchTheme.setPreference('dark');
  } catch (e) {
    assert(false, "CASE 13: Must not throw uncaught error on setItem failure");
  }
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "CASE 13: Memory preference must be dark");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "CASE 13: Effective theme must be dark");
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 13: data-theme must be dark");

  const darkControl = env.elements.find(e => e.getAttribute('data-theme-value') === 'dark');
  assert(darkControl.getAttribute('aria-checked') === 'true', "CASE 13: Dark picker must be selected");
  console.log("PASS: CASE 13");
})();

// CASE 14: cross-tab external storage event (newValue=dark)
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'light' }, false);
  env.window.dispatchCrossTabStorageEvent('workbench.theme.preference', 'dark');
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "CASE 14: Cross-tab pref dark");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "CASE 14: Cross-tab effective dark");
  assert(env.document.documentElement.dataset.theme === 'dark', "CASE 14: Cross-tab data-theme dark");

  const darkControl = env.elements.find(e => e.getAttribute('data-theme-value') === 'dark');
  assert(darkControl.getAttribute('aria-checked') === 'true', "CASE 14: Cross-tab picker dark checked");
  console.log("PASS: CASE 14");
})();

// CASE 15: external storage event newValue invalid/null => safe light fallback
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'dark' }, false);
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "CASE 15: Initial dark");

  env.window.dispatchCrossTabStorageEvent('workbench.theme.preference', null);
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 15: Fallback to light on null");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 15: Effective light on null");

  env.window.dispatchCrossTabStorageEvent('workbench.theme.preference', 'unknown-junk');
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "CASE 15: Fallback to light on invalid");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 15: Effective light on invalid");
  console.log("PASS: CASE 15");
})();

// CASE 16: matchMedia unavailable + system => effective light
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, true, { noMatchMedia: true });
  assert(env.window.WorkbenchTheme.getPreference() === 'system', "CASE 16: Pref system");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "CASE 16: Fallback to light when matchMedia unavailable");
  assert(env.document.documentElement.dataset.theme === 'light', "CASE 16: Document theme light when matchMedia unavailable");
  console.log("PASS: CASE 16");
})();

// Picker Tests (Section 16)
(function() {
  const env = createEnvironment({ 'workbench.theme.preference': 'system' }, true);
  const systemControl = env.elements.find(e => e.getAttribute('data-theme-value') === 'system');
  const lightControl = env.elements.find(e => e.getAttribute('data-theme-value') === 'light');
  const darkControl = env.elements.find(e => e.getAttribute('data-theme-value') === 'dark');

  // Verify: Preference=system + Effective=dark: system aria-checked=true, dark aria-checked=false
  assert(systemControl.getAttribute('aria-checked') === 'true', "Picker: system checked");
  assert(darkControl.getAttribute('aria-checked') === 'false', "Picker: dark not checked even when effective is dark");
  assert(lightControl.getAttribute('aria-checked') === 'false', "Picker: light not checked");

  // Click light
  lightControl.click();
  assert(env.window.WorkbenchTheme.getPreference() === 'light', "Picker click light: pref light");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'light', "Picker click light: effective light");
  assert(lightControl.getAttribute('aria-checked') === 'true', "Picker click light: light checked");
  assert(systemControl.getAttribute('aria-checked') === 'false', "Picker click light: system false");
  assert(darkControl.getAttribute('aria-checked') === 'false', "Picker click light: dark false");

  // Click dark
  darkControl.click();
  assert(env.window.WorkbenchTheme.getPreference() === 'dark', "Picker click dark: pref dark");
  assert(env.window.WorkbenchTheme.getEffectiveTheme() === 'dark', "Picker click dark: effective dark");
  assert(darkControl.getAttribute('aria-checked') === 'true', "Picker click dark: dark checked");
  assert(lightControl.getAttribute('aria-checked') === 'false', "Picker click dark: light false");

  // Verify native button has NO redundant keydown handlers registered
  const keydownRegistrations = env.registeredEvents.filter(r => r.event === 'keydown');
  assert(keydownRegistrations.length === 0, "No keydown listeners should be registered on buttons");

  console.log("PASS: Picker Interaction and Keyboard Safety Tests");
})();

console.log("All Runtime and Picker Contract Checks Passed Successfully!");
