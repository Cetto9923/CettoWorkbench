(() => {
  const panel = document.querySelector(".table-panel.menu-table-panel");
  if (!panel) {
    return;
  }

  const table = panel.querySelector(".table.menu-table");
  const tbody = table.querySelector("tbody");
  const searchInput = panel.querySelector("#menuSearch");
  if (!table || !tbody) {
    return;
  }

  const rows = Array.from(tbody.querySelectorAll("tr[data-id][data-parent-id][data-type]"));
  const rowMap = new Map(rows.map((row) => [String(row.dataset.id || ""), row]));
  /** 叶子行占位：保留占位宽度，不参与展开点击（已从 .tree-toggle 剥离） */
  const TREE_TOGGLE_LEAF_CLASS = "tree-toggle-spacer";
  const childrenMap = new Map();
  rows.forEach((row) => {
    const parentID = String(row.dataset.parentId || "");
    if (!childrenMap.has(parentID)) {
      childrenMap.set(parentID, []);
    }
    childrenMap.get(parentID).push(row);
  });

  const getRowID = (row) => String(row?.dataset?.id || "");
  const getParentID = (row) => String(row?.dataset?.parentId || "");

  /** 1=顶级；2=顶级直系子级；3=更深层 */
  const getTreeDepth = (row) => {
    const pid = getParentID(row);
    if (pid === "0") {
      return 1;
    }
    const parentRow = rowMap.get(pid);
    if (!parentRow) {
      return 3;
    }
    if (getParentID(parentRow) === "0") {
      return 2;
    }
    return 3;
  };

  const setToggleIcon = (toggleEl, expanded) => {
    if (!toggleEl) {
      return;
    }
    toggleEl.classList.toggle("bi-chevron-down", expanded);
    toggleEl.classList.toggle("bi-chevron-right", !expanded);
  };

  const getChildren = (id) => childrenMap.get(String(id)) || [];

  const hasChildren = (row) => getChildren(getRowID(row)).length > 0;

  const getInteractiveToggle = (row) => row.querySelector(`i.tree-toggle:not(.${TREE_TOGGLE_LEAF_CLASS})`);

  const isExpanded = (row) => {
    const toggle = getInteractiveToggle(row);
    return !!toggle && toggle.classList.contains("bi-chevron-down");
  };

  /** 根据扁平行的 parentId 集合判定叶子，并同步折叠图标可见性（叶子保留占位对齐） */
  const applyLeafToggleUI = () => {
    const parentIds = new Set();
    rows.forEach((row) => {
      parentIds.add(String(row.dataset.parentId || ""));
    });

    rows.forEach((row) => {
      const id = getRowID(row);
      const isLeaf = !id || !parentIds.has(id);
      const toggle =
        row.querySelector(`i.tree-toggle, i.${TREE_TOGGLE_LEAF_CLASS}`) ||
        row.querySelector(".tree-cell > i:first-of-type");

      if (!toggle) {
        return;
      }

      if (isLeaf) {
        toggle.classList.remove("tree-toggle", "tree-toggle-placeholder");
        toggle.classList.add(TREE_TOGGLE_LEAF_CLASS);
        toggle.style.visibility = "hidden";
        toggle.style.pointerEvents = "none";
        return;
      }

      toggle.classList.remove(TREE_TOGGLE_LEAF_CLASS, "tree-toggle-placeholder");
      toggle.classList.add("tree-toggle");
      toggle.style.visibility = "visible";
      toggle.style.pointerEvents = "";
    });
  };

  const setRowVisible = (row, visible) => {
    row.style.display = visible ? "table-row" : "none";
  };

  const getAncestors = (row) => {
    const result = [];
    let cursor = row;
    while (cursor) {
      const parentID = getParentID(cursor);
      if (!parentID || parentID === "0") {
        break;
      }
      const parentRow = rowMap.get(parentID);
      if (!parentRow) {
        break;
      }
      result.push(parentRow);
      cursor = parentRow;
    }
    return result;
  };

  const hideDescendants = (rowID) => {
    getChildren(rowID).forEach((child) => {
      setRowVisible(child, false);
      const childToggle = getInteractiveToggle(child);
      if (childToggle) {
        setToggleIcon(childToggle, false);
      }
      hideDescendants(getRowID(child));
    });
  };

  const showExpandedBranch = (rowID) => {
    getChildren(rowID).forEach((child) => {
      setRowVisible(child, true);
      if (hasChildren(child) && isExpanded(child)) {
        showExpandedBranch(getRowID(child));
      }
    });
  };

  /**
   * 初始 / 重置树视图（一级默认展开）：
   * - 一级 (parentId=0)：显示，箭头展开 (down)
   * - 二级（父为一级）：显示，箭头收起 (right)，深层不展开
   * - 三级及以下：隐藏，箭头收起 (right)
   * 清空搜索时调用本函数，恢复该状态。
   */
  const initializeTreeState = () => {
    applyLeafToggleUI();

    rows.forEach((row) => {
      const depth = getTreeDepth(row);
      const toggle = getInteractiveToggle(row);

      if (depth === 1) {
        setRowVisible(row, true);
        if (toggle) {
          setToggleIcon(toggle, true);
        }
        return;
      }
      if (depth === 2) {
        setRowVisible(row, true);
        if (toggle) {
          setToggleIcon(toggle, false);
        }
        return;
      }
      setRowVisible(row, false);
      if (toggle) {
        setToggleIcon(toggle, false);
      }
    });
  };

  panel.addEventListener("click", (event) => {
    const toggle = event.target.closest(".tree-toggle");
    if (
      !toggle ||
      !panel.contains(toggle) ||
      toggle.classList.contains("tree-toggle-placeholder") ||
      toggle.classList.contains(TREE_TOGGLE_LEAF_CLASS) ||
      toggle.style.visibility === "hidden"
    ) {
      return;
    }

    const row = toggle.closest("tr[data-id]");
    if (!row) {
      return;
    }

    const rowID = getRowID(row);
    if (!rowID) {
      return;
    }

    const expanded = toggle.classList.contains("bi-chevron-down");
    if (expanded) {
      setToggleIcon(toggle, false);
      hideDescendants(rowID);
      return;
    }

    setToggleIcon(toggle, true);
    getChildren(rowID).forEach((child) => {
      setRowVisible(child, true);
      if (hasChildren(child) && isExpanded(child)) {
        showExpandedBranch(getRowID(child));
      }
    });
  });

  const resetTreeView = () => {
    initializeTreeState();
  };

  const expandAncestorsForRow = (row) => {
    const ancestors = getAncestors(row);
    ancestors.forEach((ancestor) => {
      setRowVisible(ancestor, true);
      const toggle = getInteractiveToggle(ancestor);
      if (toggle) {
        setToggleIcon(toggle, true);
      }
    });
  };

  if (searchInput) {
    searchInput.addEventListener("input", () => {
      const keyword = String(searchInput.value || "").trim().toLowerCase();
      if (keyword === "") {
        resetTreeView();
        return;
      }

      rows.forEach((row) => {
        setRowVisible(row, false);
      });

      rows.forEach((row) => {
        const title = String(row.dataset.title || "").toLowerCase();
        if (!title.includes(keyword)) {
          return;
        }

        setRowVisible(row, true);
        expandAncestorsForRow(row);
      });
    });
  }

  const boot = () => {
    if (rows.length === 0) {
      return;
    }
    initializeTreeState();
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
