const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");

// =============================================================================
// Stage 4 — 标题 vs ID 导航行为 + primaryAction 占位渲染测试
//
// 目标：
//   1. 标题 <a class="table-title-link"> 不带 target="_blank"，href 是工作台内部 URL；
//   2. ID <a class="table-id-link"> 带 target="_blank" rel="noopener noreferrer"，href 是禅道 URL；
//   3. 缺少 primaryAction 时不显示 "查看详情" 兜底，按 PLAN §4 显示 "—" 占位；
//   4. demand-detail.js 全局委托不再吞掉 <a class="table-title-link"> 的点击；
//   5. workboard.js 排期阶段按钮直链 /schedule/{demands|stories}/:id/scheduling；
//   6. schedule-link.js 与 primary-action.js 暴露 window API。
// =============================================================================

// -----------------------------------------------------------------------------
// 1. 解析 home.js 的 renderRow 源码：验证标题/ID/action 的契约标记
// -----------------------------------------------------------------------------
const homeSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/home.js"), "utf8");

// 标题不应携带 target="_blank"（PLAN §4：标题进工作台详情，当前页打开）
assert.ok(
  !/table-title-link[^>]*target="_blank"/.test(homeSrc),
  "home.js must not render title-link with target=_blank (same-tab navigation required)"
);
// 标题应是 <a class="table-title-link" href="..."> 当前页
assert.ok(
  /<a class="table-title-link" href="/.test(homeSrc),
  "home.js title-link must use href= for same-tab navigation"
);

// ID 必须携带 target="_blank" + rel="noopener noreferrer"
assert.ok(
  /table-id-link[^>]*target="_blank"[^>]*rel="noopener noreferrer"/.test(homeSrc),
  "home.js id-link must open in new tab with noopener noreferrer"
);
assert.ok(
  /table-id-link[^>]*rel="noopener noreferrer"[^>]*target="_blank"|table-id-link[^>]*target="_blank"[^>]*rel="noopener noreferrer"/.test(homeSrc),
  "home.js id-link must include both target=_blank and rel=noopener noreferrer"
);
console.log("PASS: home.js title-link stays in current tab, id-link opens new tab with noopener");

// 标题链接由统一详情抽屉接管；ID 链接仍保留禅道原文导航。
assert.ok(
  /workbenchHref/.test(homeSrc) && homeSrc.indexOf('"/demands/" + encodeURIComponent(cleanId)') >= 0,
  "home.js title-link must resolve to the workbench demand detail route"
);
console.log("PASS: home.js title-link targets the unified detail drawer route");

// primaryAction 占位：缺失字段时显示 "—" 而非"查看详情"（PLAN §4 兜底禁止）
assert.ok(
  /home-unavailable[^>]*>[^<]*—/.test(homeSrc) ||
  /home-unavailable[^>]*title="[^"]*"/.test(homeSrc),
  "home.js must render home-unavailable placeholder for missing primaryAction"
);
// 不允许 "查看详情" 兜底（PLAN §4 禁止；剥离注释后只检查代码体）
const homeCodeOnly = stripJsComments(homeSrc);
assert.ok(
  !/primaryAction[\s\S]*?查看详情/.test(homeCodeOnly),
  "home.js must NOT fall back to '查看详情' when primaryAction is missing (PLAN §4 forbids)"
);
console.log("PASS: home.js primaryAction placeholder is em-dash, never '查看详情' fallback");

// -----------------------------------------------------------------------------
// 2. demand-detail.js 全局委托：仅接管 /demands/:id 详情直链。
// -----------------------------------------------------------------------------
const ddSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/demand-detail.js"), "utf8");
// 委托逻辑必须识别 A 元素的详情 href，并保留其它链接。
assert.ok(
  /tag\s*===\s*"A"/.test(ddSrc) || /A"\)/.test(ddSrc),
  "demand-detail.js click delegation must special-case A elements"
);
assert.ok(
  /trigger\.getAttribute\("href"\)/.test(ddSrc) || /href/.test(ddSrc),
  "demand-detail.js must inspect href attribute on A triggers"
);
assert.ok(
  /a\[href\^='\/demands\/'\]/.test(ddSrc) && /detailMatch/.test(ddSrc),
  "demand-detail.js must intercept only /demands/:id detail links"
);
console.log("PASS: demand-detail.js intercepts the unified detail route and preserves other links");

