/* =============================================================================
   文件: web/static/js/po/schedule-link.js
   模块: PO 工作台 - 排期入口 URL 与排期阶段动作按钮助手
   职责: 根据需求 / 研发需求对象 ID 拼排期工作台入口 URL；路由与
         internal/module/schedule/handler.go 中已注册的 GET 路由对齐：
           /schedule/demands/:id/scheduling   (业务需求)
           /schedule/stories/:id/scheduling   (独立研发需求)
         同时提供 scheduleStageAction(...) 把排期阶段的 stage-action 渲染成
         可直接导航的 <a> 按钮；其它动作仍由 workboard.js 内部渲染为 button。
         不允许硬编码路径或绕过已注册路由。
   挂载: window.ScheduleLink.{url, scheduleStoryAction, scheduleDemandAction, esc}
   ============================================================================= */

(function () {
  "use strict";

  var esc = window.escapeHtml;

  // 业务需求 / 独立研发需求 各一条已注册的 GET 入口。
  function url(object) {
    if (!object) { return ""; }
    var id = Number(object.id || 0);
    if (!id) { return ""; }
    var useStory = object.kind === "story" && (object.independent || !object.parentDemandID);
    return (useStory ? "/schedule/stories/" : "/schedule/demands/") + id + "/scheduling";
  }

  // 通用 toast 按钮：保留原有"label：displayId"提示行为。
  function toastButton(label, displayId) {
    var dl = esc(displayId || "");
    var txt = esc(label);
    return '<button type="button" class="stage-action" data-toast="' + esc(label + "：" + dl) + '">' + txt + "</button>";
  }

  // 研发需求排期阶段：直链 /schedule/stories/:id/scheduling。
  function scheduleStoryAction(object, label) {
    var u = url(object);
    if (u) {
      return '<a class="stage-action" href="' + esc(u) + '" data-track-schedule="1" data-track-label="' + esc(object.displayId || "") + '">' + esc(label) + "</a>";
    }
    return toastButton(label, object.displayId);
  }

  // 业务 / 子需求排期阶段：直链 /schedule/demands/:id/scheduling。
  function scheduleDemandAction(object, label) {
    var u = url(object);
    if (u) {
      return '<a class="stage-action" href="' + esc(u) + '" data-track-schedule="1" data-track-label="' + esc(object.displayId || "") + '">' + esc(label) + "</a>";
    }
    return toastButton(label, object.displayId);
  }

  // 研发需求 / 任务抽屉打开按钮。
  function openTasksButton(object, label) {
    return '<button type="button" class="stage-action" data-open-tasks="' + object.id + '" data-rd-label="' + esc((object.displayId || "") + " " + (object.title || "")) + '" data-story-url="' + esc(object.storyUrl || "") + '">' + esc(label) + "</button>";
  }

  window.ScheduleLink = {
    url: url,
    scheduleStoryAction: scheduleStoryAction,
    scheduleDemandAction: scheduleDemandAction,
    toastButton: toastButton,
    openTasksButton: openTasksButton,
    esc: esc
  };
})();