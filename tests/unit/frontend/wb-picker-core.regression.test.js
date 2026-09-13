#!/usr/bin/env node
// shared/picker Core 回归：mount / single / clear / select value bridge.
const path = require('path');
const fs = require('fs');
const vm = require('vm');

function assert(cond, msg) {
  if (!cond) { console.error('FAIL:', msg); process.exit(1); }
}

class Node {
  constructor(tag) {
    this.tagName = tag.toUpperCase();
    this.children = [];
    this.parentNode = null;
    this.attributes = {};
    this.className = '';
    this.eventHandlers = {};
    this.style = {};
    this.textContent = '';
    this.disabled = false;
    this.value = '';
  }
  appendChild(node) { node.parentNode = this; this.children.push(node); return node; }
  removeChild(node) {
    const idx = this.children.indexOf(node);
    if (idx >= 0) this.children.splice(idx, 1);
    if (node) node.parentNode = null;
    return node;
  }
  insertBefore(node, next) {
    node.parentNode = this;
    const idx = this.children.indexOf(next);
    if (idx < 0) this.children.push(node); else this.children.splice(idx, 0, node);
    return node;
  }
  remove() {
    if (!this.parentNode) return;
    const idx = this.parentNode.children.indexOf(this);
    if (idx >= 0) this.parentNode.children.splice(idx, 1);
  }
  setAttribute(k, v) { this.attributes[k] = String(v); if (k === 'data-value') this.datasetValue = String(v); }
  getAttribute(k) { return this.attributes[k] || ''; }
  addEventListener(type, fn) { (this.eventHandlers[type] = this.eventHandlers[type] || []).push(fn); }
  dispatchEvent(evt) { (this.eventHandlers[evt.type] || []).forEach((fn) => fn.call(this, evt)); }
  focus() {}
  select() {}
  get classList() {
    const self = this;
    return {
      add: (...xs) => { const set = new Set((self.className || '').split(/\s+/).filter(Boolean)); xs.forEach((x) => set.add(x)); self.className = Array.from(set).join(' '); },
      remove: (...xs) => { const set = new Set((self.className || '').split(/\s+/).filter(Boolean)); xs.forEach((x) => set.delete(x)); self.className = Array.from(set).join(' '); },
      contains: (x) => (self.className || '').split(/\s+/).includes(x)
    };
  }
  getBoundingClientRect() { return { left: 10, top: 10, bottom: 42, width: 180 }; }
  set innerHTML(html) {
    this.children = [];
    this._html = html;
    if (html.indexOf('wb-picker-label') >= 0) {
      const label = new Node('span'); label.className = 'wb-picker-label'; this.appendChild(label);
      const caret = new Node('span'); caret.className = 'wb-picker-caret'; this.appendChild(caret);
    }
    if (html.indexOf('wb-picker-search') >= 0) {
      const wrap = new Node('div'); wrap.className = 'wb-picker-search-wrap';
      const input = new Node('input'); input.className = 'wb-picker-search'; wrap.appendChild(input); this.appendChild(wrap);
      const list = new Node('div'); list.className = 'wb-picker-list'; this.appendChild(list);
      const footer = new Node('div'); footer.className = 'wb-picker-footer';
      const clear = new Node('button'); clear.className = 'wb-picker-clear'; footer.appendChild(clear); this.appendChild(footer);
    }
    if (html.indexOf('wb-picker-option') >= 0) {
      const re = /data-value="([^"]*)"/g; let m;
      while ((m = re.exec(html))) {
        const b = new Node('button'); b.className = 'wb-picker-option'; b.setAttribute('data-value', m[1]);
        // 简易同步：每个 option 都按真实 html 配 avatar / sub，便于回归里 querySelectorAll('.wb-picker-avatar') 能命中。
        const blockRe = new RegExp('data-value="' + m[1].replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '"[\\s\\S]*?</button>');
        const block = (html.match(blockRe) || [''])[0];
        if (block.indexOf('wb-picker-avatar') >= 0) {
          const av = new Node('span'); av.className = 'wb-picker-avatar'; av.textContent = (block.match(/wb-picker-avatar[^>]*>([^<]+)/) || [])[1] || '';
          b.appendChild(av);
        }
        const main = new Node('span'); main.className = 'wb-picker-main'; b.appendChild(main);
        const sub = new Node('span'); sub.className = 'wb-picker-sub';
        sub.textContent = (block.match(/wb-picker-sub[^>]*>([^<]+)/) || [])[1] || '';
        main.appendChild(sub);
        this.appendChild(b);
      }
    }
    if (html.indexOf('wb-picker-empty') >= 0) { const e = new Node('div'); e.className = 'wb-picker-empty'; this.appendChild(e); }
  }
  get innerHTML() { return this._html || ''; }
  querySelector(sel) {
    return this.querySelectorAll(sel)[0] || null;
  }
  querySelectorAll(sel) {
    const out = [];
    function walk(n) {
      n.children.forEach((c) => {
        if (sel[0] === '.' && (c.className || '').split(/\s+/).includes(sel.slice(1))) out.push(c);
        if (sel === 'select[data-wb-person-picker]' && c.tagName === 'SELECT' && c.getAttribute('data-wb-person-picker')) out.push(c);
        walk(c);
      });
    }
    walk(this);
    return out;
  }
}
class OptionNode extends Node {
  constructor(value, label) { super('option'); this.value = value; this.textContent = label; }
}
class SelectNode extends Node {
  constructor() { super('select'); this.options = []; }
  appendChild(node) { super.appendChild(node); if (node.tagName === 'OPTION') this.options.push(node); return node; }
}

