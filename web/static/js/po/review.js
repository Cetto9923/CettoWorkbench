/*
 * 文件: web/static/js/po/review.js
 * 模块: PO工作台
 * 职责: 业需审批右侧抽屉；通过/驳回提交 POST /demands/:id/review。
 */
(function ($) {
  "use strict";

  var Common = window.PoDemandDrawer;
  if (!Common) {
    return;
  }

  var drawer = Common.create("poDemandReview");
  var REJECT_MODAL_ID = "poDemandReviewRejectModal";
  var PASS_MODAL_ID = "poDemandReviewPassModal";
  var currentItem = null;
  var detailCtrl = { abort: null };
  var submitting = false;

  function reviewFailMessage(res, data, text) {
    if (data) {
      var fromApi = String(data.message || data.error || "").trim();
      if (fromApi) {
        return fromApi;
      }
    }
    if (res && res.status === 401) {
      return "未登录或会话已过期，请刷新后重试";
    }
    if (res && res.status === 403) {
      return "没有评审权限";
    }
    if (text && /CSRF/i.test(text)) {
      return "安全校验失败，请刷新页面后重试";
    }
    if (res && res.status) {
      return "评审失败（HTTP " + res.status + "）";
    }
    return "评审失败";
  }

  function setActionButtonsDisabled(disabled) {
    $(
      "#poDemandReviewPassBtn, #poDemandReviewRejectBtn, #poDemandReviewRejectConfirmBtn, #poDemandReviewPassConfirmBtn"
    ).prop("disabled", !!disabled);
  }

  function submitReview(result, comment) {
    var id = Common.demandNumericId(currentItem);
    if (!id) {
      Common.showToast("需求 ID 无效", "error");
      return;
    }
    if (submitting) {
      return;
    }
    submitting = true;
    setActionButtonsDisabled(true);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(id) + "/review", {
      method: "POST",
      headers: Common.csrfHeaders(),
      body: JSON.stringify({
        result: result,
        comment: comment || ""
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
          Common.showToast(data.message || "评审成功", "success");
          closeRejectModal();
          closePassModal();
          closeDrawer();
          if (typeof window.refreshPoHomeDemands === "function") {
            window.refreshPoHomeDemands();
          }
          return;
        }
        Common.showToast(reviewFailMessage(wrap.res, data, wrap.text), "error");
      })
      .catch(function () {
        Common.showToast("评审失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setActionButtonsDisabled(false);
      });
  }

  function openRejectModal() {
    closePassModal();
    drawer.showModal(REJECT_MODAL_ID);
    var ta = document.getElementById("poDemandReviewRejectComment");
    if (ta) {
      ta.value = "";
      ta.focus();
    }
  }

  function closeRejectModal() {
    drawer.hideModal(REJECT_MODAL_ID);
  }

  function openPassModal() {
    closeRejectModal();
    drawer.showModal(PASS_MODAL_ID);
    var confirmBtn = document.getElementById("poDemandReviewPassConfirmBtn");
    if (confirmBtn) {
      confirmBtn.focus();
    }
  }

  function closePassModal() {
    drawer.hideModal(PASS_MODAL_ID);
  }

  function closeDrawer() {
    closeRejectModal();
    closePassModal();
    if (detailCtrl.abort && typeof detailCtrl.abort.abort === "function") {
      detailCtrl.abort.abort();
      detailCtrl.abort = null;
    }
    drawer.closeDrawer();
    currentItem = null;
  }

  function openDrawer(item) {
    currentItem = item || null;
    if (!drawer.openDrawer(item)) {
      return;
    }
    drawer.fetchDetail(item, detailCtrl);
  }

  function bindEvents() {
    var mask = document.getElementById("poDemandReviewDrawer");
    if (!mask) {
      return;
    }

    $("#poDemandReviewDrawerCloseBtn").on("click", closeDrawer);
    $(mask).on("click", function (e) {
      if (e.target === mask) {
        closeDrawer();
      }
    });

    $("#poDemandReviewRejectBtn").on("click", openRejectModal);
    $("#poDemandReviewRejectCloseBtn, #poDemandReviewRejectCancelBtn").on("click", closeRejectModal);
    $("#poDemandReviewRejectConfirmBtn").on("click", function () {
      var comment = String($("#poDemandReviewRejectComment").val() || "").trim();
      if (!comment) {
        Common.showToast("驳回时请输入评审意见", "warning");
        $("#poDemandReviewRejectComment").focus();
        return;
      }
      submitReview("refuse", comment);
    });
    $("#poDemandReviewPassBtn").on("click", openPassModal);
    $("#poDemandReviewPassCloseBtn, #poDemandReviewPassCancelBtn").on("click", closePassModal);
    $("#poDemandReviewPassConfirmBtn").on("click", function () {
      submitReview("pass", "");
    });

    $(document).on("keydown.poDemandReview", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var passModal = document.getElementById(PASS_MODAL_ID);
      if (passModal && passModal.style.display === "flex") {
        closePassModal();
        return;
      }
      var rejectModal = document.getElementById(REJECT_MODAL_ID);
      if (rejectModal && rejectModal.style.display === "flex") {
        closeRejectModal();
        return;
      }
      if (mask.classList.contains("active")) {
        closeDrawer();
      }
    });
  }

  window.openPoDemandReviewDrawer = openDrawer;
  window.closePoDemandReviewDrawer = closeDrawer;

  $(bindEvents);
})(jQuery);
