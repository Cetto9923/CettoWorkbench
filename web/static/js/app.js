/*
 * workbench 全项目唯一前端脚本。
 * 原则：能用表单 PRG 就不用 JS。
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
  }

  window.appFetch = appFetch;
  window.openModal = openModal;
  bindFormLoading();
  bindConfirmAction();
  bindPagerPageSize();
  bindRolePillPicker();
  showSuccessFlashPopup();
  bindAsyncSearch();
  bindBatchActions();
})();
