#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '../../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const js = read('web/static/js/po/workboard.js');
const core = read('web/static/js/po/workboard-core.js');
const css = read('web/static/css/po/board.css');
function assert(condition, message) {
  if (!condition) {
    console.error('FAIL ' + message);
    process.exit(1);
  }
}

assert(/#taskStats \[data-flag\]/.test(js), 'task stat buttons are wired independently from demand stats');
assert(/activeTaskFlag/.test(js) && /toggleTaskFlag/.test(js), 'task filters keep their own active flag');
assert(/card\.classList\.toggle\("hidden", !match\)/.test(js), 'task cards are hidden when the selected flag does not match');
assert(/applyTaskFilters\(\);/.test(js), 'task filters reapply after loading task columns');
assert(/\.task-card\.hidden\{display:none\}/.test(css), 'filtered task cards are removed from layout');
assert(/visibleMetricKeys\s*=\s*\{ delivery: true, implement: true, overIteration: true, unscheduled: true \}/.test(js),
  'demand board temporarily keeps only delivery rhythm metrics visible');
assert(/metrics\s*=\s*\(metrics \|\| \[\]\)\.filter\(function \(m\)/.test(js),
  'demand board filters hidden metrics before rendering');
assert(/urlParams\.get\("teamgroupId"\)/.test(core), 'task board restores the requested agile group from URL');
assert(/urlParams\.get\("ownerAccount"\)/.test(core), 'task board restores the requested owner from URL');
assert(/urlParams\.get\("focus"\)/.test(core) && /state\.focus/.test(js), 'task board restores the overdue/blocked focus from URL');
assert(/function syncUrl\(nextMode\)/.test(core) && /WB\.syncUrl\("task"\)/.test(js), 'task filters stay shareable after selection and pagination');
assert(/params\.set\("teamgroupId"/.test(core) && /params\.set\("ownerAccount"/.test(core) && /params\.set\("focus"/.test(core),
  'task board deep links preserve teamgroupId, ownerAccount and focus');

console.log('workboard task filters regression passed');
