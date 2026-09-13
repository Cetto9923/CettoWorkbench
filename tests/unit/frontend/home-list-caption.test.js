const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Regression coverage for two small home.js bugs found during F12 test-entry
// review: (1) updateTitle must not report "已筛选" merely because a stage has
// more than one page; (2) initFromUrl must still restore the saved pageSize
// when the URL carries no query string at all (first visit to /home).

function makeJQueryMock(texts) {
  return function $(selector) {
    if (typeof selector === 'function') { $._ready = selector; return; }
    const obj = {
      length: 1,
      on() { return this; },
      attr(key, value) {
        if (value !== undefined) return this;
        if (key === 'data-vs-status') return 'testing';
        return '';
      },
      first() { return this; },
      removeClass() { return this; },
      addClass() { return this; },
      removeAttr() { return this; },
      prop() { return this; },
      html() { return this; },
      empty() { return this; },
      text(value) {
        if (value === undefined) return texts.get(selector) || '';
        texts.set(selector, value);
        return this;
      }
    };
    return obj;
  };
}

async function loadHomeAndResolve({ search, loadPageSize, items, total }) {
  const texts = new Map();
  const $ = makeJQueryMock(texts);
  const requests = [];
  const saveCalls = [];
  let resolveFetch;
  let paginationOptions;
  const window = {
    location: { pathname: '/home', search },
    history: { replaceState() {} },
    escapeHtml: (v) => String(v == null ? '' : v),
    appFetch(url) {
      requests.push(url);
      return new Promise((resolve) => { resolveFetch = resolve; });
    },
    PersonalList: {
      PAGE_SIZE_OPTIONS: [10, 15, 20, 50, 100],
      escapeHtml: (v) => String(v == null ? '' : v),
      priorityBadge: () => '',
      loadPageSize,
      savePageSize(key, value) { saveCalls.push([key, value]); },
      renderPagination(options) { paginationOptions = options; },
      createController: () => ({ bind() {}, destroy() {} })
    }
  };
  const ctx = {
    window, jQuery: $, URLSearchParams, document: { getElementById: () => null }
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/home-render.js'), 'utf8'), ctx);
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/home.js'), 'utf8'), ctx);
  $._ready();
  assert.equal(typeof resolveFetch, 'function', 'initial refreshDemands must call appFetch');
  resolveFetch({ ok: true, json: () => Promise.resolve({ success: true, items, total, page: 1, pageSize: 15 }) });
  for (let i = 0; i < 4; i += 1) {
    await new Promise((resolve) => setImmediate(resolve));
  }
  return { texts, requests, saveCalls, paginationOptions };
}

async function main() {
  // Bug 1: a multi-page stage with NO client-side filter applied must not be
  // mislabeled as "已筛选" (filtered) just because the page has fewer rows
  // than the stage's cross-page total.
  const items = Array.from({ length: 15 }, (_, i) => ({ id: 'US' + (i + 1), title: 't' + i, pri: 'p1' }));
  const { texts: unfilteredTexts } = await loadHomeAndResolve({
    search: '?status=testing',
    loadPageSize: (_key, fallback) => fallback,
    items,
    total: 100
  });
  const caption = unfilteredTexts.get('#homeListCaption') || '';
  assert.ok(!caption.includes('已筛选'), 'unfiltered multi-page stage must not claim to be filtered: ' + caption);
  assert.ok(caption.includes('共 100 条'), 'caption must show the true stage total: ' + caption);

  // Bug 2: visiting /home with no query string at all must still restore the
  // saved pageSize (loadPageSize), not silently fall back to the hardcoded
  // default because initFromUrl returned early on an empty search string.
  const { requests } = await loadHomeAndResolve({
    search: '',
    loadPageSize: () => 50,
    items,
    total: 100
  });
  // After WIP switched state.focus default to "my_action", the page now also
  // fires a summary probe (pageSize=1, fixed) before the real demands fetch;
  // assert the demands request specifically, which is the last call issued.
  const query = new URL(requests.at(-1), 'http://localhost').searchParams;
  assert.equal(query.get('pageSize'), '50', 'initFromUrl must apply saved pageSize even without a query string');

  // Bug 3: all offered page sizes must be accepted, and a new choice must be
  // persisted before the next refresh so it survives reopening /home.
  const persisted = await loadHomeAndResolve({
    search: '?pageSize=10',
    loadPageSize: () => 15,
    items,
    total: 100
  });
  assert.equal(new URL(persisted.requests.at(-1), 'http://localhost').searchParams.get('pageSize'), '10');
  persisted.paginationOptions.onPageSizeChange(20);
  assert.deepEqual(persisted.saveCalls, [['po.home.pageSize', 20]], 'page-size choice must be persisted');
  assert.equal(new URL(persisted.requests.at(-1), 'http://localhost').searchParams.get('pageSize'), '20');

  console.log('PASS: home caption reflects real filtering and pageSize memory persists across refreshes');
}

main().catch((err) => { console.error(err); process.exitCode = 1; });
