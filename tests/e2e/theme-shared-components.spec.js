const fs = require('fs');
const path = require('path');

const cssDir = path.join(__dirname, '../../web/static/css');
const variablesPath = path.join(cssDir, 'layout/variables.css');
const variablesCss = fs.readFileSync(variablesPath, 'utf8');

const sharedCssFiles = [
  'layout/base.css',
  'layout/app.css',
  'layout/layout.css',
  'layout/topnav.css',
  'components/components.css',
  'components/form.css',
  'components/modal.css',
  'components/pager.css',
  'components/table.css',
  'components/search.css',
  'components/autocomplete.css',
  'components/batchform.css'
];

function assert(condition, message) {
  if (!condition) {
    console.error("FAIL: " + message);
    process.exit(1);
  }
}

// 1. Prohibited patterns gate
console.log("=== 1. Checking Prohibited Theme Patterns ===");
for (const relPath of sharedCssFiles) {
  const filePath = path.join(cssDir, relPath);
  const content = fs.readFileSync(filePath, 'utf8');

  assert(!content.includes('--dark-'), `${relPath} must not contain --dark-* tokens`);
  assert(!content.includes('--light-'), `${relPath} must not contain --light-* tokens`);
  assert(!content.includes('data-theme="system"'), `${relPath} must not contain data-theme="system"`);
  assert(!/\.dark\b|\.dark-theme\b|\.night\b/.test(content), `${relPath} must not use ad-hoc dark class names`);

  // Disallow page-level dark hacks inside shared CSS
  assert(!/\[data-theme="dark"\]\s*\.(home|todo|schedule|po)-/.test(content),
    `${relPath} must not contain page-level dark override hacks`);
}
console.log("PASS: No prohibited theme patterns found in shared CSS.");

// 2. Token extraction and recursive resolution
function extractBlock(content, startRegex) {
  const lines = content.split('\n');
  let inBlock = false;
  let blockContent = '';
  let bracesCount = 0;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    if (!inBlock) {
      if (startRegex.test(line)) {
        inBlock = true;
        if (line.includes('{')) {
          bracesCount++;
          blockContent += line.substring(line.indexOf('{') + 1) + '\n';
        }
      }
    } else {
      if (line.includes('{')) bracesCount++;
      if (line.includes('}')) {
        bracesCount--;
        if (bracesCount === 0) {
          blockContent += line.substring(0, line.indexOf('}')) + '\n';
          return blockContent;
        }
      }
      blockContent += line + '\n';
    }
  }
  return null;
}

const rootBlock = extractBlock(variablesCss, /(:root,?\s*html\[data-theme="light"\]|:root)/);
const darkBlock = extractBlock(variablesCss, /html\[data-theme="dark"\]/);

assert(rootBlock, "variables.css must have :root block");
assert(darkBlock, "variables.css must have html[data-theme=\"dark\"] block");

function getValue(blockContent, token) {
  const match = blockContent.match(new RegExp(token + '\\s*:\\s*([^;]+);'));
  return match ? match[1].trim() : null;
}

function resolveValue(blockContent, token) {
  let val = getValue(blockContent, token);
  if (!val && blockContent !== rootBlock) {
    val = getValue(rootBlock, token);
  }
  if (!val) return null;
  let depth = 0;
  while (val && val.startsWith('var(') && val.endsWith(')') && depth < 10) {
    depth++;
    const refToken = val.substring(4, val.length - 1).trim();
    val = getValue(blockContent, refToken) || getValue(rootBlock, refToken);
  }
  return val;
}

