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

// 1. 测试 idChipHtml 健壮解析、短标签映射与方案 B 双段式工程微标结构 (Linear Two-tone Badges)
const cases = [
  { kind: "business", id: "123", expectedClass: "wb-type-business", expectedTag: "业需" },
  { kind: "demand", id: "456", expectedClass: "wb-type-business", expectedTag: "业需" },
  { kind: "业务需求", id: "789", expectedClass: "wb-type-business", expectedTag: "业需" },
  { kind: "story", id: "101", expectedClass: "wb-type-story", expectedTag: "研需" },
  { kind: "研发需求", id: "102", expectedClass: "wb-type-story", expectedTag: "研需" },
  { kind: "sub_demand", id: "103", expectedClass: "wb-type-sub_demand", expectedTag: "子需" },
  { kind: "task", id: "201", expectedClass: "wb-type-task", expectedTag: "任务" },
  { kind: "bug", id: "301", expectedClass: "wb-type-bug", expectedTag: "Bug" },
  { kind: "testtask", id: "401", expectedClass: "wb-type-testtask", expectedTag: "测单" },
  { kind: "test", id: "402", expectedClass: "wb-type-testtask", expectedTag: "测单" },
  { kind: "approval", id: "501", expectedClass: "wb-type-approval", expectedTag: "审批" },
  { kind: "charter", id: "601", expectedClass: "wb-type-charter", expectedTag: "章程" },
  { kind: "feedback", id: "701", expectedClass: "wb-type-feedback", expectedTag: "反馈" },
  { kind: "issue", id: "801", expectedClass: "wb-type-issue", expectedTag: "问题" }
];

for (const c of cases) {
  const html = PL.idChipHtml(c.kind, c.id);
  assert.ok(html.includes(c.expectedClass), `kind ${c.kind} must contain class ${c.expectedClass}`);
  assert.ok(html.includes('<span class="wb-type-tag">' + c.expectedTag + '</span>'), `kind ${c.kind} must contain tag ${c.expectedTag}`);
  assert.ok(html.includes('<span class="wb-type-id">#' + c.id + '</span>'), `kind ${c.kind} must contain id #${c.id}`);
}

// 1b. 验证链接 ID 场景（格式化 # 自动置于链接内部）
const linkHtml = PL.idChipHtml("business", '<a class="table-id-link" href="/demands/123">US6319</a>');
assert.ok(linkHtml.includes('<span class="wb-type-tag">业需</span>'));
assert.ok(linkHtml.includes('<span class="wb-type-id"><a class="table-id-link" href="/demands/123">#US6319</a></span>'));

console.log("PASS: PersonalList.idChipHtml renders Option B linear two-tone badges with correct tags and IDs");

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

const scheduleHtml = fs.readFileSync(path.join(root, "web/templates/schedule/index.html"), "utf8");
assert.ok(scheduleHtml.includes("对象#ID"), "schedule/index.html header must contain 对象#ID");
assert.ok(scheduleHtml.includes("需求标题"), "schedule/index.html header must contain 需求标题");
assert.ok(scheduleHtml.includes("table-scroll-container"), "schedule/index.html must use table-scroll-container");
console.log("PASS: schedule/index.html unifies to 对象#ID and 需求标题 with table-scroll-container");

// 3. 检查 JS 实现中消费 idChipHtml
const todosJs = fs.readFileSync(path.join(root, "web/static/js/po/todos.js"), "utf8");
assert.ok(todosJs.includes("todos-col-obj"), "todos.js must render todos-col-obj cell");
assert.ok(todosJs.includes("idChipHtml"), "todos.js must use idChipHtml");

const followJs = fs.readFileSync(path.join(root, "web/static/js/po/follow.js"), "utf8") +
  fs.readFileSync(path.join(root, "web/static/js/po/follow-demand.js"), "utf8");
assert.ok(followJs.includes("idChipHtml"), "follow.js must use idChipHtml");

console.log("PASS: todos.js and follow.js use idChipHtml for single-source-of-truth rendering");
console.log("\nALL: unified Object#ID contract verified successfully!");
