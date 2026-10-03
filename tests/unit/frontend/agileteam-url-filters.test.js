#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const js = fs.readFileSync(path.join(root, 'web/static/js/agileteam/agileteam.js'), 'utf8');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

assert(/new URLSearchParams\(location\.search\s*\|\|/.test(js), 'agile team view reads filters from query parameters');
assert(/get\("teamgroupId"\)/.test(js), 'deep links can select one agile team or subteam');
assert(/searchParams\.set\("teamgroupId"/.test(js), 'selected agile group is written back to the URL');
assert(/searchParams\.set\("scope"/.test(js) && /searchParams\.set\("teamgroupId"/.test(js), 'scope kind and selected group remain shareable');
assert(/history\.replaceState/.test(js), 'filter changes update the current URL without a navigation');

console.log('agileteam URL filters regression passed');

async function verifyLatestListWins(oldFails) {
  const pending = [];
  const removed = [];
  let controls = [{ id: 'oldOrgPicker' }];
  const body = {
    querySelectorAll: () => controls,
    set innerHTML(value) { this.html = value; controls = []; },
    get innerHTML() { return this.html; }
  };
  const window = {
    location: { href: 'http://workbench.invalid/agileteam', search: '' },
    escapeHtml: String,
    destroyAutocomplete: id => removed.push(id),
    appJson: () => new Promise((resolve, reject) => pending.push({ resolve, reject }))
  };
  const document = {
    readyState: 'loading',
    addEventListener() {},
    querySelector: () => null,
    getElementById: id => id === 'atListBody' ? body : null
  };
  require('vm').runInNewContext(js, { window, document, location: window.location, URL, URLSearchParams });
  const old = window.__at.loadList();
  const latest = window.__at.loadList();
  pending[1].resolve({ data: { total: 7, items: [] } });
  await latest;
  if (oldFails) pending[0].reject(new Error('过期请求失败'));
  else pending[0].resolve({ data: { total: 99, items: [] } });
  await old;
  assert(window.__at.state.total === 7, 'stale list success must not replace current totals');
  assert(body.html.includes('暂无敏捷小组'), 'stale list failure must not replace current rows');
  assert(removed.join() === 'oldOrgPicker', 'old autocomplete is destroyed before its input is removed');
}

Promise.all([verifyLatestListWins(false), verifyLatestListWins(true)]).then(() => {
  console.log('agileteam latest response and autocomplete cleanup regression passed');
}).catch(error => { console.error(error); process.exitCode = 1; });
