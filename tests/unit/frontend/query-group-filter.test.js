#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const root = path.resolve(__dirname, '../../..');
const js = fs.readFileSync(path.join(root, 'web/static/js/query/query.js'), 'utf8');
const tpl = fs.readFileSync(path.join(root, 'web/templates/query/index.html'), 'utf8');

function assert(condition, message) {
    if (!condition) throw new Error(message);
}

assert(/group:\s*""/.test(js), 'query state carries a value-stream group');
assert(/q\.set\("group", state\.group\)/.test(js), 'group is sent to the existing query endpoint');
assert(/sp\.get\("group"\)/.test(js), 'query page restores group from the URL');
assert(/p\.set\("group", state\.group\)/.test(js), 'selected group remains in shareable URLs');
assert(/state\.group\s*=\s*""/.test(js), 'reset clears the agile-group filter');
assert(/\$\("queryGroup"\)\.value = state\.group/.test(js), 'the group select is hydrated on load');
assert(/\$\("queryGroup"\)\.value = ""/.test(js), 'the group select is cleared by reset');

assert(/id="queryGroup"/.test(tpl), 'toolbar renders the agile-group select');
assert(/全部敏捷小组/.test(tpl), 'group select exposes an all-groups default option');
assert(/range \.Groups/.test(tpl), 'group options loop over the injected option list');
assert(/eq \$\.Group \(printf "%d" \.ID\)/.test(tpl), 'group select reflects the selected group back');

console.log('query agile-group filter regression passed');