// 3. Contrast Calculation
function hexToRgb(hex) {
  const h = hex.replace(/^#/, '');
  let r, g, b;
  if (h.length === 3) {
    r = parseInt(h[0] + h[0], 16);
    g = parseInt(h[1] + h[1], 16);
    b = parseInt(h[2] + h[2], 16);
  } else if (h.length === 6) {
    r = parseInt(h.substring(0, 2), 16);
    g = parseInt(h.substring(2, 4), 16);
    b = parseInt(h.substring(4, 6), 16);
  } else return null;
  return [r, g, b];
}

function luminance(r, g, b) {
  const a = [r, g, b].map(v => {
    v /= 255;
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
  });
  return a[0] * 0.2126 + a[1] * 0.7152 + a[2] * 0.0722;
}

function getContrastRatio(hex1, hex2) {
  const rgb1 = hexToRgb(hex1);
  const rgb2 = hexToRgb(hex2);
  if (!rgb1 || !rgb2) return -1;
  const lum1 = luminance(rgb1[0], rgb1[1], rgb1[2]);
  const lum2 = luminance(rgb2[0], rgb2[1], rgb2[2]);
  const brightest = Math.max(lum1, lum2);
  const darkest = Math.min(lum1, lum2);
  return (brightest + 0.05) / (darkest + 0.05);
}

console.log("=== 2. Verifying Contrast Ratios for Key Semantic Pairs ===");
const semanticPairs = [
  { name: 'body text / page', text: '--color-text-primary', bg: '--color-page-bg', minDark: 4.5, minLight: 4.5 },
  { name: 'card text / surface', text: '--color-text-primary', bg: '--color-surface', minDark: 4.5, minLight: 4.5 },
  { name: 'dropdown text / surface', text: '--color-text-secondary', bg: '--color-surface', minDark: 4.5, minLight: 4.5 },
  { name: 'input text / surface', text: '--color-text-primary', bg: '--color-surface', minDark: 4.5, minLight: 4.5 },
  { name: 'placeholder / input surface', text: '--color-text-disabled', bg: '--color-surface', minDark: 3.0, minLight: 2.0 },
  { name: 'disabled visible text / surface', text: '--color-text-disabled', bg: '--color-surface-muted', minDark: 3.0, minLight: 2.0 },
  { name: 'selected text / selected background', text: '--color-primary', bg: '--color-primary-light', minDark: 3.5, minLight: 3.5 },
  { name: 'danger text / background', text: '--color-danger', bg: '--color-danger-bg', minDark: 3.0, minLight: 3.0 },
  { name: 'on-accent / danger fill', text: '--color-text-on-accent', bg: '--color-danger-fill', minDark: 4.5, minLight: 4.5 },
  { name: 'warning text / background', text: '--color-warning', bg: '--color-warning-bg', minDark: 3.0, minLight: 3.0 },
  { name: 'info text / background', text: '--color-info', bg: '--color-info-bg', minDark: 3.0, minLight: 3.0 },
  { name: 'on-accent / info fill', text: '--color-text-on-accent', bg: '--color-info-fill', minDark: 4.5, minLight: 4.5 }
];

for (const pair of semanticPairs) {
  const lightText = resolveValue(rootBlock, pair.text);
  const lightBg = resolveValue(rootBlock, pair.bg);
  assert(lightText && lightBg, `Light tokens must resolve for ${pair.name}: ${pair.text}=${lightText}, ${pair.bg}=${lightBg}`);
  const lightRatio = getContrastRatio(lightText, lightBg);
  assert(lightRatio >= pair.minLight,
    `Light contrast ratio too low for ${pair.name}: got ${lightRatio.toFixed(2)}, expected >= ${pair.minLight}`);

  const darkText = resolveValue(darkBlock, pair.text);
  const darkBg = resolveValue(darkBlock, pair.bg);
  assert(darkText && darkBg, `Dark tokens must resolve for ${pair.name}: ${pair.text}=${darkText}, ${pair.bg}=${darkBg}`);
  const darkRatio = getContrastRatio(darkText, darkBg);
  assert(darkRatio >= pair.minDark,
    `Dark contrast ratio too low for ${pair.name}: got ${darkRatio.toFixed(2)}, expected >= ${pair.minDark}`);

  console.log(`PASS: ${pair.name} - Light: ${lightRatio.toFixed(2)}, Dark: ${darkRatio.toFixed(2)}`);
}

// 4. Shared Component Rules and Declarations Validation
console.log("=== 3. Checking Shared Component Rules ===");

// Check autocomplete.css has no hardcoded hex/rgb
const autocompleteCss = fs.readFileSync(path.join(cssDir, 'components/autocomplete.css'), 'utf8');
assert(!/#[0-9a-fA-F]{3,8}/.test(autocompleteCss), "autocomplete.css must not contain hardcoded hex colors");
assert(autocompleteCss.includes('var(--color-surface)'), "autocomplete.css must use --color-surface");
assert(autocompleteCss.includes('var(--color-surface-hover)'), "autocomplete.css must use --color-surface-hover");
assert(autocompleteCss.includes('var(--color-primary-light)'), "autocomplete.css must use --color-primary-light");
console.log("PASS: autocomplete.css cleanly adapts to semantic tokens.");

// Check toast in components.css
const componentsCss = fs.readFileSync(path.join(cssDir, 'components/components.css'), 'utf8');
assert(componentsCss.includes('var(--color-success-bg)') && componentsCss.includes('var(--color-success-border)'),
  "components.css toast-success must use semantic tokens");
assert(componentsCss.includes('var(--color-danger-bg)') && componentsCss.includes('var(--color-danger-border)'),
  "components.css toast-error must use semantic tokens");
assert(componentsCss.includes('.btn-primary') && componentsCss.includes('var(--color-text-on-accent)'),
  "btn-primary must use --color-text-on-accent");
assert(componentsCss.includes('.btn-danger') && componentsCss.includes('var(--color-danger-fill)'),
  "btn-danger must use --color-danger-fill and --color-text-on-accent");
console.log("PASS: components.css buttons and toasts adapt to semantic tokens.");

// Check app.css for dropdown header/divider and form-control dark adaptation
const appCss = fs.readFileSync(path.join(cssDir, 'layout/app.css'), 'utf8');
assert(appCss.includes('.dropdown-header') && appCss.includes('.dropdown-divider'),
  "app.css must define .dropdown-header and .dropdown-divider");
assert(appCss.includes('.form-control') && appCss.includes('var(--color-surface)'),
  "app.css must style form-control with --color-surface and --color-text-primary");
assert(appCss.includes('.btn-neutral:hover'),
  "app.css must define .btn-neutral:hover");
console.log("PASS: app.css dropdown and form-control rules verified.");

// Check form.css for disabled states
const formCss = fs.readFileSync(path.join(cssDir, 'components/form.css'), 'utf8');
assert(formCss.includes('.input[disabled]') && formCss.includes('var(--color-surface-muted)'),
  "form.css must define disabled input state");
console.log("PASS: form.css disabled states verified.");

// Check table.css for table-row-btn color
const tableCss = fs.readFileSync(path.join(cssDir, 'components/table.css'), 'utf8');
assert(tableCss.includes('.table-row-btn') && tableCss.includes('var(--color-text-on-accent)'),
  "table.css table-row-btn must use --color-text-on-accent");
console.log("PASS: table.css table-row-btn verified.");

// Check base.css for accent-color
const baseCss = fs.readFileSync(path.join(cssDir, 'layout/base.css'), 'utf8');
assert(baseCss.includes('accent-color: var(--color-primary);'),
  "base.css must define accent-color for checkbox/radio");
console.log("PASS: base.css accent-color verified.");

// Check PO shell owns legacy page aliases used by mature PO modules such as schedule.
const poShellCss = fs.readFileSync(path.join(cssDir, 'po/shell.css'), 'utf8');
for (const token of [
  '--bg', '--white', '--hover', '--active', '--border', '--border-lt',
  '--t1', '--t2', '--t3', '--blue', '--blue-bg', '--blue-bd',
  '--red', '--red-bg', '--red-bd', '--orange', '--orange-bg', '--orange-bd',
  '--green', '--green-bg', '--green-bd', '--gray', '--gray-bg',
  '--purple', '--purple-bg', '--purple-bd'
]) {
  assert(poShellCss.includes(`${token}: var(--color-`) || poShellCss.includes(`${token}: var(--po-`),
    `po/shell.css must define ${token} from a semantic token`);
}

const scheduleCssFiles = [
  'schedule/schedule.css',
  'schedule/schedulefilter.css',
  'schedule/scheduleintegrated.css',
  'schedule/schedulelist.css',
  'schedule/schedulemodal.css',
  'schedule/schedulewindow.css'
];
const prohibitedScheduleUiColors = /#(?:fff|ffffff|f8fafc|f9fafb|fafbfc|f3f4f6|f1f5f9|e2e8f0|e5e7eb|cbd5e1|94a3b8|64748b|475569|334155|1e293b|0f172a)\b/i;
for (const relPath of scheduleCssFiles) {
  const content = fs.readFileSync(path.join(cssDir, relPath), 'utf8');
  assert(!content.includes(':root {'), `${relPath} must not redefine the PO theme alias palette`);
  assert(!prohibitedScheduleUiColors.test(content), `${relPath} must use PO shell/theme tokens instead of hardcoded UI colors`);
}
console.log("PASS: PO shell owns schedule theme aliases and schedule CSS avoids hardcoded UI colors.");

const poSharedComponentsCss = fs.readFileSync(path.join(cssDir, 'po/shared-components.css'), 'utf8');
const poBaseTemplate = fs.readFileSync(path.join(__dirname, '../../web/templates/layout/base.html'), 'utf8');

assert(poShellCss.includes('--po-scrollbar-thumb: var(--color-border-strong)'),
  "po/shell.css must define shared PO scrollbar tokens");
assert(poBaseTemplate.includes('/static/css/po/shell.css') &&
  poBaseTemplate.includes('/static/css/po/shared-components.css'),
  "PO base layout must load shared component theme states after the shell tokens");
assert(poSharedComponentsCss.includes('body.po-workbench-shell *::-webkit-scrollbar-thumb'),
  "po/shared-components.css must apply shared scrollbar styling to all PO descendants");
assert(poSharedComponentsCss.includes('.po-header-right .dropdown-menu') &&
  poSharedComponentsCss.includes('.theme-picker-group .dropdown-item[aria-checked="true"]'),
  "po/shared-components.css must own PO header dropdown and theme picker states");
assert(poSharedComponentsCss.includes('.action.primary') && poSharedComponentsCss.includes('var(--color-info-fill)'),
  "po/shared-components.css must map PO primary action buttons to semantic fill tokens");

const personalWorkspaceCss = fs.readFileSync(path.join(cssDir, 'po/personal-workspace.css'), 'utf8');
assert(personalWorkspaceCss.includes('.category-tabs::-webkit-scrollbar') &&
  personalWorkspaceCss.includes('scrollbar-width: none'),
  "personal-workspace category tabs must hide their incidental scrollbar");

const doneCss = fs.readFileSync(path.join(cssDir, 'po/done.css'), 'utf8');
assert(doneCss.includes('.done-tag.gray') &&
  doneCss.includes('background: var(--color-surface-muted)') &&
  !doneCss.includes('.done-tag.gray { background: #'),
  "done gray status tag must use semantic theme tokens");

const boardCss = fs.readFileSync(path.join(cssDir, 'po/board.css'), 'utf8');
assert(boardCss.includes('.po-board .action.primary') &&
  boardCss.includes('background:var(--color-info-fill)'),
  "board primary action buttons must use semantic fill tokens");

console.log("PASS: PO shared shell, tabs, done tags and board actions adapt through semantic tokens.");

console.log("All Shared Component Theme Contracts Passed Successfully!");
