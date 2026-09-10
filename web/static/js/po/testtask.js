/*
 * 文件: web/static/js/po/testtask.js
 * 模块: PO工作台
 * 职责: 提测办理四步向导静态交互；首页弹窗打开/关闭。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poTesttaskModal", "poTesttaskOverlay"];
  var currentStep = 1;
  var isJointTest = 0;
  var bound = false;
  var autocompleteBound = false;

  var EXEC_OPTIONS = [
    { value: "P01/E-1001", label: "信贷项目/执行-1001" },
    { value: "P01/E-1002", label: "信贷项目/执行-1002" },
    { value: "P02/E-2001", label: "打印中心项目/执行-2001" }
  ];
  var EXIST_VER_OPTIONS = [
    { value: "v1", label: "20250801-已有版本-A" },
    { value: "v2", label: "20250815-已有版本-B" }
  ];
  var AUTOCOMPLETE_FIELDS = [
    { inputId: "poTtExecInput_1", hiddenId: "poTtExecValue_1", items: EXEC_OPTIONS, placeholder: "搜索执行" },
    { inputId: "poTtExecInput_2", hiddenId: "poTtExecValue_2", items: EXEC_OPTIONS, placeholder: "搜索执行" },
    { inputId: "poTtExistVerInput_1", hiddenId: "poTtExistVerValue_1", items: EXIST_VER_OPTIONS, placeholder: "搜索已有版本" },
    { inputId: "poTtExistVerInput_2", hiddenId: "poTtExistVerValue_2", items: EXIST_VER_OPTIONS, placeholder: "搜索已有版本" }
  ];

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function $root() {
    var $modal = $("#poTesttaskModal");
    if ($modal.length && $modal.hasClass("show")) {
      return $modal.find("#poTesttaskFormRoot");
    }
    var $page = $(".po-testtask-page #poTesttaskFormRoot");
    if ($page.length) {
      return $page;
    }
    return $("#poTesttaskFormRoot").first();
  }

  function updateStepIndicator(step) {
    var $r = $root();
    $r.find("[data-tt-step-dot]").each(function () {
      var n = parseInt($(this).attr("data-tt-step-dot"), 10) || 0;
      $(this).removeClass("is-active is-done");
      if (n < step) {
        $(this).addClass("is-done");
      } else if (n === step) {
        $(this).addClass("is-active");
      }
    });
  }

  function readJointFlag($r) {
    var v = $r.find('input[name="isJointTest"]:checked').val();
    return Number(v) === 1 ? 1 : 0;
  }

  function switchPanel(step) {
    var $r = $root();
    if (!$r.length) {
      return;
    }
    currentStep = step;
    $r.find("[data-tt-panel]").addClass("po-testtask-hidden");
    $r.find('[data-tt-panel="' + step + '"]').removeClass("po-testtask-hidden");
    updateStepIndicator(step);

    $r.find("#poTesttaskPrevBtn").toggleClass("po-testtask-hidden", step === 1);
    $r.find("#poTesttaskNextBtn").toggleClass("po-testtask-hidden", step === 4);
    $r.find("#poTesttaskSubmitBtn").toggleClass("po-testtask-hidden", step !== 4);

    if (step === 1) {
      isJointTest = readJointFlag($r);
    }
    if (step === 2) {
      syncSysUnits($r);
      syncVersionModeOptions($r);
    }
    if (step === 4) {
      isJointTest = readJointFlag($r);
      var joint = isJointTest === 1;
      $r.find("[data-tt-joint-single]").toggleClass("po-testtask-hidden", joint);
      $r.find("[data-tt-joint-global]").toggleClass("po-testtask-hidden", !joint);
    }
  }

  function setVersionMode(unit, isNew) {
    var $r = $root();
    $r.find('[data-tt-new-base="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-exist-base="' + unit + '"]').toggleClass("po-testtask-hidden", isNew);
    $r.find('[data-tt-integrate-wrap="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-link-edit="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-link-readonly="' + unit + '"]').toggleClass("po-testtask-hidden", isNew);
  }

  // 联调模式仅允许「使用已有版本」；非联调恢复「创建新版本 / 使用已有版本」。
  function syncVersionModeOptions($r) {
    $r = $r && $r.length ? $r : $root();
    if (!$r.length) {
      return;
    }
    var joint = readJointFlag($r) === 1;
    [1, 2].forEach(function (unit) {
      var $newRadio = $r.find('input[name="ver' + unit + 'Mode"][value="new"]');
      var $newLabel = $newRadio.closest("label");
      var wasHidden = $newLabel.hasClass("po-testtask-hidden");
      $newLabel.toggleClass("po-testtask-hidden", joint);
      if (joint) {
        $r.find('input[name="ver' + unit + 'Mode"][value="exist"]').prop("checked", true);
        setVersionMode(unit, false);
      } else if (wasHidden) {
        // 刚从联调切回：恢复默认「创建新版本」
        $newRadio.prop("checked", true);
        setVersionMode(unit, true);
      } else {
        var isNew = $r.find('input[name="ver' + unit + 'Mode"]:checked').val() !== "exist";
        setVersionMode(unit, isNew);
      }
    });
    syncExistVerPicker($r, joint);
  }

  // 联调：已有版本为标签多选；非联调：单选检索下拉。
  function syncExistVerPicker($r, joint) {
    $r = $r && $r.length ? $r : $root();
    if (!$r.length) {
      return;
    }
    if (typeof joint !== "boolean") {
      joint = readJointFlag($r) === 1;
    }
    [1, 2].forEach(function (unit) {
      $r.find('[data-tt-exist-single="' + unit + '"]').toggleClass("po-testtask-hidden", joint);
      $r.find('[data-tt-exist-multi="' + unit + '"]').toggleClass("po-testtask-hidden", !joint);
    });
  }

  function initTesttaskMultiselects($r) {
    $r = $r && $r.length ? $r : $root();
    if (!$r.length || typeof window.initFormComponents !== "function") {
      return;
    }
    window.initFormComponents($r[0]);
  }

  function syncSysUnits($r) {
    $r = $r && $r.length ? $r : $root();
    if (!$r.length) {
      return;
    }
    $r.find("[data-tt-sys]").each(function () {
      var unit = $(this).attr("data-tt-sys");
      var on = !!$(this).prop("checked");
      $r.find('[data-tt-unit="' + unit + '"]').toggleClass("po-testtask-hidden", !on);
    });
  }

  function initTesttaskAutocompletes() {
    if (typeof window.initAutocomplete !== "function" || autocompleteBound) {
      return;
    }
    // 表单可能同时存在于页面与弹窗；仅对当前文档中已存在的 input 初始化。
    AUTOCOMPLETE_FIELDS.forEach(function (field) {
      if (!document.getElementById(field.inputId) || !document.getElementById(field.hiddenId)) {
        return;
      }
      window.initAutocomplete(field.inputId, field.hiddenId, field.items, {
        value: "",
        label: "",
        placeholder: field.placeholder
      });
    });
    autocompleteBound = true;
  }

  function fillDemandHeader(item) {
    var id = String((item && item.id) || "").trim();
    var title = String((item && item.title) || "").trim();
    var stage = String((item && (item.valueStream || item.stage)) || "").trim();
    $("#poTesttaskModalDemandId").text(id);
    $("#poTesttaskModalDemandName").text(title);
    var $r = $("#poTesttaskModal #poTesttaskFormRoot");
    if (id) {
      $r.find("[data-tt-demand-id]").text(formatDemandId(id));
    }
    if (title) {
      $r.find("[data-tt-demand-title]").text(title);
    }
    if (stage) {
      $r.find("[data-tt-stage]").text("当前阶段：" + stage);
      $r.find("[data-tt-stage-text]").text(stage);
    }
  }

  function formatDemandId(id) {
    var raw = String(id || "").trim();
    if (!raw) {
      return "";
    }
    if (raw.indexOf("#") === 0) {
      return raw;
    }
    return "#" + raw.replace(/^US/i, "");
  }

  function demandNumericId(item) {
    var raw = String((item && item.id) || "").trim();
    var matched = raw.match(/^US(\d+)$/i);
    if (matched) {
      return matched[1];
    }
    if (raw.indexOf("#") === 0) {
      raw = raw.slice(1);
    }
    if (/^\d+$/.test(raw)) {
      return raw;
    }
    return "";
  }

  function setContextLoading($r) {
    $r.find("[data-tt-raw-status],[data-tt-main-system],[data-tt-estimate-launch],[data-tt-bra-name],[data-tt-rd-qd-name],[data-tt-handler-name]").text("加载中…");
  }

  function fillContextData(data) {
    var $r = $("#poTesttaskModal #poTesttaskFormRoot");
    if (!$r.length || !data) {
      return;
    }
    var demandId = data.demandId != null ? String(data.demandId) : "";
    var title = String(data.title || "").trim();
    var stage = String(data.stage || "").trim();
    if (demandId) {
      var shown = formatDemandId(demandId);
      $r.find("[data-tt-demand-id]").text(shown);
      $("#poTesttaskModalDemandId").text("US" + demandId);
    }
    if (title) {
      $r.find("[data-tt-demand-title]").text(title);
      $("#poTesttaskModalDemandName").text(title);
    }
    if (stage) {
      $r.find("[data-tt-stage]").text("当前阶段：" + stage);
      $r.find("[data-tt-stage-text]").text(stage);
    }
    $r.find("[data-tt-raw-status]").text(data.rawStatus || "—");
    $r.find("[data-tt-main-system]").text(data.mainSystemName || "—");
    $r.find("[data-tt-estimate-launch]").text(data.estimateLaunch || "—");
    $r.find("[data-tt-bra-name]").text(data.braName || "—");
    var rd = String(data.rdName || "").trim();
    var qd = String(data.qdName || "").trim();
    var rdQd = "—";
    if (rd && qd && rd !== "—" && qd !== "—") {
      rdQd = rd + " / " + qd;
    } else if (rd && rd !== "—") {
      rdQd = rd;
    } else if (qd && qd !== "—") {
      rdQd = qd;
    }
    $r.find("[data-tt-rd-qd-name]").text(rdQd);
    $r.find("[data-tt-handler-name]").text(data.handlerName || "—");
  }

  function loadContext(item) {
    var id = demandNumericId(item);
    var $r = $("#poTesttaskModal #poTesttaskFormRoot");
    if (!id) {
      showToast("需求 ID 无效", "error");
      return;
    }
    setContextLoading($r);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(id) + "/testtask", {
      method: "GET",
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest"
      }
    })
      .then(function (res) {
        return res.text().then(function (text) {
          var data = {};
          try {
            data = text ? JSON.parse(text) : {};
          } catch (ignore) {
            data = {};
          }
          return { ok: res.ok, data: data, status: res.status };
        });
      })
      .then(function (wrap) {
        if (wrap.ok && wrap.data && wrap.data.success && wrap.data.data) {
          fillContextData(wrap.data.data);
          return;
        }
        var msg = String((wrap.data && (wrap.data.message || wrap.data.error)) || "").trim();
        showToast(msg || "获取提测上下文失败", "error");
      })
      .catch(function () {
        showToast("获取提测上下文失败，请稍后重试", "error");
      });
  }

  function closeModal() {
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function openModal(item) {
    fillDemandHeader(item || null);
    switchPanel(1);
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
    loadContext(item || null);
  }

  function bindWizard($scope) {
    $scope.on("click", "#poTesttaskNextBtn", function () {
      if (currentStep < 4) {
        if (currentStep === 1) {
          isJointTest = readJointFlag($root());
        }
        switchPanel(currentStep + 1);
      }
    });
    $scope.on("click", "#poTesttaskPrevBtn", function () {
      if (currentStep > 1) {
        switchPanel(currentStep - 1);
      }
    });
    $scope.on("change", "[data-tt-sys]", function () {
      var $cb = $(this);
      if ($cb.is("[data-tt-sys-main]") && !$cb.prop("checked")) {
        $cb.prop("checked", true);
      }
      syncSysUnits($root());
    });
    $scope.on("change", 'input[name="ver1Mode"]', function () {
      setVersionMode(1, $(this).val() === "new");
    });
    $scope.on("change", 'input[name="ver2Mode"]', function () {
      setVersionMode(2, $(this).val() === "new");
    });
    $scope.on("click", "#poTesttaskSubmitBtn", function () {
      showToast("提交提测（静态演示，未提交后端）", "success");
    });
    $scope.on("click", ".po-testtask-link-add", function () {
      showToast("添加已有需求（静态演示）", "info");
    });
  }

  function bindModalChrome() {
    $("#poTesttaskCloseBtn, #poTesttaskOverlay").on("click", function () {
      closeModal();
    });
  }

  function init() {
    if (bound) {
      return;
    }
    bound = true;
    bindModalChrome();
    bindWizard($(document));
    initTesttaskAutocompletes();
    initTesttaskMultiselects($("#poTesttaskFormRoot").first());
    if ($("#poTesttaskModal #poTesttaskFormRoot").length) {
      initTesttaskMultiselects($("#poTesttaskModal #poTesttaskFormRoot"));
    }
    if ($(".po-testtask-page #poTesttaskFormRoot").length) {
      switchPanel(1);
    }
  }

  window.openPoSubmitTestModal = openModal;

  $(init);
})(jQuery);
