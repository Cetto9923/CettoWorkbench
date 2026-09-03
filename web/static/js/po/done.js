(function () {
  "use strict";

  var state = {
    tab: "all",
    timeRange: "all",
    objectType: "",
    keyword: "",
    result: "all",
    page: 1,
    pageSize: 15,
  };

  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function queryItems() {
    var params = new URLSearchParams();
    params.set("tab", state.tab);
    params.set("timeRange", state.timeRange);
    if (state.objectType) { params.set("objectType", state.objectType); }
    if (state.keyword) { params.set("keyword", state.keyword); }
        if (state.result !== "all") { params.set("result", state.result); }
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    return fetch("/done/items?" + params.toString(), { method: "GET" })
      .then(function (response) {
        if (!response.ok) { throw new Error("request failed"); }
        return response.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) { throw new Error("invalid payload"); }
        return payload;
      });
  }

  function renderRow(item) {
    var displayId = item.displayId || (item.objectType + "/" + item.objectId);
    var idCell = item.url
      ? '<a class="row-id-link" href="' + escapeHtml(item.url) + '" title="在禅道中查看">' + escapeHtml(displayId) + "</a>"
      : escapeHtml(displayId);
    var titleCell = item.url
      ? '<a class="row-title-link" href="' + escapeHtml(item.url) + '" title="在禅道中查看">' + escapeHtml(item.objectName || "—") + "</a>"
      : escapeHtml(item.objectName || "—");
    var resultLabels = { activated: "已激活", done: "处理成功", submitted: "已提交", approved: "已通过", rejected: "已驳回", closed: "已关闭", verified: "已验收", resolved: "已解决", returned: "已退回" };
    var result = resultLabels[item.result] || "已处理";
    var resultTone = ["approved", "done", "verified", "resolved"].indexOf(item.result) >= 0 ? " success" : (["rejected", "returned"].indexOf(item.result) >= 0 ? " danger" : "");
    return "<tr>" +
      '<td class="c-item" title="' + escapeHtml(item.objectName || "") + '"><div class="done-item-title">' + titleCell + '</div><div class="done-item-id">' + idCell + "</div></td>" +
      '<td class="c-type"><span class="type-tag">' + escapeHtml(item.objectTypeLabel || item.objectType || "—") + "</span></td>" +
      '<td class="c-action">' + escapeHtml(item.action || "已处理") + "</td>" +
      '<td class="c-result"><span class="result-tag' + resultTone + '">' + escapeHtml(result) + "</span></td>" +
      '<td class="c-dead">' + escapeHtml(item.date || "—") + "</td>" +
      '<td class="c-op">' + (item.url ? '<a class="todo-action" href="' + escapeHtml(item.url) + '">查看</a>' : "—") + "</td>" +
      "</tr>";
  }

  function renderSummary(summary) {
    var map = { countRangeAll: "all", countRangeToday: "today", countRange7d: "last7d", countRangeWeek: "week", countRange30d: "last30d", countRangeMonth: "month", countRangeQuarter: "quarter" };
    Object.keys(map).forEach(function (id) {
      var el = document.getElementById(id);
      if (el) { el.textContent = (summary && summary[map[id]]) || 0; }
    });
  }

  function renderList(payload) {
    var items = payload.items || [];
    document.getElementById("doneTbody").innerHTML = items.map(renderRow).join("");
    document.getElementById("doneEmpty").hidden = items.length !== 0;
    document.getElementById("doneSummary").textContent = "";
    renderSummary(payload.summary);
    renderPagination(payload.total || 0);
  }

  function renderPagination(total) {
    var host = document.getElementById("donePagination");
    var pages = Math.max(1, Math.ceil(total / state.pageSize));
    host.hidden = total === 0;
    if (total === 0) { host.innerHTML = ""; return; }
    var start = (state.page - 1) * state.pageSize + 1;
    var end = Math.min(total, state.page * state.pageSize);
    var controls = '<button class="pager-btn" data-page="' + (state.page - 1) + '"' + (state.page === 1 ? " disabled" : "") + ">‹</button>";
    for (var page = 1; page <= pages; page += 1) {
      if (pages > 7 && page > 2 && page < pages - 1 && Math.abs(page - state.page) > 1) { if (page === 3 || page === pages - 2) { controls += "<span>…</span>"; } continue; }
      controls += '<button class="pager-btn' + (page === state.page ? " active" : "") + '" data-page="' + page + '">' + page + "</button>";
    }
    controls += '<button class="pager-btn" data-page="' + (state.page + 1) + '"' + (state.page === pages ? " disabled" : "") + ">›</button>";
    host.innerHTML = "<span>显示 " + start + "–" + end + "，共 " + total + ' 条</span><div class="pager-controls"><select class="pager-page-size" aria-label="每页条数"><option value="10">10 条/页</option><option value="15">15 条/页</option><option value="20">20 条/页</option><option value="30">30 条/页</option><option value="50">50 条/页</option></select>' + controls + "</div>";
    host.querySelector(".pager-page-size").value = String(state.pageSize);
  }

  function refresh() {
    document.getElementById("doneSummary").textContent = "加载中…";
    queryItems().then(renderList).catch(function () {
      document.getElementById("doneSummary").textContent = "加载失败，请稍后重试";
      document.getElementById("doneTbody").innerHTML = "";
      document.getElementById("doneEmpty").hidden = false;
    });
  }

  function activate(selector, current) {
    document.querySelectorAll(selector).forEach(function (button) { button.classList.toggle("active", button === current); });
  }

  function bindEvents() {
    document.querySelectorAll("[data-range]").forEach(function (button) {
      button.addEventListener("click", function () {
        activate("[data-range]", button);
        state.timeRange = button.dataset.range;
        state.page = 1;
        refresh();
      });
    });
    document.querySelectorAll(".group-tab").forEach(function (button) {
      button.addEventListener("click", function () {
        activate(".group-tab", button);
		state.tab = button.dataset.tab || "all";
        state.objectType = button.dataset.object || "";
        document.getElementById("doneObjectType").value = state.objectType;
        document.getElementById("doneMoreType").value = "";
        state.page = 1;
        refresh();
      });
    });
    var keyword = document.getElementById("doneKeyword"); var timer;
    keyword.addEventListener("input", function () { clearTimeout(timer); timer = setTimeout(function () { state.keyword = keyword.value.trim(); state.page = 1; refresh(); }, 300); });
    var fields = { doneObjectType: "objectType", doneResult: "result" };
    Object.keys(fields).forEach(function (id) {
      document.getElementById(id).addEventListener("change", function (event) {
        state[fields[id]] = event.target.value; state.page = 1;
        if (id === "doneObjectType") {
          state.tab = "all";
          document.querySelectorAll(".group-tab").forEach(function (button) { button.classList.toggle("active", !state.objectType && button.dataset.tab === "all" && !button.dataset.object); });
          document.getElementById("doneMoreType").value = "";
        }
        refresh();
      });
    });
    document.getElementById("doneMoreType").addEventListener("change", function (event) {
      if (!event.target.value) { return; }
      document.querySelectorAll(".group-tab").forEach(function (button) { button.classList.remove("active"); });
      state.tab = "all"; state.objectType = event.target.value; state.page = 1;
      document.getElementById("doneObjectType").value = state.objectType;
      refresh();
    });
    document.getElementById("doneResetBtn").addEventListener("click", function () {
      state.timeRange = "all"; state.tab = "all"; state.objectType = ""; state.keyword = ""; state.result = "all"; state.page = 1;
      document.querySelectorAll("[data-range]").forEach(function (b) { b.classList.toggle("active", b.dataset.range === "all"); });
      document.querySelectorAll(".group-tab").forEach(function (b) { b.classList.remove("active"); });
      document.querySelectorAll(".group-tab").forEach(function (b) { if (b.dataset.tab === "all" && !b.dataset.object) b.classList.add("active"); });
      
      document.getElementById("doneResult").value = "all";
      document.getElementById("doneObjectType").value = "";
      document.getElementById("doneMoreType").value = "";
      keyword.value = "";
      refresh();
    });
    document.getElementById("donePagination").addEventListener("click", function (event) { var btn = event.target.closest("[data-page]"); if (!btn || btn.disabled) { return; } state.page = Number(btn.dataset.page); refresh(); });
    document.getElementById("donePagination").addEventListener("change", function (event) { if (!event.target.matches(".pager-page-size")) { return; } state.pageSize = Number(event.target.value); state.page = 1; refresh(); });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();
