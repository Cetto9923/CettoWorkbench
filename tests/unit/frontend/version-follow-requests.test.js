// 版本跟进页请求风暴回归测试。
// 加载 ui.js 与 version-follow.js 两个真实文件，复现真实耦合：
// renderWindows() 每次渲染都调 window.initAutocomplete(..., {value: state.windowId})，
// ui.js 的 selectAutocompleteItem() 会向隐藏域 #vfWindowId 派发 change 事件，
// version-follow.js 的 change 监听又去 fetchList()，形成自激循环（8098 实测 199 次）。
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

function makeClassList(node) {
  const set = new Set();
  return {
    add(...c) { c.forEach((x) => set.add(x)); },
    remove(...c) { c.forEach((x) => set.delete(x)); },
    contains(c) { return set.has(c); },
    toggle(c, force) { const on = force === undefined ? !set.has(c) : !!force; if (on) { set.add(c); } else { set.delete(c); } return on; },
    values: set,
  };
}

function makeElement(id, tag) {
  const el = {
    id: id || '',
    tagName: (tag || 'div').toUpperCase(),
    className: '',
    innerHTML: '',
    textContent: '',
    value: '',
    hidden: false,
    offsetHeight: 0,
    style: {},
    dataset: {},
    attrs: {},
    parentElement: null,
    listeners: {},
    classList: null,
  };
  el.classList = makeClassList(el);
  el.children = [];
  el.setAttribute = (k, v) => { el.attrs[k] = String(v); };
  el.getAttribute = (k) => (k in el.attrs ? el.attrs[k] : null);
  el.appendChild = (child) => { child.parentElement = el; el.children.push(child); return child; };
  el.insertBefore = (child) => { child.parentElement = el; el.children.unshift(child); return child; };
  el.remove = () => {
    if (el.parentElement) {
      const i = el.parentElement.children.indexOf(el);
      if (i >= 0) { el.parentElement.children.splice(i, 1); }
      el.parentElement = null;
    }
  };
  el.querySelector = () => null;
  el.querySelectorAll = () => [];
  el.closest = () => null;
  el.getBoundingClientRect = () => ({ left: 0, top: 0, right: 0, bottom: 0, width: 0, height: 0 });
  el.addEventListener = (type, cb) => { (el.listeners[type] ||= []).push(cb); };
  el.removeEventListener = () => {};
  el.dispatchEvent = (ev) => {
    const type = ev && ev.type;
    let node = el;
    while (node) {
      (node.listeners[type] || []).forEach((cb) => cb(ev));
      if (!ev || !ev.bubbles) { break; }
      node = node.parentElement;
    }
    return true;
  };
  return el;
}

// body 作为下拉浮层的挂载点。
const body = makeElement('body', 'body');
body.querySelector = (selector) => {
  const match = /\[data-autocomplete-for="([^"]+)"\]/.exec(selector);
  if (!match) { return null; }
  return body.children.find((c) => c.attrs['data-autocomplete-for'] === match[1]) || null;
};

const nodes = new Map();
function node(id, tag) {
  if (!nodes.has(id)) { nodes.set(id, makeElement(id, tag)); }
  return nodes.get(id);
}
const byId = (id) => node(id);

// 模板结构：搜索框与隐藏域同在 #vfWindowBar 内（见 web/templates/po/version_follow.html）。
const windowBar = node('vfWindowBar', 'header');
windowBar.appendChild(node('vfWindowSearch', 'input'));
windowBar.appendChild(node('vfWindowId', 'input'));
body.appendChild(windowBar);

const documentListeners = {};
const document = {
  body,
  readyState: 'complete',
  getElementById: byId,
  createElement: (tag) => makeElement('', tag),
  querySelector: () => null,
  querySelectorAll: () => [],
  addEventListener(type, cb) { (documentListeners[type] ||= []).push(cb); },
  removeEventListener() {},
};

const requests = [];
const STORM_CAP = 30;
let currentWindowId = 2;
function payload() {
  return {
    windowId: currentWindowId,
    windows: [
      { id: 2, name: 'V2', releaseDate: '2026-10-20', demandCount: 3 },
      { id: 3, name: 'V3', releaseDate: '2026-11-20', demandCount: 5 },
    ],
    stageCounts: [{ stage: '开发', count: 3 }],
    distanceDays: 7,
    onTrack: 3, risk: 0, blocked: 0,
    aiPilotOn: false,
    items: [{ demandNo: 'D1', title: 't', system: 's', priority: 'P1', stage: '开发', stayDays: 1, judgement: 'ontrack', owner: '张三', actionEnabled: false, actionLabel: 'x', findings: [] }],
    total: 1, pageSize: 20,
    details: [],
  };
}

