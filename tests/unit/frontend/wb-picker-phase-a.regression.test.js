#!/usr/bin/env node
// Phase A 静态契约：PO / AgileTeam 等视图使用统一 shared picker 架构规范。
const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');

function read(p) { return fs.readFileSync(path.join(root, p), 'utf8'); }
function assert(cond, msg) { if (!cond) { console.error('FAIL:', msg); process.exit(1); } }

// 1. 2026-08-27 人员控件统一规范：姓名（工号）+ 部门 sub，所有支持搜索的 picker 共用 adapter
const personPickerJs = read('web/static/js/picker/adapters/wb-person-picker.js');
assert(/subLabel:\s*u\.deptName\s*\|\|\s*''/.test(personPickerJs), 'person picker must show dept only in sub row');
assert(/avatar:\s*true/.test(personPickerJs), 'person picker must enable avatar');
assert(/avatarName:\s*u\.realname\s*\|\|\s*u\.account/.test(personPickerJs), 'person picker avatar must derive from realname');
assert(!/\.catch\s*\(\s*function\s*\(\s*\)\s*\{\s*return\s*\[\s*\]\s*;\s*\}\s*\)/.test(personPickerJs),
  'directory failure must not be disguised as empty users');

const pickerJs = read('web/static/js/picker/wb-picker.js');
assert(/wb-picker-avatar/.test(pickerJs), 'wb-picker core must render avatar element');

const pickerCss = read('web/static/css/picker/wb-picker.css');
assert(/\.wb-picker-avatar\s*\{/.test(pickerCss), 'wb-picker css must define avatar style');

// 2. 消费方契约：agileteam/index.html 必须规范引入 shared picker 依赖
const agileteamHtml = read('web/templates/agileteam/index.html');
[
  'picker/vendor/pinyin-lite.js',
  'picker/wb-picker-search.js',
  'picker/wb-directory.js',
  'picker/wb-picker.js',
  'picker/adapters/wb-person-picker.js'
].forEach((name) => assert(agileteamHtml.includes(name), 'agileteam/index.html must load ' + name));

console.log('shared/picker Phase A static regression passed');
