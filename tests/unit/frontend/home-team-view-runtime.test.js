#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');
const root = path.resolve(__dirname, '../../..');
const source = fs.readFileSync(path.join(root, 'web/static/js/po/home-team.js'), 'utf8');

function element(id) {
  const listeners = {};
  const classes = new Set();
  return {
    id, hidden: false, disabled: false, value: '', innerHTML: '', textContent: '', href: '', listeners,
    setAttribute() {}, addEventListener(name, fn) { listeners[name] = fn; },
    classList: {
      toggle(name, force) { if (force) classes.add(name); else classes.delete(name); },
      contains(name) { return classes.has(name); },
    },
  };
}

function harness(search, scopes) {
  const ids = [
    'teamHomeDeptSelect', 'teamHomeTeamSelect', 'teamHomeSubteamSelect', 'teamHomeRetry', 'teamHomeError',
    'teamHomeErrorText', 'teamHomeIssueCount', 'teamHomeRiskCount', 'teamHomeRiskStatus', 'teamHomeRiskLink',
    'teamHomeVersionRows', 'teamHomeVersionStatus', 'teamHomeValueStreamRows', 'teamHomeValueStreamStatus',
    'teamHomeDueRows', 'teamHomeDueStatus', 'teamHomeBlockedCount', 'teamHomeOverdueCount', 'teamHomeSoonCount',
    'teamHomeBlockedLink', 'teamHomeUpdatedAt',
  ];
  const nodes = new Map(ids.map(id => [id, element(id)]));
  const requests = [];
  const location = { href: 'http://workbench.local/home' + search, pathname: '/home', search, hash: '' };
  const history = {
    replaceState(_state, _title, next) {
      const url = new URL(next, location.href);
      location.href = url.href;
      location.pathname = url.pathname;
      location.search = url.search;
      location.hash = url.hash;
    },
  };
  const document = {
    getElementById(id) { return nodes.get(id) || null; },
    querySelectorAll() { return []; },
  };
  const fetch = async raw => {
    const url = new URL(String(raw), 'http://workbench.local');
    requests.push(url);
    let payload;
    if (url.pathname === '/home/team/scopes') {
      payload = { success: true, data: scopes };
    } else if (url.pathname === '/issues/risk/items') {
      payload = { success: true, total: url.searchParams.get('kind') === 'issue' ? 7 : 3 };
    } else if (url.pathname === '/home/team/version-windows') {
      payload = { success: true, data: [{ id: 77, name: '十月版本', teamgroupId: 11, teamgroup: '产品团队 / 对公一组', releaseDate: '2026-10-10', workItemCount: 12 }] };
    } else if (url.pathname === '/home/team/value-stream') {
      payload = { success: true, data: [{ status: 'accept', label: '受理', count: 10, demandCount: 8, storyCount: 2 }] };
    } else if (url.pathname === '/home/team/dashboard') {
      payload = { success: true, data: { blocked: 4, overdue: 2, soon: 5, throughDate: '2026-10-01', updatedAt: '2026-09-28T10:00:00Z', due: [{ kind: 'demand', id: 3, title: '临期需求', owner: 'alice', deadline: '2026-09-30' }] } };
    } else {
      throw new Error('unexpected URL ' + url.pathname);
    }
    return { ok: true, status: 200, json: async () => payload };
  };
  const window = { location, history, escapeHtml: value => String(value == null ? "" : value).replace(/[&<>"']/g, char => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char])) };
  vm.runInNewContext(source, { window, document, location, history, fetch, URL, URLSearchParams, encodeURIComponent, Number, String, Array });
  return { nodes, requests, location };
}

const scopes = [
  { id: 10, parent: 0, deptId: 0, kind: 'dept', name: '交易二部' },
  { id: 1, parent: 0, deptId: 10, kind: 'team', name: '产品团队' },
  { id: 11, parent: 1, deptId: 10, kind: 'team', name: '对公一组' },
  { id: 2, parent: 0, deptId: 10, kind: 'team', name: '交付团队' },
];
const tick = async () => { await new Promise(resolve => setTimeout(resolve, 0)); await new Promise(resolve => setTimeout(resolve, 0)); };

(async function main() {
  const selected = harness('?view=team&scope=team&scopeId=1&teamgroupId=1&deptId=10&subteamgroupId=11&stage=testing', scopes);
  await tick();
  const dashboard = selected.requests.find(url => url.pathname === '/home/team/dashboard');
  if (!dashboard || dashboard.searchParams.get('scope') !== 'team' || dashboard.searchParams.get('scopeId') !== '1' || dashboard.searchParams.get('subteamgroupId') !== '11') {
    throw new Error('selected subgroup was not sent with its authorized parent scope');
  }
  const riskRequests = selected.requests.filter(url => url.pathname === '/issues/risk/items');
  if (riskRequests.length !== 2 || riskRequests.some(url => url.searchParams.get('teamgroupId') !== '11')) {
    throw new Error('issue and risk counts did not use the selected subgroup');
  }
  if (selected.nodes.get('teamHomeBlockedCount').textContent !== '4' || selected.nodes.get('teamHomeOverdueCount').textContent !== '2' ||
      selected.nodes.get('teamHomeSoonCount').textContent !== '5' || !selected.nodes.get('teamHomeDueRows').innerHTML.includes('临期需求')) {
    throw new Error('four-card dashboard counts or due rows did not render');
  }
  if (selected.nodes.get('teamHomeIssueCount').textContent !== '7' || selected.nodes.get('teamHomeRiskCount').textContent !== '3') {
    throw new Error('scoped issue/risk counts did not render');
  }
  if (!selected.nodes.get('teamHomeVersionRows').innerHTML.includes('/schedule?windows=77&amp;groups=11&amp;stage=testing')) {
    throw new Error('version window deep link lost the current stage');
  }
  if (selected.nodes.get('teamHomeTeamSelect').value !== '1' || selected.nodes.get('teamHomeSubteamSelect').value !== '11') {
    throw new Error('department, organization-team and subgroup selections were not restored');
  }

  const invalid = harness('?view=team&scope=team&scopeId=1&deptId=10&subteamgroupId=99', scopes);
  await tick();
  const invalidDashboard = invalid.requests.find(url => url.pathname === '/home/team/dashboard');
  if (!invalidDashboard || invalidDashboard.searchParams.has('subteamgroupId')) {
    throw new Error('subgroup outside the available team was retained in dashboard scope');
  }

  const department = harness('?view=team&scope=dept&scopeId=10', scopes);
  await tick();
  const departmentDashboard = department.requests.find(url => url.pathname === '/home/team/dashboard');
  if (!departmentDashboard || departmentDashboard.searchParams.get('scope') !== 'dept' || departmentDashboard.searchParams.get('scopeId') !== '10') {
    throw new Error('department selection was not applied to the dashboard');
  }
  console.log('home team four-card scope runtime regression passed');
})().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
