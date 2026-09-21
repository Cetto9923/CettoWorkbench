/**
 * 敏捷小组列表 / 详情 / 确认 Drawer
 * API: /workbench/api/agile-teams
 */
(function () {
  const API = "/workbench/api/agile-teams";
  const AT_PAGE_SIZE_PRESETS = [20, 50, 100];
  const AT_PAGE_SIZE_MIN = 5;
  const AT_PAGE_SIZE_MAX = 200;
  const AT_PAGE_SIZE_CUSTOM_KEY = "wb:agileteam:pageSizeCustom";
  const AT_PAGE_SIZE_CUSTOM_SENTINEL = -1; // <select> 里 "自定义…" 的 value

  function readStoredCustomPageSize() {
    try {
      const raw = window.localStorage.getItem(AT_PAGE_SIZE_CUSTOM_KEY);
      if (raw == null) return 0;
      const n = Number(String(raw).trim());
      if (!Number.isFinite(n)) return 0;
      if (n < AT_PAGE_SIZE_MIN || n > AT_PAGE_SIZE_MAX) return 0;
      return Math.round(n);
    } catch (e) {
      return 0;
    }
  }

  function writeStoredCustomPageSize(n) {
    try {
      window.localStorage.setItem(AT_PAGE_SIZE_CUSTOM_KEY, String(n));
    } catch (e) {
      // localStorage 不可用（隐私模式 / 禁用）静默吞，不影响核心流程。
    }
  }

  function clampPageSize(n) {
    const v = Math.round(Number(n) || 0);
    if (v < AT_PAGE_SIZE_MIN) return AT_PAGE_SIZE_MIN;
    if (v > AT_PAGE_SIZE_MAX) return AT_PAGE_SIZE_MAX;
    return v;
  }

  let state = {
    status: "all",
    adjustStatus: "all",
    filters: {},
    detailId: 0,
    detailTab: "basic",
    canConfirm: false,
    canEdit: false,
    page: 1,
    pageSize: 20,
    pageSizeCustom: readStoredCustomPageSize(), // 0 表示未存
    total: 0,
    pageCount: 1,
    view: currentView(),
    scope: "team",
    scopeId: 0,
    collapsed: {},
  };

  // 启动时如果有持久化的自定义值且不是预设档，自动套用。
  if (state.pageSizeCustom > 0 && AT_PAGE_SIZE_PRESETS.indexOf(state.pageSizeCustom) < 0) {
    state.pageSize = state.pageSizeCustom;
  }

  function currentView() {
    const el = document.querySelector('meta[name="wb-role"]');
    const role = el ? String(el.getAttribute("content") || "").toLowerCase() : "";
    return role === "lead" ? "lead" : "pmo";
  }

  function isLeadView() {
    return state.view === "lead";
  }

  var esc = window.escapeHtml;

  async function apiFetch(path, opts) {
    opts = opts || {};
    const headers = Object.assign({ "Content-Type": "application/json", "X-Requested-With": "XMLHttpRequest" }, opts.headers || {});
    const csrf = typeof window.getCsrfToken === "function" ? window.getCsrfToken() : "";
    if (csrf && !headers["X-CSRF-Token"]) headers["X-CSRF-Token"] = csrf;
    const res = await fetch(path, {
      method: opts.method || "GET",
      credentials: "include",
      headers: headers,
      body: opts.body ? JSON.stringify(opts.body) : undefined,
    });
    const json = await res.json().catch(function () { return null; });
    if (!res.ok) {
      const msg = (json && (json.message || (json.errors && json.errors[0] && json.errors[0].message))) || ("HTTP " + res.status);
      const err = new Error(msg);
      err.status = res.status;
      err.body = json;
      throw err;
    }
    return json;
  }

  function showPage(id) {
    document.querySelectorAll(".page").forEach(function (p) { p.classList.remove("active"); });
    document.querySelectorAll(".wb-side button, .sidebar [data-page]").forEach(function (b) {
      b.classList.toggle("active", b.dataset.page === id);
    });
    const el = document.getElementById("page-" + id);
    if (el) el.classList.add("active");
    try { history.replaceState(null, "", "#" + id); } catch (e) {}
  }

  window.atShowList = function () {
    showPage("agileteam");
    loadList();
  };

  window.atShowDetail = function (id) {
    state.detailId = Number(id) || 0;
    state.detailTab = "basic";
    showPage("agileteam-detail");
    if (typeof window.atLoadDetail === "function") window.atLoadDetail(state.detailId);
  };

  window.atSetStatus = function (st) {
    state.status = st || "all";
    state.page = 1;
    loadList();
  };

  window.atFilterPending = function () {
    state.adjustStatus = "pending";
    state.status = "all";
    state.page = 1;
    loadList();
  };

  window.atSearch = function () {
    state.filters = {
      name: val("atFilterName"),
      parentName: val("atFilterParent"),
      coach: val("atFilterCoach"),
      po: val("atFilterPO"),
      member: val("atFilterMember"),
      adjustStatus: val("atFilterAdjust") || "all",
    };
    state.adjustStatus = state.filters.adjustStatus;
    state.page = 1;
    loadList();
  };

  window.atReset = function () {
    ["atFilterName", "atFilterParent", "atFilterCoach", "atFilterPO", "atFilterMember"].forEach(function (id) {
      const el = document.getElementById(id);
      if (el) el.value = "";
    });
    const adj = document.getElementById("atFilterAdjust");
    if (adj) adj.value = "all";
    state.filters = {};
    state.adjustStatus = "all";
    state.status = "all";
    state.page = 1;
    loadList();
  };

  function val(id) {
    const el = document.getElementById(id);
    return el ? String(el.value || "").trim() : "";
  }

  async function loadList() {
    const host = document.getElementById("atListBody");
    const summary = document.getElementById("atSummaryStrip");
    if (!host) return;
    host.innerHTML = '<tr><td colspan="9" class="at-empty">加载中…</td></tr>';
    const q = new URLSearchParams();
    q.set("status", state.status);
    q.set("adjustStatus", state.adjustStatus || "all");
    q.set("page", String(state.page || 1));
    q.set("pageSize", String(state.pageSize || 20));
    q.set("view", state.view || "pmo");
    if (isLeadView()) {
      q.set("scope", state.scope || "team");
      if (state.scopeId) q.set("scopeId", String(state.scopeId));
    }
    const f = state.filters || {};
    ["name", "parentName", "coach", "po", "member"].forEach(function (k) {
      if (f[k]) q.set(k, f[k]);
    });
    try {
      const json = await apiFetch(API + "?" + q.toString());
      const data = (json && json.data) || {};
      state.canEdit = !!data.canEdit && !isLeadView();
      state.total = data.total || 0;
      state.page = data.page || state.page;
      state.pageSize = data.pageSize || state.pageSize;
      state.pageCount = data.pageCount || 1;
      renderScope(data.scopeOptions || []);
      renderSummary(summary, data);
      renderRows(host, data.items || []);
      renderPager();
    } catch (e) {
      host.innerHTML = '<tr><td colspan="9" class="at-empty">' + esc(e.message || "加载失败") + "</td></tr>";
    }
  }

  function renderSummary(host, data) {
    if (!host) return;
    const st = state.status;
    host.innerHTML =
      '<div class="at-seg">' +
      segBtn("all", "全部", data.allCount, st) +
      segBtn("enable", "启用", data.enableCount, st) +
      segBtn("disable", "停用", data.disableCount, st) +
      "</div>" +
      '<span class="at-pending-chip" onclick="atFilterPending()">待确认调整 ' + esc(data.pendingCount || 0) + "</span>";
  }

  function segBtn(key, label, count, active) {
    return (
      '<button type="button" class="at-seg-btn' + (active === key ? " active" : "") +
      '" onclick="atSetStatus(\'' + key + '\')">' + label +
      ' <span class="count">' + esc(count || 0) + "</span></button>"
    );
  }

  function renderRows(host, items) {
    if (!items.length) {
      host.innerHTML = '<tr><td colspan="9" class="at-empty">暂无敏捷小组</td></tr>';
      return;
    }
    host.innerHTML = items.map(function (it) {
      const isChild = it.type === "child" || (it.parentId && it.parentId !== 0);
      const isParent = !isChild;
      const collapsed = isChild && state.collapsed[it.parentId];
      const rowClass = (isParent ? "at-parent-row" : "at-child-row") + (collapsed ? " hidden" : "");
      const fold = isParent && it.childCount
        ? '<button type="button" class="at-fold" onclick="atToggleChildren(' + it.id + ',this)">' + (state.collapsed[it.id] ? "›" : "⌄") + "</button>"
        : '<button type="button" class="at-fold blank">⌄</button>';
      const nameCell = isChild
        ? '<td class="at-child-indent"><span class="at-child-badge">子</span><span class="at-team-name" onclick="atShowDetail(' + it.id + ')">' + esc(it.name) +
          '</span><div class="at-sub-id" style="margin-left:28px">#' + esc(it.id) + "</div></td>"
        : '<td><div class="at-team-cell">' + fold + "<div><div class=\"at-team-name\" onclick=\"atShowDetail(" + it.id + ')\">' + esc(it.name) +
          '</div><div class="at-sub-id">#' + esc(it.id) + (it.childCount ? " · 父级小组" : "") + "</div></div></div></td>";
      const adjust =
        it.pendingAdd || it.pendingRemove
          ? '<span class="at-tag pending" onclick="atOpenReview(' + it.pendingAdjustId + ')">' +
            adjustLabel(it.pendingAdd, it.pendingRemove) + "</span>"
          : "—";
      return (
        '<tr class="' + rowClass + '" data-id="' + it.id + '" data-parent="' + (it.parentId || "") + '">' +
        nameCell +
        "<td>" + esc(it.parentName || "—") + "</td>" +
        "<td>" + esc(person(it.coachName, it.coachAccount)) + "</td>" +
        "<td>" + esc(person(it.poName, it.poAccount)) + "</td>" +
        "<td>" + esc(it.formalCount) + "</td>" +
        "<td>" + adjust + "</td>" +
        '<td><span class="at-tag enabled">' + esc(it.statusLabel || "启用") + "</span></td>" +
        "<td>" + esc(it.lastAdjustAt || "—") + "</td>" +
        '<td><button type="button" class="at-link" onclick="atShowDetail(' + it.id + ')">查看</button></td>' +
        "</tr>"
      );
    }).join("");
  }

  window.atToggleChildren = function (parentId, btn) {
    state.collapsed[parentId] = !state.collapsed[parentId];
    document.querySelectorAll('#atListBody tr[data-parent="' + parentId + '"]').forEach(function (tr) {
      tr.classList.toggle("hidden", !!state.collapsed[parentId]);
    });
    if (btn) btn.textContent = state.collapsed[parentId] ? "›" : "⌄";
  };

  function renderPager() {
    const host = document.getElementById("atPager");
    if (!host) return;
    const total = state.total || 0;
    const page = state.page || 1;
    const pages = state.pageCount || 1;
    let nums = "";
    const start = Math.max(1, page - 2);
    const end = Math.min(pages, start + 4);
    for (let i = start; i <= end; i++) {
      nums += '<button type="button" class="at-page-btn' + (i === page ? " active" : "") + '" onclick="atGoPage(' + i + ')">' + i + "</button>";
    }

    const ps = Number(state.pageSize) || AT_PAGE_SIZE_PRESETS[0];
    const isPreset = AT_PAGE_SIZE_PRESETS.indexOf(ps) >= 0;
    const customActive = !isPreset && ps >= AT_PAGE_SIZE_MIN && ps <= AT_PAGE_SIZE_MAX;

    const selectOptions = AT_PAGE_SIZE_PRESETS.map(function (n) {
      return '<option value="' + n + '"' + (ps === n ? " selected" : "") + ">" + n + " 条/页</option>";
    }).join("") +
      '<option value="' + AT_PAGE_SIZE_CUSTOM_SENTINEL + '"' +
      (customActive ? " selected" : "") + ">自定义…</option>";

    const customValue = customActive ? ps : (state.pageSizeCustom || AT_PAGE_SIZE_PRESETS[0]);
    const customRow = customActive
      ? '<span class="at-page-size-custom">' +
        '<input id="atPageSizeCustomInput" type="number" min="' + AT_PAGE_SIZE_MIN +
        '" max="' + AT_PAGE_SIZE_MAX + '" step="1" value="' + esc(customValue) +
        '" class="at-page-size-custom-input" ' +
        'onkeydown="if(event.key===\'Enter\'){event.preventDefault();atCommitCustomPageSize();}" />' +
        '<button type="button" class="at-btn small" onclick="atCommitCustomPageSize()">应用</button>' +
        '<span class="at-page-size-custom-tip">范围 ' + AT_PAGE_SIZE_MIN + "~" + AT_PAGE_SIZE_MAX + "</span>" +
        "</span>"
      : "";

    host.innerHTML =
      '<div class="at-page-info">共 ' + esc(total) + " 条，第 " + esc(page) + " / " + esc(pages) + " 页</div>" +
      '<button type="button" class="at-page-btn" onclick="atGoPage(' + (page - 1) + ')"' + (page <= 1 ? " disabled" : "") + ">‹</button>" +
      nums +
      '<button type="button" class="at-page-btn" onclick="atGoPage(' + (page + 1) + ')"' + (page >= pages ? " disabled" : "") + ">›</button>" +
      '<select id="atPageSizeSelect" class="at-page-size" onchange="atSetPageSize(this.value)">' +
      selectOptions +
      "</select>" +
      customRow;
  }

  window.atGoPage = function (p) {
    const page = Number(p) || 1;
    if (page < 1 || page > (state.pageCount || 1) || page === state.page) return;
    state.page = page;
    loadList();
  };

  window.atSetPageSize = function (n) {
    const v = Number(n);
    if (v === AT_PAGE_SIZE_CUSTOM_SENTINEL) {
      // 切到自定义档：保留之前的自定义值（如果有），刷新分页露出输入框。
      renderPager();
      // 把输入框聚焦，等用户输入。
      const input = document.getElementById("atPageSizeCustomInput");
      if (input) {
        input.focus();
        try { input.select(); } catch (e) {}
      }
      return;
    }
    if (!v || AT_PAGE_SIZE_PRESETS.indexOf(v) < 0) {
      // 兜底：未知值按 20 处理。
      state.pageSize = 20;
    } else {
      state.pageSize = v;
    }
    state.pageSizeCustom = 0; // 切回预设时清掉持久化的自定义值
    state.page = 1;
    loadList();
  };

  window.atCommitCustomPageSize = function () {
    const input = document.getElementById("atPageSizeCustomInput");
    if (!input) return;
    const raw = String(input.value || "").trim();
    const n = Number(raw);
    if (!Number.isFinite(n) || n <= 0) {
      window.showToast("请输入合法的每页条数", "error");
      try { input.focus(); } catch (e) {}
      return;
    }
    const clamped = clampPageSize(n);
    if (clamped !== n) {
      const note = "每页条数已自动夹逼到 " + AT_PAGE_SIZE_MIN + "~" + AT_PAGE_SIZE_MAX + " 范围：" + clamped;
      window.showToast(note, "info");
    }
    state.pageSize = clamped;
    state.pageSizeCustom = clamped;
    writeStoredCustomPageSize(clamped);
    state.page = 1;
    loadList();
  };

  function renderScope(options) {
    const card = document.getElementById("atScopeCard");
    if (!card) return;
    if (!isLeadView()) {
      card.style.display = "none";
      return;
    }
    card.style.display = "flex";
    const sel = document.getElementById("atScopeSelect");
    if (sel) {
      if (!state.scopeReady && !state.scopeId && options.length) {
        state.scopeReady = true;
        state.scopeId = Number(options[0].id) || 0;
        loadList();
        return;
      }
      state.scopeReady = true;
      const cur = String(state.scopeId || "");
      sel.innerHTML = '<option value="">全部可见团队</option>' + (options || []).map(function (o) {
        return '<option value="' + esc(o.id) + '"' + (String(o.id) === cur ? " selected" : "") + ">" + esc(o.name) + "</option>";
      }).join("");
    }
    card.querySelectorAll(".at-scope-tab").forEach(function (b) {
      b.classList.toggle("active", b.getAttribute("data-scope") === state.scope);
    });
  }

  window.atSetScope = function (kind) {
    state.scope = kind === "dept" ? "dept" : "team";
    state.page = 1;
    loadList();
  };

  window.atSetScopeId = function (id) {
    state.scopeId = Number(id) || 0;
    state.page = 1;
    loadList();
  };

  function adjustLabel(add, remove) {
    const parts = [];
    if (add) parts.push("+" + add);
    if (remove) parts.push("-" + remove);
    return parts.join(" / ") + " 待确认";
  }

  function person(name, account) {
    const n = String(name || "").trim();
    const a = String(account || "").trim();
    if (!n && !a) return "—";
    if (!a) return n;
    if (!n || n === a) return a;
    if (n.indexOf("(" + a + ")") >= 0) return n;
    return n + "(" + a + ")";
  }


  window.__at = {
    API: API,
    state: state,
    esc: esc,
    val: val,
    apiFetch: apiFetch,
    isLeadView: isLeadView,
    person: person,
    loadList: loadList
  };


  function boot() {
    if (!document.getElementById("page-agileteam")) return;
    const hash = (location.hash || "").replace("#", "");
    if (hash && hash.startsWith("detail-")) {
      const id = Number(hash.replace("detail-", "")) || 0;
      if (id > 0) {
        atShowDetail(id);
        return;
      }
    }
    atShowList();
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", boot);
  else boot();
})();
