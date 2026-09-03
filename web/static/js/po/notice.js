(function () {
  "use strict";

  var state = { quickView: "all", category: "all", objectType: "all", timeRange: "all", readState: "all", needAction: "all", keyword: "", page: 1, pageSize: 15 };
  var categoryLabels = { business: "业务动态", approval: "审批流程", reminder: "时效提醒", collaboration: "协作消息", risk: "风险异常", system: "系统消息" };

  function escapeHtml(value) { return String(value == null ? "" : value).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;"); }
  function queryItems() { var params = new URLSearchParams(); Object.keys(state).forEach(function (key) { params.set(key, String(state[key])); }); return fetch("/notice/items?" + params.toString()).then(function (response) { if (!response.ok) { throw new Error("request failed"); } return response.json(); }).then(function (payload) { if (!payload || payload.success !== true) { throw new Error("invalid payload"); } return payload; }); }

  function rowHtml(item) {
    var category = item.category || "business";
    var content = escapeHtml(item.subject || item.data || "—");
    var description = item.data && item.data !== item.subject ? '<small title="' + escapeHtml(item.data) + '">' + escapeHtml(item.data) + "</small>" : "";
    var object = escapeHtml(item.objectType || "—") + " / " + escapeHtml(item.objectId || "—");
    var objectLink = item.url ? '<a class="notice-object-link" href="' + escapeHtml(item.url) + '">' + object + "</a>" : object;
    var status = item.read ? '<span class="notice-read-state">已读</span>' : '<span class="notice-unread-state"><i></i>未读</span>';
    var operation = item.read ? '<button type="button" class="notice-read-btn" disabled>已读</button>' : '<button type="button" class="notice-read-btn" data-notice-id="' + escapeHtml(item.id) + '">标为已读</button>';
    return "<tr" + (item.read ? "" : ' class="is-unread"') + "><td class=\"notice-subject\"><div class=\"notice-subject-text\">" + content + "</div>" + description + "</td><td><span class=\"notice-category-tag " + escapeHtml(category) + "\">" + escapeHtml(categoryLabels[category] || category) + "</span></td><td>" + objectLink + "</td><td>" + escapeHtml(item.actor || "—") + "</td><td>" + escapeHtml(item.date || "—") + "</td><td class=\"notice-status\">" + status + "</td><td class=\"notice-operation\">" + operation + "</td></tr>";
  }

  function updateCounts(payload) {
    var quick = { qvCountAll: "total", qvCountUnread: "unread", qvCountAction: "action", qvCountAbnormal: "abnormal", qvCountToday: "today" };
    var categories = { categoryBusiness: "business", categoryApproval: "approval", categoryReminder: "reminder", categoryCollaboration: "collaboration", categoryRisk: "risk", categorySystem: "system" };
    Object.keys(quick).forEach(function (id) { document.getElementById(id).textContent = payload[quick[id]] || 0; });
    document.getElementById("categoryAll").textContent = (payload.categories || {}).all || 0;
    Object.keys(categories).forEach(function (id) { document.getElementById(id).textContent = (payload.categories || {})[categories[id]] || 0; });
  }

  function renderList(payload) { var items = payload.items || []; document.getElementById("noticeTbody").innerHTML = items.map(rowHtml).join(""); document.getElementById("noticeEmpty").hidden = items.length !== 0; document.getElementById("noticeSummary").textContent = "共 " + (payload.filteredTotal || 0) + " 条符合条件的通知"; renderPagination(payload.filteredTotal || 0); updateCounts(payload); }
  function renderPagination(total) { var host = document.getElementById("noticePagination"); var pages = Math.max(1, Math.ceil(total / state.pageSize)); host.hidden = total === 0; if (!total) { host.innerHTML = ""; return; } var start = (state.page - 1) * state.pageSize + 1; var end = Math.min(total, state.page * state.pageSize); host.innerHTML = "<span>显示 " + start + "–" + end + "，共 " + total + " 条</span><div class=\"notice-pager-controls\"><select class=\"notice-page-size\" aria-label=\"每页条数\"><option value=\"10\">10 条/页</option><option value=\"15\">15 条/页</option><option value=\"20\">20 条/页</option><option value=\"30\">30 条/页</option><option value=\"50\">50 条/页</option></select><button class=\"notice-pager-btn\" data-page=\"" + (state.page - 1) + "\"" + (state.page === 1 ? " disabled" : "") + ">‹</button><span>第 " + state.page + " / " + pages + " 页</span><button class=\"notice-pager-btn\" data-page=\"" + (state.page + 1) + "\"" + (state.page === pages ? " disabled" : "") + ">›</button></div>"; host.querySelector(".notice-page-size").value = String(state.pageSize); }
  function refresh() { document.getElementById("noticeSummary").textContent = "加载中…"; queryItems().then(renderList).catch(function () { document.getElementById("noticeSummary").textContent = "加载失败，请稍后重试"; document.getElementById("noticeTbody").innerHTML = ""; document.getElementById("noticeEmpty").hidden = false; }); }
  function activate(selector, current) { document.querySelectorAll(selector).forEach(function (button) { button.classList.toggle("active", button === current); }); }
  function markRead(id) { return window.appFetch("/notice/" + encodeURIComponent(id) + "/read", { method: "PUT" }).then(function (response) { return response.json().then(function (payload) { if (!response.ok || !payload || payload.success !== true) { throw new Error("标记已读失败"); } }); }); }

  function bindEvents() {
    document.querySelectorAll("[data-qv]").forEach(function (button) { button.addEventListener("click", function () { activate("[data-qv]", button); state.quickView = button.dataset.qv; state.page = 1; refresh(); }); });
    document.querySelectorAll("[data-category]").forEach(function (button) { button.addEventListener("click", function () { activate("[data-category]", button); state.category = button.dataset.category; state.page = 1; refresh(); }); });
    var fields = { noticeObjectType: "objectType", noticeTimeRange: "timeRange", noticeReadState: "readState", noticeNeedAction: "needAction" };
    Object.keys(fields).forEach(function (id) { document.getElementById(id).addEventListener("change", function (event) { state[fields[id]] = event.target.value; state.page = 1; refresh(); }); });
    var keyword = document.getElementById("noticeKeyword"), timer;
    keyword.addEventListener("input", function () { clearTimeout(timer); timer = setTimeout(function () { state.keyword = keyword.value.trim(); state.page = 1; refresh(); }, 300); });
    document.getElementById("noticeResetBtn").addEventListener("click", function () { Object.keys(fields).forEach(function (id) { state[fields[id]] = "all"; document.getElementById(id).value = "all"; }); state.keyword = ""; state.page = 1; keyword.value = ""; refresh(); });
    document.getElementById("noticeTbody").addEventListener("click", function (event) { var button = event.target.closest("[data-notice-id]"); if (!button) { return; } button.disabled = true; markRead(button.dataset.noticeId).then(refresh).catch(function () { button.disabled = false; }); });
    document.getElementById("noticePagination").addEventListener("click", function (event) { var button = event.target.closest("[data-page]"); if (!button || button.disabled) { return; } state.page = Number(button.dataset.page); refresh(); });
    document.getElementById("noticePagination").addEventListener("change", function (event) { if (!event.target.matches(".notice-page-size")) { return; } state.pageSize = Number(event.target.value); state.page = 1; refresh(); });
    document.getElementById("noticeMarkAllBtn").addEventListener("click", function () { var button = this; button.disabled = true; window.appFetch("/notice/read-all", { method: "PUT" }).then(function (response) { return response.json().then(function (payload) { if (!response.ok || !payload || payload.success !== true) { throw new Error("全部标记已读失败"); } }); }).then(refresh).finally(function () { button.disabled = false; }); });
  }

  document.addEventListener("DOMContentLoaded", function () { bindEvents(); refresh(); });
})();
