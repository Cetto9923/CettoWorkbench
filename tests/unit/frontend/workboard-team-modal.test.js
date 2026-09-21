#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '../../..');
function read(file) { return fs.readFileSync(path.join(root, file), 'utf8'); }
function assert(condition, message) {
  if (!condition) {
    console.error('FAIL ' + message);
    process.exit(1);
  }
}

const modal = read('web/static/js/po/workboard-modal.js');
const core = read('web/static/js/po/workboard-core.js');
const board = read('web/static/js/po/workboard.js');
const css = read('web/static/css/po/workboard-addon.css');

assert(/getSelectedTeamgroupId/.test(core), 'workboard exposes the selected teamgroup id');
assert(/selectedTeamgroupId/.test(board), 'demand board restores the server-selected default teamgroup');
assert(/renderDemandOwners\(payload\.tree \|\| \[\]\);[\s\S]*loadMetrics\(\);/.test(board),
  'demand board loads the team snapshot after resolving the default teamgroup');
assert(/\/workbench\/api\/agile-teams\/" \+ encodeURIComponent\(teamId\)/.test(modal),
  'team modal loads the selected team detail');
assert(/data\.formal \|\| \[\]/.test(modal), 'team modal renders formal members from the detail response');
assert(/\/workbench\/api\/agile-teams\/candidates\?q=/.test(modal) && /teamgroupId/.test(modal),
  'team modal searches real candidate users');
assert(/searchCandidates\("", teamId\)/.test(modal),
  'team modal loads same-level team members when search is empty');
assert(/候选人已经由服务端/.test(modal) && /return true;/.test(modal),
  'team modal preserves server-side pinyin matches');
assert(/compositionstart/.test(modal) && /compositionend/.test(modal),
  'team modal does not rerender during IME composition');
assert(!/可用工时\/天/.test(modal), 'team modal removes available-hours column');
assert(/敏捷小组角色/.test(modal) && /agileTeamRoles/.test(modal),
  'team modal labels and stores agile-team roles');
assert(/请输入姓名或账号搜索候选人/.test(modal), 'empty candidate state explains how to search');
assert(!/html\[data-theme="dark"\]/.test(css), 'modal css does not fork a hard-coded dark selector set');
assert(/kb-team-copy[^{]*\{[^}]*var\(--color-surface\)/s.test(css),
  'copy-team button uses semantic theme tokens');

console.log('workboard team modal regression passed');

const template = read('web/templates/po/workboard.html');
assert(!/id="kbTeamReason"/.test(modal), 'body must not render a discarded adjustment reason');
assert((template.match(/id="kbTeamAdjustReason"/g) || []).length === 1,
  'team modal has one submitted adjustment reason');
assert(/getElementById\("kbTeamAdjustReason"\)/.test(modal), 'submission reads the retained reason');

async function testSubmitTeamModal() {
  const vm = require('vm');
  const strict = require('assert/strict');
  const calls = [];
  const notices = [];
  let candidateClick;
  function element() {
    return { value: '', innerHTML: '', classList: { add() {}, remove() {} }, addEventListener() {} };
  }
  const elements = Object.fromEntries(['kanbanTeamModalDialog', 'kanbanModalOverlay',
    'kanbanTeamModalTitle', 'kbTeamModalBody', 'kbTeamAdjustReason'].map(id => [id, element()]));
  elements.kbTeamModalBody.querySelectorAll = selector => {
    if (selector !== '.kb-team-candidate' || !elements.kbTeamModalBody.innerHTML.includes('data-account="dev1"')) return [];
    return [{ getAttribute: () => 'dev1', addEventListener: (event, fn) => { candidateClick = fn; } }];
  };
  const window = {
    escapeHtml: value => String(value),
    showToast: (...args) => notices.push(args),
    PoWB: { getSelectedTeamgroupId: () => 7 },
    appJson: async (url, options) => {
      calls.push({ url, options });
      if (options.method === 'POST') return { success: true };
      if (url.includes('/candidates?')) return { success: true, data: [{ account: 'dev1', name: '开发员' }] };
      return { success: true, data: { name: '测试小组', formal: [] } };
    }
  };
  vm.runInNewContext(modal, {
    window,
    document: { getElementById: id => elements[id] || null, querySelector: () => null, addEventListener() {}, readyState: 'complete' },
    setTimeout
  });
  window.WorkboardModal.openTeamModal();
  await new Promise(resolve => setImmediate(resolve));
  strict.equal(typeof candidateClick, 'function');
  elements.kbTeamAdjustReason.value = '  补充成员  ';
  candidateClick();
  strict.equal(elements.kbTeamAdjustReason.value, '  补充成员  ', 'rerender preserves the submitted reason');
  window.WorkboardModal.submitTeamModal();
  await new Promise(resolve => setImmediate(resolve));
  const posts = calls.filter(call => call.options.method === 'POST');
  strict.equal(posts.length, 1);
  strict.equal(posts[0].url, '/workbench/api/agile-teams/7/adjustments');
  strict.deepEqual(JSON.parse(JSON.stringify(posts[0].options.body)), {
    reason: '补充成员', items: [{ account: 'dev1', actionType: 'add', role: '研发', availableHours: 0 }]
  });
  strict.ok(notices.some(([message]) => message === '已提交，待组织级敏捷教练确认'));
  console.log('workboard team submit behavior passed');
}
testSubmitTeamModal().catch(error => { console.error(error); process.exitCode = 1; });
