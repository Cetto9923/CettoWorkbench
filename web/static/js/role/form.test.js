// =============================================================================
// 文件: web/static/js/role/form.test.js
// 模块: 角色管理
// 职责: 模拟表单与响应，验证 JSON 提交和失败恢复。
// =============================================================================
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const source = fs.readFileSync(__dirname + '/form.js', 'utf8');
function setup(method, fetch) {
  const submit = { disabled: false };
  const error = { hidden: true, textContent: '' };
  const field = { hidden: true, textContent: '' };
  let listener;
  let redirect;
  const form = {
    action: '/admin/roles' + (method === 'PUT' ? '/8' : ''), dataset: { method },
    elements: { name: { value: '运营' }, remark: { value: '说明' } },
    querySelector: selector => selector.startsWith('button') ? submit : field,
    querySelectorAll: () => [field], addEventListener: (_, fn) => { listener = fn; }
  };
  vm.runInNewContext(source, { document: { getElementById: id => id === 'roleForm' ? form : error },
    window: { appFetch: fetch, location: { assign: url => { redirect = url; } } } });
  return { form, submit, error, field, send: () => listener({ preventDefault() {} }), redirect: () => redirect };
}
async function run() {
  let calls = 0;
  let release;
  const pending = new Promise(resolve => { release = resolve; });
  const create = setup('POST', async (_, options) => {
    calls++;
    assert.equal(options.method, 'POST');
    assert.deepEqual(JSON.parse(options.body), { name: '运营', remark: '说明' });
    await pending;
    return { ok: false, json: async () => ({ message: '请检查填写内容', errors: [{ field: 'name', message: '重复' }] }) };
  });
  const first = create.send();
  await create.send();
  assert.equal(calls, 1);
  assert.equal(create.submit.disabled, true);
  release(); await first;
  assert.equal(create.field.textContent, '重复');
  assert.equal(create.field.hidden, false);
  assert.equal(create.form.elements.name.value, '运营');
  assert.equal(create.submit.disabled, false);
  assert.equal(create.redirect(), undefined);
  const edit = setup('PUT', async (_, options) => {
    assert.equal(options.method, 'PUT');
    return { ok: true, json: async () => ({ success: true, redirectUrl: '/admin/roles' }) };
  });
  await edit.send(); assert.equal(edit.redirect(), '/admin/roles');
  const failed = setup('POST', async () => { throw new Error('network'); });
  await failed.send();
  assert.equal(failed.form.elements.remark.value, '说明');
  assert.equal(failed.error.hidden, false);
  assert.equal(failed.submit.disabled, false);
  const builtin = setup('PUT', () => { throw new Error('must not submit'); });
  builtin.submit.disabled = true; await builtin.send();
  assert.equal(builtin.error.hidden, true);
}
run().catch(error => { console.error(error); process.exitCode = 1; });
