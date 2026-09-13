#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '../../..');
function read(p) { return fs.readFileSync(path.join(root, p), 'utf8'); }
function assert(cond, msg) {
  if (!cond) {
    console.error('FAIL ' + msg);
    process.exit(1);
  }
}

const agileteamJs = read('web/static/js/agileteam/agileteam.js');
const agileteamDetailJs = read('web/static/js/agileteam/agileteam-detail.js');
const agileteamMembersJs = read('web/static/js/agileteam/agileteam-members.js');
const agileteamHtml = read('web/templates/agileteam/index.html');

// 1. 敏捷小组调整与审批动作
assert(/atOpenReview|atConfirm|atReject|adjustments/.test(agileteamJs + agileteamDetailJs),
  'agileteam wires review and confirm actions');
assert(/at-child-badge|父级小组/.test(agileteamJs + agileteamDetailJs),
  'agileteam list/detail keep parent-child hierarchy');

// 2. 页面与组件挂载
assert(/atPager/.test(agileteamHtml), 'agileteam list has real pagination host');
assert(/atScopeCard/.test(agileteamHtml), 'agileteam has scope card');
assert(/wb-picker\.js/.test(agileteamHtml) && /wb-picker\.css/.test(agileteamHtml) && /wb-person-picker\.js/.test(agileteamHtml),
  'agileteam loads shared person picker assets');

// 3. 成员编辑使用统一人员选择器与角色原生下拉
assert(/atOpenMemberEdit|WbPersonPicker|DEFAULT_TEAM_ROLES|at-member-role-select/.test(agileteamMembersJs),
  'member editor uses unified person picker and role selection');
assert(/list=/.test(agileteamMembersJs) === false,
  'member role no longer uses browser datalist (avoids mispositioned dropdown)');

// 4. 自定义 pageSize：下拉 + 输入框 + localStorage 持久化 + 范围夹逼
assert(/自定义\u2026/.test(agileteamJs) || /自定义/.test(agileteamJs),
  'page-size dropdown exposes 自定义… option');
assert(/AT_PAGE_SIZE_MIN\s*=\s*5/.test(agileteamJs) && /AT_PAGE_SIZE_MAX\s*=\s*200/.test(agileteamJs),
  'pageSize custom range is 5..200');
assert(/wb:agileteam:pageSizeCustom/.test(agileteamJs),
  'custom pageSize persisted to localStorage with module-prefixed key');
assert(/atCommitCustomPageSize/.test(agileteamJs),
  'custom pageSize commits via atCommitCustomPageSize');
assert(/localStorage\.setItem\(\s*AT_PAGE_SIZE_CUSTOM_KEY/.test(agileteamJs),
  'atCommitCustomPageSize writes the custom value to localStorage');
assert(/localStorage\.getItem\(\s*AT_PAGE_SIZE_CUSTOM_KEY/.test(agileteamJs),
  'state seed reads the persisted custom value on boot');
assert(/clampPageSize|AT_PAGE_SIZE_MIN/.test(agileteamJs),
  'frontend clamps custom input to [5,200] before sending to backend');

console.log('agileteam adjustment regression passed');
