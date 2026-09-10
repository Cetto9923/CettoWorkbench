/**
 * =============================================================================
 * 文件: web/static/js/po/follow.js
 * 模块: PO 工作台 - 我的关注主控制器
 * 职责: 业务需求 / 项目周报视图切换；周报 4 KPI、快捷过滤、7 列表格与翻页
 *       业务需求逻辑见 follow-demand.js
 * =============================================================================
 */
(function () {
  "use strict";

  var currentTab = "demand";
  var weeklyItems = [];
  var weeklyStats = { watched: 0, submitted: 0, waiting: 0, abnormal: 0, attention: 0, risk: 0, deviation: 0 };
  var weeklyFilter = "all";
  var weeklyKeyword = "";
  var weeklyPager = { page: 1, pageSize: 10, total: 0 };

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function getCsrfToken() {
    var el = document.getElementById("csrfToken"); return el ? el.value : "";
  }

  function switchTab(tab) {
    if (tab !== "demand" && tab !== "weekly") return;
    currentTab = tab;
    document.querySelectorAll(".follow-tab").forEach(function (btn) {
      btn.classList.toggle("active", btn.getAttribute("data-tab") === tab);
    });
    var weeklySec = document.getElementById("weeklySection");
    var demandSec = document.getElementById("demandSection");
    if (weeklySec) weeklySec.hidden = (tab !== "weekly");
    if (demandSec) demandSec.hidden = (tab === "weekly");
    if (tab === "weekly") loadWeeklyData();
    else if (window.FollowDemand) window.FollowDemand.load();
  }

  async function loadWeeklyData() {
    var tbody = document.getElementById("pwTbody");
    if (tbody) tbody.innerHTML = '<tr><td colspan="7" class="pw-empty-row">加载周报数据中…</td></tr>';
    var params = new URLSearchParams({
      filter: weeklyFilter || "all",
      keyword: weeklyKeyword || "",
      limit: "500",
      scope: "watched"
    });
    try {
      var res = await fetch("/follow/project-weeklies?" + params.toString(), {
        credentials: "include",
        headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
      });
      if (!res.ok) throw new Error("fetch weeklies failed");
      var json = await res.json();
      if (!json || !json.success || !json.data) throw new Error("invalid weeklies payload");
      weeklyItems = Array.isArray(json.data.items) ? json.data.items : [];
      weeklyStats = json.data.stats || weeklyStats;
      weeklyPager.total = weeklyItems.length;
      updateWeeklyStatsUI();
      renderWeeklyRows();
      updateTabBadges();
    } catch (e) {
      if (tbody) tbody.innerHTML = '<tr><td colspan="7" class="pw-empty-row">获取周报失败，请稍后重试</td></tr>';
    }
  }

  function updateWeeklyStatsUI() {
    var setEl = function (id, val) { var el = document.getElementById(id); if (el) el.firstChild.textContent = String(val); };
    var setNum = function (id, val) { var el = document.getElementById(id); if (el) el.textContent = String(val); };
    setEl("pwStatAll", weeklyStats.watched);
    setEl("pwStatSubmitted", weeklyStats.submitted);
    setEl("pwStatWaiting", weeklyStats.waiting);
    setEl("pwStatAbnormal", weeklyStats.abnormal);
    setNum("pwBtnAll", weeklyStats.watched);
    setNum("pwBtnWaiting", weeklyStats.waiting);
    setNum("pwBtnAttention", weeklyStats.attention);
    setNum("pwBtnRisk", weeklyStats.risk);
    setNum("pwBtnDeviation", weeklyStats.deviation);
  }

  function situationTag(item) {
    var label = item.overallSituationLabel || "正常";
    var cls = item.overallSituation === 1 ? "tag orange" : (item.overallSituation === 2 ? "tag red" : "tag green");
    return '<span class="' + cls + '">' + esc(label) + "</span>";
  }

  function submitStatusHtml(status) {
    if (status === "submitted") return '<span class="pw-submit submit-ok">已提交</span>';
    if (status === "overdue") return '<span class="pw-submit submit-late">逾期未报</span>';
    return '<span class="pw-submit submit-wait">待提交</span>';
  }

  function riskLineHtml(item) {
    var chips = [];
    if (item.openIssueCount > 0) chips.push('<span class="tag red">问题 ' + item.openIssueCount + "</span>");
    if (item.openRiskCount > 0) chips.push('<span class="tag orange">风险 ' + item.openRiskCount + "</span>");
    if (item.planDeviationDays > 0) chips.push('<span class="tag orange">计划偏离 ' + item.planDeviationDays + "天</span>");
    if (item.releaseRisk && item.releaseRisk !== "normal") {
      chips.push('<span class="tag red">' + esc(item.releaseRiskLabel || item.releaseRisk) + "</span>");
    }
    if (!chips.length) return '<div class="pw-riskline"><span class="tag green">无明显异常</span></div>';
    var hint = "";
    if (item.planDeviationDays > 0) hint = "存在基线偏离，建议核对排期。";
    else if (item.openIssueCount > 0 || item.openRiskCount > 0) hint = "存在未闭环问题风险，建议持续跟进。";
    return '<div class="pw-riskline">' + chips.join("") + "</div>" + (hint ? '<div class="pw-risk-text">' + esc(hint) + "</div>" : "");
  }

  function investHtml(item) {
    var staff = Number(item.staff) || 0;
    var hours = Number(item.workloadHours) || 0;
    if (staff <= 0 && hours <= 0) {
      return '<div class="pw-invest pw-invest-empty"><b>0</b><span> 人</span><div class="pw-subtle">暂无投入</div></div>';
    }
    return '<div class="pw-invest"><b>' + staff + '</b><span> 人</span><div class="pw-subtle">' +
      (hours ? (Math.round(hours * 10) / 10 + " 工时") : "—") + '</div></div>';
  }

  function renderWeeklyRows() {
    var tbody = document.getElementById("pwTbody");
    if (!tbody) return;
    var start = (weeklyPager.page - 1) * weeklyPager.pageSize;
    var pageItems = weeklyItems.slice(start, start + weeklyPager.pageSize);
    if (!pageItems.length) {
      tbody.innerHTML = '<tr><td colspan="7" class="pw-empty-row">当前条件下暂无关注项目周报</td></tr>';
      renderWeeklyPager();
      return;
    }
    var html = pageItems.map(function (item) {
      var pid = item.projectId;
      var desc = item.overallSituationDesc ? '<div class="pw-desc">' + esc(item.overallSituationDesc) + "</div>" : '<div class="pw-desc muted">—</div>';
      var pmAccount = item.pmAccount ? " (" + esc(item.pmAccount) + ")" : "";
      return "<tr>" +
        '<td><div class="pw-code">' + esc(item.projectCode || "P" + pid) + '</div><div class="pw-name" data-open-detail="' + pid + '">' + esc(item.projectName || "—") + '</div><div class="pw-meta">项目经理：' + esc(item.pmName || item.pmAccount || "—") + pmAccount + '</div></td>' +
        '<td><div class="pw-period">第 ' + (item.weekSN || "—") + ' 周</div><div class="pw-subtle">' + esc(item.weekStart) + ' ~ ' + esc(item.weekEnd) + '</div>' + submitStatusHtml(item.submitStatus) + '</td>' +
        '<td><div class="pw-situation">' + situationTag(item) + '</div>' + desc + '</td>' +
        '<td><div class="pw-progress"><div class="pw-pbox"><b>' + (item.finishedCount || 0) + '</b><span>完成</span></div><div class="pw-pbox"><b>' + (item.unfinishedCount || 0) + '</b><span>未完成</span></div><div class="pw-pbox"><b>' + (item.nextWeekCount || 0) + '</b><span>下周</span></div></div></td>' +
        '<td>' + riskLineHtml(item) + '</td><td>' + investHtml(item) + '</td>' +
        '<td><div class="pw-actions"><button type="button" class="link-btn" data-open-detail="' + pid + '">查看周报</button><button type="button" class="ghost-btn" data-open-history="' + pid + '">历史记录</button><button type="button" class="link-btn muted" data-unwatch-project="' + pid + '">取消关注</button></div></td>' +
        "</tr>";
    }).join("");
    tbody.innerHTML = html;
    renderWeeklyPager();
    bindWeeklyRowEvents();
  }

  function renderWeeklyPager() {
    var countText = document.getElementById("pwCountText");
    var pager = document.getElementById("pwPager");
    if (!pager) return;
    var total = weeklyItems.length;
    var totalPages = Math.max(1, Math.ceil(total / weeklyPager.pageSize));
    if (weeklyPager.page > totalPages) weeklyPager.page = totalPages;
    var start = total === 0 ? 0 : (weeklyPager.page - 1) * weeklyPager.pageSize + 1;
    var end = Math.min(total, weeklyPager.page * weeklyPager.pageSize);
    if (countText) countText.textContent = "显示 " + start + "-" + end + " / 共 " + total + " 个关注项目";
    if (totalPages <= 1) { pager.innerHTML = ""; return; }
    var html = '<button type="button" class="pw-page-btn" id="pwPrevPage"' + (weeklyPager.page === 1 ? " disabled" : "") + ">‹</button>";
    for (var i = 1; i <= totalPages; i++) {
      html += '<button type="button" class="pw-page-btn' + (weeklyPager.page === i ? " active" : "") + '" data-page="' + i + '">' + i + "</button>";
    }
    html += '<button type="button" class="pw-page-btn" id="pwNextPage"' + (weeklyPager.page === totalPages ? " disabled" : "") + ">›</button>";
    pager.innerHTML = html;
    pager.querySelectorAll("[data-page]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        weeklyPager.page = parseInt(btn.getAttribute("data-page"), 10);
        renderWeeklyRows();
      });
    });
    var prev = document.getElementById("pwPrevPage");
    var next = document.getElementById("pwNextPage");
    if (prev) prev.addEventListener("click", function () { weeklyPager.page--; renderWeeklyRows(); });
    if (next) next.addEventListener("click", function () { weeklyPager.page++; renderWeeklyRows(); });
  }

  function bindWeeklyRowEvents() {
    document.querySelectorAll("#weeklySection [data-open-detail]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var pid = parseInt(btn.getAttribute("data-open-detail"), 10);
        if (window.FollowDrawer) window.FollowDrawer.openDetail(pid, riskLineHtml);
      });
    });
    document.querySelectorAll("#weeklySection [data-open-history]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var pid = parseInt(btn.getAttribute("data-open-history"), 10);
        if (window.FollowDrawer) window.FollowDrawer.openHistory(pid);
      });
    });
    document.querySelectorAll("#weeklySection [data-unwatch-project]").forEach(function (btn) {
      btn.addEventListener("click", async function () {
        var pid = parseInt(btn.getAttribute("data-unwatch-project"), 10);
        if (confirm("确认取消关注该项目周报？")) { await unwatchItem("project", pid); }
      });
    });
  }

  async function unwatchItem(type, id) {
    try {
      var csrf = getCsrfToken();
      var endpoint = type === "project" ? "/follow/project-report/" + id : "/follow/demand/" + id;
      var res = await fetch(endpoint, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ followed: false })
      });
      if (res.ok) {
        if (typeof window.showToast === "function") window.showToast("已取消关注");
        if (currentTab === "weekly") loadWeeklyData();
        else if (window.FollowDemand) window.FollowDemand.load();
      }
    } catch (e) {
      if (typeof window.showToast === "function") window.showToast("取消关注失败");
    }
  }

  window.FollowUnwatchDemand = function (id) { return unwatchItem("demand", id); };
  window.FollowUpdateDemandBadge = function (n) {
    var dEl = document.getElementById("tabCountDemand");
    if (dEl) dEl.textContent = String(n || 0);
  };

  function updateTabBadges() {
    var dEl = document.getElementById("tabCountDemand");
    var wEl = document.getElementById("tabCountWeekly");
    var dCount = (window.FollowDemand && window.FollowDemand.getTotal) ? window.FollowDemand.getTotal() : 0;
    var wCount = weeklyStats.watched || weeklyItems.length || 0;
    if (dEl) dEl.textContent = String(dCount);
    if (wEl) wEl.textContent = String(wCount);
  }

  function setWeeklyFilter(filter) {
    weeklyFilter = filter || "all";
    weeklyPager.page = 1;
    document.querySelectorAll("#weeklySection .pw-sum-card").forEach(function (c) {
      c.classList.toggle("active", (c.getAttribute("data-filter") || "all") === weeklyFilter);
    });
    document.querySelectorAll("#weeklySection .pw-filter-btn").forEach(function (b) {
      b.classList.toggle("active", (b.getAttribute("data-filter") || "all") === weeklyFilter);
    });
    loadWeeklyData();
  }

  function init() {
    document.querySelectorAll(".follow-tab").forEach(function (btn) {
      btn.addEventListener("click", function () { switchTab(btn.getAttribute("data-tab")); });
    });

    document.querySelectorAll("#weeklySection .pw-sum-card, #weeklySection .pw-filter-btn").forEach(function (el) {
      el.addEventListener("click", function () {
        setWeeklyFilter(el.getAttribute("data-filter"));
      });
    });

    var pwSearch = document.getElementById("pwSearchInput");
    if (pwSearch) {
      var timer = null;
      pwSearch.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(function () {
          weeklyKeyword = pwSearch.value.trim();
          weeklyPager.page = 1;
          loadWeeklyData();
        }, 300);
      });
    }

    var pwReset = document.getElementById("pwResetBtn");
    if (pwReset) {
      pwReset.addEventListener("click", function () {
        weeklyKeyword = "";
        if (pwSearch) pwSearch.value = "";
        setWeeklyFilter("all");
      });
    }

    var mask = document.getElementById("pwDrawerMask");
    if (mask) {
      mask.addEventListener("click", function (e) {
        if (e.target === mask && window.FollowDrawer) window.FollowDrawer.close();
      });
    }

    if (window.FollowDemand) {
      window.FollowDemand.bind();
      window.FollowDemand.load();
    }
    loadWeeklyData();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