const ctx = {
  console,
  setTimeout: (fn) => fn(),
  requestAnimationFrame: (fn) => fn(),
  window: {
    innerWidth: 1024,
    innerHeight: 768,
    addEventListener() {},
    removeEventListener() {}
  },
  document: {
    body: new Node('body'),
    readyState: 'complete',
    createElement(tag) { return tag === 'select' ? new SelectNode() : new Node(tag); },
    addEventListener() {},
    removeEventListener() {},
    querySelector() { return null; },
    querySelectorAll() { return []; }
  },
  Event: function Event(type) { this.type = type; },
  MutationObserver: function MutationObserver() { this.observe = function () {}; },
  fetch: null
};
ctx.window.document = ctx.document;
ctx.window.Event = ctx.Event;
ctx.window.MutationObserver = ctx.MutationObserver;
ctx.globalThis = ctx.window;
vm.createContext(ctx);
// Load the same shared UI provider used by layout/base.html.
vm.runInContext(fs.readFileSync(path.resolve(__dirname, '../../../web/static/js/components/autocomplete-options.js'), 'utf8'), ctx);
vm.runInContext(fs.readFileSync(path.resolve(__dirname, '../../../web/static/js/ui.js'), 'utf8'), ctx);

const root = path.resolve(__dirname, '../../..');
['vendor/pinyin-lite.js', 'wb-picker-search.js', 'wb-picker.js', 'wb-directory.js', 'adapters/wb-person-picker.js'].forEach((rel) => {
  vm.runInContext(fs.readFileSync(path.join(root, 'web/static/js/picker', rel), 'utf8'), ctx);
});

const host = new Node('div');
ctx.document.body.appendChild(host);
let changed = '';
const picker = ctx.window.WbPicker.mount({
  el: host,
  options: [
    { value: 'alice', label: '张伟（alice）', avatar: true, avatarName: '张伟', meta: { account: 'alice', realname: '张伟' } },
    { value: 'bob', label: '王芳（bob）', avatar: true, avatarName: '王芳', meta: { account: 'bob', realname: '王芳' } }
  ],
  onChange: (v) => { changed = v; }
});
assert(picker.getValue() === '', 'initial value empty');
host.querySelector('.wb-picker-trigger').dispatchEvent({ type: 'click', preventDefault() {} });
function findPanel() {
  const kids = ctx.document.body.children || [];
  for (let i = 0; i < kids.length; i++) {
    const n = kids[i];
    if (n && String(n.className || '').indexOf('wb-picker-panel') >= 0) return n;
  }
  return null;
}
const openPanel = findPanel();
assert(openPanel && String(openPanel.className).indexOf('is-open') >= 0, 'panel portals to body and opens');
assert(openPanel.querySelectorAll('.wb-picker-option').length >= 2, 'options actually render without WBUtils');
const searchInput = openPanel.querySelector('.wb-picker-search');
searchInput.value = 'bob';
searchInput.dispatchEvent({ type: 'input' });
assert(openPanel.querySelectorAll('.wb-picker-option').every((o) => o.getAttribute('data-value') === 'bob'), 'search filters account');
assert(openPanel.querySelectorAll('.wb-picker-option').length > 0, 'search retains matching option');
searchInput.value = '';
searchInput.dispatchEvent({ type: 'input' });
openPanel.querySelectorAll('.wb-picker-option')[0].dispatchEvent({ type: 'click' });
assert(picker.getValue() === 'alice', 'single select stores value');
assert(changed === 'alice', 'onChange receives value');
// reopen to clear (single-select closes after choose)
host.querySelector('.wb-picker-trigger').dispatchEvent({ type: 'click', preventDefault() {} });
const panelForClear = findPanel();
assert(panelForClear, 'panel still portaled after reopen');
panelForClear.querySelector('.wb-picker-clear').dispatchEvent({ type: 'click' });
assert(picker.getValue() === '', 'clear resets value');

