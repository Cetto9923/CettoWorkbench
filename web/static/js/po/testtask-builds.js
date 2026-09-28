/*
 * 文件: web/static/js/po/testtask-builds.js
 * 模块: PO工作台
 * 职责: 提测办理「按系统拉执行 + 草稿本地缓存 + 提交新版本到禅道」后端交互。
 *       UI / 上下文 / 步骤导航在 testtask.js；本文件通过 window.PoTesttaskCore 复用其状态/工具。
 * 协议: 所有写操作用 appFetch（自动 X-CSRF-Token / X-Requested-With）。
 * 写入流程见 testtask-submit.js；此文件负责查询与本地草稿。
 */
(function ($) {
  "use strict";

  // 各单位的执行加载状态：idle/loading/loaded/empty/error。
  var executionState = {};
  // 已加载过的产品执行下拉缓存（避免 step 2 重复请求）；key=productId。
  var executionCache = {};
  var buildCache = {};
  // 防止步骤切换时重复触发执行加载。
  var executionsLoaded = false;
  // 每次新打开弹窗都要重置。
  var initialized = false;

  function core() {
    return window.PoTesttaskCore || {};
  }
  function $root() {
    var fn = core().$root;
    return fn ? fn() : $("#poTesttaskFormRoot").first();
  }
  function demandId() {
    return core().getDemandId ? core().getDemandId() : "";
  }
  // ===== 系统 / 单位映射（context 加载完成后）=====

  function assignSystemToUnits($r) {
    var systems = (core().getSystems && core().getSystems()) || [];
    var $units = $r.find("[data-tt-unit]");
    $units.each(function (idx) {
      var unitNo = idx + 1;
      var sys = systems[idx];
      var pid = (sys && sys.id) ? String(sys.id) : "";
      $(this).attr("data-tt-unit-product", pid);
      var $exec = $r.find('[data-tt-exec="' + unitNo + '"]');
      if (!pid) {
        $exec.empty().append($("<option></option>").attr("value", "").text("该单位无产品配置"));
        $exec.prop("disabled", true);
      } else {
        $exec.empty().append($("<option></option>").attr("value", "").text("加载中…"));
        $exec.prop("disabled", true);
      }
    });
  }

  function loadExecutionsForAllUnits() {
    if (executionsLoaded) {
      return;
    }
    executionsLoaded = true;
    var $r = $root();
    $r.find("[data-tt-unit]").each(function () {
      var unitNo = parseInt($(this).attr("data-tt-unit"), 10) || 0;
      if (!unitNo) {
        return;
      }
      var productId = parseInt($(this).attr("data-tt-unit-product"), 10) || 0;
      if (!productId) {
        return;
      }
      loadExecutionsForUnit(unitNo, productId);
      loadBuildsForUnit(unitNo, productId);
    });
  }

  function loadBuildsForUnit(unitNo, productId) {
    if (buildCache[productId]) { renderBuildsForUnit(unitNo, buildCache[productId]); return; }
    var $select = $root().find('[data-tt-exist-ver="' + unitNo + '"]');
    if (!$select.length) return;
    $select.empty().append($('<option></option>').val('').text('加载中…')).prop('disabled', true);
    (window.appFetch || fetch)("/products/" + encodeURIComponent(productId) + "/builds", { method: "GET", headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" } })
      .then(function (res) { return res.json().then(function (body) { return { ok: res.ok, status: res.status, body: body }; }); })
      .then(function (wrap) {
        if (!wrap.ok || !wrap.body || !wrap.body.success) throw new Error((wrap.body && wrap.body.message) || ("获取版本列表失败 (HTTP " + wrap.status + ")"));
        buildCache[productId] = Array.isArray(wrap.body.data) ? wrap.body.data : [];
        renderBuildsForUnit(unitNo, buildCache[productId]);
      })
      .catch(function (err) { $select.empty().append($('<option></option>').val('').text('加载失败，请重试')).prop('disabled', true); window.showToast((err && err.message) || '获取版本列表失败', 'error'); });
  }

  function renderBuildsForUnit(unitNo, list) {
    var $select = $root().find('[data-tt-exist-ver="' + unitNo + '"]'); if (!$select.length) return;
    $select.empty();
    if (!list.length) { $select.append($('<option></option>').val('').text('暂无已有版本')).prop('disabled', true); return; }
    $select.append($('<option></option>').val('').text('请选择已有版本'));
    list.forEach(function (item) { $select.append($('<option></option>').val(String(item.value || '')).text(String(item.label || item.value || ''))); });
    $select.prop('disabled', false);
  }

  function loadExecutionsForUnit(unitNo, productId) {
    if (executionCache[productId]) {
      renderExecutionsForUnit(unitNo, productId, executionCache[productId]);
      return;
    }
    if (executionState[productId] === "loading") {
      return;
    }
    executionState[productId] = "loading";
    var $r = $root();
    var $exec = $r.find('[data-tt-exec="' + unitNo + '"]');
    var fetchFn = window.appFetch || fetch;
    fetchFn("/products/" + encodeURIComponent(productId) + "/executions", {
      method: "GET",
      headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" }
    })
      .then(function (res) {
        return res.text().then(function (text) {
          var body = {};
          try { body = text ? JSON.parse(text) : {}; } catch (ignore) { body = {}; }
          return { ok: res.ok, status: res.status, body: body };
        });
      })
      .then(function (wrap) {
        if (!wrap.ok || !wrap.body || !wrap.body.success) {
          var msg = (wrap.body && wrap.body.message) ? wrap.body.message : ("获取执行列表失败 (HTTP " + wrap.status + ")");
          throw new Error(msg);
        }
        var list = Array.isArray(wrap.body.data) ? wrap.body.data : [];
        executionCache[productId] = list;
        executionState[productId] = list.length ? "loaded" : "empty";
        renderExecutionsForUnit(unitNo, productId, list);
      })
      .catch(function (err) {
        executionState[productId] = "error";
        $exec.empty().append($("<option></option>").attr("value", "").text("加载失败，请稍后重试"));
        $exec.prop("disabled", true);
        var msg = (err && err.message) ? ("执行下拉加载失败：" + err.message) : "执行下拉加载失败";
        window.showToast(msg, "error");
      });
  }

  function renderExecutionsForUnit(unitNo, productId, list) {
    var $r = $root();
    var $exec = $r.find('[data-tt-exec="' + unitNo + '"]');
    $exec.empty();
    if (!list || !list.length) {
      $exec.append($("<option></option>").attr("value", "").text("该系统暂无执行可选"));
      $exec.prop("disabled", true);
      return;
    }
    $exec.append($("<option></option>").attr("value", "").text("请选择执行"));
    list.forEach(function (item) {
      var $opt = $("<option></option>").attr("value", String(item.value || "")).text(String(item.label || item.value || ""));
      $opt.data("project-id", item.projectId || 0);
      $exec.append($opt);
    });
    $exec.prop("disabled", false);
    var draft = readDraft();
    var savedSel = draft && draft.units && draft.units[unitNo] && draft.units[unitNo].execValue;
    if (savedSel) {
      $exec.val(savedSel);
    }
  }

  // ===== 草稿（sessionStorage）=====

  function draftKey() {
    return "poTesttaskDraft:" + (demandId() || "unknown");
  }

  function readDraft() {
    try {
      var raw = window.sessionStorage.getItem(draftKey());
      if (!raw) {
        return null;
      }
      return JSON.parse(raw);
    } catch (ignore) {
      return null;
    }
  }

  function collectDraft() {
    var $r = $root();
    var units = {};
    $r.find("[data-tt-unit]").each(function () {
      var unitNo = parseInt($(this).attr("data-tt-unit"), 10) || 0;
      if (!unitNo) {
        return;
      }
      var $exec = $r.find('[data-tt-exec="' + unitNo + '"]');
      var $verRadio = $r.find('input[name="ver' + unitNo + 'Mode"]:checked');
      var mode = $verRadio.length ? $verRadio.val() : "new";
      units[unitNo] = {
        mode: mode,
        execValue: $exec.length ? String($exec.val() || "") : "",
        productId: parseInt($(this).attr("data-tt-unit-product"), 10) || 0,
        name: ($r.find('[data-tt-ver-name="' + unitNo + '"]').val() || "").trim(),
        date: ($r.find('[data-tt-ver-date="' + unitNo + '"]').val() || "").trim(),
        desc: ($r.find('[data-tt-ver-desc="' + unitNo + '"]').val() || "").trim()
      };
    });
    return {
      demandId: demandId(),
      savedAt: new Date().toISOString(),
      units: units
    };
  }

  function applyDraft(draft) {
    if (!draft || !draft.units) {
      return;
    }
    var $r = $root();
    Object.keys(draft.units).forEach(function (key) {
      var unitNo = parseInt(key, 10) || 0;
      if (!unitNo) {
        return;
      }
      var u = draft.units[key];
      if (u.mode) {
        var $radio = $r.find('input[name="ver' + unitNo + 'Mode"][value="' + u.mode + '"]');
        if ($radio.length) {
          $radio.prop("checked", true).trigger("change");
        }
      }
      $r.find('[data-tt-ver-name="' + unitNo + '"]').val(u.name || "");
      $r.find('[data-tt-ver-date="' + unitNo + '"]').val(u.date || "");
      $r.find('[data-tt-ver-desc="' + unitNo + '"]').val(u.desc || "");
    });
  }

  function restoreDraftIfAny() {
    var d = readDraft();
    if (!d) {
      return;
    }
    applyDraft(d);
    window.showToast("已恢复本需求的本地草稿（" + (d.savedAt || "") + "）", "info");
  }

  function handleSaveDraft() {
    if (!demandId()) {
      window.showToast("缺少需求上下文，无法保存草稿", "error");
      return;
    }
    var draft = collectDraft();
    try {
      window.sessionStorage.setItem(draftKey(), JSON.stringify(draft));
      window.showToast("草稿已存到本地（key: " + draftKey() + "）", "success");
    } catch (err) {
      window.showToast("草稿保存失败：" + (err && err.message ? err.message : "未知错误"), "error");
    }
  }

  // ===== 暴露给 testtask.js 的钩子 =====

  window.PoTesttaskBuilds_onContextLoaded = function ($r) {
    assignSystemToUnits($r);
    executionsLoaded = false;
    buildCache = {};
    executionCache = {};
    executionState = {};
    restoreDraftIfAny();
  };

  window.PoTesttaskBuilds_onEnterStep2 = function () {
    loadExecutionsForAllUnits();
  };

  window.PoTesttaskBuilds_reset = function () {
    if (window.PoTesttaskSubmit) window.PoTesttaskSubmit.reset();
    buildCache = {};
    executionCache = {};
    executionState = {};
    executionsLoaded = false;
  };

  // ===== 自身绑定 =====

  function bindHandlers() {
    if (initialized) {
      return;
    }
    initialized = true;
    $(document).on("click", "#poTesttaskSaveDraftBtn", handleSaveDraft);

  }

  $(bindHandlers);
})(jQuery);
