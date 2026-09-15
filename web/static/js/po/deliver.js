/*
 * 文件: web/static/js/po/deliver.js
 * 模块: PO工作台
 * 职责: 发起交付弹窗：检索下拉、表单校验，POST /demands/:id/deliver 代理禅道保存。
 */
(function ($) {
  "use strict";

  var bound = false;
  var submitting = false;
  var currentDemandId = "";

  var GRAY_OPTIONS = [
    { value: "1", label: "是" },
    { value: "0", label: "否" }
  ];
  var VERIFY_DATE_OPTIONS = [
    { value: "1", label: "当日验证" },
    { value: "2", label: "次日验证" },
    { value: "3", label: "首笔验证" }
  ];

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function csrfHeaders() {
    var headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest"
    };
    var meta = document.querySelector('meta[name="csrf-token"]');
    var token = meta ? String(meta.getAttribute("content") || "").trim() : "";
    if (token) {
      headers["X-CSRF-Token"] = token;
    }
    return headers;
  }

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
          label: rd ? name + " · " + rd : name,
          releaseDate: rd
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

  function resetFormDefaults() {
    $("#poLaunchWindowInput").val("");
    $("#poLaunchWindowSelect").val("");
    $('input[name="poDeliverCarReview"][value="0"]').prop("checked", true);
    $("#poDeliverGrayPlan").val("0");
    $("#poDeliverGrayPlanInput").val("否");
    $("#poDeliverVerifyDate").val("1");
    $("#poDeliverVerifyDateInput").val("当日验证");
    $("#poDeliverVerifyPlan").val("");
    $("#poDeliverVerifierValue").val("");
    $("#poDeliverVerifierInput").val("");
  }

  function deliverDateFromWindow(windowId) {
    var id = String(windowId || "").trim();
    if (!id) {
      return "";
    }
    var hit = buildLaunchWindowOptions().find(function (o) {
      return o.value === id;
    });
    return hit ? String(hit.releaseDate || "").trim() : "";
  }

  function readFormData() {
    var car = document.querySelector('input[name="poDeliverCarReview"]:checked');
    var windowId = String($("#poLaunchWindowSelect").val() || "").trim();
    return {
      windowId: windowId,
      deliverDate: deliverDateFromWindow(windowId),
      isCarReview: car ? String(car.value) : "0",
      isGrayVerifyPlan: String($("#poDeliverGrayPlan").val() || "").trim(),
      verifyDate: String($("#poDeliverVerifyDate").val() || "").trim(),
      verifyPlan: String($("#poDeliverVerifyPlan").val() || "").trim(),
      veriFier: String($("#poDeliverVerifierValue").val() || "").trim()
    };
  }

  function validateFormData(data) {
    if (!currentDemandId) {
      showToast("需求 ID 无效", "error");
      return false;
    }
    if (!data.windowId || !data.deliverDate) {
      showToast("请选择上线窗口", "error");
      return false;
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(data.deliverDate)) {
      showToast("上线窗口结束日期无效", "error");
      return false;
    }
    if (data.isGrayVerifyPlan === "") {
      showToast("请选择灰度验证计划", "error");
      return false;
    }
    if (!data.verifyDate || data.verifyDate === "0") {
      showToast("请选择生产验证时间", "error");
      return false;
    }
    if (!data.verifyPlan) {
      showToast("请填写生产验证计划", "error");
      return false;
    }
    if (!data.veriFier) {
      showToast("请选择生产验证责任人", "error");
      return false;
    }
    return true;
  }

  function deliverFailMessage(res, data, text) {
    if (data) {
      var fromApi = String(data.message || data.error || "").trim();
      if (fromApi) {
        return fromApi;
      }
      if (Array.isArray(data.errors) && data.errors.length) {
        var first = data.errors[0];
        if (first && first.message) {
          return String(first.message);
        }
      }
    }
    if (res && res.status === 401) {
      return "未登录或会话已过期，请刷新后重试";
    }
    if (res && res.status === 403) {
      return "没有发起交付权限";
    }
    if (text && /CSRF/i.test(text)) {
      return "安全校验失败，请刷新页面后重试";
    }
    if (res && res.status) {
      return "发起交付失败（HTTP " + res.status + "）";
    }
    return "发起交付失败";
  }

  function setSubmitDisabled(disabled) {
    $("#poDeliverConfirmBtn").prop("disabled", !!disabled);
  }

  function submitPoDeliver() {
    if (submitting) {
      return;
    }
    var data = readFormData();
    if (!validateFormData(data)) {
      return;
    }
    submitting = true;
    setSubmitDisabled(true);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(currentDemandId) + "/deliver", {
      method: "POST",
      headers: csrfHeaders(),
      body: JSON.stringify({
        deliverDate: data.deliverDate,
        isGrayVerifyPlan: data.isGrayVerifyPlan,
        verifyDate: data.verifyDate,
        verifyPlan: data.verifyPlan,
        veriFier: data.veriFier,
        isCarReview: data.isCarReview
      })
    })
      .then(function (res) {
        return res.text().then(function (text) {
          var parsed = {};
          try {
            parsed = text ? JSON.parse(text) : {};
          } catch (ignore) {
            parsed = {};
          }
          return { ok: res.ok, data: parsed, res: res, text: text };
        });
      })
      .then(function (wrap) {
        var resp = wrap.data;
        if (wrap.ok && resp.success) {
          showToast(resp.message || "发起交付成功", "success");
          closePoDeliverModal();
          if (typeof window.refreshPoHomeDemands === "function") {
            window.refreshPoHomeDemands();
          } else if (resp.redirectUrl) {
            window.location.href = resp.redirectUrl;
          }
          return;
        }
        showToast(deliverFailMessage(wrap.res, resp, wrap.text), "error");
      })
      .catch(function () {
        showToast("发起交付失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setSubmitDisabled(false);
      });
  }

  function openPoDeliverModal(idOrItem) {
    var raw = idOrItem && typeof idOrItem === "object" ? idOrItem.id : idOrItem;
    var numeric = demandIdOf(raw);
    var overlay = document.getElementById("poDeliverModalOverlay");
    if (!overlay) {
      return false;
    }
    if (!numeric) {
      showToast("无法识别业务需求编号", "error");
      return false;
    }
    currentDemandId = numeric;
    $("#poDeliverModalTitle").text("发起交付 · US" + numeric);
    $(overlay).addClass("show").attr("aria-hidden", "false");
    $("#poDeliverLoadingState, #poDeliverErrorState").attr("hidden", true);
    $("#poDeliverForm").removeAttr("hidden");
    resetFormDefaults();
    initSearchSelects();
    return false;
  }

  function closePoDeliverModal() {
    destroySearchSelects();
    currentDemandId = "";
    submitting = false;
    setSubmitDisabled(false);
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
      submitPoDeliver();
    });
  }

  $(bindOnce);

  window.openPoDeliverModal = openPoDeliverModal;
  window.closePoDeliverModal = closePoDeliverModal;
})(window.jQuery);
