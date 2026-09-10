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

const dataExportHeaderHtml = R.renderHeader(Object.assign({}, mockSummary, { category: "dataexport" }), "childUnit");
assert.ok(dataExportHeaderHtml.includes("数据导出"), "Header must localize the dataexport category");
console.log("PASS: renderHeader localizes ZenTao demand categories");

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

const scheduleOverviewHtml = R.renderTabOverview(Object.assign({}, mockOverviewData, {
  spotlight: {
    badge: "排期中",
    title: "版本窗口规划",
    desc: "说明",
    actionLabel: "排期",
    actionUrl: "/schedule/demands/63442/scheduling",
    targetTab: "overview",
    targetSection: "spotlightSection"
  }
}));
assert.ok(scheduleOverviewHtml.includes('href="/schedule/demands/63442/scheduling"'), "Schedule spotlight must use the shared schedule endpoint");
assert.ok(scheduleOverviewHtml.includes(">排期</a>"), "Schedule spotlight must render the action label");
console.log("PASS: schedule spotlight opens the scheduling flow rather than a detail tab");

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

// 6. renderTabRequirement & Clarification Action test
const mockReqData = {
  specHtml: "<p>业务需求说明</p>",
  verifyHtml: "<p>验收标准</p>",
  clarifications: [
    { productName: "信贷业务平台", analyst: "陆家瑜", content: "完成字段校验", devEnd: "2026-09-07", testEnd: "2026-09-12" }
  ],
  clarifyZtUrl: "http://zentao/index.php?m=demand&f=clarify&demandID=63442"
};

const reqHtml = R.renderTabRequirement(mockReqData);
assert.ok(reqHtml.includes("业务需求正文与描述"), "Requirement text must be present");
assert.ok(reqHtml.includes("信贷业务平台"), "Clarification product must be present");
assert.ok(reqHtml.includes("在禅道办理需求澄清 ↗"), "Clarification action button must be present");
assert.ok(reqHtml.includes("demandID=63442"), "Clarification URL must be present");

const reqHtmlWithId = R.renderTabRequirement(Object.assign({}, mockReqData, { demandId: 63442 }));
assert.ok(reqHtmlWithId.includes("js-drawer-clarify-btn"), "Workbench clarify modal button must be present");
assert.ok(reqHtmlWithId.includes('data-demand-id="63442"'), "Demand ID must be attached to clarify button");
console.log("PASS: renderTabRequirement includes spec, clarification table and action CTA");

// 7. renderTabExecution & Testing Progress & App Quality Tree test
const mockExecData = {
  stories: [
    { code: "ST72111", title: "活体检测前端校验", product: "移动端", owner: "张三", status: "开发中", tasksDone: 4, tasksTotal: 5, bugsTotal: 2, bugsActive: 1 }
  ],
  testOrders: [
    { code: "TO-14", title: "移动端活体SIT测试单", stage: "SIT", status: "doing", statusLabel: "进行中", owner: "李四", beginDate: "2026-09-01", endDate: "2026-09-10", ztUrl: "http://zentao/task-14" }
  ],
  testOrderSummary: { totalCount: 1, doingCount: 1, doneCount: 0, storiesCount: 1 },
  testCaseSummary: { totalCount: 120, executedCount: 82, unexecutedCount: 38, passedCount: 76, failedCount: 6, executionRate: 68.3, passRate: 92.7 },
  bugSummary: { totalCount: 14, activeCount: 5, resolvedCount: 9, deliveryBlocking: 1 },
  qualityOverview: { available: true, source: "scanner", appsCount: 1, branchesCount: 1, passedGates: 1, failedGates: 0 },
  appQualityTree: [
    {
      appName: "移动厅堂客户端",
      appCode: "mobile-hall-client",
      gatePassed: true,
      available: true,
      branchCount: 1,
      branches: [
        {
          storyCode: "ST72111",
          storyTitle: "活体检测前端校验",
          branchName: "feature/US10729-st72111",
          latestMr: "!402",
          gateStatus: "pass",
          available: true,
          score: 92.5,
          bugsCount: 0,
          codeSmells: 2,
          coverage: 82.0,
          scanTime: "2026-09-05 16:30",
          committer: "张三"
        }
      ]
    }
  ],
  leadsMatrix: { devLeads: ["张三"], testLead: "李四" }
};

const execHtml = R.renderTabExecution(mockExecData);
assert.ok(execHtml.includes("用例执行与测试进度"), "Must show test cases KPI card");
assert.ok(execHtml.includes("82"), "Must show executed cases");
assert.ok(execHtml.includes("68.3%"), "Must show execution rate");
assert.ok(execHtml.includes("92.7%"), "Must show pass rate");
assert.ok(execHtml.includes("1 阻塞交付"), "Must show delivery blocking bug badge");
assert.ok(execHtml.includes("移动端活体SIT测试单"), "Must show test orders table");
assert.ok(execHtml.includes("活体检测前端校验"), "Must show stories table");
assert.ok(execHtml.includes("80%"), "Must show story task completion pct");
assert.ok(execHtml.includes("1 未闭环"), "Must show story active bug status in stories table");
assert.ok(execHtml.includes("链路: 业务需求 → 研发需求 → 缺陷"), "Must show indirect bug aggregation chain note");
assert.ok(execHtml.includes("系统代码质量门禁"), "Must show system quality gate KPI card");
assert.ok(execHtml.includes("1 / 1 系统达标"), "Must show app gate pass status");
assert.ok(execHtml.includes("移动厅堂客户端"), "Must show app quality tree node");
assert.ok(execHtml.includes("feature/US10729-st72111"), "Must show branch name in quality tree");
assert.ok(execHtml.includes("!402"), "Must show latest MR in quality tree");
assert.ok(execHtml.includes("得分 <strong>92.5</strong>"), "Must show Sonar score in quality tree");
console.log("PASS: renderTabExecution renders testing KPI, dual progress bars, stories with bugs, and App Quality Tree");

// 7b. F03：未接入扫描源时不得把 0/空树渲染成「全部通过」
const unavailExec = Object.assign({}, mockExecData, {
  qualityOverview: { available: false, source: "none", appsCount: 0, branchesCount: 0, passedGates: 0, failedGates: 0 },
  appQualityTree: []
});
const unavailHtml = R.renderTabExecution(unavailExec);
assert.ok(unavailHtml.includes("代码质量未接入"), "Unavailable quality must show 未接入");
assert.ok(!unavailHtml.includes("1 / 1 系统达标"), "Unavailable quality must not claim gates passed");
console.log("PASS: renderTabExecution shows 未接入 when qualityOverview.available=false");
