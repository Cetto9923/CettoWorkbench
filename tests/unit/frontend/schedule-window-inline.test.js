#!/usr/bin/env node
'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.resolve(__dirname, '../../..');
const read = (p) => fs.readFileSync(path.join(root, p), 'utf8');

const scheduleJs = read('web/static/js/schedule/schedule.js');
const scheduleWindowJs = read('web/static/js/schedule/schedulewindow.js');
const scheduleInlineJs = read('web/static/js/schedule/schedule-inline.js');
const scheduleRowsHtml = read('web/templates/components/schedule_integrated_rows.html');

// 1. 静态断言：schedule.js 不得重复绑定 scheduleVersionWindowModal* 按钮
assert(
  !scheduleJs.includes('$("#scheduleVersionWindowModalSaveBtn").on('),
  'schedule.js: 不得重复绑定 scheduleVersionWindowModalSaveBtn 点击事件'
);
assert(
  !scheduleJs.includes('$("#scheduleVersionWindowModalOverlay").on('),
  'schedule.js: 不得重复绑定 scheduleVersionWindowModalOverlay 点击事件'
);

// 2. 静态断言：schedulewindow.js 必须统一绑定保存与关闭
assert(
  scheduleWindowJs.includes('$("#scheduleVersionWindowModalSaveBtn").on("click", saveScheduleVersionWindowModal)'),
  'schedulewindow.js: 必须绑定 scheduleVersionWindowModalSaveBtn 保存事件'
);
assert(
  scheduleWindowJs.includes('$("#scheduleVersionWindowModalOverlay, #scheduleVersionWindowModalCloseBtn, #scheduleVersionWindowModalDismissBtn")'),
  'schedulewindow.js: 必须统一绑定遮罩与关闭按钮'
);

// 3. 静态断言：schedule-inline.js 必须移除 currentCanEditWindow 拦截并保留 isSchedulingDetailLoaded
assert(
  !scheduleInlineJs.includes('currentCanEditWindow'),
  'schedule-inline.js: 必须移除已废弃的 currentCanEditWindow 检查'
);
assert(
  /if\s*\(!shared\.isSchedulingDetailLoaded\)/.test(scheduleInlineJs),
  'schedule-inline.js: 必须保留 isSchedulingDetailLoaded 判断'
);

// 4. 静态断言：schedule_integrated_rows.html 模板中必须包含 story-item-dept
assert(
  /<td class="story-item-dept"><\/td>/.test(scheduleRowsHtml),
  'schedule_integrated_rows.html: tplStoryItemRow 必须包含 class="story-item-dept"'
);

// 5. 运行时断言：保存回调与独立页面跳转逻辑
async function testRuntimeBehavior() {
  let draft;
  let success = true;
  let savedPayload = null;
  let savedResData = null;
  const requests = [];

  const boundEvents = {};
  const $ = (selector) => ({
    text() { return this; },
    on(event, handler) {
      boundEvents[selector + ':' + event] = handler;
      return this;
    }
  });

  const window = {
    location: { href: '/home' },
    showToast() {},
    openShowModals() {},
    closeShowModals() {},
    ScheduleWindowDraft: {
      createDraft: () => ({
        online: '2026-12-31',
        name: '测试窗口',
        start: '2026-12-01',
        end: '2026-12-31',
        planTestDone: '2026-12-20',
        testDone: '2026-12-25',
        acceptDone: '2026-12-28',
        teamgroupId: '146',
        productIds: []
      }),
      setDraft: (value) => { draft = value; },
      getDraft: () => draft,
      resetDraft() {},
      fillForm() {}
    },
    appFetch: async (url, options) => {
      requests.push({ url, options });
      return {
        ok: true,
        json: async () => ({
          success,
          message: '版本窗口保存成功',
          redirectUrl: '/schedule'
        })
      };
    }
  };

  vm.runInNewContext(scheduleWindowJs, {
    window,
    jQuery: $,
    document: { getElementById: () => null }
  });

  // 验证绑定的保存函数存在
  assert(
    typeof boundEvents['#scheduleVersionWindowModalSaveBtn:click'] === 'function',
    '保存按钮点击事件必须成功挂载'
  );

  const flush = () => new Promise((resolve) => setImmediate(resolve));

  // A. 首页带回调调用：保存成功后调用回调，留在当前页（不跳转）
  window.openScheduleCreateVersionWindowModal((payload, resData) => {
    savedPayload = payload;
    savedResData = resData;
  });
  window.saveScheduleVersionWindowModal();
  await flush();

  assert.notEqual(savedPayload, null, '保存成功后必须调用 onSaved 回调');
  assert.equal(savedPayload.teamgroupId, '146');
  assert.equal(savedResData && savedResData.success, true);
  assert.equal(window.location.href, '/home', '首页带回调保存成功后不得跳转');
  assert.equal(requests[0].url, '/schedule/windows');
  assert.equal(requests[0].options.method, 'POST');

  // B. 关闭弹窗后清空回调
  window.closeScheduleVersionWindowModal();
  savedPayload = null;

  // C. 排期页不带回调调用：保存成功后应按 redirectUrl 跳转
  window.openScheduleCreateVersionWindowModal();
  window.saveScheduleVersionWindowModal();
  await flush();

  assert.equal(savedPayload, null, '未传回调时不应调用先前已清除的回调');
  assert.equal(window.location.href, '/schedule', '排期页保存成功后必须保留原有跳转行为');

  console.log('PASS: schedule-window-inline.test.js verified successfully');
}

testRuntimeBehavior().catch((err) => {
  console.error(err);
  process.exit(1);
});
