(function () {
  "use strict";

  const API = "/workbench/api/agile-teams";
  const rowsHost = document.getElementById("teamHomeGroupRows");
  if (!rowsHost) return;

  const params = new URLSearchParams(window.location.search);
  const state = {
    scope: params.get("scope") === "dept" ? "dept" : "team",
    teamgroupId: positiveID(params.get("teamgroupId") || params.get("scopeId")),
    requestNo: 0,
  };

  function positiveID(raw) {
    const value = String(raw || "");
    return /^\d+$/.test(value) && Number(value) > 0 ? Number(value) : 0;
  }

  function esc(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (ch) {
      return ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch];
    });
  }

  function syncURL() {
    const url = new URL(window.location.href);
    url.searchParams.set("view", "team");
    url.searchParams.set("scope", state.scope);
    if (state.teamgroupId) url.searchParams.set("teamgroupId", String(state.teamgroupId));
    else url.searchParams.delete("teamgroupId");
    url.searchParams.delete("scopeId");
    window.history.replaceState(null, "", url.pathname + url.search + url.hash);
  }

  function showError(message) {
    const alert = document.getElementById("teamHomeError");
    const text = document.getElementById("teamHomeErrorText");
    if (text) text.textContent = message || "团队数据暂不可用";
    if (alert) alert.hidden = false;
  }

  function setScopeControls(data) {
    const allowed = Array.isArray(data.availableScopes) ? data.availableScopes : [];
    document.querySelectorAll("#teamHomeScopeSwitch [data-scope]").forEach(function (button) {
      const scope = button.getAttribute("data-scope");
      button.hidden = allowed.indexOf(scope) < 0;
      button.classList.toggle("active", scope === state.scope);
      button.setAttribute("aria-pressed", scope === state.scope ? "true" : "false");
    });

    const select = document.getElementById("teamHomeGroupSelect");
    if (!select) return true;
    const options = Array.isArray(data.scopeOptions) ? data.scopeOptions : [];
    const hasSelectedOption = !state.teamgroupId || options.some(function (option) {
      return Number(option.id) === state.teamgroupId;
    });
    if (!hasSelectedOption) {
      state.teamgroupId = 0;
      return false;
    }
    select.innerHTML = '<option value="">全部可见团队/小组</option>' + options.map(function (option) {
      const prefix = option.type === "subteam" ? "敏捷小组 · " : (option.type === "team" ? "敏捷团队 · " : "");
      return '<option value="' + esc(option.id) + '">' + esc(prefix + option.name) + "</option>";
    }).join("");
    select.value = options.some(function (option) { return Number(option.id) === state.teamgroupId; })
      ? String(state.teamgroupId) : "";
    return true;
  }

  function renderRows(items) {
    const rows = Array.isArray(items) ? items : [];
    if (!rows.length) {
      rowsHost.innerHTML = '<tr><td colspan="5" class="state-placeholder">当前授权范围内没有已挂靠的敏捷团队或小组</td></tr>';
      return;
    }
    rowsHost.innerHTML = rows.map(function (item) {
      const type = item.type === "child" ? "敏捷小组" : "敏捷团队";
      const pending = Number(item.pendingAdd || 0) + Number(item.pendingRemove || 0);
      return "<tr>" +
        '<td><strong>' + esc(item.name) + "</strong></td>" +
        "<td>" + type + "</td>" +
        "<td>" + Number(item.formalCount || 0) + "</td>" +
        "<td>" + pending + "</td>" +
        "<td>" + esc(item.statusLabel || "启用") + "</td>" +
        "</tr>";
    }).join("");
  }

  async function load() {
    const requestNo = ++state.requestNo;
    const error = document.getElementById("teamHomeError");
    if (error) error.hidden = true;
    rowsHost.innerHTML = '<tr><td colspan="5" class="state-placeholder">正在加载授权范围…</td></tr>';

    const query = new URLSearchParams({ view: "lead", scope: state.scope, page: "1", pageSize: "200" });
    if (state.teamgroupId) query.set("scopeId", String(state.teamgroupId));
    try {
      const response = await fetch(API + "?" + query.toString(), {
        credentials: "include",
        headers: { "Accept": "application/json", "X-Requested-With": "XMLHttpRequest" },
      });
      const json = await response.json().catch(function () { return null; });
      if (!response.ok || !json || !json.success) {
        throw new Error((json && (json.message || json.error)) || ("HTTP " + response.status));
      }
      if (requestNo !== state.requestNo) return;
      const data = json.data || {};
      if (data.activeScope) state.scope = data.activeScope;
      if (!setScopeControls(data)) {
        syncURL();
        load();
        return;
      }
      renderRows(data.items);
      const updated = document.getElementById("teamHomeUpdatedAt");
      if (updated) updated.textContent = "授权范围内 " + Number(data.allCount || 0) + " 个团队/小组";
      syncURL();
    } catch (error) {
      if (requestNo !== state.requestNo) return;
      rowsHost.innerHTML = "";
      showError(error.message);
    }
  }

  document.querySelectorAll("#teamHomeScopeSwitch [data-scope]").forEach(function (button) {
    button.addEventListener("click", function () {
      state.scope = button.getAttribute("data-scope") === "dept" ? "dept" : "team";
      state.teamgroupId = 0;
      syncURL();
      load();
    });
  });
  const select = document.getElementById("teamHomeGroupSelect");
  if (select) select.addEventListener("change", function () {
    state.teamgroupId = positiveID(select.value);
    syncURL();
    load();
  });
  const retry = document.getElementById("teamHomeRetry");
  if (retry) retry.addEventListener("click", load);

  syncURL();
  load();
})();