// -----------------------------------------------------------------------------
// 3. workboard.js 排期阶段直链：stage-action 应是 <a href="/schedule/{demands|stories}/:id/scheduling">
// -----------------------------------------------------------------------------
const wbSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/workboard.js"), "utf8");
assert.ok(
  /window\.ScheduleLink\.scheduleStoryAction\(/.test(wbSrc),
  "workboard.js must call ScheduleLink.scheduleStoryAction for stories in schedule stage"
);
assert.ok(
  /window\.ScheduleLink\.scheduleDemandAction\(/.test(wbSrc),
  "workboard.js must call ScheduleLink.scheduleDemandAction for demand rows in schedule stage"
);
// 禁止在 workboard.js 内硬编码 "/schedule/" 路径
assert.ok(
  !/"\/schedule\/demands\/"|"'\/schedule\/demands\/"|"\/schedule\/stories\/"|"'\/schedule\/stories\/'/.test(wbSrc),
  "workboard.js must NOT hardcode /schedule/ paths; route via ScheduleLink helper"
);
console.log("PASS: workboard.js uses ScheduleLink for schedule stage navigation (no hardcoded paths)");

// -----------------------------------------------------------------------------
// 4. schedule-link.js / primary-action.js 必须暴露 window API
// -----------------------------------------------------------------------------
const schedSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/schedule-link.js"), "utf8");
assert.ok(/window\.ScheduleLink\s*=/.test(schedSrc), "schedule-link.js must export window.ScheduleLink");
assert.ok(/url\s*:\s*url/.test(schedSrc), "ScheduleLink.url helper must exist");
assert.ok(/scheduleStoryAction/.test(schedSrc), "ScheduleLink.scheduleStoryAction must exist");
assert.ok(/scheduleDemandAction/.test(schedSrc), "ScheduleLink.scheduleDemandAction must exist");
assert.ok(/"\/schedule\/demands\/"\s*\+\s*id/.test(schedSrc) || /\/schedule\/demands\//.test(schedSrc), "ScheduleLink.url must align with /schedule/demands/:id route");
assert.ok(/"\/schedule\/stories\/"\s*\+\s*id/.test(schedSrc) || /\/schedule\/stories\//.test(schedSrc), "ScheduleLink.url must align with /schedule/stories/:id route");
console.log("PASS: schedule-link.js exports window.ScheduleLink with url/scheduleStoryAction/scheduleDemandAction");

const paSrc = fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/primary-action.js"), "utf8");
assert.ok(/window\.PrimaryAction\s*=/.test(paSrc), "primary-action.js must export window.PrimaryAction");
assert.ok(/primaryActionHtml/.test(paSrc), "PrimaryAction.primaryActionHtml must exist");
// 缺 primaryAction 时必须显示 "—" 而非"查看详情"（剥离行注释 / 块注释）
function stripJsComments(s) {
  return s
    .replace(/\/\*[\s\S]*?\*\//g, "")      // 块注释 /* ... */
    .split("\n")
    .map(function (l) {                    // 行注释 // ...
      var i = l.indexOf("//");
      return i < 0 ? l : l.slice(0, i);
    })
    .join("\n");
}
const paCodeOnly = stripJsComments(paSrc);
assert.ok(!/查看详情/.test(paCodeOnly), "primary-action.js must NOT render '查看详情' fallback in executable code");
// disabled/enabled 三态：enable=false 应显示 disabled 标签（待审批人处理等）
assert.ok(/enabled\s*===\s*false/.test(paSrc), "primary-action.js must honor enabled=false (no permission)");
console.log("PASS: primary-action.js exports window.PrimaryAction with disabled/enabled states and em-dash placeholder");

// -----------------------------------------------------------------------------
// 5. 模板：home.html / workboard.html 必须引入新的 helper 脚本
// -----------------------------------------------------------------------------
const homeTpl = fs.readFileSync(path.join(__dirname, "../../../web/templates/po/home.html"), "utf8");
assert.ok(/primary-action\.js/.test(homeTpl), "home.html must include primary-action.js before home.js");
assert.ok(homeTpl.indexOf("primary-action.js") < homeTpl.indexOf("home.js"), "primary-action.js must load before home.js (defines window.PrimaryAction used by home.js)");

const wbTpl = fs.readFileSync(path.join(__dirname, "../../../web/templates/po/workboard.html"), "utf8");
assert.ok(/schedule-link\.js/.test(wbTpl), "workboard.html must include schedule-link.js before workboard.js");
assert.ok(wbTpl.indexOf("schedule-link.js") < wbTpl.indexOf("workboard.js"), "schedule-link.js must load before workboard.js (defines window.ScheduleLink used by workboard.js)");
console.log("PASS: home.html and workboard.html include new helper scripts in correct order");

// -----------------------------------------------------------------------------
// 6. CSS：home.css 必须定义 home-unavailable 占位（否则 placeholder 不可见）
// -----------------------------------------------------------------------------
const homeCss = fs.readFileSync(path.join(__dirname, "../../../web/static/css/po/home.css"), "utf8");
assert.ok(/\.home-unavailable/.test(homeCss), "home.css must define .home-unavailable style for placeholder");
console.log("PASS: home.css defines .home-unavailable placeholder style");

console.log("\nALL: Stage 4 navigation + primaryAction contract honored");
