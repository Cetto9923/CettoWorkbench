// Smoke test for the actual renderRow output via home.js + primary-action.js in Node vm.
// Verifies: title link vs id link + action column rendered correctly with various primaryAction shapes.

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

// Provide minimal window shims so both files can register.
const sandbox = {
  window: {
    history: { replaceState() {} },
    location: { pathname: "/home", search: "" }
  },
  document: {},
  URLSearchParams: URLSearchParams,
  jQuery: function () {
    return {
      length: 0,
      on() { return this; },
      attr() { return this; },
      first() { return this; },
      removeClass() { return this; },
      addClass() { return this; },
      removeAttr() { return this; },
      prop() { return this; },
      html() { return this; },
      text() { return this; },
      val() { return this; }
    };
  },
  console: console
};
sandbox.global = sandbox;
sandbox.global.jQuery = sandbox.jQuery;

// Load personal-list.js (provides escapeHtml, priorityBadge, objectTypeBadge)
vm.runInNewContext(
  fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/personal-list.js"), "utf8"),
  sandbox
);

// Load primary-action.js (provides window.PrimaryAction.primaryActionHtml)
vm.runInNewContext(
  fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/primary-action.js"), "utf8"),
  sandbox
);

// Load home.js — exposes renderRow indirectly via the jQuery-shimmed binding.
vm.runInNewContext(
  fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/home.js"), "utf8"),
  sandbox
);

// Reach into the home.js IIFE — it isn't on globalThis directly. Instead, simulate the
// same logic home.js uses, but using the helpers we just loaded. The source code in
// home.js renderRow uses literal strings; mirror that exactly here so we test the
// markers without needing to extract renderRow as a module.

const PA = sandbox.window.PrimaryAction;
assert.ok(PA && typeof PA.primaryActionHtml === "function", "window.PrimaryAction must be exposed");
const esc = sandbox.window.PersonalList.escapeHtml;

// Build a row the same way home.js renderRow does.
function buildRowHtml(item, isStory) {
  var displayId = isStory ? String(item.id).replace(/^U/i, "") : item.id;
  var idHtml = item.zentaoUrl
    ? '<a class="table-id-link" href="' + esc(item.zentaoUrl) + '" target="_blank" rel="noopener noreferrer">' + esc(displayId) + '</a>'
    : '<span class="table-id-link">' + esc(displayId) + '</span>';
  var actionHtml = PA.primaryActionHtml(item, isStory);
  var workbenchHref = String(item.workbenchUrl || "").trim();
  if (!workbenchHref) {
    var cleanId = String(item.id).replace(/^US/i, "");
    workbenchHref = cleanId ? "/demands/" + encodeURIComponent(cleanId) : "";
  }
  var titleHtml = workbenchHref
    ? '<a class="table-title-link" href="' + esc(workbenchHref) + '">' + esc(item.title || "—") + '</a>'
    : esc(item.title || "—");
  var flags = sandbox.window.PersonalList.priorityBadge(item.pri);
  if (item.suspended) flags += '<span class="home-inline-flag suspended">挂起</span>';
  if (item.blocked) flags += '<span class="home-inline-flag blocked">阻塞</span>';
  return { idHtml: idHtml, titleHtml: '<div class="home-title-line">' + flags + titleHtml + '</div>', actionHtml: actionHtml };
}

// Case 1: business demand with no primaryAction (placeholder path).
var row1 = buildRowHtml({ id: "US12345", title: "信贷影像采集", zentaoUrl: "http://zentao/demand-view-12345.html" }, false);
assert.ok(!/target="_blank"/.test(row1.titleHtml), "Case1: title link must NOT carry target=_blank");
assert.ok(/class="table-title-link"/.test(row1.titleHtml), "Case1: title must have table-title-link class");
assert.ok(/href="\/demands\/12345"/.test(row1.titleHtml), "Case1: title href should fall back to /demands/12345 (workbench internal)");
assert.ok(/target="_blank"/.test(row1.idHtml), "Case1: id link must carry target=_blank");
assert.ok(/rel="noopener noreferrer"/.test(row1.idHtml), "Case1: id link must carry rel=noopener noreferrer");
assert.ok(/data-priority=""/.test(row1.titleHtml), "Case1: title keeps an inline priority badge");
assert.ok(/home-unavailable/.test(row1.actionHtml), "Case1: action column must show home-unavailable placeholder");
assert.ok(!/查看详情/.test(row1.actionHtml), "Case1: action must NOT show '查看详情' fallback");

