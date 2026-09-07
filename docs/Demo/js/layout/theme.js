(function() {
  'use strict';

  var STORAGE_KEY = 'workbench.theme.preference';
  var currentPreference = 'light';

  // Read initial preference safely from storage
  try {
    var stored = localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      currentPreference = stored;
    }
  } catch (e) {
    // Fallback to default compatible preference 'light' on read error
  }

  function getPreference() {
    return currentPreference;
  }

  function getEffectiveTheme(pref) {
    var p = pref || currentPreference;
    if (p === 'dark') return 'dark';
    if (p === 'system') {
      if (typeof window !== 'undefined' && window.matchMedia) {
        try {
          if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
            return 'dark';
          }
        } catch (e) {
          // Fallback to light
        }
      }
    }
    return 'light';
  }

  function applyTheme(effectiveTheme) {
    if (typeof document !== 'undefined' && document.documentElement) {
      if (document.documentElement.setAttribute) {
        document.documentElement.setAttribute('data-theme', effectiveTheme);
      }
      if (document.documentElement.dataset) {
        document.documentElement.dataset.theme = effectiveTheme;
      }
    }
  }

  function syncControls() {
    if (typeof document === 'undefined' || !document.querySelectorAll) return;
    var controls = document.querySelectorAll('[role="menuitemradio"][data-theme-value]');
    for (var i = 0; i < controls.length; i++) {
      var ctrl = controls[i];
      var val = ctrl.getAttribute('data-theme-value');
      if (val === currentPreference) {
        ctrl.setAttribute('aria-checked', 'true');
        if (ctrl.classList) ctrl.classList.add('active');
      } else {
        ctrl.setAttribute('aria-checked', 'false');
        if (ctrl.classList) ctrl.classList.remove('active');
      }
    }
  }

  function setPreference(pref) {
    if (pref !== 'light' && pref !== 'dark' && pref !== 'system') {
      pref = 'light';
    }
    currentPreference = pref;
    try {
      localStorage.setItem(STORAGE_KEY, pref);
    } catch (e) {
      // Storage write error: session stays updated with currentPreference
    }
    var effectiveTheme = getEffectiveTheme(currentPreference);
    applyTheme(effectiveTheme);
    syncControls();
  }

  // System theme listener
  if (typeof window !== 'undefined' && window.matchMedia) {
    try {
      var mql = window.matchMedia('(prefers-color-scheme: dark)');
      var handleMediaChange = function() {
        if (currentPreference === 'system') {
          applyTheme(getEffectiveTheme('system'));
        }
      };
      if (mql && mql.addEventListener) {
        mql.addEventListener('change', handleMediaChange);
      } else if (mql && mql.addListener) {
        mql.addListener(handleMediaChange);
      }
    } catch (e) {}
  }

  // Cross-tab sync via storage event
  if (typeof window !== 'undefined' && window.addEventListener) {
    window.addEventListener('storage', function(e) {
      if (!e || e.key !== STORAGE_KEY) return;
      var val = e.newValue;
      if (val === 'light' || val === 'dark' || val === 'system') {
        currentPreference = val;
      } else {
        currentPreference = 'light';
      }
      applyTheme(getEffectiveTheme(currentPreference));
      syncControls();
    });
  }

  // Apply initial theme immediately before CSS rendering
  applyTheme(getEffectiveTheme(currentPreference));

  // Expose minimal public API
  if (typeof window !== 'undefined') {
    window.WorkbenchTheme = {
      getPreference: getPreference,
      getEffectiveTheme: function() {
        return getEffectiveTheme(currentPreference);
      },
      setPreference: setPreference,
      syncControls: syncControls
    };
  }

  // Bind controls on DOM ready
  function bindControls() {
    syncControls();
    var controls = document.querySelectorAll('[role="menuitemradio"][data-theme-value]');
    for (var i = 0; i < controls.length; i++) {
      controls[i].addEventListener('click', function(e) {
        e.preventDefault();
        var val = this.getAttribute('data-theme-value');
        setPreference(val);
      });
    }
  }

  if (typeof document !== 'undefined') {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', bindControls);
    } else {
      bindControls();
    }
  }
})();
