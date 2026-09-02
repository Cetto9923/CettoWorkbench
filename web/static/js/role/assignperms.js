(function () {
  "use strict";

  function escId(id) {
    if (id === null || id === undefined) {
      return "";
    }
    var s = String(id);
    if (typeof CSS !== "undefined" && CSS.escape) {
      return CSS.escape(s);
    }
    return s.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function parseDepth(tr) {
    return parseInt(tr.getAttribute("data-perm-depth"), 10) || 0;
  }

  function toggleRowExpand(tr) {
    // 1. 获取当前状态并切换
    var isExpanded = tr.getAttribute("data-perm-expanded") === "1";
    var willExpand = !isExpanded;
    tr.setAttribute("data-perm-expanded", willExpand ? "1" : "0");

    // 2. 更新图标形态
    var btn = tr.querySelector(".js-perm-tree-expander");
    if (btn) {
      btn.setAttribute("aria-expanded", willExpand ? "true" : "false");
    }

    // 3. 核心修复：带状态记忆的递归显隐算法
    var d = parseDepth(tr);
    var cur = tr.nextElementSibling;
    var hideDepthThreshold = 9999; // 状态阈值：当遇到被折叠的子节点时，比它深的节点全隐藏

    if (!willExpand) {
      // 动作：折叠当前节点。直接隐藏其下所有子孙节点。
      while (cur && cur.classList.contains("js-perm-tree-node")) {
        if (parseDepth(cur) <= d) break;
        cur.classList.add("d-none");
        cur = cur.nextElementSibling;
      }
    } else {
      // 动作：展开当前节点。必须尊重子节点原有的折叠状态。
      while (cur && cur.classList.contains("js-perm-tree-node")) {
        var cd = parseDepth(cur);
        if (cd <= d) break; // 超出当前子树范围

        if (cd > hideDepthThreshold) {
          // 在某个被折叠的子目录深处，继续隐藏
          cur.classList.add("d-none");
        } else {
          // 当前节点应该显示
          cur.classList.remove("d-none");
          hideDepthThreshold = 9999; // 恢复正常显示阈值

          // 【关键】如果这个刚被显示的节点本身是“折叠”状态，那么它下面的子节点绝不能露出来
          if (cur.getAttribute("data-perm-expanded") === "0") {
            hideDepthThreshold = cd;
          }
        }
        cur = cur.nextElementSibling;
      }
    }
  }

  function applyExpandedState(form, expandedAll) {
    form.querySelectorAll("tr.js-perm-tree-node").forEach(function (tr) {
      if (parseDepth(tr) > 0) {
        if (expandedAll) {
          tr.classList.remove("d-none");
        } else {
          tr.classList.add("d-none");
        }
      }
      tr.setAttribute("data-perm-expanded", expandedAll ? "1" : "0");
      var btn = tr.querySelector(".js-perm-tree-expander");
      if (btn) {
        btn.setAttribute("aria-expanded", expandedAll ? "true" : "false");
      }
    });
  }

  function syncInitialCollapsedSubtrees(form) {
    var rows = Array.prototype.slice.call(form.querySelectorAll("tr.js-perm-tree-node"));
    rows.forEach(function (tr) {
      var btn = tr.querySelector(".js-perm-tree-expander");
      if (btn) {
        btn.setAttribute(
          "aria-expanded",
          tr.getAttribute("data-perm-expanded") === "0" ? "false" : "true"
        );
      }
      if (tr.getAttribute("data-perm-expanded") !== "0") {
        return;
      }
      var depth = parseDepth(tr);
      var cur = tr.nextElementSibling;
      while (cur && cur.classList.contains("js-perm-tree-node")) {
        if (parseDepth(cur) <= depth) {
          break;
        }
        cur.classList.add("d-none");
        cur = cur.nextElementSibling;
      }
    });
  }

  function isExpandedAllByDom(form) {
    var rows = form.querySelectorAll("tr.js-perm-tree-node");
    for (var i = 0; i < rows.length; i += 1) {
      var tr = rows[i];
      if (parseDepth(tr) > 0 && tr.classList.contains("d-none")) {
        return false;
      }
      if (tr.getAttribute("data-perm-expanded") === "0") {
        return false;
      }
    }
    return true;
  }

  function getDirectChildCheckboxes(form, parentMenuId) {
    if (parentMenuId === null || parentMenuId === undefined) {
      return [];
    }
    var key = escId(String(parentMenuId));
    var sel =
      '.js-perm-tree-cb-branch[data-parent-id="' +
      key +
      '"], .js-perm-tree-cb-leaf[data-parent-id="' +
      key +
      '"]';
    return Array.prototype.slice.call(form.querySelectorAll(sel));
  }

  function syncBranchFromDirectChildren(form, menuId) {
    if (menuId === null || menuId === undefined || menuId === "") {
      return;
    }
    var branch = form.querySelector(
      '.js-perm-tree-cb-branch[data-id="' + escId(String(menuId)) + '"]'
    );
    if (!branch || branch.disabled) {
      return;
    }
    var children = getDirectChildCheckboxes(form, menuId);
    if (!children.length) {
      branch.checked = false;
      branch.indeterminate = false;
      return;
    }
    var allChecked = children.every(function (c) {
      return c.checked && !c.indeterminate;
    });
    var noneChecked = children.every(function (c) {
      return !c.checked && !c.indeterminate;
    });
    if (allChecked) {
      branch.checked = true;
      branch.indeterminate = false;
    } else if (noneChecked) {
      branch.checked = false;
      branch.indeterminate = false;
    } else {
      branch.checked = false;
      branch.indeterminate = true;
    }
  }

  function syncAllBranchesBottomUp(form) {
    var rows = Array.prototype.slice.call(form.querySelectorAll("tr.js-perm-tree-node"));
    rows.sort(function (a, b) {
      return parseDepth(b) - parseDepth(a);
    });
    rows.forEach(function (tr) {
      var b = tr.querySelector(".js-perm-tree-cb-branch");
      if (b && !b.disabled) {
        syncBranchFromDirectChildren(form, b.getAttribute("data-id"));
      }
    });
  }

  function syncUpwardFromCheckbox(form, checkbox) {
    var pid = checkbox.getAttribute("data-parent-id");
    if (pid === null || pid === undefined || pid === "") {
      return;
    }
    while (pid) {
      syncBranchFromDirectChildren(form, pid);
      var pBranch = form.querySelector(
        '.js-perm-tree-cb-branch[data-id="' + escId(String(pid)) + '"]'
      );
      if (!pBranch) {
        break;
      }
      pid = pBranch.getAttribute("data-parent-id") || "";
    }
  }

  function setBranchSubtreeRecursive(form, menuId, checked) {
    getDirectChildCheckboxes(form, menuId).forEach(function (el) {
      el.checked = checked;
      el.indeterminate = false;
      if (el.classList.contains("js-perm-tree-cb-branch")) {
        setBranchSubtreeRecursive(form, el.getAttribute("data-id"), checked);
      }
    });
  }

  function setBranchDisabled(form, disabled) {
    form.querySelectorAll(".js-perm-tree-cb-branch").forEach(function (el) {
      el.disabled = disabled;
    });
  }

  function isLinkageOn(linkageEl) {
    return linkageEl && linkageEl.checked;
  }

  function getLeaves(form) {
    return Array.prototype.slice.call(form.querySelectorAll(".js-perm-tree-cb-leaf"));
  }

  function allLeavesChecked(form) {
    var leaves = getLeaves(form);
    if (!leaves.length) {
      return false;
    }
    return leaves.every(function (x) {
      return x.checked;
    });
  }

  function updateExpandButton(btn, expandedAll) {
    btn.textContent = expandedAll ? "全部折叠" : "全部展开";
    btn.setAttribute("aria-expanded", expandedAll ? "true" : "false");
  }

  function updateSelectAllButton(btn, form) {
    btn.textContent = allLeavesChecked(form) ? "全不选" : "全选";
  }

  function preparePermFormSubmit(form) {
    form.querySelectorAll(".js-perm-tree-cb-leaf").forEach(function (el) {
      el.disabled = false;
      el.indeterminate = false;
    });
  }

  function initPermTree() {
    var form = document.querySelector(".js-perm-assign-form");
    if (!form || !form.querySelector(".js-perm-tree-table")) {
      return;
    }

    var btnExpand = form.querySelector(".js-perm-tree-tool-expand");
    var btnSelect = form.querySelector(".js-perm-tree-tool-selectall");
    var linkageEl = form.querySelector(".js-perm-tree-linkage");
    var expandedAll = true;

    function onLinkageChange() {
      if (isLinkageOn(linkageEl)) {
        setBranchDisabled(form, false);
        syncAllBranchesBottomUp(form);
      } else {
        setBranchDisabled(form, true);
        form.querySelectorAll(".js-perm-tree-cb-branch").forEach(function (b) {
          b.indeterminate = false;
        });
      }
    }

    form.addEventListener("submit", function () {
      preparePermFormSubmit(form);
    });

    form.addEventListener("click", function (ev) {
      var exp = ev.target.closest(".js-perm-tree-expander");
      if (exp && form.contains(exp)) {
        ev.preventDefault();
        var tr = exp.closest("tr.js-perm-tree-node");
        if (tr) {
          toggleRowExpand(tr);
        }
        return;
      }

      if (ev.target.closest(".js-perm-tree-tool-expand")) {
        expandedAll = !expandedAll;
        applyExpandedState(form, expandedAll);
        if (btnExpand) {
          updateExpandButton(btnExpand, expandedAll);
        }
        return;
      }

      if (ev.target.closest(".js-perm-tree-tool-selectall")) {
        var wantAll = !allLeavesChecked(form);
        getLeaves(form).forEach(function (inp) {
          inp.checked = wantAll;
          inp.indeterminate = false;
        });
        if (isLinkageOn(linkageEl)) {
          syncAllBranchesBottomUp(form);
        }
        if (btnSelect) {
          updateSelectAllButton(btnSelect, form);
        }
      }
    });

    form.querySelectorAll(".js-perm-tree-cb-branch").forEach(function (branch) {
      branch.addEventListener("change", function () {
        if (!isLinkageOn(linkageEl)) {
          return;
        }
        var mid = branch.getAttribute("data-id");
        branch.indeterminate = false;
        setBranchSubtreeRecursive(form, mid, branch.checked);
        syncAllBranchesBottomUp(form);
        if (btnSelect) {
          updateSelectAllButton(btnSelect, form);
        }
      });
    });

    form.querySelectorAll(".js-perm-tree-cb-leaf").forEach(function (leaf) {
      leaf.addEventListener("change", function () {
        leaf.indeterminate = false;
        if (!isLinkageOn(linkageEl)) {
          if (btnSelect) {
            updateSelectAllButton(btnSelect, form);
          }
          return;
        }
        syncUpwardFromCheckbox(form, leaf);
        if (btnSelect) {
          updateSelectAllButton(btnSelect, form);
        }
      });
    });

    if (linkageEl) {
      linkageEl.addEventListener("change", onLinkageChange);
      onLinkageChange();
    }

    syncInitialCollapsedSubtrees(form);
    expandedAll = isExpandedAllByDom(form);
    if (btnExpand) {
      updateExpandButton(btnExpand, expandedAll);
    }
    if (btnSelect) {
      updateSelectAllButton(btnSelect, form);
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initPermTree);
  } else {
    initPermTree();
  }
})();
