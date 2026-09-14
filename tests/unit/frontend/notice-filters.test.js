const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

function loadNoticeScript(windowOverrides, documentOverrides) {
  const nodes = new Map();
  let ready;
  const document = Object.assign({
    getElementById(id) {
      if (!nodes.has(id)) nodes.set(id, {
        value: '', disabled: false, handlers: {},
        addEventListener(event, fn) { this.handlers[event] = fn; }
      });
      return nodes.get(id);
    },
    querySelectorAll() { return []; },
    addEventListener(_, fn) { ready = fn; }
  }, documentOverrides || {});
  const listCalls = [];
  const reminderAliases = {
    bug: "bug", task: "task",
    story: "story", demand: "business", issue: "issue",
    feedback: "feedback", charter: "charter", project: "project",
    testtask: "testtask", risk: "risk",
    "研发需求": "story", "业务需求": "business", "需求": "business",
    "任务": "task", "缺陷": "bug", "测试": "testtask", "问题": "issue",
    "风险": "risk", "反馈": "feedback", "章程": "charter", "项目": "project"
  };
  const reminderPattern = /(?:Bug|Task|Story|Demand|Issue|Feedback|Charter|Project|TestTask|Risk|研发需求|业务需求|需求|任务|缺陷|测试|问题|风险|反馈|章程|立项|项目)\s*[\(（]\s*\d+\s*[\)）]/i;
  function reminderKindFromSubject(subject) {
    const text = String(subject || "");
    if (!text) { return ""; }
    const m = text.match(reminderPattern);
    if (!m) { return ""; }
    const token = String(m[0]).replace(/[\(（]\s*\d+\s*[\)）]/, "").trim();
    return reminderAliases[token] || reminderAliases[token.toLowerCase()] || "";
  }
  const defaultPersonalList = {
    PAGE_SIZE_OPTIONS: [10, 15, 20, 50, 100],
    escapeHtml: (v) => String(v == null ? '' : v),
    objectTypeBadge: (k) => String(k || ''),
    loadPageSize: (_key, fallback) => fallback,
    savePageSize() {},
    renderPagination() {},
    reminderKindFromSubject,
    // 与 PersonalList.OBJECT_TYPE_SHORT_LABELS 保持一致：chip 上下文使用缩写 + "#" 分隔符。
    OBJECT_TYPE_SHORT_LABELS: {
      business: "业需", sub_demand: "子需", story: "研需",
      independent_story: "独立研需", task: "任务", bug: "Bug",
      testtask: "测单", issue: "问题", risk: "风险", approval: "审批",
      feedback: "反馈", charter: "章程", mail: "邮件", project: "项目"
    },
    idChipHtml(kind, idHtml) {
      var SHORT = this.OBJECT_TYPE_SHORT_LABELS;
      var LABEL = {
        business: "业务需求", sub_demand: "子需求", story: "研发需求",
        independent_story: "独立研发需求", task: "任务", bug: "Bug",
        testtask: "测试单", issue: "问题", risk: "风险", approval: "审批",
        feedback: "反馈", charter: "章程", mail: "邮件", project: "项目"
      };
      var k = String(kind || "").trim().toLowerCase();
      var safeId = typeof idHtml === "string" ? idHtml : "";
      var sep = safeId ? "#" : "";
      if (!k || !LABEL[k]) {
        return '<span class="wb-type wb-type-unknown">' +
          (LABEL[k] ? LABEL[k] : (k || "—")) + sep + safeId + "</span>";
      }
      return '<span class="wb-type wb-type-' + k + '">' +
        (SHORT[k] || LABEL[k]) + sep + safeId + "</span>";
    },
    createController: () => ({
      bind() {},
      destroy() {},
      fetch(url, _opts, onOk) {
        listCalls.push({ url });
        onOk({ items: [], filteredTotal: 0, total: 0, categories: {} });
      }
    })
  };
  const baseWindow = {
    location: { pathname: '/notice', search: '' },
    history: { replaceState(_s, _t, url) { window.location.search = String(url).includes('?') ? '?' + String(url).split('?')[1] : ''; } },
    escapeHtml: (v) => String(v == null ? '' : v),
    showToast() {},
    appFetch(url, options) {
      listCalls.push({ url, options });
      return Promise.resolve({
        ok: true,
        json: async () => ({ success: true, items: [], filteredTotal: 0, total: 0, categories: {} })
      });
    },
    PersonalList: defaultPersonalList
  };
  const overrides = windowOverrides || {};
  if (overrides.PersonalList) {
    baseWindow.PersonalList = Object.assign({}, defaultPersonalList, overrides.PersonalList);
    delete overrides.PersonalList;
  }
  const window = Object.assign(baseWindow, overrides);
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/notice.js'), 'utf8'), { window, document, URLSearchParams });
  return { nodes, ready, listCalls, window, document, reminderKindFromSubject };
}

