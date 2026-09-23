(function () {
  "use strict";

  function redirectToLogin() {
    var redirect = window.location.pathname + window.location.search;
    window.location.href = "/login?redirect=" + encodeURIComponent(redirect);
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
        // 与 appFetch 一致：跳转中挂起，不进各模块 catch
        return new Promise(function () {});
      }
      return resp;
    });
  }

  window.scheduleFetch = scheduleFetch;
})();
