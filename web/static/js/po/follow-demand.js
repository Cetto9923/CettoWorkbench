/**
 * =============================================================================
 * 文件: web/static/js/po/follow-demand.js
 * 模块: PO 工作台 - 我的关注 · 业务需求
 * 职责: 统计卡、关注维度筛选、单行列表（对象#ID + 星标关注/取关 + 标题，复用 PersonalList.idChipHtml）、导出
 * =============================================================================
 */
(function (root) {
  "use strict";

  var state = {
    scope: "all",
    lifecycle: "all",
    keyword: "",
    page: 1,
    pageSize: 20,
    total: 0,
    stats: { all: 0, clarifying: 0, implementing: 0, released: 0, closed: 0, key: 0, keyOpen: 0, openClean: 0 }
  };

  var esc = (root.PersonalList && root.PersonalList.escapeHtml) || function (s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  };
  var PL = root.PersonalList || {};
  if (typeof PL.loadPageSize === "function") {
    state.pageSize = PL.loadPageSize("po.follow.pageSize", state.pageSize, [10, 15, 20, 30, 50]);
  }
  var priorityBadge = PL.priorityBadge || function (raw) {
    var n = parseInt(String(raw || "").replace(/^p/i, ""), 10);
    if (isNaN(n) || n < 1 || n > 4) { return '<span class="wb-priority" data-priority="">—</span>'; }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  };

  function dash(v) {
    var t = String(v == null ? "" : v).trim();
    return t || "—";
  }

  function stageLabel(value) {
    var labels = {
      wait: "已受理", draft: "草稿", active: "已澄清", clarified: "已澄清",
      developing: "研发中", testing: "测试中", waitacceptance: "待验收",
      acceptanced: "已验收", waitdeliver: "待交付", delivered: "已交付",
      released: "已发布", closed: "已关闭", suspended: "已挂起", refuse: "已驳回",
      accept: "已受理", clarify: "澄清中", schedule: "已排期", submittest: "提测",
      publish: "发布", greyverify: "生产验证"
    };
    var key = String(value == null ? "" : value).toLowerCase();
    return labels[key] || (String(value || "").trim() || "—");
  }

  function progressTag(item) {
    var status = String(item.progressStatus || "").toLowerCase();
    var label = dash(item.progressLabel);
    if (status === "delayed") return '<span class="tag red">' + esc(label) + "</span>";
    if (status === "done") return '<span class="tag green">' + esc(label) + "</span>";
    if (status === "normal") return '<span class="tag green">' + esc(label) + "</span>";
    return '<span class="pw-subtle">' + esc(label) + "</span>";
  }

  function keyTimesHtml(item) {
    // 单行列表：只拼有值的关键日期，避免多行撑高
    var parts = [];
    if (String(item.developFinish || "").trim()) parts.push("开发 " + String(item.developFinish).trim());
    if (String(item.testFinish || "").trim()) parts.push("测试 " + String(item.testFinish).trim());
    if (String(item.deadline || "").trim()) parts.push("截止 " + String(item.deadline).trim());
    return '<span class="fd-keytimes-inline" title="' + esc(parts.join(" · ")) + '">' + esc(parts.length ? parts.join(" · ") : "—") + "</span>";
  }

  function setNumText(id, val) {
    var el = document.getElementById(id);
    if (!el) return;
    if (el.firstChild && el.firstChild.nodeType === 3) {
      el.firstChild.textContent = String(val);
    } else {
      var note = el.querySelector(".pw-sum-note");
      el.textContent = String(val);
      if (note) el.appendChild(note);
    }
  }

  function updateStatsUI() {
    var s = state.stats || {};
    setNumText("fdStatClarifying", s.clarifying || 0);
    setNumText("fdStatImplementing", s.implementing || 0);
    setNumText("fdStatReleased", s.released || 0);
    setNumText("fdStatClosed", s.closed || 0);
    var setBtn = function (id, val) { var el = document.getElementById(id); if (el) el.textContent = String(val); };
    setBtn("fdBtnAll", s.all || 0);
    setBtn("fdBtnKey", s.key || 0);
    setBtn("fdBtnKeyOpen", s.keyOpen || 0);
    setBtn("fdBtnClosed", s.closed || 0);
    setBtn("fdBtnOpenClean", s.openClean || 0);
  }

  function syncFilterActive() {
    document.querySelectorAll("#fdSummary .pw-sum-card").forEach(function (c) {
      var lc = c.getAttribute("data-lifecycle") || "all";
      c.classList.toggle("active", state.lifecycle !== "all" && lc === state.lifecycle);
    });
    document.querySelectorAll("#fdToolbar .pw-filter-btn").forEach(function (b) {
      b.classList.toggle("active", (b.getAttribute("data-scope") || "all") === state.scope);
    });
  }

  function renderRows(items) {
    var tbody = document.getElementById("followDemandTbody");
    var empty = document.getElementById("followEmpty");
    var pagEl = document.getElementById("followPagination");
    var countText = document.getElementById("fdCountText");
    if (!tbody) return;

    if (!items.length) {
      tbody.innerHTML = "";
      if (empty) empty.hidden = false;
      if (pagEl) pagEl.hidden = true;
      if (countText) countText.textContent = "显示 0 / 共 0 条";
      return;
    }
    if (empty) empty.hidden = true;

    tbody.innerHTML = items.map(function (item) {
      var did = item.id;
      var displayId = "US" + did;
      var ztUrl = String(item.url || "").trim();
      var idLink = ztUrl
        ? '<a class="table-id-link" href="' + esc(ztUrl) + '" target="_blank" rel="noopener noreferrer" title="在禅道中查看原始详情">' + esc(displayId) + "</a>"
        : '<span class="table-id-link">' + esc(displayId) + "</span>";
      // 与首页价值流同款：PersonalList.idChipHtml（业务需求 + #US…）
      var idChip = (PL.idChipHtml)
        ? PL.idChipHtml("business", idLink)
        : '<span class="wb-type wb-type-business"><span class="wb-type-tag">业务</span><span class="wb-type-id">#' + esc(displayId) + "</span></span>";

      var priTag = priorityBadge(item.priority || item.pri);
      var titleText = item.title || "—";
      var tipParts = [];
      if (item.owner) tipParts.push("负责人：" + String(item.owner));
      if (item.reason) tipParts.push(String(item.reason));
      if (item.isKey) tipParts.push("重点关注");
      var tip = tipParts.length ? tipParts.join(" · ") : titleText;
      // 禅道同款五角星：本页默认已关注（实心），点击切换关注/取关
      var starBtn = '<button type="button" class="fd-watch-toggle is-watched" data-watch-demand="' + esc(did) + '" data-watched="1" title="取消关注" aria-label="取消关注" aria-pressed="true">' +
        '<i class="fas fa-star" aria-hidden="true"></i></button>';
      var titleBtn = '<button type="button" class="table-title-link" data-open-demand="' + esc(did) + '" title="' + esc(tip) + '">' + esc(titleText) + "</button>";
      // 单行：星 + 优先级 + 标题
      var titleHtml = '<div class="home-title-line fd-title-line">' + starBtn + priTag + titleBtn + "</div>";

      var stage = '<span class="stage-tag">' + esc(stageLabel(item.stage || item.status)) + "</span>";
      var schedule = '<span class="fd-schedule-inline" title="' + esc(dash(item.scheduleSummary)) + '">' + esc(dash(item.scheduleSummary)) + "</span>";
      return '<tr data-demand-row="' + esc(did) + '">' +
        '<td class="c-id">' + idChip + "</td>" +
        '<td class="c-title">' + titleHtml + "</td>" +
        "<td>" + stage + "</td>" +
        "<td>" + progressTag(item) + "</td>" +
        "<td>" + keyTimesHtml(item) + "</td>" +
        "<td>" + schedule + "</td>" +
        '<td><div class="pw-actions">' +
        '<button type="button" class="link-btn" data-open-demand="' + esc(did) + '">查看详情</button>' +
        "</div></td></tr>";
    }).join("");

    var start = state.total === 0 ? 0 : (state.page - 1) * state.pageSize + 1;
    var end = Math.min(state.total, state.page * state.pageSize);
    if (countText) countText.textContent = "显示 " + start + "-" + end + " / 共 " + state.total + " 条";

    if (pagEl && PL.renderPagination) {
      pagEl.hidden = false;
      PL.renderPagination({
        container: pagEl,
        page: state.page,
        pageSize: state.pageSize,
        total: state.total,
        onPageChange: function (p) { state.page = p; load(); },
        onPageSizeChange: function (ps) {
          state.pageSize = ps; state.page = 1;
          if (PL.savePageSize) PL.savePageSize("po.follow.pageSize", ps);
          load();
        }
      });
    }
    bindRowEvents();
  }

  function setStarUI(btn, watched) {
    if (!btn) return;
    btn.classList.toggle("is-watched", !!watched);
    btn.setAttribute("data-watched", watched ? "1" : "0");
    btn.setAttribute("title", watched ? "取消关注" : "关注");
    btn.setAttribute("aria-label", watched ? "取消关注" : "关注");
    btn.setAttribute("aria-pressed", watched ? "true" : "false");
    var icon = btn.querySelector("i");
    if (icon) {
      icon.className = watched ? "fas fa-star" : "far fa-star";
    }
  }

  function bindRowEvents() {
    document.querySelectorAll("#demandSection [data-open-demand]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var id = btn.getAttribute("data-open-demand");
        if (root.DemandDetail && typeof root.DemandDetail.open === "function") {
          root.DemandDetail.open(id);
        }
      });
    });
    document.querySelectorAll("#demandSection [data-watch-demand]").forEach(function (btn) {
      btn.addEventListener("click", async function (ev) {
        ev.preventDefault();
        ev.stopPropagation();
        if (btn.disabled) return;
        var id = btn.getAttribute("data-watch-demand");
        var watched = btn.getAttribute("data-watched") === "1";
        var next = !watched;
        btn.disabled = true;
        var ok = false;
        if (typeof root.FollowSetDemand === "function") {
          ok = await root.FollowSetDemand(id, next);
        } else if (!next && typeof root.FollowUnwatchDemand === "function") {
          await root.FollowUnwatchDemand(id);
          ok = true;
        }
        btn.disabled = false;
        if (!ok) return;
        setStarUI(btn, next);
        // 取关后从本页列表移除并刷新统计；再关注仅改星标（本页默认全是已关注）
        if (!next) {
          load();
        }
      });
    });
  }

  async function load() {
    var tbody = document.getElementById("followDemandTbody");
    var empty = document.getElementById("followEmpty");
    var error = document.getElementById("followError");
    var summary = document.getElementById("followSummary");
    if (tbody) tbody.innerHTML = '<tr><td colspan="7" class="pw-empty-row">正在拉取关注业务需求…</td></tr>';
    if (empty) empty.hidden = true;
    if (error) error.hidden = true;
    if (summary) summary.textContent = "关注业务需求 · 加载中…";
    syncFilterActive();

    var params = new URLSearchParams({
      tab: "demand",
      scope: state.scope || "all",
      lifecycle: state.lifecycle || "all",
      keyword: state.keyword || "",
      page: String(state.page || 1),
      pageSize: String(state.pageSize || 20)
    });
    try {
      var res = await fetch("/follow/items?" + params.toString(), {
        credentials: "include",
        headers: { "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }
      });
      if (!res.ok) throw new Error("fetch demand failed");
      var json = await res.json();
      if (!json || !json.success) throw new Error("fetch demand failed");
      var items = Array.isArray(json.items) ? json.items : [];
      state.total = typeof json.total === "number" ? json.total : items.length;
      if (json.stats) state.stats = json.stats;
      updateStatsUI();
      if (summary) summary.textContent = "关注业务需求 · 共 " + (state.stats.all || state.total) + " 条";
      renderRows(items);
      if (typeof root.FollowUpdateDemandBadge === "function") {
        root.FollowUpdateDemandBadge(state.stats.all || state.total);
      }
    } catch (e) {
      if (tbody) tbody.innerHTML = "";
      if (summary) summary.textContent = "加载失败，请重试";
      if (error) error.hidden = false;
    }
  }

  function setScope(scope) {
    state.scope = scope || "all";
    state.page = 1;
    load();
  }

  function setLifecycle(lifecycle) {
    var next = lifecycle || "all";
    // 再次点击同一生命周期卡 → 回到全部
    if (state.lifecycle === next) next = "all";
    state.lifecycle = next;
    state.page = 1;
    load();
  }

  function reset() {
    state.scope = "all";
    state.lifecycle = "all";
    state.keyword = "";
    state.page = 1;
    var input = document.getElementById("followKeyword");
    if (input) input.value = "";
    load();
  }

  function exportCsv() {
    var params = new URLSearchParams({
      tab: "demand",
      scope: state.scope || "all",
      lifecycle: state.lifecycle || "all",
      keyword: state.keyword || ""
    });
    window.location.href = "/follow/demands/export?" + params.toString();
  }

  function bind() {
    document.querySelectorAll("#fdSummary .pw-sum-card").forEach(function (card) {
      card.addEventListener("click", function () {
        setLifecycle(card.getAttribute("data-lifecycle") || "all");
      });
    });
    document.querySelectorAll("#fdToolbar .pw-filter-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        setScope(btn.getAttribute("data-scope") || "all");
      });
    });
    var search = document.getElementById("followKeyword");
    if (search) {
      var timer = null;
      search.addEventListener("input", function () {
        clearTimeout(timer);
        timer = setTimeout(function () {
          state.keyword = search.value.trim();
          state.page = 1;
          load();
        }, 300);
      });
    }
    var resetBtn = document.getElementById("fdResetBtn");
    if (resetBtn) resetBtn.addEventListener("click", reset);
    var exportBtn = document.getElementById("fdExportBtn");
    if (exportBtn) exportBtn.addEventListener("click", exportCsv);
    var retry = document.getElementById("followRetryBtn");
    if (retry) retry.addEventListener("click", load);
  }

  root.FollowDemand = {
    bind: bind,
    load: load,
    getTotal: function () { return state.stats.all || state.total || 0; },
    reset: reset
  };
})(window);