// Exercise the shipped script's URL restoration and actual button request.
async function testMarkAllScopedFilters() {
  const calls = [];
  let finish;
  const { nodes, ready } = loadNoticeScript({
    location: { pathname: '/notice', search: '?quickView=unread&category=business&objectType=mail&timeRange=3d&needAction=required&readState=unread&keyword=100%25&page=3' },
    appFetch(url, options) {
      calls.push({ url, options });
      return new Promise(resolve => { finish = resolve; });
    }
  });
  ready();
  const button = nodes.get('noticeMarkAllBtn');
  button.handlers.click();
  button.handlers.click();
  assert.equal(calls.length, 1, 'double clicks must not issue concurrent mutations');
  assert.equal(calls[0].url, '/notice/read-all');
  assert.equal(calls[0].options.method, 'PUT');
  assert.deepEqual(JSON.parse(calls[0].options.body).filters, {
    quickView: 'unread', category: 'business', objectType: 'mail', timeRange: '3d',
    readState: 'unread', needAction: 'required', keyword: '100%', page: 3, pageSize: 20
  });
  finish({ ok: false, status: 500 });
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(button.disabled, false, 'failure must allow retry');
  console.log('PASS: notification URL filters, scoped request, duplicate click and retry');
}

function testDefaultQuickViewUnread() {
  const { ready, listCalls, window } = loadNoticeScript({
    location: { pathname: '/notice', search: '' }
  });
  ready();
  assert.ok(listCalls.some((c) => String(c.url).includes('quickView=unread')),
    'empty /notice must request unread by default');
  assert.ok(String(window.location.search).includes('quickView=unread'),
    'empty /notice must sync URL to quickView=unread');
  console.log('PASS: notification default quickView is unread');
}

function testExplicitQuickViewRespected() {
  const { ready, listCalls } = loadNoticeScript({
    location: { pathname: '/notice', search: '?quickView=action' }
  });
  ready();
  assert.ok(listCalls.some((c) => String(c.url).includes('quickView=action')),
    'explicit quickView must be respected');
  assert.ok(!listCalls.some((c) => /quickView=unread(?:&|$)/.test(String(c.url))),
    'explicit non-unread qv must not fall back to unread');
  console.log('PASS: explicit quickView=action is respected');
}

function testInformQuickViewAndRemovedToolbarFilter() {
  const template = fs.readFileSync(path.join(__dirname, '../../../web/templates/po/notice.html'), 'utf8');
  assert.ok(template.includes('data-qv="inform"'), 'notice page must expose the top-level inform quick filter');
  assert.equal(template.includes('id="noticeNeedAction"'), false,
    'notice page must not render the redundant processing-status select');

  const { listCalls, ready } = loadNoticeScript({
    location: { pathname: '/notice', search: '?quickView=inform' }
  });
  ready();
  assert.ok(listCalls.some((c) => String(c.url).includes('quickView=inform')),
    'quickView=inform must be sent to the notice list API');
  console.log('PASS: notification inform quick view and removed toolbar filter');
}

