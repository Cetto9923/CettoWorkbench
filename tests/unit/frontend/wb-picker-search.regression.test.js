#!/usr/bin/env node
// shared/picker 拼音 + 搜索回归（Plan §21 + §11）
// 不依赖 DOM、不依赖 jQuery；仅校验 Core 搜索 / 拼音 vendor 行为。
const path = require('path');
const fs = require('fs');
const vm = require('vm');
const root = path.resolve(__dirname, '../../..');

function assert(cond, msg) {
  if (!cond) { console.error('FAIL:', msg); process.exit(1); }
}
function eq(a, b, msg) { assert(a === b, msg + ' (expected ' + JSON.stringify(b) + ', got ' + JSON.stringify(a) + ')'); }

// 1) vendor 文件存在
const vendorPath = path.join(root, 'web/static/js/picker/vendor/pinyin-lite.js');
assert(fs.existsSync(vendorPath), '拼音 vendor 必须存在: ' + vendorPath);

// 2) vm 加载 vendor（IIFE 通过 global 注入，把 ctx 自身作为 global）
const ctx = {};
ctx.window = ctx;
ctx.global = ctx;
vm.createContext(ctx);
vm.runInContext(fs.readFileSync(vendorPath, 'utf8'), ctx);
const P = ctx.window.PinyinLite || ctx.PinyinLite;
assert(P, 'PinyinLite 必须挂在 window 或 global');
assert(P && typeof P.fullPinyin === 'function', 'PinyinLite.fullPinyin 必须可用');
assert(P && typeof P.initials === 'function', 'PinyinLite.initials 必须可用');

// 3) 真名命中（字面字符）
const common = [
  { name: '张伟', full: 'zhang wei', initials: 'zw' },
  { name: '王芳', full: 'wang fang', initials: 'wf' },
  { name: '李娜', full: 'li na', initials: 'ln' },
  { name: '刘强', full: 'liu qiang', initials: 'lq' },
  { name: '陈静', full: 'chen jing', initials: 'cj' },
  { name: '杨敏', full: 'yang min', initials: 'ym' },
  { name: '黄磊', full: 'huang lei', initials: 'hl' },
  { name: '徐磊', full: 'xu lei', initials: 'xl' }
];
common.forEach(function (c) {
  eq(P.fullPinyin(c.name).replace(/\s+/g, ' ').trim(), c.full,
     c.name + ' 全拼应匹配 ' + c.full);
  eq(P.initials(c.name), c.initials, c.name + ' 首字母应匹配 ' + c.initials);
});

// 4) 英文透传
eq(P.fullPinyin('zhangsan'), 'zhangsan', '英文 account 透传');
eq(P.fullPinyin('ZHANGSAN'), 'zhangsan', '英文大小写无关');
eq(P.initials('ZHANGSAN'), 'z', '英文首字母');

// 5) Core tokenize + token AND（与 Core 行为一致）
// searchText 必须同时含：原字段、英文 account、pinyin、原汉字（用于汉字 token 命中）。
vm.runInContext(fs.readFileSync(path.join(root, 'web/static/js/picker/wb-picker-search.js'), 'utf8'), ctx);
const search = ctx.WbPickerSearch;
const buildSearchText = search.buildSearchText;
function matches(searchText, query) { return search.matches({ searchText }, query); }

var sample = [
  { name: '张伟', fields: ['张伟', 'zhangsan01', '研发一部'], st: '' },
  { name: '张杰', fields: ['张杰', 'zhangjie02', '研发二部'], st: '' },
  { name: '王芳', fields: ['王芳', 'wangfang03', '产品部'], st: '' },
  { name: '陈静', fields: ['陈静', 'chenjing04', '研发一部'], st: '' }
];
sample.forEach(function (s) { s.st = buildSearchText(s.fields); });

assert(matches(sample[0].st, '张伟'), '中文姓名应命中');
assert(matches(sample[1].st, '张杰'), '中文姓名应命中');
assert(matches(sample[0].st, 'zhang'), '拼音全拼应命中');
assert(matches(sample[0].st, 'zhang wei'), '拼音全拼(双字)应命中');
assert(matches(sample[0].st, 'zw'), '拼音首字母应命中');
assert(matches(sample[1].st, 'zj'), '拼音首字母应命中');
assert(matches(sample[0].st, 'zhangsan01'), 'account 应命中');
assert(matches(sample[0].st, 'zhangsan01 张伟'), 'account + 中文多 token AND');
assert(matches(sample[0].st, '研发 张'), '中文 + 中文 多 token AND');
assert(!matches(sample[0].st, '张 王'), '无关 token 必须不命中');

// 5.1) 服务端权威拼音 (zt_user.pinyin) 注入 meta 后命中生僻字
// 禅道原生列含真实姓名全拼 + 空格分隔的首字母组合（如 "maoyuzhang(001196) myz0"）。
// 即使前端 pinyin-lite 不含 "煜/璋"，叠加服务端拼音后也能命中。
var serverPinyinOptions = [
  { account: '001196', realname: '毛煜璋(001196)', pinyin: 'maoyuzhang(001196) myz0', deptName: '小微专项团队' },
  { account: '001277', realname: '张伟(001277)', pinyin: 'zhangwei(001277) zw0', deptName: '研发一部' },
  { account: '004069', realname: '张伟(004069)', pinyin: 'zhangwei(004069) zw4', deptName: '研发二部' }
];
serverPinyinOptions.forEach(function (o) {
  o.st = buildSearchText([o.realname, o.account, o.pinyin, o.deptName]);
});
assert(matches(serverPinyinOptions[0].st, 'maoyuzhang'), '服务端拼音全拼应命中（含生僻字）');
assert(matches(serverPinyinOptions[0].st, 'myz'), '服务端拼音首字母应命中（含生僻字）');
assert(matches(serverPinyinOptions[0].st, '毛煜'), '中文姓名仍应命中（不因注入拼音退化）');
assert(matches(serverPinyinOptions[0].st, '001196'), 'account 命中（不因注入拼音退化）');
assert(matches(serverPinyinOptions[0].st, '小微'), '部门命中（不因注入拼音退化）');
// 多 token AND 仍工作
assert(matches(serverPinyinOptions[0].st, 'maoyuzhang 毛煜'), '拼音全拼 + 中文 多 token AND');
assert(!matches(serverPinyinOptions[0].st, 'maoyuzhang 张伟'), '无关组合必须不命中');

// Initial account / ambiguous-name resolution is exercised through the real adapter
// in wb-picker-core.regression.test.js, not a copied resolver here.
console.log('shared/picker production search + pinyin regression passed');
