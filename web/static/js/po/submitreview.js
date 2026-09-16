/*
 * 文件: web/static/js/po/submitreview.js
 * 模块: PO工作台
 * 职责: 业需提交评审右侧抽屉；确认后 POST /demands/:id/submit-review 代理禅道。
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
  var PICKER_ID = "poDemandSubmitReviewPicker";
  var OPTION_LIST_ID = "poDemandSubmitReviewOptionList";
  var currentItem = null;
  var currentDetail = null;
  var detailCtrl = { abort: null };
  var submitting = false;

  function parseUsers() {
    var raw = $("#" + DRAWER_ID).attr("data-users") || "[]";
    try {
      var list = JSON.parse(raw);
      return Array.isArray(list) ? list : [];
    } catch (e) {
      return [];
    }
  }

  function buildUserOptions() {
    return parseUsers()
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

  function demandReviewerAccounts() {
    if (!currentDetail || !Array.isArray(currentDetail.reviewerAccounts)) {
      return [];
    }
    return currentDetail.reviewerAccounts
      .map(function (account) {
        return String(account || "").trim();
      })
      .filter(Boolean);
  }

  function ensureSelectedOptions(users, selectedAccounts) {
    var exists = {};
    users.forEach(function (u) {
      exists[u.value] = true;
    });
    selectedAccounts.forEach(function (account) {
      if (!exists[account]) {
        users.unshift({ value: account, label: account });
        exists[account] = true;
      }
    });
    return users;
  }

  function optionHtml(item, checked) {
    return (
      '<label class="checkbox-label form-multiselect-option">' +
      '<input type="checkbox" class="checkbox form-multiselect-checkbox" value="' +
      Common.escapeHtml(item.value) +
      '"' +
      (checked ? " checked" : "") +
      ">" +
      '<span class="checkbox-box"></span>' +
      '<span class="form-multiselect-name">' +
      Common.escapeHtml(item.label) +
      "</span>" +
      "</label>"
    );
  }

  function getSelectedReviewers() {
    var picker = document.getElementById(PICKER_ID);
    if (!picker) {
      return [];
    }
    var out = [];
    picker.querySelectorAll(".form-multiselect-checkbox:checked").forEach(function (cb) {
      var value = String(cb.value || "").trim();
      if (value) {
        out.push(value);
      }
    });
    return out;
  }

  function remountReviewerMultiselect() {
    var picker = document.getElementById(PICKER_ID);
    var list = document.getElementById(OPTION_LIST_ID);
    if (!picker || !list) {
      return;
    }

    var selectedAccounts = demandReviewerAccounts();
    var selectedSet = {};
    selectedAccounts.forEach(function (account) {
      selectedSet[account] = true;
    });

    var users = ensureSelectedOptions(buildUserOptions(), selectedAccounts);
    list.innerHTML = users
      .map(function (item) {
        return optionHtml(item, !!selectedSet[item.value]);
      })
      .join("");

    // 再次打开时克隆节点，丢掉旧监听后再 init（与提测多选一致）
    if (picker.dataset.formMultiselectInit === "1") {
      var clone = picker.cloneNode(true);
      clone.removeAttribute("data-form-multiselect-init");
      clone.classList.remove("open");
      picker.parentNode.replaceChild(clone, picker);
      picker = clone;
    }

    if (typeof window.initFormComponents === "function") {
      window.initFormComponents(picker);
    }
  }

  function openSubmitModal() {
    var modal = document.getElementById(MODAL_ID);
    if (!modal) {
      return;
    }
    remountReviewerMultiselect();
    var ta = document.getElementById("poDemandSubmitReviewComment");
    if (ta) {
      ta.value = "";
    }
    drawer.showModal(MODAL_ID);
    // 不自动 focus，避免弹窗打开时下拉默认展开
  }

  function closeSubmitModal() {
    var picker = document.getElementById(PICKER_ID);
    if (picker) {
      picker.classList.remove("open");
    }
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

    $(document).on("keydown.poDemandSubmitReview", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var modal = document.getElementById(MODAL_ID);
      if (modal && modal.style.display === "flex") {
        var picker = document.getElementById(PICKER_ID);
        if (picker && picker.classList.contains("open")) {
          picker.classList.remove("open");
          return;
        }
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
