// =============================================================================
// 文件: web/static/js/role/form.js
// 模块: 角色管理
// 职责: 新增与编辑角色的JSON提交、字段错误展示和重复提交保护。
// =============================================================================
(function () {
  "use strict";
  var form = document.getElementById("roleForm");
  if (!form) return;
  var submit = form.querySelector('button[type="submit"]');
  var errorBox = document.getElementById("roleFormError");
  var saving = false;

  function renderErrors(data) {
    form.querySelectorAll("[data-field-error]").forEach(function (el) {
      el.textContent = "";
      el.hidden = true;
    });
    errorBox.textContent = data.message || "保存失败，请重试";
    errorBox.hidden = false;
    (data.errors || []).forEach(function (error) {
      var el = form.querySelector('[data-field-error="' + error.field + '"]');
      if (el) {
        el.textContent = error.message;
        el.hidden = false;
      }
    });
  }

  form.addEventListener("submit", async function (event) {
    event.preventDefault();
    if (saving || submit.disabled) return;
    saving = true;
    submit.disabled = true;
    errorBox.hidden = true;
    try {
      var response = await window.appFetch(form.action, {
        method: form.dataset.method,
        headers: { "Content-Type": "application/json", Accept: "application/json" },
        body: JSON.stringify({ name: form.elements.name.value, remark: form.elements.remark.value })
      });
      var data = await response.json();
      if (!response.ok || data.success !== true) {
        renderErrors(data);
        return;
      }
      window.location.assign(data.redirectUrl);
    } catch (error) {
      renderErrors({ message: "请求失败，请重试" });
    } finally {
      saving = false;
      submit.disabled = false;
    }
  });
})();
