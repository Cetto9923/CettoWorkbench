/*
 * 登录页提交：用 fetch 拿 JSON，等浏览器收下 session cookie 后再跳转，
 * 避免 POST 303 跟跳时 Chrome 还没带上新 cookie，又被送回登录页。
 */
(function () {
  "use strict";

  var form = document.getElementById("login-form");
  if (!form) {
    return;
  }

  function showFormError(message) {
    var box = document.getElementById("login-error");
    if (!box) {
      return;
    }
    box.innerHTML = "";
    var alert = document.createElement("div");
    alert.className = "alert alert-danger";
    alert.textContent = message || "登录失败";
    box.appendChild(alert);
  }

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    var submitBtn = form.querySelector('button[type="submit"],input[type="submit"]');
    if (submitBtn) {
      submitBtn.disabled = true;
    }

    var body = new URLSearchParams(new FormData(form));
    fetch(form.getAttribute("action") || "/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest"
      },
      credentials: "same-origin",
      redirect: "manual",
      body: body.toString()
    })
      .then(function (resp) {
        return resp.json().then(function (data) {
          return { status: resp.status, data: data || {} };
        });
      })
      .then(function (result) {
        var data = result.data;
        if (data.success && data.redirectUrl) {
          window.location.href = data.redirectUrl;
          return;
        }
        var msg = data.message || "登录失败";
        if (Array.isArray(data.errors) && data.errors.length > 0) {
          var formErr = data.errors.filter(function (item) {
            return item && item.field === "_form";
          })[0];
          msg = (formErr && formErr.message) || data.errors[0].message || msg;
        }
        showFormError(msg);
        if (submitBtn) {
          submitBtn.disabled = false;
        }
      })
      .catch(function () {
        showFormError("登录失败，请稍后重试");
        if (submitBtn) {
          submitBtn.disabled = false;
        }
      });
  });
})();
