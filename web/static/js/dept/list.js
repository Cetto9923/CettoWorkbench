(() => {
  const treeRoot = document.getElementById("deptTree");
  const currentDeptName = document.getElementById("currentDeptName");
  const currentDeptCount = document.getElementById("currentDeptCount");
  const emptyTip = document.getElementById("deptEmptyTip");
  const tableWrap = document.getElementById("deptTableWrap");
  const childRows = document.getElementById("deptChildRows");
  const tablePanel = document.querySelector(".table-panel");
  const rowTplWrap = document.getElementById("deptRowTemplates");
  if (!treeRoot || !currentDeptName || !currentDeptCount || !emptyTip || !tableWrap || !childRows || !tablePanel) {
    return;
  }

  let nodes = [];
  let nodeMap = new Map();
  let treeItemMap = new Map();
  const refreshTreeCaches = () => {
    nodes = Array.from(treeRoot.querySelectorAll(".tree-node-wrap[data-id]"));
    nodeMap = new Map(nodes.map((node) => [node.dataset.id || "", node]));
    treeItemMap = new Map(
      nodes.map((node) => [node.dataset.id || "", node.querySelector(".tree-node")]).filter((entry) => !!entry[1])
    );
  };
  refreshTreeCaches();

  const globalData = typeof window !== "undefined" ? window.__DEPT_DATA__ : null;
  const deptData = Array.isArray(globalData)
    ? globalData
    : nodes.map((node) => ({
        id: Number(node.dataset.id || "0"),
        parentId: Number(node.dataset.parentId || "0"),
        name: node.dataset.name || "",
        status: Number(node.dataset.status || "1"),
        leader: node.dataset.leader || "",
        sort: Number(node.dataset.sort || "0"),
        createdAt: node.dataset.createdAt || "—",
        editUrl: node.dataset.editUrl || "#",
      }));

  const childrenMap = new Map();
  const deptMap = new Map();
  const rebuildChildrenMap = () => {
    childrenMap.clear();
    deptData.forEach((item) => {
      const key = String(item.parentId);
      const group = childrenMap.get(key) || [];
      group.push(item);
      childrenMap.set(key, group);
    });
    rebuildDeptMap();
  };
  const rebuildDeptMap = () => {
    deptMap.clear();
    deptData.forEach((item) => {
      deptMap.set(String(item.id), item);
    });
  };
  rebuildChildrenMap();
  rebuildDeptMap();
  let currentSelectedDeptId = "";

  const setIcon = (toggleEl, expanded) => {
    if (toggleEl) {
      toggleEl.classList.toggle("bi-chevron-down", expanded);
      toggleEl.classList.toggle("bi-chevron-right", !expanded);
    }
  };

  const getChildNodes = (id) => nodes.filter((node) => node.dataset.parentId === String(id));
  const getChildData = (id) => (childrenMap.get(String(id)) || []).slice();
  const normalizeStatus = (value) => (value === 0 || value === "0" ? "0" : "1");
  const getDeptStatus = (id) => {
    const dept = deptMap.get(String(id));
    return dept ? normalizeStatus(dept.status) : "1";
  };
  const getDescendantIDs = (id) => {
    const output = [];
    const queue = [...getChildData(id)];
    while (queue.length > 0) {
      const current = queue.shift();
      if (!current) {
        continue;
      }
      output.push(String(current.id));
      getChildData(current.id).forEach((child) => queue.push(child));
    }
    return output;
  };
  const getAncestorIDs = (id) => {
    const output = [];
    let cursor = deptMap.get(String(id));
    while (cursor && Number(cursor.parentId) > 0) {
      const parentID = String(cursor.parentId);
      output.push(parentID);
      cursor = deptMap.get(parentID);
    }
    return output;
  };
  const isDeptNormal = (dept) => {
    const rawStatus = dept && Object.prototype.hasOwnProperty.call(dept, "status")
      ? dept.status
      : dept && Object.prototype.hasOwnProperty.call(dept, "Status")
        ? dept.Status
        : 1;
    return rawStatus === 0 || rawStatus === "0";
  };

  const collapseDescendants = (id, includeDirect) => {
    const children = getChildNodes(id);
    children.forEach((node) => {
      if (includeDirect) {
        node.style.display = "none";
      }
      const childItem = node.querySelector(".tree-node");
      if (childItem) {
        childItem.classList.remove("tree-node-active");
      }
      const childToggle = node.querySelector(".tree-node-toggle");
      if (childToggle && !childToggle.classList.contains("placeholder")) {
        setIcon(childToggle, false);
      }
      collapseDescendants(node.dataset.id, true);
    });
  };

  const setParentAsBranch = (parentID) => {
    const parentNode = nodeMap.get(String(parentID));
    if (!parentNode) {
      return;
    }
    const toggle = parentNode.querySelector(".tree-node-toggle");
    if (!toggle) {
      return;
    }
    toggle.classList.remove("bi-dot", "placeholder");
    toggle.classList.add("bi-chevron-down");
    toggle.style.pointerEvents = "";
  };

  const updateAncestorsUI = (startParentID) => {
    let currentID = String(startParentID || "").trim();
    while (currentID && currentID !== "0") {
      const node = treeRoot.querySelector(`.tree-node-wrap[data-id="${currentID}"]`);
      if (!node) {
        break;
      }
      node.setAttribute("data-status", "0");
      updateLocalStatus(currentID, "0");
      const treeItem = node.querySelector(":scope > .tree-node");
      if (treeItem) {
        treeItem.classList.remove("tree-node-dimmed");
      }
      const toggleEl = node.querySelector(":scope > .tree-node .tree-node-toggle");
      const childCount = getChildData(currentID).length;
      if (toggleEl && childCount > 0) {
        toggleEl.classList.remove("bi-dot", "placeholder", "bi-chevron-right");
        toggleEl.classList.add("bi-chevron-down");
        toggleEl.style.pointerEvents = "";
      }
      currentID = (node.getAttribute("data-parent-id") || "").trim();
    }
  };

  const createTreeNode = (dept) => {
    const level = Number(dept.level || 1);
    const li = document.createElement("li");
    li.className = "tree-node-wrap";
    li.dataset.id = String(dept.id || dept.ID || "");
    li.dataset.parentId = String(dept.parentId || dept.ParentId || dept.parentID || 0);
    li.dataset.level = String(dept.level || 1);
    li.dataset.name = dept.name || dept.Name || "";
    li.dataset.status = String(dept.status || dept.Status || 0);
    li.dataset.leader = dept.leader || dept.Leader || "";
    li.dataset.sort = String(dept.sort || dept.Sort || 0);
    li.dataset.createdAt = dept.createdAt || dept.CreatedAt || "刚刚";
    li.dataset.editUrl = dept.editUrl || dept.EditUrl || `/admin/depts/${li.dataset.id}/edit`;
    li.dataset.hasChildren = "0";

    li.innerHTML = `
      <a href="#" class="tree-node" data-id="${li.dataset.id}" style="--depth:${Math.max(level - 1, 0)};">
        <i class="bi bi-dot tree-node-toggle placeholder"></i>
        <span class="tree-node-name"></span>
      </a>`;
    const nameEl = li.querySelector(".tree-node-name");
    if (nameEl) {
      nameEl.textContent = li.dataset.name;
    }
    return li;
  };

  const setCurrentTitle = (id) => {
    const path = [];
    let current = nodeMap.get(String(id));
    while (current) {
      path.unshift(current.dataset.name || "");
      const parentID = current.dataset.parentId || "";
      current = parentID ? nodeMap.get(parentID) : null;
    }
    currentDeptName.textContent = path.length ? path[path.length - 1] : "请在左侧选择部门";
  };

  const getTemplateHtml = (selector) => {
    const tpl = rowTplWrap ? rowTplWrap.querySelector(selector) : null;
    return tpl ? tpl.innerHTML.trim() : "";
  };

  const renderStatusSwitchHtml = (dept) => {
    const status = normalizeStatus(dept.status);
    const id = String(dept.id || "");
    const parentId = String(dept.parentId || 0);
    return `<label class="status-switch"><input type="checkbox" class="dept-status-switch" data-id="${id}" data-parent-id="${parentId}" data-status="${status}" data-status-value="${status}" ${status === "0" ? "checked" : ""}><span class="status-switch-slider"></span></label>`;
  };

  const renderActionHtml = (dept) => {
    const tplHtml = getTemplateHtml(`template[data-action-id="${String(dept.id)}"]`);
    if (tplHtml) {
      return tplHtml;
    }
    const editUrl = dept.editUrl || `/admin/depts/${dept.id}/edit`;
    const deleteUrl = `/admin/depts/${dept.id}`;
    const safeName = String(dept.name || "").replace(/'/g, "\\'");
    return `<div class="action-group"><a href="${editUrl}" class="btn-icon btn-icon-edit" title="编辑"><i class="bi bi-pencil"></i></a><button type="button" class="btn-icon btn-icon-danger" data-url="${deleteUrl}" data-id="${dept.id}" onclick="return confirmDelete(this, '${safeName}', '${deleteUrl}')" title="删除"><i class="bi bi-trash"></i></button></div>`;
  };

  const renderChildrenTable = (id) => {
    const children = getChildData(id).sort((a, b) => {
      if (a.sort === b.sort) {
        return a.id - b.id;
      }
      return a.sort - b.sort;
    });
    childRows.innerHTML = "";
    currentDeptCount.textContent = `子部门 ${children.length} 个`;
    if (children.length === 0) {
      childRows.innerHTML = '<tr><td colspan="7" class="text-center text-muted">该部门暂无直属下级</td></tr>';
    } else {
      children.forEach((dept) => {
        const isNormal = isDeptNormal(dept);
        const tr = document.createElement("tr");
        tr.dataset.rowId = String(dept.id || "");
        if (!isNormal) {
          tr.classList.add("row-disabled");
        }
        const tdID = document.createElement("td");
        tdID.className = "text-muted";
        tdID.textContent = String(dept.id || "");
        tr.appendChild(tdID);

        const tdName = document.createElement("td");
        tdName.className = "text-strong";
        tdName.textContent = dept.name || "";
        tr.appendChild(tdName);

        const tdLeader = document.createElement("td");
        tdLeader.className = "text-secondary";
        tdLeader.textContent = dept.leader || dept.Leader || "—";
        tr.appendChild(tdLeader);

        const tdStatus = document.createElement("td");
        tdStatus.innerHTML = renderStatusSwitchHtml(dept);
        tr.appendChild(tdStatus);

        const tdSort = document.createElement("td");
        tdSort.className = "text-center text-muted";
        tdSort.textContent = String(dept.sort || 0);
        tr.appendChild(tdSort);

        const tdCreatedAt = document.createElement("td");
        tdCreatedAt.className = "text-muted";
        tdCreatedAt.textContent = dept.createdAt || "—";
        tr.appendChild(tdCreatedAt);

        const tdAction = document.createElement("td");
        tdAction.className = "text-right";
        tdAction.innerHTML = renderActionHtml(dept);
        tr.appendChild(tdAction);
        childRows.appendChild(tr);
      });
    }
    const quickAddRow = document.createElement("tr");
    quickAddRow.className = "quick-add-trigger-row";
    quickAddRow.innerHTML = '<td colspan="7" class="text-center"><button type="button" class="btn btn-neutral btn-sm quick-add-trigger" title="快速新增新部门"><i class="bi bi-plus-circle"></i> 快速新增新部门</button></td>';
    childRows.appendChild(quickAddRow);
    emptyTip.style.display = "none";
    tableWrap.style.display = "";
  };

  const removeQuickInputRow = () => {
    const existing = childRows.querySelector("tr.quick-add-row");
    if (existing) {
      existing.remove();
    }
  };

  const insertQuickInputRow = () => {
    removeQuickInputRow();
    const parentId = String(currentSelectedDeptId || "");
    const tr = document.createElement("tr");
    tr.className = "quick-add-row";
    tr.innerHTML = `
      <td>—</td>
      <td>
        <input type="text" class="input input-quick-add" name="quickName" placeholder="输入部门名称">
      </td>
      <td>
        <input type="text" class="input" name="quickLeader" placeholder="负责人">
      </td>
      <td><label class="status-switch"><input type="checkbox" checked disabled><span class="status-switch-slider"></span></label></td>
      <td>
        <input type="number" class="input" name="quickSort" value="0" style="padding: 0 4px; text-align: center;">
      </td>
      <td class="text-muted">—</td>
      <td class="text-right">
        <div class="action-group">
          <button type="button" class="btn btn-primary btn-sm quick-add-save btn-quick-save" data-parent-id="${parentId}">保存</button>
          <button type="button" class="btn btn-neutral btn-sm quick-add-cancel btn-quick-cancel">取消</button>
        </div>
      </td>`;
    const triggerRow = childRows.querySelector(".quick-add-trigger-row");
    if (triggerRow) {
      childRows.insertBefore(tr, triggerRow);
    } else {
      childRows.appendChild(tr);
    }
    const nameInput = tr.querySelector('input[name="quickName"]');
    if (nameInput) {
      nameInput.focus();
    }
  };

  const activateNode = (id) => {
    refreshTreeCaches();
    treeRoot.querySelectorAll(".tree-node.tree-node-active").forEach((el) => el.classList.remove("tree-node-active"));
    const item = treeItemMap.get(String(id));
    if (item) {
      item.classList.add("tree-node-active");
    }
    currentSelectedDeptId = String(id || "");
    setCurrentTitle(id);
    renderChildrenTable(id);
  };

  treeRoot.addEventListener("click", (event) => {
    const toggle = event.target.closest(".tree-node-toggle");
    if (toggle && !toggle.classList.contains("placeholder")) {
      event.stopPropagation();
      const node = toggle.closest(".tree-node-wrap");
      if (!node) {
        return;
      }
      const expanded = toggle.classList.contains("bi-chevron-down");
      if (expanded) {
        setIcon(toggle, false);
        collapseDescendants(node.dataset.id, true);
      } else {
        setIcon(toggle, true);
        getChildNodes(node.dataset.id).forEach((child) => {
          child.style.display = "";
        });
      }
      return;
    }

    const item = event.target.closest(".tree-node");
    if (!item) {
      return;
    }
    event.preventDefault();
    treeRoot.querySelectorAll(".tree-node.tree-node-active").forEach((el) => el.classList.remove("tree-node-active"));
    item.classList.add("tree-node-active");
    const node = item.closest(".tree-node-wrap");
    if (!node) {
      return;
    }
    activateNode(node.dataset.id);
  });

  const firstRootNodeItem = treeRoot.querySelector('.tree-node-wrap[data-level="1"] .tree-node');
  if (firstRootNodeItem) {
    firstRootNodeItem.click();
  }

  const getCsrfToken = () => {
    const tokenEl = document.querySelector('meta[name="csrf-token"]');
    if (!tokenEl) {
      return "";
    }
    return (tokenEl.getAttribute("content") || "").trim();
  };

  const updateLocalStatus = (id, status) => {
    const normalized = normalizeStatus(status);
    const dept = deptMap.get(String(id));
    if (dept) {
      dept.status = normalized === "0" ? 0 : 1;
    }
    const treeNode = nodeMap.get(String(id));
    if (treeNode) {
      treeNode.dataset.status = normalized;
      const treeItem = treeNode.querySelector(".tree-node");
      if (treeItem) {
        treeItem.classList.toggle("tree-node-dimmed", normalized !== "0");
      }
    }
    const switchEl = childRows.querySelector(`.dept-status-switch[data-id="${String(id)}"]`);
    if (switchEl) {
      switchEl.checked = normalized === "0";
      switchEl.dataset.status = normalized;
      switchEl.dataset.statusValue = normalized;
    }
    const row = childRows.querySelector(`tr[data-row-id="${String(id)}"]`);
    if (row) {
      row.classList.toggle("row-disabled", normalized !== "0");
    }
  };

  const setSwitchDisabled = (id, disabled) => {
    const switchEl = childRows.querySelector(`.dept-status-switch[data-id="${String(id)}"]`);
    if (switchEl) {
      switchEl.disabled = disabled;
    }
  };

  const requestStatusUpdate = async (id, status, csrfToken) => {
    const response = await fetch(`/admin/depts/${id}/status`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
        "X-CSRF-Token": csrfToken,
        "X-Requested-With": "XMLHttpRequest",
        Accept: "application/json",
      },
      body: new URLSearchParams({ status }).toString(),
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message = payload && payload.message ? payload.message : "状态更新失败";
      throw new Error(message);
    }
  };

  tablePanel.addEventListener("change", async (event) => {
    const switchEl = event.target.closest(".dept-status-switch");
    if (!switchEl || !tablePanel.contains(switchEl)) {
      return;
    }

    const id = (switchEl.dataset.id || "").trim();
    if (!id) {
      return;
    }
    const originalStatus = normalizeStatus(switchEl.dataset.statusValue || switchEl.dataset.status);
    const requestedStatus = switchEl.checked ? "0" : "1";
    if (requestedStatus === originalStatus) {
      return;
    }

    const updateTargets = [];
    if (requestedStatus === "1") {
      updateTargets.push(id, ...getDescendantIDs(id));
    } else {
      const ancestorIDs = getAncestorIDs(id);
      const disabledAncestors = ancestorIDs.filter((ancestorID) => getDeptStatus(ancestorID) === "1");
      if (disabledAncestors.length > 0) {
        const confirmed = window.confirm("当前部门的上级节点已停用，开启本部门需要同时开启所有上级节点，是否继续？");
        if (!confirmed) {
          switchEl.checked = false;
          return;
        }
      }
      updateTargets.push(...disabledAncestors.reverse(), id);
    }

    const uniqueTargets = Array.from(new Set(updateTargets.map((targetID) => String(targetID))));
    const snapshot = new Map();
    uniqueTargets.forEach((targetID) => {
      snapshot.set(targetID, getDeptStatus(targetID));
      setSwitchDisabled(targetID, true);
      updateLocalStatus(targetID, requestedStatus);
    });

    const csrfToken = getCsrfToken();
    try {
      for (const targetID of uniqueTargets) {
        await requestStatusUpdate(targetID, requestedStatus, csrfToken);
      }
      if (typeof window.showToast === "function") {
        window.showToast("修改成功", "success");
      }
    } catch (err) {
      snapshot.forEach((statusValue, targetID) => {
        updateLocalStatus(targetID, statusValue);
      });
      const message = err && err.message ? err.message : "网络异常，状态更新失败";
      if (typeof window.showToast === "function") {
        window.showToast(message, "error");
      }
      alert(message);
    } finally {
      uniqueTargets.forEach((targetID) => {
        setSwitchDisabled(targetID, false);
      });
    }
  });

  const handleQuickSave = async (quickSaveBtn) => {
      const quickRow = quickSaveBtn.closest("tr.quick-add-row");
      if (!quickRow || !currentSelectedDeptId) {
        return;
      }
      const nameInput = quickRow.querySelector('input[name="quickName"]');
      const leaderInput = quickRow.querySelector('input[name="quickLeader"]');
      const sortInput = quickRow.querySelector('input[name="quickSort"]');
      const name = ((nameInput && nameInput.value) || "").trim();
      const leader = ((leaderInput && leaderInput.value) || "").trim();
      const sortRaw = ((sortInput && sortInput.value) || "0").trim();
      const sort = sortRaw === "" ? "0" : sortRaw;
      if (!name) {
        if (nameInput) {
          nameInput.focus();
        }
        alert("请输入部门名称");
        return;
      }
      quickSaveBtn.disabled = true;
      const csrfToken = getCsrfToken();
      try {
        const selectedParentNode = nodeMap.get(String(currentSelectedDeptId));
        const parentStatus = ((selectedParentNode && selectedParentNode.dataset && selectedParentNode.dataset.status) || "0").trim();
        let status = "0";
        let enableAncestors = "false";
        if (parentStatus === "1") {
          const confirmed = window.confirm("当前上级部门处于停用状态。是否同时启用上级部门？\n\n点击【确定】将同步启用上级并按正常状态创建。\n点击【取消】将按停用状态创建本部门。");
          if (confirmed) {
            enableAncestors = "true";
            status = "0";
          } else {
            enableAncestors = "false";
            status = "1";
          }
        }
        const payload = new URLSearchParams({
          parentId: currentSelectedDeptId,
          name,
          leader,
          sort,
          status,
          enableAncestors,
        });
        const response = await fetch("/admin/depts", {
          method: "POST",
          headers: {
            "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
            "X-CSRF-Token": csrfToken,
            "X-Requested-With": "XMLHttpRequest",
            Accept: "application/json, text/html",
          },
          body: payload.toString(),
          redirect: "follow",
        });
        if (!response.ok) {
          alert("创建失败，请稍后重试");
          return;
        }
        let createdID = Date.now();
        let createdName = name;
        let createdParentID = Number(currentSelectedDeptId);
        const jsonPayload = await response.clone().json().catch(() => null);
        if (jsonPayload) {
          if (jsonPayload.id) {
            createdID = Number(jsonPayload.id) || createdID;
          }
          if (jsonPayload.name || jsonPayload.Name) {
            createdName = jsonPayload.name || jsonPayload.Name;
          }
          if (jsonPayload.parentId || jsonPayload.ParentId) {
            createdParentID = Number(jsonPayload.parentId || jsonPayload.ParentId) || createdParentID;
          }
        }
        const createdDept = {
          id: createdID,
          parentId: createdParentID,
          name: createdName,
          leader,
          status: status === "0" ? 0 : 1,
          sort: Number(sort) || 0,
          createdAt: "刚刚",
          editUrl: `/admin/depts/${createdID}/edit`,
        };
        const parentNode = nodeMap.get(String(createdParentID));
        createdDept.level = parentNode ? Number(parentNode.dataset.level || "1") + 1 : 1;
        deptData.push(createdDept);
        rebuildChildrenMap();
        if (parentNode) {
          const createdNode = createTreeNode(createdDept);
          createdNode.style.display = "none";
          treeRoot.appendChild(createdNode);
          setParentAsBranch(createdParentID);
        }
        refreshTreeCaches();
        if (enableAncestors === "true") {
          updateAncestorsUI(String(createdParentID));
        }
        removeQuickInputRow();
        activateNode(currentSelectedDeptId);
      } catch (_) {
        alert("创建失败，请检查网络后重试");
      } finally {
        quickSaveBtn.disabled = false;
      }
  };

  tablePanel.addEventListener("click", async (event) => {
    const quickAddTrigger = event.target.closest(".quick-add-trigger");
    if (quickAddTrigger && tablePanel.contains(quickAddTrigger)) {
      if (!currentSelectedDeptId) {
        return;
      }
      insertQuickInputRow();
      return;
    }

    const quickSaveBtn = event.target.closest(".btn-quick-save");
    if (quickSaveBtn && tablePanel.contains(quickSaveBtn)) {
      await handleQuickSave(quickSaveBtn);
      return;
    }

    const quickCancelBtn = event.target.closest(".btn-quick-cancel");
    if (quickCancelBtn && tablePanel.contains(quickCancelBtn)) {
      removeQuickInputRow();
    }
  });

  tablePanel.addEventListener("keydown", async (event) => {
    if (event.key !== "Enter") {
      return;
    }
    const inputEl = event.target.closest(".input-quick-add");
    if (!inputEl || !tablePanel.contains(inputEl)) {
      return;
    }
    event.preventDefault();
    const quickRow = inputEl.closest("tr.quick-add-row");
    if (!quickRow) {
      return;
    }
    const quickSaveBtn = quickRow.querySelector(".btn-quick-save");
    if (!quickSaveBtn) {
      return;
    }
    await handleQuickSave(quickSaveBtn);
  });

})();
