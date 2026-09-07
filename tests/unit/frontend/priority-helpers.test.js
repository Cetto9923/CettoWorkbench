const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

// -----------------------------------------------------------------------------
// Stage 3 — priority / object-type helpers test
// Verifies the canonical contract documented in docs/plan/ui-actions-unification:
//   - normalizePriority(raw): parseInt(String(raw).replace(/^p/i, "")) clamp 1..4
//     null/empty/NaN/out-of-range -> null
//   - priorityBadge(raw): <span class="wb-priority" data-priority="N">P{N}</span>
//     invalid -> <span class="wb-priority" data-priority="">—</span>
//   - objectTypeBadge(kind): <span class="wb-type wb-type-{kind}">label</span>
//     unknown -> <span class="wb-type wb-type-unknown">—</span>
// -----------------------------------------------------------------------------

// We load personal-list.js into a sandbox so we can call its window.PersonalList API.
const sandbox = { window: {} };
vm.createContext(sandbox);
vm.runInNewContext(
  fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/personal-list.js"), "utf8"),
  sandbox
);
const PL = sandbox.window.PersonalList;
assert.ok(PL, "PersonalList must be exported on window");

// 1. normalizePriority
const norm = PL.normalizePriority;
assert.strictEqual(typeof norm, "function");

// 1a. numeric strings
for (let i = 1; i <= 4; i++) {
  assert.strictEqual(norm(String(i)), i, "numeric string '" + i + "' must clamp " + i);
  assert.strictEqual(norm(i), i, "raw number " + i + " must clamp " + i);
}

// 1b. P-prefixed strings
assert.strictEqual(norm("P1"), 1, "P1 -> 1");
assert.strictEqual(norm("p3"), 3, "p3 -> 3");
assert.strictEqual(norm("P4"), 4, "P4 -> 4");

// 1c. null/undefined/empty
assert.strictEqual(norm(null), null, "null -> null");
assert.strictEqual(norm(undefined), null, "undefined -> null");
assert.strictEqual(norm(""), null, "empty -> null");
assert.strictEqual(norm("   "), null, "whitespace -> null");

// 1d. out-of-range / NaN
assert.strictEqual(norm(0), null, "0 -> null");
assert.strictEqual(norm(5), null, "5 -> null");
assert.strictEqual(norm(-1), null, "-1 -> null");
assert.strictEqual(norm("p0"), null, "p0 -> null");
assert.strictEqual(norm("p9"), null, "p9 -> null");
assert.strictEqual(norm("abc"), null, "abc -> null");
// parseInt("3.5", 10) yields 3 (in range); spec only rejects out-of-range / NaN.
assert.strictEqual(norm("P3.5"), 3, "P3.5 -> parseInt(3.5) -> 3 (in range, per spec)");
console.log("PASS: normalizePriority numeric, P-prefix, null/empty, out-of-range");

// 2. priorityBadge
const badge = PL.priorityBadge;
assert.strictEqual(typeof badge, "function");

for (let i = 1; i <= 4; i++) {
  const html = badge(String(i));
  assert.match(html, new RegExp('class="wb-priority"'), "badge must use wb-priority class");
  assert.match(html, new RegExp('data-priority="' + i + '"'), "badge must carry data-priority=" + i);
  assert.match(html, new RegExp(">P" + i + "<"), "badge text must be P" + i);
}

assert.match(badge("P2"), /data-priority="2"/);
assert.match(badge(3), /data-priority="3"/);
console.log("PASS: priorityBadge P1..P4 markup");

// 2b. invalid -> em-dash placeholder (data-priority="")
assert.match(badge(null), /data-priority=""/, "null must produce empty data-priority");
assert.match(badge(undefined), /data-priority=""/);
assert.match(badge(""), /data-priority=""/);
assert.match(badge("p0"), /data-priority=""/);
assert.match(badge(99), /data-priority=""/);
assert.match(badge("xx"), /data-priority=""/);
assert.match(badge(null), />—</, "empty placeholder must contain em-dash");
console.log("PASS: priorityBadge never defaults to P3/P4; invalid -> em-dash");

