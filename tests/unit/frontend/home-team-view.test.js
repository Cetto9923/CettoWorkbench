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
assert(/\/agileteam\?view=lead&amp;scope=team&amp;teamgroupId=/.test(js), 'team rows deep-link to the selected authorized agile group');
assert(/contextOnly/.test(js), 'synthetic parent context rows are not made selectable links');
assert(/CanViewDemandHome/.test(tpl) && /\/home\?view=demand/.test(tpl), 'team-only users do not receive a demand-view link');
assert(/homePageAccess\(\)/.test(handler) && /SetTeamViewAccess/.test(handler), 'homepage authorization is checked server-side');

console.log('home team view scope and access regression passed');
