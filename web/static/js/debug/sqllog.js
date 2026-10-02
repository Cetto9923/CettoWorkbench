/**
 * 慢 SQL 明细页
 * 数据通过 /debug/sqllog/queries 读取 logs/sql-YYYY-MM-DD.log，按耗时降序
 */
const SQLLOG_API = "/debug/sqllog/queries";
const SLOW_MS = 200;

function todayLocal() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}



function shortFile(path) {
  const raw = String(path ?? "");
  const marker = "/internal/";
  const idx = raw.lastIndexOf(marker);
  if (idx >= 0) {
    return raw.slice(idx + 1);
  }
  return raw;
}

function renderSummary(total, shown, date) {
  const el = document.getElementById("sqllog-summary");
  if (!el) return;
  el.textContent = `${date} 共 ${total} 条，按耗时降序展示前 ${shown} 条`;
}

function renderEmpty(message) {
  const tbody = document.getElementById("sqllog-tbody");
  if (!tbody) return;
  tbody.innerHTML = `<tr><td colspan="5" class="sqllog-empty">${window.escapeHtml(message)}</td></tr>`;
}

function formatAPI(method, route) {
  const m = String(method ?? "").trim();
  const r = String(route ?? "").trim();
  if (!m && !r) return "-";
  if (!m) return r;
  if (!r) return m;
  return `${m} ${r}`;
}

function fillRouteSelect(routes) {
  const select = document.getElementById("sqllog-route");
  if (!select) return;
  const prev = select.value;
  const list = Array.isArray(routes) ? routes : [];
  const options = [`<option value="">全部</option>`].concat(
    list.map((r) => `<option value="${window.escapeHtml(r)}">${window.escapeHtml(r)}</option>`)
  );
  select.innerHTML = options.join("");
  if (prev && list.includes(prev)) {
    select.value = prev;
  } else {
    select.value = "";
  }
}

function markClampedSQL(tbody) {
  tbody.querySelectorAll(".sqllog-sql-wrap").forEach((wrap) => {
    const sql = wrap.querySelector(".sqllog-sql");
    if (!sql) return;
    wrap.classList.remove("is-clamped", "is-expanded");
    const toggle = wrap.querySelector(".sqllog-sql-toggle");
    if (toggle) toggle.textContent = "展开";
    if (sql.scrollHeight > sql.clientHeight + 1) {
      wrap.classList.add("is-clamped");
    }
  });
}

async function copyText(text) {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.left = "-9999px";
  document.body.appendChild(ta);
  ta.select();
  document.execCommand("copy");
  document.body.removeChild(ta);
}

function renderRows(queries) {
  const tbody = document.getElementById("sqllog-tbody");
  if (!tbody) return;
  if (!queries.length) {
    renderEmpty("当天暂无匹配的 SQL 日志");
    return;
  }

  const html = queries.map((q) => {
    const slowClass = (q.elapsed_ms || 0) >= SLOW_MS ? " is-slow" : "";
    const api = formatAPI(q.method, q.route);
    const err = q.error
      ? `<div class="sqllog-error-tag">${window.escapeHtml(q.error)}</div>`
      : "";
    return `<tr>
      <td class="sqllog-elapsed${slowClass}">${window.escapeHtml(q.elapsed)}</td>
      <td>${window.escapeHtml(q.time)}</td>
      <td class="sqllog-api" title="${window.escapeHtml(api)}">${window.escapeHtml(api)}</td>
      <td class="sqllog-file" title="${window.escapeHtml(q.file)}">${window.escapeHtml(shortFile(q.file))}</td>
      <td>
        <div class="sqllog-sql-wrap">
          <div class="sqllog-sql">${window.escapeHtml(q.sql)}</div>
          <div class="sqllog-sql-actions">
            <button type="button" class="sqllog-sql-toggle">展开</button>
            <button type="button" class="sqllog-sql-copy">复制</button>
          </div>
        </div>
        ${err}
      </td>
    </tr>`;
  }).join("");
  tbody.innerHTML = html;
  markClampedSQL(tbody);
}

async function loadQueries() {
  const dateInput = document.getElementById("sqllog-date");
  const limitSelect = document.getElementById("sqllog-limit");
  const routeSelect = document.getElementById("sqllog-route");
  if (!dateInput || !limitSelect || !routeSelect) return;

  const date = dateInput.value || todayLocal();
  const limit = limitSelect.value || "200";
  const route = routeSelect.value || "";
  dateInput.value = date;

  renderEmpty("加载中…");
  try {
    let url = `${SQLLOG_API}?date=${encodeURIComponent(date)}&limit=${encodeURIComponent(limit)}`;
    if (route) {
      url += `&route=${encodeURIComponent(route)}`;
    }
    const res = await fetch(url, { headers: { Accept: "application/json" } });
    const json = await res.json();
    if (!res.ok || !json.success) {
      renderEmpty(json.message || "读取失败");
      return;
    }
    fillRouteSelect(json.routes);
    if (route && routeSelect.value !== route) {
      // 切换日期后原接口不存在，已重置为全部，再拉一次无 route 的数据
      loadQueries();
      return;
    }
    const queries = Array.isArray(json.queries) ? json.queries : [];
    renderSummary(json.total ?? queries.length, queries.length, json.date || date);
    renderRows(queries);
  } catch (err) {
    renderEmpty("读取失败");
  }
}

function init() {
  const dateInput = document.getElementById("sqllog-date");
  const limitSelect = document.getElementById("sqllog-limit");
  const routeSelect = document.getElementById("sqllog-route");
  const tbody = document.getElementById("sqllog-tbody");
  if (!dateInput || !limitSelect || !routeSelect) return;

  dateInput.value = todayLocal();
  dateInput.addEventListener("change", loadQueries);
  limitSelect.addEventListener("change", loadQueries);
  routeSelect.addEventListener("change", loadQueries);

  if (tbody) {
    tbody.addEventListener("click", async (e) => {
      const toggle = e.target.closest(".sqllog-sql-toggle");
      if (toggle) {
        const wrap = toggle.closest(".sqllog-sql-wrap");
        if (!wrap) return;
        const expanded = wrap.classList.toggle("is-expanded");
        toggle.textContent = expanded ? "收起" : "展开";
        return;
      }

      const copyBtn = e.target.closest(".sqllog-sql-copy");
      if (!copyBtn) return;
      const wrap = copyBtn.closest(".sqllog-sql-wrap");
      const sqlEl = wrap && wrap.querySelector(".sqllog-sql");
      if (!sqlEl) return;
      try {
        await copyText(sqlEl.textContent || "");
        copyBtn.textContent = "已复制";
        copyBtn.classList.add("is-copied");
        setTimeout(() => {
          copyBtn.textContent = "复制";
          copyBtn.classList.remove("is-copied");
        }, 1500);
      } catch (err) {
        copyBtn.textContent = "复制失败";
        setTimeout(() => {
          copyBtn.textContent = "复制";
        }, 1500);
      }
    });
  }

  loadQueries();
}

document.addEventListener("DOMContentLoaded", init);
