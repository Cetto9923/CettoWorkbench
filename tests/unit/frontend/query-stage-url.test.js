#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const js = fs.readFileSync(path.join(root, 'web/static/js/query/query.js'), 'utf8');

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

assert(/stage:\s*""/.test(js), 'query state carries a value-stream stage');
assert(/q\.set\("stage", state\.stage\)/.test(js), 'stage is sent to the existing query endpoint');
assert(/sp\.get\("stage"\)/.test(js), 'query page restores stage from the URL');
assert(/p\.set\("stage", state\.stage\)/.test(js), 'selected stage remains in shareable URLs');
assert(/state\.stage\s*=\s*""/.test(js), 'reset clears the stage deep-link filter');

console.log('query stage URL filter regression passed');
