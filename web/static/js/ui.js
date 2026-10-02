/*
 * UI 交互层脚本：仅负责视觉交互与通用组件行为。
 */
(function () {
  "use strict";

  var pendingDeleteUrl = "";

  // escapeHtml 通用 HTML 转义（文本与属性值均可用）；PO 工作台各页面脚本通过 window.escapeHtml 复用。
  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function getCsrfToken() {
    var el = document.querySelector('meta[name="csrf-token"]');
    return el ? (el.getAttribute("content") || "").trim() : "";
  }

  function removeToast(toast, timerId) {
    if (timerId) {
      window.clearTimeout(timerId);
    }
    if (toast && toast.parentNode) {
      toast.remove();
    }
  }

  function showToast(message, type) {
    var text = (message || "").trim();
    if (!text) {
      return;
    }
    var level = type || "success";
    if (level === "danger") {
      level = "error";
    }

    var container = document.getElementById("toast-container");
    if (!container) {
      container = document.createElement("div");
      container.id = "toast-container";
      container.className = "toast-container";
      document.body.appendChild(container);
    }

    var toast = document.createElement("div");
    toast.className = "toast toast-" + level;
    toast.setAttribute("role", "alert");

    var textNode = document.createElement("span");
    textNode.className = "toast-message";
    textNode.textContent = text;
    toast.appendChild(textNode);

    var closeBtn = document.createElement("button");
    closeBtn.type = "button";
    closeBtn.className = "toast-close";
    closeBtn.setAttribute("aria-label", "关闭");
    closeBtn.textContent = "✕";
    toast.appendChild(closeBtn);

    container.appendChild(toast);

    var timerId = window.setTimeout(function () {
      removeToast(toast);
    }, 5000);

    closeBtn.onclick = function () {
      removeToast(toast, timerId);
    };
  }

  function confirmDelete(triggerEl, name, deleteUrl) {
    var modal = document.getElementById("modal-delete");
    if (!modal) {
      return false;
    }
    pendingDeleteUrl = (deleteUrl || "").trim();
    if (!pendingDeleteUrl) {
      pendingDeleteUrl = triggerEl ? (triggerEl.getAttribute("data-url") || triggerEl.getAttribute("href") || "").trim() : "";
    }
    var displayName = (name || "").trim();
    if (!displayName && triggerEl && triggerEl.getAttribute) {
      displayName = (triggerEl.getAttribute("data-name") || "").trim();
    }
    var title = modal.querySelector(".modal-title");
    if (title) {
      title.textContent = "确认删除「" + displayName + "」？";
    }
    openShowModals("modal-delete");
    return false;
  }

  /**
   * 切换带 .show 类名的弹窗与遮罩（schedule 等模块使用）。
   * @param {string|string[]} elementIds 弹窗/遮罩元素 id，可传数组
   * @param {boolean} visible 是否显示
   */
  function setShowModals(elementIds, visible) {
    var ids = Array.isArray(elementIds) ? elementIds : [elementIds];
    for (var i = 0; i < ids.length; i++) {
      var el = document.getElementById(ids[i]);
      if (!el) {
        continue;
      }
      el.classList.toggle("show", visible);
    }
  }

  function openShowModals(elementIds) {
    setShowModals(elementIds, true);
  }

  function closeShowModals(elementIds) {
    setShowModals(elementIds, false);
  }

  /**
   * 切换下拉：点击触发器时切换父级 .dropdown 的 .open；点击页面其他区域关闭所有已打开的下拉。
   * @param {Element} el 触发器或其子节点（在 .dropdown 内）
   * @returns {boolean} false（便于内联 onclick 阻止默认行为）
   */
  function toggleDropdown(el) {
    if (!el || !el.closest) {
      return false;
    }
    var dropdown = el.closest(".dropdown");
    if (!dropdown) {
      return false;
    }
    var willOpen = !dropdown.classList.contains("open");
    var opened = document.querySelectorAll(".dropdown.open");
    for (var i = 0; i < opened.length; i++) {
      opened[i].classList.remove("open");
    }
    if (willOpen) {
      dropdown.classList.add("open");
    }
    return false;
  }

  function closeAllDropdowns() {
    var opened = document.querySelectorAll(".dropdown.open");
    for (var i = 0; i < opened.length; i++) {
      opened[i].classList.remove("open");
    }
  }

  var autocompleteInstances = {};

  function trimText(value) {
    return (value || "").trim();
  }

  function normalizeAutocompleteItems(items) {
    var seen = {};
    var out = [];
    (items || []).forEach(function (item) {
      if (!item) return;
      if (item.isGroupHeader) {
        out.push({
          label: trimText(item.label),
          value: "",
          isGroupHeader: true
        });
        return;
      }
      var value = trimText(item.value);
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      out.push({
        value: value,
        label: trimText(item.label) || value,
        selectedLabel: trimText(item.selectedLabel),
        dept: trimText(item.dept),
        title: trimText(item.title),
        avatarText: trimText(item.avatarText),
        badge: trimText(item.badge),
        pinyin: trimText(item.pinyin)
      });
    });
    return out;
  }

  function findAutocompleteItem(items, value) {
    value = trimText(value);
    if (!value) {
      return null;
    }
    for (var i = 0; i < items.length; i++) {
      if (!items[i].isGroupHeader && items[i].value === value) {
        return items[i];
      }
    }
    return null;
  }

  function ensureAutocompleteItem(items, value, label) {
    value = trimText(value);
    if (!value) {
      return items;
    }
    if (findAutocompleteItem(items, value)) {
      return items;
    }
    return items.concat([
      {
        value: value,
        label: trimText(label) || value,
      },
    ]);
  }

  function formatAutocompleteLabel(item, labelOnly) {
    var label = trimText(item.label);
    var value = trimText(item.value);
    if (!value) {
      return label;
    }
    if (!label || label === value) {
      return value;
    }
    if (labelOnly || label.indexOf(value) >= 0) {
      return label;
    }
    return label + "(" + value + ")";
  }

  function filterAutocompleteItems(items, query, maxShow, mode) {
    query = trimText(query).toLowerCase();
    if (!query) {
      var total = items.length;
      return {
        items: items.slice(0, maxShow),
        showHint: total > maxShow,
        remainingCount: total > maxShow ? total - maxShow : 0,
        hasQuery: false,
      };
    }

    var matches = [];
    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      if (item.isGroupHeader) {
        continue;
      }
      var fields = [item.label, item.value];
      if (item.badge) fields.push(item.badge);
      if (item.pinyin) fields.push(item.pinyin);
      if (mode === "user") {
        if (item.dept) fields.push(item.dept);
        if (item.title) fields.push(item.title);
        if (item.selectedLabel) fields.push(item.selectedLabel);
        if (window.PinyinLite) {
          if (typeof window.PinyinLite.fullPinyin === "function") fields.push(window.PinyinLite.fullPinyin(item.label));
          if (typeof window.PinyinLite.initials === "function") fields.push(window.PinyinLite.initials(item.label));
        }
      }
      var haystack = fields.join(" ").toLowerCase();
      if (haystack.indexOf(query) === -1) {
        continue;
      }
      matches.push(item);
    }

    return {
      items: matches.slice(0, maxShow),
      showHint: matches.length > maxShow,
      remainingCount: matches.length > maxShow ? matches.length - maxShow : 0,
      hasQuery: true,
    };
  }

  function updateAutocompleteClearButton(state) {
    if (!state.clearBtn) {
      return;
    }
    var hasValue = !!trimText(state.input.value) || !!trimText(state.hidden.value);
    state.clearBtn.classList.toggle("is-visible", hasValue);
  }

  function closeAutocomplete(state) {
    if (!state || !state.dropdown) {
      return;
    }
    state.dropdown.classList.remove("is-open");
    state.dropdown.innerHTML = "";
    state.activeIndex = -1;
    state.open = false;
  }

  function resolveAutocompleteDropdownPortal() {
    return document.body;
  }

  function positionAutocompleteDropdown(state) {
    var input = state.input;
    var dropdown = state.dropdown;
    if (!input || !dropdown) {
      return;
    }
    var rect = input.getBoundingClientRect();
    dropdown.style.position = "fixed";
    dropdown.style.left = rect.left + "px";
    dropdown.style.top = rect.bottom + "px";
    dropdown.style.width = rect.width + "px";
    dropdown.style.zIndex = "99999";
  }

  function ensureAutocompleteDropdown(input, inputId) {
    var wrap = input.closest(".ui-autocomplete-input-wrap");
    if (wrap) {
      var legacy = wrap.querySelector(".ui-autocomplete-dropdown");
      if (legacy) {
        legacy.remove();
      }
    }

    var portal = resolveAutocompleteDropdownPortal();
    var dropdown = portal.querySelector('.ui-autocomplete-dropdown[data-autocomplete-for="' + inputId + '"]');
    if (!dropdown) {
      dropdown = document.createElement("div");
      dropdown.className = "ui-autocomplete-dropdown ui-autocomplete-dropdown--floating";
      dropdown.setAttribute("role", "listbox");
      dropdown.setAttribute("data-autocomplete-for", inputId);
      dropdown.addEventListener("mousedown", function (ev) {
        ev.stopPropagation();
      });
      portal.appendChild(dropdown);
    }
    return dropdown;
  }

  function isModalScrollTarget(target) {
    if (!target || !target.closest) {
      return false;
    }
    return !!target.closest(".batch-modal-body, .batch-modal, .modal, .drawer-body");
  }

  function bindAutocompleteReposition(state) {
    if (state.repositionBound) {
      return;
    }
    state.repositionBound = true;
    state.repositionHandler = function () {
      if (state.open) {
        positionAutocompleteDropdown(state);
      }
    };
    state.modalScrollHandler = function (ev) {
      if (!state.open) {
        return;
      }
      if (isModalScrollTarget(ev.target)) {
        closeAutocomplete(state);
      }
    };
    window.addEventListener("resize", state.repositionHandler);
    document.addEventListener("scroll", state.modalScrollHandler, true);
  }

  function unbindAutocompleteReposition(state) {
    if (!state || !state.repositionBound) {
      return;
    }
    if (state.repositionHandler) {
      window.removeEventListener("resize", state.repositionHandler);
    }
    if (state.modalScrollHandler) {
      document.removeEventListener("scroll", state.modalScrollHandler, true);
    }
    state.repositionBound = false;
    state.repositionHandler = null;
    state.modalScrollHandler = null;
  }

  function closeAllAutocompletes(exceptInputId) {
    Object.keys(autocompleteInstances).forEach(function (inputId) {
      if (exceptInputId && inputId === exceptInputId) {
        return;
      }
      closeAutocomplete(autocompleteInstances[inputId]);
    });
  }

  function renderAutocompleteDropdown(state) {
    var query = state.input.value;
    var selectedValue = trimText(state.hidden.value);
    if (selectedValue) {
      var selectedItem = findAutocompleteItem(state.items, selectedValue);
      if (selectedItem && trimText(query) === (selectedItem.selectedLabel || formatAutocompleteLabel(selectedItem, state.labelOnly))) {
        query = "";
      }
    }
    var result = filterAutocompleteItems(state.items, query, state.maxShow, state.mode);
    var matches = result.items;
    var dropdown = state.dropdown;

    dropdown.innerHTML = "";
    state.activeIndex = -1;

    if (!matches.length) {
      var empty = document.createElement("div");
      empty.className = "ui-autocomplete-empty";
      empty.textContent = result.hasQuery ? "无匹配结果" : (state.mode === "user" ? "暂无可选用户" : "暂无可用数据");
      dropdown.appendChild(empty);
      dropdown.classList.add("is-open");
      state.open = true;
      state.filteredItems = [];
      positionAutocompleteDropdown(state);
      return;
    }

    var selectableIndex = 0;
    var selectableItems = [];
    matches.forEach(function (item) {
      if (item.isGroupHeader) {
        var groupHeader = document.createElement("div");
        groupHeader.className = "ui-autocomplete-group-header";
        groupHeader.textContent = item.label;
        dropdown.appendChild(groupHeader);
        return;
      }

      var currentIndex = selectableIndex++;
      selectableItems.push(item);
      var displayLabel = formatAutocompleteLabel(item, state.labelOnly);
      var selectLabel = item.selectedLabel || displayLabel;
      var option = document.createElement("div");
      option.className = "ui-autocomplete-option";
      if (isAutocompleteValueSelected(state, item.value)) {
        option.classList.add("is-selected");
      }
      option.setAttribute("role", "option");
      option.setAttribute("data-index", String(currentIndex));
      option.setAttribute("data-value", item.value);
      option.setAttribute("data-label", displayLabel);

      if (state.mode === "user") {
        option.classList.add("ui-autocomplete-option--user");
        var avatar = document.createElement("span");
        var hash = 0, seed = item.value || item.label;
        for (var h = 0; h < seed.length; h++) hash = ((hash * 31) + seed.charCodeAt(h)) >>> 0;
        avatar.className = "ui-autocomplete-avatar";
        avatar.setAttribute("data-tone", String(hash % 4));
        avatar.setAttribute("aria-hidden", "true");
        avatar.textContent = item.avatarText || Array.from(item.label || "?")[0];
        option.appendChild(avatar);

        var content = document.createElement("span");
        content.className = "ui-autocomplete-content";
        var name = document.createElement("span");
        name.className = "ui-autocomplete-main";
        name.textContent = displayLabel;
        if (item.dept) {
          var dept = document.createElement("span");
          dept.className = "ui-autocomplete-dept";
          dept.textContent = " " + item.dept;
          name.appendChild(dept);
        }
        content.appendChild(name);
        var meta = [item.title, item.badge].filter(Boolean).join(" ");
        if (meta) {
          var secondary = document.createElement("span");
          secondary.className = "ui-autocomplete-meta";
          secondary.textContent = meta;
          content.appendChild(secondary);
        }
        option.appendChild(content);
      } else {
        option.textContent = displayLabel;
        if (item.badge) {
          var badge = document.createElement("span");
          badge.className = "ui-autocomplete-badge";
          badge.textContent = item.badge;
          option.appendChild(badge);
        }
      }

      option.addEventListener("mousedown", function (ev) {
        ev.preventDefault();
      });
      option.addEventListener("click", function () {
        selectAutocompleteItem(state, item.value, selectLabel);
      });
      dropdown.appendChild(option);
    });

    if (result.showHint) {
      var hint = document.createElement("div");
      hint.className = "ui-autocomplete-hint";
      hint.textContent = "还有 " + result.remainingCount + (state.mode === "user"
        ? " 个选项没有显示，可尝试搜索来查找" : " 条，输入关键词缩小范围");
      dropdown.appendChild(hint);
    }

    dropdown.classList.add("is-open");
    state.open = true;
    state.filteredItems = selectableItems;
    positionAutocompleteDropdown(state);

    if (selectedValue) {
      for (var i = 0; i < selectableItems.length; i++) {
        if (selectableItems[i].value === selectedValue) {
          setActiveAutocompleteOption(state, i);
          break;
        }
      }
    }
  }

  function setActiveAutocompleteOption(state, index) {
    var options = state.dropdown.querySelectorAll(".ui-autocomplete-option");
    if (!options.length) {
      state.activeIndex = -1;
      return;
    }
    if (index < 0) {
      index = options.length - 1;
    }
    if (index >= options.length) {
      index = 0;
    }
    state.activeIndex = index;
    for (var i = 0; i < options.length; i++) {
      options[i].classList.toggle("is-active", i === index);
    }
    options[index].scrollIntoView({ block: "nearest" });
  }

  function selectAutocompleteItem(state, value, label) {
    value = trimText(value);
    label = trimText(label) || value;
    state.hidden.value = value;
    state.input.value = value ? label : "";
    state.items = ensureAutocompleteItem(state.items, value, label);
    updateAutocompleteClearButton(state);
    closeAutocomplete(state);
    try {
      state.hidden.dispatchEvent(new Event("change", { bubbles: true }));
    } catch (e) {
      /* ignore */
    }
  }

  function clearAutocompleteValue(state) {
    state.input.value = "";
    state.hidden.value = "";
    updateAutocompleteClearButton(state);
    closeAutocomplete(state);
    try {
      state.hidden.dispatchEvent(new Event("change", { bubbles: true }));
    } catch (e) {
      /* ignore */
    }
  }

  function ensureAutocompleteStructure(input, hidden) {
    var host = input.parentElement;
    if (!host) {
      return null;
    }
    host.classList.add("ui-autocomplete");

    var wrap = input.parentElement;
    if (!wrap.classList.contains("ui-autocomplete-input-wrap")) {
      wrap = document.createElement("div");
      wrap.className = "ui-autocomplete-input-wrap";
      host.insertBefore(wrap, input);
      wrap.appendChild(input);
    }
    input.classList.add("ui-autocomplete-input");

    var clearBtn = wrap.querySelector(".ui-autocomplete-clear");
    if (!clearBtn) {
      clearBtn = document.createElement("button");
      clearBtn.type = "button";
      clearBtn.className = "ui-autocomplete-clear";
      clearBtn.setAttribute("aria-label", "清除");
      clearBtn.innerHTML =
        '<svg viewBox="0 0 10 10" aria-hidden="true">' +
        '<path d="M2 2l6 6M8 2L2 8" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"></path>' +
        "</svg>";
      wrap.appendChild(clearBtn);
    }

    if (hidden.parentElement !== host) {
      host.appendChild(hidden);
    }

    return {
      host: host,
      wrap: wrap,
      clearBtn: clearBtn,
    };
  }

  function openAutocompleteDropdown(state) {
    closeAllAutocompletes(state.inputId);
    renderAutocompleteDropdown(state);
  }

  function bindAutocompleteEvents(state) {
    if (state.bound) {
      return;
    }
    state.bound = true;

    state.host.addEventListener("mousedown", function (ev) {
      ev.stopPropagation();
    });

    state.input.addEventListener("focus", function () {
      positionAutocompleteDropdown(state);
      openAutocompleteDropdown(state);
    });

    state.input.addEventListener("mousedown", function () {
      openAutocompleteDropdown(state);
    });

    state.input.addEventListener("input", function () {
      state.hidden.value = "";
      updateAutocompleteClearButton(state);
      renderAutocompleteDropdown(state);
    });

    state.input.addEventListener("keydown", function (ev) {
      if (!state.open) {
        if (ev.key === "ArrowDown" || ev.key === "ArrowUp") {
          openAutocompleteDropdown(state);
        }
        return;
      }

      var options = state.dropdown.querySelectorAll(".ui-autocomplete-option");
      if (!options.length) {
        if (ev.key === "Escape") {
          closeAutocomplete(state);
        }
        return;
      }

      if (ev.key === "ArrowDown") {
        ev.preventDefault();
        setActiveAutocompleteOption(state, state.activeIndex + 1);
        return;
      }
      if (ev.key === "ArrowUp") {
        ev.preventDefault();
        setActiveAutocompleteOption(state, state.activeIndex - 1);
        return;
      }
      if (ev.key === "Enter") {
        if (state.activeIndex >= 0 && state.filteredItems[state.activeIndex]) {
          ev.preventDefault();
          var picked = state.filteredItems[state.activeIndex];
          var selectLabel = picked.selectedLabel || formatAutocompleteLabel(picked, state.labelOnly);
          selectAutocompleteItem(state, picked.value, selectLabel);
        }
        return;
      }
      if (ev.key === "Escape") {
        ev.preventDefault();
        closeAutocomplete(state);
      }
    });

    state.clearBtn.addEventListener("click", function (ev) {
      ev.preventDefault();
      ev.stopPropagation();
      clearAutocompleteValue(state);
      state.input.focus();
    });
  }

  function normalizeAutocompleteSelectedValues(values) {
    if (!Array.isArray(values)) {
      return [];
    }
    var out = [];
    var seen = {};
    values.forEach(function (raw) {
      var value = trimText(raw);
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      out.push(value);
    });
    return out;
  }

  function isAutocompleteValueSelected(state, value) {
    value = trimText(value);
    if (!value || !state || !Array.isArray(state.selectedValues)) {
      return false;
    }
    return state.selectedValues.indexOf(value) >= 0;
  }

  function markSelectedAutocompleteOptions(state) {
    if (!state || !state.dropdown) {
      return;
    }
    var options = state.dropdown.querySelectorAll(".ui-autocomplete-option");
    for (var i = 0; i < options.length; i++) {
      var value = options[i].getAttribute("data-value") || "";
      options[i].classList.toggle("is-selected", isAutocompleteValueSelected(state, value));
    }
  }

  function setAutocompleteSelectedValues(inputId, values) {
    var state = autocompleteInstances[inputId];
    if (!state) {
      return;
    }
    state.selectedValues = normalizeAutocompleteSelectedValues(values);
    markSelectedAutocompleteOptions(state);
  }

  function createAutocompleteState(input, hidden, inputId) {
    var structure = ensureAutocompleteStructure(input, hidden);
    if (!structure) {
      return null;
    }
    var dropdown = ensureAutocompleteDropdown(input, inputId);

    return {
      inputId: inputId,
      input: input,
      hidden: hidden,
      host: structure.host,
      clearBtn: structure.clearBtn,
      dropdown: dropdown,
      mode: "",
      items: [],
      filteredItems: [],
      selectedValues: [],
      maxShow: 100,
      labelOnly: false,
      activeIndex: -1,
      open: false,
      bound: false,
      repositionBound: false,
      repositionHandler: null,
      modalScrollHandler: null,
    };
  }

  /**
   * 初始化自动补全输入框。
   * @param {string} inputId 可见输入框 ID
   * @param {string} hiddenId 隐藏字段 ID（存 value）
   * @param {Array<{value:string,label:string}>} items 选项列表
   * @param {{placeholder?:string,maxShow?:number,value?:string,label?:string,labelOnly?:boolean,selectedValues?:string[]}} options
   */
  function initAutocomplete(inputId, hiddenId, items, options) {
    var input = document.getElementById(inputId);
    var hidden = document.getElementById(hiddenId);
    if (!input || !hidden) {
      return;
    }

    options = options || {};
    var state = autocompleteInstances[inputId];
    if (!state || state.input !== input || state.hidden !== hidden) {
      if (state) {
        unbindAutocompleteReposition(state);
        if (state.dropdown && state.dropdown.parentElement) {
          state.dropdown.remove();
        }
      }
      state = createAutocompleteState(input, hidden, inputId);
      if (!state) {
        return;
      }
      autocompleteInstances[inputId] = state;
      bindAutocompleteEvents(state);
      bindAutocompleteReposition(state);
    }

    state.mode = options.mode || "";
    if (state.dropdown) {
      state.dropdown.classList.toggle("ui-autocomplete-dropdown--user", state.mode === "user");
    }
    state.items = normalizeAutocompleteItems(items);
    state.selectedValues = normalizeAutocompleteSelectedValues(options.selectedValues);
    state.maxShow = options.maxShow > 0 ? options.maxShow : (state.mode === "user" ? 20 : 100);
    state.labelOnly = !!options.labelOnly;
    if (options.placeholder) {
      state.input.placeholder = options.placeholder;
    }

    if (Object.prototype.hasOwnProperty.call(options, "value")) {
      if (trimText(options.value)) {
        state.items = ensureAutocompleteItem(state.items, options.value, options.label);
        var selectedItem = findAutocompleteItem(state.items, options.value);
        var displayLabel = selectedItem
          ? (selectedItem.selectedLabel || formatAutocompleteLabel(selectedItem, state.labelOnly))
          : formatAutocompleteLabel({ value: options.value, label: options.label }, state.labelOnly);
        selectAutocompleteItem(state, options.value, displayLabel);
      } else {
        clearAutocompleteValue(state);
      }
    } else {
      updateAutocompleteClearButton(state);
    }
  }

  function clearAutocomplete(inputId) {
    var state = autocompleteInstances[inputId];
    if (!state) {
      return;
    }
    clearAutocompleteValue(state);
  }

  function destroyAutocomplete(inputId) {
    var state = autocompleteInstances[inputId];
    if (!state) {
      return;
    }
    closeAutocomplete(state);
    unbindAutocompleteReposition(state);
    if (state.dropdown && state.dropdown.parentElement) {
      state.dropdown.remove();
    }
    delete autocompleteInstances[inputId];
  }

  document.addEventListener("click", function (ev) {
    var t = ev.target;
    if (!t || !t.closest) {
      return;
    }
    if (t.closest(".dropdown")) {
      return;
    }
    closeAllDropdowns();
    if (!t.closest(".ui-autocomplete") && !t.closest(".ui-autocomplete-dropdown")) {
      closeAllAutocompletes();
    }
  });

  function submitDelete() {
    if (!pendingDeleteUrl) {
      showToast("删除地址无效，请刷新后重试", "error");
      closeShowModals("modal-delete");
      return false;
    }
    window.appJson(pendingDeleteUrl, { method: "DELETE", body: {} })
      .then(function (data) {
        if (!data || !data.success) throw new Error((data && data.message) || "删除失败");
        closeShowModals("modal-delete");
        showToast(data.message || "删除成功", "success");
        if (data.redirectUrl) window.location.href = data.redirectUrl;
        else window.setTimeout(function () { window.location.reload(); }, 250);
      })
      .catch(function (err) { showToast(err.message || "删除失败，请稍后重试", "error"); });
    return false;
  }

  window.showToast = showToast;
  window.escapeHtml = escapeHtml;
  window.confirmDelete = confirmDelete;
  window.openShowModals = openShowModals;
  window.closeShowModals = closeShowModals;
  window.submitDelete = submitDelete;
  window.toggleDropdown = toggleDropdown;
  window.closeAllDropdowns = closeAllDropdowns;
  window.initAutocomplete = initAutocomplete;
  window.clearAutocomplete = clearAutocomplete;
  window.destroyAutocomplete = destroyAutocomplete;
  window.setAutocompleteSelectedValues = setAutocompleteSelectedValues;
  window.getCsrfToken = getCsrfToken;
  // 人员选择器：PO 澄清 / 提交评审 / 提测 / 敏捷小组成员等页面按此名调用；
  // 支持 mode:"user" 模式（首字彩色头像、部门与角色标签、拼音与姓名工号检索、分组标题）。
  window.initUserPicker = function (inputId, hiddenId, items, opts) {
    return initAutocomplete(inputId, hiddenId, items, Object.assign({}, opts, { mode: "user" }));
  };
  window.destroyUserPicker = destroyAutocomplete;
})();
