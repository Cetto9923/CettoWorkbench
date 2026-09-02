/*
 * 用户批量新增页面提交脚本（AJAX）。
 */
(function () {
  "use strict";

  var form = document.querySelector("form.batch-form");
  if (!form) {
    return;
  }

  var submitBtn = form.querySelector('button[type="submit"]');
  var rowSelector = "[data-batch-row]";

  function readText(row, suffix) {
    var el = row.querySelector('input[name$="' + suffix + '"]');
    return el ? (el.value || "").trim() : "";
  }

  function readDeptID(row) {
    var el = row.querySelector("select.form-tree-select-native");
    if (!el || !el.value) {
      return 0;
    }
    var parsed = Number(el.value);
    return Number.isFinite(parsed) ? parsed : 0;
  }

  function readGender(row) {
    var checked = row.querySelector('input[type="radio"][name$=".gender"]:checked');
    return checked ? (checked.value || "").trim() : "";
  }

  function readRoleIDs(row) {
    var ids = [];
    var checked = row.querySelectorAll('input[type="checkbox"][name$=".roleIds"]:checked');
    for (var i = 0; i < checked.length; i++) {
      var parsed = Number(checked[i].value);
      if (Number.isFinite(parsed)) {
        ids.push(parsed);
      }
    }
    return ids;
  }

  function buildPayload() {
    var rows = form.querySelectorAll(rowSelector);
    var users = [];
    for (var i = 0; i < rows.length; i++) {
      var row = rows[i];
      users.push({
        account: readText(row, ".account"),
        displayName: readText(row, ".displayName"),
        deptID: readDeptID(row),
        email: readText(row, ".email"),
        gender: readGender(row),
        password: readText(row, ".password"),
        roleIds: readRoleIDs(row)
      });
    }
    return { users: users };
  }

  function showError(message) {
    if (typeof window.showToast === "function") {
      window.showToast(message || "操作失败，请稍后重试", "error");
      return;
    }
    window.alert(message || "操作失败，请稍后重试");
  }

  function showSuccess(message) {
    if (typeof window.showToast === "function") {
      window.showToast(message || "操作成功", "success");
      return;
    }
    window.alert(message || "操作成功");
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();

    var payload = buildPayload();
    if (submitBtn) {
      submitBtn.disabled = true;
      submitBtn.classList.add("loading");
    }

    fetch(form.action, {
      method: "POST",
      body: JSON.stringify(payload),
      headers: {
        "X-Requested-With": "XMLHttpRequest",
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "same-origin",
    })
      .then(function (response) {
        return response
          .json()
          .catch(function () {
            return {};
          })
          .then(function (data) {
            return { status: response.status, ok: response.ok, data: data };
          });
      })
      .then(function (result) {
        if (!result.ok) {
          if (result.status === 422 && Array.isArray(result.data.errors) && result.data.errors.length > 0) {
            var first = result.data.errors[0];
            var msg = "第 " + first.row + " 行：" + first.message;
            showError(msg);
            return;
          }
          showError(result.data.message || "批量创建失败");
          return;
        }

        showSuccess(result.data.message || "批量创建成功");
        window.setTimeout(function () {
          window.location.href = "/admin/users";
        }, 350);
      })
      .catch(function () {
        showError("网络异常，请稍后重试");
      })
      .finally(function () {
        if (submitBtn) {
          submitBtn.disabled = false;
          submitBtn.classList.remove("loading");
        }
      });
  });
})();