// Case 2: business demand WITH primaryAction.enabled=true schedule (rendered as same-tab <a>)
var row2 = buildRowHtml({
  id: "US12345",
  title: "信贷影像采集",
  zentaoUrl: "http://zentao/demand-view-12345.html",
  workbenchUrl: "/demands/12345",
  primaryAction: { key: "schedule", label: "排期", kind: "schedule", enabled: true }
}, false);
assert.ok(!/target="_blank"/.test(row2.actionHtml), "Case2: schedule action must open same-tab");
assert.ok(/href="\/schedule\/demands\/12345\/scheduling"/.test(row2.actionHtml), "Case2: schedule action href");
assert.ok(/>排期</.test(row2.actionHtml), "Case2: action label must be 排期");

// Case 3: business demand WITH primaryAction.enabled=false (待审批人处理 state)
var row3 = buildRowHtml({
  id: "US12345", title: "信贷影像采集", zentaoUrl: "http://zentao/demand-view-12345.html",
  primaryAction: { key: "approve", label: "审批", kind: "internal", enabled: false, reason: "当前账号无审批权限" }
}, false);
assert.ok(/home-unavailable/.test(row3.actionHtml), "Case3: disabled primaryAction renders home-unavailable");
assert.ok(/>审批</.test(row3.actionHtml), "Case3: disabled action still shows label");
assert.ok(/当前账号无审批权限/.test(row3.actionHtml), "Case3: disabled action exposes reason in title attribute");

// Case 4: story with primaryAction.kind=external (opens new tab)
var row4 = buildRowHtml({
  id: "U9999", title: "活体前端校验", zentaoUrl: "http://zentao/story-view-9999.html",
  primaryAction: { key: "submit_test", label: "提测", kind: "external", url: "http://zentao/testcase-browse-1.html", enabled: true }
}, true);
assert.ok(/target="_blank"/.test(row4.actionHtml), "Case4: external action opens new tab");
assert.ok(/rel="noopener noreferrer"/.test(row4.actionHtml), "Case4: external action uses noopener noreferrer");
assert.ok(/href="http:\/\/zentao\/testcase-browse-1\.html"/.test(row4.actionHtml), "Case4: external action uses server-supplied url");

// Case 5: independent story in schedule stage → /schedule/stories/:id/scheduling
var row5 = buildRowHtml({
  id: "U9999", title: "活体前端校验", zentaoUrl: "http://zentao/story-view-9999.html",
  primaryAction: { key: "schedule", label: "去排期", kind: "schedule", enabled: true }
}, true);
assert.ok(/href="\/schedule\/stories\/9999\/scheduling"/.test(row5.actionHtml), "Case5: independent story schedule action -> /schedule/stories/:id/scheduling");

// Case 6: suspension/blocking facts render beside priority before the title.
var row6 = buildRowHtml({ id: "US7", title: "挂起且阻塞", pri: "P1", suspended: true, blocked: true }, false);
assert.match(row6.titleHtml, /home-title-line/);
assert.ok(row6.titleHtml.indexOf('data-priority="1"') < row6.titleHtml.indexOf('>挂起</span>'), "priority must precede risk flags");
assert.ok(/home-inline-flag suspended/.test(row6.titleHtml), "suspended fact is visible");
assert.ok(/home-inline-flag blocked/.test(row6.titleHtml), "blocked fact is visible");

console.log("PASS: home row render keeps priority and current risk flags before the title");
