/*
 * 文件: web/static/js/po/deliver.js
 * 模块: PO工作台
 * 职责: 发起交付弹窗开关与静态检索下拉（不接提交接口）
 */
(function ($) {
  "use strict";

  var LAUNCH_WINDOW_OTHER = "__launch_other__";
  var bound = false;

  var WINDOW_OPTIONS = [{ value: LAUNCH_WINDOW_OTHER, label: "＋ 选择其他上线时间" }];
  var GRAY_OPTIONS = [
    { value: "1", label: "是" },
    { value: "0", label: "否" }
  ];
  var VERIFY_DATE_OPTIONS = [
    { value: "1", label: "当日验证" },
    { value: "2", label: "次日验证" },
    { value: "3", label: "首笔验证" }
  ];

  function demandIdOf(value) {
    var m = String(value == null ? "" : value)
      .trim()
      .match(/(?:US|REQ|DEMAND)?[-#]?(\d+)/i);
    return m ? m[1] : "";
  }

  function initSearchSelects() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("poLaunchWindowInput", "poLaunchWindowSelect", WINDOW_OPTIONS, {
      placeholder: "搜索上线窗口"
    });
    window.initAutocomplete("poDeliverGrayPlanInput", "poDeliverGrayPlan", GRAY_OPTIONS, {
      value: "0",
      label: "否",
      labelOnly: true,
      placeholder: "搜索"
    });
    window.initAutocomplete("poDeliverVerifyDateInput", "poDeliverVerifyDate", VERIFY_DATE_OPTIONS, {
      value: "1",
      label: "当日验证",
      labelOnly: true,
      placeholder: "搜索"
    });
    window.initAutocomplete("poDeliverVerifierInput", "poDeliverVerifierValue", [], {
      placeholder: "输入姓名或工号搜索"
    });
  }

  function destroySearchSelects() {
    if (typeof window.destroyAutocomplete !== "function") {
      return;
    }
    window.destroyAutocomplete("poLaunchWindowInput");
    window.destroyAutocomplete("poDeliverGrayPlanInput");
    window.destroyAutocomplete("poDeliverVerifyDateInput");
    window.destroyAutocomplete("poDeliverVerifierInput");
  }

  function syncLaunchDateField() {
    var raw = String($("#poLaunchWindowSelect").val() || "");
    var $wrap = $("#poLaunchDateFieldWrap");
    var $notice = $("#poLaunchWindowNotice");
    if (raw === LAUNCH_WINDOW_OTHER) {
      $wrap.removeAttr("hidden");
      $notice
        .text("当前日期暂无上线窗口，保存后将自动创建窗口并关联本次交付")
        .removeAttr("hidden");
    } else {
      $wrap.attr("hidden", true);
      $notice.attr("hidden", true).text("");
    }
  }

  function openPoDeliverModal(idOrItem) {
    var raw = idOrItem && typeof idOrItem === "object" ? idOrItem.id : idOrItem;
    var numeric = demandIdOf(raw);
    var overlay = document.getElementById("poDeliverModalOverlay");
    if (!overlay) {
      return false;
    }
    $("#poDeliverModalTitle").text(numeric ? "发起交付 · US" + numeric : "发起交付");
    $(overlay).addClass("show").attr("aria-hidden", "false");
    $("#poDeliverLoadingState, #poDeliverErrorState").attr("hidden", true);
    $("#poDeliverForm").removeAttr("hidden");
    initSearchSelects();
    syncLaunchDateField();
    return false;
  }

  function closePoDeliverModal() {
    destroySearchSelects();
    var overlay = document.getElementById("poDeliverModalOverlay");
    if (overlay) {
      $(overlay).removeClass("show").attr("aria-hidden", "true");
    }
  }

  function bindOnce() {
    if (bound) {
      return;
    }
    bound = true;
    $("#poDeliverCloseBtn, #poDeliverCancelBtn").on("click", closePoDeliverModal);
    $("#poDeliverModalOverlay").on("click", function (e) {
      if (e.target === this) {
        closePoDeliverModal();
      }
    });
    $("#poDeliverForm").on("submit", function (e) {
      e.preventDefault();
    });
    $(document).on(
      "click",
      '.ui-autocomplete-dropdown[data-autocomplete-for="poLaunchWindowInput"] .ui-autocomplete-option',
      function () {
        setTimeout(syncLaunchDateField, 0);
      }
    );
    $("#poLaunchWindowInput").on("blur input", function () {
      setTimeout(syncLaunchDateField, 50);
    });
  }

  $(bindOnce);

  window.openPoDeliverModal = openPoDeliverModal;
  window.closePoDeliverModal = closePoDeliverModal;
})(window.jQuery);
