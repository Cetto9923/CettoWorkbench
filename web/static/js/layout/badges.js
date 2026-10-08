// =============================================================================
// 文件: web/static/js/layout/badges.js
// 模块: 公共导航
// 职责: 补取导航真实计数，显示加载状态及失败重试。
// =============================================================================
(function () {
  "use strict";
  var badges = Array.from(document.querySelectorAll("[data-sidebar-badge]"));
  var loading = false;
  async function load() {
    if (loading) return;
    loading = true;
    badges.forEach(function (badge) { badge.textContent = "…"; badge.title = "计数加载中"; });
    try {
      var response = await window.appFetch("/navigation/badges", { headers: { Accept: "application/json" } });
      var result = await response.json();
      if (!response.ok || !result.success || result.data.unavailable) throw new Error("unavailable");
      badges.forEach(function (badge) {
        var key = badge.dataset.sidebarBadge;
        var count = result.data[key];
        badge.textContent = key === "notice" && count > 99 ? "99+" : String(count);
        badge.title = key === "notice" ? count + " 条未读通知" : count + " 条待办事项";
        badge.dataset.badgePending = "false";
        badge.dataset.badgeFailed = "false";
      });
    } catch (error) {
      badges.forEach(function (badge) {
        badge.textContent = "重试";
        badge.title = "计数加载失败，点击重试";
        badge.dataset.badgeFailed = "true";
      });
    } finally { loading = false; }
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
