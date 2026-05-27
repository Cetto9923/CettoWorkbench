/*
 * UI 交互层脚本：仅负责视觉交互与通用组件行为。
 */
(function () {
  "use strict";

  var pendingDeleteUrl = "";
  var pendingDeleteMode = "";

  function getCsrfToken() {
    var el = document.querySelector('meta[name="csrf-token"]');
    return el ? (el.getAttribute("content") || "").trim() : "";
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

    var textNode = document.createElement("span");
    textNode.textContent = text;
    toast.appendChild(textNode);

    if (level === "error") {
      var closeBtn = document.createElement("button");
      closeBtn.textContent = "✕";
      closeBtn.style.cssText = "margin-left:8px;background:none;border:none;cursor:pointer;font-size:12px;color:inherit;padding:0;";
      closeBtn.onclick = function () {
        toast.remove();
      };
      toast.appendChild(closeBtn);
    }

    container.appendChild(toast);
    if (level === "success") {
      window.setTimeout(function () {
        toast.remove();
      }, 3000);
    }
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
    pendingDeleteMode = triggerEl && triggerEl.getAttribute ? (triggerEl.getAttribute("data-delete-mode") || "").trim() : "";
    var displayName = (name || "").trim();
    if (!displayName && triggerEl && triggerEl.getAttribute) {
      displayName = (triggerEl.getAttribute("data-name") || "").trim();
    }
    var title = modal.querySelector(".modal-title");
    if (title) {
      title.textContent = "确认删除「" + displayName + "」？";
    }
    modal.classList.add("open");
    return false;
  }

  function closeModal(id) {
    var modal = document.getElementById(id);
    if (modal) {
      modal.classList.remove("open");
    }
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

  document.addEventListener("click", function (ev) {
    var t = ev.target;
    if (!t || !t.closest) {
      return;
    }
    if (t.closest(".dropdown")) {
      return;
    }
    closeAllDropdowns();
  });

  function submitDelete() {
    if (!pendingDeleteUrl) {
      showToast("删除地址无效，请刷新后重试", "error");
      closeModal("modal-delete");
      return false;
    }

    if (pendingDeleteMode === "json") {
      var headers = {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-CSRF-Token": getCsrfToken(),
        "X-Requested-With": "XMLHttpRequest"
      };
      var init = {
        method: "DELETE",
        headers: headers,
        body: "{}"
      };
      var req = window.appFetch ? window.appFetch(pendingDeleteUrl, init) : fetch(pendingDeleteUrl, init);
      req
        .then(function (res) {
          return res.json().then(function (data) {
            return { ok: res.ok, status: res.status, data: data };
          });
        })
        .then(function (result) {
          var data = result.data || {};
          if (data.success && data.redirectUrl) {
            closeModal("modal-delete");
            showToast(data.message || "删除成功", "success");
            window.location.href = data.redirectUrl;
            return;
          }
          if (result.status === 401) {
            window.location.href = "/login";
            return;
          }
          showToast(data.message || "删除失败", "error");
        })
        .catch(function () {
          showToast("删除失败，请稍后重试", "error");
        });
      return false;
    }

    var payload = new URLSearchParams();
    payload.append("_method", "DELETE");
    payload.append("csrf_token", getCsrfToken());

    fetch(pendingDeleteUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
        "X-CSRF-Token": getCsrfToken(),
        "X-Requested-With": "XMLHttpRequest"
      },
      body: payload.toString()
    })
      .then(function (res) {
        if (!res.ok) {
          throw new Error("删除失败");
        }
        closeModal("modal-delete");
        showToast("删除成功", "success");
        window.setTimeout(function () {
          window.location.reload();
        }, 250);
      })
      .catch(function () {
        showToast("删除失败，请稍后重试", "error");
      });

    return false;
  }

  window.showToast = showToast;
  window.confirmDelete = confirmDelete;
  window.closeModal = closeModal;
  window.submitDelete = submitDelete;
  window.toggleDropdown = toggleDropdown;
  window.closeAllDropdowns = closeAllDropdowns;
})();
