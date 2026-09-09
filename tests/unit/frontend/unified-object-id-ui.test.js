/* =============================================================================
   文件: tests/unit/frontend/unified-object-id-ui.test.js
   模块: PO 工作台 - 统一对象与 ID 的 UI 规范回归测试
   职责: 验证 PersonalList.idChipHtml 归一化解析、各页面对象#ID表头规范及合并列渲染。
   ============================================================================= */

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

const root = path.join(__dirname, "../../..");
const plPath = path.join(root, "web/static/js/po/personal-list.js");
global.window = global;
require(plPath);

const PL = global.PersonalList;
assert.ok(PL, "PersonalList must be exported");
assert.strictEqual(typeof PL.idChipHtml, "function", "idChipHtml must be a function");

// 1. 测试 idChipHtml 健壮解析与短标签映射
const cases = [
  { kind: "business", id: "123", expectedClass: "wb-type-business", expectedLabel: "业需#123" },
  { kind: "demand", id: "456", expectedClass: "wb-type-business", expectedLabel: "业需#456" },
  { kind: "业务需求", id: "789", expectedClass: "wb-type-business", expectedLabel: "业需#789" },
  { kind: "story", id: "101", expectedClass: "wb-type-story", expectedLabel: "研需#101" },
  { kind: "研发需求", id: "102", expectedClass: "wb-type-story", expectedLabel: "研需#102" },
  { kind: "sub_demand", id: "103", expectedClass: "wb-type-sub_demand", expectedLabel: "子需#103" },
  { kind: "task", id: "201", expectedClass: "wb-type-task", expectedLabel: "任务#201" },
  { kind: "bug", id: "301", expectedClass: "wb-type-bug", expectedLabel: "Bug#301" },
  { kind: "testtask", id: "401", expectedClass: "wb-type-testtask", expectedLabel: "测单#401" },
  { kind: "test", id: "402", expectedClass: "wb-type-testtask", expectedLabel: "测单#402" },
  { kind: "approval", id: "501", expectedClass: "wb-type-approval", expectedLabel: "审批#501" },
  { kind: "charter", id: "601", expectedClass: "wb-type-charter", expectedLabel: "章程#601" },
  { kind: "feedback", id: "701", expectedClass: "wb-type-feedback", expectedLabel: "反馈#701" },
  { kind: "issue", id: "801", expectedClass: "wb-type-issue", expectedLabel: "问题#801" }
];

for (const c of cases) {
  const html = PL.idChipHtml(c.kind, c.id);
  assert.ok(html.includes(c.expectedClass), `kind ${c.kind} must contain class ${c.expectedClass}`);
  assert.ok(html.includes(c.expectedLabel), `kind ${c.kind} must contain label ${c.expectedLabel}`);
}
console.log("PASS: PersonalList.idChipHtml normalizes canonical keys, API kinds, and Chinese labels correctly");

// 2. 检查 HTML 模板表头契约
const homeHtml = fs.readFileSync(path.join(root, "web/templates/po/home.html"), "utf8");
assert.ok(homeHtml.includes("对象#ID"), "home.html header must contain 对象#ID");
assert.ok(!homeHtml.includes(">需求编号<"), "home.html header must not contain 需求编号");
console.log("PASS: home.html header is unified to 对象#ID");

const doneHtml = fs.readFileSync(path.join(root, "web/templates/po/done.html"), "utf8");
assert.ok(doneHtml.includes(">对象#ID<"), "done.html header must contain 对象#ID");
assert.ok(!doneHtml.includes(">对象<"), "done.html header must not contain solitary 对象");
console.log("PASS: done.html header is unified to 对象#ID");

const todosHtml = fs.readFileSync(path.join(root, "web/templates/po/todos.html"), "utf8");
assert.ok(todosHtml.includes("todos-col-obj"), "todos.html must have todos-col-obj column");
assert.ok(todosHtml.includes(">对象#ID<"), "todos.html header must contain 对象#ID");
console.log("PASS: todos.html has independent 对象#ID column");

const followHtml = fs.readFileSync(path.join(root, "web/templates/po/follow.html"), "utf8");
assert.ok(followHtml.includes(">对象#ID<"), "follow.html header must contain 对象#ID");
assert.ok(!followHtml.includes(">对象类型<"), "follow.html must not have separate 对象类型 column");
console.log("PASS: follow.html merges ID and object type into 对象#ID");

// 3. 检查 JS 实现中消费 idChipHtml
const todosJs = fs.readFileSync(path.join(root, "web/static/js/po/todos.js"), "utf8");
assert.ok(todosJs.includes("todos-col-obj"), "todos.js must render todos-col-obj cell");
assert.ok(todosJs.includes("idChipHtml"), "todos.js must use idChipHtml");

const followJs = fs.readFileSync(path.join(root, "web/static/js/po/follow.js"), "utf8");
assert.ok(followJs.includes("idChipHtml"), "follow.js must use idChipHtml");

console.log("PASS: todos.js and follow.js use idChipHtml for single-source-of-truth rendering");
console.log("\nALL: unified Object#ID contract verified successfully!");
