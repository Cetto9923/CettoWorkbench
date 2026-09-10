/**
 * =============================================================================
 * 文件: web/static/js/po/follow.js
 * 模块: PO 工作台 - 我的关注主控制器
 * 职责: 业务需求 / 项目周报视图切换；周报 4 KPI、快捷过滤、6 列表格与翻页
 *       默认 scope: "mine"（我参与 ∪ 我关注），参与排前并标来源
 *       业务需求逻辑见 follow-demand.js
 * =============================================================================
 */
(function () {
  "use strict";

  var currentTab = "demand";
  var weeklyItems = [];
  var weeklyStats = { watched: 0, submitted: 0, waiting: 0, abnormal: 0, risk: 0, deviation: 0 };
  var weeklyFilter = "all";
  var weeklyScope = "mine";
  var weeklyKeyword = "";
  var PL = window.PersonalList || {};
  var weeklyPager = {
    page: 1,
    pageSize: (typeof PL.loadPageSize === "function") ? PL.loadPageSize("po.follow.weekly.pageSize", 10, [10, 15, 20, 30, 50]) : 10,
    total: 0
  };

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
    if (tbody) tbody.innerHTML = '<tr><td colspan="6" class="pw-empty-row">加载周报数据中…</td></tr>';
    var params = new URLSearchParams({
      filter: weeklyFilter || "all",
      keyword: weeklyKeyword || "",
      limit: "500",
      scope: weeklyScope || "mine"
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
      if (tbody) tbody.innerHTML = '<tr><td colspan="6" class="pw-empty-row">获取周报失败，请稍后重试</td></tr>';
    }
  }

  function updateWeeklyStatsUI() {
    var setEl = function (id, val) { var el = document.getElementById(id); if (el) el.firstChild.textContent = String(val); };
    var setNum = function (id, val) { var el = document.getElementById(id); if (el) el.textContent = String(val); };
    var totalCount = weeklyStats.watched || weeklyItems.length || 0;
    setEl("pwStatAll", totalCount);
    setEl("pwStatSubmitted", weeklyStats.submitted || 0);
    setEl("pwStatWaiting", weeklyStats.waiting || 0);
    setEl("pwStatAbnormal", weeklyStats.abnormal || 0);
    setNum("pwBtnAll", totalCount);
    setNum("pwBtnWaiting", weeklyStats.waiting || 0);
    setNum("pwBtnRisk", weeklyStats.risk || 0);
    setNum("pwBtnDeviation", weeklyStats.deviation || 0);
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

  function renderWeeklyRows() {
    var tbody = document.getElementById("pwTbody");
    if (!tbody) return;
    var start = (weeklyPager.page - 1) * weeklyPager.pageSize;
    var pageItems = weeklyItems.slice(start, start + weeklyPager.pageSize);
    if (!pageItems.length) {
      tbody.innerHTML = '<tr><td colspan="6" class="pw-empty-row">当前条件下暂无项目周报</td></tr>';
      renderWeeklyPager();
      return;
    }
    var html = pageItems.map(function (item) {
      var pid = item.projectId;
      var desc = item.overallSituationDesc ? '<div class="pw-desc">' + esc(item.overallSituationDesc) + "</div>" : '<div class="pw-desc muted">—</div>';
      var pmAccount = item.pmAccount ? " (" + esc(item.pmAccount) + ")" : "";

      var sourceBadge = "";
      if (item.source === "both") {
        sourceBadge = '<span class="tag blue" style="margin-left:6px;font-size:11px;">参与·关注</span>';
      } else if (item.source === "participated" || item.isParticipated) {
        sourceBadge = '<span class="tag blue" style="margin-left:6px;font-size:11px;">参与</span>';
      } else {
        sourceBadge = '<span class="tag" style="margin-left:6px;font-size:11px;">关注</span>';
      }

      var unwatchBtn = "";
      if (item.isWatched || item.source === "watched" || item.source === "both") {
        unwatchBtn = '<button type="button" class="link-btn muted" data-unwatch-project="' + pid + '">取消关注</button>';
      } else {
        unwatchBtn = '<button type="button" class="link-btn muted" disabled title="参与项目请在禅道团队管理，不在此取关" style="opacity:0.4;cursor:not-allowed;">取消关注</button>';
      }

      return "<tr>" +
        '<td><div class="pw-code">' + esc(item.projectCode || "P" + pid) + '</div><div class="pw-name" data-open-detail="' + pid + '">' + esc(item.projectName || "—") + sourceBadge + '</div><div class="pw-meta">项目经理：' + esc(item.pmName || item.pmAccount || "—") + pmAccount + '</div></td>' +
        '<td><div class="pw-period">第 ' + (item.weekSN || "—") + ' 周</div><div class="pw-subtle">' + esc(item.weekStart) + ' ~ ' + esc(item.weekEnd) + '</div>' + submitStatusHtml(item.submitStatus) + '</td>' +
        '<td><div class="pw-situation">' + situationTag(item) + '</div>' + desc + '</td>' +
        '<td><div class="pw-progress"><div class="pw-pbox"><b>' + (item.finishedCount || 0) + '</b><span>完成</span></div><div class="pw-pbox"><b>' + (item.unfinishedCount || 0) + '</b><span>未完成</span></div><div class="pw-pbox"><b>' + (item.nextWeekCount || 0) + '</b><span>下周</span></div></div></td>' +
        '<td>' + riskLineHtml(item) + '</td>' +
        '<td><div class="pw-actions"><button type="button" class="link-btn" data-open-detail="' + pid + '">查看周报</button><button type="button" class="ghost-btn" data-open-history="' + pid + '">历史记录</button>' + unwatchBtn + '</div></td>' +
        "</tr>";
    }).join("");
    tbody.innerHTML = html;
    renderWeeklyPager();
    bindWeeklyRowEvents();
  }

  function renderWeeklyPager() {
    var pager = document.getElementById("pwPagination") || document.getElementById("pwPager");
    if (!pager) return;
    if (window.PersonalList && typeof window.PersonalList.renderPagination === "function") {
      window.PersonalList.renderPagination({
        container: pager,
        page: weeklyPager.page,
        pageSize: weeklyPager.pageSize,
        total: weeklyPager.total,
        onPageChange: function (newPage) {
          weeklyPager.page = newPage;
          renderWeeklyRows();
        },
        onPageSizeChange: function (newPageSize) {
          weeklyPager.pageSize = newPageSize;
          weeklyPager.page = 1;
          if (window.PersonalList.savePageSize) {
            window.PersonalList.savePageSize("po.follow.weekly.pageSize", newPageSize);
          }
          renderWeeklyRows();
        }
      });
      return;
    }
    var countText = document.getElementById("pwCountText");
    var total = weeklyItems.length;
    var totalPages = Math.max(1, Math.ceil(total / weeklyPager.pageSize));
    if (weeklyPager.page > totalPages) weeklyPager.page = totalPages;
    var start = total === 0 ? 0 : (weeklyPager.page - 1) * weeklyPager.pageSize + 1;
    var end = Math.min(total, weeklyPager.page * weeklyPager.pageSize);
    if (countText) countText.textContent = "显示 " + start + "-" + end + " / 共 " + total + " 个项目";
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
        if (btn.disabled) return;
        var pid = parseInt(btn.getAttribute("data-unwatch-project"), 10);
        if (confirm("确认取消关注该项目周报？")) { await unwatchItem("project", pid); }
      });
    });
  }

  async function unwatchItem(type, id) {
    try {
      var endpoint = type === "project" ? "/follow/project-report/" + id : "/follow/demand/" + id;
      var fetchFn = window.appFetch || fetch;
      var res = await fetchFn(endpoint, {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json", "Accept": "application/json" },
        body: JSON.stringify({ followed: false })
      });
      if (res.ok) {
        if (typeof window.showToast === "function") window.showToast("已取消关注");
        if (currentTab === "weekly") loadWeeklyData();
        else if (window.FollowDemand) window.FollowDemand.load();
      } else {
        if (typeof window.showToast === "function") window.showToast("取消关注失败", "error");
      }
    } catch (e) {
      if (typeof window.showToast === "function") window.showToast("取消关注失败", "error");
    }
  }

  window.FollowUnwatchDemand = function (id) { return unwatchItem("demand", id); };
  window.FollowSetDemand = async function (id, followed) {
    id = String(id || "").replace(/^US/i, "");
    if (!id) return false;
    try {
      var fetchFn = window.appFetch || fetch;
      var res = await fetchFn("/follow/demand/" + encodeURIComponent(id), {
        method: "PUT",
        credentials: "include",
        headers: { "Content-Type": "application/json", "Accept": "application/json" },
        body: JSON.stringify({ followed: !!followed })
      });
      if (!res.ok) throw new Error("set follow failed");
      if (typeof window.showToast === "function") {
        window.showToast(followed ? "已关注" : "已取消关注", "success");
      }
      return true;
    } catch (e) {
      if (typeof window.showToast === "function") window.showToast(followed ? "关注失败" : "取消关注失败", "error");
      return false;
    }
  };
  window.FollowUpdateDemandBadge = function (n) {
    var dEl = document.getElementById("tabCountDemand");
    if (dEl) dEl.textContent = String(n || 0);
  };

  function updateTabBadges() {
    var dEl = document.getElementById("tabCountDemand");
    var wEl = document.getElementById("tabCountWeekly");
    var dCount = (window.FollowDemand && window.FollowDemand.getTotal) ? window.FollowDemand.getTotal() : 0;
    var wCount = weeklyPager.total || weeklyStats.watched || weeklyItems.length || 0;
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

    document.querySelectorAll("#weeklySection .pw-scope-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        weeklyScope = btn.getAttribute("data-scope") || "mine";
        document.querySelectorAll("#weeklySection .pw-scope-btn").forEach(function (b) {
          var active = (b.getAttribute("data-scope") || "mine") === weeklyScope;
          b.classList.toggle("active", active);
          b.style.background = active ? "var(--color-primary, #2563eb)" : "transparent";
          b.style.color = active ? "#fff" : "var(--color-text-body, #475569)";
        });
        weeklyPager.page = 1;
        loadWeeklyData();
      });
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
        weeklyScope = "mine";
        if (pwSearch) pwSearch.value = "";
        document.querySelectorAll("#weeklySection .pw-scope-btn").forEach(function (b) {
          var active = (b.getAttribute("data-scope") || "mine") === "mine";
          b.classList.toggle("active", active);
          b.style.background = active ? "var(--color-primary, #2563eb)" : "transparent";
          b.style.color = active ? "#fff" : "var(--color-text-body, #475569)";
        });
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
