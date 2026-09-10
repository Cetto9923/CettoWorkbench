// PO 工作台 - 发起交付交互逻辑 (对齐禅道 deliver 校验与上线窗口选择)

(function () {
  "use strict";

  var LAUNCH_WINDOW_OTHER = "__launch_other__";
  var currentCtx = null, isSubmitting = false;

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function demandIdOf(value) {
    var m = String(value == null ? "" : value).trim().match(/(?:US|REQ|DEMAND)?[-#]?(\d+)/i);
    return m ? m[1] : "";
  }

  function showToast(msg, type) {
    if (typeof window.showToast === "function") window.showToast(msg, type);
    else alert(msg);
  }

  function pad2(n) { return String(n).padStart(2, "0"); }
  function localCalendarIsoFromDate(d) {
    return d.getFullYear() + "-" + pad2(d.getMonth() + 1) + "-" + pad2(d.getDate());
  }
  function todayIso(refDate) { return localCalendarIsoFromDate(refDate || new Date()); }

  function cleanDate(value) {
    var s = String(value || "").trim().slice(0, 10);
    if (!s || s.indexOf("0000") === 0 || s === "0001-01-01") return "";
    return /^\d{4}-\d{2}-\d{2}$/.test(s) ? s : "";
  }

  function findWindowInList(windows, windowId) {
    var id = Number(windowId || 0);
    if (!id || !Array.isArray(windows)) return null;
    for (var i = 0; i < windows.length; i++) {
      if (Number(windows[i] && windows[i].id || 0) === id) return windows[i];
    }
    return null;
  }

  function findWindowByReleaseDate(windows, releaseDate) {
    var date = cleanDate(releaseDate);
    if (!date || !Array.isArray(windows)) return null;
    for (var i = 0; i < windows.length; i++) {
      if (cleanDate(windows[i] && windows[i].releaseDate) === date) return windows[i];
    }
    return null;
  }

  function windowReleaseDate(windows, windowId) {
    var hit = findWindowInList(windows, windowId);
    return hit ? cleanDate(hit.releaseDate) : "";
  }

  function defaultLaunchWindowName(releaseDate) {
    var date = cleanDate(releaseDate);
    return date ? (date + " 窗口") : "新版本窗口";
  }

  function buildLaunchWindowOptions(windows, selectedValue) {
    var html = '<option value="">请选择上线窗口</option>';
    (windows || []).forEach(function (w) {
      var id = Number(w && w.id || 0);
      if (!id) return;
      var name = String(w.name || ("窗口" + id)).trim(), rd = cleanDate(w.releaseDate);
      var label = rd ? (name + " · " + rd) : name;
      html += '<option value="' + id + '"' + (String(selectedValue) === String(id) ? " selected" : "") + ">" + esc(label) + "</option>";
    });
    html += '<option value="' + LAUNCH_WINDOW_OTHER + '"' + (selectedValue === LAUNCH_WINDOW_OTHER ? " selected" : "") + ">＋ 选择其他上线时间</option>";
    return html;
  }

  function resolveLaunchNotice(ctx) {
    if (!ctx) return { text: "", visible: false };
    if (ctx.launchMode === "window" && Number(ctx.selectedWindowId || 0) > 0) return { text: "", visible: false };
    if (ctx.launchMode !== "other") return { text: "", visible: false };
    var date = cleanDate(ctx.deliverDate);
    if (!date) return { text: "", visible: false };
    var hit = findWindowByReleaseDate((ctx.scheduling && ctx.scheduling.windows) || [], date);
    if (hit) {
      return { text: "该日期已有 " + String(hit.name || defaultLaunchWindowName(date)).trim() + " 窗口，将直接关联", visible: true };
    }
    return { text: "当前日期暂无上线窗口，保存后将自动创建『" + defaultLaunchWindowName(date) + "』并关联本次交付", visible: true };
  }

  function updateLaunchWindowUI(ctx) {
    if (!ctx) return;
    var notice = resolveLaunchNotice(ctx);
    var noticeEl = document.getElementById("poLaunchWindowNotice");
    var dateWrap = document.getElementById("poLaunchDateFieldWrap");
    if (noticeEl) {
      noticeEl.textContent = notice.text;
      noticeEl.style.display = notice.visible ? "" : "none";
    }
    if (dateWrap) dateWrap.style.display = ctx.launchMode === "other" ? "" : "none";
    if (ctx.launchMode === "other") {
      var dateEl = document.getElementById("poLaunchDate");
      if (dateEl && !dateEl.value && ctx.deliverDate) dateEl.value = ctx.deliverDate;
    }
  }

  function onPoLaunchWindowChange() {
    var ctx = currentCtx, sel = document.getElementById("poLaunchWindowSelect");
    if (!ctx || !sel) return;
    var raw = String(sel.value || "");
    if (raw === LAUNCH_WINDOW_OTHER) {
      ctx.launchMode = "other"; ctx.selectedWindowId = 0;
      if (!ctx.deliverDate) ctx.deliverDate = cleanDate((ctx.defaults && ctx.defaults.deliverDate) || "") || todayIso();
    } else if (Number(raw) > 0) {
      ctx.launchMode = "window"; ctx.selectedWindowId = Number(raw);
      ctx.deliverDate = windowReleaseDate((ctx.scheduling && ctx.scheduling.windows) || [], ctx.selectedWindowId);
    } else {
      ctx.launchMode = ""; ctx.selectedWindowId = 0; ctx.deliverDate = "";
    }
    updateLaunchWindowUI(ctx);
  }

  function onPoLaunchDateChange() {
    var ctx = currentCtx, el = document.getElementById("poLaunchDate");
    if (!ctx || !el) return;
    ctx.launchMode = "other"; ctx.selectedWindowId = 0;
    ctx.deliverDate = cleanDate(el.value);
    updateLaunchWindowUI(ctx);
  }

  function renderPrecheck(precheck) {
    var container = document.getElementById("poDeliverChecklist");
    var errEl = document.getElementById("poDeliverPrecheckError"), hintEl = document.getElementById("poDeliverPrecheckHint");
    if (!container) return;

    var rows = (precheck && precheck.rows) || [];
    container.innerHTML = rows.map(function (row) {
      var tone = row.ok ? "ad-tag ok" : "ad-tag bad";
      return '<div class="ad-check-row"><span>' + esc(row.label) + '</span><span class="' + tone + '">' + esc(row.value) + "</span></div>";
    }).join("");

    if (precheck && !precheck.canSubmit && precheck.blockReason) {
      if (errEl) { errEl.textContent = precheck.blockReason; errEl.style.display = ""; }
      if (hintEl) hintEl.style.display = "none";
    } else {
      if (errEl) errEl.style.display = "none";
      if (hintEl) hintEl.style.display = "";
    }

    var submitBtn = document.getElementById("poDeliverConfirmBtn");
    if (submitBtn) {
      var disabled = precheck && precheck.canSubmit === false;
      submitBtn.disabled = disabled;
      submitBtn.title = disabled ? (precheck.blockReason || "前置条件未满足") : "";
    }
  }

  function findUserLabel(users, val) {
    if (!val) return "";
    for (var i = 0; i < (users || []).length; i++) {
      if (String(users[i].value) === String(val)) return users[i].label;
    }
    return val;
  }

  function bindVerifierAutocomplete(users, selectedVal) {
    if (typeof window.destroyAutocomplete === "function") window.destroyAutocomplete("poDeliverVerifierInput");
    if (typeof window.initAutocomplete === "function") {
      window.initAutocomplete("poDeliverVerifierInput", "poDeliverVerifierValue", users || [], {
        value: selectedVal || "", label: findUserLabel(users, selectedVal), placeholder: "输入姓名或工号搜索"
      });
    } else {
      var inp = document.getElementById("poDeliverVerifierInput"), val = document.getElementById("poDeliverVerifierValue");
      if (inp) inp.value = selectedVal || "";
      if (val) val.value = selectedVal || "";
    }
  }

  function renderForm(ctx) {
    var d = ctx.defaults || {}, sched = ctx.scheduling || {};
    var selectedValue = ctx.launchMode === "other" ? LAUNCH_WINDOW_OTHER : String(ctx.selectedWindowId || sched.windowId || "");
    var sel = document.getElementById("poLaunchWindowSelect");
    if (sel) sel.innerHTML = buildLaunchWindowOptions(sched.windows, selectedValue);

    var isCar = String(d.isCarReview || "0") === "1";
    document.querySelectorAll('input[name="poDeliverCarReview"]').forEach(function (r) {
      if (r.value === "1") r.checked = isCar;
      if (r.value === "0") r.checked = !isCar;
    });

    var grayEl = document.getElementById("poDeliverGrayPlan");
    if (grayEl) {
      var grayVal = String(d.isGrayVerifyPlan == null ? "" : d.isGrayVerifyPlan);
      grayEl.value = grayVal !== "" ? grayVal : "0";
    }

    var vDateEl = document.getElementById("poDeliverVerifyDate");
    if (vDateEl) vDateEl.value = String(d.verifyDate || "1");

    var vPlanEl = document.getElementById("poDeliverVerifyPlan");
    if (vPlanEl) vPlanEl.value = d.verifyPlan || "";

    bindVerifierAutocomplete(ctx.userOptions || [], d.verifier || "");
    updateLaunchWindowUI(ctx);
  }

  function loadDeliverData(demandId) {
    var fetchFn = window.appFetch || window.fetch;
    return fetchFn("/demands/" + encodeURIComponent(demandId) + "/deliver", {
      method: "GET", headers: { Accept: "application/json" }
    })
      .then(function (res) {
        if (!res.ok) return res.json().then(function (d) { throw new Error(d.message || "加载交付信息失败"); });
        return res.json();
      })
      .then(function (data) {
        var precheck = data.precheck || { canSubmit: true, rows: [] };
        var linkedWindow = data.linkedWindow || {}, windows = Array.isArray(data.windows) ? data.windows : [];
        var defaults = data.defaults || {};
        var initialWindowId = Number(linkedWindow.id || 0);
        var initialDeliverDate = cleanDate(defaults.deliverDate || linkedWindow.releaseDate || "");

        var ctx = {
          demandId: String(demandId), defaults: defaults, userOptions: data.userOptions || [],
          scheduling: { windowId: initialWindowId, windowName: linkedWindow.name || "", releaseDate: cleanDate(linkedWindow.releaseDate || ""), windows: windows },
          launchMode: initialWindowId > 0 ? "window" : "", selectedWindowId: initialWindowId,
          deliverDate: initialDeliverDate, precheck: precheck
        };
        if (initialWindowId === 0 && initialDeliverDate) ctx.launchMode = "other";

        currentCtx = ctx; window._poDeliverCtx = ctx;
        renderPrecheck(precheck); renderForm(ctx);
        $("#poDeliverLoadingState").hide(); $("#poDeliverForm").show();
      })
      .catch(function (err) {
        $("#poDeliverLoadingState").hide();
        $("#poDeliverErrorMsg").text(err.message || "加载失败，请稍后重试");
        $("#poDeliverErrorState").show();
      });
  }

  function readFormData(ctx) {
    var car = document.querySelector('input[name="poDeliverCarReview"]:checked');
    var windowId = 0, deliverDate = "";

    if (ctx && ctx.launchMode === "window") {
      windowId = Number(ctx.selectedWindowId || 0);
      deliverDate = cleanDate(ctx.deliverDate);
    } else if (ctx && ctx.launchMode === "other") {
      var dateEl = document.getElementById("poLaunchDate");
      deliverDate = cleanDate(dateEl && dateEl.value) || cleanDate(ctx.deliverDate);
      var hit = findWindowByReleaseDate((ctx.scheduling && ctx.scheduling.windows) || [], deliverDate);
      windowId = hit ? Number(hit.id || 0) : 0;
      if (ctx) ctx.deliverDate = deliverDate;
    }

    var verifierVal = "";
    var verifierInput = document.getElementById("poDeliverVerifierValue");
    if (verifierInput) verifierVal = String(verifierInput.value || "").trim();
    if (!verifierVal) {
      var plainInput = document.getElementById("poDeliverVerifierInput");
      if (plainInput) verifierVal = String(plainInput.value || "").trim();
    }

    return {
      windowId: windowId, deliverDate: deliverDate, isCarReview: car ? car.value : "0",
      isGrayVerifyPlan: (document.getElementById("poDeliverGrayPlan") || {}).value || "",
      verifyDate: (document.getElementById("poDeliverVerifyDate") || {}).value || "",
      verifyPlan: String((document.getElementById("poDeliverVerifyPlan") || {}).value || "").trim(),
      verifier: verifierVal
    };
  }

  function validateFormData(data, ctx) {
    if (ctx && ctx.precheck && ctx.precheck.canSubmit === false) {
      showToast(ctx.precheck.blockReason || "前置条件未满足，暂不能发起交付", "error"); return false;
    }
    if (!ctx || !ctx.launchMode) { showToast("请选择上线窗口", "error"); return false; }
    if (ctx.launchMode === "window" && !data.windowId) { showToast("请选择上线窗口", "error"); return false; }
    if (ctx.launchMode === "other" && !data.deliverDate) { showToast("请选择上线时间", "error"); return false; }
    if (!data.deliverDate) { showToast("请确认上线时间", "error"); return false; }
    if (data.deliverDate < todayIso()) { showToast("上线时间不能早于今天", "error"); return false; }
    if (data.isGrayVerifyPlan === "") { showToast("请选择灰度验证计划", "error"); return false; }
    if (!data.verifyDate) { showToast("请选择生产验证时间", "error"); return false; }
    if (!data.verifyPlan) { showToast("请填写生产验证计划", "error"); return false; }
    if (!data.verifier) { showToast("请选择生产验证责任人", "error"); return false; }
    return true;
  }

  function openPoDeliverModal(id) {
    var numeric = demandIdOf(id);
    if (!numeric) { showToast("无法识别业务需求编号", "error"); return false; }
    var overlay = document.getElementById("poDeliverModalOverlay");
    if (!overlay) return false;

    $("#poDeliverModalTitle").text("发起交付 · US" + numeric);
    $(overlay).addClass("show").css("display", "flex").attr("aria-hidden", "false");
    $("#poDeliverLoadingState").show();
    $("#poDeliverErrorState, #poDeliverForm").hide();

    loadDeliverData(numeric);
    return false;
  }

  function closePoDeliverModal() {
    if (typeof window.destroyAutocomplete === "function") window.destroyAutocomplete("poDeliverVerifierInput");
    var overlay = document.getElementById("poDeliverModalOverlay");
    if (overlay) {
      $(overlay).removeClass("show").attr("aria-hidden", "true");
      setTimeout(function () { if (!$(overlay).hasClass("show")) $(overlay).css("display", "none"); }, 240);
    }
    currentCtx = null; window._poDeliverCtx = null; isSubmitting = false;
  }

  function submitPoDeliverModal() {
    var ctx = currentCtx;
    if (!ctx || !ctx.demandId || isSubmitting) return;

    var data = readFormData(ctx);
    if (!validateFormData(data, ctx)) return;

    var btn = document.getElementById("poDeliverConfirmBtn");
    if (btn) btn.disabled = true;
    isSubmitting = true;

    var fetchFn = window.appFetch || window.fetch;
    fetchFn("/demands/" + encodeURIComponent(ctx.demandId) + "/deliver", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({
        windowId: data.windowId || 0, deliverDate: data.deliverDate,
        isCarReview: data.isCarReview, isGrayVerifyPlan: data.isGrayVerifyPlan,
        verifyDate: data.verifyDate, verifyPlan: data.verifyPlan,
        verifier: data.verifier, comment: "工作台发起交付"
      })
    })
      .then(function (res) {
        return res.json().then(function (body) {
          if (!res.ok || body.success === false) {
            throw new Error(body.message || (body.errors && body.errors[0] && body.errors[0].message) || "发起交付失败");
          }
          return body;
        });
      })
      .then(function () {
        showToast("US" + ctx.demandId + " 已发起交付，进入待发布", "success");
        closePoDeliverModal();
        if (typeof window.refreshPoHomeDemands === "function") window.refreshPoHomeDemands();
        if (window.DemandDetail && typeof window.DemandDetail.refresh === "function") {
          window.DemandDetail.refresh();
        } else if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
      })
      .catch(function (err) {
        showToast(err.message || "发起交付失败，请稍后重试", "error");
      })
      .finally(function () {
        isSubmitting = false;
        if (btn) btn.disabled = false;
      });
  }

  $(function () {
    $("#poDeliverCloseBtn, #poDeliverCancelBtn").on("click", closePoDeliverModal);
    $("#poDeliverModalOverlay").on("click", function (e) { if (e.target === this) closePoDeliverModal(); });
    $("#poDeliverRetryBtn").on("click", function () {
      if (currentCtx && currentCtx.demandId) {
        $("#poDeliverLoadingState").show(); $("#poDeliverErrorState").hide();
        loadDeliverData(currentCtx.demandId);
      }
    });
    $("#poLaunchWindowSelect").on("change", onPoLaunchWindowChange);
    $("#poLaunchDate").on("change", onPoLaunchDateChange);
    $("#poDeliverForm").on("submit", function (e) { e.preventDefault(); submitPoDeliverModal(); });
  });

  window.openPoDeliverModal = openPoDeliverModal;
  window.closePoDeliverModal = closePoDeliverModal;
  window.submitPoDeliverModal = submitPoDeliverModal;
  window.onPoLaunchWindowChange = onPoLaunchWindowChange;
  window.onPoLaunchDateChange = onPoLaunchDateChange;
  window.__poDeliverLaunchHelpers = {
    LAUNCH_WINDOW_OTHER: LAUNCH_WINDOW_OTHER,
    buildLaunchWindowOptions: buildLaunchWindowOptions,
    findWindowByReleaseDate: findWindowByReleaseDate,
    defaultLaunchWindowName: defaultLaunchWindowName,
    resolveLaunchNotice: resolveLaunchNotice,
    localCalendarIsoFromDate: localCalendarIsoFromDate,
    todayIso: todayIso,
    readFormData: readFormData,
    validateFormData: validateFormData,
    demandIdOf: demandIdOf
  };
})();
