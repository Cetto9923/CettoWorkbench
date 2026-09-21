const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const path = require('node:path');

const source = fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/workboard.js'), 'utf8');
const issueSource = fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/workboard-issue.js'), 'utf8');
// Parse the whole entry point: an invalid callback statement prevents every request.
new vm.Script(source);
new vm.Script(issueSource);
assert.match(issueSource, /\/board\/issues\/" \+ encodeURIComponent\(issueID\) \+ "\/transition/);
assert.doesNotMatch(issueSource, /currentDrawerIssue\.status\s*=/);
const render = source.slice(source.indexOf('  function renderDemandMatrix('), source.indexOf('  function refreshDemandToggleAll('));
const standalone = source.slice(source.indexOf('  function renderStandaloneRow('), source.indexOf('  function collectStories('));
const host = { innerHTML: '', querySelectorAll: () => [] };
const context = {
  $: id => id === 'demandGroups' ? host : id === 'demandEmpty' ? { style: {} } : null,
  nodeTypeOf: () => 'business', isOverdue: () => false, isBlocked: () => false,
  collectStories: () => [], esc: String, priTag: () => '', typeTag: () => '',
  renderStandaloneRow: node => 'root:' + node.id,
  renderCollapsedRow: () => '', renderDemandRow: (node, depth, last) => 'child:' + node.id + ':' + last,
  refreshDemandToggleAll: () => {}, applyDemandFilters: () => {}, renderSummaryFromRows: () => {}
};
vm.createContext(context);
vm.runInContext(render, context);
context.nodeTypeOf = () => 'independentRd';
context.ownerBadge = () => '';
context.renderStageCards = () => '';
vm.runInContext(standalone, context);
const independentHTML = context.renderStandaloneRow({ id: 61532, displayId: '61532', title: '独立研发需求', independent: true, url: 'http://zentao.test/story/view?id=61532' });
assert.match(independentHTML, /href="http:\/\/zentao\.test\/story\/view\?id=61532"/);
assert.doesNotMatch(independentHTML, /data-open-demand/);
const independentWithoutURL = context.renderStandaloneRow({ id: 61533, displayId: '61533', title: '无链接独立研发需求', independent: true });
assert.doesNotMatch(independentWithoutURL, /data-open-demand/);
assert.match(independentWithoutURL, /<span class="node-title"/);
context.nodeTypeOf = () => 'business';
const demandHTML = context.renderStandaloneRow({ id: 100, displayId: 'US100', title: '业务需求', independent: false });
assert.match(demandHTML, /data-open-demand="100"/);
assert.match(demandHTML, /node-meta"><span class="code"># US100<\/span>/);
assert.doesNotMatch(demandHTML, /node-title-line"><span class="code">/);
context.renderStandaloneRow = node => 'root:' + node.id;
context.renderDemandMatrix([{ id: 1, children: [
  { id: 2, kind: 'sub_demand' },
  { id: 3, kind: 'story', independent: false }
] }]);
assert.match(host.innerHTML, /child:2:true/);
assert.doesNotMatch(host.innerHTML, /child:3/);
context.renderDemandMatrix([{ id: 4, children: [{ id: 5, kind: 'story', independent: false }] }]);
assert.equal(host.innerHTML, 'root:4');
context.renderDemandMatrix([{ id: 6, kind: 'story', independent: true }]);
assert.equal(host.innerHTML, 'root:6');
console.log('PASS: board script parses; child demands remain; associated stories create no rows or empty groups; independent roots remain');
