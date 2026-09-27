#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const vm = require('vm');
const root = path.resolve(__dirname, '../../..');
const source = fs.readFileSync(path.join(root, 'web/static/js/po/home-team.js'), 'utf8');

function element(id, attrs = {}) {
  const listeners = {};
  const classes = new Set();
  return {
    id,
    hidden: false,
    value: '',
    innerHTML: '',
    textContent: '',
    dataset: {},
    handlers: listeners,
    style: {},
    getAttribute(name) { return attrs[name] || ''; },
    setAttribute(name, value) { attrs[name] = String(value); },
    addEventListener(name, fn) { listeners[name] = fn; },
    classList: {
      toggle(name, force) {
        const shouldAdd = force === undefined ? !classes.has(name) : !!force;
        if (shouldAdd) classes.add(name); else classes.delete(name);
        return shouldAdd;
      },
      contains(name) { return classes.has(name); },
    },
  };
}

function createHarness(search, replies) {
  const rows = element('teamHomeGroupRows');
  const error = element('teamHomeError');
  const errorText = element('teamHomeErrorText');
  const updated = element('teamHomeUpdatedAt');
  const select = element('teamHomeGroupSelect');
  const retry = element('teamHomeRetry');
  const buttons = [element('team', { 'data-scope': 'team' }), element('dept', { 'data-scope': 'dept' })];
  const byID = {
    teamHomeGroupRows: rows,
    teamHomeError: error,
    teamHomeErrorText: errorText,
    teamHomeUpdatedAt: updated,
    teamHomeGroupSelect: select,
    teamHomeRetry: retry,
  };
  const requests = [];
  let replyIndex = 0;
  const loc = { href: 'http://workbench.local/home' + search, pathname: '/home', search, hash: '' };
  const history = {
    replaceState(_state, _title, next) {
      const url = new URL(next, loc.href);
      loc.href = url.href;
      loc.pathname = url.pathname;
      loc.search = url.search;
      loc.hash = url.hash;
    },
  };
  const document = {
    getElementById(id) { return byID[id] || null; },
    querySelectorAll(selector) {
      return selector === '#teamHomeScopeSwitch [data-scope]' ? buttons : [];
    },
  };
  const window = { location: loc, history };
  const fetch = async url => {
    requests.push(String(url));
    const payload = replies[Math.min(replyIndex++, replies.length - 1)];
    return { ok: true, status: 200, json: async () => payload };
  };
  vm.runInNewContext(source, { window, document, location: loc, history, fetch, URL, URLSearchParams, encodeURIComponent, Number, String, Array });
  return { loc, rows, error, errorText, updated, select, buttons, requests };
}

function response(options, items, activeScope = 'team') {
  return { success: true, data: { activeScope, availableScopes: [activeScope], scopeOptions: options, items, allCount: items.length } };
}

function tick() { return new Promise(resolve => setTimeout(resolve, 0)); }

(async function main() {
  const selected = createHarness('?view=team&scope=team&teamgroupId=11', [response(
    [{ id: 1, name: '产品团队', type: 'team' }, { id: 11, name: '对公一组', type: 'subteam' }],
    [{ id: 11, name: '对公一组', type: 'child', formalCount: 6, pendingAdd: 1, pendingRemove: 0 }]
  )]);
  await tick();
  const firstRequest = new URL(selected.requests[0], 'http://workbench.local');
  if (firstRequest.searchParams.get('scopeId') !== '11' || firstRequest.searchParams.get('scope') !== 'team') {
    throw new Error('selected child scope was not sent to the authorized team-list API');
  }
  if (!selected.rows.innerHTML.includes('对公一组') || !selected.rows.innerHTML.includes('teamgroupId=11')) {
    throw new Error('authorized child row or its scoped agile-team link was not rendered');
  }
  if (!selected.rows.innerHTML.includes('/issues/risk?kind=issue&amp;loop=open&amp;scope=team&amp;scopeId=11')) {
    throw new Error('child row did not preserve its team scope in the issue-risk deep link');
  }
  if (selected.select.value !== '11' || new URL(selected.loc.href).searchParams.get('teamgroupId') !== '11') {
    throw new Error('the selected group was not restored into the control and URL');
  }

  const invalid = createHarness('?view=team&scope=team&teamgroupId=999', [
    response([{ id: 11, name: '对公一组', type: 'subteam' }], []),
    response([{ id: 11, name: '对公一组', type: 'subteam' }], [{ id: 11, name: '对公一组', type: 'child' }]),
  ]);
  await tick();
  await tick();
  if (invalid.requests.length !== 2 || new URL(invalid.requests[1], 'http://workbench.local').searchParams.has('scopeId')) {
    throw new Error('unauthorized group ID was not cleared and reloaded as the full authorized scope');
  }
  if (new URL(invalid.loc.href).searchParams.has('teamgroupId')) {
    throw new Error('unauthorized group ID remained in the shareable URL');
  }

  const department = createHarness('?view=team&scope=dept&teamgroupId=1', [response(
    [{ id: 1, name: '组织变革团队', type: 'team' }, { id: 11, name: '项目赋能组', type: 'subteam' }],
    [{ id: 1, name: '组织变革团队', type: 'parent', formalCount: 8 }],
    'dept'
  )]);
  await tick();
  if (!department.rows.innerHTML.includes('/agileteam?view=lead&amp;scope=dept&amp;teamgroupId=1') ||
      !department.rows.innerHTML.includes('/issues/risk?kind=issue&amp;loop=open&amp;scope=dept&amp;scopeId=1')) {
    throw new Error('department row did not preserve the department authorization scope on both links');
  }

  console.log('home team scope runtime regression passed');
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
