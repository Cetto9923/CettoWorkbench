const assert = require("assert");
global.window = global;

const R = require("../../../web/static/js/po/demand-detail-render.js");

console.log("=== Running DemandDetailRender unit tests ===");

// 1. esc tests
assert.strictEqual(R.esc("<script>"), "&lt;script&gt;");
assert.strictEqual(R.esc(null), "");
console.log("PASS: esc sanitizes HTML input");

// 2. renderHeader test
const mockSummary = {
  id: "US63442",
  code: "US63442",
  demandId: 63442,
  title: "信贷影像采集合规改造",
  priority: "P1",
  valueStageLabel: "测试中",
  proposerName: "王敏",
  originator: "009871",
  proposerDept: "零售信贷部",
  ownerName: "程统",
  editedDate: "2026-08-28 14:32",
  category: "功能",
  source: "业务部门",
  product: "影像平台",
  poolName: "信贷需求池",
  bsa: "A 级",
  zentaoStatus: "已评审 / 待澄清"
};

const headerHtml = R.renderHeader(mockSummary, "childUnit");
assert.ok(headerHtml.includes("US63442"), "Header must include demand code");
assert.ok(headerHtml.includes("子业务需求 · 交付单元"), "Header must show child unit tag");
assert.ok(headerHtml.includes("已评审 / 待澄清"), "Header must show native zentao status");
console.log("PASS: renderHeader correctly renders header attributes");

// 3. renderRelationNav test
const mockRelCtx = {
  parent: { demandId: 63439, code: "US63439", title: "父需求标题" },
  siblings: [
    { demandId: 63441, code: "US63441", stage: "提测", isCurrent: false, hasRisk: false },
    { demandId: 63442, code: "US63442", stage: "测试", isCurrent: true, hasRisk: false },
    { demandId: 63443, code: "US63443", stage: "验收", isCurrent: false, hasRisk: true }
  ]
};

const navHtml = R.renderRelationNav(mockRelCtx, 63442);
assert.ok(navHtml.includes("US63439"), "Relation nav must include parent link");
assert.ok(navHtml.includes("dd-sibling-chip active"), "Current demand sibling chip must have active class");
assert.ok(navHtml.includes("dd-sibling-chip risk"), "Risk demand sibling chip must have risk class");
console.log("PASS: renderRelationNav renders parent and sibling states");

// 4. renderTabOverview & ValueStream deduplication test
const mockOverviewData = {
  summary: mockSummary,
  spotlight: {
    badge: "待你处理",
    title: "完成澄清",
    desc: "说明",
    actionLabel: "进入澄清办理区 →",
    targetTab: "requirement",
    targetSection: "clarificationSection"
  },
  valueStream: {
    estimatedCycleDays: 31,
    usedCycleDays: 5,
    targetCycleDays: 28,
    diffCycleDays: 3,
    isOverdue: true,
    stages: [
      { key: "accept", label: "受理", role: "PO", status: "done", durationText: "实际 1 天" },
      { key: "clarify", label: "澄清", role: "PO", status: "current", durationText: "已持续 4 天" },
      { key: "schedule", label: "排期", role: "协同", status: "future", durationText: "预计 3 天" }
    ]
  }
};

const overviewHtml = R.renderTabOverview(mockOverviewData);
assert.ok(overviewHtml.includes("进入澄清办理区 →"), "Spotlight CTA must be present");
assert.ok(overviewHtml.includes("预计交付周期"), "Value stream cycle bar must be present");
assert.ok(overviewHtml.includes("+3 天"), "Overdue diff must be formatted with plus sign");
console.log("PASS: renderTabOverview includes spotlight and deduplicated value stream");

// 5. renderParentAggregate test
const mockParentData = {
  unitTotal: 4,
  unitOnline: 1,
  risksCount: 1,
  attentionItems: [
    { demandId: 63443, code: "US63443", title: "风险子需求", riskDesc: "严重缺陷", isRisk: true }
  ],
  deliveryUnits: [
    { demandId: 63441, code: "US63441", title: "子需求1", stage: "提测", owner: "张三", storiesNum: 2, tasksNum: 5, launchDate: "2026-09-10" }
  ]
};

const parentHtml = R.renderParentAggregate(mockParentData);
assert.ok(parentHtml.includes("父需求聚合汇总"), "Parent aggregate must show summary badge");
assert.ok(parentHtml.includes("已拆分为 4 个独立交付单元"), "Must show unit total");
assert.ok(parentHtml.includes("US63443"), "Attention items must render");
console.log("PASS: renderParentAggregate renders aggregate cards and delivery units");