const window = {
  document,
  addEventListener() {},
  removeEventListener() {},
  innerHeight: 900,
  appJson(url) {
    requests.push(url);
    // 请求风暴下切断循环，保证失败信息能打印、进程能退出。
    if (requests.length > STORM_CAP) { return new Promise(() => {}); }
    // 用宏任务兑现，便于测试中断失控的请求循环；同步兑现会饿死事件循环。
    return new Promise((resolve) => setImmediate(() => resolve({ data: payload() })));
  },
  showToast(msg, kind) { window.lastToast = msg + '/' + kind; },
};
const context = vm.createContext({
  window, document, console, URLSearchParams, Event, setTimeout, clearTimeout,
  Object, Array, String, Number, Promise, Math, JSON, RegExp, Error, TypeError,
});
vm.runInContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/ui.js'), 'utf8'), context);
vm.runInContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/version-follow.js'), 'utf8'), context);

const flush = () => new Promise((resolve) => setImmediate(resolve));
// 让出若干轮事件循环，确认没有后续请求被排进来。
const settle = async (rounds) => {
  for (let i = 0; i < (rounds || 10); i += 1) { await flush(); }
};

function fire(targetId, type, target) {
  const el = nodes.get(targetId);
  assert.ok(el, 'missing node ' + targetId);
  const ev = { type, target: target || el, preventDefault() {}, stopPropagation() {} };
  (el.listeners[type] || []).forEach((cb) => cb(ev));
}

const closest = (match) => ({ closest: (selector) => (match[selector] ? match[selector] : null) });

(async () => {
  await settle();
  assert.equal(requests.length, 1, '首次加载只应发出 1 次 items 请求，实际 ' + requests.length + ' 次：' + requests.join(' | '));
  assert.equal(nodes.get('vfWindowId').value, '2', '隐藏域应回填当前窗口');
  assert.equal(nodes.get('vfWindowSearch').value, 'V2 · 2026-10-20', '搜索框应显示当前窗口标签');

  const afterLoad = requests.length;
  await settle();
  assert.equal(requests.length, afterLoad, '首屏渲染不得再触发新请求');

  // 切换窗口（点击窗口条 chip）：只应新增 1 次请求。
  currentWindowId = 3;
  fire('vfWindowChips', 'click', closest({ '[data-window-id]': { dataset: { windowId: '3' } } }));
  assert.equal(requests.length, afterLoad + 1, '切换窗口只应发出 1 次请求，实际 ' + (requests.length - afterLoad) + ' 次');
  assert.match(requests.at(-1), /windowId=3/, '切换窗口应带上新 windowId');
  await settle();
  assert.equal(requests.length, afterLoad + 1, '切换窗口的响应渲染不得再触发新请求');

  // 切换阶段筛选：只应新增 1 次请求。
  const afterWindow = requests.length;
  fire('vfStageBar', 'click', closest({ '[data-stage]': { dataset: { stage: '开发' } } }));
  assert.equal(requests.length, afterWindow + 1, '切换阶段只应发出 1 次请求，实际 ' + (requests.length - afterWindow) + ' 次');
  await settle();
  assert.equal(requests.length, afterWindow + 1, '切换阶段的响应渲染不得再触发新请求');

  // 在搜索框选中另一个窗口（ui.js 派发 change）：只应新增 1 次请求。
  const afterStage = requests.length;
  nodes.get('vfWindowId').value = '2';
  fire('vfWindowId', 'change');
  assert.equal(requests.length, afterStage + 1, '下拉切换窗口只应发出 1 次请求，实际 ' + (requests.length - afterStage) + ' 次');
  await settle();
  assert.equal(requests.length, afterStage + 1, '下拉切换窗口的响应渲染不得再触发新请求');

  console.log('PASS version-follow requests: load / window switch / stage filter each send exactly 1 items request');
})().catch((err) => { console.error(err); process.exit(1); });