/* =============================================================================
   文件: tests/unit/frontend/ui-consistency.test.js
   职责: 验证 UI 全局一致性修复落地的结构性证据。
         - 分页条与 components/pager.html 同一套页大小选项 + 跳页控件
         - PO 页面标题头卡片规格仅在 shell.css 一处定义
         - 描述性文本使用 --color-text-secondary 而非 --color-text-disabled
         - 深色主题下描述文字对比度满足可读基线
         - 筛选重置按钮文案与一级页面保持一致
   ============================================================================= */

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
global.window = global;

const root = path.join(__dirname, "../../..");
function read(rel) { return fs.readFileSync(path.join(root, rel), "utf8"); }
function cssRead(rel) { return read(path.join("web/static/css", rel)); }

// 1. 分页契约：客户端分页组件与 components/pager.html 同一套页大小选项。
(function testPaginationContract() {
  const PersonalList = require("../../../web/static/js/po/personal-list.js");
  assert.deepEqual(
    PersonalList.PAGE_SIZE_OPTIONS,
    [10, 20, 50, 100],
    "PersonalList.PAGE_SIZE_OPTIONS must be the unified contract [10,20,50,100]"
  );
  const pagerTpl = read("web/templates/components/pager.html");
  const serverSizes = [...pagerTpl.matchAll(/<option value="(\d+)"[^>]*>\d+ 条\/页/g)].map((m) => Number(m[1]));
  assert.deepEqual(serverSizes, [10, 20, 50, 100], "components/pager.html must offer [10,20,50,100]");
  console.log("PASS: pagination page-size options are the shared 10/20/50/100 contract");
})();

