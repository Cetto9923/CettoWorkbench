// =============================================================================
// 文件: web/static/js/po/demand-detail-render.js
// 模块: PO 工作台
// 职责: 业务需求统一详情五大 Tab、父子导航与去重模板纯渲染函数。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define(["./demand-detail-richtext", "./demand-detail-parent"], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(require("./demand-detail-richtext.js"), require("./demand-detail-parent.js"));
  } else {
    root.DemandDetailRender = factory(root.DemandDetailRichText, root.DemandDetailParent);
  }
})(typeof self !== "undefined" ? self : this, function (RichText, Parent) {
  "use strict";

  var esc = (RichText && RichText.esc) || function (str) {
    if (str === null || str === undefined) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  };
  var sanitizeRichText = (RichText && RichText.sanitizeRichText) || function (raw) { return esc(raw); };
  var excerpt = (RichText && RichText.excerpt) || function (text, max) {
    if (!text || text === "—") return "";
    var plain = String(text).replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim();
    if (!plain) return "";
    var lim = max || 140;
    return plain.length > lim ? (plain.slice(0, lim) + "…") : plain;
  };
  var renderParentAggregate = (Parent && Parent.renderParentAggregate) || function () { return ""; };
  // 公共优先级渲染 (Stage 3): PersonalList.priorityBadge 优先, 缺失时降级到本地实现。
  var priorityBadge = (typeof window !== "undefined" && window.PersonalList && window.PersonalList.priorityBadge) || function (raw) {
    var n = parseInt(String(raw == null ? "" : raw).replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };

  function categoryLabel(value) {
    var key = String(value || "").trim().toLowerCase();
    var labels = { experience: "体验优化", feature: "功能需求", request: "业务需求", business: "业务需求", research: "调研需求", other: "其他" };
    return labels[key] || value || "—";
  }

  function zentaoStatusLabel(value) {
    var raw = String(value || "").trim(), key = raw.toLowerCase();
    var labels = { developing: "开发中", testing: "测试中", wait: "待评审", draft: "草稿", active: "已评审", closed: "已关闭", canceled: "已取消", cancelled: "已取消", suspended: "已挂起", blocked: "已阻塞", done: "已完成", resolved: "已解决", verified: "已验证", reviewing: "评审中", changed: "已变更", postponed: "已延期" };
    return labels[key] || raw || "—";
  }

  function renderHeader(summary, mode) {
    var tagText = mode === "parentAggregate" ? "父业务需求 · 聚合对象" : (mode === "childUnit" ? "子业务需求 · 交付单元" : "独立交付单元");
    var reviewer = summary.reviewer || "待确认";
    var launch = summary.estimateLaunch || "—";
    var stageLabel = String(summary.zentaoStatus || "").toLowerCase() === "wait" ? "待受理" : (summary.valueStageLabel || summary.valueStage || "—");
    var zentaoUrl = summary.zentaoUrl || "";

    return '<div class="dd-idrow">' +
      (zentaoUrl ? '<a class="dd-id table-id-link" href="' + esc(zentaoUrl) + '" target="_blank" rel="noopener noreferrer">' + esc(summary.code) + '</a>' : '<span class="dd-id">' + esc(summary.code) + '</span>') +
      '<span class="dd-tag blue">' + esc(tagText) + '</span>' +
      priorityBadge(summary.priority) +
      '<div class="dd-head-actions">' +
      '  <button class="ui-close-btn" onclick="DemandDetail.close()" title="关闭" aria-label="关闭">×</button>' +
      '</div></div>' +
      '<div class="dd-title">' + esc(summary.title) + '</div>' +
      '<div class="dd-meta">' +
      '  <span>提出人：' + esc(summary.proposerName) + '</span>' +
      '  <span>PO：' + esc(summary.ownerName) + '</span>' +
      '  <span>最近更新：' + esc(summary.editedDate) + '</span>' +
      '</div>' +
      '<div class="dd-spot-cards-3">' +
      '  <div class="dd-spot-card"><div class="lab">当前阶段</div><div class="val blue">' + esc(stageLabel) + '</div></div>' +
      '  <div class="dd-spot-card"><div class="lab">业务评审人</div><div class="val">' + esc(reviewer) + '</div></div>' +
      '  <div class="dd-spot-card"><div class="lab">目标上线</div><div class="val">' + esc(launch) + '</div></div>' +
      '</div>' +
      '<div class="dd-core-strip">' +
      '  <div class="dd-core-item"><div class="k">需求类别</div><div class="v">' + esc(categoryLabel(summary.category)) + '</div></div>' +
      '  <div class="dd-core-item"><div class="k">需求来源</div><div class="v">' + esc(summary.source) + '</div></div>' +
      '  <div class="dd-core-item"><div class="k">所属产品</div><div class="v">' + esc(summary.product) + '</div></div>' +
      '  <div class="dd-core-item"><div class="k">所属需求池</div><div class="v">' + esc(summary.poolName) + '</div></div>' +
      '</div>' +
      '<div class="dd-native-line"><b>禅道状态</b><span class="dd-native-state">' + esc(zentaoStatusLabel(summary.zentaoStatus)) + '</span><span style="color:#cbd5e1">•</span><span>原始状态仅用于追溯，工作台按价值流阶段统一展示与办理</span></div>';
  }

  function renderRelationNav(ctx, currentId) {
    if (!ctx || (!ctx.parent && (!ctx.siblings || ctx.siblings.length === 0))) return "";
    var html = '<div class="dd-relation-nav">';
    if (ctx.parent) {
      html += '<div class="dd-relation-top"><button class="dd-parent-link" type="button" onclick="DemandDetail.open(' + ctx.parent.demandId + ')">' +
        '<span class="dd-back-icon">←</span><span><small style="color:#8a99ad;font-size:11px;display:block">父业务需求</small><strong>' + esc(ctx.parent.code) + ' · ' + esc(ctx.parent.title) + '</strong></span>' +
        '</button></div>';
    }
    if (ctx.siblings && ctx.siblings.length > 0) {
      html += '<div class="dd-sibling-row"><span style="font-size:11px;color:#8a99ad;margin-right:4px">同级交付单元：</span>';
      for (var i = 0; i < ctx.siblings.length; i++) {
        var sib = ctx.siblings[i];
        var cls = "dd-sibling-chip" + (sib.isCurrent ? " active" : (sib.hasRisk ? " risk" : (sib.isDone ? " done" : "")));
        html += '<button class="' + cls + '" onclick="DemandDetail.open(' + sib.demandId + ')"><strong>' + esc(sib.code) + '</strong><span>' + esc(sib.stage) + '</span></button>';
      }
      html += '</div>';
    }
    return html + '</div>';
  }

  function renderSpotlight(spotlight) {
    if (!spotlight) return "";
    var action = spotlight.actionUrl
      ? '<a class="dd-btn primary" href="' + esc(spotlight.actionUrl) + '">' + esc(spotlight.actionLabel) + '</a>'
      : '<button class="dd-btn primary" onclick="DemandDetail.switchTab(\'' + esc(spotlight.targetTab) + '\',\'' + esc(spotlight.targetSection) + '\')">' + esc(spotlight.actionLabel) + '</button>';
    return [
      '<div class="dd-card dd-spot" id="spotlightSection">',
      '  <div>',
      '    <span class="dd-tag blue">' + esc(spotlight.badge) + '</span>',
      '    <div class="dd-spot-title">' + esc(spotlight.title) + '</div>',
      '    <div class="dd-spot-desc">' + esc(spotlight.desc) + '</div>',
      '  </div>',
      '  <div>' + action + '</div>',
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
      '    <div class="dd-cardhead"><h3>交付单元价值流</h3><span class="dd-note">9 阶段 · 已发生节点显示实际耗时</span></div>',
      '    <div class="dd-cycle-chips-4">',
      '      <div class="dd-cycle-chip"><div class="k">预计交付周期</div><div class="v">' + vs.estimatedCycleDays + ' 天</div><div class="s">禅道预计交付周期</div></div>',
      '      <div class="dd-cycle-chip"><div class="k">当前已用周期</div><div class="v">' + vs.usedCycleDays + ' 天</div><div class="s">未发布，统计至今天</div></div>',
      '      <div class="dd-cycle-chip"><div class="k">周期目标</div><div class="v">' + vs.targetCycleDays + ' 天</div><div class="s">目标周期</div></div>',
      '      <div class="dd-cycle-chip ' + overdueClass + '"><div class="k">周期偏差</div><div class="v">' + diffSign + ' 天</div><div class="s">预计交付周期 - 周期目标</div></div>',
      '    </div>',
      '    <div class="dd-flow">'
    ];

    for (var i = 0; i < vs.stages.length; i++) {
      var s = vs.stages[i];
      html.push(
        '<div class="dd-flow-stage ' + esc(s.status) + '">',
        '  <div class="nm">' + esc(s.label) + '</div>',
        '  <div class="who">' + esc(s.role) + '</div>',
        '  <div class="duration" title="' + esc(s.durationText) + '">' + esc(excerpt(s.durationText, 18)) + '</div>',
        '</div>'
      );
    }

    html.push('</div>',
      '<div class="dd-flow-legend"><span><b>已完成：</b>实际耗时</span><span><b>当前阶段：</b>已持续时间</span><span><b>未开始：</b>待计划</span></div>',
      '</div></div>');
    return html.join("");
  }

  function renderTabOverview(data) {
    var vsHtml = renderValueStream(data.valueStream);
    var spHtml = renderSpotlight(data.spotlight);
    var summary = data.summary;
    var descTxt = excerpt(summary.desc, 160) || "暂无需求描述";
    var verifyTxt = excerpt(summary.verifyPlan, 160) || "暂无验收标准摘要";

    var tasksPair = summary.tasksTotal > 0 ? (summary.tasksDone + " / " + summary.tasksTotal) : "—";
    var casesPair = summary.casesTotal > 0 ? (summary.casesExecuted + " / " + summary.casesTotal) : "—";
    var blocked = summary.bugsUnresolved > 0 ? (summary.bugsUnresolved + " 阻塞") : "无";

    return [
      '<div class="dd-grid">',
      '  <div class="dd-main">',
      spHtml,
      vsHtml,
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>需求内容摘要</h3><span class="dd-note">详情在「需求与澄清」</span></div>',
      '      <div class="dd-summary-block">',
      '        <div class="st"><b>需求描述</b><a href="javascript:void(0)" onclick="DemandDetail.switchTab(\'requirement\')">查看完整内容 →</a></div>',
      '        <p>' + esc(descTxt) + '</p>',
      '      </div>',
      '      <div class="dd-summary-block" style="margin-top:10px">',
      '        <div class="st"><b>验收标准</b><a href="javascript:void(0)" onclick="DemandDetail.switchTab(\'requirement\')">查看完整内容 →</a></div>',
      '        <p>' + esc(verifyTxt) + '</p>',
      '      </div>',
      '    </div></div>',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>当前交付状态</h3><span class="dd-note">跨研发、测试、验收的关键推进指标</span></div>',
      '      <div class="dd-metrics-5">',
      '        <div class="dd-metric-box"><div class="n">' + (summary.storiesCount || 0) + '</div><div class="l">已分发研发需求</div></div>',
      '        <div class="dd-metric-box"><div class="n">' + esc(tasksPair) + '</div><div class="l">任务完成</div></div>',
      '        <div class="dd-metric-box"><div class="n">' + esc(casesPair) + '</div><div class="l">用例执行</div></div>',
      '        <div class="dd-metric-box"><div class="n">' + (summary.bugsUnresolved || 0) + '</div><div class="l">未解决 Bug</div></div>',
      '        <div class="dd-metric-box"><div class="n">' + esc(summary.acceptanceStatus || "—") + '</div><div class="l">业务验收</div></div>',
      '      </div>',
      '    </div></div>',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>关键计划与实际</h3><span class="dd-note">只展示跨阶段关键里程碑</span></div>',
      '      <div class="dd-milestones">',
      '        <div class="dd-ms"><div class="k">开发完成</div><div class="v">' + esc(summary.developFinish || "—") + '</div><div class="sub">计划完成</div></div>',
      '        <div class="dd-ms"><div class="k">测试完成</div><div class="v">' + esc(summary.testFinish || "—") + '</div><div class="sub">计划完成</div></div>',
      '        <div class="dd-ms"><div class="k">验收完成</div><div class="v">' + esc(summary.verifyFinish || "—") + '</div><div class="sub">计划完成</div></div>',
      '        <div class="dd-ms"><div class="k">目标上线</div><div class="v">' + esc(summary.estimateLaunch || "—") + '</div><div class="sub">当前目标日期</div></div>',
      '      </div>',
      '    </div></div>',
      '  </div>',
      '  <aside class="dd-sidebar">',
      '    <div class="dd-card"><div class="dd-card-body">',
      '      <div class="dd-cardhead"><h3>需求基本信息</h3></div>',
      '      <div class="dd-aside-title">业务属性</div>',
      '      <div class="dd-kv-list compact">',
      '        <div class="k">交付形态</div><div class="v">独立交付单元</div>',
      '        <div class="k">需求类别</div><div class="v">' + esc(categoryLabel(summary.category)) + '</div>',
      '        <div class="k">需求来源</div><div class="v">' + esc(summary.source) + '</div>',
      '        <div class="k">优先级</div><div class="v">' + esc(summary.priority) + '</div>',
      '        <div class="k">主系统</div><div class="v">' + esc(summary.mainSystemName) + '</div>',
      '        <div class="k">所属产品</div><div class="v">' + esc(summary.product) + '</div>',
      '        <div class="k">所属需求池</div><div class="v">' + esc(summary.poolName) + '</div>',
      '        <div class="k">BSA等级</div><div class="v">' + esc(summary.bsa) + '</div>',
      '        <div class="k">来源备注</div><div class="v">' + esc(summary.sourceNote) + '</div>',
      '      </div>',
      '      <div class="dd-aside-title" style="margin-top:14px">责任与状态</div>',
      '      <div class="dd-kv-list compact">',
      '        <div class="k">负责人 / PO</div><div class="v">' + esc(summary.ownerName) + '</div>',
      '        <div class="k">需求提出人</div><div class="v">' + esc(summary.proposerName) + '</div>',
      '        <div class="k">测试负责人</div><div class="v">' + esc(summary.testOwner) + '</div>',
      '        <div class="k">验收负责人</div><div class="v">' + esc(summary.acceptOwner) + '</div>',
      '        <div class="k">业务评审人</div><div class="v">' + esc(summary.reviewer) + '</div>',
      '        <div class="k">当前责任人</div><div class="v">' + esc(summary.currentOwner || "待确认") + '</div>',
      '        <div class="k">禅道状态</div><div class="v">' + esc(summary.zentaoStatus) + '</div>',
      '        <div class="k">当前阻塞</div><div class="v">' + esc(blocked) + '</div>',
      '      </div>',
      '    </div></div>',
      '  </aside>',
      '</div>'
    ].join("");
  }

  function renderTabRequirement(req) {
    if (!req) return '<div class="dd-card dd-card-body">暂无需求说明</div>';
    var clarifyRows = (req.clarifications || []).map(function (c) {
      return '<tr><td><strong>' + esc(c.productName) + '</strong></td><td>' + esc(c.analyst) + '</td><td>' + esc(c.content) + '</td><td>' + esc(c.devEnd) + '</td><td>' + esc(c.testEnd) + '</td></tr>';
    }).join("");
    var filesRows = (req.attachments || []).map(function (f) {
      return '<li><a href="' + esc(f.download) + '" target="_blank">' + esc(f.title) + '</a> (' + esc(f.size) + ')</li>';
    }).join("");

    var safeSpecHtml = sanitizeRichText(req.specHtml);
    var safeVerifyHtml = sanitizeRichText(req.verifyHtml);

    return '<div class="dd-card" id="requirementSection"><div class="dd-card-body">' +
      '<div class="dd-cardhead"><h3>业务需求正文与描述</h3></div><div style="font-size:13px;line-height:1.7;color:#334155;margin-bottom:16px;">' + (safeSpecHtml || "—") + '</div>' +
      '<div class="dd-cardhead"><h3>验收标准 (Verify Plan)</h3></div><div style="font-size:13px;line-height:1.7;color:#334155;">' + (safeVerifyHtml || "—") + '</div>' +
      '</div></div>' +
      '<div class="dd-card" id="clarificationSection"><div class="dd-card-body">' +
      '<div class="dd-cardhead"><h3>系统/产品维度澄清说明</h3></div>' +
      (clarifyRows ? '<table class="dd-table"><thead><tr><th>产品/系统</th><th>需求分析师</th><th>澄清要点</th><th>计划开发完成</th><th>计划测试完成</th></tr></thead><tbody>' + clarifyRows + '</tbody></table>' : '<div style="color:#8a99ad;font-size:12px;">暂无多系统澄清拆解</div>') +
      '</div></div>' +
      '<div class="dd-card" id="clarificationActionSection"><div class="dd-card-body">' +
      '<div class="dd-cardhead"><h3>需求澄清协同与办理动作</h3></div>' +
      '<div style="display:flex;align-items:center;justify-content:space-between;background:#f8fafc;padding:12px 16px;border-radius:6px;border:1px solid #e2e8f0;">' +
      '  <div><div style="font-weight:600;font-size:13px;color:#1e293b;">需求澄清办理</div><div style="font-size:12px;color:#64748b;margin-top:2px;">支持在工作台直接办理澄清并更新涉及系统与交付节点。</div></div>' +
      '  <div style="display:flex;gap:8px;">' +
      (req.demandId ? '<button type="button" class="dd-btn js-drawer-clarify-btn" data-demand-id="' + esc(req.demandId) + '" style="background:#2563eb;color:#fff;border-color:#2563eb;cursor:pointer;">办理需求澄清</button>' : (req.clarifyZtUrl ? '<a href="' + esc(req.clarifyZtUrl) + '" target="_blank" rel="noopener noreferrer" class="dd-btn" style="background:#2563eb;color:#fff;border-color:#2563eb;text-decoration:none;">在禅道办理需求澄清 ↗</a>' : '<span style="font-size:12px;color:#94a3b8;">暂无澄清入口</span>')) +
      '  </div></div></div>' +
      (filesRows ? '<div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>需求附件</h3></div><ul style="padding-left:18px;margin:0;font-size:12px;color:#2563eb;">' + filesRows + '</ul></div></div>' : '');
  }

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

  function renderTabDelivery(delivery) {
    if (!delivery) return '<div class="dd-card dd-card-body">暂无交付数据</div>';
    function pill(ok) { return ok ? '<span class="dd-tag green">已就绪</span>' : '<span class="dd-tag">进行中</span>'; }
    return '<div class="dd-card" id="deliverySection"><div class="dd-card-body">' +
      '<div class="dd-cardhead"><h3>交付就绪度评估 (Readiness)</h3></div>' +
      '<div style="display:grid;grid-template-columns:repeat(4,1fr);gap:12px;margin-bottom:16px;">' +
      '  <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd"><div style="font-size:11px;color:#8a99ad;margin-bottom:4px">研发完成</div>' + pill(delivery.devDone) + '</div>' +
      '  <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd"><div style="font-size:11px;color:#8a99ad;margin-bottom:4px">测试通过</div>' + pill(delivery.testPassed) + '</div>' +
      '  <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd"><div style="font-size:11px;color:#8a99ad;margin-bottom:4px">严重缺陷闭环</div>' + pill(delivery.bugsResolved) + '</div>' +
      '  <div style="border:1px solid #e2e8f0;padding:12px;border-radius:6px;background:#fafbfd"><div style="font-size:11px;color:#8a99ad;margin-bottom:4px">业务验收</div>' + pill(delivery.acceptanceDone) + '</div>' +
      '</div>' +
      '<div class="dd-cardhead"><h3>发布与投产信息</h3></div>' +
      '<div class="dd-kv-list" style="grid-template-columns:120px 1fr;">' +
      '  <div class="k">目标上线时间</div><div class="v">' + esc(delivery.estimateLaunch) + '</div>' +
      '  <div class="k">发布规划窗口</div><div class="v">' + esc(delivery.publishWindow) + '</div>' +
      '  <div class="k">生产验证结论</div><div class="v">' + esc(delivery.verifyConclusion) + '</div>' +
      '</div></div></div>';
  }

  function renderTabHistory(history) {
    if (!history) return '<div class="dd-card dd-card-body">暂无过程记录</div>';
    var actionRows = (history.actions || []).map(function (a) {
      return '<tr><td style="white-space:nowrap;">' + esc(a.date) + '</td><td><strong>' + esc(a.actor) + '</strong></td><td>' + esc(a.action) + '</td><td>' + esc(a.extra) + '</td></tr>';
    }).join("");
    var lc = history.lifecycle || {};
    return '<div class="dd-grid"><div class="dd-main">' +
      '<div class="dd-card" id="historySection"><div class="dd-card-body"><div class="dd-cardhead"><h3>禅道操作审计日志</h3></div>' +
      (actionRows ? '<table class="dd-table"><thead><tr><th>时间</th><th>操作人</th><th>动作</th><th>说明</th></tr></thead><tbody>' + actionRows + '</tbody></table>' : '<div style="color:#8a99ad;font-size:12px;">暂无历史动作</div>') +
      '</div></div></div>' +
      '<aside class="dd-sidebar"><div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>生命周期经办人</h3></div>' +
      '<div class="dd-kv-list">' +
      '<div class="k">创建人</div><div class="v">' + esc(lc.createdBy) + '</div><div class="k">创建时间</div><div class="v">' + esc(lc.createdDate) + '</div>' +
      '<div class="k">评审人</div><div class="v">' + esc(lc.reviewer) + '</div><div class="k">评审时间</div><div class="v">' + esc(lc.reviewedDate) + '</div>' +
      '<div class="k">最后编辑</div><div class="v">' + esc(lc.lastEditedBy) + '</div><div class="k">最后更新</div><div class="v">' + esc(lc.lastEditedDate) + '</div>' +
      '<div class="k">关闭人</div><div class="v">' + esc(lc.closedBy) + '</div><div class="k">关闭原因</div><div class="v">' + esc(lc.closedReason) + '</div>' +
      '</div></div></div></aside></div>';
  }

  return {
    esc: esc, sanitizeRichText: sanitizeRichText, renderHeader: renderHeader, renderRelationNav: renderRelationNav,
    renderTabOverview: renderTabOverview, renderTabRequirement: renderTabRequirement,
    renderTabExecution: renderTabExecution, renderTabDelivery: renderTabDelivery,
    renderTabHistory: renderTabHistory, renderParentAggregate: renderParentAggregate,
    priorityBadge: priorityBadge
  };
});
