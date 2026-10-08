/* 文件: po/demand-title.test.js 模块: 业务需求 职责: 验证列表标题按对象类型打开详情。 */
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const escapeHtml = (value) => String(value ?? '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');

function render(file, name, item) {
  const context = {
    window: { escapeHtml, PersonalList: { PAGE_SIZE_OPTIONS: [], priorityBadge: escapeHtml, objectTypeBadgeFromKind: escapeHtml, idChipHtml: (_, id) => id } },
    document: { readyState: 'loading', addEventListener() {}, getElementById() { return null; } },
    URLSearchParams, Date,
  };
  const source = fs.readFileSync(path.join(__dirname, file), 'utf8')
    .replace('  "use strict";', '  "use strict"; globalThis.renderTitleTest = ' + name + ';');
  vm.runInNewContext(source, context);
  return context.renderTitleTest(item, 0);
}

test('待办按对象类型路由：业务详情、研发和其他对象原链接', () => {
  for (const kind of ['demand', 'story', 'task', 'bug', 'approval']) {
    const html = render('todos.js', 'rowHtml', { kind, id: 123, title: '<标题>', url: 'http://zentao.test/view-123' });
    const title = html.match(/<div class="todos-item-title">(.*?)<\/div>/)[1];
    assert.match(title, /&lt;标题>/);
    assert.match(title, kind === 'demand' ? /href="\/demands\/123"/ : /href="http:\/\/zentao.test\/view-123" target="_blank"/);
  }
});

test('通知按真实对象类型打开需求详情，保留研发及无对象通知入口', () => {
  const business = render('notice.js', 'formatNoticeSubject', { objectType: 'demand', objectId: 123, subject: '<标题>', url: 'http://zentao.test/demand-view-123' });
  assert.match(business, /notice-title-main" href="\/demands\/123"/);
  const story = render('notice.js', 'formatNoticeSubject', { objectType: 'story', objectId: 123, subject: '研发需求', url: 'http://zentao.test/story-view-123' });
  assert.match(story, /notice-title-main" href="http:\/\/zentao.test\/story-view-123" target="_blank"/);
  const message = render('notice.js', 'formatNoticeSubject', { id: 7, objectType: 'mail', subject: '系统通知' });
  assert.match(message, /data-notice-open="7"/);
});

test('版本跟进标题可打开业务详情，展开按钮仍单独保留', () => {
  const html = render('version-follow.js', 'rowHtml', { demandId: 123, demandNo: 'US123', title: '<标题>', stage: '澄清' });
  assert.match(html, /href="\/demands\/123"/);
  assert.match(html, /data-expand="0"/);
  assert.match(html, /&lt;标题>/);
});

test('父需求聚合中的子需求标题打开相同详情', () => {
  const window = { escapeHtml };
  const context = { window, self: window };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, 'demand-detail-parent.js'), 'utf8'), context);
  const html = window.DemandDetailParent.renderParentAggregate({
    unitTotal: 1, unitOnline: 0,
    deliveryUnits: [{ demandId: 123, title: '<子需求>', code: 'US123' }],
  });
  assert.match(html, /class="table-title-link" onclick="DemandDetail.open\(123\)"/);
  assert.match(html, /&lt;子需求>/);
});
