/*
 * 文件: web/static/js/po/testtask.js
 * 模块: PO工作台
 * 职责: 提测办理四步向导；按需求涉及产品动态渲染系统与配置单元。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poTesttaskModal", "poTesttaskOverlay"];
  var currentStep = 1;
  var isJointTest = 0;
  var bound = false;
  var currentSystems = [];
  var contextMeta = {
    demandId: "",
    title: "",
    estimateLaunch: "",
    qdName: ""
  };

  var EXEC_OPTIONS = [
    { value: "P01/E-1001", label: "信贷项目/执行-1001" },
    { value: "P01/E-1002", label: "信贷项目/执行-1002" },
    { value: "P02/E-2001", label: "打印中心项目/执行-2001" }
  ];
  var EXIST_VER_OPTIONS = [
    { value: "v1", label: "20250801-已有版本-A" },
    { value: "v2", label: "20250815-已有版本-B" },
    { value: "v3", label: "20250901-已有版本-C" }
  ];

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
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

  function todayYMD() {
    var d = new Date();
    var m = String(d.getMonth() + 1).padStart(2, "0");
    var day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }

  function compactToday() {
    return todayYMD().replace(/-/g, "");
  }

  function defaultVersionName() {
    var id = contextMeta.demandId || "";
    return compactToday() + (id ? "-US" + id : "") + "-提测版本";
  }

  function defaultTesttaskName(joint) {
    var id = contextMeta.demandId || "";
    var suffix = joint ? "-联调总测试单" : "-测试单";
    return compactToday() + (id ? "-US" + id : "") + suffix;
  }

  function unitIds() {
    return currentSystems.map(function (s) {
      return String(s.id);
    });
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
    if (step === 3 || step === 4) {
      syncSysUnits($r);
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
    unitIds().forEach(function (unit) {
      var $newRadio = $r.find('input[name="ver' + unit + 'Mode"][value="new"]');
      var $newLabel = $newRadio.closest("label");
      var wasHidden = $newLabel.hasClass("po-testtask-hidden");
      $newLabel.toggleClass("po-testtask-hidden", joint);
      if (joint) {
        $r.find('input[name="ver' + unit + 'Mode"][value="exist"]').prop("checked", true);
        setVersionMode(unit, false);
      } else if (wasHidden) {
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
    unitIds().forEach(function (unit) {
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
      $r.find('[data-tt-link-unit="' + unit + '"]').toggleClass("po-testtask-hidden", !on);
      $r.find('[data-tt-test-unit="' + unit + '"]').toggleClass("po-testtask-hidden", !on);
    });
  }

  function initUnitAutocompletes($r) {
    if (typeof window.initAutocomplete !== "function" || !$r || !$r.length) {
      return;
    }
    unitIds().forEach(function (unit) {
      var execInput = "poTtExecInput_" + unit;
      var execHidden = "poTtExecValue_" + unit;
      var verInput = "poTtExistVerInput_" + unit;
      var verHidden = "poTtExistVerValue_" + unit;
      if ($r.find("#" + execInput).length && $r.find("#" + execHidden).length) {
        window.initAutocomplete(execInput, execHidden, EXEC_OPTIONS, {
          value: "",
          label: "",
          placeholder: "搜索执行"
        });
      }
      if ($r.find("#" + verInput).length && $r.find("#" + verHidden).length) {
        window.initAutocomplete(verInput, verHidden, EXIST_VER_OPTIONS, {
          value: "",
          label: "",
          placeholder: "搜索已有版本"
        });
      }
    });
  }

  function existVerMultiHTML(unit) {
    var name = "existVer" + unit;
    var opts = EXIST_VER_OPTIONS.map(function (o) {
      return (
        '<label class="checkbox-label form-multiselect-option">' +
        '<input name="' +
        esc(name) +
        '" value="' +
        esc(o.value) +
        '" type="checkbox" class="checkbox form-multiselect-checkbox">' +
        '<span class="checkbox-box"></span>' +
        '<span class="form-multiselect-name">' +
        esc(o.label) +
        "</span>" +
        "</label>"
      );
    }).join("");
    return (
      '<div class="dropdown form-multiselect" data-role-dropdown data-multiselect-keep-placeholder="true">' +
      '<div class="form-input-clear-wrap">' +
      '<div class="form-multiselect-display" data-role-display>' +
      '<div class="form-multiselect-tags" data-role-tags></div>' +
      '<input type="text" class="input form-multiselect-input" data-multiselect-filter' +
      ' placeholder="搜索并选择已有版本（可多选）"' +
      ' data-placeholder-empty="搜索并选择已有版本（可多选）"' +
      ' autocomplete="off" aria-autocomplete="list" aria-label="检索已有版本" />' +
      "</div>" +
      '<button type="button" class="form-dropdown-chevron" aria-label="展开已有版本" onclick="return toggleDropdown(this);">' +
      '<i class="bi bi-chevron-down" aria-hidden="true"></i></button>' +
      '<button type="button" class="form-dropdown-clear" data-form-multiselect-clear aria-label="清除已有版本" hidden>×</button>' +
      "</div>" +
      '<div class="dropdown-menu form-multiselect-menu"><div class="form-multiselect-scroll">' +
      '<div class="form-multiselect-list">' +
      opts +
      "</div></div></div></div>"
    );
  }

  function versionUnitHTML(sys) {
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var badgeClass = isMain ? "is-main" : "is-sub";
    var badgeText = isMain ? "主" : "配";
    var hiddenClass = isMain ? "" : " po-testtask-hidden";
    var verName = defaultVersionName();
    var launch = contextMeta.estimateLaunch && contextMeta.estimateLaunch !== "—" ? contextMeta.estimateLaunch : todayYMD();
    return (
      '<div class="po-testtask-unit' +
      hiddenClass +
      '" data-tt-unit="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-unit-head"><div class="po-testtask-unit-name">' +
      '<i class="fas fa-folder-open"></i><span>' +
      esc(name) +
      '</span><span class="po-testtask-sys-badge ' +
      badgeClass +
      '">' +
      badgeText +
      "</span></div></div>" +
      '<div class="po-testtask-unit-body">' +
      '<div class="po-testtask-field-label"><span class="req">*</span> 版本模式</div>' +
      '<div class="po-testtask-radio-row">' +
      '<label><input type="radio" name="ver' +
      esc(id) +
      'Mode" value="new" data-tt-ver-mode="' +
      esc(id) +
      '" checked /> 创建新版本</label>' +
      '<label><input type="radio" name="ver' +
      esc(id) +
      'Mode" value="exist" data-tt-ver-mode="' +
      esc(id) +
      '" /> 使用已有版本</label>' +
      "</div>" +
      '<div data-tt-new-base="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 所属执行</div>' +
      '<div class="ui-autocomplete">' +
      '<input type="text" id="poTtExecInput_' +
      esc(id) +
      '" class="po-testtask-ac-input" placeholder="搜索执行" autocomplete="off" />' +
      '<input type="hidden" id="poTtExecValue_' +
      esc(id) +
      '" data-tt-exec="' +
      esc(id) +
      '" value="" />' +
      "</div></div>" +
      '<div class="po-testtask-grid-2">' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 新版本名称</div>' +
      '<input type="text" value="' +
      esc(verName) +
      '" /></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 计划上线日期</div>' +
      '<input type="date" value="' +
      esc(launch) +
      '" /></div></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label">版本说明 (可选)</div><input type="text" /></div>' +
      "</div>" +
      '<div class="po-testtask-hidden" data-tt-exist-base="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 已有版本名称</div>' +
      '<div data-tt-exist-single="' +
      esc(id) +
      '"><div class="ui-autocomplete">' +
      '<input type="text" id="poTtExistVerInput_' +
      esc(id) +
      '" class="po-testtask-ac-input" placeholder="搜索已有版本" autocomplete="off" />' +
      '<input type="hidden" id="poTtExistVerValue_' +
      esc(id) +
      '" data-tt-exist-ver="' +
      esc(id) +
      '" value="" />' +
      "</div></div>" +
      '<div class="po-testtask-hidden" data-tt-exist-multi="' +
      esc(id) +
      '">' +
      existVerMultiHTML(id) +
      "</div></div></div>" +
      "</div></div>"
    );
  }

  function linkUnitHTML(sys) {
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var badgeClass = isMain ? "is-main" : "is-sub";
    var badgeText = isMain ? "主" : "配";
    var hiddenClass = isMain ? "" : " po-testtask-hidden";
    var demandLabel = "";
    if (contextMeta.demandId) {
      demandLabel =
        "#" +
        esc(contextMeta.demandId) +
        " " +
        esc(contextMeta.title || "") +
        "(当前需求)";
    }
    return (
      '<div class="po-testtask-unit' +
      hiddenClass +
      '" data-tt-link-unit="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-unit-name" style="margin-bottom:8px">' +
      '<i class="fas fa-folder-open"></i><span>' +
      esc(name) +
      '</span><span class="po-testtask-sys-badge ' +
      badgeClass +
      '">' +
      badgeText +
      "</span></div>" +
      '<div class="po-testtask-unit-body">' +
      '<div data-tt-link-edit="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-link-actions">' +
      '<div class="po-testtask-field-label">版本关联需求（可多选）</div>' +
      '<button type="button" class="po-testtask-link-add">+ 添加已有需求</button></div>' +
      '<div class="po-testtask-link-list">' +
      (demandLabel
        ? '<label><input type="checkbox" checked /> ' + demandLabel + "</label>"
        : "") +
      "</div></div>" +
      '<div class="po-testtask-hidden" data-tt-link-readonly="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-field-label">版本已关联需求（只读，取自已有版本）</div>' +
      '<div class="po-testtask-link-readonly">' +
      (demandLabel ? "<div>" + demandLabel.replace("(当前需求)", "") + "</div>" : "") +
      "</div></div>" +
      "</div></div>"
    );
  }

  function testUnitHTML(sys) {
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var hiddenClass = isMain ? "" : " po-testtask-hidden";
    var ttName = defaultTesttaskName(false);
    var qd = contextMeta.qdName && contextMeta.qdName !== "—" ? contextMeta.qdName : "";
    var start = todayYMD();
    var end = contextMeta.estimateLaunch && contextMeta.estimateLaunch !== "—" ? contextMeta.estimateLaunch : "";
    return (
      '<div class="po-testtask-unit' +
      hiddenClass +
      '" data-tt-test-unit="' +
      esc(id) +
      '">' +
      '<div class="po-testtask-unit-title">' +
      esc(name) +
      "-测试单</div>" +
      '<div class="po-testtask-grid-4">' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 测试单名称</div>' +
      '<input type="text" value="' +
      esc(ttName) +
      '" /></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label"><span class="req">*</span> 测试负责人</div>' +
      '<input type="text" value="' +
      esc(qd) +
      '" /></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label">开始日期</div>' +
      '<input type="date" value="' +
      esc(start) +
      '" /></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label">结束日期</div>' +
      '<input type="date" value="' +
      esc(end) +
      '" /></div></div>' +
      '<div class="po-testtask-field"><div class="po-testtask-field-label">测试说明 / 重点 (可选)</div><textarea></textarea></div>' +
      "</div>"
    );
  }

  function renderSystems($r, systems) {
    $r = $r && $r.length ? $r : $root();
    if (!$r.length) {
      return;
    }
    currentSystems = Array.isArray(systems) ? systems.slice() : [];
    var $list = $r.find("[data-tt-sys-list]");
    var $units = $r.find("[data-tt-sys-units]");
    var $links = $r.find("[data-tt-link-units]");
    var $tests = $r.find("[data-tt-test-units]");

    if (!currentSystems.length) {
      $list.html('<span class="po-testtask-sys-empty">当前需求暂无涉及产品</span>');
      $units.empty();
      $links.empty();
      $tests.empty();
      return;
    }

    var checks = currentSystems
      .map(function (sys) {
        var id = String(sys.id);
        var name = String(sys.name || "");
        var isMain = !!sys.isMain;
        var badgeClass = isMain ? "is-main" : "is-sub";
        var badgeText = isMain ? "主" : "配";
        var attrs = 'type="checkbox" data-tt-sys="' + esc(id) + '"';
        if (isMain) {
          attrs += " data-tt-sys-main checked disabled";
        }
        return (
          "<label><input " +
          attrs +
          " /> " +
          esc(name) +
          ' <span class="po-testtask-sys-badge ' +
          badgeClass +
          '">' +
          badgeText +
          "</span></label>"
        );
      })
      .join("");

    $list.html(checks);
    $units.html(currentSystems.map(versionUnitHTML).join(""));
    $links.html(currentSystems.map(linkUnitHTML).join(""));
    $tests.html(currentSystems.map(testUnitHTML).join(""));

    $r.find("[data-tt-joint-name]").val(defaultTesttaskName(true));
    $r.find("[data-tt-joint-qd]").val(
      contextMeta.qdName && contextMeta.qdName !== "—" ? contextMeta.qdName : ""
    );
    $r.find("[data-tt-joint-start]").val(todayYMD());
    $r.find("[data-tt-joint-end]").val(
      contextMeta.estimateLaunch && contextMeta.estimateLaunch !== "—"
        ? contextMeta.estimateLaunch
        : ""
    );

    syncSysUnits($r);
    syncVersionModeOptions($r);
    initUnitAutocompletes($r);
    initTesttaskMultiselects($r);
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
    $r.find(
      "[data-tt-raw-status],[data-tt-main-system],[data-tt-estimate-launch],[data-tt-bra-name],[data-tt-rd-qd-name],[data-tt-handler-name]"
    ).text("加载中…");
    $r.find("[data-tt-sys-list]").html('<span class="po-testtask-sys-empty">加载中…</span>');
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

    contextMeta.demandId = demandId;
    contextMeta.title = title;
    contextMeta.estimateLaunch = String(data.estimateLaunch || "").trim();
    contextMeta.qdName = qd;

    renderSystems($r, data.systems || []);
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
    $scope.on("change", "input[data-tt-ver-mode]", function () {
      var unit = $(this).attr("data-tt-ver-mode");
      setVersionMode(unit, $(this).val() === "new");
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
    if ($(".po-testtask-page #poTesttaskFormRoot").length) {
      switchPanel(1);
    }
  }

  window.openPoSubmitTestModal = openModal;

  $(init);
})(jQuery);
