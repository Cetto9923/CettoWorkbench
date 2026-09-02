// =============================================================================
// 文件: web/static/js/components/search.js
// 模块: 搜索组件
// 类型: 页面脚本
// 职责: 控制搜索组件展开与收起
// =============================================================================

(function () {
  var panels = document.querySelectorAll(".js-search-panel");
  if (!panels.length) return;

  panels.forEach(function (panel) {
    var toggleBtn = panel.querySelector("[data-search-toggle]");
    if (!toggleBtn) return;

    var textEl = toggleBtn.querySelector(".search-panel-toggle-text");
    var iconEl = toggleBtn.querySelector(".search-panel-toggle-icon");

    function setCollapsed(collapsed) {
      panel.classList.toggle("collapsed", collapsed);
      toggleBtn.setAttribute("aria-expanded", collapsed ? "false" : "true");
      toggleBtn.title = collapsed ? "展开检索条件" : "收起检索条件";
      if (textEl) {
        textEl.textContent = collapsed ? "展开" : "收起";
      }
      if (iconEl) {
        iconEl.classList.toggle("bi-chevron-down", collapsed);
        iconEl.classList.toggle("bi-chevron-up", !collapsed);
      }
    }

    toggleBtn.addEventListener("click", function () {
      setCollapsed(!panel.classList.contains("collapsed"));
    });

    setCollapsed(panel.classList.contains("collapsed"));
  });
})();