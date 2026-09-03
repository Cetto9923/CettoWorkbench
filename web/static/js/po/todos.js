(function () {
  "use strict";

  var state = { tab: "all", focus: "pending", action: "all", stage: "all", objectType: "all", relation: "all", responsibility: "all", keyword: "", page: 1, pageSize: 15 };

  function escapeHtml(value) {
    return String(value == null ? "" : value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function queryItems() {
    var params = new URLSearchParams();
    Object.keys(state).forEach(function (key) { if (state[key] !== "") { params.set(key, String(state[key])); } });
    return fetch("/todos/items?" + params.toString()).then(function (response) {
      if (!response.ok) { throw new Error("request failed"); }
      return response.json();
    }).then(function (payload) {
      if (!payload || payload.success !== true) { throw new Error("invalid payload"); }
      return payload;
    });
  }

  function stageLabel(value) {
    var labels = { draft: "草稿", wait: "待受理", refuse: "已挂起", active: "待澄清", clarified: "待排期", developing: "研发中", testing: "测试中", waitacceptance: "待验收", acceptanced: "待交付", waitdeliver: "待发布", opened: "处理中", doing: "进行中" };
    return labels[value] || value || "—";
  }

  function rowHtml(item) {
    var priority = ["P1", "P2", "P3", "P4"].indexOf(item.priority) >= 0 ? item.priority.toLowerCase() : "normal";
    var id = escapeHtml(item.displayId || item.id);
    var title = escapeHtml(item.title || "—");
    var idContent = item.url ? '<a class="row-id-link" href="' + escapeHtml(item.url) + '" target="_blank" rel="noopener noreferrer">' + id + "</a>" : id;
    var titleContent = item.url ? '<a class="row-title-link" href="' + escapeHtml(item.url) + '" target="_blank" rel="noopener noreferrer">' + title + "</a>" : title;
    var action = escapeHtml(item.action || "查看");
    return "<tr>" +
      '<td class="c-id">' + idContent + "</td>" +
      '<td class="c-title" title="' + title + '">' + titleContent + "</td>" +
      '<td class="c-type"><span class="type-tag">' + escapeHtml(item.type || "—") + "</span></td>" +
      '<td class="c-pri"><span class="pri-tag ' + priority + '">' + escapeHtml(item.priority || "—") + "</span></td>" +
      '<td class="c-relation"><span class="relation-tag">' + escapeHtml(item.relation || "—") + "</span></td>" +
      '<td class="c-reason" title="' + escapeHtml(stageLabel(item.reason)) + '">' + escapeHtml(stageLabel(item.reason)) + "</td>" +
      '<td class="c-dead">' + escapeHtml(item.deadline || "—") + "</td>" +
      '<td class="c-owner" title="' + escapeHtml(item.owner || "—") + '">' + escapeHtml(item.owner || "—") + "</td>" +
      '<td class="c-action">' + (item.url ? '<button type="button" class="todo-action" data-url="' + escapeHtml(item.url) + '">' + action + "</button>" : "—") + "</td></tr>";
  }

  function renderCounts(summary, groups) {
    var summaryMap = { countPending: "pending", countToday: "today", countOverdue: "overdue", countBlocked: "blocked", countP1: "p1" };
    var groupMap = { groupAll: "all", groupApproval: "approval", groupDemand: "demand", groupExecution: "execution", groupTesting: "testing", groupRisk: "risk", groupPersonal: "personal" };
    Object.keys(summaryMap).forEach(function (id) { document.getElementById(id).textContent = summary[summaryMap[id]] || 0; });
    Object.keys(groupMap).forEach(function (id) { document.getElementById(id).textContent = groups[groupMap[id]] || 0; });
  }

  function renderList(payload) {
    var items = payload.items || [];
    document.getElementById("todosTbody").innerHTML = items.map(rowHtml).join("");
    document.getElementById("todosEmpty").hidden = items.length !== 0;
    document.getElementById("todosSummary").textContent = "共 " + (payload.total || 0) + " 条符合条件的待办";
    renderCounts(payload.summary || {}, payload.groups || {});
    renderPagination(payload.total || 0);
  }

  function renderPagination(total) {
    var host = document.getElementById("todosPagination");
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
    document.getElementById("todosSummary").textContent = "加载中…";
    queryItems().then(renderList).catch(function () {
      document.getElementById("todosSummary").textContent = "加载失败，请稍后重试";
      document.getElementById("todosTbody").innerHTML = "";
      document.getElementById("todosEmpty").hidden = false;
    });
  }

  function activate(selector, current) {
    document.querySelectorAll(selector).forEach(function (button) { button.classList.toggle("active", button === current); });
  }

  function syncObjectTypeOptions() {
    var optionsByTab = {
      all: [["all", "全部"], ["demand", "业务需求"], ["story", "研发需求"], ["task", "任务"], ["bug", "Bug"], ["testtask", "测试单"], ["issue", "问题"], ["risk", "风险"], ["todo", "个人待办"]],
      approval: [["all", "全部"]],
      demand: [["all", "全部"], ["demand", "业务需求"], ["story", "研发需求"]],
      execution: [["all", "全部"], ["task", "任务"]],
      testing: [["all", "全部"], ["bug", "Bug"], ["testtask", "测试单"]],
      risk: [["all", "全部"], ["issue", "问题"], ["risk", "风险"]],
      personal: [["all", "全部"], ["todo", "个人待办"]],
    };
    var select = document.getElementById("todosObjectType");
    select.innerHTML = (optionsByTab[state.tab] || optionsByTab.all).map(function (option) {
      return '<option value="' + option[0] + '">具体对象：' + option[1] + "</option>";
    }).join("");
    state.objectType = "all";
    select.value = "all";
  }

  function bindEvents() {
    document.querySelectorAll(".relation-tab").forEach(function (button) { button.addEventListener("click", function () { activate(".relation-tab", button); state.relation = button.dataset.relation; state.page = 1; refresh(); }); });
    document.querySelectorAll(".focus-card").forEach(function (button) { button.addEventListener("click", function () { activate(".focus-card", button); state.focus = button.dataset.focus; state.page = 1; refresh(); }); });
    document.querySelectorAll(".group-tab").forEach(function (button) { button.addEventListener("click", function () { activate(".group-tab", button); state.tab = button.dataset.tab; state.page = 1; syncObjectTypeOptions(); refresh(); }); });
    var fields = { todosAction: "action", todosStage: "stage", todosObjectType: "objectType", todosResponsibility: "responsibility" };
    Object.keys(fields).forEach(function (id) { document.getElementById(id).addEventListener("change", function (event) { state[fields[id]] = event.target.value; state.page = 1; refresh(); }); });
    var keyword = document.getElementById("todosKeyword"); var timer;
    keyword.addEventListener("input", function () { clearTimeout(timer); timer = setTimeout(function () { state.keyword = keyword.value.trim(); state.page = 1; refresh(); }, 300); });
    document.getElementById("todosResetBtn").addEventListener("click", function () { state.action = "all"; state.stage = "all"; state.objectType = "all"; state.responsibility = "all"; state.keyword = ""; state.page = 1; Object.keys(fields).forEach(function (id) { document.getElementById(id).value = "all"; }); keyword.value = ""; refresh(); });
    document.getElementById("todosTbody").addEventListener("click", function (event) { var button = event.target.closest("[data-url]"); if (button) { window.location.href = button.dataset.url; } });
    document.getElementById("todosPagination").addEventListener("click", function (event) { var button = event.target.closest("[data-page]"); if (!button || button.disabled) { return; } state.page = Number(button.dataset.page); refresh(); });
    document.getElementById("todosPagination").addEventListener("change", function (event) { if (!event.target.matches(".pager-page-size")) { return; } state.pageSize = Number(event.target.value); state.page = 1; refresh(); });
  }

  document.addEventListener("DOMContentLoaded", function () { syncObjectTypeOptions(); bindEvents(); refresh(); });
})();
