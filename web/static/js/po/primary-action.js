/* =============================================================================
   文件: web/static/js/po/primary-action.js
   模块: PO 个人工作台 - 主操作按钮渲染
   职责: 为每行需求 / 研发需求 / 测试单派生"主要阶段动作"。动作合同由服务端
         primaryAction 字段提供（key/label/kind/url/enabled/reason）；后端未
         落地时按 PLAN §4 显示 "—" 占位，不使用"查看详情"兜底。
   依赖: PersonalList.escapeHtml（来自 personal-list.js）
   挂载: window.PrimaryAction.primaryActionHtml(item, isStory)
   ============================================================================= */

(function () {
  "use strict";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { return String(v == null ? "" : v); };

  // 排期入口 URL：与 schedule handler 已注册的路由对齐（不允许硬编码）。
  function scheduleUrl(item, isStory) {
    var id = (item && item.id) || "";
    if (!id) { return ""; }
    var cleanId = String(id).replace(/^US/i, "").replace(/^U/i, "");
    if (isStory) { return "/schedule/stories/" + encodeURIComponent(cleanId) + "/scheduling"; }
    return "/schedule/demands/" + encodeURIComponent(cleanId) + "/scheduling";
  }

  function renderEnabled(item, pa, isStory) {
    // approve 的业务动作名称是“评审”；服务端旧缓存或旧接口响应不能把展示文案降回“受理”。
    var label = String(pa.key || "") === "approve" ? "评审" : String(pa.label || "").trim();
    if (!label) { return '<span class="home-unavailable">—</span>'; }
    var kind = String(pa.kind || "").toLowerCase();
    var url = String(pa.url || "").trim();
    if (kind === "external" && url) {
      return '<a class="table-action-btn primary" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer">' + esc(label) + ' ↗</a>';
    }
    if (kind === "schedule") {
      var schedUrl = url || scheduleUrl(item, isStory);
      return '<button type="button" class="table-action-btn primary js-home-schedule-action" data-demand-id="' + esc(isStory ? '' : String(item && item.id || '').replace(/^US/i, '')) + '" data-story-id="' + esc(isStory ? String(item && item.id || '').replace(/^U/i, '') : '') + '" data-schedule-url="' + esc(schedUrl) + '">' + esc(label) + '</button>';
    }
    if (kind === "drawer" || kind === "internal") {
      var did = String(pa.demandId || item.id || "").replace(/^US/i, "");
      return '<button type="button" class="table-action-btn primary js-po-drawer-action" data-action-key="' + esc(pa.key) + '" data-action-url="' + esc(url) + '" data-demand-id="' + esc(did) + '">' + esc(label) + '</button>';
    }
    if (url) {
      if (/^https?:\/\//i.test(url)) {
        return '<a class="table-action-btn primary" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer">' + esc(label) + ' ↗</a>';
      }
      return '<a class="table-action-btn primary" href="' + esc(url) + '">' + esc(label) + '</a>';
    }
    return '<span class="home-unavailable">' + esc(label) + '</span>';
  }

  function primaryActionHtml(item, isStory, availableKeys) {
    var pa = item && item.primaryAction;
    if (pa && typeof pa === "object") {
      // 页面可进一步禁用未接线的交互，不能提升服务端权限。
      if (pa.enabled !== false && pa.kind !== "external" && availableKeys && availableKeys.indexOf(pa.key) < 0) {
        pa = Object.assign({}, pa, { enabled: false, reason: "当前待办页尚未接入此操作，请前往工作台办理" });
      }
      if (pa.enabled === false) {
        var reason = String(pa.reason || "无办理权限").trim();
        var label = String(pa.label || "").trim() || "待审批人处理";
        return '<span class="home-unavailable" title="' + esc(reason) + '">' + esc(label) + '</span>';
      }
      return renderEnabled(item, pa, isStory);
    }
    // 占位：等待 Stage 5 primaryAction 合同落地；不显示"查看详情"伪动作。
    return '<span class="home-unavailable" title="等待服务端动作合同落地">—</span>';
  }

  window.PrimaryAction = {
    primaryActionHtml: primaryActionHtml,
    scheduleUrl: scheduleUrl
  };
})();
