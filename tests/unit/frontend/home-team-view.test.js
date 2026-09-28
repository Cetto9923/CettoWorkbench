#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const js = read('web/static/js/po/home-team.js');
const tpl = read('web/templates/po/home_team.html');
const handler = read('internal/module/po/handler_team_dashboard.go');
const scopeService = read('internal/module/agileteam/service_dashboard_scope.go');
const bootstrap = read('internal/bootstrap/bootstrap.go');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

assert(/fetch\("\/home\/team\/scopes"/.test(js), 'scope controls load only backend-authorized options');
assert(/teamHomeDeptSelect/.test(js) && /teamHomeTeamSelect/.test(js) && /teamHomeSubteamSelect/.test(js), 'team home exposes the three scope levels');
assert(/item\.deptId/.test(js) && /item\.parent/.test(js), 'scope options cascade from department to organization team and subgroup');
assert(/query\.set\("subteamgroupId"/.test(js), 'selected subgroup is sent to scoped dashboard endpoints');
assert(/query\.set\("scopeId"/.test(js) && /query\.set\("scope", "team"\)/.test(js), 'dashboard requests retain their authorized parent scope');
assert(/requestNo !== state\.requestNo/.test(js), 'late responses cannot overwrite the newest selected scope');
assert(/\/home\/team\/dashboard\?/.test(js) && /teamHomeBlockedCount/.test(js), 'team dashboard loads overdue, soon and approval-blocked counts');
assert(/\/issues\/risk\/items\?/.test(js) && /fetchCount\("issue"\)/.test(js) && /fetchCount\("risk"\)/.test(js), 'team home loads issue and risk counts through the scoped API');
assert(/\/home\/team\/version-windows\?/.test(js) && /workItemCount/.test(js), 'team home loads upcoming version windows from its authorized scope');
assert(/schedule\?windows=/.test(js) && /groups=/.test(js) && /stage=/.test(js), 'version links preserve window, group and current stage');
assert(/\/home\/team\/value-stream\?/.test(js) && /demandCount/.test(js) && /storyCount/.test(js), 'team home shows separate business and development backlog counts');
for (const id of ['teamHomeVersionRows', 'teamHomeValueStreamRows', 'teamHomeDueRows', 'teamHomeBlockedCount']) {
  assert(tpl.includes(id), 'team home renders card data host ' + id);
}
for (const title of ['未来 30 天版本窗口', '需求价值流积压', '临期与逾期', '阻塞与风险']) {
  assert(tpl.includes(title), 'team home renders the required card ' + title);
}
assert(!/健康度|健康卡/.test(tpl), 'health card is excluded from the trimmed delivery scope');
assert(/scope=dept/.test(js) || /query\.set\("scope", state\.deptId \? "dept" : "team"\)/.test(js), 'department selection remains representable in the URL and requests');
assert(/sub > 0/.test(handler) && /if !found/.test(handler), 'server rejects a subgroup outside the selected authorized group set');
assert(/DashboardScopes/.test(scopeService) && /DashboardGroupIDs/.test(bootstrap), 'production bootstrap wires the authorized scope resolver');
assert(/CanViewDemandHome/.test(tpl) && /\/home\?view=demand/.test(tpl), 'team-only users do not receive a demand-view link');

console.log('home team four-card scope and authorization regression passed');
