/* =============================================================================
   文件: tests/unit/frontend/todos-object-chips.test.js
   模块: PO 个人工作台 - 我的待办
   职责: 回归"我的待办分类芯片与我的已办同形"：芯片来自服务端 facets、对象维度只有
         objectType 单一状态、不再渲染未接入的对象类型或冗余的"具体对象"下拉。
   ============================================================================= */

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const root = path.join(__dirname, "../../..");
const scriptSource = fs.readFileSync(path.join(root, "web/static/js/po/todos.js"), "utf8");
const template = fs.readFileSync(path.join(root, "web/templates/po/todos.html"), "utf8");
const doneScript = fs.readFileSync(path.join(root, "web/static/js/po/done.js"), "utf8");
const sharedCss = fs.readFileSync(path.join(root, "web/static/css/po/personal-workspace.css"), "utf8");
const todosCss = fs.readFileSync(path.join(root, "web/static/css/po/todos.css"), "utf8");

// makeNode 提供最小 DOM 桩。芯片按钮按当前 innerHTML 生成一次并缓存，
// 这样 renderObjectChips 绑定的 click 处理器和测试后续取到的是同一批对象。
function makeNode() {
  let html = "";
  let buttons = null;
  return {
    value: "",
    hidden: false,
    textContent: "",
    handlers: {},
    classList: { toggle() {}, add() {}, remove() {} },
    addEventListener(event, fn) { this.handlers[event] = fn; },
    setAttribute() {},
    removeAttribute() {},
    get innerHTML() { return html; },
    set innerHTML(value) { html = value; buttons = null; },
    querySelectorAll(selector) {
      if (selector !== ".wb-done-tab") { return []; }
      if (!buttons) {
        buttons = [...html.matchAll(/data-object-type="([^"]*)"/g)].map(([, key]) => ({
          handlers: {},
          getAttribute(name) { return name === "data-object-type" ? key : null; },
          addEventListener(event, fn) { this.handlers[event] = fn; },
          click() { if (this.handlers.click) { this.handlers.click(); } }
        }));
      }
      return buttons;
    }
  };
}

// loadTodos 在受控 DOM 中执行 todos.js，返回芯片宿主节点与最近一次列表请求信息。
function loadTodos(search) {
  const nodes = new Map();
  const requests = [];
  let ready = null;

  const document = {
    getElementById(id) {
      if (!nodes.has(id)) { nodes.set(id, makeNode()); }
      return nodes.get(id);
    },
    querySelectorAll() { return []; },
    addEventListener(event, fn) { if (event === "DOMContentLoaded") { ready = fn; } }
  };

  const sandbox = {
    document,
    URLSearchParams,
    setTimeout,
    clearTimeout,
    console,
    Math,
    Number,
    Object,
    String
  };
  sandbox.window = sandbox;
  sandbox.escapeHtml = (v) => String(v == null ? "" : v);
  sandbox.location = { pathname: "/todos", search: search || "" };
  sandbox.history = { replaceState(_state, _title, url) { sandbox.lastUrl = url; } };
  sandbox.PersonalList = {
    PAGE_SIZE_OPTIONS: [10, 12, 15, 20, 50, 100],
    escapeHtml: (v) => String(v == null ? "" : v),
    loadPageSize: (_key, fallback) => fallback,
    savePageSize() {},
    renderPagination() {},
    priorityBadge: () => "",
    objectTypeBadgeFromKind: () => "",
    idChipHtml: () => "",
    createController: () => ({
      fetch(url, _state, onDone) { requests.push({ url, onDone }); }
    })
  };
  sandbox.PrimaryAction = { primaryActionHtml: () => "" };

  vm.createContext(sandbox);
  vm.runInContext(scriptSource, sandbox);
  assert.ok(ready, "todos.js must register a DOMContentLoaded handler");
  ready();

  return { chips: document.getElementById("todosObjectChips"), requests, sandbox, document };
}

/* 1. 模板：分类栏改为与已办同款空容器，并移除冗余的"具体对象"下拉 */
assert.match(
  template,
  /<div class="wb-done-domains" id="todosObjectChips" role="group" aria-label="操作对象过滤">/,
  "待办必须复用已办的 wb-done-domains 芯片容器"
);
assert.doesNotMatch(template, /category-tab/, "不应保留旧的硬编码分类 Tab");
assert.doesNotMatch(template, /todosObjectType/, '分类芯片已是对象类型，"具体对象"下拉必须移除');

