(function () {
  "use strict";

  var state = {
    tab: "demand",
    action: "all",
    stage: "all",
    objectType: "all",
    relation: "all",
    responsibility: "all",
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
    if (state.action) { params.set("action", state.action); }
    if (state.stage) { params.set("stage", state.stage); }
    if (state.objectType) { params.set("objectType", state.objectType); }
    if (state.relation) { params.set("relation", state.relation); }
    if (state.responsibility) { params.set("responsibility", state.responsibility); }
    if (state.keyword) { params.set("keyword", state.keyword); }
    params.set("page", String(state.page));
    params.set("pageSize", String(state.pageSize));
    return fetch("/todos/items?" + params.toString(), { method: "GET" })
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
    var today = new Date();
    today.setHours(0, 0, 0, 0);
    var dead = item.deadline ? new Date(item.deadline) : null;
    var deadColor = "";
    if (dead) {
      deadColor = dead < today ? "var(--red, #d4380d)" : "var(--po-t2, #65758b)";
    } else {
      deadColor = "var(--po-t2, #65758b)";
    }
    var priClass = ["P1", "P2", "P3", "P4"].indexOf(item.priority) >= 0
      ? item.priority.toLowerCase()
      : "normal";
    return (
      "<tr>" +
      '<td class="c-id">' + escapeHtml(item.displayId || item.id) + "</td>" +
      '<td class="c-title" title="' + escapeHtml(item.title) + '">' +
      escapeHtml(item.title || "—") +
      "</td>" +
      '<td class="c-stage">' + escapeHtml(item.stage || "—") + "</td>" +
      '<td class="c-pri"><span class="pri-tag ' + priClass + '">' +
      escapeHtml(item.priority || "—") +
      "</span></td>" +
      '<td class="c-dead" style="color:' + deadColor + '">' +
      escapeHtml(item.deadline || "—") +
      "</td>" +
      '<td class="c-owner">' + escapeHtml(item.owner || "—") + "</td>" +
      '<td class="c-action"><button type="button" class="action-btn primary" data-item-id="' +
      escapeHtml(item.id) + '">办理</button></td>' +
      "</tr>"
    );
  }

  function renderList(items) {
    var tbody = document.getElementById("todosTbody");
    var empty = document.getElementById("todosEmpty");
    if (!items || items.length === 0) {
      tbody.innerHTML = "";
      empty.hidden = false;
      return;
    }
    empty.hidden = true;
    tbody.innerHTML = items.map(renderRow).join("");
  }

  function refresh() {
    document.getElementById("todosSummary").textContent = "加载中…";
    fetchItems()
      .then(function (payload) {
        document.getElementById("todosSummary").textContent =
          "需求治理 · " + (payload.total || 0) + " 条";
        renderList(payload.items || []);
      })
      .catch(function () {
        document.getElementById("todosSummary").textContent = "加载失败";
        renderList([]);
      });
  }

  function bindEvents() {
    var tabs = document.querySelectorAll(".todos-tab");
    tabs.forEach(function (btn) {
      btn.addEventListener("click", function () {
        tabs.forEach(function (b) { b.classList.remove("active"); });
        btn.classList.add("active");
        state.tab = btn.dataset.tab;
        state.page = 1;
        refresh();
      });
    });

    var kw = document.getElementById("todosKeyword");
    var kwTimer = null;
    kw.addEventListener("input", function () {
      clearTimeout(kwTimer);
      kwTimer = setTimeout(function () {
        state.keyword = kw.value.trim();
        state.page = 1;
        refresh();
      }, 300);
    });

    // 7 维筛选：所有 select 变化都触发刷新
    var selectIds = ["todosAction", "todosStage", "todosObjectType", "todosRelation", "todosResponsibility"];
    var stateKeyMap = {
      todosAction: "action",
      todosStage: "stage",
      todosObjectType: "objectType",
      todosRelation: "relation",
      todosResponsibility: "responsibility",
    };
    selectIds.forEach(function (id) {
      var el = document.getElementById(id);
      if (!el) { return; }
      el.addEventListener("change", function () {
        state[stateKeyMap[id]] = el.value;
        state.page = 1;
        refresh();
      });
    });

    document.getElementById("todosResetBtn").addEventListener("click", function () {
      state.action = "all";
      state.stage = "all";
      state.objectType = "all";
      state.relation = "all";
      state.responsibility = "all";
      state.keyword = "";
      state.page = 1;
      document.getElementById("todosAction").value = "all";
      document.getElementById("todosStage").value = "all";
      document.getElementById("todosObjectType").value = "all";
      document.getElementById("todosRelation").value = "all";
      document.getElementById("todosResponsibility").value = "all";
      kw.value = "";
      refresh();
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    bindEvents();
    refresh();
  });
})();