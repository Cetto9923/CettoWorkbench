const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '../../..');
const home = fs.readFileSync(path.join(root, 'web/templates/po/home.html'), 'utf8');
const issueRisk = fs.readFileSync(path.join(root, 'web/static/js/po/issue-risk.js'), 'utf8');
const handler = fs.readFileSync(path.join(root, 'internal/module/po/handler.go'), 'utf8');

assert.match(home, /阻塞需求/);
assert.match(home, /href="\/home\?focus=blocked"/);
assert.match(home, /IssueRiskCounts\.Issues/);
assert.match(home, /IssueRiskCounts\.Risks/);
assert.match(home, /kind=issue(?:&amp;|&)relation=allRelated(?:&amp;|&)loop=open/);
assert.match(home, /kind=risk(?:&amp;|&)relation=allRelated(?:&amp;|&)loop=open/);
assert.match(handler, /currentUserHasPerm\(c, perm\.PoBoardDemandList\)/);
assert.match(issueRisk, /function initFromUrl\(\)/);
assert.match(issueRisk, /function syncUrl\(\)/);
for (const key of ['kind', 'relation', 'loop', 'overdue', 'status', 'keyword', 'project', 'page', 'pageSize']) {
  assert.match(issueRisk, new RegExp(`params\\.get\\("${key}"\\)`), `issue-risk URL must read ${key}`);
}

console.log('PASS: homepage blocker and issue/risk counts have matching destinations and deep links');
