(function () {
  "use strict";

  var state = {
    tab: "demand",
    scope: "all",
    keyword: "",
    page: 1,
    pageSize: 20,
  };

  function escapeHtml(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function fetchItems() {
    var params = new URLSearchParams();
    params.set("tab", state.tab);
    params.set("scope", state.scope);
    if (state.keyword) {
      params.set("keyword", state.keyword);
    }
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    return fetch("/follow/items?" + params.toString(), { method: "GET" })
      .then(function (r) {
        if (!r.ok) { throw new Error("fetch failed"); }
        return r.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("invalid payload"); }
        return payload;
      });
  }

  function renderItem(item) {
    var priClass = ["P1", "P2", "P3", "P4"].indexOf(item.priority) >= 0
      ? item.priority.toLowerCase() : "normal";
    var tags = "";
    if (item.isKey) { tags += '<span class="follow-key">重点</span> '; }
    if (item.isClosed) { tags += '<span class="follow-closed">已关闭</span> '; }
    return (
      '<div class="follow-item">' +
      '<div class="follow-main">' +
      '<div class="follow-title">' +
      '<span style="color:var(--po-t2);margin-right:6px">' +
      (state.tab === "demand" ? "US" : "P") + escapeHtml(item.id) + '</span>' +
      escapeHtml(item.title) +
      '</div>' +
      '<div class="follow-meta">' +
      '<span class="follow-pri ' + priClass + '">' + escapeHtml(item.priority) + '</span> ' +
      tags +
      '状态: ' + escapeHtml(item.status) + ' · 负责人: ' + escapeHtml(item.owner || "—") +
      (item.latestNote ? ' · 最新周报: ' + escapeHtml(item.latestNote) : '') +
      (item.date ? ' · ' + escapeHtml(item.date) : '') +
      '</div>' +
      '</div>' +
      '<div class="follow-actions">' +
      '<a class="follow-btn" href="' + escapeHtml(item.url || "#") + '" target="_blank" rel="noopener noreferrer">查看</a>' +
      (state.tab === "demand" ? '<button type="button" class="follow-btn" data-action="unfollow" data-id="' + escapeHtml(item.id) + '">取消关注</button>' : '') +
      '</div>' +
      '</div>'
    );
  }

  function renderList(items) {
    var host = document.getElementById("followListHost");
    var empty = document.getElementById("followEmpty");
    if (!items || items.length === 0) {
      host.innerHTML = "";
      host.appendChild(empty);
      empty.hidden = false;
      return;
    }
    empty.hidden = true;
    host.innerHTML = items.map(renderItem).join("");
    bindItemActions();
  }

  function renderPagination(total) {
    var host = document.getElementById("followPagination");
    var pages = Math.max(1, Math.ceil(total / state.pageSize));
    host.hidden = total === 0;
    if (total === 0) { host.innerHTML = ""; return; }
    var start = (state.page - 1) * state.pageSize + 1;
    var end = Math.min(total, state.page * state.pageSize);
    host.innerHTML = "<span>显示 " + start + "–" + end + "，共 " + total + " 个</span>" +
      '<div class="follow-pager-controls"><select class="follow-page-size" aria-label="每页条数"><option value="10">10 条/页</option><option value="15">15 条/页</option><option value="20">20 条/页</option><option value="30">30 条/页</option><option value="50">50 条/页</option></select>' +
      '<button type="button" class="follow-pager-btn" data-page="' + (state.page - 1) + '"' + (state.page === 1 ? " disabled" : "") + '>‹</button><span>第 ' + state.page + " / " + pages + ' 页</span><button type="button" class="follow-pager-btn" data-page="' + (state.page + 1) + '"' + (state.page === pages ? " disabled" : "") + ">›</button></div>";
    host.querySelector(".follow-page-size").value = String(state.pageSize);
  }

  function bindItemActions() {
    var btns = document.querySelectorAll(".follow-btn[data-action]");
    btns.forEach(function (b) {
      b.addEventListener("click", function () {
        var action = b.dataset.action;
        var id = b.dataset.id;
        if (action === "unfollow") {
          b.disabled = true;
          window.appFetch("/follow/demand/" + encodeURIComponent(id), {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ followed: false }),
          })
            .then(function (r) {
              return r.json().then(function (payload) {
                if (!r.ok || !payload || payload.success !== true) {
                  throw new Error((payload && payload.message) || "更新关注失败");
                }
                return payload;
              });
            })
            .then(function (payload) { window.location.href = payload.redirectUrl; })
            .catch(function () { b.disabled = false; });
        }
      });
    });
  }

  function refresh() {
    document.getElementById("followSummary").textContent = "加载中…";
    fetchItems()
      .then(function (payload) {
        document.getElementById("followSummary").textContent =
          (state.tab === "demand" ? "业务需求" : "项目周报") + " · " +
          (payload.items || []).length + " / " + (payload.total || 0) + " 个";
        renderList(payload.items || []);
        renderPagination(payload.total || 0);
      })
      .catch(function () {
        document.getElementById("followSummary").textContent = "加载失败";
        renderList([]);
        document.getElementById("followPagination").hidden = true;
      });
  }

  function bindEvents() {
    document.querySelectorAll(".follow-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".follow-tab").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.tab = btn.dataset.tab;
        state.scope = "all";
        state.page = 1;
        document.querySelectorAll(".scope-chip").forEach(function (chip) {
          chip.classList.toggle("active", chip.dataset.scope === "all");
        });
        document.getElementById("followScope").hidden = state.tab !== "demand";
        refresh();
      });
    });
    document.querySelectorAll(".scope-chip").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".scope-chip").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.scope = btn.dataset.scope;
        state.page = 1;
        refresh();
      });
    });
    var kw = document.getElementById("followKeyword");
    var kwTimer = null;
    kw.addEventListener("input", function () {
      clearTimeout(kwTimer);
      kwTimer = setTimeout(function () {
        state.keyword = kw.value.trim();
        state.page = 1;
        refresh();
      }, 300);
    });
    document.getElementById("followPagination").addEventListener("click", function (event) {
      var button = event.target.closest("[data-page]");
      if (!button || button.disabled) { return; }
      state.page = Number(button.dataset.page);
      refresh();
    });
    document.getElementById("followPagination").addEventListener("change", function (event) {
      if (!event.target.matches(".follow-page-size")) { return; }
      state.pageSize = Number(event.target.value);
      state.page = 1;
      refresh();
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();