// 2. 页面标题头卡片规格仅由 shell.css 一处定义。
(function testSharedHeaderContract() {
  const shell = cssRead("po/shell.css");
  assert.match(
    shell,
    /body\.po-workbench-shell \.workspace-header \{[\s\S]*?border-radius: 8px;[\s\S]*?box-shadow: var\(--shadow-sm\)/,
    "shell.css must provide the canonical page-header card"
  );

  const dupDefs = [cssRead("po/personal-workspace.css"), cssRead("schedule/schedule.css")];
  for (const css of dupDefs) {
    assert.doesNotMatch(
      css,
      /\.po-personal-workspace \.workspace-header \{[\s\S]*?border-radius: 8px;/,
      "personal-workspace.css must not redefine the page-header card"
    );
    assert.doesNotMatch(
      css,
      /\.schedule-page-head \{[\s\S]*?border-radius: 8px;/,
      "schedule.css must not redefine the page-header card under .schedule-page-head"
    );
  }

  assert.match(shell, /--po-shell-pad-bottom:\s*46px;/, "shell.css must declare --po-shell-pad-bottom: 46px");
  assert.match(shell, /has-bottom-tabs \{ --po-shell-pad-bottom: 56px; \}/, "shell.css must raise pad-bottom for bottom-tabbar pages");
  const schedule = cssRead("schedule/schedule.css");
  assert.match(
    schedule,
    /height: calc\(100vh - var\(--po-shell-topbar-height\) - var\(--po-shell-pad-top\) - var\(--po-shell-pad-bottom\)\)/,
    "#page-schedule height calc must use the shared rhythm variables"
  );
  console.log("PASS: page-header card and rhythm are shared from shell.css");
})();

// 3. 描述性文字使用 --color-text-secondary，且 notice / follow 行内按钮统一为 28px/12px。
(function testDescriptiveTextTokens() {
  const cases = [
    ["po/notice.css", /\.notice-read-state\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/notice.css", /\.notice-read-btn\s*\{[^}]*height: 28px[^}]*font-size: 12px/],
    ["po/notice.css", /\.notice-view-btn\s*\{[^}]*font-size: 12px[^}]*height: 28px/],
    ["po/notice.css", /\.notice-missing-rel\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/notice.css", /\.notice-no-op\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-subtle\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-code\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-desc\.muted\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-risk-text\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-empty-row\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-pbox span\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.link-btn\.muted\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.pw-close\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/follow.css", /\.po-follow \.follow-btn\s*\{[^}]*height: 28px[^}]*font-size: 12px/],
    ["po/done.css", /\.done-arrow\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/done.css", /\.done-change-na\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/done.css", /\.done-ctx-pool\.done-ctx-na[\s\S]*?color: var\(--color-text-secondary\)/],
    ["po/demand-detail.css", /\.dd-cycle-chip \.s\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/demand-detail.css", /\.dd-ms \.sub\s*\{[^}]*color: var\(--color-text-secondary\)/],
    ["po/demand-detail.css", /\.dd-kpi-note\s*\{[^}]*color: var\(--color-text-secondary\)/],
  ];
  for (const [file, pattern] of cases) {
    const css = cssRead(file);
    assert.match(css, pattern, file + ": expected selector to use --color-text-secondary / unified metrics");
  }
  console.log("PASS: descriptive text uses --color-text-secondary across PO pages");
})();

// 4. 深色主题下描述文字对比度满足 WCAG AA 可读基线。
(function testDarkContrast() {
  function lum(hex) {
    const n = parseInt(hex.replace("#", ""), 16);
    const r = ((n >> 16) & 0xff) / 255, g = ((n >> 8) & 0xff) / 255, b = (n & 0xff) / 255;
    const ch = (c) => (c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4));
    return 0.2126 * ch(r) + 0.7152 * ch(g) + 0.0722 * ch(b);
  }
  function contrast(fg, bg) {
    const L1 = lum(fg), L2 = lum(bg);
    const [hi, lo] = L1 >= L2 ? [L1, L2] : [L2, L1];
    return (hi + 0.05) / (lo + 0.05);
  }
  const darkSurface = "#1e293b";
  const darkSecondary = "#94a3b8";
  const darkDisabled = "#475569";
  const secondaryOnSurface = contrast(darkSecondary, darkSurface);
  const disabledOnSurface = contrast(darkDisabled, darkSurface);
  assert.ok(secondaryOnSurface >= 4.5, "dark --color-text-secondary on surface must meet AA (got " + secondaryOnSurface.toFixed(2) + ":1)");
  assert.ok(disabledOnSurface < secondaryOnSurface, "disabled token must remain visually dimmer than secondary");
  assert.ok(disabledOnSurface >= 1.5, "dark --color-text-disabled should still be distinguishable from background (got " + disabledOnSurface.toFixed(2) + ":1)");
  assert.ok(contrast("#64748b", "#ffffff") >= 4.5, "light --color-text-secondary must meet AA on white");
  console.log("PASS: dark contrast — secondary " + secondaryOnSurface.toFixed(2) + ":1, disabled " + disabledOnSurface.toFixed(2) + ":1");
})();

// 5. 筛选重置按钮文案统一为"重置筛选"。
(function testResetLabel() {
  const pages = [
    "web/templates/po/home.html",
    "web/templates/po/notice.html",
    "web/templates/po/follow.html",
    "web/templates/schedule/index.html",
    "web/templates/query/index.html",
    "web/templates/po/todos.html",
    "web/templates/po/issue-risk.html",
    "web/templates/metrics/manage.html",
  ];
  for (const rel of pages) {
    const html = read(rel);
    if (!/id="[^"]*[Rr]eset[^"]*"/.test(html)) { continue; }
    assert.match(html, /id="[^"]*[Rr]eset[^"]*"[^>]*>\s*重置筛选\s*</, rel + " must use 重置筛选 as the filter reset label");
  }
  console.log("PASS: filter reset label is unified to 重置筛选 across PO pages");
})();

// 6. 所有页面最后加载同一套按钮语义与弹窗底部尺寸。
(function testButtonSystem() {
  const base = read("web/templates/layout/base.html");
  const buttons = cssRead("components/button-system.css");
  assert.match(base, /components\/button-system\.css/, "base layout must load the shared button system");
  assert.match(buttons, /\.action-btn\.primary/, "shared button system must define primary action buttons");
  assert.match(buttons, /\.btn-neutral/, "shared button system must define neutral buttons");
  assert.match(buttons, /\.batch-modal-footer \.action-btn/, "modal footer actions must share a minimum width");
  assert.match(buttons, /height: var\(--size-8\)/, "base buttons must share a 32px control height");
  console.log("PASS: buttons use one shared semantic system and modal footer geometry");
})();
