(function () {
  "use strict";

  var state = {
    timeRange: "all",
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
    params.set("timeRange", state.timeRange);
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    return fetch("/done/items?" + params.toString(), { method: "GET" })
      .then(function (r) {
        if (!r.ok) {
          throw new Error("fetch failed");
        }
        return r.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) {
          throw new Error("invalid payload");
        }
        return payload;
      });
  }

  function renderRow(item) {
    var objectCell = item.url
      ? '<a href="' + escapeHtml(item.url) + '" title="在禅道中查看">' +
        escapeHtml(item.objectType) + "/" + escapeHtml(String(item.objectId)) + "</a>"
      : escapeHtml(item.objectType) + "/" + escapeHtml(String(item.objectId));
    var nameCell = item.url
      ? '<a href="' + escapeHtml(item.url) + '" title="在禅道中查看">' +
        escapeHtml(item.objectName || "—") + "</a>"
      : escapeHtml(item.objectName || "—");
    return (
      "<tr>" +
      '<td class="c-when">' + escapeHtml(item.date) + "</td>" +
      '<td class="c-actor">' + escapeHtml(item.actor) + "</td>" +
      '<td class="c-action"><span class="action-tag">' +
      escapeHtml(item.action) +
      "</span></td>" +
      '<td class="c-object" title="' + escapeHtml(item.objectName) + '">' +
      nameCell +
      "</td>" +
      '<td class="c-id">' + objectCell + "</td>" +
      "</tr>"
    );
  }

  function renderList(items) {
    var tbody = document.getElementById("doneTbody");
    var empty = document.getElementById("doneEmpty");
    if (!items || items.length === 0) {
      tbody.innerHTML = "";
      empty.hidden = false;
      return;
    }
    empty.hidden = true;
    tbody.innerHTML = items.map(renderRow).join("");
  }

  function refresh() {
    document.getElementById("doneSummary").textContent = "加载中…";
    fetchItems()
      .then(function (payload) {
        document.getElementById("doneSummary").textContent =
          "已办 · " + (payload.total || 0) + " 条";
        renderList(payload.items || []);
      })
      .catch(function () {
        document.getElementById("doneSummary").textContent = "加载失败";
        renderList([]);
      });
  }

  function bindEvents() {
    var chips = document.querySelectorAll(".time-chip");
    chips.forEach(function (btn) {
      btn.addEventListener("click", function () {
        chips.forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.timeRange = btn.dataset.range;
        state.page = 1;
        refresh();
      });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();
