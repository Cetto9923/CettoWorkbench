(function () {
  "use strict";

  function redirectToLogin() {
    window.location.href = "/login?redirect=" + encodeURIComponent(window.location.pathname);
  }

  function isSessionExpired(resp, init) {
    if (!resp) {
      return false;
    }
    var url = String(resp.url || "");
    if (resp.redirected && url.indexOf("/login") !== -1) {
      return true;
    }
    if (url.indexOf("/login") !== -1) {
      return true;
    }
    if (resp.status === 401) {
      return true;
    }
    var accept = "";
    if (init && init.headers) {
      var headers = init.headers instanceof Headers ? init.headers : new Headers(init.headers);
      accept = headers.get("Accept") || "";
    }
    if (accept.indexOf("application/json") !== -1) {
      var ct = (resp.headers.get("content-type") || "").toLowerCase();
      if (ct.indexOf("application/json") === -1) {
        return true;
      }
    }
    return false;
  }

  function scheduleFetch(input, init) {
    var options = init || {};
    var fetchFn = window.appFetch || fetch;
    return fetchFn(input, options).then(function (resp) {
      if (isSessionExpired(resp, options)) {
        redirectToLogin();
        return Promise.reject(new Error("session expired"));
      }
      return resp;
    });
  }

  /** 显示/隐藏带 .show 类名的弹窗与遮罩（schedule 模块自包含，不依赖 ui.js）。 */
  function setScheduleModals(elementIds, visible) {
    var ids = Array.isArray(elementIds) ? elementIds : [elementIds];
    for (var i = 0; i < ids.length; i++) {
      var el = document.getElementById(ids[i]);
      if (!el) {
        continue;
      }
      if (visible) {
        el.classList.add("show");
      } else {
        el.classList.remove("show");
      }
    }
  }

  function showScheduleModals(elementIds) {
    setScheduleModals(elementIds, true);
  }

  function hideScheduleModals(elementIds) {
    setScheduleModals(elementIds, false);
  }

  window.scheduleFetch = scheduleFetch;
  window.showScheduleModals = showScheduleModals;
  window.hideScheduleModals = hideScheduleModals;
})();
