#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const js = fs.readFileSync(path.join(root, 'web/static/js/agileteam/agileteam.js'), 'utf8');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

assert(/new URLSearchParams\(location\.search\s*\|\|/.test(js), 'agile team view reads filters from query parameters');
assert(/get\("teamgroupId"\)/.test(js), 'deep links can select one agile team or subteam');
assert(/searchParams\.set\("teamgroupId"/.test(js), 'selected agile group is written back to the URL');
assert(/searchParams\.set\("scope"/.test(js) && /searchParams\.set\("teamgroupId"/.test(js), 'scope kind and selected group remain shareable');
assert(/history\.replaceState/.test(js), 'filter changes update the current URL without a navigation');

console.log('agileteam URL filters regression passed');
