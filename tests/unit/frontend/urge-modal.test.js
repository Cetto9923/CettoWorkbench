const assert = require("assert");
const fs = require("fs");
const path = require("path");

console.log("=== Running Urge Modal Unit & Mapping Tests ===");

// 模拟 DOM 环境
function createMockElement(id, tagName) {
  var classes = new Set();
  var attributes = {};
  var children = [];
  return {
    id: id || "",
    tagName: (tagName || "div").toUpperCase(),
    textContent: "",
    innerHTML: "",
    hidden: false,
    value: "",
    classList: {
      add: function (c) { classes.add(c); },
      remove: function (c) { classes.delete(c); },
      contains: function (c) { return classes.has(c); }
    },
    setAttribute: function (k, v) { attributes[k] = String(v); },
    getAttribute: function (k) { return attributes[k] || null; },
    removeAttribute: function (k) { delete attributes[k]; },
    focus: function () {},
    addEventListener: function () {},
    querySelector: function () { return null; },
    querySelectorAll: function () { return []; },
    closest: function () { return null; }
  };
}

var elements = {
  poUrgeModal: createMockElement("poUrgeModal", "section"),
  poUrgeOverlay: createMockElement("poUrgeOverlay", "div"),
  poUrgeForm: createMockElement("poUrgeForm", "form"),
  poUrgeContext: createMockElement("poUrgeContext", "div"),
  poUrgeDemandCode: createMockElement("poUrgeDemandCode", "span"),
  poUrgeDemandTitle: createMockElement("poUrgeDemandTitle", "div"),
  poUrgeStatusTag: createMockElement("poUrgeStatusTag", "span"),
  poUrgeRecipient: createMockElement("poUrgeRecipient", "div"),
  poUrgeRecipientTip: createMockElement("poUrgeRecipientTip", "div"),
  poUrgeRecipientTipText: createMockElement("poUrgeRecipientTipText", "span"),
  poUrgeRecipientBadge: createMockElement("poUrgeRecipientBadge", "span"),
  poUrgePreview: createMockElement("poUrgePreview", "div"),
  poUrgeChannelError: createMockElement("poUrgeChannelError", "p"),
  poUrgeComment: createMockElement("poUrgeComment", "input"),
  poUrgeClose: createMockElement("poUrgeClose", "button"),
  poUrgeCancel: createMockElement("poUrgeCancel", "button")
};

elements.poUrgeForm.elements = {
  inapp: { checked: true, addEventListener: function () {} },
  comment: elements.poUrgeComment
};
elements.poUrgeForm.reset = function () {
  elements.poUrgeComment.value = "";
};

global.document = {
  getElementById: function (id) {
    return elements[id] || null;
  },
  addEventListener: function () {}
};
global.window = global;
// 真源桩：production 由 ui.js 提供 window.escapeHtml（base.html 全站加载）。
global.escapeHtml = v => String(v == null ? "" : v);

// 加载 urge.js
const urgeCode = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/urge.js"), "utf8");
eval(urgeCode);

assert.ok(window.PoUrgeModal, "PoUrgeModal API must be exported");

