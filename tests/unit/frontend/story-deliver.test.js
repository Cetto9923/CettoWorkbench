// =============================================================================
// 文件: tests/unit/frontend/story-deliver.test.js
// 职责: 执行交付弹窗请求链，验证对象区分、重试、过期响应和提交失败保留输入。
// =============================================================================
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const nodes = new Map(), events = new Map(), requests = [], toasts = [];
let refreshes = 0;
function node(id) {
  if (!nodes.has(id)) nodes.set(id, {value: '', style: {}, textContent: '', disabled: false});
  return nodes.get(id);
}
function jquery(selector) {
  if (typeof selector === 'function') { selector(); return; }
  const el = typeof selector === 'string' ? node(selector) : selector;
  const chain = {
    on(event, handler) { events.set(event + ':' + selector, handler); return chain; },
    text(value) { el.textContent = value; return chain; },
    addClass() { el.open = true; return chain; },
    removeClass() { el.open = false; return chain; },
    hasClass() { return !!el.open; },
    css() { return chain; }, attr() { return chain; },
    prop(name, value) { el[name] = value; return chain; },
    show() { el.visible = true; return chain; }, hide() { el.visible = false; return chain; }
  };
  return chain;
}
const window = {
  escapeHtml: String, showToast: (message, type) => toasts.push({message, type}),
  refreshPoHomeDemands: () => refreshes++,
  fetch(url, options) {
    return new Promise(resolve => requests.push({url, options, resolve}));
  }
};
const document = {
  getElementById: node, querySelectorAll: () => [], querySelector: () => ({value: '0'})
};
vm.runInNewContext(fs.readFileSync('web/static/js/po/po-deliver.js', 'utf8'), {
  window, document, $: jquery, setTimeout: () => 0
});
const flush = () => new Promise(resolve => setImmediate(resolve));
function respond(request, body, ok = true) {
  request.resolve({ok, json: async () => body});
}
function meta(id) {
  return {success: true, displayId: String(id), windowId: 9, windowName: '十月窗口', releaseDate: '2099-10-20',
    deliverDate: '2099-10-20', verifyDate: '1', verifyPlan: '生产验证', verifier: 'pm',
    precheck: {canSubmit: true, rows: []}, windows: [{id: 9, name: '十月窗口', releaseDate: '2099-10-20'}]};
}
async function verifyRetryAndStaleResponse() {
  window.openPoDeliverModal('US1');
  respond(requests.at(-1), meta(1)); await flush();
  window.openPoDeliverModal('2', {kind: 'story'});
  assert.equal(window._poDeliverCtx, null);
  assert.equal(requests.at(-1).url, '/stories/2/deliver');
  respond(requests.at(-1), {message: '加载失败'}, false); await flush();
  events.get('click:#poDeliverRetryBtn')();
  assert.equal(requests.at(-1).url, '/stories/2/deliver');
  const stale = requests.at(-1);
  window.openPoDeliverModal('3', {kind: 'story'});
  respond(requests.at(-1), meta(3)); await flush();
  respond(stale, meta(2)); await flush();
  assert.equal(window._poDeliverCtx.demandId, '3');
  window.closePoDeliverModal();
  window.openPoDeliverModal('4', {kind: 'story'});
  const closed = requests.at(-1);
  window.closePoDeliverModal(); respond(closed, meta(4)); await flush();
  assert.equal(window._poDeliverCtx, null);
}
async function verifySubmit() {
  window.openPoDeliverModal('5', {kind: 'story'});
  respond(requests.at(-1), meta(5)); await flush();
  node('poDeliverGrayPlan').value = '0';
  const before = requests.length;
  window.submitPoDeliverModal(); window.submitPoDeliverModal();
  assert.equal(requests.length, before + 1);
  assert.equal(requests.at(-1).url, '/stories/5/deliver');
  assert.equal(requests.at(-1).options.method, 'POST');
  assert.equal(JSON.parse(requests.at(-1).options.body).verifier, 'pm');
  respond(requests.at(-1), {success: false, message: '字段错误'}, false); await flush();
  assert.equal(node('poDeliverVerifyPlan').value, '生产验证');
  assert.equal(node('poDeliverConfirmBtn').disabled, false);
  assert.equal(window._poDeliverCtx.demandId, '5');
  assert.equal(refreshes, 0);
  assert.equal(toasts.at(-1).type, 'error');
  window.submitPoDeliverModal();
  respond(requests.at(-1), {success: true, redirectUrl: '/home'}); await flush();
  assert.equal(window._poDeliverCtx, null);
  assert.equal(refreshes, 1);
  assert.equal(toasts.at(-1).type, 'success');
}
async function verifyStaleSubmit() {
  window.openPoDeliverModal('6', {kind: 'story'});
  respond(requests.at(-1), meta(6)); await flush();
  window.submitPoDeliverModal();
  const stale = requests.at(-1);
  window.closePoDeliverModal();
  window.openPoDeliverModal('7', {kind: 'story'});
  respond(requests.at(-1), meta(7)); await flush();
  respond(stale, {success: true}); await flush();
  assert.equal(window._poDeliverCtx.demandId, '7');
  assert.equal(refreshes, 1);
}
verifyRetryAndStaleResponse().then(verifySubmit).then(verifyStaleSubmit).then(() => {
  console.log('story delivery routing, retry, stale response and submission checks passed');
}).catch(error => { console.error(error); process.exitCode = 1; });
