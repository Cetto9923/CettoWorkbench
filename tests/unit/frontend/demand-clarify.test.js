const assert = require("assert");
const fs = require("fs");
const path = require("path");

const root = path.join(__dirname, "../../..");

// 1. po_clarify_modal.html: 抽屉容器必须使用独立的 demand-clarify-modal，彻底解除对 modal 的依赖
const modalTplPath = path.join(root, "web/templates/components/po_clarify_modal.html");
const modalTpl = fs.readFileSync(modalTplPath, "utf8");

const modalClassMatch = modalTpl.match(/<div id="poDemandClarifyModal"[^>]*class="([^"]*)"/);
assert.ok(modalClassMatch, "poDemandClarifyModal div must exist and have class attribute");
const modalClasses = modalClassMatch[1].trim().split(/\s+/);

assert.ok(
  modalClasses.includes("demand-clarify-modal"),
  "po_clarify_modal.html must include independent 'demand-clarify-modal' class"
);
assert.ok(
  !modalClasses.includes("modal"),
  "po_clarify_modal.html must NOT have standalone 'modal' class to prevent global modal pollution"
);
assert.match(
  modalTpl,
  /class="demand-clarify-dialog"/,
  "po_clarify_modal.html must have .demand-clarify-dialog"
);
assert.match(
  modalTpl,
  /class="demand-clarify-backdrop"/,
  "po_clarify_modal.html must have .demand-clarify-backdrop"
);
console.log("PASS: po_clarify_modal.html eliminates modal class and uses independent drawer structure");

// 2. demand-clarify.css: 必须拥有独立且具强特异性的抽屉样式，防御全局样式污染，并支持宽抽屉
const cssPath = path.join(root, "web/static/css/po/demand-clarify.css");
const css = fs.readFileSync(cssPath, "utf8");

assert.match(
  css,
  /\.demand-clarify-modal,\s*#poDemandClarifyModal/,
  "demand-clarify.css must reset and isolate .demand-clarify-modal and #poDemandClarifyModal"
);
assert.match(
  css,
  /\.demand-clarify-dialog\s*\{[^}]*right:\s*0/s,
  "demand-clarify.css must position drawer on the right"
);
assert.match(
  css,
  /\.demand-clarify-dialog\s*\{[^}]*width:\s*clamp\(\d+px,\s*\d+vw/s,
  "demand-clarify.css must support wide responsive drawer (80vw+ / 920px+)"
);
assert.match(
  css,
  /\.demand-clarify-dialog\s*\{[^}]*transform:\s*translateX\(100%\)/s,
  "demand-clarify.css must hide drawer with translateX(100%) when closed"
);
assert.match(
  css,
  /\.demand-clarify-modal\.show\s+\.demand-clarify-dialog\s*\{[^}]*transform:\s*translateX\(0\)/s,
  "demand-clarify.css must slide drawer in with translateX(0) when open"
);
console.log("PASS: demand-clarify.css supports 80vw/920px+ right-side drawer and resists global modal pollution");

// 3. 各入口页面挂载检查（home.html, todos.html, done.html, follow.html）
const pages = [
  { name: "home.html", file: "web/templates/po/home.html" },
  { name: "todos.html", file: "web/templates/po/todos.html" },
  { name: "done.html", file: "web/templates/po/done.html" },
  { name: "follow.html", file: "web/templates/po/follow.html" }
];

pages.forEach(({ name, file }) => {
  const content = fs.readFileSync(path.join(root, file), "utf8");
  assert.match(
    content,
    /static\/css\/po\/demand-clarify\.css/,
    `${name} must mount demand-clarify.css`
  );
  assert.match(
    content,
    /template\s+"po\/clarify_modal"/,
    `${name} must mount po/clarify_modal template`
  );
  assert.match(
    content,
    /static\/js\/po\/demand-clarify\.js/,
    `${name} must mount demand-clarify.js script`
  );
  console.log(`PASS: ${name} mounts clarify CSS, modal template, and JS script`);
});

// 4. demand-clarify.js: 导出 openPoDemandClarifyModal 且委托绑定 clarify 动作
const jsPath = path.join(root, "web/static/js/po/demand-clarify.js");
const js = fs.readFileSync(jsPath, "utf8");

assert.match(
  js,
  /window\.openPoDemandClarifyModal\s*=\s*openModal/,
  "demand-clarify.js must export window.openPoDemandClarifyModal"
);
assert.match(
  js,
  /\.js-po-drawer-action\[data-action-key='clarify'\]/,
  "demand-clarify.js must listen for clarify action buttons"
);
assert.match(
  js,
  /key\s*===\s*"Escape"/,
  "demand-clarify.js must support closing on Escape key"
);
console.log("PASS: demand-clarify.js exports modal API and handles clarify action clicks");