// Regression: the row must render the backend's subject verbatim (when the
// subject does NOT match a canonical 「TYPE #ID」prefix) and keep the structured
// object badge. Frontend regex-based prefix stripping is forbidden beyond the
// narrowly scoped canon+oid match, so legitimate subjects like "需求 X - 标题"
// or "STORY 父需求挂起逻辑优化" stay intact and never get carved into something
// that looks fake / 拼凑。
function testNoticeSubjectNotCharStripped() {
  const fixtureItem = {
    id: 1106459,
    objectType: 'story',
    objectId: 70526,
    subject: '父需求挂起逻辑优化 - 项目管理系统2.0',
    title: '父需求挂起逻辑优化 - 项目管理系统2.0',
    summary: '需求描述 当前子需求的交付周期未扣除父需求的挂起时段',
    data: '需求描述 当前子需求的交付周期未扣除父需求的挂起时段',
    actor: '胡昶(004481)',
    action: '',
    category: 'business',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2025-07-09 18:39:37',
    url: 'http://example.com/story/70526'
  };
  const { nodes, ready } = loadNoticeScript({
    location: { pathname: '/notice', search: '' },
    PersonalList: {
      escapeHtml: (v) => String(v == null ? '' : v),
      loadPageSize: (_key, fallback) => fallback,
      savePageSize() {},
      renderPagination() {},
      createController: () => ({
        bind() {}, destroy() {},
        fetch(_url, _opts, onOk) {
          onOk({ items: [fixtureItem], filteredTotal: 1, total: 1, categories: { all: 1 } });
        }
      })
    }
  });
  ready();
  const tbody = nodes.get('noticeTbody');
  const html = tbody && tbody.innerHTML ? String(tbody.innerHTML) : '';
  // 1. The visible title text must be the raw subject, not a regex-stripped version.
  const titleMatch = html.match(/(?:<button|<a)[^>]*class="[^"]*notice-title-main[^"]*"[^>]*>([^<]*)(?:<\/button>|<\/a>)/);
  assert.ok(titleMatch, 'row HTML must contain the title link or button');
  const visibleTitle = titleMatch[1];
  assert.equal(visibleTitle, '父需求挂起逻辑优化 - 项目管理系统2.0',
    'visible title must equal the raw backend subject; frontend must not strip anything beyond canon+oid prefix match');
  // 2. The structured object badge must still be rendered from objectType/objectId.
  assert.ok(html.indexOf('wb-type wb-type-story') !== -1,
    'row HTML must still render the structured object badge from objectType/objectId');
  assert.ok(/研发需求[\s\S]*?70526|研需[\s\S]*?70526/.test(html),
    'row HTML must contain the badge label (研发需求 or 研需) and the object id 70526 (id may be wrapped in <span> or <a>)');
  assert.ok(/<a class="table-id-link" href="http:\/\/example\.com\/story\/70526" target="_blank" rel="noopener noreferrer">#?70526<\/a>/.test(html),
    'row HTML must render clickable ID hyperlink when item.url is present');
  assert.ok(/<a class="table-title-link notice-title-main" href="http:\/\/example\.com\/story\/70526" target="_blank" rel="noopener noreferrer">/.test(html),
    'row HTML must render clickable title hyperlink when item.url is present');
  console.log('PASS: notice subject is not char-stripped; object badge remains from structured fields');
}

// helpers for fixture-driven badge assertions.
function loadSingleItem(item) {
  return loadNoticeScript({
    location: { pathname: '/notice', search: '' },
    PersonalList: {
      escapeHtml: (v) => String(v == null ? '' : v),
      loadPageSize: (_key, fallback) => fallback,
      savePageSize() {},
      renderPagination() {},
      createController: () => ({
        bind() {}, destroy() {},
        fetch(_url, _opts, onOk) {
          onOk({ items: [item], filteredTotal: 1, total: 1, categories: { all: 1 } });
        }
      })
    }
  });
}

function extractButtonTitle(html) {
  const m = html.match(/(?:<button|<a)[^>]*class="[^"]*notice-title-main[^"]*"[^>]*>([^<]*)(?:<\/button>|<\/a>)/);
  return m ? m[1] : null;
}

