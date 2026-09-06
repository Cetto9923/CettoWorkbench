const fs = require('fs');
const path = require('path');

const variablesCssPath = path.join(__dirname, '../../web/static/css/layout/variables.css');
const cssContent = fs.readFileSync(variablesCssPath, 'utf8');

const requiredTokens = [
  '--color-page-bg',
  '--color-surface',
  '--color-surface-hover',
  '--color-text-primary',
  '--color-text-secondary',
  '--color-text-disabled',
  '--color-border',
  '--color-border-strong',
  '--color-primary',
  '--color-success',
  '--color-warning',
  '--color-danger'
];

function extractBlock(startRegex) {
  const lines = cssContent.split('\n');
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
      if (line.includes('{')) {
        bracesCount++;
      }
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

const rootBlock = extractBlock(/(:root,?\s*html\[data-theme="light"\]|:root)/);
const darkBlock = extractBlock(/html\[data-theme="dark"\]/);

if (!rootBlock) {
  console.error("Missing :root / light block");
  process.exit(1);
}

if (!darkBlock) {
  console.error("Missing html[data-theme=\"dark\"] block");
  process.exit(1);
}

function checkTokens(blockContent, blockName) {
  for (const token of requiredTokens) {
    if (!blockContent.includes(token + ':')) {
      console.error(`Missing token ${token} in ${blockName}`);
      process.exit(1);
    }
  }
}

checkTokens(rootBlock, ':root');
checkTokens(darkBlock, 'html[data-theme="dark"]');

// Check resolved values difference for some key tokens
function getValue(blockContent, token) {
  const match = blockContent.match(new RegExp(token + '\\s*:\\s*([^;]+);'));
  return match ? match[1].trim() : null;
}

function resolveValue(blockContent, token) {
  let val = getValue(blockContent, token);
  if (!val) return null;
  // If it's a var reference, resolve it once
  if (val.startsWith('var(') && val.endsWith(')')) {
    const refToken = val.substring(4, val.length - 1);
    const resolved = getValue(blockContent, refToken);
    if (resolved) {
        return resolved;
    }
  }
  return val;
}

const tokensToCompare = [
  '--color-page-bg',
  '--color-surface',
  '--color-text-primary',
  '--color-border'
];

let hasDifference = false;
for (const token of tokensToCompare) {
  const lightVal = getValue(rootBlock, token);
  const darkVal = getValue(darkBlock, token);
  if (lightVal !== darkVal) {
    hasDifference = true;
    break;
  }
}

if (!hasDifference) {
  console.error("Light and Dark blocks seem identical, they must differ for key surface/text/border tokens");
  process.exit(1);
}

if (cssContent.includes('data-theme="system"')) {
  console.error("data-theme=\"system\" is forbidden in CSS");
  process.exit(1);
}

if (cssContent.includes('--dark-') || cssContent.includes('--light-')) {
  console.error("Theme specific token names like --dark-* or --light-* are forbidden");
  process.exit(1);
}

// Check Light Compatibility of Legacy PO Aliases
const expectedLightValues = {
  '--po-bg': '#f4f5f7',
  '--po-white': '#ffffff',
  '--po-border': '#e5e7eb',
  '--po-t1': '#1f2937',
  '--po-t2': '#6b7280',
  '--po-t3': '#9ca3af',
  '--po-blue': '#2563eb',
  '--po-blue-bg': '#eff6ff',
  '--po-blue-bd': '#bfdbfe',
  '--po-red': '#dc2626',
  '--po-red-bg': '#fef2f2',
  '--po-red-bd': '#fecaca',
  '--po-orange': '#ea580c',
  '--po-orange-bg': '#fff7ed',
  '--po-orange-bd': '#fed7aa',
  '--po-green': '#16a34a',

  // Ensure new action tokens maintain correct historic visual values in Light mode
  '--color-danger-action': '#be123c', // Must visually match old danger-deep light value
  '--color-danger-action-hover': '#9f1239', // Must visually match old danger-deeper light value
};

for (const [alias, expectedVal] of Object.entries(expectedLightValues)) {
  let resolvedVal = resolveValue(rootBlock, alias);
  if (!resolvedVal) {
    console.error(`Legacy PO alias ${alias} is missing or unresolvable!`);
    process.exit(1);
  }
  if (resolvedVal.toLowerCase() !== expectedVal.toLowerCase()) {
    console.error(`Legacy PO alias ${alias} resolved light value changed! Expected ${expectedVal}, got ${resolvedVal}`);
    process.exit(1);
  }
}

// WCAG Contrast ratio helper
function hexToRgb(hex) {
  // Support 3 and 6 digit hex
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
  } else {
    return null;
  }
  return [r, g, b];
}

function luminance(r, g, b) {
  const a = [r, g, b].map(function (v) {
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

const contrastChecks = [
  { text: '--color-text-primary', bg: '--color-page-bg', min: 4.5 },
  { text: '--color-text-primary', bg: '--color-surface', min: 4.5 },
  { text: '--color-text-disabled', bg: '--color-page-bg', min: 4.5 },
  { text: '--color-text-disabled', bg: '--color-surface', min: 4.5 },

  { text: '--color-success', bg: '--color-success-bg', min: 3.0 },
  { text: '--color-danger', bg: '--color-danger-bg', min: 3.0 },
  { text: '--color-warning', bg: '--color-warning-bg', min: 3.0 },

  { text: '--color-text-on-accent', bg: '--color-danger-fill', min: 4.5 },
  { text: '--color-text-on-accent', bg: '--color-warning-fill', min: 4.5 },
  { text: '--color-text-on-accent', bg: '--color-info-fill', min: 4.5 },

  { text: '--color-danger-action', bg: '--color-surface', min: 4.5 },
  { text: '--color-danger-action-hover', bg: '--color-surface', min: 4.5 },
];

for (const check of contrastChecks) {
  const textHex = resolveValue(darkBlock, check.text);
  const bgHex = resolveValue(darkBlock, check.bg);

  if (!textHex || !bgHex) {
    console.error(`Missing colors for contrast check: ${check.text} (${textHex}) vs ${check.bg} (${bgHex})`);
    process.exit(1);
  }

  const ratio = getContrastRatio(textHex, bgHex);
  if (ratio < check.min) {
    console.error(`Contrast check failed in Dark Mode: ${check.text} vs ${check.bg}. Ratio ${ratio.toFixed(2)} is less than minimum ${check.min}`);
    process.exit(1);
  }
}

console.log("Theme tokens contract check passed!");
