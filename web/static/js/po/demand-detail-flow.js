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
   * F4: 关键计划与实际（4 项并排对照 + 4 项时间明细）
   */
  function renderPlanAndActual(summary, actualTimes) {
    summary = summary || {};
    actualTimes = actualTimes || {};

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
      '        <div class="plan-actual-title"><span>开发完成</span>' + calcDeviation(devPlanned, devActual) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(devPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (devActual ? ('<strong>' + esc(devActual) + '</strong>') : '<span class="dd-text-muted">进行中</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>测试完成</span>' + calcDeviation(testPlanned, testActual) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(testPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (testActual ? ('<strong>' + esc(testActual) + '</strong>') : '<span class="dd-text-muted">进行中</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>验收完成</span>' + calcDeviation(verifyPlanned, verifyActual) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计完成:</span><strong>' + esc(verifyPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>实际完成:</span>' + (verifyActual ? ('<strong>' + esc(verifyActual) + '</strong>') : '<span class="dd-text-muted">进行中</span>') + '</div>',
      '        </div>',
      '      </div>',
      '      <div class="plan-actual-item">',
      '        <div class="plan-actual-title"><span>上线/交付</span>' + calcDeviation(launchPlanned, launchActual) + '</div>',
      '        <div class="plan-actual-vals">',
      '          <div class="plan-val-line"><span>预计上线:</span><strong>' + esc(launchPlanned) + '</strong></div>',
      '          <div class="plan-val-line"><span>交付时间:</span>' + (launchActual ? ('<strong>' + esc(launchActual) + '</strong>') : '<span class="dd-text-muted">待交付</span>') + '</div>',
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

  return {
    esc: esc,
    renderZentaoActionBtn: renderZentaoActionBtn,
    renderPlanAndActual: renderPlanAndActual,
    renderImportantChecks: renderImportantChecks,
    toggleCard: toggleCard
  };
});
