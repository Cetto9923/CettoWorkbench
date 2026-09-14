#!/usr/bin/env node
const assert = require("assert");
const fs = require("fs");
const vm = require("vm");

const source = fs.readFileSync("web/static/js/po/done.js", "utf8");
const item = {
  id: 1, objectType: "task", objectId: 2, objectTitle: "任务", actionName: "完成任务",
  handledAt: "2026-09-07T12:00:00Z", resultText: "已完成", beforeStatus: "doing",
  afterStatus: "done", currentStatus: "developing"
};
const demandItem = {
  id: 2, objectType: "demand", objectId: 63319, objectTitle: "需求", actionName: "评审需求",
  handledAt: "2026-09-07T12:00:00Z", resultText: "已通过", beforeStatus: "wait",
  afterStatus: "active", currentStatus: "active"
};

function stubSelect() {
  return { disabled: false, innerHTML: "", addEventListener() {} };
}

function run(fetchImpl, withMetaSelects) {
  const listeners = {};
  const tbody = { innerHTML: "", querySelectorAll: () => [] };
  const selects = withMetaSelects
    ? { doneAction: stubSelect(), doneResult: stubSelect(), doneProject: stubSelect() }
    : {};
  const elements = Object.assign({
    doneTbody: tbody,
    doneEmpty: { hidden: true },
    doneError: { hidden: true },
    donePagination: { hidden: true },
    doneKeyword: { value: "", addEventListener() {} }
  }, selects);
  const toasts = [];
  const context = {
    document: {
      addEventListener: (n, fn) => { listeners[n] = fn; },
      getElementById: (id) => elements[id] || null,
      querySelectorAll: () => []
    },
    window: {
      escapeHtml: (v) => String(v == null ? "" : v),
      PersonalList: { loadPageSize: () => 20, escapeHtml: (v) => String(v == null ? "" : v), PAGE_SIZE_OPTIONS: [10, 15, 20, 50, 100] },
      showToast: (msg, type) => { toasts.push({ msg, type }); }
    },
    URLSearchParams,
    fetch: fetchImpl,
    setTimeout: () => 0,
    clearTimeout() {}
  };
  vm.runInNewContext(source, context);
  listeners.DOMContentLoaded();
  return {
    tbody, selects, toasts,
    flush: () => new Promise((r) => setImmediate(() => setImmediate(r)))
  };
}

function itemsFetch(url) {
  if (String(url).indexOf("/done/items") === 0) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ data: { items: [item, demandItem], total: 2, summary: {}, facets: [] } })
    });
  }
  return Promise.resolve({ ok: true, json: () => Promise.resolve({ data: {} }) });
}

(async function () {
  const labels = run(itemsFetch, false);
  await labels.flush();
  assert.match(labels.tbody.innerHTML, /进行中/);
  assert.match(labels.tbody.innerHTML, /已完成/);
  assert.match(labels.tbody.innerHTML, /开发中/);
  assert.match(labels.tbody.innerHTML, /待评审/);
  assert.match(labels.tbody.innerHTML, /已评审/);
  assert.doesNotMatch(labels.tbody.innerHTML, /已激活/);
  assert.doesNotMatch(labels.tbody.innerHTML, />doing</);
  assert.doesNotMatch(labels.tbody.innerHTML, />developing</);

  const ok = run((url) => {
    if (String(url).indexOf("/done/meta") === 0) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({
          data: {
            actionTypes: [{ key: "finish", label: "完成" }],
            results: [{ key: "done", label: "已完成" }],
            projects: [{ key: "1", label: "项目A" }]
          }
        })
      });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve({ data: { items: [], total: 0, summary: {}, facets: [] } }) });
  }, true);
  await ok.flush();
  assert.equal(ok.toasts.length, 0);
  assert.equal(ok.selects.doneAction.disabled, false);
  assert.match(ok.selects.doneAction.innerHTML, /完成/);

  const fail = run((url) => {
    if (String(url).indexOf("/done/meta") === 0) return Promise.reject(new Error("network"));
    return Promise.resolve({ ok: true, json: () => Promise.resolve({ data: { items: [], total: 0, summary: {}, facets: [] } }) });
  }, true);
  await fail.flush();
  assert.equal(fail.toasts.length, 1);
  assert.equal(fail.toasts[0].msg, "筛选条件加载失败，请稍后重试");
  assert.equal(fail.selects.doneAction.disabled, true);
  assert.match(fail.selects.doneAction.innerHTML, /全部操作/);

  console.log("PASS: done status labels + loadMeta failure toast");
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
