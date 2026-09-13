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
const css = read('web/static/css/po/workboard-addon.css');

assert(/getSelectedTeamgroupId/.test(core), 'workboard exposes the selected teamgroup id');
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
