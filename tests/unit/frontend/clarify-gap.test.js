// =============================================================================
// 文件: tests/unit/frontend/clarify-gap.test.js
// 职责: 澄清入口显示、重复监听及错误响应回归。
// =============================================================================
const assert = require('node:assert/strict');
const fs = require('node:fs');
global.window = { PersonalList: { priorityBadge: () => '' }, escapeHtml: value => String(value == null ? '' : value) };
const render = require('../../../web/static/js/po/demand-detail-render.js');
for (const canClarify of [true, false]) {
  const html = render.renderTabRequirement({ demandId: 42, canClarify });
  assert.equal(html.includes('js-drawer-clarify-btn'), canClarify);
}
const clarify = fs.readFileSync('web/static/js/po/demand-clarify.js', 'utf8');
const detail = fs.readFileSync('web/static/js/po/demand-detail.js', 'utf8');
assert.equal(detail.includes('var clarifyBtn ='), false);
assert.ok(detail.includes('e.target.closest(".js-drawer-clarify-btn, .js-po-drawer-action')); 
assert.equal(clarify.includes('handleClarifySuccess("该需求已完成澄清或状态已更新")'), false);
assert.ok(clarify.includes('if (!res.ok)'));
console.log('clarify gap regression passed');
