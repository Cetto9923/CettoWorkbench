(() => {
  const form = document.querySelector("form[data-menu-form]");
  if (!form) {
    return;
  }

  const typeSelect = document.getElementById("type") || form.querySelector("[data-menu-type]");
  const iconInput = form.querySelector("#icon");
  const iconPreviewWrap = form.querySelector("[data-icon-preview]");
  const iconDropdown = form.querySelector("[data-icon-dropdown]");
  const iconPickerOpen = form.querySelector(".menu-icon-picker-open");
  const iconSearch = form.querySelector("[data-icon-search]");
  const iconGrid = form.querySelector("[data-icon-grid]");
  const parentSelect = form.querySelector("[data-parent-select]");
  const parentDropdown = form.querySelector("[data-parent-dropdown]");
  const parentDisplay = form.querySelector("[data-parent-display]");
  const parentSearch = form.querySelector("[data-parent-search]");
  const parentTree = form.querySelector("[data-parent-tree]");
  const currentID = String(form.dataset.currentId || "").trim();

  /** 常用 Bootstrap Icons 类名（12×4 栅格共 48 个） */
  const MENU_FORM_ICON_CLASSES = [
    "fa-bars",
    "fa-list-ul",
    "fa-list",
    "fa-table-cells-large",
    "fa-table-cells",
    "fa-columns",
    "fa-list-check",
    "fa-user",
    "fa-users",
    "fa-id-badge",
    "fa-user-gear",
    "fa-user",
    "fa-house",
    "fa-house",
    "fa-building",
    "fa-globe",
    "fa-gear",
    "fa-gear",
    "fa-sliders",
    "fa-sliders",
    "fa-screwdriver-wrench",
    "fa-wrench",
    "fa-folder",
    "fa-folder-open",
    "fa-file",
    "fa-file-lines",
    "fa-book",
    "fa-right-to-bracket",
    "fa-right-from-bracket",
    "fa-box",
    "fa-shield-alt",
    "fa-shield",
    "fa-key",
    "fa-lock",
    "fa-unlock",
    "fa-gauge",
    "fa-chart-line",
    "fa-chart-column",
    "fa-chart-pie",
    "fa-magnifying-glass",
    "fa-bell",
    "fa-envelope",
    "fa-comment-dots",
    "fa-calendar-days",
    "fa-clock-rotate-left",
    "fa-pencil",
    "fa-pen-to-square",
    "fa-trash",
  ];

  const closeDropdowns = () => {
    if (typeof window.closeAllDropdowns === "function") {
      window.closeAllDropdowns();
      return;
    }
    document.querySelectorAll(".dropdown.open").forEach((el) => {
      el.classList.remove("open");
    });
  };

  /**
   * 菜单类型联动（M=目录≈1，C=菜单≈2，F=按钮≈3）：
   * 目录：上级、类型、标题、图标、排序；隐藏路由、权限。
   * 菜单：全部显示。
   * 按钮：上级、类型、标题、权限、排序；隐藏图标、路由。
   */
  const handleTypeChange = () => {
    const v = typeSelect ? String(typeSelect.value || "C") : "C";
    const setGroupDisplay = (key, visible) => {
      const el = form.querySelector(`[data-menu-field="${key}"]`);
      if (!el) {
        return;
      }
      el.style.display = visible ? "" : "none";
    };

    setGroupDisplay("parent", true);
    setGroupDisplay("type", true);
    setGroupDisplay("title", true);
    setGroupDisplay("sort", true);

    let showPath = true;
    let showPerm = true;
    let showIcon = true;
    if (v === "M") {
      showPath = false;
      showPerm = false;
    } else if (v === "F") {
      showPath = false;
      showIcon = false;
    }

    setGroupDisplay("path", showPath);
    setGroupDisplay("perm", showPerm);
    setGroupDisplay("icon", showIcon);

    const pathInput = form.querySelector("#path");
    const permInput = form.querySelector("#perm");
    if (pathInput) {
      pathInput.disabled = !showPath;
    }
    if (permInput) {
      permInput.disabled = !showPerm;
    }
    if (iconInput) {
      iconInput.disabled = !showIcon;
    }
    if (iconPickerOpen) {
      iconPickerOpen.disabled = !showIcon;
    }
    if (!showIcon && iconDropdown) {
      iconDropdown.classList.remove("open");
    }
  };

  const updateIconPreview = () => {
    if (!iconInput || !iconPreviewWrap) {
      return;
    }
    const iconClass = String(iconInput.value || "").trim();
    const iconEl = iconPreviewWrap.querySelector("i");
    if (!iconEl) {
      return;
    }
    iconEl.className = iconClass ? `fas ${iconClass}` : "fas fa-window-maximize";
  };

  const parseParentOptions = () => {
    if (!parentSelect) {
      return [];
    }
    return Array.from(parentSelect.options).map((option) => {
      const rawTitle = String(option.textContent || "").trim();
      const matched = rawTitle.match(/^(—\s)*/);
      const prefix = matched && matched[0] ? matched[0] : "";
      const level = prefix ? Math.floor(prefix.length / 2) : 0;
      return {
        id: String(option.value || "0"),
        title: rawTitle.replace(/^(—\s)*/, "").trim() || "未命名菜单",
        level,
        selected: option.selected,
      };
    });
  };

  const buildParentTree = (items) => {
    const stack = [];
    const roots = [];
    items.forEach((item) => {
      while (stack.length > item.level) {
        stack.pop();
      }
      const parent = stack.length > 0 ? stack[stack.length - 1] : null;
      const node = {
        ...item,
        parentId: parent ? parent.id : "",
        children: [],
      };
      if (parent) {
        parent.children.push(node);
      } else {
        roots.push(node);
      }
      stack.push(node);
    });
    return roots;
  };

  const renderParentTree = (roots) => {
    if (!parentTree) {
      return;
    }
    parentTree.innerHTML = "";

    const render = (nodes, container) => {
      nodes.forEach((node) => {
        const li = document.createElement("li");
        li.className = "menu-parent-item";
        li.dataset.id = node.id;
        li.dataset.title = node.title.toLowerCase();
        li.dataset.parentId = node.parentId;

        const row = document.createElement("div");
        row.className = "menu-parent-row";
        row.style.paddingLeft = `${node.level * 14 + 6}px`;

        const toggle = document.createElement("i");
        if (node.children.length > 0) {
          toggle.className = "fas fa-chevron-down menu-parent-toggle";
        } else {
          toggle.className = "fas fa-circle menu-parent-toggle placeholder";
        }
        row.appendChild(toggle);

        const label = document.createElement("span");
        label.className = "menu-parent-label";
        label.textContent = node.title;
        if (node.selected) {
          label.classList.add("active");
          if (parentDisplay) {
            parentDisplay.value = node.title;
          }
        }
        if (currentID !== "" && node.id === currentID) {
          label.classList.add("text-muted");
          li.dataset.disabled = "1";
        }
        row.appendChild(label);
        li.appendChild(row);

        if (node.children.length > 0) {
          const childUL = document.createElement("ul");
          childUL.className = "menu-parent-tree";
          render(node.children, childUL);
          li.appendChild(childUL);
        }
        container.appendChild(li);
      });
    };

    render(roots, parentTree);
  };

  const filterParentTree = () => {
    if (!parentTree || !parentSearch) {
      return;
    }
    const keyword = String(parentSearch.value || "").trim().toLowerCase();
    const items = Array.from(parentTree.querySelectorAll("li[data-id]"));
    if (!keyword) {
      items.forEach((item) => {
        item.style.display = "";
      });
      return;
    }
    items.forEach((item) => {
      item.style.display = "none";
    });
    items.forEach((item) => {
      const title = String(item.dataset.title || "");
      if (!title.includes(keyword)) {
        return;
      }
      item.style.display = "";
      let parentId = String(item.dataset.parentId || "");
      while (parentId) {
        const parent = parentTree.querySelector(`li[data-id="${parentId}"]`);
        if (!parent) {
          break;
        }
        parent.style.display = "";
        const toggle = parent.querySelector(":scope > .menu-parent-row .menu-parent-toggle");
        const childUL = parent.querySelector(":scope > ul");
        if (toggle && childUL) {
          toggle.classList.add("fa-chevron-down");
          toggle.classList.remove("fa-chevron-right");
          childUL.style.display = "";
        }
        parentId = String(parent.dataset.parentId || "");
      }
    });
  };

  const renderIconGrid = () => {
    if (!iconGrid) {
      return;
    }
    const keyword = iconSearch ? String(iconSearch.value || "").trim().toLowerCase() : "";
    iconGrid.innerHTML = "";
    MENU_FORM_ICON_CLASSES.filter((cls) => {
      if (!keyword) {
        return true;
      }
      return cls.toLowerCase().includes(keyword);
    }).forEach((cls) => {
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "menu-icon-picker-cell";
      btn.dataset.iconClass = cls;
      btn.title = cls;
      btn.innerHTML = `<i class="fas ${cls}"></i>`;
      iconGrid.appendChild(btn);
    });
  };

  if (typeSelect) {
    typeSelect.addEventListener("change", handleTypeChange);
    handleTypeChange();
  }

  if (iconInput) {
    iconInput.addEventListener("input", updateIconPreview);
    iconInput.addEventListener("change", updateIconPreview);
    updateIconPreview();
  }

  if (iconPickerOpen && typeof window.toggleDropdown === "function") {
    iconPickerOpen.addEventListener("click", (event) => {
      event.preventDefault();
      window.toggleDropdown(iconPickerOpen);
    });
  }

  if (iconGrid && iconInput) {
    renderIconGrid();
    if (iconSearch) {
      iconSearch.addEventListener("input", renderIconGrid);
    }
    iconGrid.addEventListener("click", (event) => {
      const cell = event.target.closest(".menu-icon-picker-cell[data-icon-class]");
      if (!cell) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      const cls = String(cell.dataset.iconClass || "").trim();
      iconInput.value = cls;
      updateIconPreview();
      if (iconDropdown) {
        iconDropdown.classList.remove("open");
      }
    });
  }

  if (parentSelect && parentDropdown && parentDisplay && parentTree) {
    const roots = buildParentTree(parseParentOptions());
    renderParentTree(roots);

    if (!parentDisplay.value.trim()) {
      parentDisplay.value = "请选择上级菜单";
    }

    if (parentSearch) {
      parentSearch.addEventListener("input", filterParentTree);
    }

    parentTree.addEventListener("click", (event) => {
      const toggle = event.target.closest(".menu-parent-toggle");
      if (toggle && !toggle.classList.contains("placeholder")) {
        event.stopPropagation();
        const li = toggle.closest("li[data-id]");
        if (!li) {
          return;
        }
        const childUL = li.querySelector(":scope > ul");
        if (!childUL) {
          return;
        }
        const expanded = toggle.classList.contains("fa-chevron-down");
        toggle.classList.toggle("fa-chevron-down", !expanded);
        toggle.classList.toggle("fa-chevron-right", expanded);
        childUL.style.display = expanded ? "none" : "";
        return;
      }

      const label = event.target.closest(".menu-parent-label");
      if (!label) {
        return;
      }
      const li = label.closest("li[data-id]");
      if (!li || li.dataset.disabled === "1") {
        return;
      }
      const id = String(li.dataset.id || "0");
      parentSelect.value = id;
      parentDisplay.value = String(label.textContent || "请选择上级菜单");
      parentTree.querySelectorAll(".menu-parent-label.active").forEach((el) => {
        el.classList.remove("active");
      });
      label.classList.add("active");
      closeDropdowns();
    });
  }
})();