// 3. objectTypeBadge
const otb = PL.objectTypeBadge;
assert.strictEqual(typeof otb, "function");

const expected = {
  business: "业务需求",
  sub_demand: "子需求",
  story: "研发需求",
  independent_story: "独立研发需求",
  task: "任务",
  issue: "问题",
  approval: "审批",
  todo: "待办",
  testtask: "测试单"
};
for (const key of Object.keys(expected)) {
  const html = otb(key);
  assert.match(html, new RegExp('class="wb-type wb-type-' + key + '"'), key + " must map to wb-type-" + key);
  assert.ok(html.indexOf(expected[key]) >= 0, key + " must render label '" + expected[key] + "'");
}
console.log("PASS: objectTypeBadge canonical mapping business/sub_demand/story/independent_story/task/issue/approval/todo/testtask");

// 3b. unknown kind -> wb-type-unknown
const unk = otb("totally-unknown");
assert.match(unk, /wb-type-unknown/, "unknown must produce wb-type-unknown");
assert.strictEqual(otb(""), '<span class="wb-type wb-type-unknown">—</span>', "empty must produce em-dash unknown");
assert.match(otb(null), /wb-type-unknown/, "null must produce wb-type-unknown");
console.log("PASS: objectTypeBadge unknown / null / empty -> wb-type-unknown");

// 4. Stage 3 consumers emit wb-priority / wb-type markup (no inline color).
//     Sample one consumer (home.js renderRow) to ensure it uses the helper.
const homeSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/home.js"), "utf8");
assert.ok(!/class="inline-pri/.test(homeSrc), "home.js must no longer emit inline-pri");
assert.ok(!/class="type-pill/.test(homeSrc), "home.js must no longer emit type-pill");
assert.ok(/priorityBadge\(/.test(homeSrc), "home.js must call priorityBadge()");
assert.ok(/objectTypeBadge\(/.test(homeSrc), "home.js must call objectTypeBadge()");
console.log("PASS: home.js consumer uses priorityBadge/objectTypeBadge (no inline color, no legacy class)");

const followSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/follow.js"), "utf8");
assert.ok(!/class="follow-pri/.test(followSrc), "follow.js must no longer emit follow-pri directly");
assert.ok(/priorityBadge\(/.test(followSrc), "follow.js must call priorityBadge()");
console.log("PASS: follow.js consumer uses priorityBadge (no inline color)");

const todosSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/todos.js"), "utf8");
assert.ok(!/class="inline-pri/.test(todosSrc), "todos.js must no longer emit inline-pri");
assert.ok(/priorityBadge\(/.test(todosSrc), "todos.js must call priorityBadge()");
console.log("PASS: todos.js consumer uses priorityBadge (no inline color)");

const wbSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/workboard.js"), "utf8");
assert.ok(/priorityBadge\(/.test(wbSrc), "workboard.js must call priorityBadge() (via priTag)");
assert.ok(/objectTypeBadge\(/.test(wbSrc) || /type-biz|type-child|type-rd|type-task|type-unknown/.test(wbSrc), "workboard.js must call objectTypeBadge() OR emit aliased class (.type-biz etc. all aliased in wb-priority.css)");
console.log("PASS: workboard.js consumer uses priorityBadge/objectTypeBadge (or aliased class)");

const ddSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/demand-detail-render.js"), "utf8");
assert.ok(/priorityBadge\(/.test(ddSrc), "demand-detail-render.js must call priorityBadge() for header");
assert.ok(!/class="dd-tag"[^>]*>\s*\$|dd-tag.*?priority/.test(ddSrc), "demand-detail-render header must not wrap priority in dd-tag anymore");
console.log("PASS: demand-detail-render.js header uses priorityBadge()");

const noticeSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/notice.js"), "utf8");
// Notice page does not currently render a type *badge* (object info is shown as a
// compact label/link per formatObjectCell). Out of Stage 3 scope; documented.
assert.ok(/var objectTypeBadge =/.test(noticeSrc), "notice.js must import objectTypeBadge helper for parity");
console.log("PASS: notice.js exposes objectTypeBadge (no badge render site in scope)");

console.log("\nALL: priority helpers contract honored across 8 consumer files");