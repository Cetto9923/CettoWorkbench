const assert = require("assert");
const fs = require("fs");
const path = require("path");

const root = path.join(__dirname, "../../..");

// 1. po_deliver_modal.html: 结构与元素验证
const modalTplPath = path.join(root, "web/templates/components/po_deliver_modal.html");
assert.ok(fs.existsSync(modalTplPath), "po_deliver_modal.html must exist");
const modalTpl = fs.readFileSync(modalTplPath, "utf8");

assert.match(modalTpl, /id="poDeliverModalOverlay"/, "Modal template must have overlay");
assert.match(modalTpl, /id="poDeliverModal"/, "Modal template must have modal container");
assert.match(modalTpl, /id="poDeliverChecklist"/, "Modal template must have checklist container");
assert.match(modalTpl, /id="poLaunchWindowSelect"/, "Modal template must have launch window select");
assert.match(modalTpl, /id="poLaunchDate"/, "Modal template must have launch date input");
assert.match(modalTpl, /name="poDeliverCarReview"/, "Modal template must have car review radio");
assert.match(modalTpl, /id="poDeliverGrayPlan"/, "Modal template must have gray plan select");
assert.match(modalTpl, /id="poDeliverVerifyDate"/, "Modal template must have verify date select");
assert.match(modalTpl, /id="poDeliverVerifyPlan"/, "Modal template must have verify plan textarea");
assert.match(modalTpl, /id="poDeliverVerifierInput"/, "Modal template must have verifier input");
console.log("PASS: po_deliver_modal.html template structure verified");

// 2. 样式文件与独立命名空间
const cssPath = path.join(root, "web/static/css/po/po-deliver.css");
assert.ok(fs.existsSync(cssPath), "po-deliver.css must exist");
const css = fs.readFileSync(cssPath, "utf8");
assert.match(css, /\.po-deliver-modal-overlay/, "po-deliver.css must style overlay");
assert.match(css, /\.po-deliver-modal/, "po-deliver.css must style modal dialog");
assert.match(css, /\.po-deliver-checklist/, "po-deliver.css must style checklist");
console.log("PASS: po-deliver.css styling and scoping verified");

// 3. 页面挂载检查（home.html, todos.html）
const pages = [
  { name: "home.html", file: "web/templates/po/home.html" },
  { name: "todos.html", file: "web/templates/po/todos.html" }
];
pages.forEach(({ name, file }) => {
  const content = fs.readFileSync(path.join(root, file), "utf8");
  assert.match(content, /static\/css\/po\/po-deliver\.css/, `${name} must mount po-deliver.css`);
  assert.match(content, /template\s+"po\/deliver_modal"/, `${name} must mount po/deliver_modal template`);
  assert.match(content, /static\/js\/po\/po-deliver\.js/, `${name} must mount po-deliver.js script`);
  console.log(`PASS: ${name} mounts deliver CSS, modal template, and JS script`);
});

// 4. homereview.js 中 deliver 动作委托
const homereviewPath = path.join(root, "web/static/js/po/homereview.js");
const homereviewContent = fs.readFileSync(homereviewPath, "utf8");
assert.match(
  homereviewContent,
  /\.js-po-drawer-action\[data-action-key=['"]deliver['"]\]/,
  "homereview.js must bind deliver action click"
);
assert.match(
  homereviewContent,
  /window\.openPoDeliverModal/,
  "homereview.js must invoke window.openPoDeliverModal"
);
console.log("PASS: homereview.js deliver action routing verified");

// 5. demand-detail.js 中交付按钮委托与 refresh 暴露
const ddPath = path.join(root, "web/static/js/po/demand-detail.js");
const ddContent = fs.readFileSync(ddPath, "utf8");
assert.match(
  ddContent,
  /\.js-drawer-deliver-btn/,
  "demand-detail.js must handle .js-drawer-deliver-btn click"
);
assert.match(
  ddContent,
  /refresh:\s*function/,
  "demand-detail.js must expose refresh method"
);
console.log("PASS: demand-detail.js deliver integration verified");

// 6. po-deliver.js 中的辅助计算与验证函数逻辑测试
const jsPath = path.join(root, "web/static/js/po/po-deliver.js");
const jsContent = fs.readFileSync(jsPath, "utf8");

// Mock window and document to evaluate helpers
// 真源桩：production 由 ui.js 提供 window.escapeHtml（base.html 全站加载）。
const mockWindow = { escapeHtml: v => String(v == null ? "" : v) };
const mockDoc = {};
const evalContext = new Function("window", "document", "jQuery", "$", jsContent);
try {
  evalContext(mockWindow, mockDoc, () => {}, () => ({ on: () => {} }));
} catch (e) {
  // DOM element lookup during IIFE evaluation is mocked or ignored
}

const helpers = mockWindow.__poDeliverLaunchHelpers;
assert.ok(helpers, "po-deliver.js must export __poDeliverLaunchHelpers");

// demandIdOf
assert.strictEqual(helpers.demandIdOf("US123"), "123");
assert.strictEqual(helpers.demandIdOf("123"), "123");
assert.strictEqual(helpers.demandIdOf("REQ-456"), "456");

// buildLaunchWindowOptions
const windows = [
  { id: 101, name: "2026-09-15 窗口", releaseDate: "2026-09-15" },
  { id: 102, name: "2026-09-22 窗口", releaseDate: "2026-09-22" }
];
const optionsHtml = helpers.buildLaunchWindowOptions(windows, "101");
assert.match(optionsHtml, /value="101" selected/, "Window 101 should be selected");
assert.match(optionsHtml, /value="__launch_other__"/, "Options must include __launch_other__");

// findWindowByReleaseDate
const found = helpers.findWindowByReleaseDate(windows, "2026-09-22");
assert.ok(found, "Should find window by date");
assert.strictEqual(found.id, 102);

// resolveLaunchNotice
const noticeExisting = helpers.resolveLaunchNotice({
  launchMode: "other",
  deliverDate: "2026-09-15",
  scheduling: { windows }
});
assert.strictEqual(noticeExisting.visible, true);
assert.match(noticeExisting.text, /该日期已有/);

const noticeNew = helpers.resolveLaunchNotice({
  launchMode: "other",
  deliverDate: "2026-10-01",
  scheduling: { windows }
});
assert.strictEqual(noticeNew.visible, true);
assert.match(noticeNew.text, /自动创建/);

// validateFormData
let toastMsg = "";
mockWindow.showToast = (msg) => { toastMsg = msg; };

// blocked by precheck
const precheckBlocked = helpers.validateFormData({}, { precheck: { canSubmit: false, blockReason: "验收未通过" } });
assert.strictEqual(precheckBlocked, false);
assert.strictEqual(toastMsg, "验收未通过");

// valid form data
const validData = {
  windowId: 101,
  deliverDate: "2099-01-01",
  isCarReview: "0",
  isGrayVerifyPlan: "0",
  verifyDate: "1",
  verifyPlan: "生产验证步骤",
  verifier: "zhangsan"
};
const validRes = helpers.validateFormData(validData, { launchMode: "window", precheck: { canSubmit: true } });
assert.strictEqual(validRes, true);

console.log("PASS: po-deliver.js helper logic and validation verified");
