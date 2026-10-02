/**
 * 禅道 API 请求日志页
 * 数据通过 /debug/apilog/entries 读取 logs/api-YYYY-MM-DD.log，按时间倒序分页
 */
const APILOG_API = "/debug/apilog/entries";
const SLOW_MS = 200;

/** @type {Array<{request: string, response: string}>} */
let apilogPayloads = [];
let apilogPage = 1;

function todayLocal() {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}



/** 若字符串本身是 JSON，则解析一层；否则原样返回。 */
function tryParseJSONString(text) {
  if (typeof text !== "string") {
    return text;
  }
  const trimmed = text.trim();
  if (!trimmed) {
    return text;
  }
  const first = trimmed[0];
  if (first !== "{" && first !== "[" && first !== '"') {
    return text;
  }
  try {
    return JSON.parse(trimmed);
  } catch (err) {
    return text;
  }
}

/** 递归解开被二次 stringify 的 JSON 字符串（含 \\uXXXX）。 */
function unwrapJSON(value, depth) {
  const maxDepth = 6;
  const level = depth || 0;
  if (level > maxDepth) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = tryParseJSONString(value);
    if (parsed !== value) {
      return unwrapJSON(parsed, level + 1);
    }
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((item) => unwrapJSON(item, level + 1));
  }
  if (value && typeof value === "object") {
    const out = {};
    Object.keys(value).forEach((key) => {
      out[key] = unwrapJSON(value[key], level + 1);
    });
    return out;
  }
  return value;
}

function prettyJSON(value) {
  if (value === undefined || value === null || value === "") {
    return "";
  }
  try {
    return JSON.stringify(unwrapJSON(value), null, 2);
  } catch (err) {
    return String(value);
  }
}

function renderSummary(total, page, pageSize, date) {
  const el = document.getElementById("apilog-summary");
  if (!el) return;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  el.textContent = `${date} 共 ${total} 条，第 ${page}/${totalPages} 页`;
}

