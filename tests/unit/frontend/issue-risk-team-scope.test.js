#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const js = read('web/static/js/po/issue-risk.js');
const html = read('web/templates/po/issue-risk.html');
const handler = read('internal/module/po/handler_issue_risk.go');
const routes = read('internal/module/po/handler.go');
const repo = read('internal/module/po/repo_issue_risk.go');
const bootstrap = read('internal/bootstrap/bootstrap.go');

function assert(condition, message) {
  if (!condition) {
    console.error('FAIL ' + message);
    process.exit(1);
  }
}

assert(/scope:\s*""/.test(js) && /scopeId:\s*0/.test(js), 'issue-risk page keeps an explicit team-scope state');
assert(/params\.get\("scope"\)/.test(js) && /params\.get\("scopeId"\)/.test(js), 'issue-risk page restores scope from URL');
assert(/params\.set\("scope", state\.scope\)/.test(js) && /params\.set\("scopeId", String\(state\.scopeId\)\)/.test(js), 'issue-risk API request and URL retain scope');
assert(/if \(state\.scope\) \{ state\.relation = "allRelated"; \}/.test(js) && /relSel\.disabled = !!state\.scope/.test(js), 'team scope cannot be narrowed by personal-only relation controls');
assert(/irTeamScopeNotice/.test(html) && /按创建人或处理人筛选/.test(js), 'team scope is visible to the user');
assert(/teamScopeAccounts\(c\.Request\.Context\(\), middleware\.CurrentUser\(c\), req\.Scope, req\.ScopeID\)/.test(handler), 'API resolves authorized members server-side');
assert(/issueRiskPageAccess\(\)/.test(routes) && /issueRiskItemsAccess\(\)/.test(routes), 'team-only users pass explicit server-side page and API access checks');
assert(/SetTeamScopeAccounts\(agileTeamSvc\.LeadScopeMemberAccounts\)/.test(bootstrap), 'production bootstrap wires the authorized scope resolver');
assert(/createdBy IN \? OR %s\.assignedTo IN/.test(repo) && /len\(req\.teamAccounts\) == 0/.test(repo), 'database filters on scoped members and fails closed on empty membership');

console.log('issue-risk team scope regression passed');
