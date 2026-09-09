const assert = require("assert");
const fs = require("fs");
const vm = require("vm");

const source = fs.readFileSync("web/static/js/po/done.js", "utf8");
const listeners = {};
const tbody = { innerHTML: "", querySelectorAll: () => [] };
const elements = {
  doneTbody: tbody,
  doneEmpty: { hidden: false },
  doneError: { hidden: false },
  donePagination: { hidden: false }
};
const document = {
  addEventListener: (name, fn) => { listeners[name] = fn; },
  getElementById: id => elements[id] || null,
  querySelectorAll: () => []
};
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
const context = {
  document,
  window: { PersonalList: { loadPageSize: () => 20 } },
  URLSearchParams,
  fetch: url => Promise.resolve({ ok: true, json: () => Promise.resolve(String(url).indexOf("/done/items") === 0 ? { data: { items: [item, demandItem], total: 2, summary: {}, facets: [] } } : { data: {} }) })
};

vm.runInNewContext(source, context);
listeners.DOMContentLoaded();
setImmediate(() => setImmediate(() => {
  assert.match(tbody.innerHTML, /进行中/);
  assert.match(tbody.innerHTML, /已完成/);
  assert.match(tbody.innerHTML, /开发中/);
  assert.match(tbody.innerHTML, /待评审/);
  assert.match(tbody.innerHTML, /已评审/);
  assert.doesNotMatch(tbody.innerHTML, /已激活/);
  assert.doesNotMatch(tbody.innerHTML, />doing</);
  assert.doesNotMatch(tbody.innerHTML, />developing</);
  console.log("PASS: done list maps ZenTao status codes to Chinese labels");
}));
