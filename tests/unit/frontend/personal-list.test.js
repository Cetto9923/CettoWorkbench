const assert = require("assert");
global.window = global;

const PersonalList = require("../../../web/static/js/po/personal-list.js");

console.log("=== Running PersonalList unit & behavioral tests ===");

// 1. escapeHtml tests
assert.strictEqual(PersonalList.escapeHtml("<script>alert(1)</script>"), "&lt;script&gt;alert(1)&lt;/script&gt;");
assert.strictEqual(PersonalList.escapeHtml("a & b \"c\" 'd'"), "a &amp; b &quot;c&quot; &#39;d&#39;");
assert.strictEqual(PersonalList.escapeHtml(null), "");
assert.strictEqual(PersonalList.escapeHtml(undefined), "");
console.log("PASS: escapeHtml escapes special characters securely");

// 2. Race condition / Out-of-order execution test
(function testRaceCondition() {
  let latestPayload = null;
  let summaryText = "";

  const mockOpts = {
    summaryEl: { set textContent(v) { summaryText = v; }, get textContent() { return summaryText; } },
    emptyEl: { hidden: true },
    errorEl: { hidden: true, querySelector: () => ({ textContent: "" }) },
    tbodyEl: { innerHTML: "" },
    onSuccess: (p) => { latestPayload = p; }
  };

  const controller = PersonalList.createController(mockOpts);

  // Simulate slow Request 1 and fast Request 2
  let resolveReq1, resolveReq2;
  const p1 = new Promise((res) => { resolveReq1 = res; });
  const p2 = new Promise((res) => { resolveReq2 = res; });

  global.window.fetch = (url) => {
    if (url === "/api/req1") {
      return p1.then(() => ({
        ok: true,
        status: 200,
        json: async () => ({ success: true, id: 1, data: "slow" })
      }));
    }
    if (url === "/api/req2") {
      return p2.then(() => ({
        ok: true,
        status: 200,
        json: async () => ({ success: true, id: 2, data: "fast" })
      }));
    }
    return Promise.reject(new Error("unexpected url"));
  };

  // Dispatch Req 1 then Req 2
  controller.fetch("/api/req1", {});
  controller.fetch("/api/req2", {});

  // Req 2 finishes first
  resolveReq2();
  setTimeout(() => {
    assert.strictEqual(latestPayload.id, 2, "Req 2 (fast) must update state first");

    // Req 1 finishes later
    resolveReq1();
    setTimeout(() => {
      assert.strictEqual(latestPayload.id, 2, "Req 1 (slow) must be ignored and NOT overwrite Req 2");
      console.log("PASS: Out-of-order race conditions are protected (stale responses discarded)");
    }, 20);
  }, 20);
})();

// 3. Error vs Empty state separation test
(function testErrorState() {
  let errorShown = false;
  let emptyShown = false;
  let errorMsg = "";

  const mockOpts = {
    summaryEl: { textContent: "" },
    emptyEl: { set hidden(v) { emptyShown = !v; } },
    errorEl: {
      set hidden(v) { errorShown = !v; },
      querySelector: () => ({ set textContent(v) { errorMsg = v; } })
    },
    tbodyEl: { innerHTML: "old-rows" },
    onError: (err) => {}
  };

  const controller = PersonalList.createController(mockOpts);

  global.window.fetch = () => Promise.resolve({
    ok: false,
    status: 500,
    json: async () => ({ success: false, error: "Internal Error" })
  });

  controller.fetch("/api/fail", {}, () => {
    assert.strictEqual(errorShown, true, "Error element must be visible on 500");
    assert.strictEqual(emptyShown, false, "Empty element must NOT be visible on error (E05 fix)");
    assert.strictEqual(mockOpts.tbodyEl.innerHTML, "", "Stale rows must be cleared on failure");
    console.log("PASS: Error state and empty state are strictly isolated");
  });
})();

// 4. Pagination rendering logic tests
(function testPagination() {
  const container = { hidden: false, innerHTML: "", querySelector: () => null, querySelectorAll: () => [] };

  // Case 4.1: total = 0 -> hidden
  PersonalList.renderPagination({ container, total: 0 });
  assert.strictEqual(container.hidden, true, "Pagination must be hidden when total is 0");
  console.log("PASS: Pagination is hidden when total = 0");

  // Case 4.2: total = 30, pageSize = 10 -> 3 pages
  PersonalList.renderPagination({ container, total: 30, pageSize: 10, page: 2 });
  assert.strictEqual(container.hidden, false);
  assert.ok(container.innerHTML.includes('data-page="1"'));
  assert.ok(container.innerHTML.includes('class="pager-btn active" data-page="2"'));
  assert.ok(container.innerHTML.includes('data-page="3"'));
  assert.ok(container.innerHTML.includes("显示 11–20，共 30 条"));
  console.log("PASS: Pagination renders exact pages and meta for small page counts");

  // Case 4.3: total = 100, pageSize = 10, page = 5 -> ellipsis on both sides
  PersonalList.renderPagination({ container, total: 100, pageSize: 10, page: 5 });
  assert.ok(container.innerHTML.includes('<span class="pager-ellipsis">…</span>'));
  assert.ok(container.innerHTML.includes('data-page="1"'));
  assert.ok(container.innerHTML.includes('data-page="10"'));
  console.log("PASS: Pagination renders ellipsis window for large page counts");
})();

// 5. Callback arity (1-arg vs 2-arg onDone) test
(function testCallbackArity() {
  global.window.fetch = () => Promise.resolve({
    ok: true,
    status: 200,
    json: async () => ({ success: true, items: [1, 2, 3], total: 3 })
  });

  // 1-arg callback: function(payload)
  const controller1 = PersonalList.createController({});
  controller1.fetch("/api/test-1", {}, function (payload) {
    assert.ok(payload !== null, "1-arg callback must receive payload");
    assert.strictEqual(payload.total, 3);
    console.log("PASS: 1-arg onDone(payload) correctly receives response payload");
  });

  // 2-arg callback: function(err, payload)
  const controller2 = PersonalList.createController({});
  controller2.fetch("/api/test-2", {}, function (err, payload) {
    assert.strictEqual(err, null, "2-arg callback err must be null on success");
    assert.ok(payload !== null, "2-arg callback must receive payload");
    assert.strictEqual(payload.total, 3);
    console.log("PASS: 2-arg onDone(err, payload) correctly receives err and payload");
  });
})();
