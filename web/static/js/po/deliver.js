/*
 * 文件: web/static/js/po/deliver.js
 * 模块: PO工作台
 * 职责: 发起交付弹窗开关与检索下拉（上线窗口取 zt_versionwindow 未删除数据）
 */
(function ($) {
  "use strict";

  var bound = false;

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

  function parseJSONAttr(attrName) {
    var raw = $("#poDeliverModalOverlay").attr(attrName) || "[]";
    try {
      var list = JSON.parse(raw);
      return Array.isArray(list) ? list : [];
    } catch (e) {
      return [];
    }
  }

  function parseLaunchWindows() {
    return parseJSONAttr("data-launch-windows");
  }

  function parseUsers() {
    return parseJSONAttr("data-users");
  }

  function buildLaunchWindowOptions() {
    return parseLaunchWindows()
      .map(function (w) {
        var id = Number(w && w.id ? w.id : 0);
        if (!id) {
          return null;
        }
        var name = String((w && w.name) || "").trim() || "窗口" + id;
        var rd = String((w && w.releaseDate) || "").trim();
        return {
          value: String(id),
          label: rd ? name + " · " + rd : name
        };
      })
      .filter(Boolean);
  }

  function buildVerifierOptions() {
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

  function initSearchSelects() {
    if (typeof window.initAutocomplete !== "function") {
      return;
    }
    window.initAutocomplete("poLaunchWindowInput", "poLaunchWindowSelect", buildLaunchWindowOptions(), {
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
    window.initAutocomplete("poDeliverVerifierInput", "poDeliverVerifierValue", buildVerifierOptions(), {
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
  }

  $(bindOnce);

  window.openPoDeliverModal = openPoDeliverModal;
  window.closePoDeliverModal = closePoDeliverModal;
})(window.jQuery);
