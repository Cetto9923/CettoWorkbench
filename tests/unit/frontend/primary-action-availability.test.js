const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const window = { escapeHtml: value => String(value) };
vm.runInNewContext(fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/primary-action.js"), "utf8"), { window });
const render = window.PrimaryAction.primaryActionHtml;
for (const key of ["approve", "withdraw_review", "submit_review", "schedule", "submit_test", "accept", "remind_accept", "deliver"]) {
  const item = { id: "US42", primaryAction: { key, label: key, kind: key === "schedule" ? "schedule" : "drawer", enabled: true, url: "/demands/42/action" } };
  const html = render(item, false, ["clarify"]);
  assert.ok(html.includes("当前待办页尚未接入此操作"), key);
  assert.ok(!html.includes("<button"), key);
  assert.equal(item.primaryAction.enabled, true, "must not mutate server descriptor");
  assert.ok(render(item, false).includes("<button"), "home action retained: " + key);
}
assert.ok(render({ primaryAction: { key: "clarify", label: "澄清", kind: "drawer", enabled: true } }, false, ["clarify"]).includes("<button"));
assert.ok(render({ primaryAction: { key: "evaluate", label: "评价", kind: "external", url: "https://zentao.test/", enabled: true } }, false, ["clarify"]).includes("<a "));
assert.ok(!render({ primaryAction: { key: "clarify", label: "澄清", kind: "drawer", enabled: false, reason: "没有权限" } }, false, ["clarify"]).includes("<button"));
console.log("PASS: page action availability preserves home, external links and server denials");
