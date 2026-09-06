// =============================================================================
// 文件: web/static/js/po/demand-detail-render.js
// 模块: PO 工作台
// 职责: 业务需求统一详情五大 Tab、父子导航与去重模板纯渲染函数。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.DemandDetailRender = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  function esc(str) {
    if (str === null || str === undefined) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function renderHeader(summary, mode) {
    var tagClass = mode === "parentAggregate" ? "tag blue" : "tag blue";
    var tagText = mode === "parentAggregate" ? "父业务需求 · 聚合对象" : (mode === "childUnit" ? "子业务需求 · 交付单元" : "独立交付单元");

    return [
      '<div class="dd-idrow">',
      '  <span class="dd-id">' + esc(summary.code) + '</span>',
      '  <span class="dd-tag blue">' + tagText + '</span>',
      '  <span class="dd-tag">' + esc(summary.priority) + '</span>',
      '  <span class="dd-tag green">' + esc(summary.valueStageLabel) + '</span>',
      '  <div class="dd-head-actions">',
      '    <a class="dd-iconbtn" href="/demands/' + esc(summary.demandId) + '" target="_blank" title="在新页面打开">↗</a>',
      '    <button class="dd-iconbtn" onclick="DemandDetail.close()" title="关闭">×</button>',
      '  </div>',
      '</div>',
      '<div class="dd-title">' + esc(summary.title) + '</div>',
      '<div class="dd-meta">',
      '  <span>提出人：' + esc(summary.proposerName) + ' (' + esc(summary.originator) + ')</span>',
      '  <span>提出部门：' + esc(summary.proposerDept) + '</span>',
      '  <span>PO / 负责人：' + esc(summary.ownerName) + '</span>',
      '  <span>最近更新：' + esc(summary.editedDate) + '</span>',
      '</div>',
      '<div class="dd-core-strip">',
      '  <div class="dd-core-item"><div class="k">需求类别</div><div class="v">' + esc(summary.category) + '</div></div>',
      '  <div class="dd-core-item"><div class="k">需求来源</div><div class="v">' + esc(summary.source) + '</div></div>',
      '  <div class="dd-core-item"><div class="k">所属产品</div><div class="v">' + esc(summary.product) + '</div></div>',
      '  <div class="dd-core-item"><div class="k">所属需求池</div><div class="v">' + esc(summary.poolName) + '</div></div>',
      '  <div class="dd-core-item"><div class="k">BSA</div><div class="v">' + esc(summary.bsa) + '</div></div>',
      '</div>',
      '<div class="dd-native-line">',
      '  <b>禅道原生状态：</b>',
      '  <span class="dd-native-state">' + esc(summary.zentaoStatus) + '</span>',
      '  <span style="color:#cbd5e1">•</span>',
      '  <span>工作台按 9 阶段价值流统一流转与指标跟踪</span>',
      '</div>'
    ].join("");
  }

  function renderRelationNav(ctx, currentId) {
    if (!ctx || (!ctx.parent && (!ctx.siblings || ctx.siblings.length === 0))) {
      return "";
    }
    var html = ['<div class="dd-relation-nav">'];

    if (ctx.parent) {
      html.push(
        '<div class="dd-relation-top">',
        '  <button class="dd-parent-link" type="button" onclick="DemandDetail.open(' + ctx.parent.demandId + ')">',
        '    <span class="dd-back-icon">←</span>',
        '    <span><small style="color:#8a99ad;font-size:11px;display:block">父业务需求</small><strong>' + esc(ctx.parent.code) + ' · ' + esc(ctx.parent.title) + '</strong></span>',
        '  </button>',
        '</div>'
      );
    }

    if (ctx.siblings && ctx.siblings.length > 0) {
      html.push('<div class="dd-sibling-row"><span style="font-size:11px;color:#8a99ad;margin-right:4px">同级交付单元：</span>');
      for (var i = 0; i < ctx.siblings.length; i++) {
        var sib = ctx.siblings[i];
        var cls = "dd-sibling-chip";
        if (sib.isCurrent) cls += " active";
        else if (sib.hasRisk) cls += " risk";
        else if (sib.isDone) cls += " done";

        html.push(
          '<button class="' + cls + '" onclick="DemandDetail.open(' + sib.demandId + ')">',
          '  <strong>' + esc(sib.code) + '</strong>',
          '  <span>' + esc(sib.stage) + '</span>',
          '</button>'
        );
      }
      html.push('</div>');
    }

    html.push('</div>');
    return html.join("");
  }

  function renderSpotlight(spotlight) {
    if (!spotlight) return "";
    return [
      '<div class="dd-card dd-spot" id="spotlightSection">',
      '  <div>',
      '    <span class="dd-tag blue">' + esc(spotlight.badge) + '</span>',
      '    <div class="dd-spot-title">' + esc(spotlight.title) + '</div>',
      '    <div class="dd-spot-desc">' + esc(spotlight.desc) + '</div>',
      '  </div>',
      '  <div>',
      '    <button class="dd-btn primary" onclick="DemandDetail.switchTab(\'' + esc(spotlight.targetTab) + '\',\'' + esc(spotlight.targetSection) + '\')">' + esc(spotlight.actionLabel) + '</button>',
      '  </div>',
      '</div>'
    ].join("");
  }

  function renderValueStream(vs) {
    if (!vs || !vs.stages) return "";
    var overdueClass = vs.isOverdue ? "overdue" : "";
    var diffSign = vs.diffCycleDays > 0 ? ("+" + vs.diffCycleDays) : String(vs.diffCycleDays);

    var html = [
      '<div class="dd-card">',
      '  <div class="dd-card-body">',
      '    <div class="dd-cycle-bar">',
      '      <div class="dd-cycle-item">预计交付周期<strong>' + vs.estimatedCycleDays + ' 天</strong></div>',
      '      <div class="dd-cycle-item">当前已用<strong>' + vs.usedCycleDays + ' 天</strong></div>',
      '      <div class="dd-cycle-item">周期目标<strong>' + vs.targetCycleDays + ' 天</strong></div>',
      '      <div class="dd-cycle-item ' + overdueClass + '">周期偏差<strong>' + diffSign + ' 天</strong></div>',
      '    </div>',
      '    <div class="dd-flow">'
    ];

    for (var i = 0; i < vs.stages.length; i++) {
      var s = vs.stages[i];
      html.push(
        '<div class="dd-flow-stage ' + esc(s.status) + '">',
        '  <div class="nm">' + esc(s.label) + '</div>',
        '  <div class="who">' + esc(s.role) + '</div>',
        '  <div class="duration">' + esc(s.durationText) + '</div>',
        '</div>'
      );
    }

    html.push('</div></div></div>');
    return html.join("");
  }

  function renderTabOverview(data) {
    var vsHtml = renderValueStream(data.valueStream);
    var spHtml = renderSpotlight(data.spotlight);
    var summary = data.summary;

    return [
      '<div class="dd-grid">',
      '  <div class="dd-main">',
      spHtml,
      vsHtml,
      '  </div>',
      '  <aside class="dd-sidebar">',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>业务核心属性</h3></div>',
      '      <div class="dd-kv-list">',
      '        <div class="k">提出人</div><div class="v">' + esc(summary.proposerName) + '</div>',
      '        <div class="k">业务部门</div><div class="v">' + esc(summary.proposerDept) + '</div>',
      '        <div class="k">业务负责人</div><div class="v">' + esc(summary.ownerName) + '</div>',
      '        <div class="k">测试负责人</div><div class="v">' + esc(summary.testOwner) + '</div>',
      '        <div class="k">目标上线</div><div class="v">' + esc(summary.estimateLaunch) + '</div>',
      '        <div class="k">创建时间</div><div class="v">' + esc(summary.createdDate) + '</div>',
      '      </div>',
      '    </div></div>',
      '  </aside>',
      '</div>'
    ].join("");
  }

  function renderTabRequirement(req) {
    if (!req) return '<div class="dd-card dd-card-body">暂无需求说明</div>';
    var clarifyRows = "";
    if (req.clarifications && req.clarifications.length > 0) {
      clarifyRows = req.clarifications.map(function (c) {
        return [
          '<tr>',
          '  <td><strong>' + esc(c.productName) + '</strong></td>',
          '  <td>' + esc(c.analyst) + '</td>',
          '  <td>' + esc(c.content) + '</td>',
          '  <td>' + esc(c.devEnd) + '</td>',
          '  <td>' + esc(c.testEnd) + '</td>',
          '</tr>'
        ].join("");
      }).join("");
    }

    var filesRows = "";
    if (req.attachments && req.attachments.length > 0) {
      filesRows = req.attachments.map(function (f) {
        return '<li><a href="' + esc(f.download) + '" target="_blank">' + esc(f.title) + '</a> (' + esc(f.size) + ')</li>';
      }).join("");
    }

    return [
      '<div class="dd-card" id="requirementSection"><div class="dd-card-body">',
      '  <div class="dd-cardhead"><h3>业务需求正文与描述</h3></div>',
      '  <div style="font-size:13px;line-height:1.7;color:#334155;margin-bottom:16px;">' + (req.specHtml || "—") + '</div>',
      '  <div class="dd-cardhead"><h3>验收标准 (Verify Plan)</h3></div>',
      '  <div style="font-size:13px;line-height:1.7;color:#334155;">' + (req.verifyHtml || "—") + '</div>',
      '</div></div>',
      '<div class="dd-card" id="clarificationSection"><div class="dd-card-body">',
      '  <div class="dd-cardhead"><h3>系统/产品维度澄清说明</h3></div>',
      clarifyRows ? (
        '<table class="dd-table"><thead><tr><th>产品/系统</th><th>需求分析师</th><th>澄清要点</th><th>计划开发完成</th><th>计划测试完成</th></tr></thead><tbody>' + clarifyRows + '</tbody></table>'
      ) : '<div style="color:#8a99ad;font-size:12px;">暂无多系统澄清拆解</div>',
      '</div></div>',
      filesRows ? (
        '<div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>需求附件</h3></div><ul style="padding-left:18px;margin:0;font-size:12px;color:#2563eb;">' + filesRows + '</ul></div></div>'
      ) : ""
    ].join("");
  }

  function renderTabExecution(exec) {
    if (!exec) return '<div class="dd-card dd-card-body">暂无研发执行数据</div>';
    var storyRows = "";
    if (exec.stories && exec.stories.length > 0) {
      storyRows = exec.stories.map(function (s) {
        return [
          '<tr>',
          '  <td><strong>' + esc(s.code) + '</strong></td>',
          '  <td>' + esc(s.title) + '</td>',
          '  <td>' + esc(s.product) + '</td>',
          '  <td>' + esc(s.owner) + '</td>',
          '  <td>' + esc(s.status) + '</td>',
          '  <td>' + s.tasksDone + ' / ' + s.tasksTotal + '</td>',
          '</tr>'
        ].join("");
      }).join("");
    }

    var tc = exec.testCaseSummary || {};
    var bg = exec.bugSummary || {};
    var qs = exec.qualitySummary || {};

    return [
      '<div class="dd-card" id="storiesSection"><div class="dd-card-body">',
      '  <div class="dd-cardhead"><h3>已分发研发需求 (Stories)</h3></div>',
      storyRows ? (
        '<table class="dd-table"><thead><tr><th>编号</th><th>标题</th><th>所属产品</th><th>负责人</th><th>状态</th><th>任务完成</th></tr></thead><tbody>' + storyRows + '</tbody></table>'
      ) : '<div style="color:#8a99ad;font-size:12px;">暂未分发研发需求</div>',
      '</div></div>',
      '<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:14px;">',
      '  <div class="dd-card"><div class="dd-card-body">',
      '    <div class="dd-cardhead"><h3>测试用例执行</h3></div>',
      '    <div style="font-size:20px;font-weight:700;color:#1e293b;margin:8px 0;">' + (tc.executedCount || 0) + ' / ' + (tc.totalCount || 0) + '</div>',
      '    <div style="font-size:12px;color:#64748b;">执行率 <strong>' + (tc.executionRate || 0) + '%</strong> · 通过率 <strong>' + (tc.passRate || 0) + '%</strong></div>',
      '  </div></div>',
      '  <div class="dd-card" id="bugSection"><div class="dd-card-body">',
      '    <div class="dd-cardhead"><h3>缺陷情况</h3></div>',
      '    <div style="font-size:20px;font-weight:700;color:#1e293b;margin:8px 0;">' + (bg.activeCount || 0) + ' <small style="font-size:12px;font-weight:400;color:#8a99ad">未解决</small></div>',
      '    <div style="font-size:12px;color:#64748b;">影响交付：<strong style="color:' + (bg.deliveryBlocking > 0 ? "#d32f2f" : "#1e293b") + '">' + (bg.deliveryBlocking || 0) + '</strong> · 已解决：' + (bg.resolvedCount || 0) + '</div>',
      '  </div></div>',
      '  <div class="dd-card"><div class="dd-card-body">',
      '    <div class="dd-cardhead"><h3>代码质量得分</h3></div>',
      '    <div style="font-size:20px;font-weight:700;color:#2563eb;margin:8px 0;">' + (qs.avgScore || 0) + '</div>',
      '    <div style="font-size:12px;color:#64748b;">关联分支 ' + (qs.branchesCount || 0) + ' · 门禁通过 ' + (qs.passedGates || 0) + ' / ' + (qs.totalGates || 0) + '</div>',
      '  </div></div>',
      '</div>'
    ].join("");
  }

  function renderTabDelivery(delivery) {
    if (!delivery) return '<div class="dd-card dd-card-body">暂无交付数据</div>';
    function pill(ok) {
      return ok ? '<span class="dd-tag green">已就绪</span>' : '<span class="dd-tag">进行中</span>';
    }

    return [
      '<div class="dd-card" id="deliverySection"><div class="dd-card-body">',
      '  <div class="dd-cardhead"><h3>交付就绪度评估 (Readiness)</h3></div>',
      '  <div style="display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:16px;">',
      '    <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd">',
      '      <div style="font-size:11px;color:#8a99ad;margin-bottom:4px">研发完成</div>' + pill(delivery.devDone),
      '    </div>',
      '    <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd">',
      '      <div style="font-size:11px;color:#8a99ad;margin-bottom:4px">测试通过</div>' + pill(delivery.testPassed),
      '    </div>',
      '    <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd">',
      '      <div style="font-size:11px;color:#8a99ad;margin-bottom:4px">严重缺陷闭环</div>' + pill(delivery.bugsResolved),
      '    </div>',
      '    <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd">',
      '      <div style="font-size:11px;color:#8a99ad;margin-bottom:4px">业务验收</div>' + pill(delivery.acceptanceDone),
      '    </div>',
      '  </div>',
      '  <div class="dd-cardhead"><h3>发布与投产信息</h3></div>',
      '  <div class="dd-kv-list" style="grid-template-columns:120px 1fr;">',
      '    <div class="k">目标上线时间</div><div class="v">' + esc(delivery.estimateLaunch) + '</div>',
      '    <div class="k">发布规划窗口</div><div class="v">' + esc(delivery.publishWindow) + '</div>',
      '    <div class="k">生产验证结论</div><div class="v">' + esc(delivery.verifyConclusion) + '</div>',
      '  </div>',
      '</div></div>'
    ].join("");
  }

  function renderTabHistory(history) {
    if (!history) return '<div class="dd-card dd-card-body">暂无过程记录</div>';
    var actionRows = "";
    if (history.actions && history.actions.length > 0) {
      actionRows = history.actions.map(function (a) {
        return [
          '<tr>',
          '  <td style="white-space:nowrap;">' + esc(a.date) + '</td>',
          '  <td><strong>' + esc(a.actor) + '</strong></td>',
          '  <td>' + esc(a.action) + '</td>',
          '  <td>' + esc(a.extra) + '</td>',
          '</tr>'
        ].join("");
      }).join("");
    }

    var lc = history.lifecycle || {};

    return [
      '<div class="dd-grid">',
      '  <div class="dd-main">',
      '    <div class="dd-card" id="historySection"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>禅道操作审计日志</h3></div>',
      actionRows ? (
        '<table class="dd-table"><thead><tr><th>时间</th><th>操作人</th><th>动作</th><th>说明</th></tr></thead><tbody>' + actionRows + '</tbody></table>'
      ) : '<div style="color:#8a99ad;font-size:12px;">暂无历史动作</div>',
      '    </div></div>',
      '  </div>',
      '  <aside class="dd-sidebar">',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>生命周期经办人</h3></div>',
      '      <div class="dd-kv-list">',
      '        <div class="k">创建人</div><div class="v">' + esc(lc.createdBy) + '</div>',
      '        <div class="k">创建时间</div><div class="v">' + esc(lc.createdDate) + '</div>',
      '        <div class="k">评审人</div><div class="v">' + esc(lc.reviewer) + '</div>',
      '        <div class="k">评审时间</div><div class="v">' + esc(lc.reviewedDate) + '</div>',
      '        <div class="k">最后编辑</div><div class="v">' + esc(lc.lastEditedBy) + '</div>',
      '        <div class="k">最后更新</div><div class="v">' + esc(lc.lastEditedDate) + '</div>',
      '        <div class="k">关闭人</div><div class="v">' + esc(lc.closedBy) + '</div>',
      '        <div class="k">关闭原因</div><div class="v">' + esc(lc.closedReason) + '</div>',
      '      </div>',
      '    </div></div>',
      '  </aside>',
      '</div>'
    ].join("");
  }

  function renderParentAggregate(pa) {
    if (!pa) return '<div class="dd-card dd-card-body">暂无子需求数据</div>';
    var attentionHtml = "";
    if (pa.attentionItems && pa.attentionItems.length > 0) {
      attentionHtml = [
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead"><h3>当前需要关注的子需求</h3></div>',
        pa.attentionItems.map(function (att) {
          return [
            '<div style="padding:10px 12px;border:1px solid ' + (att.isRisk ? "#f1b7b7" : "#e2e8f0") + ';background:' + (att.isRisk ? "#fff6f6" : "#f8fafc") + ';border-radius:6px;margin-bottom:8px;display:flex;align-items:center;justify-content:space-between;">',
            '  <div><strong>' + esc(att.code) + ' · ' + esc(att.title) + '</strong><div style="font-size:12px;color:#64748b;margin-top:2px;">' + esc(att.riskDesc) + '</div></div>',
            '  <button class="dd-btn" onclick="DemandDetail.open(' + att.demandId + ')">查看 →</button>',
            '</div>'
          ].join("");
        }).join(""),
        '</div></div>'
      ].join("");
    }

    var unitsHtml = "";
    if (pa.deliveryUnits && pa.deliveryUnits.length > 0) {
      unitsHtml = [
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead"><h3>交付单元列表 (' + pa.deliveryUnits.length + ')</h3></div>',
        '  <table class="dd-table"><thead><tr><th>编号</th><th>子需求名称</th><th>阶段</th><th>负责人</th><th>研发需求</th><th>任务</th><th>计划上线</th><th>操作</th></tr></thead><tbody>',
        pa.deliveryUnits.map(function (u) {
          return [
            '<tr>',
            '  <td><strong>' + esc(u.code) + '</strong></td>',
            '  <td>' + esc(u.title) + '</td>',
            '  <td>' + esc(u.stage) + '</td>',
            '  <td>' + esc(u.owner) + '</td>',
            '  <td>' + u.storiesNum + '</td>',
            '  <td>' + u.tasksNum + '</td>',
            '  <td>' + esc(u.launchDate) + '</td>',
            '  <td><button class="dd-btn" onclick="DemandDetail.open(' + u.demandId + ')">详情</button></td>',
            '</tr>'
          ].join("");
        }).join(""),
        '  </tbody></table>',
        '</div></div>'
      ].join("");
    }

    return [
      '<div class="dd-card dd-spot">',
      '  <div>',
      '    <span class="dd-tag blue">父需求聚合汇总</span>',
      '    <div class="dd-spot-title">该需求已拆分为 ' + pa.unitTotal + ' 个独立交付单元（已上线 ' + pa.unitOnline + ' / ' + pa.unitTotal + '）</div>',
      '    <div class="dd-spot-desc">父需求自身不再承接具体澄清与提测，状态由各子交付单元独立流转汇聚。</div>',
      '  </div>',
      '</div>',
      attentionHtml,
      unitsHtml
    ].join("");
  }

  return {
    esc: esc,
    renderHeader: renderHeader,
    renderRelationNav: renderRelationNav,
    renderTabOverview: renderTabOverview,
    renderTabRequirement: renderTabRequirement,
    renderTabExecution: renderTabExecution,
    renderTabDelivery: renderTabDelivery,
    renderTabHistory: renderTabHistory,
    renderParentAggregate: renderParentAggregate
  };
});