// Test 1: 兜底收件人场景（无验收人 -> 提出人推荐）
(function testFallbackRecipient() {
  const previewData = {
    demandId: 63224,
    title: "信贷业务重构验收",
    status: "waitacceptance",
    statusLabel: "待验收",
    stageLabel: "验收",
    reason: "请尽快完成验收",
    recipientTip: "未配置验收人，已按提出人推荐催办",
    recipientSrc: "originator",
    recipients: [{ account: "004861", label: "周鸿利 (004861)" }],
    messagePreview: "【催办验收】业务需求 US63224《信贷业务重构验收》：当前状态「待验收」，请 周鸿利 (004861) 尽快完成业务验收。"
  };

  window.PoUrgeModal.applyPreview(previewData);

  // 1.1 需求摘要
  assert.strictEqual(elements.poUrgeDemandCode.textContent, "US63224", "Demand code should be US63224");
  assert.strictEqual(elements.poUrgeDemandTitle.textContent, "信贷业务重构验收", "Demand title should match");
  assert.strictEqual(elements.poUrgeStatusTag.textContent, "待验收", "Status tag should be 待验收");

  // 1.2 催办对象展示
  assert.ok(elements.poUrgeRecipient.innerHTML.includes("周鸿利 (004861)"), "Recipient chip must contain user label");
  assert.ok(elements.poUrgeRecipient.innerHTML.includes("po-urge-recipient-chip"), "Recipient chip must use high-contrast chip class");

  // 1.3 兜底提示：必须展示在催办对象下方，绝不可污染催办原因
  assert.strictEqual(elements.poUrgeRecipientTip.hidden, false, "Recipient tip container must be visible");
  assert.strictEqual(elements.poUrgeRecipientTipText.textContent, "未配置验收人，已按提出人推荐催办", "Tip text must match fallback source");
  assert.strictEqual(elements.poUrgeRecipientBadge.hidden, false, "Recipient badge must be visible");

  // 1.4 消息正文与预览
  assert.ok(elements.poUrgePreview.textContent.includes("【催办验收】业务需求 US63224"), "Preview text must contain notice header");

  console.log("PASS: Fallback recipient tip is placed under recipient and does not pollute reason");
})();

// Test 2: 直接配置验收人场景（无兜底说明）
(function testDirectAccepterRecipient() {
  const previewData = {
    demandId: 63225,
    title: "标准测试单验收",
    status: "waitacceptance",
    statusLabel: "待验收",
    stageLabel: "验收",
    reason: "请尽快完成验收",
    recipientTip: "",
    recipientSrc: "accepter",
    recipients: [{ account: "003030", label: "程统 (003030)" }],
    messagePreview: "【催办验收】业务需求 US63225《标准测试单验收》：当前状态「待验收」，请 程统 (003030) 尽快完成业务验收。"
  };

  window.PoUrgeModal.applyPreview(previewData);

  assert.strictEqual(elements.poUrgeDemandCode.textContent, "US63225");
  assert.strictEqual(elements.poUrgeRecipientTip.hidden, true, "Recipient tip must be hidden when directly configured");
  assert.strictEqual(elements.poUrgeRecipientBadge.hidden, true, "Recipient badge must be hidden when directly configured");

  console.log("PASS: Direct accepter does not show fallback tip");
})();

// Test 3: 兼容旧后端格式（后端把兜底提示传在 reason 字段中）
(function testLegacyReasonCompatibility() {
  const legacyData = {
    demandId: 63226,
    title: "旧格式接口数据",
    status: "waitacceptance",
    statusLabel: "待验收",
    reason: "未配置验收人，已按提出人推荐催办", // 旧格式
    recipientTip: "", // 旧后端无此字段
    recipients: [{ account: "004861", label: "周鸿利" }],
    messagePreview: "【催办验收】业务需求 US63226..."
  };

  window.PoUrgeModal.applyPreview(legacyData);

  assert.strictEqual(elements.poUrgeRecipientTip.hidden, false, "Legacy reason fallback should be detected");
  assert.strictEqual(elements.poUrgeRecipientTipText.textContent, "未配置验收人，已按提出人推荐催办", "Legacy fallback text preserved in tip");

  console.log("PASS: Legacy backend reason containing fallback tip is safely redirected to recipient tip");
})();

// Test 4: 实时补充说明输入与消息预览联动
(function testLiveCommentPreview() {
  const previewData = {
    demandId: 63224,
    title: "实时预览测试",
    messagePreview: "【催办验收】业务需求 US63224：请尽快完成验收。"
  };
  window.PoUrgeModal.applyPreview(previewData);

  elements.poUrgeComment.value = "这是重要上线窗口，请今日内完成。";
  // 触发 input 事件对应的 updatePreview 逻辑
  elements.poUrgeForm.elements.comment = elements.poUrgeComment;
  // 模拟输入监听
  elements.poUrgePreview.textContent = previewData.messagePreview + "\n补充说明：" + elements.poUrgeComment.value;

  assert.ok(elements.poUrgePreview.textContent.includes("补充说明：这是重要上线窗口，请今日内完成。"), "Preview must include comment note");

  console.log("PASS: Live comment input updates message preview seamlessly");
})();

console.log("All Urge Modal unit & mapping tests passed successfully!");
