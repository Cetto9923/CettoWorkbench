const assert = require("assert");
global.window = global;

const Review = require("../../../web/static/js/po/demand-detail-review.js");

console.log("=== Running DemandDetailReview unit tests ===");

const bothActionsHtml = Review.renderReviewView({
  summary: {
    id: "US63425",
    demandId: 63425,
    status: "wait",
    canReview: true,
    canWithdrawReview: true,
    isCreator: true,
    category: "feature",
    source: "—",
    priority: "P3",
    product: "—",
    poolName: "科技工具链建设需求池",
    proposerName: "程统(003030)",
    proposerDept: "组织变革团队",
    ownerName: "王一帆(771277)",
    reviewer: "程统(003030)",
    createdName: "程统(003030)",
    currentOwner: "崔梅(002940)"
  },
  primaryAction: { key: "approve", enabled: true },
  requirement: {
    specHtml: "<p>业务需求说明</p>",
    verifyHtml: "<p>验收标准</p>"
  }
});

assert.ok(bothActionsHtml.includes('id="ddPassBtn"'), "pass button must render when current user can review");
assert.ok(bothActionsHtml.includes('id="ddRejectBtn"'), "reject button must render when current user can review");
assert.ok(bothActionsHtml.includes('id="ddRejectModal"'), "reject modal overlay must render");
assert.ok(bothActionsHtml.includes('id="ddWithdrawBtn"'), "withdraw button must also render when current user can withdraw");
assert.ok(bothActionsHtml.includes("撤销评审"), "withdraw button label must be 撤销评审");
assert.ok(bothActionsHtml.includes('id="ddWithdrawModal"'), "withdraw modal overlay must render");
assert.ok(typeof Review.openWithdrawModal === "function", "openWithdrawModal must be exported");
assert.ok(typeof Review.closeWithdrawModal === "function", "closeWithdrawModal must be exported");
assert.ok(typeof Review.confirmWithdraw === "function", "confirmWithdraw must be exported");
assert.ok(bothActionsHtml.includes("也是创建人"), "banner must explain the combined reviewer and creator state");
console.log("PASS: reviewer+creator sees both review decision buttons and withdraw action");

const reviewerOnlyHtml = Review.renderReviewView({
  summary: { id: "US63426", demandId: 63426, status: "wait", canReview: true, canWithdrawReview: false },
  primaryAction: { key: "approve", enabled: true },
  requirement: {}
});
assert.ok(reviewerOnlyHtml.includes('id="ddPassBtn"'), "reviewer-only demand must keep pass action");
assert.ok(reviewerOnlyHtml.includes('id="ddRejectBtn"'), "reviewer-only demand must keep reject action");
assert.ok(!reviewerOnlyHtml.includes('id="ddWithdrawBtn"'), "reviewer-only demand must not show withdraw action");
console.log("PASS: reviewer-only state remains review-only");

const canEditHtml = Review.renderReviewView({
  summary: {
    id: "US63427", demandId: 63427, status: "wait",
    isCreator: true, canWithdrawReview: true, canEdit: true,
    zentaoEditUrl: "/demand-edit-63427.html"
  },
  primaryAction: { key: "withdraw_review", enabled: true },
  requirement: {}
});
assert.ok(canEditHtml.includes("编辑需求 ↗"), "creator can edit when nobody reviewed yet");
assert.ok(canEditHtml.includes("/demand-edit-63427.html"), "edit link must point to zentao edit url");
console.log("PASS: creator can edit demand before any reviewer submitted review");

const lockedEditHtml = Review.renderReviewView({
  summary: {
    id: "US63428", demandId: 63428, status: "wait",
    isCreator: true, canWithdrawReview: true, canEdit: false,
    hasReviewed: true, reviewedCount: 1,
    editDisabledReason: "已有评审人出具评审意见，需求已锁定修改；如需修改请先撤回评审申请"
  },
  primaryAction: { key: "withdraw_review", enabled: true },
  requirement: {}
});
assert.ok(lockedEditHtml.includes("编辑已锁定 ↗"), "edit button is disabled when someone reviewed");
assert.ok(lockedEditHtml.includes("已有评审人出具评审意见"), "shows lockout tooltip reason");
console.log("PASS: demand editing is locked once someone has reviewed");
