#!/usr/bin/env node
const assert = require("assert");
const fs = require("fs");
const vm = require("vm");

const code = fs.readFileSync("web/static/js/po/demand-detail-submit-review.js", "utf8");

function setupContext(candidatesData) {
  const elements = {};
  function makeEl(id, tagName = "div") {
    const el = {
      id,
      tagName,
      style: { display: "" },
      textContent: "",
      innerHTML: "",
      disabled: false,
      value: "",
      placeholder: "",
      querySelectorAll: () => [],
      addEventListener: () => {},
      setAttribute: () => {}
    };
    elements[id] = el;
    return el;
  }

  makeEl("ddSubmitReviewModal");
  makeEl("ddReviewCandidatesLoading");
  makeEl("ddReviewCandidatesError");
  makeEl("ddReviewCandidatesEmpty");
  makeEl("ddConfirmSubmitReviewBtn", "button");
  makeEl("ddReviewReviewerInput", "input");
  makeEl("ddReviewReviewerValue", "input");
  makeEl("ddReviewSelected");

  const documentStub = {
    getElementById: (id) => elements[id] || null,
    querySelectorAll: () => []
  };

  const windowStub = {
    DemandDetailReview: {},
    escapeHtml: (s) => s,
    initUserPicker: () => {},
    destroyUserPicker: () => {},
    clearAutocomplete: () => {},
    showToast: () => {},
    document: documentStub,
    appFetch: () => Promise.resolve({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ success: true, data: candidatesData })
    })
  };

  windowStub.window = windowStub;
  const sandbox = vm.createContext(windowStub);
  vm.runInContext(code, sandbox);

  return { window: windowStub, elements };
}

// Case 1: Empty candidates (需求池未配置业务评审人)
{
  const { window, elements } = setupContext({ demandId: 100, users: [], selected: [] });
  window.DemandDetailReviewSubmitReview.open(100);

  // Wait for promise tick
  setImmediate(() => {
    assert.strictEqual(elements.ddReviewCandidatesEmpty.hidden, false, "empty hint should be displayed");
    assert.strictEqual(elements.ddConfirmSubmitReviewBtn.disabled, true, "submit button should be disabled when empty");
    assert.strictEqual(elements.ddReviewReviewerInput.disabled, true, "input should be disabled when empty");

    // Case 2: Non-empty candidates (正常配置业务评审人)
    const ctx2 = setupContext({ demandId: 101, users: [{ value: "u1", label: "User 1" }], selected: [] });
    ctx2.window.DemandDetailReviewSubmitReview.open(101);

    setImmediate(() => {
      assert.strictEqual(ctx2.elements.ddReviewCandidatesEmpty.hidden, true, "empty hint should be hidden");
      assert.strictEqual(ctx2.elements.ddConfirmSubmitReviewBtn.disabled, false, "submit button should be enabled when users exist");
      assert.strictEqual(ctx2.elements.ddReviewReviewerInput.disabled, false, "input should be enabled when users exist");
      console.log("PASS: submit-review-empty-candidates (empty hint and button disable)");
    });
  });
}
