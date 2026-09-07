const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

// Exercise the shipped script's URL restoration and actual button request.
async function main() {
  const nodes = new Map();
  const calls = [];
  let ready;
  let finish;
  const document = {
    getElementById(id) {
      if (!nodes.has(id)) nodes.set(id, {
        value: '', disabled: false, handlers: {},
        addEventListener(event, fn) { this.handlers[event] = fn; }
      });
      return nodes.get(id);
    },
    querySelectorAll() { return []; },
    addEventListener(_, fn) { ready = fn; }
  };
  const window = {
    location: { pathname: '/notice', search: '?quickView=unread&category=business&objectType=mail&timeRange=3d&needAction=required&readState=unread&keyword=100%25&page=3' },
    history: { replaceState() {} },
    appFetch(url, options) {
      calls.push({ url, options });
      return new Promise(resolve => { finish = resolve; });
    }
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../../../web/static/js/po/notice.js'), 'utf8'), { window, document, URLSearchParams });
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
main().catch(err => { console.error(err); process.exitCode = 1; });
