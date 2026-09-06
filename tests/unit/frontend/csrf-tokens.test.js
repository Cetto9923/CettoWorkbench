const assert = require("assert");
const { getCsrfToken, scheduleFetch, isSessionExpired } = require("../../../web/static/js/schedule/schedulefetch.js");

console.log("=== Running CSRF token propagation tests (Phase X0) ===");

// 1. getCsrfToken without document
assert.strictEqual(getCsrfToken(), "", "getCsrfToken without document should return empty string");
console.log("PASS: getCsrfToken returns empty string when document is undefined");

// 2. Mock document with meta tag
let mockMetaContent = "csrf-token-abc-123";
global.document = {
  querySelector: function (selector) {
    if (selector === 'meta[name="csrf-token"]') {
      if (mockMetaContent === null) return null;
      return {
        getAttribute: function (attr) {
          if (attr === "content") return mockMetaContent;
          return null;
        },
      };
    }
    return null;
  },
};

assert.strictEqual(getCsrfToken(), "csrf-token-abc-123", "getCsrfToken should return token from meta");
console.log("PASS: getCsrfToken returns trimmed token from meta");

// 3. scheduleFetch attaches X-CSRF-Token and X-Requested-With when fallback to fetch
let lastFetchedUrl = null;
let lastFetchedOptions = null;
global.fetch = function (url, options) {
  lastFetchedUrl = url;
  lastFetchedOptions = options;
  return Promise.resolve({
    status: 200,
    url: url,
    redirected: false,
  });
};

scheduleFetch("/api/test/submit", { method: "POST" }).then(function (resp) {
  assert.strictEqual(lastFetchedUrl, "/api/test/submit");
  assert.ok(lastFetchedOptions.headers, "headers must exist");
  assert.strictEqual(
    lastFetchedOptions.headers["X-CSRF-Token"],
    "csrf-token-abc-123",
    "X-CSRF-Token must match meta token"
  );
  assert.strictEqual(
    lastFetchedOptions.headers["X-Requested-With"],
    "XMLHttpRequest",
    "X-Requested-With must be XMLHttpRequest"
  );
  console.log("PASS: scheduleFetch attaches X-CSRF-Token and X-Requested-With to fetch calls");

  // 4. scheduleFetch preserves existing X-CSRF-Token
  scheduleFetch("/api/test/custom", {
    method: "POST",
    headers: { "X-CSRF-Token": "custom-token-999" },
  }).then(function () {
    assert.strictEqual(
      lastFetchedOptions.headers["X-CSRF-Token"],
      "custom-token-999",
      "pre-existing X-CSRF-Token must not be overwritten"
    );
    console.log("PASS: scheduleFetch preserves existing custom X-CSRF-Token");

    // 5. scheduleFetch with empty meta token does NOT attach header
    mockMetaContent = "";
    lastFetchedOptions = null;
    scheduleFetch("/api/test/empty", { method: "POST" }).then(function () {
      assert.ok(
        !lastFetchedOptions.headers || !lastFetchedOptions.headers["X-CSRF-Token"],
        "empty meta token must not attach X-CSRF-Token header"
      );
      console.log("PASS: empty meta token does not attach X-CSRF-Token header");

      // Cleanup
      delete global.document;
      delete global.fetch;
      console.log("All CSRF token propagation tests passed!");
    });
  });
});
