// =============================================================================
// 文件: web/static/js/po/demand-detail-flow.js
// 模块: PO 工作台
// 职责: 需求详情 B3 流程与审批、管理信息（检查项、实际时间）、价值模型渲染及禅道跳转组件
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.DemandDetailFlow = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  var esc = window.escapeHtml || function (s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  };

  /**
   * F3: 全站统一「在禅道办理 ↗」按钮组件渲染函数
   * @param {string} url 禅道详情/操作 URL
   * @param {string} [label] 按钮文字，默认「在禅道办理 ↗」
   * @returns {string} HTML 字符串
   */
  function renderZentaoActionBtn(url, label) {
    var text = label || "在禅道办理 ↗";
    if (!url) return "";
    return '<a href="' + esc(url) + '" target="_blank" rel="noopener noreferrer" class="zt-link-btn">' + esc(text) + '</a>';
  }

  return {
    esc: esc,
    renderZentaoActionBtn: renderZentaoActionBtn
  };
});
