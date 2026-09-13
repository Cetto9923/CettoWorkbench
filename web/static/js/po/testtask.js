/*
 * 文件: web/static/js/po/testtask.js
 * 模块: PO工作台
 * 职责: 提测办理四步向导 UI 与上下文加载（新版本提交见 testtask-builds.js）。
 * 边界: 本阶段只同步版本（POST /demands/:id/testtask/builds）；不创建 zt_testtask。
 * 协议: appFetch 自动附带 X-CSRF-Token / X-Requested-With。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poTesttaskModal", "poTesttaskOverlay"];
  var currentStep = 1;
  var isJointTest = 0;
  var bound = false;
  // 当前需求上下文：id / systems 由 loadContext 写入；执行加载与提交复用。
  var currentDemandId = "";
  var currentSystems = [];

  function showToast(message, level) { window.showToast(message, level || "info"); }

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

  function updateSystemEnableState() {
    var $r = $root();
    $r.find("[data-tt-unit]").each(function () {
      var unitNo = $(this).attr("data-tt-unit");
      var isMain = $(this).find(".po-testtask-sys-badge.is-main").length > 0;
      var $check = $r.find('input[name="sys' + unitNo + 'Enable"]');
      if (isMain) {
        $check.prop("checked", true).prop("disabled", true);
      }
      var enabled = isMain || $check.is(":checked");
      $(this).find(".po-testtask-unit-body").toggleClass("po-testtask-hidden", !enabled);
      $(this).find("[data-tt-unit-skipped]").toggleClass("po-testtask-hidden", enabled);
      $r.find('[data-tt-link-unit="' + unitNo + '"]').toggleClass("po-testtask-hidden", !enabled);
      $r.find('[data-tt-task-unit="' + unitNo + '"]').toggleClass("po-testtask-hidden", !enabled);
    });
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

    if (step === 1) {
      isJointTest = readJointFlag($r);
    }
    if (step === 2) {
      if (typeof window.PoTesttaskBuilds_onEnterStep2 === "function") {
        window.PoTesttaskBuilds_onEnterStep2();
      }
      updateSystemEnableState();
    }
    if (step === 3 || step === 4) {
      updateSystemEnableState();
      if (step === 4) {
        isJointTest = readJointFlag($r);
        var joint = isJointTest === 1;
        $r.find("[data-tt-joint-single]").toggleClass("po-testtask-hidden", joint);
        $r.find("[data-tt-joint-global]").toggleClass("po-testtask-hidden", !joint);
      }
    }
  }

  function setVersionMode(unit, isNew) {
    var $r = $root();
    $r.find('[data-tt-new-base="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-exist-base="' + unit + '"]').toggleClass("po-testtask-hidden", isNew);
    $r.find('[data-tt-integrate-wrap="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-link-edit="' + unit + '"]').removeClass("po-testtask-hidden");
    $r.find('[data-tt-link-readonly="' + unit + '"]').toggleClass("po-testtask-hidden", isNew);
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

    // 写入当前上下文（执行拉取与提交时使用）。
    currentDemandId = demandId;
    currentSystems = Array.isArray(data.systems) ? data.systems.slice() : [];
    window.PoTesttaskRender(data);
    updateSystemEnableState();
    if (typeof window.PoTesttaskBuilds_onContextLoaded === "function") {
      window.PoTesttaskBuilds_onContextLoaded($r);
    }
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
        if (String(id) !== String(currentDemandId)) return;
        if (wrap.ok && wrap.data && wrap.data.success && wrap.data.data) {
          fillContextData(wrap.data.data);
          return;
        }
        var msg = String((wrap.data && (wrap.data.message || wrap.data.error)) || "").trim();
        if (!msg && wrap.status === 401) {
          msg = "未登录或登录已过期，请重新登录";
        } else if (!msg && wrap.status === 403) {
          msg = "您暂无权限办理该需求的提测";
        } else if (!msg && wrap.status === 404) {
          msg = "未找到该需求的提测信息";
        }
        showToast(msg || "获取提测上下文失败", "error");
      })
      .catch(function (err) {
        var msg = (err && err.message) ? ("获取提测上下文失败: " + err.message) : "获取提测上下文失败，请稍后重试";
        showToast(msg, "error");
      });
  }

  function closeModal() {
    if (window.PoTesttaskSubmit && window.PoTesttaskSubmit.busy()) return;
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function openModal(item) {
    if (window.PoTesttaskSubmit && window.PoTesttaskSubmit.busy()) return;
    fillDemandHeader(item || null);
    currentDemandId = demandNumericId(item || null);
    currentSystems = [];
    var $r = $root();
    $r.find('input[name="isJointTest"][value="0"]').prop("checked", true);
    $r.find("#poTesttaskUnitsWrap").html('<div class="po-testtask-loading" style="padding:24px;text-align:center;color:var(--t3);font-size:12px;"><i class="fas fa-spinner fa-spin"></i> 正在加载涉及系统与研发需求…</div>');
    $r.find("#poTesttaskLinksWrap").html('<div class="po-testtask-loading" style="padding:24px;text-align:center;color:var(--t3);font-size:12px;"><i class="fas fa-spinner fa-spin"></i> 正在加载各系统实际转出的研发需求…</div>');
    $r.find("#poTesttaskTasksWrap").empty();
    if (typeof window.PoTesttaskBuilds_reset === "function") {
      window.PoTesttaskBuilds_reset();
    }
    switchPanel(1);
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
    loadContext(item || null);
  }

  function bindWizard($scope) {
    $scope.on("change", 'input[name="isJointTest"]', function () {
      isJointTest = readJointFlag($root());
      var $r = $root();
      if (isJointTest === 0) {
        $r.find('input[name="sys2Enable"]').prop("checked", false);
      } else {
        $r.find('input[name="sys2Enable"]').prop("checked", true);
      }
      updateSystemEnableState();
    });
    $scope.on("change", 'input[name$="Enable"]', function () {
      updateSystemEnableState();
    });
    $scope.on("click", "#poTesttaskNextBtn", function () {
      if (currentStep < 4) {
        if (currentStep === 1) {
          isJointTest = readJointFlag($root());
        }
        var step = currentStep;
        window.PoTesttaskSubmit.beforeNext(step).then(function (ok) {
          if (ok) switchPanel(step + 1);
        });
      }
    });
    $scope.on("click", "#poTesttaskPrevBtn", function () {
      if (currentStep > 1) {
        switchPanel(currentStep - 1);
      }
    });
    $scope.on("change", 'input[name$="Mode"]', function () {
      var name = $(this).attr("name") || "";
      var m = name.match(/^ver(\d+)Mode$/);
      if (m) {
        setVersionMode(m[1], $(this).val() === "new");
      }
    });
    $scope.on("change", "[data-tt-exist-ver]", function () {
      var unit = $(this).attr("data-tt-exist-ver");
      if (!$(this).val()) {
        return;
      }
      $scope.find('[data-tt-exist-date="' + unit + '"]').val("2025-09-01");
      $scope.find('[data-tt-exist-desc="' + unit + '"]').val("复用已有版本");
    });
    $scope.on("click", ".po-testtask-link-add", function () {
      showToast("添加已有需求功能尚未开放", "info");
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
    if ($(".po-testtask-page #poTesttaskFormRoot").length) {
      switchPanel(1);
    }
  }

  // 暴露给 testtask-builds.js 的上下文接口
  window.PoTesttaskCore = {
    getDemandId: function () { return currentDemandId; },
    getSystems: function () { return currentSystems.slice(); },
    showToast: showToast,
    $root: $root,
    formatDemandId: formatDemandId
  };

  window.openPoSubmitTestModal = openModal;

  $(init);
})(jQuery);