function renderPager(total, page, pageSize) {
  const pager = document.getElementById("apilog-pager");
  const info = document.getElementById("apilog-page-info");
  const prev = document.getElementById("apilog-prev");
  const next = document.getElementById("apilog-next");
  if (!pager || !info || !prev || !next) return;

  if (total <= 0) {
    pager.hidden = true;
    return;
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  pager.hidden = false;
  info.textContent = `第 ${page} / ${totalPages} 页`;
  prev.disabled = page <= 1;
  next.disabled = page >= totalPages;
}

function renderEmpty(message) {
  const tbody = document.getElementById("apilog-tbody");
  if (!tbody) return;
  tbody.innerHTML = `<tr><td colspan="7" class="apilog-empty">${window.escapeHtml(message)}</td></tr>`;
}

function payloadCell(idx, kind, text) {
  if (!text) {
    return `<span class="apilog-muted">-</span>`;
  }
  return `<button type="button" class="apilog-view-btn" data-idx="${idx}" data-kind="${kind}">查看</button>`;
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

function openModal(title, content) {
  const modal = document.getElementById("apilog-modal");
  const titleEl = document.getElementById("apilog-modal-title");
  const bodyEl = document.getElementById("apilog-modal-body");
  const copyBtn = document.getElementById("apilog-modal-copy");
  if (!modal || !titleEl || !bodyEl) return;
  titleEl.textContent = title;
  bodyEl.textContent = content;
  if (copyBtn) {
    copyBtn.textContent = "复制";
    copyBtn.classList.remove("is-copied");
  }
  modal.hidden = false;
  document.body.classList.add("apilog-modal-open");
}

function closeModal() {
  const modal = document.getElementById("apilog-modal");
  if (!modal) return;
  modal.hidden = true;
  document.body.classList.remove("apilog-modal-open");
}

function renderRows(entries) {
  const tbody = document.getElementById("apilog-tbody");
  if (!tbody) return;
  if (!entries.length) {
    apilogPayloads = [];
    renderEmpty("当天暂无 API 日志");
    return;
  }

  apilogPayloads = entries.map((e) => ({
    request: prettyJSON(e.request),
    response: prettyJSON(e.response),
  }));

  const html = entries.map((e, idx) => {
    const slowClass = (e.elapsed_ms || 0) >= SLOW_MS ? " is-slow" : "";
    const ok = !!e.success;
    const statusClass = ok ? "is-ok" : "is-fail";
    const statusText = e.status ? String(e.status) : (ok ? "OK" : "FAIL");
    const url = e.url || e.path || "-";
    return `<tr>
      <td class="apilog-elapsed${slowClass}">${window.escapeHtml(e.elapsed)}</td>
      <td>${window.escapeHtml(e.time)}</td>
      <td class="apilog-method">${window.escapeHtml(e.method)}</td>
      <td class="apilog-url" title="${window.escapeHtml(url)}">${window.escapeHtml(url)}</td>
      <td class="apilog-status ${statusClass}">${window.escapeHtml(statusText)}</td>
      <td>${payloadCell(idx, "request", apilogPayloads[idx].request)}</td>
      <td>${payloadCell(idx, "response", apilogPayloads[idx].response)}</td>
    </tr>`;
  }).join("");
  tbody.innerHTML = html;
}

async function loadEntries() {
  const dateInput = document.getElementById("apilog-date");
  const pageSizeSelect = document.getElementById("apilog-page-size");
  if (!dateInput || !pageSizeSelect) return;

  const date = dateInput.value || todayLocal();
  const pageSize = Number(pageSizeSelect.value) || 100;
  dateInput.value = date;

  renderEmpty("加载中…");
  try {
    const url = `${APILOG_API}?date=${encodeURIComponent(date)}&page=${encodeURIComponent(apilogPage)}&pageSize=${encodeURIComponent(pageSize)}`;
    const res = await fetch(url, { headers: { Accept: "application/json" } });
    const json = await res.json();
    if (!res.ok || !json.success) {
      renderEmpty(json.message || "读取失败");
      renderPager(0, 1, pageSize);
      return;
    }
    const entries = Array.isArray(json.entries) ? json.entries : [];
    const total = Number(json.total) || 0;
    const page = Number(json.page) || apilogPage;
    const size = Number(json.pageSize) || pageSize;
    apilogPage = page;
    renderSummary(total, page, size, json.date || date);
    renderPager(total, page, size);
    renderRows(entries);
  } catch (err) {
    renderEmpty("读取失败");
    renderPager(0, 1, pageSize);
  }
}

function init() {
  const dateInput = document.getElementById("apilog-date");
  const pageSizeSelect = document.getElementById("apilog-page-size");
  const tbody = document.getElementById("apilog-tbody");
  const modal = document.getElementById("apilog-modal");
  const prev = document.getElementById("apilog-prev");
  const next = document.getElementById("apilog-next");
  if (!dateInput || !pageSizeSelect) return;

  dateInput.value = todayLocal();
  dateInput.addEventListener("change", () => {
    apilogPage = 1;
    loadEntries();
  });
  pageSizeSelect.addEventListener("change", () => {
    apilogPage = 1;
    loadEntries();
  });

  if (prev) {
    prev.addEventListener("click", () => {
      if (apilogPage <= 1) return;
      apilogPage -= 1;
      loadEntries();
    });
  }
  if (next) {
    next.addEventListener("click", () => {
      apilogPage += 1;
      loadEntries();
    });
  }

  if (tbody) {
    tbody.addEventListener("click", (e) => {
      const btn = e.target.closest(".apilog-view-btn");
      if (!btn) return;
      const idx = Number(btn.getAttribute("data-idx"));
      const kind = btn.getAttribute("data-kind");
      const payload = apilogPayloads[idx];
      if (!payload) return;
      if (kind === "request") {
        openModal("请求参数", payload.request);
        return;
      }
      if (kind === "response") {
        openModal("返回参数", payload.response);
      }
    });
  }

  if (modal) {
    modal.addEventListener("click", (e) => {
      if (e.target.closest("[data-apilog-close]")) {
        closeModal();
      }
    });
  }

  const copyBtn = document.getElementById("apilog-modal-copy");
  if (copyBtn) {
    copyBtn.addEventListener("click", async () => {
      const bodyEl = document.getElementById("apilog-modal-body");
      if (!bodyEl) return;
      try {
        await copyText(bodyEl.textContent || "");
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
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      closeModal();
    }
  });

  loadEntries();
}

document.addEventListener("DOMContentLoaded", init);