// 2a. 审批对象额外展示服务端返回的具体审批场景。
assert.match(template, /<th class="todos-col-approval" id="todosApprovalHeader" hidden>审批场景<\/th>/);
assert.match(template, /id="todosMetricHeader"/);
assert.match(scriptSource, /item\.approvalScene/);
assert.match(scriptSource, /todosApprovalHeader/);
assert.match(scriptSource, /item\.severity/);
assert.match(scriptSource, /风险等级/);
assert.match(scriptSource, /严重程度/);
assert.match(scriptSource, /unconfirmed:\s*"未确认"/);
assert.match(todosCss, /\.todos-col-approval\s*\{/);

/* 2. CSS：芯片几何在共享层定义一次，页面 CSS 不复制色板 */
assert.match(sharedCss, /\.wb-done-tab\s*\{/, "芯片样式必须提取到 personal-workspace.css 共享层");
assert.doesNotMatch(todosCss, /\.wb-done-tab/, "todos.css 不应复制一份芯片样式");

/* 3. 芯片集合与已办一致的渲染契约，且只覆盖待办实际接入的对象类型 */
const { chips, requests, sandbox } = loadTodos("");
assert.equal(requests.length, 1, "首屏应发起一次列表请求");
assert.match(requests[0].url, /^\/todos\/items\?/, "列表请求必须打到 /todos/items");
assert.doesNotMatch(requests[0].url, /(\?|&)tab=/, "对象维度只保留 objectType，不再并存 tab");

requests[0].onDone({
  items: [],
  total: 9,
  summary: { pending: 9, today: 1, overdue: 2, blocked: 0, p1: 3 },
  facets: [
    { key: "demand", label: "业务需求", count: 5 },
    { key: "task", label: "任务", count: 3 },
    { key: "bug", label: "Bug", count: 1 }
  ]
});

const renderedKeys = [...chips.innerHTML.matchAll(/data-object-type="([^"]*)"/g)].map((m) => m[1]);
assert.deepEqual(renderedKeys, ["", "demand", "task", "bug"], "芯片顺序必须是 全部/业务需求/任务/Bug");
assert.doesNotMatch(chips.innerHTML, /data-object-type="story"/, "story 待办数据源未接入，不能渲染成可点芯片");
const renderedCounts = [...chips.innerHTML.matchAll(/wb-done-tab-count">(\d+)</g)].map((m) => Number(m[1]));
assert.deepEqual(renderedCounts, [9, 5, 3, 1], '"全部待办"必须等于各分面之和，其余取服务端聚合值');
assert.match(chips.innerHTML, /class="wb-done-tab active" data-object-type=""/, "默认选中全部待办");
assert.match(chips.innerHTML, /fa-lightbulb/, "芯片必须带与已办同款图标");
console.log("PASS: todos object chips render from server facets with done-page contract");

/* 4. 点击芯片切换 objectType 并重新拉取（回到第一页） */
const demandChip = chips.querySelectorAll(".wb-done-tab")[1];
demandChip.click();
assert.equal(requests.length, 2, "点击芯片必须触发新的列表请求");
assert.match(requests[1].url, /objectType=demand/, "点击芯片按 objectType 过滤");
assert.match(requests[1].url, /page=1/, "切换分类后回到第一页");
assert.equal(sandbox.lastUrl, "/todos?objectType=demand", "URL 只同步 objectType，不写回 tab");
console.log("PASS: todos chip click drives objectType filtering and URL sync");

/* 5. URL 深链恢复选中态；all 归一化为空 key */
const deep = loadTodos("?objectType=bug");
deep.requests[0].onDone({ items: [], total: 1, summary: {}, facets: [{ key: "bug", label: "Bug", count: 1 }] });
assert.match(deep.chips.innerHTML, /class="wb-done-tab active" data-object-type="bug"/, "深链必须恢复芯片选中态");

const legacy = loadTodos("?objectType=all");
legacy.requests[0].onDone({ items: [], total: 0, summary: {}, facets: [] });
assert.match(legacy.chips.innerHTML, /class="wb-done-tab active" data-object-type=""/, "objectType=all 归一化为全部待办");
console.log("PASS: todos chip state restores from URL and normalizes legacy objectType=all");

/* 6. 与已办保持同一组 class 名，避免两页各自分裂出一套芯片实现 */
assert.ok(template.includes("wb-done-domains"), "待办模板必须复用已办的芯片容器 class");
["wb-done-tab", "wb-done-tab-count"].forEach((cls) => {
  assert.ok(doneScript.includes(cls), `${cls} 必须仍是已办的芯片 class`);
  assert.ok(scriptSource.includes(cls), `todos.js 必须复用 ${cls}`);
});
console.log("PASS: todos and done share one object-chip class contract");

/* 7. 支持全量对象类型（approval, demand, story, task, bug, risk, issue, todo, testtask），只要有数据即展示 */
const fullFlow = loadTodos("");
fullFlow.requests[0].onDone({
  items: [],
  total: 20,
  summary: { pending: 20, today: 2, overdue: 1, blocked: 0, p1: 3 },
  facets: [
    { key: "approval", label: "审批", count: 2 },
    { key: "demand", label: "业务需求", count: 5 },
    { key: "story", label: "研发需求", count: 3 },
    { key: "task", label: "任务", count: 4 },
    { key: "bug", label: "Bug", count: 2 },
    { key: "risk", label: "风险", count: 1 },
    { key: "issue", label: "问题", count: 1 },
    { key: "todo", label: "待办", count: 1 },
    { key: "testtask", label: "测试单", count: 1 }
  ]
});
const fullKeys = [...fullFlow.chips.innerHTML.matchAll(/data-object-type="([^"]*)"/g)].map((m) => m[1]);
assert.deepEqual(fullKeys, ["", "approval", "demand", "story", "task", "bug", "risk", "issue", "todo", "testtask"], "全量对象有数据时按预定顺序完整展示分类芯片");
console.log("PASS: todos full object types render chips when data exists");

/* 8. 模板：支持审批场景下拉，包含项目章程、建设指引、计划变更、需求评审、需求变更、主管审批 */
assert.match(template, /<select id="todosApprovalType"/, "待办工具栏必须包含审批场景下拉");
["charter", "buildguideline", "planchange", "review", "reviewchange", "reviewbymanager"].forEach((val) => {
  assert.match(template, new RegExp(`value="${val}"`), `审批场景必须包含 ${val}`);
});
console.log("PASS: todos template includes todosApprovalType with canonical scenarios");

/* 9. 动态筛选项显隐：默认全部待办下，所有对象特化下拉均隐藏 */
const defaultView = loadTodos("");
const defaultDoc = defaultView.document;
assert.equal(defaultDoc.getElementById("todosAction").hidden, true, "全部待办下办理场景必须隐藏");
assert.equal(defaultDoc.getElementById("todosStage").hidden, true, "全部待办下价值阶段必须隐藏");
assert.equal(defaultDoc.getElementById("todosResponsibility").hidden, true, "全部待办下办理责任必须隐藏");
assert.equal(defaultDoc.getElementById("todosApprovalType").hidden, true, "全部待办下审批场景必须隐藏");
console.log("PASS: default todos view hides all specialized dropdowns");

/* 10. 切换到各对象TAB时：
       - demand: 显示办理场景、价值阶段、办理责任，隐藏审批场景
       - story: 仅显示办理责任，隐藏办理场景、价值阶段、审批场景
       - approval: 仅显示审批场景，隐藏办理场景、价值阶段、办理责任
       - bug / risk / issue 等: 全部隐藏保持简洁 */
const dynamicFlow = loadTodos("");
dynamicFlow.requests[0].onDone({
  items: [],
  total: 10,
  summary: {},
  facets: [
    { key: "approval", label: "审批", count: 2 },
    { key: "demand", label: "业务需求", count: 3 },
    { key: "story", label: "研发需求", count: 2 },
    { key: "bug", label: "Bug", count: 3 }
  ]
});
const dChips = dynamicFlow.chips.querySelectorAll(".wb-done-tab");
const dynDoc = dynamicFlow.document;

// 点击 approval (index 1)
dChips[1].click();
assert.equal(dynDoc.getElementById("todosApprovalType").hidden, false, "审批TAB必须显示审批场景");
assert.equal(dynDoc.getElementById("todosAction").hidden, true, "审批TAB必须隐藏办理场景");
assert.equal(dynDoc.getElementById("todosStage").hidden, true, "审批TAB必须隐藏价值阶段");
assert.equal(dynDoc.getElementById("todosResponsibility").hidden, true, "审批TAB必须隐藏办理责任");

// 点击 demand (index 2)
dChips[2].click();
assert.equal(dynDoc.getElementById("todosAction").hidden, false, "业务需求TAB必须显示办理场景");
assert.equal(dynDoc.getElementById("todosStage").hidden, false, "业务需求TAB必须显示价值阶段");
assert.equal(dynDoc.getElementById("todosResponsibility").hidden, false, "业务需求TAB必须显示办理责任");
assert.equal(dynDoc.getElementById("todosApprovalType").hidden, true, "业务需求TAB必须隐藏审批场景");

// 点击 story (index 3)
dChips[3].click();
assert.equal(dynDoc.getElementById("todosResponsibility").hidden, false, "研发需求TAB必须显示办理责任");
assert.equal(dynDoc.getElementById("todosAction").hidden, true, "研发需求TAB必须隐藏办理场景保持简洁");
assert.equal(dynDoc.getElementById("todosStage").hidden, true, "研发需求TAB必须隐藏价值阶段保持简洁");
assert.equal(dynDoc.getElementById("todosApprovalType").hidden, true, "研发需求TAB必须隐藏审批场景");

// 点击 bug (index 4)
dChips[4].click();
assert.equal(dynDoc.getElementById("todosAction").hidden, true, "Bug TAB必须隐藏办理场景");
assert.equal(dynDoc.getElementById("todosStage").hidden, true, "Bug TAB必须隐藏价值阶段");
assert.equal(dynDoc.getElementById("todosResponsibility").hidden, true, "Bug TAB必须隐藏办理责任");
assert.equal(dynDoc.getElementById("todosApprovalType").hidden, true, "Bug TAB必须隐藏审批场景");
console.log("PASS: todos toolbar dropdowns dynamically toggle visibility per object type");