// feedback #2556: must produce a feedback badge with id, and the visible title
// must drop the 「反馈 #2556 」 prefix only because it cleanly matches canon+oid.
function testNoticeFeedbackBadgeAndTitle() {
  const item = {
    id: 'n-feedback-1',
    objectType: 'feedback',
    objectId: 2556,
    subject: '反馈 #2556 用例搜索条件维护带来的选择',
    summary: '描述: 列表筛选条件下拉项',
    actor: 'reviewer',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2026-09-07 12:00:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-feedback') !== -1,
    'feedback row must render wb-type-feedback class');
  assert.ok(/反馈[\s\S]*?2556/.test(html),
    'feedback row must render the badge text (反馈 + 2556 with possible <span> wrap)');
  const title = extractButtonTitle(html);
  assert.ok(title && title.indexOf('反馈 #2556') === -1,
    `feedback title button must not repeat "反馈 #2556" prefix; got: ${title}`);
  assert.ok(title && title.indexOf('用例搜索条件维护带来的选择') !== -1,
    `feedback title must preserve the original subject tail; got: ${title}`);
  console.log('PASS: feedback badge is rendered, title strips 反馈 #ID prefix once');
}

// demand 5418: 后端 objType=demand 必须归一到 business，徽章显示「业需#US5418」，
// 标题剥掉中文「需求 #5418 」前缀（与首页/待办统一）。
function testNoticeDemandBadgeAndTitle() {
  const item = {
    id: 'n-demand-1',
    objectType: 'demand',
    objectId: 5418,
    subject: '需求 #5418 业务动态同步',
    summary: '',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2026-09-07 12:01:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-business') !== -1,
    'demand row must map to wb-type-business (与首页/待办一致)');
  assert.ok(/业需[\s\S]*?US5418/.test(html),
    'demand row badge text must be 业需 + US5418 (chip 上下文缩写 + "#" 分隔符)');
  assert.ok(html.indexOf('wb-type wb-type-demand') === -1,
    'demand row must NOT render the legacy 需求 badge after canonicalisation');
  const title = extractButtonTitle(html);
  assert.ok(title && title.indexOf('需求 #5418') === -1,
    `demand title button must not repeat "需求 #5418" prefix; got: ${title}`);
  assert.ok(title && title.indexOf('业务动态同步') !== -1,
    `demand title must keep the original subject tail; got: ${title}`);
  console.log('PASS: demand objType maps to business badge; title strips 需求 #ID prefix once');
}

// STORY 4181 (story canon): 英文前缀同样命中时剥；徽章为「研发需求 4181」。
function testNoticeStoryEngPrefixStripped() {
  const item = {
    id: 'n-story-1',
    objectType: 'story',
    objectId: 4181,
    subject: 'STORY #4181 看板优化',
    summary: '',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2026-09-07 12:02:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-story') !== -1,
    'story row must render wb-type-story class');
  assert.ok(/研发需求[\s\S]*?4181|研需[\s\S]*?4181/.test(html),
    'story row badge text must be (研发需求 or 研需) + 4181 (with possible <span> wrap)');
  const title = extractButtonTitle(html);
  assert.ok(title && title.indexOf('STORY #4181') === -1,
    `story title button must not repeat "STORY #4181" prefix; got: ${title}`);
  assert.ok(title && title.indexOf('看板优化') !== -1,
    `story title must keep the original subject tail; got: ${title}`);
  console.log('PASS: story objType maps to 研发需求 badge; title strips STORY #ID prefix once');
}

// 未识别的 objType（如 system / sync）：不渲染 notice-tag 徽章，subject 原文照搬不剥。
function testNoticeUnknownKindKeepsRawSubject() {
  const item = {
    id: 'n-system-1',
    objectType: 'system',
    objectId: 0,
    subject: '系统通知：今日定时任务已完成',
    summary: '',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2026-09-07 12:03:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-system') === -1,
    'system kind must not render a wb-type-system badge (unknown canon)');
  // 只有 mail 是 canon，但 mail 也不该出徽章；这里测的是另一种 unknown。
  assert.ok(html.indexOf('class="wb-type') === -1,
    'unknown objectType must not emit any wb-type badge span');
  const title = extractButtonTitle(html);
  assert.equal(title, '系统通知：今日定时任务已完成',
    'unknown objectType must render the raw backend subject verbatim, no prefix stripping');
  console.log('PASS: unknown objectType keeps raw subject and renders no badge');
}

