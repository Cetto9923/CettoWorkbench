(function () {
  "use strict";

  var state = {
    quickView: "all",
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
    params.set("quickView", state.quickView);
    if (state.keyword) {
      params.set("keyword", state.keyword);
    }
    return fetch("/notice/items?" + params.toString(), { method: "GET" })
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
    var cat = (item.category || "business").toLowerCase();
    var catLabels = { business: "业务", approval: "审批", reminder: "提醒", collaboration: "协作", risk: "风险", system: "系统" };
    var catLabel = catLabels[cat] || item.category || "—";
    var unread = item.read ? "" : '<span class="unread-dot"></span>';
    var objLabel = (item.objectType || "—") + "/" + (item.objectId || 0);
    var readBtn = item.read
      ? '<button type="button" class="notice-read-btn" disabled>已读</button>'
      : '<button type="button" class="notice-read-btn" data-notice-id="' + escapeHtml(item.id) + '">标为已读</button>';

    return (
      '<article class="notice-item' + (item.read ? "" : " unread") + '">' +
      '<div class="notice-rail ' + escapeHtml(cat) + '"></div>' +
      '<div class="notice-body">' +
      '<div class="notice-row1">' + unread +
      '<span class="notice-title">' + escapeHtml(item.subject || item.data || "—") + '</span>' +
      '<span class="cat-tag ' + escapeHtml(cat) + '">' + escapeHtml(catLabel) + '</span>' +
      '<span class="notice-meta">' + escapeHtml(item.date) + ' · ' + escapeHtml(item.actor || "—") + '</span>' +
      '</div>' +
      (item.data ? '<div class="notice-summary">' + escapeHtml(item.data.substring(0, 200)) + '</div>' : "") +
      '<div class="notice-row1" style="margin-top:6px">' +
      '<span class="notice-meta">关联对象: ' + escapeHtml(objLabel) + '</span>' +
      '</div>' +
      '</div>' +
      '<div class="notice-actions-cell">' + readBtn + '</div>' +
      '</article>'
    );
  }

  function renderList(items) {
    var host = document.getElementById("noticeListHost");
    var empty = document.getElementById("noticeEmpty");
    if (!items || items.length === 0) {
      host.innerHTML = "";
      host.appendChild(empty);
      empty.hidden = false;
      return;
    }
    empty.hidden = true;
    host.innerHTML = items.map(renderItem).join("");
    bindReadButtons();
  }

  function bindReadButtons() {
    var btns = document.querySelectorAll(".notice-read-btn[data-notice-id]");
    btns.forEach(function (b) {
      b.addEventListener("click", function () {
        var id = b.dataset.noticeId;
        b.disabled = true;
        fetch("/notice/" + encodeURIComponent(id) + "/read", { method: "PUT" })
          .then(function (r) { return r.json(); })
          .then(function () { refresh(); })
          .catch(function () { b.disabled = false; });
      });
    });
  }

  function updateQVCounts(payload) {
    var setText = function (id, val) {
      var el = document.getElementById(id);
      if (el) { el.textContent = String(val || 0); }
    };
    setText("qvCountAll", payload.total);
    setText("qvCountUnread", payload.unread);
    setText("qvCountAction", payload.action);
    setText("qvCountAbnormal", payload.abnormal);
    setText("qvCountToday", payload.today);
  }

  function refresh() {
    document.getElementById("noticeSummary").textContent = "加载中…";
    fetchItems()
      .then(function (payload) {
        updateQVCounts(payload);
        document.getElementById("noticeSummary").textContent =
          state.quickView + " · " + (payload.items || []).length + " / " + (payload.total || 0) + " 条";
        renderList(payload.items || []);
      })
      .catch(function () {
        document.getElementById("noticeSummary").textContent = "加载失败";
        renderList([]);
      });
  }

  function bindEvents() {
    document.querySelectorAll(".quick-tab").forEach(function (btn) {
      btn.addEventListener("click", function () {
        document.querySelectorAll(".quick-tab").forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.quickView = btn.dataset.qv;
        refresh();
      });
    });

    var kw = document.getElementById("noticeKeyword");
    var kwTimer = null;
    kw.addEventListener("input", function () {
      clearTimeout(kwTimer);
      kwTimer = setTimeout(function () {
        state.keyword = kw.value.trim();
        refresh();
      }, 300);
    });

    document.getElementById("noticeMarkAllBtn").addEventListener("click", function () {
      fetch("/notice/read-all", { method: "POST" })
        .then(function (r) { return r.json(); })
        .then(function () { refresh(); });
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();