#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const js = read('web/static/js/po/home-team.js');
const tpl = read('web/templates/po/home_team.html');
const handler = read('internal/module/po/handler.go');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

assert(/view:\s*"lead"/.test(js), 'team home asks the backend for the caller scoped read-only view');
assert(/query\.set\("scopeId"/.test(js) && /teamgroupId/.test(js), 'team home filters by the selected agile team or subteam');
assert(/availableScopes/.test(js) && /scopeOptions/.test(js), 'scope controls use only backend-authorized options');
assert(/requestNo !== state\.requestNo/.test(js), 'late responses cannot overwrite the newest selected scope');
assert(/formalCount/.test(js) && /pendingAdd/.test(js) && /pendingRemove/.test(js), 'team list renders real membership and pending-adjustment counts');
assert(/\/issues\/risk\/items\?/.test(js) && /fetchCount\("issue"\)/.test(js) && /fetchCount\("risk"\)/.test(js), 'team home loads both open issue and risk counts from the scoped API');
assert(/teamHomeIssueCount/.test(tpl) && /teamHomeRiskCount/.test(tpl), 'team home exposes separate issue and risk totals');
assert(/teamHomeRiskLink/.test(js) && /scopeId/.test(js), 'issue-risk summary link preserves the selected team scope');
assert(/\/home\/team\/version-windows\?/.test(js) && /workItemCount/.test(js), 'team home loads deduplicated upcoming version windows from its authorized scope');
assert(/teamHomeVersionRows/.test(tpl) && /未来 30 天版本窗口/.test(tpl), 'team home exposes an upcoming version-window panel');
assert(/schedule\?windows=/.test(js) && /&amp;groups=/.test(js), 'window links preserve the existing schedule filters');
assert(/\/home\/team\/value-stream\?/.test(js) && /demandCount/.test(js) && /storyCount/.test(js), 'team home loads separate business and development demand stage counts');
assert(/teamHomeValueStreamRows/.test(tpl) && /需求价值流积压/.test(tpl), 'team home exposes its nine-stage backlog panel');
assert(/\/agileteam\?view=lead&amp;scope=' \+ encodeURIComponent\(state\.scope\)/.test(js), 'team rows preserve the selected authorized scope when opening an agile group');
assert(/\/issues\/risk\?kind=issue&amp;loop=open&amp;scope=' \+ encodeURIComponent\(state\.scope\)/.test(js), 'team rows deep-link to issue-risk with the selected authorized scope');
assert(/contextOnly/.test(js), 'synthetic parent context rows are not made selectable links');
assert(/CanViewDemandHome/.test(tpl) && /\/home\?view=demand/.test(tpl), 'team-only users do not receive a demand-view link');
assert(/homePageAccess\(\)/.test(handler) && /SetTeamViewAccess/.test(handler), 'homepage authorization is checked server-side');

console.log('home team view scope and access regression passed');
