// =============================================================================
// 文件: web/static/js/po/demand-detail-render-execution.js
// 模块: PO 工作台
// 职责: 详情「研发执行」Tab 与代码质量树纯渲染。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define(["./demand-detail-richtext"], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(require("./demand-detail-richtext.js"));
  } else {
    root.DemandDetailRenderExecution = factory(root.DemandDetailRichText);
  }
})(typeof self !== "undefined" ? self : this, function (RichText) {
  "use strict";

  var esc = window.escapeHtml;

  function renderQualityTree(tree) {
    if (!tree || tree.length === 0) {
      return '<div style="color:#8a99ad;font-size:12px;padding:12px 0;">暂无关联代码分支或静态扫描记录</div>';
    }
    return tree.map(function (app) {
      var gateClass = app.gatePassed ? "green" : "red";
      var gateText = app.gatePassed ? "✅ MR 门禁通过" : "❌ MR 门禁阻断";
      var branchRows = "";
      if (app.branches && app.branches.length > 0) {
        branchRows = app.branches.map(function (b) {
          var bGate = b.gateStatus === "pass" ? '<span class="dd-tag green">通过</span>' : '<span class="dd-tag red">阻断</span>';
          return [
            '<tr>',
            '  <td><div><strong>' + esc(b.storyCode) + '</strong></div><div style="color:#64748b;font-size:11px;margin-top:2px;">' + esc(b.storyTitle) + '</div></td>',
            '  <td class="nowrap"><code style="background:#f1f5f9;padding:2px 6px;border-radius:4px;color:#1e293b;font-size:11px;">' + esc(b.branchName) + '</code></td>',
            '  <td class="nowrap"><span class="dd-tag blue">' + esc(b.latestMr) + '</span><div style="color:#8a99ad;font-size:10px;margin-top:2px;">' + esc(b.scanTime) + '</div></td>',
            '  <td class="nowrap" style="text-align:center;">' + bGate + '</td>',
            '  <td class="nowrap">',
            '    <div style="display:grid;grid-template-columns:repeat(2,auto);gap:4px;width:max-content;">',
            '      <span class="dd-metric-pill">得分 <strong>' + b.score + '</strong></span>',
            '      <span class="dd-metric-pill">覆盖率 <strong>' + b.coverage + '%</strong></span>',
            '      <span class="dd-metric-pill">异味 <strong>' + b.codeSmells + '</strong></span>',
            '      <span class="dd-metric-pill">漏洞 <strong>' + b.bugsCount + '</strong></span>',
            '    </div>',
            '  </td>',
            '  <td class="nowrap">' + esc(b.committer) + '</td>',
            '</tr>'
          ].join("");
        }).join("");
      }
      return [
        '<div class="dd-tree-app">',
        '  <div class="dd-tree-head">',
        '    <div class="dd-tree-app-title">',
        '      <span>💻 ' + esc(app.appName) + '</span>',
        '      <span class="dd-tag">' + esc(app.appCode) + '</span>',
        '    </div>',
        '    <div class="dd-tree-app-badges">',
        '      <span class="dd-tag ' + gateClass + '">' + gateText + '</span>',
        '      <span style="font-size:12px;color:#64748b;">' + app.branchCount + ' 条分支</span>',
        '    </div>',
        '  </div>',
        branchRows ? (
          '<table class="dd-table"><thead><tr><th>关联研发需求</th><th>代码分支</th><th>最新 MR 扫描</th><th style="text-align:center;">门禁</th><th>代码扫描指标</th><th>提交人</th></tr></thead><tbody>' + branchRows + '</tbody></table>'
        ) : '<div style="padding:10px 14px;color:#8a99ad;font-size:12px;">该应用下暂无扫描分支</div>',
        '</div>'
      ].join("");
    }).join("");
  }

  function renderTabExecution(exec) {
    if (!exec) return '<div class="dd-card dd-card-body">暂无研发执行数据</div>';
    var tc = exec.testCaseSummary || {};
    var bg = exec.bugSummary || {};
    var to = exec.testOrderSummary || {};
    var leads = exec.leadsMatrix || {};
    var qo = exec.qualityOverview || {};
    var qTree = exec.appQualityTree || [];
    var qualityAvailable = qo.available === true;

    var appsCount = qualityAvailable ? (qo.appsCount || (qTree ? qTree.length : 0)) : 0;
    var passedApps = qualityAvailable && qo.passedGates !== undefined ? qo.passedGates : 0;
    var isAllPass = qualityAvailable && appsCount > 0 && passedApps === appsCount;
    var mainColor = !qualityAvailable ? "#64748b" : (isAllPass ? "#059669" : "#2563eb");
    var mainTitle = !qualityAvailable
      ? "代码质量未接入"
      : (appsCount > 0 ? (passedApps + " / " + appsCount + " 系统达标") : "未涉及系统");
    var branchCount = qualityAvailable ? (qo.branchesCount || (exec.stories ? exec.stories.length : 0)) : 0;

    var storyRows = "";
    if (exec.stories && exec.stories.length > 0) {
      storyRows = exec.stories.map(function (s) {
        var pct = s.tasksTotal > 0 ? Math.round((s.tasksDone / s.tasksTotal) * 100) : 0;
        var bugCol = s.bugsTotal > 0
          ? (s.bugsActive > 0 ? '<span class="dd-tag red">' + s.bugsActive + ' 未闭环</span>' : '<span class="dd-tag green">已闭环(' + s.bugsTotal + ')</span>')
          : '<span style="color:#94a3b8;font-size:11px;">无缺陷</span>';
        return [
          '<tr>',
          '  <td class="nowrap"><strong>' + esc(s.code) + '</strong></td>',
          '  <td>' + esc(s.title) + '</td>',
          '  <td class="nowrap">' + esc(s.product) + '</td>',
          '  <td class="nowrap">' + esc(s.owner) + '</td>',
          '  <td class="nowrap"><span class="dd-tag">' + esc(s.status) + '</span></td>',
          '  <td class="nowrap">' + bugCol + '</td>',
          '  <td class="nowrap">',
          '    <div style="display:flex;align-items:center;gap:8px;">',
          '      <div class="dd-progress-bar" style="width:70px;margin-top:0;"><div class="dd-progress-fill" style="width:' + pct + '%;background:#2563eb;"></div></div>',
          '      <span style="font-size:11px;color:#64748b;">' + s.tasksDone + '/' + s.tasksTotal + ' (' + pct + '%)</span>',
          '    </div>',
          '  </td>',
          '</tr>'
        ].join("");
      }).join("");
    }

    var orderRows = "";
    if (exec.testOrders && exec.testOrders.length > 0) {
      orderRows = exec.testOrders.map(function (t) {
        var tagClass = t.status === "done" ? "green" : (t.status === "doing" ? "blue" : (t.status === "blocked" ? "red" : ""));
        return [
          '<tr>',
          '  <td class="nowrap"><strong>' + esc(t.code) + '</strong></td>',
          '  <td>' + esc(t.title) + '</td>',
          '  <td class="nowrap"><span class="dd-tag">' + esc(t.stage) + '</span></td>',
          '  <td class="nowrap"><span class="dd-tag ' + tagClass + '">' + esc(t.statusLabel) + '</span></td>',
          '  <td class="nowrap">' + esc(t.owner) + '</td>',
          '  <td class="nowrap">' + esc(t.beginDate) + ' ~ ' + esc(t.endDate) + '</td>',
          '  <td class="nowrap"><a href="' + esc(t.ztUrl || '#') + '" target="_blank" rel="noopener noreferrer" style="color:#2563eb;text-decoration:none;">在禅道打开 ↗</a></td>',
          '</tr>'
        ].join("");
      }).join("");
    }

    var blockingBadge = bg.deliveryBlocking > 0
      ? '<span class="dd-tag red" style="font-size:11px;">⚠️ ' + bg.deliveryBlocking + ' 阻塞交付</span>'
      : '<span class="dd-tag green" style="font-size:11px;">无阻塞</span>';

    var leadsHtml = (leads.devLeads && leads.devLeads.length > 0) ? leads.devLeads.join(" / ") : "—";

    return [
      '<div class="dd-grid">',
      '  <div class="dd-main">',
      '    <div class="dd-kpi-grid-4">',
      '      <div class="dd-kpi-card">',
      '        <div class="dd-cardhead"><h3>测试单</h3></div>',
      '        <div class="dd-kpi-main">' + (to.totalCount || 0) + ' <small style="font-size:13px;font-weight:400;color:#8a99ad">个</small></div>',
      '        <div class="dd-kpi-sub">进行中 <strong>' + (to.doingCount || 0) + '</strong> · 已完成 <strong>' + (to.doneCount || 0) + '</strong></div>',
      '      </div>',
      '      <div class="dd-kpi-card">',
      '        <div class="dd-cardhead"><h3>用例执行与测试进度</h3></div>',
      '        <div class="dd-kpi-main">' + (tc.executedCount || 0) + ' <small style="font-size:14px;color:#64748b;font-weight:400">/ ' + (tc.totalCount || 0) + '</small></div>',
      '        <div class="dd-kpi-sub">执行率 <strong>' + (tc.executionRate || 0) + '%</strong> · 通过率 <strong>' + (tc.passRate || 0) + '%</strong></div>',
      '        <div class="dd-progress-bar"><div class="dd-progress-fill" style="width:' + (tc.executionRate || 0) + '%;background:#2563eb;"></div></div>',
      '        <div class="dd-progress-bar" style="margin-top:2px;"><div class="dd-progress-fill" style="width:' + (tc.passRate || 0) + '%;background:#059669;"></div></div>',
      '      </div>',
      '      <div class="dd-kpi-card" id="bugSection">',
      '        <div class="dd-cardhead"><h3>关联缺陷汇总</h3></div>',
      '        <div class="dd-kpi-main">',
      '          <span>' + (bg.totalCount || 0) + ' <small style="font-size:13px;font-weight:400;color:#8a99ad">个</small></span>',
      '          ' + blockingBadge,
      '        </div>',
      '        <div class="dd-kpi-sub">未解决 <strong>' + (bg.activeCount || 0) + '</strong> · 已解决 <strong>' + (bg.resolvedCount || 0) + '</strong></div>',
      '        <div class="dd-kpi-note">链路: 业务需求 → 研发需求 → 缺陷</div>',
      '      </div>',
      '      <div class="dd-kpi-card" id="qualitySection">',
      '        <div class="dd-cardhead"><h3>系统代码质量门禁</h3></div>',
      '        <div class="dd-kpi-main" style="color:' + mainColor + ';font-size:20px;">' + mainTitle + '</div>',
      '        <div class="dd-kpi-sub">涉及 <strong>' + appsCount + '</strong> 应用 · <strong>' + branchCount + '</strong> 分支MR</div>',
      '        <div class="dd-kpi-note">分支绑定应用 · MR 触发扫描</div>',
      '      </div>',
      '    </div>',
      '    <div class="dd-card" id="testOrdersSection"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>测试阶段与测试单流转</h3><span style="font-size:12px;color:#64748b;">同步自禅道 SIT / UAT 任务</span></div>',
      orderRows ? (
        '      <table class="dd-table"><thead><tr><th>测试单号</th><th>标题</th><th>阶段</th><th>状态</th><th>负责人</th><th>计划周期</th><th>操作</th></tr></thead><tbody>' + orderRows + '</tbody></table>'
      ) : '      <div style="color:#8a99ad;font-size:12px;padding:12px 0;">暂无关联测试单 · 提测后将在此同步集成与验收测试单</div>',
      '    </div></div>',
      '    <div class="dd-card" id="storiesSection"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>已分发研发需求推进 (Stories)</h3></div>',
      storyRows ? (
        '      <table class="dd-table"><thead><tr><th>编号</th><th>标题</th><th>所属产品</th><th>负责人</th><th>状态</th><th>关联缺陷</th><th>任务推进</th></tr></thead><tbody>' + storyRows + '</tbody></table>'
      ) : '      <div style="color:#8a99ad;font-size:12px;padding:12px 0;">暂未分发研发需求</div>',
      '    </div></div>',
      '    <div class="dd-card" id="qualityTreeSection"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>各系统代码质量与分支门禁 (按应用分层树状视图)</h3><span style="font-size:12px;color:#64748b;">取各研发分支最新 MR 触发的代码扫描记录</span></div>',
      renderQualityTree(qTree),
      '      <div class="dd-rule-note">💡 <strong>质量治理规范：</strong>代码质量不直接归属业务需求；研发需求（Story）按业务领域关联到应用，代码分支绑定应用并在合并请求（MR）时触发静态扫描与门禁校验。缺陷（Bug）亦通过研发需求间接关联并汇聚。</div>',
      '    </div></div>',
      '  </div>',
      '  <aside class="dd-sidebar">',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>研发与测试责任矩阵</h3></div>',
      '      <div class="dd-kv-list">',
      '        <div class="k">开发负责人</div><div class="v">' + esc(leadsHtml) + '</div>',
      '        <div class="k">测试负责人</div><div class="v">' + esc(leads.testLead || "测试团队") + '</div>',
      '        <div class="k">测试单总数</div><div class="v">' + (to.totalCount || 0) + ' 单</div>',
      '        <div class="k">执行覆盖</div><div class="v">' + (tc.executedCount || 0) + ' / ' + (tc.totalCount || 0) + '</div>',
      '      </div>',
      '    </div></div>',
      '  </aside>',
      '</div>'
    ].join("");
  }


  return {
    renderQualityTree: renderQualityTree,
    renderTabExecution: renderTabExecution
  };
});
