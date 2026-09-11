/*
 * 文件: web/static/js/po/testtask.js
 * 模块: PO工作台
 * 职责: 提测办理四步向导 UI 与上下文加载（不直接发新版本提交请求；该部分见 testtask-builds.js）。
 * 边界: 本阶段只同步版本（POST /demands/:id/testtask/builds），由 testtask-builds.js 处理；
 *       测试单 zt_testtask 不创建，留到 Phase D。
 * 协议: appFetch 自动附带 X-CSRF-Token / X-Requested-With，与 LinkStory 一致。
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

  function esc(v) {
    return String(v == null ? "" : v)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function renderDynamicForm(data) {
    var $r = $root();
    var demandId = data.demandId != null ? String(data.demandId) : "";
    var launchDate = $.trim(data.estimateLaunch || "");
    if (launchDate.indexOf("0001") === 0 || launchDate.indexOf("0000") === 0 || launchDate === "—") {
      launchDate = "";
    }
    var systems = Array.isArray(data.systems) ? data.systems : [];
    if (!systems.length) {
      if (data.mainSystemName && data.mainSystemName !== "—") {
        systems = [{ id: 0, name: data.mainSystemName, isMain: true, stories: [] }];
      }
    }

    $r.find("#poTesttaskSysMeta").text("共 " + systems.length + " 个系统");

    var todayStr = new Date().toISOString().slice(0, 10).replace(/-/g, "");
    var qdName = $.trim(data.qdName || data.qd || "");
    if (qdName === "—") qdName = "";

    // 1. 动态生成步骤 2：按系统配置版本基础信息
    var unitsHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var badgeHtml = isMain ? '<span class="po-testtask-sys-badge is-main">主</span>' : '<span class="po-testtask-sys-badge is-sub">配</span>';
      var toggleHtml = isMain
        ? '<label class="po-testtask-sys-toggle"><input type="checkbox" name="sys' + unitNo + 'Enable" value="1" checked disabled /> <span>主系统（默认提测）</span></label>'
        : '<label class="po-testtask-sys-toggle"><input type="checkbox" name="sys' + unitNo + 'Enable" value="1" ' + (isJointTest === 1 ? 'checked' : '') + ' /> <span>参与本次提测</span></label>';

      var defVerName = todayStr + "-US" + demandId + "-" + sysName + "-提测版本";

      unitsHtml +=
        '<div class="po-testtask-unit" data-tt-unit="' + unitNo + '" data-tt-unit-product="' + (sys.id || "") + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-head">' +
        '    <div class="po-testtask-unit-name"><i class="fas fa-folder-open"></i> <span>' + sysName + '</span> ' + badgeHtml + '</div>' +
        '    ' + toggleHtml +
        '  </div>' +
        '  <div class="po-testtask-unit-skipped po-testtask-hidden" data-tt-unit-skipped="' + unitNo + '">' +
        '    <i class="fas fa-circle-info"></i> 已跳过该系统提测，不创建版本与测试单，不显示该系统需求' +
        '  </div>' +
        '  <div class="po-testtask-unit-body">' +
        '    <div class="po-testtask-field-label"><span class="req">*</span> 版本模式</div>' +
        '    <div class="po-testtask-radio-row">' +
        '      <label><input type="radio" name="ver' + unitNo + 'Mode" value="new" checked /> 创建新版本</label>' +
        '      <label><input type="radio" name="ver' + unitNo + 'Mode" value="exist" /> 使用已有版本</label>' +
        '    </div>' +
        (isMain ? '' : '    <div class="po-testtask-check-row" data-tt-integrate-wrap="' + unitNo + '"><label><input type="checkbox" name="ver' + unitNo + 'IsIntegrate" value="1" /> 集成版本</label></div>') +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 所属执行</div>' +
        '      <select data-tt-exec="' + unitNo + '"><option value="">加载中…</option></select>' +
        '    </div>' +
        '    <div data-tt-new-base="' + unitNo + '">' +
        '      <div class="po-testtask-grid-2">' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 新版本名称</div>' +
        '          <input type="text" data-tt-ver-name="' + unitNo + '" value="' + esc(defVerName) + '" />' +
        '        </div>' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 计划上线日期</div>' +
        '          <input type="date" data-tt-ver-date="' + unitNo + '" value="' + esc(launchDate) + '" />' +
        '        </div>' +
        '      </div>' +
        '      <div class="po-testtask-field">' +
        '        <div class="po-testtask-field-label">版本说明 (可选)</div>' +
        '        <input type="text" data-tt-ver-desc="' + unitNo + '" placeholder="说明此版本主要内容（可选）" />' +
        '      </div>' +
        '    </div>' +
        '    <div class="po-testtask-hidden" data-tt-exist-base="' + unitNo + '">' +
        '      <div class="po-testtask-grid-2">' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 已有版本名称</div>' +
        '          <select data-tt-exist-ver="' + unitNo + '"><option value="">请先选择所属执行，再选取版本</option></select>' +
        '        </div>' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label">计划上线日期</div>' +
        '          <input type="date" class="is-readonly" data-tt-exist-date="' + unitNo + '" readonly />' +
        '        </div>' +
        '      </div>' +
        '      <div class="po-testtask-field">' +
        '        <div class="po-testtask-field-label">版本说明 (可选)</div>' +
        '        <input type="text" class="is-readonly" data-tt-exist-desc="' + unitNo + '" readonly />' +
        '      </div>' +
        '    </div>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskUnitsWrap").html(unitsHtml);

    // 2. 动态生成步骤 3：各系统版本关联需求配置
    var linksHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var badgeHtml = isMain ? '<span class="po-testtask-sys-badge is-main">主</span>' : '<span class="po-testtask-sys-badge is-sub">配</span>';
      var stories = Array.isArray(sys.stories) ? sys.stories : [];

      var storiesListHtml = "";
      if (stories.length > 0) {
        stories.forEach(function (st) {
          storiesListHtml += '<label><input type="checkbox" checked value="' + esc(st.id) + '" /> #' + esc(st.id) + ' ' + esc(st.title) + '</label>';
        });
      } else {
        storiesListHtml = '<div class="po-testtask-empty-hint" style="color:var(--t3);font-size:12px;padding:6px 0">该系统下暂未关联转出的研发需求</div>';
      }

      linksHtml +=
        '<div class="po-testtask-unit" data-tt-link-unit="' + unitNo + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-name" style="margin-bottom:8px">' +
        '    <i class="fas fa-folder-open"></i> <span>' + sysName + '</span> ' + badgeHtml +
        '  </div>' +
        '  <div class="po-testtask-unit-body">' +
        '    <div data-tt-link-edit="' + unitNo + '">' +
        '      <div class="po-testtask-link-actions">' +
        '        <div class="po-testtask-field-label">版本关联研发需求（已自动获取该系统实际转出的研发需求）</div>' +
        '      </div>' +
        '      <div class="po-testtask-link-list">' + storiesListHtml + '</div>' +
        '    </div>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskLinksWrap").html(linksHtml);

    // 3. 动态生成步骤 4：各系统独立测试单
    var tasksHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var defTaskName = todayStr + "-US" + demandId + "-" + sysName + "-测试单";

      tasksHtml +=
        '<div class="po-testtask-unit" data-tt-task-unit="' + unitNo + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-title">' + sysName + ' - 测试单</div>' +
        '  <div class="po-testtask-grid-4">' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 测试单名称</div>' +
        '      <input type="text" data-tt-task-name="' + unitNo + '" value="' + esc(defTaskName) + '" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 测试负责人</div>' +
        '      <input type="text" data-tt-task-owner="' + unitNo + '" value="' + esc(qdName) + '" placeholder="输入姓名或工号搜索" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label">开始日期</div>' +
        '      <input type="date" data-tt-task-begin="' + unitNo + '" value="' + todayStr.slice(0,4)+"-"+todayStr.slice(4,6)+"-"+todayStr.slice(6,8) + '" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label">结束日期</div>' +
        '      <input type="date" data-tt-task-end="' + unitNo + '" value="' + esc(launchDate) + '" />' +
        '    </div>' +
        '  </div>' +
        '  <div class="po-testtask-field">' +
        '    <div class="po-testtask-field-label">测试说明 / 重点 (可选)</div>' +
        '    <textarea data-tt-task-desc="' + unitNo + '" placeholder="说明本次提测重点（可选）"></textarea>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskTasksWrap").html(tasksHtml);

    updateSystemEnableState();
  }

  function setVersionMode(unit, isNew) {
    var $r = $root();
    $r.find('[data-tt-new-base="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-exist-base="' + unit + '"]').toggleClass("po-testtask-hidden", isNew);
    $r.find('[data-tt-integrate-wrap="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
    $r.find('[data-tt-link-edit="' + unit + '"]').toggleClass("po-testtask-hidden", !isNew);
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
    renderDynamicForm(data);
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
    if (typeof window.closeShowModals === "function") {
      window.closeShowModals(MODAL_IDS);
    }
  }

  function openModal(item) {
    fillDemandHeader(item || null);
    currentDemandId = demandNumericId(item || null);
    currentSystems = [];
    if (typeof window.PoTesttaskBuilds_reset === "function") {
      window.PoTesttaskBuilds_reset();
    }
    switchPanel(1);
    var $r = $root();
    $r.find('input[name="isJointTest"][value="0"]').prop("checked", true);
    $r.find('input[name="sys2Enable"]').prop("checked", false);
    updateSystemEnableState();
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
        switchPanel(currentStep + 1);
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
      showToast("添加已有需求（Phase D 接入）", "info");
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
