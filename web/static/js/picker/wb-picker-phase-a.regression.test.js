#!/usr/bin/env node
// Phase A 静态契约：PO 页加载 shared picker，不再引用已删除临时选人脚本。
const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../../../..');
function read(p) { return fs.readFileSync(path.join(root, p), 'utf8'); }
function assert(cond, msg) { if (!cond) { console.error('FAIL:', msg); process.exit(1); } }

const po = read('web/templates/workbench/po.html');
const schedule = ['po-schedule.js', 'schedule/schedule-window-form.js', 'schedule/schedule-row-modal.js', 'schedule/schedule-integrated.js', 'schedule/schedule-task-actions.js', 'schedule/schedule-runtime-truth.js']
  .map((file) => read('web/static/workbench/po/' + file)).join('\n');
// R5-E 拆分: kanban 域跨 6 个文件, 断言基于域合并源 (kbCreateAssignee/formatKanbanAssigneeLabel 在 kanban-create)。
const kanban = ['po-kanban.js', 'kanban/kanban-agile.js', 'kanban/kanban-format.js', 'kanban/kanban-issue.js', 'kanban/kanban-team.js', 'kanban/kanban-create.js']
  .map((file) => read('web/static/workbench/po/' + file)).join('\n');

assert(!po.includes('po-person-search-select.js'), 'po.html must not reference deleted po-person-search-select.js');
assert(!po.includes('po-person-search-select.css'), 'po.html must not reference deleted po-person-search-select.css');
[
  'vendor/pinyin-lite.js',
  'wb-picker-search.js',
  'wb-directory.js',
  'wb-picker.js',
  'adapters/wb-person-picker.js'
].forEach((name) => assert(po.includes(name), 'po.html must load ' + name));

assert((schedule.match(/data-wb-person-picker/g) || []).length >= 5, 'schedule should mark migrated person selects');
assert(kanban.includes('id="kbCreateAssignee" data-account-source="zentao"'), 'kanban create assignee must keep account-valued select');
assert(!/id="kbCreateAssignee"[^>]*data-wb-person-picker/.test(kanban), 'kanban create assignee must stay native select (current agile members only)');
assert(/function\s+formatKanbanAssigneeLabel/.test(kanban), 'kanban assignee label helper must exist');
assert(/function\s+resolveAccount[\s\S]*String\(u\.account \|\| ''\) === raw[\s\S]*byName\.length === 1/.test(schedule), 'resolveAccount must account-pass and unique-realname only');
assert(/assignedTo:\s*\(assigned && assigned\.account\)/.test(schedule), 'task draft save must submit account');

// 2026-08-27 人员控件统一：姓名（工号）+ 部门 sub，所有支持搜索的 picker 必须共用 adapter。
const personPickerJs = read('web/static/workbench/shared/picker/adapters/wb-person-picker.js');
assert(/subLabel:\s*u\.deptName \|\| ''/.test(personPickerJs), 'person picker must show dept only in sub row');
assert(/avatar:\s*true/.test(personPickerJs), 'person picker must enable avatar');
assert(/avatarName:\s*u\.realname \|\| u\.account/.test(personPickerJs), 'person picker avatar must derive from realname');
const pickerJs = read('web/static/workbench/shared/picker/wb-picker.js');
assert(/wb-picker-avatar/.test(pickerJs), 'wb-picker core must render avatar element');
const pickerCss = read('web/static/workbench/shared/picker/wb-picker.css');
assert(/\.wb-picker-avatar\s*\{/.test(pickerCss), 'wb-picker css must define avatar style');

console.log('shared/picker Phase A static regression passed');
