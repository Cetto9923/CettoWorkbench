const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

// 1. Verify template contract
const htmlPath = path.join(__dirname, '../../../web/templates/schedule/index.html');
const html = fs.readFileSync(htmlPath, 'utf8');

assert.ok(html.includes('class="schedule-ms-search"'), 'index.html must include schedule-ms-search');
assert.ok(html.includes('class="schedule-ms-search-input"'), 'index.html must include schedule-ms-search-input');
assert.ok(html.includes('class="schedule-ms-search-clear"'), 'index.html must include schedule-ms-search-clear');
assert.ok(html.includes('class="schedule-ms-tag">我参与</span>'), 'index.html must include 我参与 tag for IsMyProduct');
assert.ok(html.includes('class="schedule-ms-no-match"'), 'index.html must include schedule-ms-no-match element');

// 2. Verify CSS styles
const cssPath = path.join(__dirname, '../../../web/static/css/schedule/schedulefilter.css');
const css = fs.readFileSync(cssPath, 'utf8');

assert.ok(css.includes('.schedule-ms-search'), 'schedulefilter.css must style .schedule-ms-search');
assert.ok(css.includes('.schedule-ms-search-input'), 'schedulefilter.css must style .schedule-ms-search-input');
assert.ok(css.includes('.schedule-ms-tag'), 'schedulefilter.css must style .schedule-ms-tag');
assert.ok(css.includes('.schedule-ms-no-match'), 'schedulefilter.css must style .schedule-ms-no-match');

// 3. Verify JS behavior logic in schedulefilter.js
const jsPath = path.join(__dirname, '../../../web/static/js/schedule/schedulefilter.js');
const js = fs.readFileSync(jsPath, 'utf8');

assert.ok(js.includes('filterMultiselectOptions'), 'schedulefilter.js must define filterMultiselectOptions');
assert.ok(js.includes('.schedule-ms-search-input'), 'schedulefilter.js must bind to .schedule-ms-search-input');
assert.ok(js.includes('.schedule-ms-search-clear'), 'schedulefilter.js must handle .schedule-ms-search-clear');
assert.ok(js.includes('.schedule-ms-no-match'), 'schedulefilter.js must toggle .schedule-ms-no-match');

console.log('PASS: schedule filter search template, css, and js contracts verified');
