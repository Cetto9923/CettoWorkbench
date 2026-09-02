/*
 * 表单组件脚本：角色多选下拉、树形单选下拉等。
 * 多选：根节点可设 data-multiselect-keep-placeholder="true"，有已选项时仍显示检索占位（与属性默认值「标签+框内搜索」一致）。
 * 依赖：ui.js（window.toggleDropdown、window.closeAllDropdowns）。
 */
(function () {
  "use strict";

  function openDropdownOnly(dropdown) {
    if (!dropdown) {
      return;
    }
    if (typeof window.closeAllDropdowns === "function") {
      window.closeAllDropdowns();
    }
    dropdown.classList.add("open");
  }

  function initFormRoleMultiselect(dropdown) {
    if (!dropdown || dropdown.dataset.formMultiselectInit === "1") {
      return;
    }
    dropdown.dataset.formMultiselectInit = "1";

    var tagsWrap = dropdown.querySelector("[data-role-tags]");
    var filterInput = dropdown.querySelector("[data-multiselect-filter]");
    var display = dropdown.querySelector(".form-multiselect-display");
    var multiselectClearBtn = dropdown.querySelector(
      "[data-form-multiselect-clear]"
    );
    var checkboxes = dropdown.querySelectorAll(".form-multiselect-checkbox");
    if (!tagsWrap || !filterInput) {
      return;
    }
    var emptyPlaceholder = (
      filterInput.getAttribute("data-placeholder-empty") ||
      filterInput.getAttribute("placeholder") ||
      "请选择"
    ).trim();

    function getSelectedRoles() {
      var selected = [];
      for (var i = 0; i < checkboxes.length; i++) {
        if (!checkboxes[i].checked) {
          continue;
        }
        var nameEl = checkboxes[i]
          .closest(".form-multiselect-option")
          .querySelector(".form-multiselect-name");
        selected.push({
          value: checkboxes[i].value,
          name: nameEl ? nameEl.textContent.trim() : "",
        });
      }
      return selected;
    }

    function filterRoleOptions(keyword) {
      var kw = (keyword || "").trim().toLowerCase();
      var opts = dropdown.querySelectorAll(".form-multiselect-option");
      for (var i = 0; i < opts.length; i++) {
        var nameEl = opts[i].querySelector(".form-multiselect-name");
        var t = nameEl ? nameEl.textContent.trim().toLowerCase() : "";
        opts[i].style.display = !kw || t.indexOf(kw) !== -1 ? "" : "none";
      }
    }

    function syncMultiselectClearVis() {
      if (!multiselectClearBtn) {
        return;
      }
      var hasSel = getSelectedRoles().length > 0;
      var hasFilter = (filterInput.value || "").trim().length > 0;
      multiselectClearBtn.hidden = !hasSel && !hasFilter;
    }

    function renderSelectedTags() {
      var selected = getSelectedRoles();
      tagsWrap.innerHTML = "";
      if (selected.length === 0) {
        return;
      }
      for (var i = 0; i < selected.length; i++) {
        var item = selected[i];
        var tag = document.createElement("span");
        tag.className = "form-multiselect-tag";

        var name = document.createElement("span");
        name.className = "form-multiselect-tag-name";
        name.textContent = item.name;
        tag.appendChild(name);

        var remove = document.createElement("button");
        remove.type = "button";
        remove.className = "form-multiselect-tag-remove";
        remove.setAttribute("data-role-remove", item.value);
        remove.setAttribute("aria-label", "移除 " + item.name);
        remove.textContent = "×";
        tag.appendChild(remove);
        tagsWrap.appendChild(tag);
      }
    }

    function syncRoleInputState() {
      var selected = getSelectedRoles();
      renderSelectedTags();
      var keepPlaceholder =
        dropdown.getAttribute("data-multiselect-keep-placeholder") === "true" ||
        dropdown.getAttribute("data-multiselect-keep-placeholder") === "1";
      filterInput.setAttribute(
        "placeholder",
        keepPlaceholder || selected.length === 0 ? emptyPlaceholder : ""
      );
      syncMultiselectClearVis();
      filterRoleOptions(filterInput.value || "");
    }

    for (var i = 0; i < checkboxes.length; i++) {
      checkboxes[i].addEventListener("change", syncRoleInputState);
    }

    tagsWrap.addEventListener("click", function (event) {
      var target = event.target;
      if (!target || !target.closest) {
        return;
      }
      var removeEl = target.closest("[data-role-remove]");
      if (!removeEl) {
        return;
      }
      var value = removeEl.getAttribute("data-role-remove");
      for (var i = 0; i < checkboxes.length; i++) {
        if (checkboxes[i].value === value) {
          checkboxes[i].checked = false;
          break;
        }
      }
      syncRoleInputState();
      event.stopPropagation();
    });

    filterInput.addEventListener("input", function () {
      filterRoleOptions(filterInput.value || "");
      syncMultiselectClearVis();
    });
    filterInput.addEventListener("focus", function () {
      openDropdownOnly(dropdown);
      filterRoleOptions(filterInput.value || "");
    });
    filterInput.addEventListener("click", function () {
      openDropdownOnly(dropdown);
    });
    filterInput.addEventListener("blur", function () {
      filterInput.value = "";
      filterRoleOptions("");
      syncMultiselectClearVis();
    });
    filterInput.addEventListener("keydown", function (event) {
      if (event.key === "Enter") {
        event.preventDefault();
        if (typeof window.toggleDropdown === "function") {
          window.toggleDropdown(filterInput);
        }
      }
    });

    if (display) {
      display.addEventListener("click", function (event) {
        var onRemove = event.target.closest(
          ".form-multiselect-tag-remove, [data-role-remove]"
        );
        if (onRemove) {
          return;
        }
        if (event.target.closest(".form-multiselect-input")) {
          return;
        }
        filterInput.focus();
        openDropdownOnly(dropdown);
        filterRoleOptions(filterInput.value || "");
      });
    }

    if (multiselectClearBtn) {
      multiselectClearBtn.addEventListener("click", function (event) {
        event.preventDefault();
        event.stopPropagation();
        if (getSelectedRoles().length > 0) {
          for (var i = 0; i < checkboxes.length; i++) {
            checkboxes[i].checked = false;
          }
          syncRoleInputState();
          return;
        }
        filterInput.value = "";
        filterRoleOptions("");
        syncMultiselectClearVis();
      });
    }

    syncRoleInputState();
    filterRoleOptions("");
    syncMultiselectClearVis();
  }

  function treeEmptyLabel(selectEl) {
    var opts = selectEl.options;
    for (var i = 0; i < opts.length; i++) {
      if (!(opts[i].value || "").trim()) {
        var t = opts[i].textContent.trim();
        return t || "请选择";
      }
    }
    return "请选择";
  }

  function initFormTreeSelect(field) {
    if (!field || field.dataset.formTreeSelectInit === "1") {
      return;
    }
    var dataEl = field.querySelector(".form-tree-select-data");
    var deptSelect = field.querySelector("select.form-tree-select-native");
    var treeList = field.querySelector(
      ".form-tree-select-scroll > ul.form-tree-select-list"
    );
    var treeInput = field.querySelector("[data-form-tree-input]");
    var treeDropdown = field.querySelector(".form-tree-select-dropdown");
    var treeClearBtn = field.querySelector("[data-form-tree-clear]");
    if (!dataEl || !deptSelect || !treeList || !treeInput || !treeDropdown) {
      return;
    }
    field.dataset.formTreeSelectInit = "1";

    var emptyLabel = treeEmptyLabel(deptSelect);
    treeInput.setAttribute("placeholder", emptyLabel);

    function syncTreeTriggerClearVis() {
      if (!treeClearBtn) {
        return;
      }
      var hasSel = !!(deptSelect.value || "").trim();
      var hasText = !!(treeInput.value || "").trim();
      treeClearBtn.hidden = !hasSel && !hasText;
    }

    function resetTreeListVisibility() {
      var allItems = treeList.querySelectorAll("li.form-tree-select-item");
      for (var z = 0; z < allItems.length; z++) {
        allItems[z].style.display = "";
      }
    }

    function applyTreeFilter() {
      var keyword = (treeInput.value || "").trim().toLowerCase();
      var allItems = treeList.querySelectorAll("li.form-tree-select-item");
      if (!keyword) {
        for (var z = 0; z < allItems.length; z++) {
          allItems[z].style.display = "";
        }
        return;
      }
      for (var h = 0; h < allItems.length; h++) {
        allItems[h].style.display = "none";
      }
      for (var k = 0; k < allItems.length; k++) {
        var item = allItems[k];
        var title = item.getAttribute("data-title") || "";
        if (title.indexOf(keyword) === -1) {
          continue;
        }
        item.style.display = "";
        var parentId = item.getAttribute("data-parent-id") || "";
        while (parentId) {
          var parentLi = treeList.querySelector(
            'li.form-tree-select-item[data-id="' + parentId + '"]'
          );
          if (!parentLi) {
            break;
          }
          parentLi.style.display = "";
          var parentToggle = parentLi.querySelector(
            ":scope > .form-tree-select-row .form-tree-select-toggle"
          );
          var ul = parentLi.querySelector(":scope > ul");
          if (parentToggle && ul) {
            parentToggle.classList.add("bi-chevron-down");
            parentToggle.classList.remove("bi-chevron-right");
            ul.style.display = "";
          }
          parentId = parentLi.getAttribute("data-parent-id") || "";
        }
      }
    }

    function readFlat() {
      var nodes = [];
      var spans = dataEl.querySelectorAll("span[data-id]");
      for (var i = 0; i < spans.length; i++) {
        var el = spans[i];
        var id = (el.getAttribute("data-id") || "").trim();
        if (!id) {
          continue;
        }
        nodes.push({
          id: id,
          parentId: (el.getAttribute("data-parent-id") || "0").trim(),
          name:
            (el.getAttribute("data-name") || "").trim() || "未命名项",
        });
      }
      return nodes;
    }

    function buildTree(flat) {
      var byId = {};
      for (var i = 0; i < flat.length; i++) {
        var n = flat[i];
        byId[n.id] = { id: n.id, name: n.name, children: [] };
      }
      var roots = [];
      for (var j = 0; j < flat.length; j++) {
        var item = flat[j];
        var node = byId[item.id];
        var pid = item.parentId || "0";
        if (pid === "0" || !byId[pid]) {
          roots.push(node);
        } else {
          byId[pid].children.push(node);
        }
      }
      return roots;
    }

    function syncInputFromSelect() {
      var v = (deptSelect.value || "").trim();
      if (!v) {
        treeInput.value = "";
        syncTreeTriggerClearVis();
        resetTreeListVisibility();
        return;
      }
      var opt = deptSelect.options[deptSelect.selectedIndex];
      treeInput.value = opt ? opt.textContent.trim() : "";
      syncTreeTriggerClearVis();
      resetTreeListVisibility();
    }

    function renderNodes(nodes, container, depth, parentDeptId) {
      for (var i = 0; i < nodes.length; i++) {
        var node = nodes[i];
        var li = document.createElement("li");
        li.className = "form-tree-select-item";
        li.setAttribute("data-id", node.id);
        li.setAttribute("data-parent-id", parentDeptId || "");
        li.setAttribute("data-title", (node.name || "").toLowerCase());

        var row = document.createElement("div");
        row.className = "form-tree-select-row";
        row.style.paddingLeft = depth * 14 + 6 + "px";

        var toggle = document.createElement("i");
        if (node.children.length > 0) {
          toggle.className = "bi bi-chevron-down form-tree-select-toggle";
        } else {
          toggle.className =
            "bi bi-dot form-tree-select-toggle form-tree-select-toggle--placeholder";
        }
        row.appendChild(toggle);

        var label = document.createElement("span");
        label.className = "form-tree-select-label";
        label.textContent = node.name;
        if (String(deptSelect.value || "") === String(node.id)) {
          label.classList.add("form-tree-select-label--active");
        }
        row.appendChild(label);
        li.appendChild(row);

        if (node.children.length > 0) {
          var childUl = document.createElement("ul");
          childUl.className = "form-tree-select-list";
          renderNodes(node.children, childUl, depth + 1, node.id);
          li.appendChild(childUl);
        }
        container.appendChild(li);
      }
    }

    var flat = readFlat();
    var roots = buildTree(flat);
    treeList.innerHTML = "";
    renderNodes(roots, treeList, 0, "");

    syncInputFromSelect();

    if (treeClearBtn) {
      treeClearBtn.addEventListener("click", function (event) {
        event.preventDefault();
        event.stopPropagation();
        if ((deptSelect.value || "").trim()) {
          deptSelect.value = "";
          deptSelect.dispatchEvent(new Event("change", { bubbles: true }));
          syncInputFromSelect();
          var act = treeList.querySelectorAll(".form-tree-select-label--active");
          for (var ac = 0; ac < act.length; ac++) {
            act[ac].classList.remove("form-tree-select-label--active");
          }
        } else {
          treeInput.value = "";
          resetTreeListVisibility();
          applyTreeFilter();
          syncTreeTriggerClearVis();
        }
        var ddOpen = treeInput.closest(".dropdown");
        if (ddOpen) {
          ddOpen.classList.remove("open");
        }
      });
    }

    treeInput.addEventListener("input", function () {
      applyTreeFilter();
      syncTreeTriggerClearVis();
    });
    treeInput.addEventListener("focus", function () {
      openDropdownOnly(treeDropdown);
      resetTreeListVisibility();
      applyTreeFilter();
    });
    treeInput.addEventListener("click", function () {
      openDropdownOnly(treeDropdown);
    });

    treeList.addEventListener("click", function (event) {
      var t = event.target;
      if (!t || !t.closest) {
        return;
      }
      var toggleIcon = t.closest(".form-tree-select-toggle");
      if (
        toggleIcon &&
        !toggleIcon.classList.contains("form-tree-select-toggle--placeholder")
      ) {
        event.stopPropagation();
        var li = toggleIcon.closest("li.form-tree-select-item");
        if (!li) {
          return;
        }
        var childUl = li.querySelector(":scope > ul");
        if (!childUl) {
          return;
        }
        var expanded = toggleIcon.classList.contains("bi-chevron-down");
        toggleIcon.classList.toggle("bi-chevron-down", !expanded);
        toggleIcon.classList.toggle("bi-chevron-right", expanded);
        childUl.style.display = expanded ? "none" : "";
        return;
      }

      var lab = t.closest(".form-tree-select-label");
      if (!lab) {
        return;
      }
      var itemLi = lab.closest("li.form-tree-select-item");
      if (!itemLi) {
        return;
      }
      var id = itemLi.getAttribute("data-id") || "";
      deptSelect.value = id;
      syncInputFromSelect();
      var all = treeList.querySelectorAll(".form-tree-select-label--active");
      for (var a = 0; a < all.length; a++) {
        all[a].classList.remove("form-tree-select-label--active");
      }
      lab.classList.add("form-tree-select-label--active");
      var dd = treeInput.closest(".dropdown");
      if (dd) {
        dd.classList.remove("open");
      }
    });

    treeInput.addEventListener("keydown", function (event) {
      if (event.key === "Enter") {
        event.preventDefault();
        if (typeof window.toggleDropdown === "function") {
          window.toggleDropdown(treeInput);
        }
      }
    });

    deptSelect.addEventListener("change", syncInputFromSelect);
  }

  function initFormComponents(container) {
    var root = container || document;
    // querySelectorAll 不包含容器自身；optionsform 等会以「带 data-role-dropdown 的根节点」
    // 为范围重初始化，此时必须显式处理该根节点，否则列表重建后 checkbox 无监听、标签不刷新。
    if (root.nodeType === 1 && root.matches && root.matches("[data-role-dropdown]")) {
      initFormRoleMultiselect(root);
    }
    var roleDropdowns = root.querySelectorAll("[data-role-dropdown]");
    for (var r = 0; r < roleDropdowns.length; r++) {
      initFormRoleMultiselect(roleDropdowns[r]);
    }
    var treeFields = root.querySelectorAll(".form-tree-select-field");
    for (var f = 0; f < treeFields.length; f++) {
      initFormTreeSelect(treeFields[f]);
    }
  }

  window.initFormComponents = initFormComponents;
  initFormComponents();
})();
