const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const events = new Map();
const requests = [];
const urls = [];
let ready;
function $(selector) {
  if (typeof selector === 'function') { ready = selector; return; }
  const obj = {
    length: 1,
    on(event, fn) { events.set(`${selector}:${event}`, fn); return this; },
    attr(key, value) {
      if (value !== undefined) return this;
      if (typeof selector === 'object') return selector[key];
      if (key === 'data-vs-status') return 'testing';
      return '';
    },
    first() { return this; }, removeClass() { return this; }, addClass() { return this; },
    removeAttr() { return this; }, prop() { return this; }, html() { return this; },
    text() { return this; },
    data(key) {
      if (typeof selector === 'object') return selector[key];
      return '';
    }
  };
  return obj;
}
const window = {
  location: { pathname: '/home', search: '?status=testing&page=4' },
  history: { replaceState(_, __, url) { urls.push(url); } },
  appFetch(url) { requests.push(url); return new Promise(() => {}); },
  // pageSize 记忆（Stage 4）在沙箱中预装，避免 home.js 读取 loadPageSize 触发 TypeError。
  PersonalList: {
      PAGE_SIZE_OPTIONS: [10, 12, 15, 20, 50, 100],
    escapeHtml: (v) => String(v == null ? '' : v),
    loadPageSize: (_key, fallback) => fallback,
    savePageSize() {},
    renderPagination() {},
    createController: () => ({ bind() {}, destroy() {} })
  }
};
vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/home.js'), 'utf8'), {
  window, jQuery: $, URLSearchParams, document: {}
});
ready();
const click = events.get('#homeQuickChips [data-home-focus]:click');
assert.equal(typeof click, 'function');
for (const focus of ['today', 'blocked', 'overdue', 'suspended', 'all']) {
  let prevented = false;
  click.call({'data-home-focus': focus}, {preventDefault() { prevented = true; }});
  assert.ok(prevented);
  const query = new URL(requests.at(-1), 'http://localhost').searchParams;
  assert.equal(query.get('focus'), focus);
  assert.equal(query.get('status'), 'testing', 'focus must retain the stage');
  assert.equal(query.get('page'), '1', 'focus must reset pagination');
  assert.equal(new URL(urls.at(-1), 'http://localhost').pathname, '/home');
}
const html = fs.readFileSync(path.join(__dirname, '../../../web/templates/po/home.html'), 'utf8');
assert.ok(!html.includes('/todos?focus='), 'homepage focus must not link to todos');
const homeSource = fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/home.js'), 'utf8');
assert.match(homeSource, /renderValueStreamSummary\(res\.stageSummary, state\.focus\)/,
  'homepage list response must render stage summary with the current focus');
for (const focus of ['today','blocked','overdue','suspended']) {
  assert.match(html, new RegExp('<button[^>]+data-home-focus="'+focus+'"'));
}

// 首页“我的关系”工具栏按钮校验：切为“我牵头”和“我参与”
assert.match(html, /<div[^>]+id="homeRelationSegment"/);
assert.match(html, /data-relation="lead">我牵头<\/button>/);
assert.match(html, /data-relation="participate">我参与<\/button>/);
assert.ok(!html.includes('data-relation="handling"'), 'handling relation must be replaced');
assert.ok(!html.includes('data-relation="following"'), 'following relation must be replaced');

const relationClick = events.get('#homeRelationSegment button:click');
assert.equal(typeof relationClick, 'function');
for (const rel of ['lead', 'participate']) {
  relationClick.call({'relation': rel});
  const query = new URL(requests.at(-1), 'http://localhost').searchParams;
  assert.equal(query.get('relation'), rel);
  assert.equal(query.get('page'), '1', 'relation change must reset pagination');
}

console.log('PASS: home focus stays on home, preserves stage, resets page, requests server filtering');
