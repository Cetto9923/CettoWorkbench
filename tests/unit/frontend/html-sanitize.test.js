// =============================================================================
// 文件: tests/unit/frontend/html-sanitize.test.js
// 模块: tests/unit/frontend
// 类型: test
// 职责: F02 渲染层净化防线单元测试。Node 没有原生 DOMParser，html-sanitize.js
//       的 DOMParser 分支在浏览器外不可执行；本测试聚焦于需求详情渲染器
//       的「最弱链路」：当 sanitizer 与 DOMParser 均不可用时，恶意 spec /
//       verify HTML 必须经 esc() 转义，事件属性与危险标签被中和，绝不会
//       出现在最终 innerHTML 输出中。DOMParser 路径由浏览器真实页面验收。
// =============================================================================

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

global.window = global;
global.document = {
  querySelector: () => null,
  addEventListener: () => {},
  removeEventListener: () => {}
};
// Load base shared scripts matching base.html load order
vm.runInThisContext(fs.readFileSync(path.join(__dirname, "../../../web/static/js/components/autocomplete-options.js"), "utf8"));
vm.runInThisContext(fs.readFileSync(path.join(__dirname, "../../../web/static/js/ui.js"), "utf8"));
vm.runInThisContext(fs.readFileSync(path.join(__dirname, "../../../web/static/js/po/personal-list.js"), "utf8"));

global.HtmlSanitize = undefined; // 强制走兜底分支

const R = require("../../../web/static/js/po/demand-detail-render.js");

console.log("=== Running F02 sanitization unit tests (renderer fallback path) ===");

// 1. sanitizeRichText 接受恶意 HTML 后必须不输出可执行事件属性
const maliciousSamples = [
  `<script>alert('xss-script')</script>`,
  `<img src="/x" onerror="alert('xss-onerror')" />`,
  `<a href="javascript:alert('xss-js')">click</a>`,
  `<svg onload="alert('xss-svg')"></svg>`,
  `<iframe src="https://evil"><p>child</p></iframe>`,
];

function hasUnescapedEventAttr(html) {
  // 任何 attribute 名以 on 开头 = 事件处理器；必须有 = 直接跟随原始串，
  // 即未经过 &quot; 等转义。
  return /(?<![\\w])on[a-z]+\s*=\s*["'][^"']+["']/i.test(html);
}

function hasUnescapedDangerousURL(html) {
  // 检测 href="javascript:..." / href="vbscript:..." 在未转义 href 里的形态
  return /\bhref\s*=\s*["']?(?:javascript|vbscript):/i.test(html);
}

for (const sample of maliciousSamples) {
  const out = R.sanitizeRichText(sample);
  // 必须不包含作为标签或属性的危险形态（已转义成 &lt; / &quot; 视为安全）
  assert.ok(!/<script[\s>]/i.test(out), `unescaped script tag leaked: ${out}`);
  assert.ok(!/<iframe[\s>]/i.test(out), `unescaped iframe tag leaked: ${out}`);
  assert.ok(!/<svg[\s>]/i.test(out), `unescaped svg tag leaked: ${out}`);
  assert.ok(!hasUnescapedEventAttr(out), `unescaped event attr leaked: ${out}`);
  assert.ok(!hasUnescapedDangerousURL(out), `dangerous url leaked: ${out}`);
  console.log("PASS: malicious payload neutralized by fallback path");
}

// 2. renderTabRequirement 必须不暴露恶意事件 / 标签
for (const sample of maliciousSamples) {
  const html = R.renderTabRequirement({ specHtml: sample, verifyHtml: sample });
  assert.ok(!/<script[\s>]/i.test(html), `unescaped script tag leaked into requirement HTML`);
  assert.ok(!/<iframe[\s>]/i.test(html), `unescaped iframe tag leaked into requirement HTML`);
  assert.ok(!/<svg[\s>]/i.test(html), `unescaped svg tag leaked into requirement HTML`);
  assert.ok(!hasUnescapedEventAttr(html), `unescaped event attr leaked into requirement HTML`);
  assert.ok(!hasUnescapedDangerousURL(html), `dangerous url leaked into requirement HTML`);
  console.log("PASS: renderTabRequirement blocks dangerous payload via fallback path");
}

// 3. 合法富文本在兜底分支下保留为可读转义文本
const safe = `<p>Hello <strong>world</strong></p>`;
const escaped = R.sanitizeRichText(safe);
assert.ok(!/<script/i.test(escaped));
assert.ok(/world/.test(escaped));
console.log("PASS: safe payload is escaped without losing readability");

// 4. 空输入返回空
assert.strictEqual(R.sanitizeRichText(""), "");
assert.strictEqual(R.sanitizeRichText(null), "");
assert.strictEqual(R.sanitizeRichText(undefined), "");
console.log("PASS: empty/null/undefined input returns empty string");

console.log("=== F02 sanitization unit tests passed ===");