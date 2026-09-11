// =============================================================================
// 文件: web/static/js/po/demand-detail-render.js
// 模块: PO 工作台
// 职责: 业务需求统一详情 Tab 纯渲染入口（概览/需求/交付/历史）；研发执行
//       Tab 见 demand-detail-render-execution.js。全局 API 不变。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define(["./demand-detail-richtext", "./demand-detail-parent", "./demand-detail-render-execution"], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(
      require("./demand-detail-richtext.js"),
      require("./demand-detail-parent.js"),
      require("./demand-detail-render-execution.js")
    );
  } else {
    root.DemandDetailRender = factory(root.DemandDetailRichText, root.DemandDetailParent, root.DemandDetailRenderExecution);
  }
})(typeof self !== "undefined" ? self : this, function (RichText, Parent, Execution) {
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
  var renderTabExecution = (Execution && Execution.renderTabExecution) || function () { return ""; };
  // 公共优先级渲染 (Stage 3): PersonalList.priorityBadge 优先, 缺失时降级到本地实现。
  var priorityBadge = (typeof window !== "undefined" && window.PersonalList && window.PersonalList.priorityBadge) || function (raw) {
    var n = parseInt(String(raw == null ? "" : raw).replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };

  function categoryLabel(value) {
    var key = String(value || "").trim().toLowerCase();
    var labels = { experience: "体验优化", feature: "功能需求", request: "业务需求", business: "业务需求", research: "调研需求", bug: "BUG", tecopt: "技术优化", performance: "性能", safe: "安全", datacg: "数据变更", datachange: "数据变更", dataexport: "数据导出", other: "其他" };
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

  function renderSpotlight(spotlight, data) {
    if (!spotlight) return "";
    data = data || {};
    var pa = data.primaryAction || {};
    var summary = data.summary || {};
    var action;
    var isSubmitTest = String(pa.key || "") === "submit_test" && pa.enabled !== false;
    var isRemindAccept = String(pa.key || "") === "remind_accept" && pa.enabled !== false;
    if (isSubmitTest) {
      var did = String(summary.demandId || summary.id || "").replace(/^US/i, "");
      var title = String(summary.title || "").replace(/"/g, "&quot;");
      action = '<button type="button" class="dd-btn primary js-submit-test" data-demand-id="' + esc(did) + '" data-demand-title="' + esc(summary.title || "") + '">' + esc(pa.label || spotlight.actionLabel || "提测") + '</button>';
    } else if (isRemindAccept) {
      var did = String(summary.demandId || summary.id || "").replace(/^US/i, "");
      var title = String(summary.title || "").replace(/"/g, "&quot;");
      action = '<button type="button" class="dd-btn primary js-po-drawer-action" data-action-key="remind_accept" data-demand-id="' + esc(did) + '" data-demand-title="' + esc(summary.title || "") + '">' + esc(pa.label || spotlight.actionLabel || "催办验收") + '</button>';
    } else if (spotlight.actionUrl) {
      var href = String(spotlight.actionUrl || "");
      var external = /^https?:\/\//i.test(href);
      action = external
        ? '<a class="dd-btn primary" href="' + esc(href) + '" target="_blank" rel="noopener noreferrer">' + esc(spotlight.actionLabel) + '</a>'
        : '<a class="dd-btn primary" href="' + esc(href) + '">' + esc(spotlight.actionLabel) + '</a>';
    } else {
      action = '<button class="dd-btn primary" onclick="DemandDetail.switchTab(\'' + esc(spotlight.targetTab) + '\',\'' + esc(spotlight.targetSection) + '\')">' + esc(spotlight.actionLabel) + '</button>';
    }
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
    var spHtml = renderSpotlight(data.spotlight, data);
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
