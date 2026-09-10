const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Small DOM fixture: exercise the public API and actual registered handlers.
class Element {
  constructor(tag = 'div') {
    this.tag = tag; this.children = []; this.attrs = {}; this.listeners = {};
    this.style = {}; this.value = ''; this.className = ''; this.textContent = '';
    this.classList = {
      contains: c => this.className.split(' ').includes(c),
      add: c => { if (!this.classList.contains(c)) this.className += ' ' + c; },
      remove: c => { this.className = this.className.split(' ').filter(x => x !== c).join(' '); },
      toggle: (c, on) => { this.classList[on ? 'add' : 'remove'](c); }
    };
  }
  set innerHTML(v) { this.children.forEach(c => { c.parentElement = null; }); this.children = []; }
  appendChild(c) { if (c.parentElement) c.remove(); this.children.push(c); c.parentElement = this; return c; }
  insertBefore(c, before) { this.children.splice(this.children.indexOf(before), 0, c); c.parentElement = this; }
  remove() { if (this.parentElement) this.parentElement.children = this.parentElement.children.filter(c => c !== this); this.parentElement = null; }
  setAttribute(k, v) { this.attrs[k] = v; }
  matches(s) {
    const cls = s.match(/^\.([\w-]+)/), attr = s.match(/\[([^=]+)="([^"]+)"\]/);
    return (!cls || this.classList.contains(cls[1])) && (!attr || this.attrs[attr[1]] === attr[2]);
  }
  closest(s) { return this.matches(s) ? this : this.parentElement?.closest(s); }
  querySelectorAll(s) { return this.children.flatMap(c => [...(c.matches(s) ? [c] : []), ...c.querySelectorAll(s)]); }
  querySelector(s) { return this.querySelectorAll(s)[0] || null; }
  addEventListener(k, f) { (this.listeners[k] ||= []).push(f); }
  removeEventListener(k, f) { this.listeners[k] = (this.listeners[k] || []).filter(x => x !== f); }
  dispatchEvent(e) { (this.listeners[e.type] || []).slice().forEach(f => f(e)); }
  fire(type, key) { this.dispatchEvent({ type, key, preventDefault() {}, stopPropagation() {} }); }
  focus() { this.fire('focus'); }
  getBoundingClientRect() { return { left: 10, bottom: 40, width: 240 }; }
  scrollIntoView() {}
}
const body = new Element(), input = new Element('input'), hidden = new Element('input'), host = new Element();
body.appendChild(host); host.appendChild(input); host.appendChild(hidden);
const doc = new Element();
doc.body = body; doc.createElement = tag => new Element(tag);
doc.getElementById = id => ({ picker: input, account: hidden }[id]);
const win = new Element();
const sandbox = { window: win, document: doc, Event: class { constructor(type) { this.type = type; } } };
for (const file of ['components/autocomplete-options.js', 'ui.js']) {
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js', file), 'utf8'), sandbox);
}
const people = [{ value: '003030', label: '程统', dept: '组织变革团队', title: '项目管理岗', badge: '相关人员' },
  ...Array.from({ length: 24 }, (_, i) => ({ value: 'u' + i, label: '用户' + i }))];
