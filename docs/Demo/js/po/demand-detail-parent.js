// =============================================================================
// 文件: web/static/js/po/demand-detail-parent.js
// 模块: PO 工作台
// 职责: 父业务需求聚合视图渲染（renderParentAggregate）；与 detail 其他
//       Tab 拆分以保持主渲染器在 500 行硬性上限内。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define(["./demand-detail-richtext"], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(require("./demand-detail-richtext.js"));
  } else {
    root.DemandDetailParent = factory(root.DemandDetailRichText);
  }
})(typeof self !== "undefined" ? self : this, function (RichText) {
  "use strict";

  var esc = (RichText && RichText.esc) || function (s) { return String(s == null ? "" : s); };

  function renderParentAggregate(pa) {
    if (!pa) return '<div class="dd-card dd-card-body">暂无子需求数据</div>';
    var attentionHtml = "";
    if (pa.attentionItems && pa.attentionItems.length > 0) {
      attentionHtml = '<div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>当前需要关注的子需求</h3></div>' +
        pa.attentionItems.map(function (att) {
          return '<div style="padding:10px 12px;border:1px solid ' + (att.isRisk ? "#f1b7b7" : "#e2e8f0") + ';background:' + (att.isRisk ? "#fff6f6" : "#f8fafc") + ';border-radius:6px;margin-bottom:8px;display:flex;align-items:center;justify-content:space-between;">' +
            '<div><strong>' + esc(att.code) + ' · ' + esc(att.title) + '</strong><div style="font-size:12px;color:#64748b;margin-top:2px;">' + esc(att.riskDesc) + '</div></div>' +
            '<button class="dd-btn" onclick="DemandDetail.open(' + att.demandId + ')">查看 →</button></div>';
        }).join("") + '</div></div>';
    }
    var unitsHtml = "";
    if (pa.deliveryUnits && pa.deliveryUnits.length > 0) {
      unitsHtml = '<div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>交付单元列表 (' + pa.deliveryUnits.length + ')</h3></div>' +
        '<table class="dd-table"><thead><tr><th>编号</th><th>子需求名称</th><th>阶段</th><th>负责人</th><th>研发需求</th><th>任务</th><th>计划上线</th><th>操作</th></tr></thead><tbody>' +
        pa.deliveryUnits.map(function (u) {
          return '<tr><td><strong>' + esc(u.code) + '</strong></td><td>' + esc(u.title) + '</td><td>' + esc(u.stage) + '</td><td>' + esc(u.owner) + '</td><td>' + u.storiesNum + '</td><td>' + u.tasksNum + '</td><td>' + esc(u.launchDate) + '</td><td><button class="dd-btn" onclick="DemandDetail.open(' + u.demandId + ')">详情</button></td></tr>';
        }).join("") + '</tbody></table></div></div>';
    }
    return '<div class="dd-card dd-spot"><div><span class="dd-tag blue">父需求聚合汇总</span>' +
      '<div class="dd-spot-title">该需求已拆分为 ' + pa.unitTotal + ' 个独立交付单元（已上线 ' + pa.unitOnline + ' / ' + pa.unitTotal + '）</div>' +
      '<div class="dd-spot-desc">父需求自身不再承接具体澄清与提测，状态由各子交付单元独立流转汇聚。</div>' +
      '</div></div>' + attentionHtml + unitsHtml;
  }

  return { renderParentAggregate: renderParentAggregate };
});