(function () {
  "use strict";

  var SCHEDULE_LIST_WINDOWS_URL = "/schedule/windows";
  var MANAGE_MODAL_IDS = ["manageVersionWindowsModal", "manageVersionWindowsOverlay"];

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
      var indexCell = tr.querySelector('[data-field="index"]');
      var nameCell = tr.querySelector('[data-field="name"]');
      var releaseCell = tr.querySelector('[data-field="releaseDate"]');
      var rangeCell = tr.querySelector('[data-field="range"]');
      var demandCell = tr.querySelector('[data-field="demandCount"]');
      var capacityCell = tr.querySelector('[data-field="capacity"]');
      var usedHoursCell = tr.querySelector('[data-field="usedHours"]');
      var remainingHoursCell = tr.querySelector('[data-field="remainingHours"]');
      var blockedCountCell = tr.querySelector('[data-field="blockedCount"]');
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
      if (demandCell) {
        demandCell.textContent = String(Number(item.demandCount || 0));
      }
      if (capacityCell) {
        capacityCell.textContent = String(Number(item.capacityHours || 0));
      }
      if (usedHoursCell) {
        usedHoursCell.textContent = String(Number(item.usedHours || 0));
      }
      if (remainingHoursCell) {
        remainingHoursCell.textContent = String(Number(item.remainingHours || 0));
      }
      if (blockedCountCell) {
        blockedCountCell.textContent = String(Number(item.blockedCount || 0));
      }
      if (actionsCell) {
        var actionsWrap = actionsCell.querySelector(".schedule-manage-actions") || actionsCell;
        var editBtn = buildManageActionBtn("edit", item);
        var deleteBtn = buildManageActionBtn("delete", item);
        if (editBtn) {
          actionsWrap.appendChild(editBtn);
        }
        if (deleteBtn) {
          actionsWrap.appendChild(deleteBtn);
        }
      }
      body.appendChild(tr);
    });
  }

  function isSessionExpiredError(err) {
    return !!(err && err.message === "session expired");
  }

  function scheduleRequestFetch(url, options) {
    var fetchFn = window.scheduleFetch || window.appFetch || fetch;
    return fetchFn(url, options);
  }

  function openManageVersionWindowsModal() {
    if (typeof window.closeAllScheduleWindowCardMenus === "function") {
      window.closeAllScheduleWindowCardMenus();
    }

    scheduleRequestFetch(SCHEDULE_LIST_WINDOWS_URL, {
      method: "GET",
      headers: { Accept: "application/json" },
    })
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
        window.openShowModals(MANAGE_MODAL_IDS);
      })
      .catch(function (err) {
        if (isSessionExpiredError(err)) {
          return;
        }
        if (typeof window.showToast === "function") {
          window.showToast("加载版本窗口列表失败，请稍后重试", "error");
        }
      });
  }

  window.openManageVersionWindowsModal = openManageVersionWindowsModal;
})();
