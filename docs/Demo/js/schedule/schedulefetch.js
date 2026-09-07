(function () {
  "use strict";

  function redirectToLogin() {
    window.location.href = "/login?redirect=" + encodeURIComponent(window.location.pathname);
  }

  function isSessionExpired(resp, init) {
    if (!resp) {
      return false;
    }
    if (resp.status === 401) {
      return true;
    }
    var url = String(resp.url || "");
    if (resp.redirected && url.indexOf("/login") !== -1) {
      return true;
    }
    // 403 (权限不足)、409 (并发冲突)、500+ (服务端错误) 均属于明确的业务/系统错误，不得当作会话过期处理
    if (resp.status === 403 || resp.status === 409 || resp.status >= 500) {
      return false;
    }
    if (url.indexOf("/login") !== -1) {
      return true;
    }
    return false;
  }

  function getCsrfToken() {
    if (typeof document === "undefined") {
      return "";
    }
    var el = document.querySelector('meta[name="csrf-token"]');
    if (!el) {
      return "";
    }
    return (el.getAttribute("content") || "").trim();
  }

  function scheduleFetch(input, init) {
    var options = init || {};
    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    if (fetchFn === fetch) {
      var csrf = (typeof window !== "undefined" && typeof window.getCsrfToken === "function")
        ? window.getCsrfToken()
        : getCsrfToken();
      if (csrf) {
        if (typeof Headers !== "undefined" && options.headers instanceof Headers) {
          if (!options.headers.has("X-CSRF-Token")) {
            options.headers.set("X-CSRF-Token", csrf);
          }
          if (!options.headers.has("X-Requested-With")) {
            options.headers.set("X-Requested-With", "XMLHttpRequest");
          }
        } else {
          options.headers = options.headers || {};
          if (!options.headers["X-CSRF-Token"] && !options.headers["x-csrf-token"]) {
            options.headers["X-CSRF-Token"] = csrf;
          }
          if (!options.headers["X-Requested-With"] && !options.headers["x-requested-with"]) {
            options.headers["X-Requested-With"] = "XMLHttpRequest";
          }
        }
      }
    }
    return fetchFn(input, options).then(function (resp) {
      if (isSessionExpired(resp, options)) {
        redirectToLogin();
        return Promise.reject(new Error("session expired"));
      }
      return resp;
    });
  }

  if (typeof window !== "undefined") {
    window.scheduleFetch = scheduleFetch;
  }
  if (typeof module !== "undefined" && module.exports) {
    module.exports = { isSessionExpired: isSessionExpired, scheduleFetch: scheduleFetch, getCsrfToken: getCsrfToken };
  }
})();