const dropdown = () => body.querySelector('.ui-autocomplete-dropdown');
const options = () => dropdown().querySelectorAll('.ui-autocomplete-option');
win.initUserPicker('picker', 'account', people);
input.focus();
assert.equal(dropdown().parentElement, body, 'uses body portal');
assert.equal(options().length, 20);
assert.equal(dropdown().querySelector('.ui-autocomplete-hint').textContent, '还有 5 个选项没有显示，可尝试搜索来查找');
assert.equal(options()[0].querySelector('.ui-autocomplete-avatar').textContent, '程');
assert.equal(options()[0].querySelector('.ui-autocomplete-main').textContent, '程统(003030)');
assert.equal(options()[0].querySelector('.ui-autocomplete-dept').textContent, ' 组织变革团队');
assert.equal(options()[0].querySelector('.ui-autocomplete-meta').textContent, '项目管理岗 相关人员');
const tone = options()[0].querySelector('.ui-autocomplete-avatar').attrs['data-tone'];
for (const query of ['程统', '003030', '组织变革', '项目管理', '相关人员']) {
  input.value = query; input.fire('input'); assert.equal(options().length, 1, query);
}
input.fire('keydown', 'ArrowDown'); input.fire('keydown', 'ArrowUp'); input.fire('keydown', 'Enter');
assert.equal(hidden.value, '003030'); assert.equal(input.value, '程统(003030)');
win.destroyUserPicker('picker');
assert.equal(body.querySelector('.ui-autocomplete-dropdown'), null);
assert.equal(input.listeners.input.length, 0, 'destroy unbinds old state');
win.initUserPicker('picker', 'account', people, { value: '003030' });
input.focus();
assert.equal(options()[0].querySelector('.ui-autocomplete-avatar').attrs['data-tone'], tone);
assert.equal(options()[0].attrs['aria-selected'], 'true');
let changes = 0; hidden.addEventListener('change', () => changes++);
host.querySelector('.ui-autocomplete-clear').fire('click');
assert.equal(hidden.value, ''); assert.equal(changes, 1, 'no duplicate listeners on reopen');
assert.equal(options().length, 20);
input.value = 'no such user'; input.fire('input');
assert.equal(dropdown().querySelector('.ui-autocomplete-empty').textContent, '无匹配结果');
win.initUserPicker('picker', 'account', [], { value: '' }); input.focus();
assert.equal(dropdown().querySelector('.ui-autocomplete-empty').textContent, '暂无可选用户');
win.initUserPicker('picker', 'account', [{ value: '7', label: '<张>', selectedLabel: '张(7) 特别展示' }], { value: '7', maxShow: 1 });
assert.equal(input.value, '张(7) 特别展示');
input.fire('input'); assert.equal(options().length, 1, 'selectedLabel is searchable');
assert.equal(options()[0].querySelector('.ui-autocomplete-content').children[0].textContent, '<张>(7)', 'text is never injected as HTML');
win.initAutocomplete('picker', 'account', [{ value: '805', label: '智慧产品', badge: '参与产品' }], { value: '' });
input.value = '805'; input.fire('input');
assert.equal(options().length, 1); assert.equal(options()[0].querySelector('.ui-autocomplete-avatar'), null);
assert.equal(options()[0].querySelector('.ui-autocomplete-badge').textContent, '参与产品');
options()[0].fire('click'); assert.equal(hidden.value, '805'); assert.equal(input.value, '智慧产品(805)');
console.log('PASS: UserPicker rendering/filter/hint/keyboard/hidden/clear/destroy/reopen and product autocomplete parity');
const clarify = fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/demand-clarify.js'), 'utf8');
const pmBuilder = clarify.slice(clarify.indexOf('  function buildPmOptionsForProduct'), clarify.indexOf('  function bindUserPicker'));
const pmItems = vm.runInNewContext(pmBuilder + '\nbuildPmOptionsForProduct("805")', {
  currentFormData: { userOptions: [{ value: '003030', label: '程统', dept: '组织变革团队' }],
    productMembers: { '805': [{ account: '003030', realname: '程统(003030)', role: '产品负责人 (PO)' }] } }
});
win.initUserPicker('picker', 'account', pmItems, { value: '003030' }); input.focus();
assert.equal(input.value, '程统(003030)', 'PM source names may already include account');
assert.equal(options()[0].querySelector('.ui-autocomplete-main').textContent, '程统(003030)');
assert.equal(options()[0].querySelector('.ui-autocomplete-dept').textContent, ' 组织变革团队');
assert.equal(options()[0].querySelector('.ui-autocomplete-meta').textContent, '产品负责人 (PO)');
console.log('PASS: clarify PM recommendations preserve metadata without duplicate account');
