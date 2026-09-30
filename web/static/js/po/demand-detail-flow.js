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

  /**
   * F4: 偏差天数计算与徽标渲染
   */
  function calcDeviation(plannedStr, actualStr) {
    if (!plannedStr || plannedStr === "—") return "";
    var pDate = new Date(plannedStr.slice(0, 10));
    if (isNaN(pDate.getTime())) return "";
    if (actualStr && actualStr !== "—") {
      var aDate = new Date(actualStr.slice(0, 10));
      if (!isNaN(aDate.getTime())) {
        var diff = Math.round((aDate.getTime() - pDate.getTime()) / 86400000);
        if (diff < 0) {
          return '<span class="deviation-badge ahead">' + diff + '天 提前</span>';
        } else if (diff === 0) {
          return '<span class="deviation-badge ontime">0天 按期</span>';
        } else {
          return '<span class="deviation-badge delayed">+' + diff + '天 延期</span>';
        }
      }
    }
    var today = new Date();
    today.setHours(0, 0, 0, 0);
    var diffFromToday = Math.round((today.getTime() - pDate.getTime()) / 86400000);
    if (diffFromToday > 0) {
      return '<span class="deviation-badge delayed">超期 ' + diffFromToday + '天</span>';
    }
    return '<span class="deviation-badge ontime">预计按期</span>';
  }

  /**
   * R1-3: 终态需求（已关闭 / 已驳回）不再按今天核算超期
   */
  function isTerminalStatus(status) {
    var key = String(status || "").trim().toLowerCase();
    return key === "closed" || key === "refuse";
  }

  /**
   * F4: 关键计划与实际（4 项并排对照 + 4 项时间明细）
   */
  function renderPlanAndActual(summary, actualTimes) {
    summary = summary || {};
    actualTimes = actualTimes || {};

    var terminal = isTerminalStatus(summary.zentaoStatus);
    var pendingActual = terminal ? "—" : "进行中";
    var pendingLaunch = terminal ? "—" : "待交付";
    var devPlanned = summary.developFinish || "—";
    var devActual = actualTimes.actualDevCompletionDate || actualTimes.actualTestStartDate || "";
    var testPlanned = summary.testFinish || "—";
    var testActual = actualTimes.actualTestCompletionDate || actualTimes.submitAcceptanceDate || "";
    var verifyPlanned = summary.verifyFinish || "—";
    var verifyActual = actualTimes.acceptancedDate || "";
    var launchPlanned = summary.estimateLaunch || "—";
    var launchActual = actualTimes.deliverDate || actualTimes.realReleaseDate || "";

    var hasData = Boolean(
      (summary.developFinish && summary.developFinish !== "—") ||
      (summary.testFinish && summary.testFinish !== "—") ||
      (summary.verifyFinish && summary.verifyFinish !== "—") ||
      (summary.estimateLaunch && summary.estimateLaunch !== "—") ||
      devActual || testActual || verifyActual || launchActual ||
      actualTimes.reviewedDate || actualTimes.clarifyDate || actualTimes.firstToStoryDate || actualTimes.actualDevStartDate || actualTimes.realReleaseDate
    );

    var cardCls = "dd-card phase2-collapse-card" + (hasData ? "" : " is-collapsed");
    var toggleBtnText = hasData ? '收起明细 <i class="bi bi-chevron-up"></i>' : '展开查看 <i class="bi bi-chevron-down"></i>';

    var html = [
      '<div class="' + cardCls + '" id="cardPlanActual"><div class="dd-card-body">',
      '  <div class="dd-cardhead">',
      '    <div><h3>关键计划与实际</h3><span class="dd-note">预计与实际并排对照 · 偏差实时核算</span></div>',
      '    <button type="button" class="dd-btn secondary req-sm-btn" onclick="DemandDetailFlow.toggleCard(\'cardPlanActual\', this)">' + toggleBtnText + '</button>',
      '  </div>',
      '  <div class="phase2-card-body">',
      '    <div class="plan-actual-row">',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>开发完成</span>' + (terminal ? "" : calcDeviation(devPlanned, devActual)) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(devPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (devActual ? ('<strong>' + esc(devActual) + '</strong>') : '<span class="dd-text-muted">' + pendingActual + '</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>测试完成</span>' + (terminal ? "" : calcDeviation(testPlanned, testActual)) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(testPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (testActual ? ('<strong>' + esc(testActual) + '</strong>') : '<span class="dd-text-muted">' + pendingActual + '</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>验收完成</span>' + (terminal ? "" : calcDeviation(verifyPlanned, verifyActual)) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(verifyPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (verifyActual ? ('<strong>' + esc(verifyActual) + '</strong>') : '<span class="dd-text-muted">' + pendingActual + '</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>上线/交付</span>' + (terminal ? "" : calcDeviation(launchPlanned, launchActual)) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计上线:</span><strong>' + esc(launchPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>交付时间:</span>' + (launchActual ? ('<strong>' + esc(launchActual) + '</strong>') : '<span class="dd-text-muted">' + pendingLaunch + '</span>') + '</div>',
      '        </div>',
      '      </div>',
      '    </div>',
      '    <div class="req-fields-grid-box">',
      '      <div class="req-fields-grid">',
      '        <div class="req-field-item"><span class="req-field-label">业务评审时间</span><span class="req-field-value">' + esc(actualTimes.reviewedDate || "—") + '</span></div>',
      '        <div class="req-field-item"><span class="req-field-label">首次澄清时间</span><span class="req-field-value">' + esc(actualTimes.clarifyDate || "—") + '</span></div>',
      '        <div class="req-field-item"><span class="req-field-label">首次转研发时间</span><span class="req-field-value">' + esc(actualTimes.firstToStoryDate || actualTimes.actualDevStartDate || "—") + '</span></div>',
      '        <div class="req-field-item"><span class="req-field-label">发版窗口</span><span class="req-field-value">' + esc(summary.publishWindow || (actualTimes.realReleaseDate ? (actualTimes.realReleaseDate + "【发布】") : "—")) + '</span></div>',
      '      </div>',
      '    </div>',
      '  </div>',
      '</div></div>'
    ];
    return html.join("");
  }

  function formatCheckValue(val, raw) {
    if (!val && !raw) return '<span class="req-checklist-val normal">—</span>';
    var text = String(val || raw || "—").trim();
    if (text === "是" || text === "1" || /是\(1\)/.test(text)) {
      return '<span class="req-checklist-val yes"><i class="bi bi-check-circle-fill"></i> ' + esc(text) + '</span>';
    }
    if (text === "否" || text === "0" || /否\(0\)/.test(text)) {
      return '<span class="req-checklist-val no"><i class="bi bi-dash-circle"></i> ' + esc(text) + '</span>';
    }
    return '<span class="req-checklist-val normal">' + esc(text) + '</span>';
  }

  /**
   * F4: 重要检查项（7 项完整呈现）
   */
  function renderImportantChecks(checks) {
    checks = checks || {};
    var c1 = checks.multiLegalPersonLogoLabel || checks.multiLegalPersonLogo;
    var c2 = checks.isRelatedAccountsLabel || checks.isRelatedAccounts;
    var c3 = checks.isImportantOrderLabel || checks.isImportantOrder;
    var c4 = checks.isNeedReviewLabel || checks.isNeedReview;
    var c5 = checks.onetimeAcceptanceLabel || checks.onetimeAcceptance;
    var c6 = checks.isCarReviewLabel || checks.isCarReview;
    var c7 = checks.verifyDateLabel || checks.verifyDate;

    var hasData = Boolean(c1 || c2 || c3 || c4 || c5 || c6 || c7);
    var cardCls = "dd-card phase2-collapse-card" + (hasData ? "" : " is-collapsed");
    var toggleBtnText = hasData ? '收起明细 <i class="bi bi-chevron-up"></i>' : '展开查看 <i class="bi bi-chevron-down"></i>';

    var html = [
      '<div class="' + cardCls + '" id="cardImportantChecks"><div class="dd-card-body">',
      '  <div class="dd-cardhead">',
      '    <div><h3>重要检查项</h3><span class="dd-note">禅道 7 项核心检查基线</span></div>',
      '    <button type="button" class="dd-btn secondary req-sm-btn" onclick="DemandDetailFlow.toggleCard(\'cardImportantChecks\', this)">' + toggleBtnText + '</button>',
      '  </div>',
      '  <div class="phase2-card-body">',
      '    <div class="req-checklist-grid">',
      '      <div class="req-checklist-item"><span class="req-checklist-label">多法人标志</span>' + formatCheckValue(checks.multiLegalPersonLogoLabel, checks.multiLegalPersonLogo) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">涉及账务</span>' + formatCheckValue(checks.isRelatedAccountsLabel, checks.isRelatedAccounts) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">需要质量评审</span>' + formatCheckValue(checks.isImportantOrderLabel, checks.isImportantOrder) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">需要架构评审</span>' + formatCheckValue(checks.isNeedReviewLabel, checks.isNeedReview) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">一次性验收通过</span>' + formatCheckValue(checks.onetimeAcceptanceLabel, checks.onetimeAcceptance) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">快速评审</span>' + formatCheckValue(checks.isCarReviewLabel, checks.isCarReview) + '</div>',
      '      <div class="req-checklist-item"><span class="req-checklist-label">生产验证时间</span>' + formatCheckValue(checks.verifyDateLabel, checks.verifyDate) + '</div>',
      '    </div>',
      '  </div>',
      '</div></div>'
    ];
    return html.join("");
  }

  /**
   * 折叠卡片切换
   */
  function toggleCard(cardId, btnEl) {
    var card = document.getElementById(cardId);
    if (!card) return;
    var isCollapsed = card.classList.toggle("is-collapsed");
    if (btnEl) {
      btnEl.innerHTML = isCollapsed ? '展开查看 <i class="bi bi-chevron-down"></i>' : '收起明细 <i class="bi bi-chevron-up"></i>';
    }
  }

  /**
   * F5: 流程与审批页签渲染（四块只读表，接口没返回的块不渲染，全空显示「暂无流程与审批记录」，各带一个「在禅道办理 ↗」）
   */
  function renderFlowApprovalTab(flowApproval, summary) {
    flowApproval = flowApproval || {};
    summary = summary || {};
    var zentaoUrl = summary.zentaoUrl || "";
    var html = [];

    // 1. 主管部门审批表
    if (flowApproval.managerReviews && flowApproval.managerReviews.length > 0) {
      var mrRows = flowApproval.managerReviews.map(function (m, idx) {
        var prods = (m.products || []).join("、") || "—";
        var reviewer = m.reviewerName || m.reviewer || (m.departmentReviewers || []).join("、") || "—";
        var submitedBy = m.submitedByName || m.submitedBy || "—";
        var res = m.resultLabel || m.result || "—";
        return '<tr>' +
          '<td class="cell-center">' + (idx + 1) + '</td>' +
          '<td>' + esc(prods) + '</td>' +
          '<td>' + esc(reviewer) + '</td>' +
          '<td><span class="dedup-pos-tag">' + esc(res) + '</span></td>' +
          '<td>' + esc(submitedBy) + '</td>' +
          '<td>' + esc(m.submitedDate || m.reviewDate || "—") + '</td>' +
          '</tr>';
      }).join("");

      html.push(
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead">',
        '    <div><h3><i class="bi bi-shield-check"></i> 主管部门审批表</h3><span class="dd-note">涉及产品主管部门会签记录</span></div>',
        '    <div class="dd-cardhead-actions">' + renderZentaoActionBtn(zentaoUrl) + '</div>',
        '  </div>',
        '  <table class="dd-table">',
        '    <thead><tr><th class="col-w-50 cell-center">序号</th><th>涉及产品</th><th>审批人</th><th>审批结果</th><th>提交人</th><th>提交时间</th></tr></thead>',
        '    <tbody>' + mrRows + '</tbody>',
        '  </table>',
        '</div></div>'
      );
    }

    // 2. 需求变更记录表
    if (flowApproval.demandChanges && flowApproval.demandChanges.length > 0) {
      var dcRows = flowApproval.demandChanges.map(function (c) {
        var changeBy = c.changeByName || c.changeBy || "—";
        var changeType = c.changeTypeLabel || c.changeType || "—";
        var res = c.resultLabel || c.result || "—";
        return '<tr>' +
          '<td>' + esc(c.changeDate || c.createdDate || "—") + '</td>' +
          '<td>' + esc(changeBy) + '</td>' +
          '<td>' + esc(changeType) + '</td>' +
          '<td>' + esc(c.desc || "—") + '</td>' +
          '<td>' + esc(c.changeReason || c.reasonType || "—") + '</td>' +
          '<td><span class="dedup-pos-tag">' + esc(res) + '</span></td>' +
          '</tr>';
      }).join("");

      html.push(
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead">',
        '    <div><h3><i class="bi bi-arrow-repeat"></i> 需求变更记录表</h3><span class="dd-note">内容与计划变更审计轨迹</span></div>',
        '    <div class="dd-cardhead-actions">' + renderZentaoActionBtn(zentaoUrl) + '</div>',
        '  </div>',
        '  <table class="dd-table">',
        '    <thead><tr><th>变更时间</th><th>变更人</th><th>变更类型</th><th>变更内容</th><th>变更原因</th><th>评审结果</th></tr></thead>',
        '    <tbody>' + dcRows + '</tbody>',
        '  </table>',
        '</div></div>'
      );
    }

    // 3. 挂起日志表
    if (flowApproval.hangLogs && flowApproval.hangLogs.length > 0) {
      var hlRows = flowApproval.hangLogs.map(function (h) {
        var act = h.actionLabel || h.action || "—";
        var hangType = h.hangUpTypeLabel || h.hangUpType || "—";
        var who = h.accountName || h.account || "—";
        var rawComment = h.comment || h.extra || "—";
        var cleanComment = String(rawComment).replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim() || "—";
        return '<tr>' +
          '<td><span class="dedup-pos-tag">' + esc(act) + '</span></td>' +
          '<td>' + esc(h.date || "—") + '</td>' +
          '<td>' + esc(hangType) + '</td>' +
          '<td>' + esc(cleanComment) + '</td>' +
          '<td>' + esc(who) + '</td>' +
          '</tr>';
      }).join("");

      html.push(
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead">',
        '    <div><h3><i class="bi bi-pause-circle"></i> 挂起日志表</h3><span class="dd-note">需求挂起与重启审计轨迹</span></div>',
        '    <div class="dd-cardhead-actions">' + renderZentaoActionBtn(zentaoUrl) + '</div>',
        '  </div>',
        '  <table class="dd-table">',
        '    <thead><tr><th>操作类型</th><th>操作日期</th><th>挂起类型</th><th>挂起原因</th><th>操作人</th></tr></thead>',
        '    <tbody>' + hlRows + '</tbody>',
        '  </table>',
        '</div></div>'
      );
    }

    // 4. 评审信息表
    if (flowApproval.reviewRecords && flowApproval.reviewRecords.length > 0) {
      var rrRows = flowApproval.reviewRecords.map(function (r) {
        var revType = r.reviewTypeLabel || r.reviewType || "—";
        var createdBy = r.createdByName || r.createdBy || "—";
        var status = r.reviewStatusLabel || r.reviewStatus || "—";
        return '<tr>' +
          '<td>' + esc(revType) + '</td>' +
          '<td>' + esc(r.reviewDate || "—") + '</td>' +
          '<td>' + esc(r.reviewResult || "—") + '</td>' +
          '<td>' + esc(createdBy) + '</td>' +
          '<td>' + esc(r.createdDate || "—") + '</td>' +
          '<td><span class="dedup-pos-tag">' + esc(status) + '</span></td>' +
          '</tr>';
      }).join("");

      html.push(
        '<div class="dd-card"><div class="dd-card-body">',
        '  <div class="dd-cardhead">',
        '    <div><h3><i class="bi bi-card-checklist"></i> 评审信息表</h3><span class="dd-note">业务评审留痕登记</span></div>',
        '    <div class="dd-cardhead-actions">' + renderZentaoActionBtn(zentaoUrl) + '</div>',
        '  </div>',
        '  <table class="dd-table">',
        '    <thead><tr><th>评审类型</th><th>评审日期</th><th>评审内容</th><th>登记人</th><th>登记日期</th><th>评审情况</th></tr></thead>',
        '    <tbody>' + rrRows + '</tbody>',
        '  </table>',
        '</div></div>'
      );
    }

    if (html.length === 0) {
      return '<div class="dd-card"><div class="dd-card-body"><div class="dd-empty-tip">暂无流程与审批记录</div></div></div>';
    }

    return html.join("");
  }

  return {
    esc: esc,
    renderZentaoActionBtn: renderZentaoActionBtn,
    renderPlanAndActual: renderPlanAndActual,
    renderImportantChecks: renderImportantChecks,
    renderFlowApprovalTab: renderFlowApprovalTab,
    toggleCard: toggleCard
  };
});
