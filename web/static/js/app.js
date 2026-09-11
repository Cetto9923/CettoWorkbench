/*
 * workbench 基础共享前端脚本。
 * 职责：提供基础 appFetch、CSRF token 提取、全局表单提交防重与弹窗辅助能力。
 * 注意：各业务模块可拥有独立专用脚本（如 schedule、user、po）；appFetch 返回原始 Response，不自动反序列化 JSON。
 */
(function () {
  "use strict";

  function getCsrfToken() {
    var el = document.querySelector('meta[name="csrf-token"]');
    if (!el) {
      return "";
    }
    return (el.getAttribute("content") || "").trim();
  }

  function appFetch(input, init) {
    var options = init || {};
    var headers = new Headers(options.headers || {});
    var csrf = getCsrfToken();
    if (csrf) {
      headers.set("X-CSRF-Token", csrf);
    }
    headers.set("X-Requested-With", "XMLHttpRequest");
    options.headers = headers;
    return fetch(input, options);
  }

  // JSON 请求统一入口；保留 appFetch 的原始 Response 契约供旧调用方使用。
  function appJson(input, init) {
    var options = Object.assign({}, init || {});
    options.headers = Object.assign({ "Accept": "application/json" }, options.headers || {});
    if (options.body && typeof options.body !== "string" && !(options.body instanceof FormData)) {
      options.headers["Content-Type"] = "application/json";
      options.body = JSON.stringify(options.body);
    }
    return appFetch(input, options).then(function (response) {
      if (response.status === 401 || (response.redirected && String(response.url || "").indexOf("/login") >= 0)) {
        window.location.href = "/login?redirect=" + encodeURIComponent(window.location.pathname);
        throw new Error("session expired");
      }
      return response.text().then(function (text) {
        var payload = null;
        try { payload = text ? JSON.parse(text) : null; } catch (e) { throw new Error("数据格式解析失败"); }
        if (!response.ok) {
          var error = new Error((payload && (payload.error || payload.message)) || "请求失败 (" + response.status + ")");
          error.status = response.status;
          error.payload = payload;
          throw error;
        }
        return payload;
      });
    });
  }

  function setSubmittingState(form) {
    var submitBtn = form.querySelector('button[type="submit"],input[type="submit"]');
    if (!submitBtn || submitBtn.dataset.loading === "1") {
      return;
    }

    var originalDisabled = submitBtn.disabled;
    var originalHtml = submitBtn.innerHTML;
    var originalValue = submitBtn.value;
    submitBtn.dataset.loading = "1";
    submitBtn.disabled = true;

    var spinnerHtml = '<span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>处理中...';
    if (submitBtn.tagName === "INPUT") {
      submitBtn.value = "处理中...";
    } else {
      submitBtn.innerHTML = spinnerHtml;
    }

    window.setTimeout(function () {
      if (submitBtn.dataset.loading !== "1") {
        return;
      }
      submitBtn.dataset.loading = "0";
      submitBtn.disabled = originalDisabled;
      if (submitBtn.tagName === "INPUT") {
        submitBtn.value = originalValue;
      } else {
        submitBtn.innerHTML = originalHtml;
      }
    }, 3000);
  }

  function bindFormLoading() {
    document.addEventListener("submit", function (event) {
      var form = event.target;
      if (!form || form.tagName !== "FORM") {
        return;
      }
      if (event.defaultPrevented) {
        return;
      }
      setSubmittingState(form);
    });
  }

  function bindConfirmAction() {
    document.addEventListener("click", function (event) {
      var el = event.target.closest("[data-confirm]");
      if (!el) {
        return;
      }
      var message = el.getAttribute("data-confirm") || "确认执行该操作？";
      if (!window.confirm(message)) {
        event.preventDefault();
        event.stopPropagation();
      }
    });
  }

  function bindPagerPageSize() {
    document.addEventListener("click", function (event) {
      var option = event.target.closest(".js-page-size-option");
      if (option) {
        var form = option.closest("form");
        if (!form) {
          return;
        }
        var input = form.querySelector(".js-page-size-input");
        if (!input) {
          return;
        }
        input.value = option.getAttribute("data-page-size") || input.value;
        form.submit();
        return;
      }

      var openPickers = document.querySelectorAll(".zen-page-size-picker[open]");
      openPickers.forEach(function (picker) {
        if (!picker.contains(event.target)) {
          picker.removeAttribute("open");
        }
      });
    });
  }

  function bindRolePillPicker() {
    function syncRolePillState(input) {
      if (!input || input.type !== "checkbox") {
        return;
      }
      var pill = input.closest(".zen-role-pill");
      if (!pill) {
        return;
      }
      pill.classList.toggle("is-active", input.checked);
    }

    document.querySelectorAll(".zen-role-pill input[type='checkbox']").forEach(function (input) {
      syncRolePillState(input);
    });

    document.addEventListener("change", function (event) {
      var input = event.target;
      if (!input || !input.matches(".zen-role-pill input[type='checkbox']")) {
        return;
      }
      syncRolePillState(input);
    });
  }

  function showSuccessFlashPopup() {
    var successFlash = document.querySelector(".js-flash-success[data-flash-text]");
    if (!successFlash) {
      return;
    }
    var text = (successFlash.getAttribute("data-flash-text") || "").trim();
    if (!text) {
      return;
    }
    if (typeof window.showToast === "function") {
      window.showToast(text, "success");
    }
  }

  function createDebounceRunner(delayMs) {
    var timer = 0;
    return function (fn) {
      if (timer) {
        window.clearTimeout(timer);
      }
      timer = window.setTimeout(function () {
        fn();
      }, delayMs);
    };
  }

  function renderSearchResult(target, items) {
    if (!target) {
      return;
    }
    if (target.tagName === "DATALIST") {
      target.innerHTML = "";
      items.forEach(function (item) {
        var option = document.createElement("option");
        option.value = item.label || "";
        option.setAttribute("data-id", item.id || "");
        target.appendChild(option);
      });
      return;
    }
    target.innerHTML = "";
    if (!items.length) {
      return;
    }
    var ul = document.createElement("ul");
    ul.className = "list-group list-group-flush";
    items.forEach(function (item) {
      var li = document.createElement("li");
      li.className = "list-group-item py-1 px-2 small";
      li.textContent = item.label || "";
      li.setAttribute("data-id", item.id || "");
      ul.appendChild(li);
    });
    target.appendChild(ul);
  }

  function bindAsyncSearch() {
    var inputs = document.querySelectorAll("[data-search-url]");
    inputs.forEach(function (input) {
      var runDebounce = createDebounceRunner(300);
      input.addEventListener("input", function () {
        runDebounce(function () {
          var baseUrl = (input.getAttribute("data-search-url") || "").trim();
          var targetSelector = (input.getAttribute("data-search-target") || "").trim();
          if (!baseUrl || !targetSelector) {
            return;
          }
          var target = document.querySelector(targetSelector);
          if (!target) {
            return;
          }
          var urlObj = new URL(baseUrl, window.location.origin);
          urlObj.searchParams.set("q", input.value || "");
          appFetch(urlObj.toString(), { method: "GET" })
            .then(function (res) {
              if (!res.ok) {
                throw new Error("搜索请求失败");
              }
              return res.json();
            })
            .then(function (payload) {
              var items = Array.isArray(payload && payload.items) ? payload.items : [];
              renderSearchResult(target, items);
            })
            .catch(function () {
              renderSearchResult(target, []);
              if (typeof window.showToast === "function") {
                window.showToast("搜索失败，请稍后重试", "danger");
              }
            });
        });
      });
    });
  }

  function getRowChecks(scope) {
    return Array.prototype.slice.call(scope.querySelectorAll("[data-row-check]"));
  }

  function syncSelectAllState(scope) {
    var selectAll = scope.querySelector("[data-select-all]");
    if (!selectAll) {
      return;
    }
    var rowChecks = getRowChecks(scope);
    var checkedCount = rowChecks.filter(function (el) {
      return el.checked;
    }).length;
    if (!rowChecks.length) {
      selectAll.checked = false;
      selectAll.indeterminate = false;
      return;
    }
    selectAll.checked = checkedCount === rowChecks.length;
    selectAll.indeterminate = checkedCount > 0 && checkedCount < rowChecks.length;
  }

  function bindBatchActions() {
    document.querySelectorAll("[data-select-all]").forEach(function (selectAll) {
      var scope = selectAll.closest("form,table,.zen-card,.card,body") || document.body;
      selectAll.addEventListener("change", function () {
        var rowChecks = getRowChecks(scope);
        rowChecks.forEach(function (rowCheck) {
          rowCheck.checked = selectAll.checked;
        });
        syncSelectAllState(scope);
      });
      getRowChecks(scope).forEach(function (rowCheck) {
        rowCheck.addEventListener("change", function () {
          syncSelectAllState(scope);
        });
      });
      syncSelectAllState(scope);
    });

    document.addEventListener("click", function (event) {
      var batchBtn = event.target.closest("[data-batch-url]");
      if (!batchBtn) {
        return;
      }
      event.preventDefault();
      var scope = batchBtn.closest("form,table,.zen-card,.card,body") || document.body;
      var selected = getRowChecks(scope)
        .filter(function (el) { return el.checked; })
        .map(function (el) { return (el.value || el.getAttribute("data-id") || "").trim(); })
        .filter(function (val) { return val !== ""; });
      if (!selected.length) {
        if (typeof window.showToast === "function") {
          window.showToast("请先选择至少一条记录", "warning");
        }
        return;
      }
      var confirmText = (batchBtn.getAttribute("data-confirm") || "确认执行批量操作？").trim();
      if (!window.confirm(confirmText)) {
        return;
      }
      var payload = new URLSearchParams();
      selected.forEach(function (id) {
        payload.append("ids", id);
      });
      payload.append("csrf_token", getCsrfToken());
      appFetch(batchBtn.getAttribute("data-batch-url"), {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" },
        body: payload.toString()
      })
        .then(function (res) {
          if (!res.ok) {
            throw new Error("批量操作失败");
          }
          window.location.reload();
        })
        .catch(function () {
          if (typeof window.showToast === "function") {
            window.showToast("批量操作失败，请稍后重试", "danger");
          }
        });
    });
  }

  function openModal(id) {
    var modal = document.getElementById(id);
    if (!modal) {
      return;
    }
    modal.classList.add("open");
    if (!document.body.dataset.modalOverflow) document.body.dataset.modalOverflow = document.body.style.overflow || "";
    document.body.style.overflow = "hidden";
    modal.setAttribute("aria-hidden", "false");
  }

  function closeModal(id) {
    var modal = document.getElementById(id);
    if (!modal) return;
    modal.classList.remove("open");
    modal.setAttribute("aria-hidden", "true");
    if (!document.querySelector(".modal.open, .batch-modal.open")) {
      document.body.style.overflow = document.body.dataset.modalOverflow || "";
      delete document.body.dataset.modalOverflow;
    }
  }

  document.addEventListener("keydown", function (event) {
    if (event.key !== "Escape") return;
    var open = document.querySelectorAll(".modal.open, .batch-modal.open");
    if (!open.length) return;
    var modal = open[open.length - 1];
    if (modal.getAttribute("data-static") === "true") return;
    closeModal(modal.id);
  });

  function bindShellPlaceholderNotice() {
    document.addEventListener("click", function (event) {
      var btn = event.target.closest("button.po-shell-placeholder, button.po-role-tab:not(.active)");
      if (!btn) {
        return;
      }
      event.preventDefault();
      var label = "";
      if (btn.classList.contains("po-role-tab")) {
        label = (btn.textContent || "").trim() + " 工作台";
      } else {
        var textEl = btn.querySelector(".nav-text");
        label = textEl ? textEl.textContent.trim() : (btn.getAttribute("data-upcoming-page") || "该功能");
      }
      if (typeof window.showToast === "function") {
        window.showToast(label + " 正在规划建设中", "info");
      }
    });
  }

  window.appFetch = appFetch;
  window.appJson = appJson;
  window.getCsrfToken = getCsrfToken;
  window.openModal = openModal;
  window.closeModal = closeModal;
  bindFormLoading();
  bindConfirmAction();
  bindPagerPageSize();
  bindRolePillPicker();
  showSuccessFlashPopup();
  bindAsyncSearch();
  bindBatchActions();
  bindShellPlaceholderNotice();
  bindZentaoNewTabGuard();

  // 禅道外链一律新标签打开，避免工作台当前页被带走。
  function bindZentaoNewTabGuard() {
    document.addEventListener("click", function (event) {
      if (event.defaultPrevented) return;
      if (event.button != null && event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      var a = event.target && event.target.closest ? event.target.closest("a[href]") : null;
      if (!a) return;
      var href = a.getAttribute("href") || "";
      if (!href || href.charAt(0) === "#" || href.toLowerCase().indexOf("javascript:") === 0) return;
      var meta = document.querySelector('meta[name="zentao-url"]');
      var base = meta ? String(meta.getAttribute("content") || "").replace(/\/+$/, "") : "";
      if (!base) return;
      var abs;
      var zt;
      try {
        abs = new URL(href, window.location.href);
        zt = new URL(base);
      } catch (err) {
        return;
      }
      if (abs.origin !== zt.origin) return;
      if (String(a.target || "").toLowerCase() === "_blank") return;
      event.preventDefault();
      window.open(abs.href, "_blank", "noopener,noreferrer");
    }, true);
  }
})();
