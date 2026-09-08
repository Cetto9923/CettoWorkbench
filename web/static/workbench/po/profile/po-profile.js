/* =============================================================================
 * 文件: web/static/workbench/po/profile/po-profile.js
 * 模块: PO 工作台 / 个人资料（/profile）
 * 职责: GET /profile/data 加载 → 渲染表单 → PUT /profile 保存；
 *       独立的修改密码表单 → PUT /profile/password。
 * 字段: 5 只读 (account / displayName / dept / deptName / 由后端模板渲染)；
 *       4 可编辑 (displayName / email / mobile / gender) + 默认小组下拉。
 * 简化: 不接入 workbench role switch / preferredRoles。
 * 依赖: window.appFetch (app.js), window.showToast (ui.js)。
 * ============================================================================= */

(function () {
  "use strict";

  /* ------------------------------ helpers ------------------------------ */

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }
  function getEl(id) { return document.getElementById(id); }
  function val(id, fb) { var el = getEl(id); return el ? String(el.value || "").trim() : (fb || ""); }
  function showToast(msg, type) {
    if (typeof window.showToast === "function") { window.showToast(msg, type || "info"); }
  }
  function setBusy(btn, busy) {
    if (!btn) { return; }
    btn.disabled = !!busy;
    btn.dataset.busy = busy ? "1" : "";
  }

  /* -------------------------- state (loading/ready/error) -------------------------- */

  function setState(state, opts) {
    var loadingEl = getEl("poProfileLoading");
    var formEl = getEl("poProfileForm");
    var errorEl = getEl("poProfileError");
    var errorMsg = getEl("poProfileErrorMsg");
    if (loadingEl) { loadingEl.hidden = state !== "loading"; }
    if (formEl) { formEl.hidden = state !== "ready"; }
    if (errorEl) { errorEl.hidden = state !== "error"; }
    if (state === "error" && opts && opts.message && errorMsg) { errorMsg.textContent = opts.message; }
  }

  /* -------------------- field / form-level error rendering -------------------- */

  var FIELD_IDS = {
    displayName: "profileDisplayName",
    email: "profileEmail",
    mobile: "profileMobile",
    gender: "profileGender",
    mainTeamId: "profileMainTeam"
  };

  function clearErrors() {
    Object.keys(FIELD_IDS).forEach(function (k) {
      var id = FIELD_IDS[k];
      var errBox = getEl(id + "Error");
      if (errBox) { errBox.textContent = ""; errBox.hidden = true; }
    });
    var formAlert = getEl("poProfileFormAlert");
    if (formAlert) { formAlert.hidden = true; formAlert.innerHTML = ""; }
  }

  function applyErrors(errors) {
    clearErrors();
    if (!Array.isArray(errors) || !errors.length) { return; }
    var summary = [];
    errors.forEach(function (err) {
      if (!err || typeof err !== "object") { return; }
      var f = err.field || err.key || "";
      var msg = err.message || err.msg || "校验失败";
      if (FIELD_IDS[f]) {
        var box = getEl(FIELD_IDS[f] + "Error");
        if (box) { box.textContent = msg; box.hidden = false; }
      }
      summary.push(msg);
    });
    var alert = getEl("poProfileFormAlert");
    if (alert && summary.length) {
      alert.hidden = false;
      alert.innerHTML = "<div><i class=\"bi bi-exclamation-triangle-fill\"></i></div><ul>" +
        summary.map(function (m) { return "<li>" + escapeHtml(m) + "</li>"; }).join("") + "</ul>";
    }
  }

  /* -------------------- form rendering from /profile/data -------------------- */

  function renderAgileGroups(data) {
    var sel = getEl("profileMainTeam");
    if (!sel) { return; }
    var current = data && data.mainTeamId != null ? Number(data.mainTeamId) : 0;
    var groups = (data && Array.isArray(data.agileGroups)) ? data.agileGroups : [];
    var html = '<option value=""' + (current === 0 ? " selected" : "") + ">（未设置）</option>";
    if (!groups.length) {
      html += '<option value="" disabled>（未加入任何敏捷小组）</option>';
    } else {
      groups.forEach(function (g) {
        var id = Number(g && g.id || 0);
        if (!id) { return; }
        var name = (g && g.name) || ("小组#" + id);
        html += '<option value="' + id + '"' + (id === current ? " selected" : "") +
          ">" + escapeHtml(name) + "</option>";
      });
    }
    sel.innerHTML = html;
  }

  function applyData(data) {
    if (!data) { return; }
    var accountEl = getEl("profileAccount");
    var nameEl = getEl("profileDisplayName");
    var emailEl = getEl("profileEmail");
    var mobileEl = getEl("profileMobile");
    var deptEl = getEl("profileDept");
    if (accountEl) { accountEl.textContent = data.account || ""; }
    if (nameEl) { nameEl.value = data.displayName || ""; }
    if (emailEl) { emailEl.value = data.email || ""; }
    if (mobileEl) { mobileEl.value = data.mobile || ""; }
    if (deptEl) {
      var deptText = data.deptName || (data.deptId ? ("部门#" + data.deptId) : "");
      deptEl.textContent = deptText || "（未分配）";
    }
    var g = (data.gender || "");
    var gm = getEl("profileGenderM"), gf = getEl("profileGenderF"), gu = getEl("profileGenderU");
    if (gm) { gm.checked = g === "m"; }
    if (gf) { gf.checked = g === "f"; }
    if (gu) { gu.checked = g === "" || g == null; }
    renderAgileGroups(data);
  }

  /* -------------------- network (uses window.appFetch → CSRF) -------------------- */

  function requestJson(method, url, body) {
    var fetchFn = window.appFetch || window.fetch;
    var init = { method: method, headers: { "Accept": "application/json" } };
    if (body != null) {
      init.headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }
    return fetchFn(url, init).then(function (res) {
      if (res.status === 401) {
        var e = new Error("登录已过期，请重新登录");
        e.isAuth = true;
        throw e;
      }
      if (!res.ok) {
        var se = new Error("服务响应异常 (" + res.status + ")");
        se.status = res.status;
        throw se;
      }
      return res.json().catch(function () { throw new Error("数据格式解析失败"); });
    });
  }

  /* -------------------- submit handlers -------------------- */

  function collectPayload() {
    var gender = "";
    if (getEl("profileGenderM") && getEl("profileGenderM").checked) { gender = "m"; }
    else if (getEl("profileGenderF") && getEl("profileGenderF").checked) { gender = "f"; }
    var teamEl = getEl("profileMainTeam");
    var mainTeamId = teamEl && teamEl.value ? Number(teamEl.value) : 0;
    if (!Number.isFinite(mainTeamId) || mainTeamId < 0) { mainTeamId = 0; }
    return {
      displayName: val("profileDisplayName", ""),
      email: val("profileEmail", ""),
      mobile: val("profileMobile", ""),
      gender: gender,
      mainTeamId: mainTeamId
    };
  }

  function showPwdAlert(msg) {
    var alert = getEl("poProfilePwdAlert");
    if (!alert) { return; }
    alert.hidden = false;
    alert.innerHTML = "<div><i class=\"bi bi-exclamation-triangle-fill\"></i></div><ul>" +
      "<li>" + escapeHtml(msg) + "</li></ul>";
  }

  function onSaveProfile() {
    clearErrors();
    var btn = getEl("poProfileSaveBtn");
    setBusy(btn, true);
    requestJson("PUT", "/profile", collectPayload())
      .then(function (json) {
        if (json && Array.isArray(json.errors) && json.errors.length) {
          applyErrors(json.errors);
          showToast(json.errors[0].message || "请检查表单", "danger");
          return;
        }
        if (!json || json.success === false) {
          showToast((json && (json.message || json.error)) || "保存失败", "danger");
          return;
        }
        if (json && json.data) { applyData(json.data); }
        showToast((json && json.message) || "个人资料已保存", "success");
      })
      .catch(function (err) {
        if (err && err.isAuth) { showToast(err.message || "登录已过期", "warning"); return; }
        showToast((err && err.message) || "保存失败，请稍后重试", "danger");
      })
      .then(function () { setBusy(btn, false); });
  }

  function onSavePassword() {
    var oldPwd = val("profileOldPwd", "");
    var newPwd = val("profileNewPwd", "");
    var confirm = val("profileNewPwd2", "");
    var alert = getEl("poProfilePwdAlert");
    if (alert) { alert.hidden = true; alert.innerHTML = ""; }

    if (!oldPwd) { showPwdAlert("请输入当前密码"); return; }
    if (!newPwd) { showPwdAlert("请输入新密码"); return; }
    if (newPwd.length < 8 || newPwd.length > 64) { showPwdAlert("新密码长度需为 8-64 位"); return; }
    if (newPwd !== confirm) { showPwdAlert("两次输入的新密码不一致"); return; }

    var btn = getEl("poProfilePwdSaveBtn");
    setBusy(btn, true);
    requestJson("PUT", "/profile/password", {
      oldPassword: oldPwd, newPassword: newPwd, confirmPassword: confirm
    }).then(function (json) {
      if (json && Array.isArray(json.errors) && json.errors.length) {
        var msgs = json.errors.map(function (e) { return (e && e.message) || "校验失败"; });
        if (alert) {
          alert.hidden = false;
          alert.innerHTML = "<div><i class=\"bi bi-exclamation-triangle-fill\"></i></div><ul>" +
            msgs.map(function (m) { return "<li>" + escapeHtml(m) + "</li>"; }).join("") + "</ul>";
        }
        showToast(msgs[0] || "请检查密码", "danger");
        return;
      }
      if (!json || json.success === false) {
        var msg = (json && (json.message || json.error)) || "改密失败";
        showPwdAlert(msg);
        showToast(msg, "danger");
        return;
      }
      ["profileOldPwd", "profileNewPwd", "profileNewPwd2"].forEach(function (id) {
        var el = getEl(id); if (el) { el.value = ""; }
      });
      showToast((json && json.message) || "密码已更新", "success");
    }).catch(function (err) {
      if (err && err.isAuth) { showToast(err.message || "登录已过期", "warning"); return; }
      showToast((err && err.message) || "改密失败，请稍后重试", "danger");
    }).then(function () { setBusy(btn, false); });
  }

  function onRetry() { bootstrap(); }

  /* -------------------- bootstrap -------------------- */

  function bindOnce() {
    var ids = ["poProfileSaveBtn", "poProfilePwdSaveBtn", "poProfileRetryBtn"];
    ids.forEach(function (id) {
      var el = getEl(id);
      if (!el || el.dataset.bound === "1") { return; }
      el.dataset.bound = "1";
      if (id === "poProfileSaveBtn") { el.addEventListener("click", onSaveProfile); }
      else if (id === "poProfilePwdSaveBtn") { el.addEventListener("click", onSavePassword); }
      else if (id === "poProfileRetryBtn") { el.addEventListener("click", onRetry); }
    });
  }

  function bootstrap() {
    bindOnce();
    if (!getEl("poProfileForm")) { return; }
    setState("loading");
    requestJson("GET", "/profile/data").then(function (json) {
      if (!json || json.success === false || !json.data) {
        setState("error", { message: (json && (json.message || json.error)) || "返回数据无效" });
        return;
      }
      applyData(json.data);
      setState("ready");
    }).catch(function (err) {
      if (err && err.isAuth) { setState("error", { message: err.message }); return; }
      setState("error", { message: (err && err.message) || "加载失败" });
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", bootstrap);
  } else {
    bootstrap();
  }
})();
