/**
 * =============================================================================
 * 文件: web/static/js/po/follow-drawer.js
 * 模块: PO 工作台 - 关注周报抽屉
 * 职责: 项目周报侧滑详情抽屉与历史周报侧滑抽屉展示
 * =============================================================================
 */
(function () {
  "use strict";

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function closeWeeklyDrawer() {
    var mask = document.getElementById("pwDrawerMask");
    if (mask) mask.hidden = true;
  }

  async function openWeeklyDrawer(projectId, riskLineFn) {
    var mask = document.getElementById("pwDrawerMask");
    var drawer = document.getElementById("pwDrawer");
    if (!mask || !drawer) return;

    drawer.innerHTML = '<div class="pw-drawer-body">加载周报详情中…</div>';
    mask.hidden = false;

    try {
      var res = await fetch("/follow/project-weeklies/" + projectId, {
        credentials: "include",
        headers: { "Content-Type": "application/json" }
      });
      var json = await res.json();
      var item = json && json.success && json.data ? json.data : null;
      if (!item) {
        drawer.innerHTML = '<div class="pw-drawer-body">未找到该周报详情</div>';
        return;
      }
      var submitLabel = item.submitStatus === "submitted" ? "已提交" : (item.submitStatus === "overdue" ? "逾期未报" : "待提交");
      var riskHtml = typeof riskLineFn === "function" ? riskLineFn(item) : "";

      drawer.innerHTML =
        '<div class="pw-drawer-head">' +
        '  <div>' +
        '    <h2>' + esc(item.projectName) + '</h2>' +
        '    <div class="pw-subtle">' + esc(item.projectCode) + ' · ' + esc(item.weekStart) + ' ~ ' + esc(item.weekEnd) + '</div>' +
        '    <div class="pw-subtle" style="margin-top:4px">项目经理：' + esc(item.pmName || item.pmAccount || "—") +
        (item.pmAccount ? " (" + esc(item.pmAccount) + ")" : "") +
        (item.pmDeptName ? " · 承建团队：" + esc(item.pmDeptName) : "") + '</div>' +
        '  </div>' +
        '  <button type="button" class="pw-close" id="pwDrawerClose">×</button>' +
        '</div>' +
        '<div class="pw-drawer-body">' +
        (item.warning ? '<div class="pw-warning">' + esc(item.warning) + '</div>' : '') +
        '<div class="pw-drawer-grid">' +
        '  <div class="pw-info"><span>项目状态评估</span><b>' + esc(item.overallSituationLabel || "正常") + '</b></div>' +
        '  <div class="pw-info"><span>周报提交</span><b>' + esc(submitLabel) + '</b></div>' +
        '  <div class="pw-info"><span>本周投入</span><b>' + (item.staff || 0) + ' 人 / ' + (Math.round((item.workloadHours || 0) * 10) / 10) + ' 工时</b></div>' +
        '  <div class="pw-info"><span>问题 / 风险</span><b>' + (item.openIssueCount || 0) + ' / ' + (item.openRiskCount || 0) + '</b></div>' +
        '</div>' +
        '<div class="pw-block"><div class="pw-block-title">项目状态描述</div><div class="pw-note">' + esc(item.overallSituationDesc || "暂无描述") + '</div></div>' +
        '<div class="pw-block"><div class="pw-block-title">本周推进情况</div><div class="pw-progress">' +
        '  <div class="pw-pbox"><b>' + (item.finishedCount || 0) + '</b><span>本周完成</span></div>' +
        '  <div class="pw-pbox"><b>' + (item.unfinishedCount || 0) + '</b><span>未完成</span></div>' +
        '  <div class="pw-pbox"><b>' + (item.nextWeekCount || 0) + '</b><span>下周计划</span></div>' +
        '</div></div>' +
        '<div class="pw-block"><div class="pw-block-title">基线与发布偏离</div>' + riskHtml +
        '  <div class="pw-subtle" style="margin-top:6px">计划偏离 ' + (item.planDeviationDays || 0) + ' 天；上线 ' + esc(item.releaseRiskLabel || "正常") + '</div>' +
        '</div>' +
        (item.url ? '<div class="pw-block" style="margin-top:20px"><a class="link-btn" href="' + esc(item.url) + '" target="_blank" rel="noopener">在禅道打开完整周报 ↗</a></div>' : '') +
        '</div>';

      var closeBtn = document.getElementById("pwDrawerClose");
      if (closeBtn) closeBtn.addEventListener("click", closeWeeklyDrawer);
    } catch (e) {
      drawer.innerHTML = '<div class="pw-drawer-body">加载失败</div>';
    }
  }

  async function openWeeklyHistory(projectId) {
    var mask = document.getElementById("pwDrawerMask");
    var drawer = document.getElementById("pwDrawer");
    if (!mask || !drawer) return;

    drawer.innerHTML = '<div class="pw-drawer-body">加载周报历史中…</div>';
    mask.hidden = false;

    try {
      var res = await fetch("/follow/project-weeklies/" + projectId + "/history", {
        credentials: "include",
        headers: { "Content-Type": "application/json" }
      });
      var json = await res.json();
      var items = json && json.success && Array.isArray(json.items) ? json.items : [];

      var rows = items.map(function (row) {
        return (
          "<tr>" +
          "  <td>" + esc(row.weekStart) + " ~ " + esc(row.weekEnd) + "</td>" +
          "  <td>" + esc(row.overallSituationLabel || "正常") + "</td>" +
          "  <td>问题 " + (row.openIssueCount || 0) + " / 风险 " + (row.openRiskCount || 0) + "</td>" +
          "  <td>" + (row.planDeviationDays || 0) + " 天</td>" +
          "  <td>" + esc(row.releaseRiskLabel || "正常") + "</td>" +
          "  <td>" + (row.staff || 0) + " 人 / " + Math.round((row.workloadHours || 0) * 10) / 10 + " 工时</td>" +
          "</tr>"
        );
      }).join("");

      drawer.innerHTML =
        '<div class="pw-drawer-head">' +
        "  <h2>项目历史周报</h2>" +
        '  <button type="button" class="pw-close" id="pwDrawerClose">×</button>' +
        "</div>" +
        '<div class="pw-drawer-body">' +
        '  <table class="pw-history-table">' +
        "    <thead><tr><th>周期</th><th>评估</th><th>问题/风险</th><th>偏离</th><th>上线状态</th><th>投入</th></tr></thead>" +
        "    <tbody>" + (rows || '<tr><td colspan="6" style="text-align:center;color:#94a3b8">暂无历史记录</td></tr>') + "</tbody>" +
        "  </table>" +
        "</div>";

      var closeBtn = document.getElementById("pwDrawerClose");
      if (closeBtn) closeBtn.addEventListener("click", closeWeeklyDrawer);
    } catch (e) {
      drawer.innerHTML = '<div class="pw-drawer-body">加载历史失败</div>';
    }
  }

  document.addEventListener("keydown", function (e) {
    if (e.key === "Escape" || e.key === "Esc") {
      closeWeeklyDrawer();
    }
  });

  window.FollowDrawer = {
    openDetail: openWeeklyDrawer,
    openHistory: openWeeklyHistory,
    close: closeWeeklyDrawer
  };
})();
