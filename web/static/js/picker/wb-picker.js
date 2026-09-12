/* =============================================================================
 * 文件: web/static/workbench/shared/picker/wb-picker.js
 * 职责: shared picker Core。只暴露 WbPicker.mount，不理解 account / deptId。
 * ============================================================================= */
(function (global) {
  'use strict';

  var EMPTY_LAZY_THRESHOLD = 60;
  var SEARCH_DEBOUNCE_MS = 100;

  function esc(s) {
    return WBUtils.escapeHtml(s);
  }
  function arr(v) { return WBUtils.safeArray(v); }
  function text(v) { return String(v == null ? '' : v).trim(); }
  function avatarHtml(option) {
    var meta = (option && option.meta) || {};
    var raw = text(option && option.avatarName) || text(meta.avatarName) || text(meta.realname) || text(option && option.label);
    if (!raw) return '';
    var ch = raw.charAt(0);
    var hue = 0;
    for (var i = 0; i < raw.length; i += 1) hue = (hue * 31 + raw.charCodeAt(i)) & 0xffff;
    var bg = 'hsl(' + (hue % 360) + ', 60%, 86%)';
    var fg = 'hsl(' + (hue % 360) + ', 45%, 30%)';
    return '<span class="wb-picker-avatar" style="background:' + bg + ';color:' + fg + '">' + esc(ch) + '</span>';
  }
  function flat(options) {
    return global.WbPickerSearch && typeof global.WbPickerSearch.flatten === 'function'
      ? global.WbPickerSearch.flatten(options || [])
      : (options || []);
  }
  function optionValue(o) { return text(o && (o.value != null ? o.value : o.id)); }
  function optionLabel(o) { return text(o && (o.label || o.value || o.id)); }
  function compactEmptyOptions(options) {
    var out = [];
    arr(options).forEach(function (g) {
      if (g && g.id === '__preferred') out.push(g);
    });
    out.push({ id: '__lazy_hint', label: '输入姓名、拼音或工号搜索更多', selectable: false, children: [] });
    return out;
  }
  function viewOptions(state) {
    var q = text(state.query);
    if (global.WbPickerSearch && typeof global.WbPickerSearch.filter === 'function') {
      if (!q && flat(state.options).length > EMPTY_LAZY_THRESHOLD) {
        return compactEmptyOptions(state.options);
      }
      return global.WbPickerSearch.filter(state.options, q);
    }
    return state.options;
  }

  function mount(opts) {
    opts = opts || {};
    var host = typeof opts.el === 'string' ? document.querySelector(opts.el) : opts.el;
    if (!host) throw new Error('WbPicker.mount requires el');
    var mode = opts.mode === 'multi' ? 'multi' : 'single';
    var state = {
      options: arr(opts.options),
      value: mode === 'multi' ? arr(opts.value).map(String) : text(opts.value),
      query: '',
      disabled: !!opts.disabled
    };
    host.innerHTML = '';
    host.classList.add('wb-picker');
    var trigger = document.createElement('button');
    trigger.type = 'button';
    trigger.className = 'wb-picker-trigger';
    trigger.disabled = state.disabled;
    trigger.innerHTML = '<span class="wb-picker-label"></span><span class="wb-picker-caret">⌄</span>';
    var panel = document.createElement('div');
    panel.className = 'wb-picker-panel';
    panel.innerHTML =
      '<div class="wb-picker-search-wrap"><input class="wb-picker-search" placeholder="' + esc(opts.placeholder || '搜索') + '"></div>' +
      '<div class="wb-picker-list"></div>' +
      '<div class="wb-picker-footer"><button type="button" class="wb-picker-clear">清空</button></div>';
    host.appendChild(trigger);
    var panelMounted = false;
    var scrollParents = [];
    var renderTimer = null;
    function ensurePanelMounted() {
      if (panelMounted || !document.body) return;
      document.body.appendChild(panel);
      panelMounted = true;
    }
    var labelEl = trigger.querySelector('.wb-picker-label');
    var input = panel.querySelector('.wb-picker-search');
    var list = panel.querySelector('.wb-picker-list');
    var clear = panel.querySelector('.wb-picker-clear');

    function selectedValues() {
      return mode === 'multi' ? state.value.slice() : (state.value ? [state.value] : []);
    }
    function selectedOptions() {
      var values = new Set(selectedValues());
      var seen = new Set();
      var out = [];
      flat(state.options).forEach(function (o) {
        var v = optionValue(o);
        if (!values.has(v) || seen.has(v)) return;
        seen.add(v);
        out.push(o);
      });
      return out;
    }
    function updateLabel() {
      var selected = selectedOptions();
      if (!selected.length) {
        labelEl.textContent = opts.emptyLabel || '请选择';
        labelEl.style.color = 'var(--t3, #98a2b3)';
        return;
      }
      labelEl.style.color = '';
      labelEl.textContent = selected.map(optionLabel).join('、');
    }
    function bindScrollParents() {
      unbindScrollParents();
      var node = host.parentElement;
      while (node && node !== document.body) {
        try {
          var style = window.getComputedStyle(node);
          var oy = style.overflowY;
          if (/(auto|scroll|overlay)/.test(oy) && node.scrollHeight > node.clientHeight + 1) {
            node.addEventListener('scroll', onReposition, { passive: true });
            scrollParents.push(node);
          }
        } catch (e) { /* ignore */ }
        node = node.parentElement;
      }
    }
    function unbindScrollParents() {
      scrollParents.forEach(function (n) { n.removeEventListener('scroll', onReposition); });
      scrollParents = [];
    }
    function measurePanelHeight(maxPanelH) {
      var prevDisplay = panel.style.display;
      var prevVisibility = panel.style.visibility;
      var prevPointer = panel.style.pointerEvents;
      var prevLeft = panel.style.left;
      var prevTop = panel.style.top;
      panel.style.display = 'flex';
      panel.style.visibility = 'hidden';
      panel.style.pointerEvents = 'none';
      panel.style.left = '-99999px';
      panel.style.top = '0';
      var h = panel.offsetHeight || panel.getBoundingClientRect().height || 0;
      panel.style.display = prevDisplay;
      panel.style.visibility = prevVisibility;
      panel.style.pointerEvents = prevPointer;
      panel.style.left = prevLeft;
      panel.style.top = prevTop;
      if (!h) h = Math.min(200, maxPanelH);
      return Math.min(h, maxPanelH);
    }
    function placePanel() {
      if (!host.classList.contains('is-open')) return;
      ensurePanelMounted();
      var rect = trigger.getBoundingClientRect();
      if (!rect.width && !rect.height) return;
      var gap = 4;
      var margin = 8;
      var vw = window.innerWidth || document.documentElement.clientWidth || 0;
      var vh = window.innerHeight || document.documentElement.clientHeight || 0;
      var maxPanelH = Math.min(Number(opts.maxPanelHeight) || 420, Math.max(160, vh - margin * 2));
      var width = Math.max(rect.width, Number(opts.panelWidth) || 0) || 220;
      width = Math.min(Math.max(width, 220), Math.max(180, vw - margin * 2));
      panel.style.width = width + 'px';
      panel.style.maxHeight = maxPanelH + 'px';
      var panelH = measurePanelHeight(maxPanelH);
      var left = rect.left;
      if (left + width > vw - margin) left = vw - margin - width;
      if (left < margin) left = margin;
      var belowTop = rect.bottom + gap;
      var aboveTop = rect.top - gap - panelH;
      var top;
      if (belowTop + panelH <= vh - margin) top = belowTop;
      else if (aboveTop >= margin) top = aboveTop;
      else top = Math.max(margin, Math.min(belowTop, vh - margin - panelH));
      panel.style.left = Math.round(left) + 'px';
      panel.style.top = Math.round(top) + 'px';
    }
    function isSelected(o) {
      var v = optionValue(o);
      return mode === 'multi' ? state.value.indexOf(v) >= 0 : state.value === v;
    }
    function renderOptions(options, depth) {
      depth = depth || 0;
      var html = '';
      arr(options).forEach(function (o) {
        if (!o) return;
        var children = arr(o.children);
        var selectable = o.selectable !== false && optionValue(o);
        if (children.length && !selectable) {
          html += '<div class="wb-picker-group-label" style="padding-left:' + (8 + depth * 12) + 'px">' + esc(optionLabel(o)) + '</div>';
          html += renderOptions(children, depth + 1);
          return;
        }
        var sub = text(o.subLabel || (o.meta && o.meta.subLabel));
        var showAvatar = o.avatar !== false && (o.avatar || (o.meta && (o.meta.realname || o.meta.account)));
        html += '<button type="button" class="wb-picker-option' + (isSelected(o) ? ' is-active' : '') + '" data-value="' + esc(optionValue(o)) + '" style="padding-left:' + (8 + depth * 12) + 'px">' +
          (showAvatar ? avatarHtml(o) : '') +
          '<span class="wb-picker-main"><span class="wb-picker-title">' + esc(optionLabel(o)) + '</span>' +
          (sub ? '<span class="wb-picker-sub">' + esc(sub) + '</span>' : '') + '</span>' +
          '<span class="wb-picker-check">' + (isSelected(o) ? '✓' : '') + '</span></button>';
        if (children.length) html += renderOptions(children, depth + 1);
      });
      return html;
    }
    function renderList() {
      var view = viewOptions(state);
      var html = renderOptions(view, 0);
      list.innerHTML = html || '<div class="wb-picker-empty">无匹配结果</div>';
      list.querySelectorAll('.wb-picker-option').forEach(function (btn) {
        function choose(e) {
          if (e && typeof e.preventDefault === 'function') e.preventDefault();
          if (btn.__wbPickerChosen) return;
          btn.__wbPickerChosen = true;
          setTimeout(function () { btn.__wbPickerChosen = false; }, 0);
          var value = btn.getAttribute('data-value') || '';
          if (mode === 'multi') {
            var idx = state.value.indexOf(value);
            if (idx >= 0) state.value.splice(idx, 1);
            else state.value.push(value);
          } else {
            state.value = value;
            close();
          }
          updateLabel();
          renderList();
          if (typeof opts.onChange === 'function') opts.onChange(api.getValue(), selectedOptions());
        }
        btn.addEventListener('mousedown', choose);
        btn.addEventListener('click', choose);
      });
    }
    function scheduleRenderList(reposition) {
      if (renderTimer) clearTimeout(renderTimer);
      var delay = state.query ? SEARCH_DEBOUNCE_MS : 0;
      renderTimer = setTimeout(function () {
        renderTimer = null;
        renderList();
        if (reposition) placePanel();
      }, delay);
    }
    function focusSearchInput() {
      if (!input || !input.focus) return;
      try { input.focus({ preventScroll: true }); } catch (e) { input.focus(); }
    }
    function open() {
      if (state.disabled) return;
      host.classList.add('is-open');
      panel.classList.add('is-open');
      bindScrollParents();
      renderList();
      placePanel();
      requestAnimationFrame(function () {
        placePanel();
        focusSearchInput();
      });
    }
    function close() {
      host.classList.remove('is-open');
      panel.classList.remove('is-open');
      unbindScrollParents();
      if (renderTimer) {
        clearTimeout(renderTimer);
        renderTimer = null;
      }
    }

    trigger.addEventListener('click', function (e) {
      e.preventDefault();
      if (host.classList.contains('is-open')) close();
      else open();
    });
    input.addEventListener('input', function () {
      state.query = input.value || '';
      scheduleRenderList(true);
    });
    clear.addEventListener('click', function () {
      state.value = mode === 'multi' ? [] : '';
      updateLabel();
      renderList();
      if (typeof opts.onChange === 'function') opts.onChange(api.getValue(), selectedOptions());
      if (mode === 'single') close();
    });
    function onDocMouseDown(e) {
      if (!host.contains(e.target) && !panel.contains(e.target)) close();
    }
    function onReposition() {
      if (host.classList.contains('is-open')) placePanel();
    }
    document.addEventListener('mousedown', onDocMouseDown);
    window.addEventListener('resize', onReposition);
    window.addEventListener('scroll', onReposition, true);

    var api = {
      getValue: function () { return mode === 'multi' ? state.value.slice() : state.value; },
      setValue: function (value) {
        state.value = mode === 'multi' ? arr(value).map(String) : text(value);
        updateLabel();
        if (host.classList.contains('is-open')) renderList();
      },
      setOptions: function (options) {
        state.options = arr(options);
        updateLabel();
        if (host.classList.contains('is-open')) renderList();
      },
      setDisabled: function (disabled) {
        state.disabled = !!disabled;
        trigger.disabled = state.disabled;
        if (state.disabled) close();
      },
      destroy: function () {
        close();
        document.removeEventListener('mousedown', onDocMouseDown);
        window.removeEventListener('resize', onReposition);
        window.removeEventListener('scroll', onReposition, true);
        unbindScrollParents();
        if (panel.parentNode) panel.parentNode.removeChild(panel);
        panelMounted = false;
        host.innerHTML = '';
        host.classList.remove('wb-picker', 'is-open');
      }
    };
    updateLabel();
    renderList();
    return api;
  }

  global.WbPicker = { mount: mount };
})(typeof window !== 'undefined' ? window : globalThis);
