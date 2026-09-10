// Shared autocomplete item normalization, filtering and row presentation.
(function () {
  "use strict";
  function trimText(value) { return String(value == null ? "" : value).trim(); }

  function normalizeAutocompleteItems(items) {
    var seen = Object.create(null);
    var out = [];
    (items || []).forEach(function (item) {
      var value = trimText(item && item.value);
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
      if (items[i].value === value) {
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

  function formatAutocompleteLabel(item) {
    var label = trimText(item.label);
    var value = trimText(item.value);
    if (!value) {
      return label;
    }
    if (!label || label === value) {
      return value;
    }
    if (label.indexOf(value) >= 0) {
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
      var fields = [item.label, item.value, item.badge];
      if (mode === "user") fields.push(item.dept, item.title, item.selectedLabel || formatAutocompleteLabel(item));
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

  // Pure option presentation; state, selection, portal and events remain in ui.js.
  function renderOption(option, item, displayLabel, mode) {
    if (mode === "user") {
      option.classList.add("ui-autocomplete-option--user");
      var avatar = document.createElement("span");
      var hash = 0, seed = item.value || item.label;
      for (var i = 0; i < seed.length; i++) hash = ((hash * 31) + seed.charCodeAt(i)) >>> 0;
      avatar.className = "ui-autocomplete-avatar";
      avatar.setAttribute("data-tone", String(hash % 4));
      avatar.setAttribute("aria-hidden", "true");
      avatar.textContent = item.avatarText || Array.from(item.label)[0];
      option.appendChild(avatar);
    }
    var content = document.createElement("span");
    content.className = "ui-autocomplete-content";
    var name = document.createElement("span");
    name.className = "ui-autocomplete-main";
    name.textContent = displayLabel;
    if (mode === "user" && item.dept) {
      var dept = document.createElement("span");
      dept.className = "ui-autocomplete-dept";
      dept.textContent = " " + item.dept;
      name.appendChild(dept);
    }
    content.appendChild(name);
    if (mode === "user") {
      var meta = [item.title, item.badge].filter(Boolean).join(" ");
      if (meta) {
        var secondary = document.createElement("span");
        secondary.className = "ui-autocomplete-meta";
        secondary.textContent = meta;
        content.appendChild(secondary);
      }
    }
    option.appendChild(content);
    if (mode !== "user" && item.badge) {
      var badge = document.createElement("span");
      badge.className = "ui-autocomplete-badge";
      badge.textContent = item.badge;
      option.appendChild(badge);
    }
  }

  window.AutocompleteOptions = {
    normalize: normalizeAutocompleteItems, find: findAutocompleteItem,
    ensure: ensureAutocompleteItem, label: formatAutocompleteLabel,
    filter: filterAutocompleteItems, render: renderOption
  };
})();
