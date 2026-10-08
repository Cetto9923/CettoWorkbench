// =============================================================================
// 文件: web/static/js/schedule/scheduleintegratedwindow.js
// 模块: 排期工作台
// 职责: 排期弹窗「版本窗口」下拉的占位文案判断，供 scheduleintegrated.js 渲染空态
// 依赖: 无（纯函数，挂载到 window.ScheduleIntegratedWindow）
// =============================================================================

(function () {
  "use strict";

  // 接口只返回未过期窗口；全部过期时下拉无任何可选项，需与「待选择」区分提示去新建。
  function placeholderText(windows) {
    return Array.isArray(windows) && windows.length
      ? "请选择版本窗口"
      : "暂无可用版本窗口，请先新建窗口";
  }

  window.ScheduleIntegratedWindow = {
    placeholderText: placeholderText,
  };
})();