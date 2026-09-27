/*
 * 文件: web/static/js/po/submitreview.js
 * 模块: PO工作台
 * 职责: 业需提交评审右侧抽屉；确认后 POST /demands/:id/submit-review 代理禅道。
 *       业务评审人：数据源为详情 businessReviewers（需求池）；initAutocomplete + 标签多人会签。
 */
(function ($) {
  "use strict";

  var Common = window.PoDemandDrawer;
  if (!Common) {
    return;
  }

  var drawer = Common.create("poDemandSubmitReview");
  var DRAWER_ID = "poDemandSubmitReviewDrawer";
  var MODAL_ID = "poDemandSubmitReviewModal";
  var INPUT_ID = "poDemandSubmitReviewReviewerInput";
  var HIDDEN_ID = "poDemandSubmitReviewReviewerValue";
  var SELECTED_ID = "poDemandSubmitReviewSelected";
  var MAX_SHOW = 100;

  var currentItem = null;
  var currentDetail = null;
  var detailCtrl = { abort: null };
  var submitting = false;
  var selectedAccounts = [];
  var userOptions = [];
  var clearingPicker = false;
  var valueBound = false;

  // 下拉数据源：业需所属需求池的业务评审人（详情接口 businessReviewers，对齐禅道 demand-submit）。
  function buildUserOptions() {
    var list =
      currentDetail && Array.isArray(currentDetail.businessReviewers)
        ? currentDetail.businessReviewers
        : [];
    return list
      .map(function (u) {
        var account = String((u && u.account) || "").trim();
        if (!account) {
          return null;
        }
        var label = String((u && u.realname) || "").trim() || account;
        return { value: account, label: label };
      })
      .filter(Boolean);
  }

  // 回显：仅保留仍在池内名单中的已选评审人；不在池内的不补、不回显。
  function demandReviewerAccounts(poolOptions) {
    if (!currentDetail || !Array.isArray(currentDetail.reviewerAccounts)) {
      return [];
    }
    var allowed = {};
    (poolOptions || []).forEach(function (u) {
      if (u && u.value) {
        allowed[u.value] = true;
      }
    });
    return currentDetail.reviewerAccounts
      .map(function (account) {
        return String(account || "").trim();
      })
      .filter(function (account) {
        return account && allowed[account];
      });
  }

  function accountLabel(account) {
    var acc = String(account || "").trim();
    for (var i = 0; i < userOptions.length; i++) {
      if (userOptions[i].value === acc) {
        return userOptions[i].label || acc;
      }
    }
    return acc;
  }

  function renderChips() {
    var host = document.getElementById(SELECTED_ID);
    if (!host) {
      return;
    }
    host.innerHTML = selectedAccounts
      .map(function (account) {
        var name = accountLabel(account);
        return (
          '<span class="dd-review-chip">' +
          Common.escapeHtml(name) +
          '<button type="button" class="dd-review-chip-remove" data-reviewer-account="' +
          Common.escapeHtml(account) +
          '" aria-label="移除 ' +
          Common.escapeHtml(name) +
          '">×</button></span>'
        );
      })
      .join("");
  }

  function syncPickerSelectedValues() {
    if (typeof window.setAutocompleteSelectedValues === "function") {
      window.setAutocompleteSelectedValues(INPUT_ID, selectedAccounts);
    }
  }

  function addReviewer(account) {
    account = String(account || "").trim();
    if (!account || selectedAccounts.indexOf(account) >= 0) {
      return;
    }
    selectedAccounts.push(account);
    renderChips();
    syncPickerSelectedValues();
  }

  function removeReviewer(account) {
    account = String(account || "").trim();
    selectedAccounts = selectedAccounts.filter(function (item) {
      return item !== account;
    });
    renderChips();
    syncPickerSelectedValues();
  }

  function getSelectedReviewers() {
    return selectedAccounts.slice();
  }

  function destroyReviewerPicker() {
    if (typeof window.destroyAutocomplete === "function") {
      window.destroyAutocomplete(INPUT_ID);
    }
  }

  function remountReviewerPicker() {
    userOptions = buildUserOptions();
    var preselected = demandReviewerAccounts(userOptions);
    selectedAccounts = [];
    preselected.forEach(function (account) {
      addReviewer(account);
    });
    renderChips();

    destroyReviewerPicker();
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete(INPUT_ID, HIDDEN_ID, userOptions, {
      placeholder: "输入姓名或工号搜索",
      maxShow: MAX_SHOW,
      value: "",
      label: "",
      selectedValues: selectedAccounts.slice()
    });

    if (!valueBound) {
      valueBound = true;
      var hidden = document.getElementById(HIDDEN_ID);
      if (hidden) {
        hidden.addEventListener("change", function () {
          if (clearingPicker) {
            return;
          }
          var account = String(hidden.value || "").trim();
          if (!account) {
            return;
          }
          addReviewer(account);
          if (typeof window.clearAutocomplete === "function") {
            clearingPicker = true;
            try {
              window.clearAutocomplete(INPUT_ID);
            } finally {
              clearingPicker = false;
            }
          }
        });
      }
    }
  }

  function openSubmitModal() {
    var modal = document.getElementById(MODAL_ID);
    if (!modal) {
      return;
    }
    remountReviewerPicker();
    var ta = document.getElementById("poDemandSubmitReviewComment");
    if (ta) {
      ta.value = "";
    }
    drawer.showModal(MODAL_ID);
  }

  function closeSubmitModal() {
    destroyReviewerPicker();
    drawer.hideModal(MODAL_ID);
  }

  function submitFailMessage(res, data, text) {
    if (data && data.message) {
      return data.message;
    }
    if (data && Array.isArray(data.errors) && data.errors.length) {
      var first = data.errors[0];
      if (first && first.message) {
        return first.message;
      }
    }
    if (text && String(text).trim()) {
      return String(text).trim().slice(0, 200);
    }
    if (res && res.status) {
      return "提交评审失败（HTTP " + res.status + "）";
    }
    return "提交评审失败";
  }

  function setSubmitButtonsDisabled(disabled) {
    $("#poDemandSubmitReviewConfirmBtn, #poDemandSubmitReviewOpenModalBtn").prop("disabled", !!disabled);
  }

  function confirmSubmit() {
    var reviewers = getSelectedReviewers();
    if (!reviewers.length) {
      Common.showToast("请至少选择一位业务评审人", "warning");
      return;
    }
    var id = Common.demandNumericId(currentItem);
    if (!id) {
      Common.showToast("需求 ID 无效", "error");
      return;
    }
    if (submitting) {
      return;
    }

    var comment = "";
    var ta = document.getElementById("poDemandSubmitReviewComment");
    if (ta) {
      comment = String(ta.value || "").trim();
    }

    submitting = true;
    setSubmitButtonsDisabled(true);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(id) + "/submit-review", {
      method: "POST",
      headers: Common.csrfHeaders(),
      body: JSON.stringify({
        reviewer: reviewers,
        comment: comment
      })
    })
      .then(function (res) {
        return res.text().then(function (text) {
          var data = {};
          try {
            data = text ? JSON.parse(text) : {};
          } catch (ignore) {
            data = {};
          }
          return { ok: res.ok, data: data, res: res, text: text };
        });
      })
      .then(function (wrap) {
        var data = wrap.data;
        if (wrap.ok && data.success) {
          Common.showToast(data.message || "提交评审成功", "success");
          closeSubmitModal();
          closeDrawer();
          if (typeof window.refreshPoHomeDemands === "function") {
            window.refreshPoHomeDemands();
          }
          return;
        }
        Common.showToast(submitFailMessage(wrap.res, data, wrap.text), "error");
      })
      .catch(function () {
        Common.showToast("提交评审失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setSubmitButtonsDisabled(false);
      });
  }

  function closeDrawer() {
    closeSubmitModal();
    if (detailCtrl.abort && typeof detailCtrl.abort.abort === "function") {
      detailCtrl.abort.abort();
      detailCtrl.abort = null;
    }
    drawer.closeDrawer();
    currentItem = null;
    currentDetail = null;
  }

  function openDrawer(item) {
    currentItem = item || null;
    currentDetail = null;
    if (!drawer.openDrawer(item)) {
      return;
    }
    drawer.fetchDetail(item, detailCtrl, function (data) {
      currentDetail = data || null;
    });
  }

  function bindEvents() {
    var mask = document.getElementById(DRAWER_ID);
    if (!mask) {
      return;
    }

    $("#poDemandSubmitReviewDrawerCloseBtn").on("click", closeDrawer);
    $(mask).on("click", function (e) {
      if (e.target === mask) {
        closeDrawer();
      }
    });

    $("#poDemandSubmitReviewOpenModalBtn").on("click", openSubmitModal);
    $("#poDemandSubmitReviewModalCloseBtn, #poDemandSubmitReviewModalCancelBtn").on("click", closeSubmitModal);
    $("#poDemandSubmitReviewConfirmBtn").on("click", confirmSubmit);

    // 点框内空白聚焦检索；标签与输入同一行
    $(document).on("click", "#poDemandSubmitReviewPicker .dd-review-picker-display", function (e) {
      if ($(e.target).closest(".dd-review-chip-remove, .ui-autocomplete-clear").length) {
        return;
      }
      var input = document.getElementById(INPUT_ID);
      if (input) {
        input.focus();
      }
    });

    $(document).on("click", "#" + SELECTED_ID + " .dd-review-chip-remove", function (e) {
      e.preventDefault();
      e.stopPropagation();
      removeReviewer($(this).attr("data-reviewer-account"));
    });

    $(document).on("keydown.poDemandSubmitReview", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var modal = document.getElementById(MODAL_ID);
      if (modal && modal.style.display === "flex") {
        closeSubmitModal();
        return;
      }
      if (mask.classList.contains("active")) {
        closeDrawer();
      }
    });
  }

  window.openPoDemandSubmitReviewDrawer = openDrawer;
  window.closePoDemandSubmitReviewDrawer = closeDrawer;

  $(bindEvents);
})(jQuery);