const select = new SelectNode();
select.setAttribute('data-wb-person-picker', '1');
select.setAttribute('data-wb-directory-source', 'all');
select.appendChild(new OptionNode('张伟', '张伟'));
select.value = '张伟';
ctx.document.body.appendChild(select);
ctx.window.WbDirectory.users = () => Promise.resolve([
  { account: 'alice', realname: '张伟', deptName: '研发部' },
  { account: 'sunming', realname: '孙明', deptName: '投资管理行（外派）' }
]);
ctx.window.WbPersonPicker.mount({ el: select });
setImmediate(async () => {
  assert(select.__wbPersonPicker, 'adapter attaches api');
  assert(select.value === 'alice', 'adapter resolves unique realname to account');
  // 人员控件统一：副行只显示部门，不重复 account；头像必须有
  const adapterHost = ctx.document.body.children.find((n) => n.classList.contains('wb-person-picker-host'));
  adapterHost.querySelector('.wb-picker-trigger').dispatchEvent({ type: 'click', preventDefault() {} });
  const adapterPanel = ctx.document.body.children.find((n) => n.classList.contains('wb-picker-panel') && n.classList.contains('is-open'));
  const opts = adapterPanel.querySelectorAll('.wb-picker-option');
  assert(opts.length > 0, 'adapter avatar assertions inspect populated portal');
  let anySubHasAccount = false;
  let anyAvatarMissing = false;
  opts.forEach(function (o) {
    const sub = o.querySelector('.wb-picker-sub');
    if (sub && /alice/.test(sub.textContent)) anySubHasAccount = true;
    if (!o.querySelector('.wb-picker-avatar')) anyAvatarMissing = true;
  });
  assert(!anySubHasAccount, 'person sub row no longer suffixes account');
  assert(!anyAvatarMissing, 'person option must render avatar');
  let changes = 0;
  select.addEventListener('change', () => changes++);
  opts.find((o) => o.getAttribute('data-value') === 'sunming').dispatchEvent({ type: 'click' });
  assert(select.value === 'sunming', 'selection synchronizes hidden native select account');
  assert(changes === 1, 'selection emits one change event');
  adapterHost.querySelector('.wb-picker-trigger').dispatchEvent({ type: 'click', preventDefault() {} });
  adapterPanel.querySelector('.wb-picker-clear').dispatchEvent({ type: 'click' });
  assert(select.value === '' && changes === 2, 'clear synchronizes hidden value and emits change');
  select.__wbPersonPicker.setValue('alice');
  assert(select.value === 'alice', 'programmatic setValue synchronizes native select');
  // Preserve the adapter's current compatibility contract: ambiguous/unknown
  // legacy values remain unchanged; never guess a different person's account.
  const users = [
    { account: 'lina01', realname: '李娜' },
    { account: 'lina02', realname: '李娜' }
  ];
  for (const value of ['李娜', '未知姓名', 'lina01']) {
    const legacySelect = new SelectNode();
    legacySelect.value = value;
    ctx.document.body.appendChild(legacySelect);
    ctx.window.WbPersonPicker.mount({ el: legacySelect, users });
    await new Promise(setImmediate);
    assert(legacySelect.value === value, 'ambiguous/unknown values retained and exact account preserved: ' + value);
  }
  let toasted = '';
  ctx.window.showToast = (msg, type) => { toasted = String(msg) + '|' + String(type); };
  ctx.window.WbDirectory.users = () => Promise.reject(new Error('network'));
  let rejected = false;
  try {
    await ctx.window.WbPersonPicker.prefetchUsers();
  } catch (_) {
    rejected = true;
  }
  assert(rejected, 'prefetchUsers rejects on directory failure');
  assert(/人员目录加载失败/.test(toasted) && /error/.test(toasted), 'prefetchUsers toasts on directory failure');
  console.log('shared/picker core regression passed');
});
