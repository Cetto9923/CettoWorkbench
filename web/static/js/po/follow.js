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
  var PAGE_SIZE_OPTIONS = PL.PAGE_SIZE_OPTIONS;
  var weeklyPager = {
    page: 1,
    pageSize: (typeof PL.loadPageSize === "function") ? PL.loadPageSize("po.follow.weekly.pageSize", 10, PAGE_SIZE_OPTIONS) : 10,
    total: 0
  };

  var esc = window.escapeHtml;

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
    var quickChips = document.getElementById("followQuickChips");
    if (quickChips) quickChips.hidden = (tab !== "demand");
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
      var isFallback = false;
      if (!res.ok && res.status === 422 && (weeklyScope === "mine" || weeklyScope === "participated")) {
        // 向下兼容：后端未重启生效时，降级取 all 数据并在前端按参与/关注过滤
        var fallbackParams = new URLSearchParams({
          filter: weeklyFilter || "all",
          keyword: weeklyKeyword || "",
          limit: "500",
          scope: "all"
        });
        res = await fetch("/follow/project-weeklies?" + fallbackParams.toString(), {
          credentials: "include",
          headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
        });
        isFallback = true;
      }
      if (!res.ok) throw new Error("fetch weeklies failed");
      var json = await res.json();
      if (!json || !json.success || !json.data) throw new Error("invalid weeklies payload");
      weeklyItems = Array.isArray(json.data.items) ? json.data.items : [];
      weeklyStats = json.data.stats || weeklyStats;

      if (isFallback) {
        var curAccount = (document.getElementById("currentUserAccount") ? document.getElementById("currentUserAccount").value.trim() : "");
        if (weeklyScope === "participated") {
          weeklyItems = weeklyItems.filter(function (it) {
            return it.isParticipated || it.source === "participated" || it.source === "both" ||
              (curAccount && (it.pmAccount === curAccount || it.pm === curAccount));
          });
        } else if (weeklyScope === "mine") {
          weeklyItems = weeklyItems.filter(function (it) {
            return it.isWatched || it.source === "watched" || it.source === "both" ||
              it.isParticipated || it.source === "participated" ||
              (curAccount && (it.pmAccount === curAccount || it.pm === curAccount));
          });
        }
        weeklyStats = {
          watched: weeklyItems.length,
          submitted: weeklyItems.filter(function (it) { return it.submitStatus === "submitted"; }).length,
          waiting: weeklyItems.filter(function (it) { return it.submitStatus !== "submitted"; }).length,
          abnormal: weeklyItems.filter(function (it) { return it.hasAbnormal; }).length,
          risk: weeklyItems.filter(function (it) { return (it.openIssueCount > 0 || it.openRiskCount > 0); }).length,
          deviation: weeklyItems.filter(function (it) { return (it.planDeviationDays > 0 || (it.releaseRisk && it.releaseRisk !== "normal")); }).length
        };
      }

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
    return window.PersonalList.statusTagHtml(label);
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
      var pmName = String(item.pmName || item.pmAccount || "—").trim();
      var pmDisplay = esc(pmName);
      if (item.pmAccount && pmName.indexOf(item.pmAccount) === -1) {
        pmDisplay += " (" + esc(item.pmAccount) + ")";
      }

      var sourceBadge = "";
      if (item.isParticipated || item.source === "participated" || item.source === "both") {
        sourceBadge = '<span class="tag blue" style="margin-left:6px;font-size:11px;">参与</span>';
      }

      var isWatched = !!(item.isWatched || item.source === "watched" || item.source === "both");
      var watchBtn = "";
      if (isWatched) {
        watchBtn = '<button type="button" class="pw-action-btn pw-watch-btn is-watched" data-unwatch-project="' + pid + '" data-watched="1" title="取消关注" aria-label="取消关注" aria-pressed="true">' +
          '<i class="fas fa-star" aria-hidden="true"></i></button>';
      } else {
        watchBtn = '<button type="button" class="pw-action-btn pw-watch-btn" data-unwatch-project="' + pid + '" data-watched="0" disabled title="参与项目请在禅道团队管理，不在此取关" aria-label="参与项目" style="opacity:0.35;cursor:not-allowed;">' +
          '<i class="far fa-star" aria-hidden="true"></i></button>';
      }

      return "<tr>" +
        '<td><div class="pw-code">' + esc(item.projectCode || "P" + pid) + '</div><div class="pw-name" data-open-detail="' + pid + '">' + esc(item.projectName || "—") + sourceBadge + '</div><div class="pw-meta">项目经理：' + pmDisplay + '</div></td>' +
        '<td><div class="pw-period">第 ' + (item.weekSN || "—") + ' 周</div><div class="pw-subtle">' + esc(item.weekStart) + ' ~ ' + esc(item.weekEnd) + '</div>' + submitStatusHtml(item.submitStatus) + '</td>' +
        '<td><div class="pw-situation">' + situationTag(item) + '</div>' + desc + '</td>' +
        '<td><div class="pw-progress"><div class="pw-pbox"><b>' + (item.finishedCount || 0) + '</b><span>完成</span></div><div class="pw-pbox"><b>' + (item.unfinishedCount || 0) + '</b><span>未完成</span></div><div class="pw-pbox"><b>' + (item.nextWeekCount || 0) + '</b><span>下周</span></div></div></td>' +
        '<td>' + riskLineHtml(item) + '</td>' +
        '<td><div class="pw-actions">' +
        '<button type="button" class="pw-action-btn" data-open-detail="' + pid + '" title="查看周报" aria-label="查看周报"><i class="fas fa-file-lines" aria-hidden="true"></i></button>' +
        '<button type="button" class="pw-action-btn" data-open-history="' + pid + '" title="历史记录" aria-label="历史记录"><i class="fas fa-clock-rotate-left" aria-hidden="true"></i></button>' +
        watchBtn +
        '</div></td>' +
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
        window.showToast("已取消关注");
        if (currentTab === "weekly") loadWeeklyData();
        else if (window.FollowDemand) window.FollowDemand.load();
      } else {
        window.showToast("取消关注失败", "error");
      }
    } catch (e) {
      window.showToast("取消关注失败", "error");
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
      window.showToast(followed ? "已关注" : "已取消关注", "success");
      return true;
    } catch (e) {
      window.showToast(followed ? "关注失败" : "取消关注失败", "error");
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
    }
    var sp = new URLSearchParams(window.location.search);
    switchTab(sp.get("tab") || "demand");
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