// 「提醒：您有 Bug(9)」模板：后端 objType 仍为空时，前端按 subject 命中兜底
// 渲染 bug 徽章，徽章不带 ID，subject 文本完整保留为按钮文本。
function testNoticeReminderBugBadge() {
  const item = {
    id: 'n-reminder-bug-1',
    objectType: '',
    objectId: 0,
    subject: '提醒：您有 Bug(9)',
    summary: '2025-01-15 http://pms.csr.cmbchina.com/bug-view-7890.html',
    data: '2025-01-15 http://pms.csr.cmbchina.com/bug-view-7890.html',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2025-01-15 09:00:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-bug') !== -1,
    'reminder template must render wb-type-bug class');
  assert.ok(html.indexOf('>Bug<') !== -1,
    'reminder badge text must be exactly "Bug" (no trailing #ID)');
  assert.ok(html.indexOf('Bug #7890') === -1,
    'reminder badge must not append #ID even when backend data hints at the bug id');
  const title = extractButtonTitle(html);
  assert.equal(title, '提醒：您有 Bug(9)',
    'reminder subject must remain verbatim as the button text; no prefix stripping');
  console.log('PASS: 提醒：您有 Bug(9) renders bug badge without id and keeps raw subject');
}

// 「您有 Task(3)」模板：兜底 task 徽章，subject 完整保留。
function testNoticeReminderTaskBadge() {
  const item = {
    id: 'n-reminder-task-1',
    objectType: '',
    objectId: 0,
    subject: '您有 Task(3)',
    summary: 'http://pms.csr.cmbchina.com/task-view-100.html',
    data: 'http://pms.csr.cmbchina.com/task-view-100.html',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2025-01-15 09:01:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-task') !== -1,
    'reminder template must render wb-type-task class');
  assert.ok(html.indexOf('>任务<') !== -1,
    'reminder badge text must be the canonical task label "任务" (no id)');
  const title = extractButtonTitle(html);
  assert.equal(title, '您有 Task(3)',
    'reminder subject must remain verbatim as the button text; no prefix stripping');
  console.log('PASS: 您有 Task(3) renders task badge without id and keeps raw subject');
}

// 「您有 需求(2)」模板：中文类型不在英文白名单，按业务需求 (business) 兜底。
function testNoticeReminderDemandBadge() {
  const item = {
    id: 'n-reminder-demand-1',
    objectType: '',
    objectId: 0,
    subject: '您有 需求(2)',
    summary: '',
    data: '',
    actor: 'system',
    needAction: false,
    anomaly: false,
    read: false,
    date: '2025-01-15 09:02:00'
  };
  const { nodes, ready } = loadSingleItem(item);
  ready();
  const html = String(nodes.get('noticeTbody').innerHTML || '');
  assert.ok(html.indexOf('wb-type wb-type-business') !== -1,
    'reminder template 需求(N) must fall back to business badge (与首页/待办一致)');
  assert.ok(html.indexOf('>业需<') !== -1,
    'reminder badge text must be the abbreviated "业需" label (chip 上下文使用 OBJECT_TYPE_SHORT_LABELS)');
  const title = extractButtonTitle(html);
  assert.equal(title, '您有 需求(2)',
    'reminder subject must remain verbatim as the button text');
  console.log('PASS: 您有 需求(2) falls back to business badge and keeps raw subject');
}

async function main() {
  await testMarkAllScopedFilters();
  testDefaultQuickViewUnread();
  testExplicitQuickViewRespected();
  testInformQuickViewAndRemovedToolbarFilter();
  testNoticeSubjectNotCharStripped();
  testNoticeFeedbackBadgeAndTitle();
  testNoticeDemandBadgeAndTitle();
  testNoticeStoryEngPrefixStripped();
  testNoticeUnknownKindKeepsRawSubject();
  testNoticeReminderBugBadge();
  testNoticeReminderTaskBadge();
  testNoticeReminderDemandBadge();
}
main().catch(err => { console.error(err); process.exitCode = 1; });
