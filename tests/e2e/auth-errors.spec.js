const assert = require("assert");
const { isSessionExpired } = require("../../web/static/js/schedule/schedulefetch.js");

console.log("=== Running auth error & session expiry tests (Phase H0) ===");

// 1. 401 Unauthorized -> must be identified as session expired
const resp401 = {
  status: 401,
  url: "https://example.com/api/schedule/windows",
  redirected: false,
  headers: new Map([["content-type", "application/json"]]),
};
assert.strictEqual(isSessionExpired(resp401), true, "401 status must expire session");
console.log("PASS: 401 status expires session");

// 2. 403 Forbidden with JSON -> must NOT be treated as session expired
const resp403JSON = {
  status: 403,
  url: "https://example.com/api/schedule/windows",
  redirected: false,
  headers: new Map([["content-type", "application/json"]]),
};
assert.strictEqual(isSessionExpired(resp403JSON), false, "403 JSON must NOT expire session");
console.log("PASS: 403 JSON does not expire session");

// 3. 403 Forbidden with HTML -> must NOT be treated as session expired (historical bug fix)
const resp403HTML = {
  status: 403,
  url: "https://example.com/schedule",
  redirected: false,
  headers: new Map([["content-type", "text/html; charset=utf-8"]]),
};
const initAcceptJSON = { headers: { Accept: "application/json" } };
assert.strictEqual(
  isSessionExpired(resp403HTML, initAcceptJSON),
  false,
  "403 HTML must NOT expire session even when Accept: application/json"
);
console.log("PASS: 403 HTML does not expire session even when client expected JSON");

// 4. 409 Conflict -> must NOT be treated as session expired
const resp409 = {
  status: 409,
  url: "https://example.com/api/schedule/windows/1",
  redirected: false,
  headers: new Map([["content-type", "application/json"]]),
};
assert.strictEqual(isSessionExpired(resp409), false, "409 Conflict must NOT expire session");
console.log("PASS: 409 Conflict does not expire session");

// 5. 500 Server Error -> must NOT be treated as session expired
const resp500 = {
  status: 500,
  url: "https://example.com/api/schedule/windows/1",
  redirected: false,
  headers: new Map([["content-type", "text/html"]]),
};
assert.strictEqual(isSessionExpired(resp500, initAcceptJSON), false, "500 Server Error must NOT expire session");
console.log("PASS: 500 Server Error does not expire session");

// 6. Redirect to /login -> must expire session
const respRedirectLogin = {
  status: 200,
  url: "https://example.com/login?redirect=%2Fschedule",
  redirected: true,
  headers: new Map([["content-type", "text/html"]]),
};
assert.strictEqual(isSessionExpired(respRedirectLogin), true, "Redirect to /login must expire session");
console.log("PASS: Redirect to /login expires session");

// 7. Normal 200 JSON -> must NOT expire session
const resp200 = {
  status: 200,
  url: "https://example.com/api/schedule/windows",
  redirected: false,
  headers: new Map([["content-type", "application/json"]]),
};
assert.strictEqual(isSessionExpired(resp200), false, "200 JSON must NOT expire session");
console.log("PASS: 200 JSON does not expire session");

console.log("All auth error and session expiry tests passed!");
