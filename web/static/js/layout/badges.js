// =============================================================================
// 文件: web/static/js/layout/badges.js
// 模块: 公共导航
// 职责: 补取导航真实计数，显示加载状态及失败重试。
//       角标只是装饰性计数，静默处理 401/超时/5xx，绝不触发全站跳登录。
// =============================================================================
(function () {
  "use strict";
  var BADGES_URL = "/navigation/badges";
  var BADGES_TIMEOUT_MS = 8000;
  var badges = Array.from(document.querySelectorAll("[data-sidebar-badge]"));
  var loading = false;

  // 角标失败一律收敛到同一个「重试」态，避免 401 时 response.json() 抛错
  // 与业务不可用共用两套文案。
  function markUnavailable() {
    badges.forEach(function (badge) {
      badge.textContent = "重试";
      badge.title = "计数加载失败，点击重试";
      badge.dataset.badgePending = "false";
      badge.dataset.badgeFailed = "true";
    });
  }

  // 超时兜底：后台轮询不能无限期悬挂，否则角标永远停在「…」且 loading 卡死。
  function withTimeout() {
    var controller = new AbortController();
    var timer = window.setTimeout(function () { controller.abort(); }, BADGES_TIMEOUT_MS);
    return { signal: controller.signal, clear: function () { window.clearTimeout(timer); } };
  }

  async function load() {
    if (loading) return;
    loading = true;
    badges.forEach(function (badge) { badge.textContent = "…"; badge.title = "计数加载中"; });
    var timeout = withTimeout();
    try {
      // quietUnauthorized：401 交由本模块降级，不跳登录页。
      var response = await window.appFetch(BADGES_URL, {
        headers: { Accept: "application/json" },
        quietUnauthorized: true,
        signal: timeout.signal
      });
      if (response.status === 401) { markUnavailable(); return; }
      if (!response.ok) throw new Error("badges http " + response.status);
      var result = await response.json();
      if (!result.success || !result.data || result.data.unavailable) throw new Error("unavailable");
      badges.forEach(function (badge) {
        var key = badge.dataset.sidebarBadge;
        var count = result.data[key];
        badge.textContent = key === "notice" && count > 99 ? "99+" : String(count);
        badge.title = key === "notice" ? count + " 条未读通知" : count + " 条待办事项";
        badge.dataset.badgePending = "false";
        badge.dataset.badgeFailed = "false";
      });
    } catch (error) {
      markUnavailable();
    } finally {
      timeout.clear();
      loading = false;
    }
  }

  badges.forEach(function (badge) {
    badge.addEventListener("click", function (event) {
      if (badge.dataset.badgeFailed !== "true") return;
      event.preventDefault();
      event.stopPropagation();
      load();
    });
  });
  if (badges.some(function (badge) { return badge.dataset.badgePending === "true"; })) load();
})();
