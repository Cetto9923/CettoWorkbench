/* =============================================================================
 * 文件: web/static/workbench/po/profile/data/profile-adapter.js
 * 模块: PO 工作台 / 个人资料（/profile）—— API 适配层
 * 职责: 对 /profile/data、PUT /profile、PUT /profile/password 三个端点的
 *       统一 fetch + CSRF + JSON 适配；当前由 po-profile.js 直接调用。
 *       后续若其它视图需要共用（例如顶栏编辑资料入口），可复用本模块。
 * 端点契约（与 B agent 后端握手）：
 *   GET  /profile/data
 *     200 {success: true, data: {account, displayName, email, mobile,
 *          gender, deptId, deptName, agileGroups:[{id,name}], mainTeamId}}
 *   PUT  /profile  body: {displayName, email, mobile, gender, mainTeamId}
 *     200 {success: true, message, data?}
 *     422 {success: false, errors:[{field,message}]}
 *   PUT  /profile/password
 *     body: {oldPassword, newPassword, confirmPassword}
 *     200 {success: true, message}
 *     400 {success: false, message}    当前密码错误等
 *     422 {success: false, errors:[{field,message}]}
 * ============================================================================= */

(function () {
  "use strict";

  function getFetch() {
    return (typeof window.appFetch === "function") ? window.appFetch : window.fetch;
  }

  function request(method, url, body) {
    var init = {
      method: method,
      headers: { "Accept": "application/json" }
    };
    if (body != null) {
      init.headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }
    return getFetch()(url, init).then(function (res) {
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
      return res.json().catch(function () {
        throw new Error("数据格式解析失败");
      });
    });
  }

  function fetchProfile() {
    return request("GET", "/profile/data");
  }

  function updateProfile(payload) {
    return request("PUT", "/profile", payload);
  }

  function updatePassword(payload) {
    return request("PUT", "/profile/password", payload);
  }

  var ProfileAdapter = {
    fetchProfile: fetchProfile,
    updateProfile: updateProfile,
    updatePassword: updatePassword
  };

  if (typeof window !== "undefined") {
    window.ProfileAdapter = ProfileAdapter;
  }
})();
