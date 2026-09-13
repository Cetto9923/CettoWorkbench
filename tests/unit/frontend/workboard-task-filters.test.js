#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

const root = path.resolve(__dirname, '../../..');
const read = file => fs.readFileSync(path.join(root, file), 'utf8');
const js = read('web/static/js/po/workboard.js');
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

console.log('workboard task filters regression passed');
