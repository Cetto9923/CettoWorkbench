(function () {
  "use strict";

  var state = {
    tab: "demand",
    scope: "all",
    keyword: "",
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
      '<span style="color:var(--po-t2);margin-right:6px">US' + escapeHtml(item.id) + '</span>' +
      escapeHtml(item.title) +
      '</div>' +
      '<div class="follow-meta">' +
      '<span class="follow-pri ' + priClass + '">' + escapeHtml(item.priority) + '</span> ' +
      tags +
      '状态: ' + escapeHtml(item.status) + ' · 负责人: ' + escapeHtml(item.owner || "—") +
      '</div>' +
      '</div>' +
      '<div class="follow-actions">' +
      '<button type="button" class="follow-btn" data-action="view" data-id="' + escapeHtml(item.id) + '">查看</button>' +
      '<button type="button" class="follow-btn" data-action="unfollow" data-id="' + escapeHtml(item.id) + '">取消关注</button>' +
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

  function bindItemActions() {
    var btns = document.querySelectorAll(".follow-btn[data-action]");
    btns.forEach(function (b) {
      b.addEventListener("click", function () {
        var action = b.dataset.action;
        var id = b.dataset.id;
        if (action === "unfollow") {
          b.disabled = true;
          fetch("/follow/demand/" + encodeURIComponent(id), {
            method: "POST",
            headers: { "Content-Type": "application/x-www-form-urlencoded" },
            body: "followed=0",
          })
            .then(function (r) { return r.json(); })
            .then(function () { refresh(); })
            .catch(function () { b.disabled = false; });
        } else if (action === "view") {
          // 本期占位，后续接 SSO
          window.open("about:blank", "_blank");
        }
      });
    });
  }

  function refresh() {
    document.getElementById("followSummary").textContent = "加载中…";
    fetchItems()
      .then(function (payload) {
        document.getElementById("followSummary").textContent =
          "业务需求 · " + (payload.items || []).length + " / " + (payload.total || 0) + " 个";
        renderList(payload.items || []);
      })
      .catch(function () {
        document.getElementById("followSummary").textContent = "加载失败";
        renderList([]);
      });
  }

  function bindEvents() {
    document.querySelectorAll(".follow-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".follow-tab").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.tab = btn.dataset.tab;
        refresh();
      });
    });
    document.querySelectorAll(".scope-chip").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".scope-chip").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.scope = btn.dataset.scope;
        refresh();
      });
    });
    var kw = document.getElementById("followKeyword");
    var kwTimer = null;
    kw.addEventListener("input", function () {
      clearTimeout(kwTimer);
      kwTimer = setTimeout(function () {
        state.keyword = kw.value.trim();
        refresh();
      }, 300);
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();