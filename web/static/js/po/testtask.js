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
  var createdBuildsByProduct = {};
  var contextMeta = {
    demandId: "",
    title: "",
    estimateLaunch: "",
    qdName: ""
  };

  var EXIST_VER_OPTIONS = [
    { value: "v1", label: "20250801-已有版本-A" },
    { value: "v2", label: "20250815-已有版本-B" },
    { value: "v3", label: "20250901-已有版本-C" }
  ];
  var execOptionsCache = {};

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

  function todayYMD() {
    var d = new Date();
    var m = String(d.getMonth() + 1).padStart(2, "0");
    var day = String(d.getDate()).padStart(2, "0");
    return d.getFullYear() + "-" + m + "-" + day;
  }

  function compactToday() {
    return todayYMD().replace(/-/g, "");
  }

  function defaultVersionName(productName) {
    var name = String(productName || "").trim();
    var ymd = compactToday();
    return name ? name + " - " + ymd : ymd;
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

  function fetchJSON(url) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(url, {
      method: "GET",
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest"
      }
    }).then(function (res) {
      return res.text().then(function (text) {
        var data = {};
        try {
          data = text ? JSON.parse(text) : {};
        } catch (ignore) {
          data = {};
        }
        return { ok: res.ok, data: data, status: res.status };
      });
    });
  }

  function postJSON(url, payload) {
    var fetchFn = window.appFetch || fetch;
    return fetchFn(url, {
      method: "POST",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        "X-Requested-With": "XMLHttpRequest"
      },
      body: JSON.stringify(payload || {})
    }).then(function (res) {
      return res.text().then(function (text) {
        var data = {};
        try {
          data = text ? JSON.parse(text) : {};
        } catch (ignore) {
          data = {};
        }
        return { ok: res.ok, data: data, status: res.status };
      });
    });
  }

  function checkedSystemIds($r) {
    var ids = [];
    $r.find("[data-tt-sys]:checked").each(function () {
      var id = String($(this).attr("data-tt-sys") || "").trim();
      if (id) {
        ids.push(id);
      }
    });
    return ids;
  }

  function parseExecValue(raw) {
    var s = String(raw || "").trim();
    var parts = s.split("-");
    if (parts.length < 2) {
      return null;
    }
    var projectId = parseInt(parts[0], 10);
    var executionId = parseInt(parts[1], 10);
    if (!projectId || !executionId) {
      return null;
    }
    return { projectId: projectId, executionId: executionId };
  }

  function collectExistVerIds($r, unit, joint) {
    if (joint) {
      var multi = [];
      $r.find('input[name="existVer' + unit + '"]:checked').each(function () {
        var v = String($(this).val() || "").trim();
        if (v) {
          multi.push(v);
        }
      });
      return multi;
    }
    var single = String($r.find('[data-tt-exist-ver="' + unit + '"]').val() || "").trim();
    return single ? [single] : [];
  }

  // 校验第 2 步；返回 { ok, builds }，builds 仅为「创建新版本」项。
  function validateAndCollectBuilds($r) {
    var joint = readJointFlag($r) === 1;
    var ids = checkedSystemIds($r);
    if (!ids.length) {
      showToast("请至少选择一个系统", "error");
      return { ok: false, builds: [] };
    }
    var builds = [];
    for (var i = 0; i < ids.length; i++) {
      var unit = ids[i];
      var mode = $r.find('input[name="ver' + unit + 'Mode"]:checked').val();
      if (mode === "exist") {
        var existIds = collectExistVerIds($r, unit, joint);
        if (!existIds.length) {
          showToast("请选择已有版本", "error");
          return { ok: false, builds: [] };
        }
        continue;
      }
      var execRaw = $r.find('[data-tt-exec="' + unit + '"]').val();
      var exec = parseExecValue(execRaw);
      if (!exec) {
        showToast("请选择所属执行", "error");
        return { ok: false, builds: [] };
      }
      var name = String($r.find('[data-tt-unit="' + unit + '"] [data-tt-fill="ver-name"]').val() || "").trim();
      if (!name) {
        showToast("请填写新版本名称", "error");
        return { ok: false, builds: [] };
      }
      var date = String($r.find('[data-tt-unit="' + unit + '"] [data-tt-fill="launch"]').val() || "").trim();
      if (!date) {
        showToast("请填写计划上线日期", "error");
        return { ok: false, builds: [] };
      }
      var desc = String($r.find('[data-tt-unit="' + unit + '"] [data-tt-fill="ver-desc"]').val() || "").trim();
      builds.push({
        productId: parseInt(unit, 10),
        projectId: exec.projectId,
        executionId: exec.executionId,
        name: name,
        date: date,
        desc: desc
      });
    }
    return { ok: true, builds: builds };
  }

  function saveNewBuilds(demandId, builds) {
    return postJSON("/demands/" + encodeURIComponent(demandId) + "/testtask/builds", {
      builds: builds
    }).then(function (wrap) {
      if (wrap.ok && wrap.data && wrap.data.success) {
        var list = (wrap.data.data && wrap.data.data.builds) || [];
        createdBuildsByProduct = {};
        list.forEach(function (item) {
          if (!item) {
            return;
          }
          createdBuildsByProduct[String(item.productId)] = {
            buildId: item.buildId,
            name: item.name || ""
          };
        });
        return true;
      }
      var msg = String((wrap.data && (wrap.data.message || wrap.data.error)) || "").trim();
      if (!msg && wrap.data && Array.isArray(wrap.data.errors) && wrap.data.errors.length) {
        msg = String(wrap.data.errors[0].message || "").trim();
      }
      showToast(msg || "保存版本失败", "error");
      return false;
    }).catch(function () {
      showToast("保存版本失败，请稍后重试", "error");
      return false;
    });
  }

  function goNextFromStep2($btn) {
    var $r = $root();
    var collected = validateAndCollectBuilds($r);
    if (!collected.ok) {
      return;
    }
    if (!collected.builds.length) {
      switchPanel(3);
      return;
    }
    var demandId = contextMeta.demandId;
    if (!demandId) {
      showToast("需求 ID 无效", "error");
      return;
    }
    $btn.prop("disabled", true);
    saveNewBuilds(demandId, collected.builds).then(function (ok) {
      $btn.prop("disabled", false);
      if (ok) {
        switchPanel(3);
      }
    });
  }

  function normalizeExecOptions(list) {
    if (!Array.isArray(list)) {
      return [];
    }
    return list
      .map(function (item) {
        var value = String((item && item.value) || "").trim();
        var label = String((item && item.label) || "").trim();
        if (!value) {
          return null;
        }
        return { value: value, label: label || value };
      })
      .filter(Boolean);
  }

  function loadProductExecutions(productId) {
    var key = String(productId || "");
    if (!key) {
      return Promise.resolve([]);
    }
    if (Object.prototype.hasOwnProperty.call(execOptionsCache, key)) {
      return Promise.resolve(execOptionsCache[key]);
    }
    return fetchJSON("/products/" + encodeURIComponent(key) + "/executions").then(function (wrap) {
      var list = [];
      if (wrap.ok && wrap.data && wrap.data.success) {
        list = normalizeExecOptions(wrap.data.data);
      } else {
        var msg = String((wrap.data && (wrap.data.message || wrap.data.error)) || "").trim();
        showToast(msg || "获取所属执行失败", "error");
      }
      execOptionsCache[key] = list;
      return list;
    }).catch(function () {
      showToast("获取所属执行失败，请稍后重试", "error");
      execOptionsCache[key] = [];
      return [];
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
        window.initAutocomplete(execInput, execHidden, [], {
          value: "",
          label: "",
          placeholder: "加载执行中…"
        });
        loadProductExecutions(unit).then(function (items) {
          if (!$r.find("#" + execInput).length) {
            return;
          }
          window.initAutocomplete(execInput, execHidden, items, {
            value: "",
            label: "",
            placeholder: items.length ? "搜索执行" : "暂无可用执行"
          });
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

  function cloneTpl(id) {
    var tpl = document.getElementById(id);
    if (!tpl || !tpl.content) {
      return null;
    }
    return tpl.content.cloneNode(true);
  }

  function setBadge($el, isMain) {
    $el
      .text(isMain ? "主" : "配")
      .removeClass("is-main is-sub")
      .addClass(isMain ? "is-main" : "is-sub");
  }

  function fillExistVerOptions($list, unit) {
    var optTpl = document.getElementById("poTtExistVerOptTpl");
    if (!$list.length || !optTpl || !optTpl.content) {
      return;
    }
    var name = "existVer" + unit;
    $list.empty();
    EXIST_VER_OPTIONS.forEach(function (o) {
      var frag = optTpl.content.cloneNode(true);
      var label = frag.querySelector("label");
      if (!label) {
        return;
      }
      var input = label.querySelector('input[type="checkbox"]');
      if (input) {
        input.name = name;
        input.value = o.value;
      }
      var labelEl = label.querySelector('[data-tt-fill="label"]');
      if (labelEl) {
        labelEl.textContent = o.label;
      }
      $list.append(frag);
    });
  }

  function versionUnitNode(sys) {
    var frag = cloneTpl("poTtVersionUnitTpl");
    if (!frag) {
      return null;
    }
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var $root = $(frag.querySelector("[data-tt-unit]"));
    $root.attr("data-tt-unit", id);
    if (!isMain) {
      $root.addClass("po-testtask-hidden");
    }
    $root.find('[data-tt-fill="sys-name"]').text(name);
    setBadge($root.find('[data-tt-fill="sys-badge"]'), isMain);

    $root.find("input[data-tt-ver-mode]").each(function () {
      $(this)
        .attr("name", "ver" + id + "Mode")
        .attr("data-tt-ver-mode", id);
    });
    $root.find("[data-tt-new-base]").attr("data-tt-new-base", id);
    $root.find("[data-tt-exist-base]").attr("data-tt-exist-base", id);
    $root.find("[data-tt-exist-single]").attr("data-tt-exist-single", id);
    $root.find("[data-tt-exist-multi]").attr("data-tt-exist-multi", id);

    $root.find('[data-tt-fill-id="execInput"]').attr("id", "poTtExecInput_" + id);
    $root
      .find('[data-tt-fill-id="execValue"]')
      .attr("id", "poTtExecValue_" + id)
      .attr("data-tt-exec", id);
    $root.find('[data-tt-fill-id="existVerInput"]').attr("id", "poTtExistVerInput_" + id);
    $root
      .find('[data-tt-fill-id="existVerValue"]')
      .attr("id", "poTtExistVerValue_" + id)
      .attr("data-tt-exist-ver", id);

    $root.find('[data-tt-fill="ver-name"]').val(defaultVersionName(name));
    $root.find('[data-tt-fill="launch"]').val(todayYMD());
    fillExistVerOptions($root.find("[data-tt-exist-opt-list]"), id);
    return frag;
  }

  function linkUnitNode(sys) {
    var frag = cloneTpl("poTtLinkUnitTpl");
    if (!frag) {
      return null;
    }
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var $root = $(frag.querySelector("[data-tt-link-unit]"));
    $root.attr("data-tt-link-unit", id);
    if (!isMain) {
      $root.addClass("po-testtask-hidden");
    }
    $root.find('[data-tt-fill="sys-name"]').text(name);
    setBadge($root.find('[data-tt-fill="sys-badge"]'), isMain);
    $root.find("[data-tt-link-edit]").attr("data-tt-link-edit", id);
    $root.find("[data-tt-link-readonly]").attr("data-tt-link-readonly", id);

    var $list = $root.find('[data-tt-fill="link-list"]');
    var $readonly = $root.find('[data-tt-fill="link-readonly"]');
    $list.empty();
    $readonly.empty();
    if (contextMeta.demandId) {
      var demandText =
        "#" + contextMeta.demandId + " " + (contextMeta.title || "") + "(当前需求)";
      var readonlyText = "#" + contextMeta.demandId + " " + (contextMeta.title || "");
      $list.append(
        $("<label>").append($('<input type="checkbox" checked />'), document.createTextNode(" " + demandText))
      );
      $readonly.append($("<div>").text(readonlyText));
    }
    return frag;
  }

  function testUnitNode(sys) {
    var frag = cloneTpl("poTtTestUnitTpl");
    if (!frag) {
      return null;
    }
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var $root = $(frag.querySelector("[data-tt-test-unit]"));
    $root.attr("data-tt-test-unit", id);
    if (!isMain) {
      $root.addClass("po-testtask-hidden");
    }
    $root.find('[data-tt-fill="test-title"]').text(name + "-测试单");
    $root.find('[data-tt-fill="tt-name"]').val(defaultTesttaskName(false));
    $root
      .find('[data-tt-fill="qd"]')
      .val(contextMeta.qdName && contextMeta.qdName !== "—" ? contextMeta.qdName : "");
    $root.find('[data-tt-fill="start"]').val(todayYMD());
    $root
      .find('[data-tt-fill="end"]')
      .val(
        contextMeta.estimateLaunch && contextMeta.estimateLaunch !== "—"
          ? contextMeta.estimateLaunch
          : ""
      );
    return frag;
  }

  function sysCheckNode(sys) {
    var frag = cloneTpl("poTtSysCheckTpl");
    if (!frag) {
      return null;
    }
    var id = String(sys.id);
    var name = String(sys.name || "");
    var isMain = !!sys.isMain;
    var $label = $(frag.querySelector("label"));
    var $input = $label.find("[data-tt-sys]");
    $input.attr("data-tt-sys", id);
    if (isMain) {
      $input.attr("data-tt-sys-main", "").prop("checked", true).prop("disabled", true);
    }
    $label.find('[data-tt-fill="sys-name"]').text(name);
    setBadge($label.find('[data-tt-fill="sys-badge"]'), isMain);
    return frag;
  }

  function appendNodes($container, nodes) {
    $container.empty();
    nodes.forEach(function (node) {
      if (node) {
        $container.append(node);
      }
    });
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

    appendNodes(
      $list,
      currentSystems.map(sysCheckNode)
    );
    appendNodes(
      $units,
      currentSystems.map(versionUnitNode)
    );
    appendNodes(
      $links,
      currentSystems.map(linkUnitNode)
    );
    appendNodes(
      $tests,
      currentSystems.map(testUnitNode)
    );

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
    fetchJSON("/demands/" + encodeURIComponent(id) + "/testtask")
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
    execOptionsCache = {};
    createdBuildsByProduct = {};
    fillDemandHeader(item || null);
    switchPanel(1);
    if (typeof window.openShowModals === "function") {
      window.openShowModals(MODAL_IDS);
    }
    loadContext(item || null);
  }

  function bindWizard($scope) {
    $scope.on("click", "#poTesttaskNextBtn", function () {
      if (currentStep >= 4) {
        return;
      }
      if (currentStep === 1) {
        isJointTest = readJointFlag($root());
        switchPanel(2);
        return;
      }
      if (currentStep === 2) {
        goNextFromStep2($(this));
        return;
      }
      switchPanel(currentStep + 1);
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
