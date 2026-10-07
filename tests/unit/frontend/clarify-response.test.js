// =============================================================================
// 文件: tests/unit/frontend/clarify-response.test.js
// 职责: 执行真实澄清事件与响应链，锁定单次打开及422错误处理。
// =============================================================================
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const handlers = new Map(), listeners = [], requests = [], toasts = [];
const state = new Map();
let refreshes = 0, drawerCloses = 0;
function jquery(selector) {
  if (typeof selector === 'function') { selector(); return; }
  const key = typeof selector === 'string' ? selector : 'row';
  const values = state.get(key) || {};
  state.set(key, values);
  const chain = new Proxy({
    on(event, target, handler) { handlers.set(event + ':' + target, handler); return chain; },
    val(value) {
      if (value !== undefined) { values.value = value; return chain; }
      if (values.value !== undefined) return values.value;
      return key.includes('mainSystemRadio') ? '0' : '1';
    },
    text(value) { values.text = value; return chain; },
    attr(name, value) { if (value !== undefined) { values[name] = value; return chain; } return '0'; },
    prop(name, value) { values[name] = value; return chain; },
    addClass() { values.open = true; return chain; },
    removeClass() { values.open = false; return chain; },
    hasClass() { return !!values.open; },
    is() { return false; },
    each(callback) { if (key === '#poClarifyProductTbody tr') callback.call({}, 0); return chain; },
    find(name) { return jquery(name); }
  }, { get(target, prop) { return prop in target ? target[prop] : () => chain; } });
  return chain;
}
const window = {
  escapeHtml: String, showToast: (message, type) => toasts.push({message, type}),
  DemandDetail: { close: () => drawerCloses++ }, refreshPoHomeDemands: () => refreshes++,
  fetch: async (url, options) => {
    requests.push({url, method: options.method});
    return {ok: false, status: 422, json: async () => ({message: '禅道拒绝澄清'})};
  }
};
const document = {addEventListener: (event, handler, capture) => listeners.push({event, handler, capture})};
vm.runInNewContext(fs.readFileSync('web/static/js/po/demand-clarify.js', 'utf8'), {
  window, document, $: jquery, setTimeout: () => 0
});
async function verify() {
  const clicks = listeners.filter(listener => listener.event === 'click');
  assert.equal(clicks.length, 1);
  assert.equal(clicks[0].capture, true);
  clicks[0].handler({preventDefault() {}, target: {closest: () => ({getAttribute: () => 'US63360'})}});
  await new Promise(resolve => setImmediate(resolve));
  assert.deepEqual(requests, [{url: '/demands/63360/clarify', method: 'GET'}]);
  const submit = handlers.get('submit:#poDemandClarifyForm');
  submit({preventDefault() {}});
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(requests.filter(request => request.method === 'POST').length, 1);
  assert.deepEqual(toasts, [{message: '禅道拒绝澄清', type: 'error'}]);
  assert.equal(state.get('#poDemandClarifyModal').open, true);
  assert.equal(state.get('#poClarifySubmitBtn').disabled, false);
  assert.equal(state.get('#poClarifyCancelBtn').disabled, false);
  assert.equal(refreshes, 0);
  assert.equal(drawerCloses, 0);
  console.log('clarify single GET and 422 behavior passed');
}
verify().catch(error => { console.error(error); process.exitCode = 1; });
