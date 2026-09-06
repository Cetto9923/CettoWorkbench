/*
 * 文件: web/static/js/po/homereview.js
 * 模块: PO工作台
 * 职责: 业需评审弹窗提交到 POST /demands/:id/review（对齐禅道 demand/review 字段）。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poDemandReviewModal", "poDemandReviewOverlay"];

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  var currentItem = null;

  function csrfHeaders() {
    var headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest"
    };
    var token = "";
    var meta = document.querySelector('meta[name="csrf-token"]');
    if (meta) {
      token = (meta.getAttribute("content") || "").trim();
    }
    if (token) {
      headers["X-CSRF-Token"] = token;
    }
    return headers;
  }

  function closeModal() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function openModal(item) {
    currentItem = item || null;
    var id = String((item && item.id) || "").trim();
    var title = String((item && item.title) || "").trim();
    $("#poDemandReviewName").text(title);
    $("#poDemandReviewId").text(id);
    $("#poDemandReviewResult").val("pass");
    $("#poDemandReviewFocus").val("0");
    $("#poDemandReviewComment").val("");
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
  }

  function bindModal() {
    $("#poDemandReviewCloseBtn, #poDemandReviewOverlay").on("click", function () {
      closeModal();
    });
    $("#poDemandReviewForm").on("submit", function (e) {
      e.preventDefault();
      var id = currentItem && currentItem.id ? String(currentItem.id).trim() : "";
      var result = $("#poDemandReviewResult").val();
      var focus = $("#poDemandReviewFocus").val();
      if (!id) {
        showToast("需求 ID 无效", "error");
        return;
      }
      if (!result || focus === "") {
        showToast("请填写必填项", "error");
        return;
      }
      var $btn = $("#poDemandReviewSubmitBtn");
      $btn.prop("disabled", true);
      var fetchFn = window.appFetch || fetch;
      fetchFn("/demands/" + encodeURIComponent(id) + "/review", {
        method: "POST",
        headers: csrfHeaders(),
        body: JSON.stringify({
          result: result,
          isNeedFocus: focus,
          comment: $("#poDemandReviewComment").val() || ""
        })
      })
        .then(function (res) {
          return res.text().then(function (text) {
            var data = {};
            try {
              data = text ? JSON.parse(text) : {};
            } catch (ignore) {
              data = { message: res.status === 403 ? "没有评审权限" : "评审失败" };
            }
            return { ok: res.ok, data: data };
          });
        })
        .then(function (resultWrap) {
          var data = resultWrap.data;
          if (resultWrap.ok && data.success) {
            showToast(data.message || "评审成功", "success");
            closeModal();
            if (typeof window.refreshPoHomeDemands === "function") {
              window.refreshPoHomeDemands();
            }
            return;
          }
          showToast(data.message || "评审失败", "error");
        })
        .catch(function () {
          showToast("评审失败，请稍后重试", "error");
        })
        .then(function () {
          $btn.prop("disabled", false);
        });
    });
  }

  window.openPoDemandReviewModal = openModal;

  $(bindModal);
})(jQuery);
