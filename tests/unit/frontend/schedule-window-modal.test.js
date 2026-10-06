#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const assert = require('assert');

const root = path.resolve(__dirname, '../../..');
const read = (p) => fs.readFileSync(path.join(root, p), 'utf8');

const scheduleHtml = read('web/templates/schedule/index.html');
const componentHtml = read('web/templates/components/schedule_window_modal.html');
const homeHtml = read('web/templates/po/home.html');
const windowJs = read('web/static/js/schedule/schedulewindow.js');

for (const [name, html] of [['schedule/index.html', scheduleHtml], ['schedule_window_modal.html', componentHtml]]) {
  assert(
    /<label[^>]*for="scheduleWindowEnd"[^>]*>窗口结束 <span[^>]*>\*<\/span><\/label>/.test(html),
    name + ': 窗口结束必须带必填标记'
  );
  assert(/<input[^>]*id="scheduleWindowEnd"[^>]*required/.test(html), name + ': #scheduleWindowEnd 必须 required');
}

assert(/schedule\/window_modal/.test(homeHtml), 'home.html 必须引用 schedule/window_modal 组件');
for (const [name, html] of [['schedule/index.html', scheduleHtml], ['home.html', homeHtml]]) {
  assert(/schedulewindow\.js/.test(html), name + ': 必须加载 schedulewindow.js，保存前统一校验');
}

const validate = windowJs.match(/function validateScheduleCreateSavePayload\(payload\) \{[\s\S]*?\n  \}/);
assert(validate, 'schedulewindow.js: 缺少 validateScheduleCreateSavePayload');
const fn = new Function('payload', validate[0].replace(/^function [^{]+\{/, '').replace(/\}$/, ''));
const ok = {
  releaseDate: '2026-12-31', name: 'w', startDate: '2026-12-01', endDate: '2026-12-31',
  planTestDone: '2026-12-10', testDone: '2026-12-20', acceptDone: '2026-12-25', teamgroupId: 1,
};
assert.strictEqual(fn(ok), '');
assert.strictEqual(fn(Object.assign({}, ok, { endDate: '' })), '请填写窗口结束日期');

console.log('schedule-window-modal: ok');
