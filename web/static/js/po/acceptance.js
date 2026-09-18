/*
 * 文件: web/static/js/po/acceptance.js
 * 模块: PO工作台
 * 职责: 业需验收右侧抽屉；确认后 POST /demands/:id/acceptance 代理禅道。
 */
(function ($) {
  "use strict";

  var Common = window.PoDemandDrawer;
  if (!Common) {
    return;
  }

  var drawer = Common.create("poDemandAcceptance");
  var DRAWER_ID = "poDemandAcceptanceDrawer";
  var MODAL_ID = "poDemandAcceptanceModal";
  var RESULT_OPTIONS = [
    { value: "yes", label: "通过" },
    { value: "no", label: "不通过" }
  ];

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

  function findUserOption(account) {
    var acc = String(account || "").trim();
    if (!acc) {
      return null;
    }
    var hit = buildUserOptions().find(function (o) {
      return o.value === acc;
    });
    if (hit) {
      return hit;
    }
    return { value: acc, label: acc };
  }

  function defaultAssignedAccount() {
    if (currentDetail && currentDetail.braAccount) {
      return String(currentDetail.braAccount || "").trim();
    }
    return "";
  }

  function resetComment() {
    var ta = document.getElementById("poDemandAcceptanceComment");
    if (ta) {
      ta.value = "";
    }
  }

  // 与 scheduleIntRDInput 相同：initAutocomplete 单选检索
  function initModalSelects() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete(
      "poDemandAcceptanceResultInput",
      "poDemandAcceptanceResultValue",
      RESULT_OPTIONS,
      {
        value: "yes",
        label: "通过",
        labelOnly: true,
        placeholder: "选择验收结果"
      }
    );

    var assigned = defaultAssignedAccount();
    var assignedOpt = findUserOption(assigned);
    var userOpts = buildUserOptions();
    if (assignedOpt && !userOpts.some(function (o) { return o.value === assignedOpt.value; })) {
      userOpts = [assignedOpt].concat(userOpts);
    }
    window.initAutocomplete(
      "poDemandAcceptanceAssignedInput",
      "poDemandAcceptanceAssignedValue",
      userOpts,
      assignedOpt
        ? {
            value: assignedOpt.value,
            label: assignedOpt.label,
            placeholder: "输入姓名或工号搜索"
          }
        : { placeholder: "输入姓名或工号搜索" }
    );
  }

  function destroyModalSelects() {
    if (typeof window.destroyAutocomplete !== "function") {
      return;
    }
    window.destroyAutocomplete("poDemandAcceptanceResultInput");
    window.destroyAutocomplete("poDemandAcceptanceAssignedInput");
  }

  function openAcceptanceModal() {
    var modal = document.getElementById(MODAL_ID);
    if (!modal) {
      return;
    }
    destroyModalSelects();
    initModalSelects();
    resetComment();
    drawer.showModal(MODAL_ID);
  }

  function closeAcceptanceModal() {
    destroyModalSelects();
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
      return "验收失败（HTTP " + res.status + "）";
    }
    return "验收失败";
  }

  function setSubmitButtonsDisabled(disabled) {
    $("#poDemandAcceptanceConfirmBtn, #poDemandAcceptanceOpenModalBtn").prop("disabled", !!disabled);
  }

  function confirmAcceptance() {
    var acceptance = String($("#poDemandAcceptanceResultValue").val() || "").trim();
    if (!acceptance) {
      Common.showToast("请选择验收结果", "warning");
      return;
    }
    var assignedTo = String($("#poDemandAcceptanceAssignedValue").val() || "").trim();
    if (!assignedTo) {
      Common.showToast("请选择指派给", "warning");
      return;
    }
    var comment = "";
    var ta = document.getElementById("poDemandAcceptanceComment");
    if (ta) {
      comment = String(ta.value || "").trim();
    }
    if (acceptance === "no" && !comment) {
      Common.showToast("不通过时请填写验收意见", "warning");
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

    submitting = true;
    setSubmitButtonsDisabled(true);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(id) + "/acceptance", {
      method: "POST",
      headers: Common.csrfHeaders(),
      body: JSON.stringify({
        acceptance: acceptance,
        assignedTo: assignedTo,
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
          Common.showToast(data.message || "验收成功", "success");
          closeAcceptanceModal();
          closeDrawer();
          if (typeof window.refreshPoHomeDemands === "function") {
            window.refreshPoHomeDemands();
          }
          return;
        }
        Common.showToast(submitFailMessage(wrap.res, data, wrap.text), "error");
      })
      .catch(function () {
        Common.showToast("验收失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setSubmitButtonsDisabled(false);
      });
  }

  function closeDrawer() {
    closeAcceptanceModal();
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
    window.PoAcceptanceRelations && window.PoAcceptanceRelations.reset("加载中…");
    if (!drawer.openDrawer(item)) {
      return;
    }
    drawer.fetchDetail(
      item,
      detailCtrl,
      function (data) {
        currentDetail = data || null;
        window.PoAcceptanceRelations && window.PoAcceptanceRelations.render(data);
      },
      function () {
        window.PoAcceptanceRelations && window.PoAcceptanceRelations.reset("加载失败");
      }
    );
  }

  function bindEvents() {
    var mask = document.getElementById(DRAWER_ID);
    if (!mask) {
      return;
    }

    $("#poDemandAcceptanceDrawerCloseBtn").on("click", closeDrawer);
    $(mask).on("click", function (e) {
      if (e.target === mask) {
        closeDrawer();
      }
    });

    $("#poDemandAcceptanceOpenModalBtn").on("click", openAcceptanceModal);
    $("#poDemandAcceptanceModalCloseBtn, #poDemandAcceptanceModalCancelBtn").on("click", closeAcceptanceModal);
    $("#poDemandAcceptanceConfirmBtn").on("click", confirmAcceptance);

    $(document).on("keydown.poDemandAcceptance", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var modal = document.getElementById(MODAL_ID);
      if (modal && modal.style.display === "flex") {
        closeAcceptanceModal();
        return;
      }
      if (mask.classList.contains("active")) {
        closeDrawer();
      }
    });
  }

  window.openPoDemandAcceptanceDrawer = openDrawer;
  window.closePoDemandAcceptanceDrawer = closeDrawer;

  $(bindEvents);
})(window.jQuery);
