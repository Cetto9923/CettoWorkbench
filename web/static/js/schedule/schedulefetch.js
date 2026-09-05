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

  if (typeof window !== "undefined") {
    window.scheduleFetch = scheduleFetch;
  }
  if (typeof module !== "undefined" && module.exports) {
    module.exports = { isSessionExpired: isSessionExpired, scheduleFetch: scheduleFetch };
  }
})();
