(function () {
  "use strict";

  var SCHEDULE_LIST_WINDOWS_URL = "/po/schedule/windows";

  function getCsrfToken() {
    var el = document.querySelector('meta[name="csrf-token"]');
    return el ? String(el.getAttribute("content") || "").trim() : "";
  }

  function scheduleWindowEditDisabledTip(canEdit) {
    return canEdit ? "" : "非本人创建，无法维护";
  }

  function scheduleWindowDeleteDisabledTip(item) {
    if (!item || item.canDelete) {
      return "";
    }
    if (!item.canEdit) {
      return "非本人创建，无法维护";
    }
    if (item.hasLinkedDemands) {
      return "已关联需求，无法删除";
    }
    return "无法删除";
  }

  function getManageWindowStatusTagClass(statusLabel) {
    var label = String(statusLabel || "").trim();
    if (label === "当前") {
      return "st-progress";
    }
    if (label === "规划中") {
      return "st-pending";
    }
    if (label === "已发布") {
      return "st-success";
    }
    return "st-gray";
  }

  function cloneManageTemplate(id) {
    var tpl = document.getElementById(id);
    if (!tpl || !tpl.content || !tpl.content.firstElementChild) {
      return null;
    }
    return document.importNode(tpl.content.firstElementChild, true);
  }

  function buildManageActionBtn(type, item) {
    var enabled = type === "edit" ? !!item.canEdit : !!item.canDelete;
    var tip = type === "edit" ? scheduleWindowEditDisabledTip(item.canEdit) : scheduleWindowDeleteDisabledTip(item);
    var tplId = type === "edit" ? "scheduleManageEditBtn" : "scheduleManageDeleteBtn";
    var btn = cloneManageTemplate(tplId);
    if (!btn) {
      return null;
    }
    btn.dataset.windowId = String(item.id);
    btn.dataset.windowName = item.name || "";
    if (!enabled) {
      btn.classList.add("action-btn--disabled");
      btn.disabled = true;
      if (tip) {
        btn.title = tip;
      }
    }
    return btn;
  }

  function renderManageVersionWindowsTable(windows) {
    var body = document.getElementById("manageVersionWindowsBody");
    var countEl = document.getElementById("manageWindowCount");
    if (!body) {
      return;
    }
    body.textContent = "";
    var rows = windows || [];
    if (countEl) {
      countEl.textContent = String(rows.length);
    }
    if (!rows.length) {
      return;
    }

    rows.forEach(function (item, index) {
      var tr = cloneManageTemplate("scheduleManageWindowRow");
      if (!tr) {
        return;
      }
      var statusLabel = String(item.status || "").trim();
      var statusTag = tr.querySelector('[data-field="status"]');
      var indexCell = tr.querySelector('[data-field="index"]');
      var nameCell = tr.querySelector('[data-field="name"]');
      var releaseCell = tr.querySelector('[data-field="releaseDate"]');
      var rangeCell = tr.querySelector('[data-field="range"]');
      var capacityCell = tr.querySelector('[data-field="capacity"]');
      var actionsCell = tr.querySelector('[data-field="actions"]');

      if (indexCell) {
        indexCell.textContent = String(index + 1);
      }
      if (nameCell) {
        nameCell.textContent = item.name || "";
      }
      if (releaseCell) {
        releaseCell.textContent = item.releaseDate || "—";
      }
      if (rangeCell) {
        rangeCell.textContent = item.range || "—";
      }
      if (statusTag) {
        statusTag.textContent = statusLabel;
        statusTag.classList.add(getManageWindowStatusTagClass(statusLabel));
      }
      if (capacityCell) {
        capacityCell.textContent = String(Number(item.capacityHours || 0));
      }
      if (actionsCell) {
        var editBtn = buildManageActionBtn("edit", item);
        var deleteBtn = buildManageActionBtn("delete", item);
        if (editBtn) {
          actionsCell.appendChild(editBtn);
        }
        if (deleteBtn) {
          actionsCell.appendChild(deleteBtn);
        }
      }
      body.appendChild(tr);
    });
  }

  function openManageVersionWindowsModal() {
    if (typeof window.closeAllScheduleWindowCardMenus === "function") {
      window.closeAllScheduleWindowCardMenus();
    }
    var fetchFn = window.appFetch || fetch;
    var headers = { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" };
    var csrf = getCsrfToken();
    if (csrf) {
      headers["X-CSRF-Token"] = csrf;
    }

    fetchFn(SCHEDULE_LIST_WINDOWS_URL, { method: "GET", headers: headers })
      .then(function (resp) {
        return resp
          .json()
          .catch(function () {
            return {};
          })
          .then(function (data) {
            return { ok: resp.ok, data: data || {} };
          });
      })
      .then(function (result) {
        if (!result.data || !result.data.success) {
          if (typeof window.showToast === "function") {
            window.showToast((result.data && result.data.error) || "加载版本窗口列表失败", "error");
          }
          return;
        }
        renderManageVersionWindowsTable(result.data.windows || []);
        document.getElementById("manageVersionWindowsModal").classList.add("show");
        document.getElementById("manageVersionWindowsOverlay").classList.add("show");
      })
      .catch(function () {
        if (typeof window.showToast === "function") {
          window.showToast("加载版本窗口列表失败，请稍后重试", "error");
        }
      });
  }

  function closeManageVersionWindowsModal() {
    var modal = document.getElementById("manageVersionWindowsModal");
    var overlay = document.getElementById("manageVersionWindowsOverlay");
    if (modal) {
      modal.classList.remove("show");
    }
    if (overlay) {
      overlay.classList.remove("show");
    }
  }

  window.openManageVersionWindowsModal = openManageVersionWindowsModal;
  window.closeManageVersionWindowsModal = closeManageVersionWindowsModal;
})();
