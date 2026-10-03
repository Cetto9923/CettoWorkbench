/* =============================================================================
   文件: tests/unit/frontend/agileteamdetail.test.js
   模块: 前端回归测试
   职责: 敏捷小组详情时序、父级选择与权限显示回归
   ============================================================================= */
const assert = require('assert');
const fs = require('fs');
const vm = require('vm');
const source = fs.readFileSync('web/static/js/agileteam/agileteam-detail.js', 'utf8');

async function latestDetailWins(rejectOld) {
  const state = {}, pending = [];
  const host = { innerHTML: '' }, parent = { value: '' };
  let writes = 0, toast = '', picker;
  const window = {
    escapeHtml: value => String(value ?? ''),
    destroyAutocomplete() {},
    initAutocomplete(...args) { picker = args; },
    showToast(message) { toast = message; },
    __at: {
      state, API: '/workbench/api/agile-teams', val: () => '',
      isLeadView: () => false, person: (name, account) => name || account,
      apiFetch(path, options) {
        if (options) { writes++; return Promise.resolve({}); }
        return new Promise((resolve, reject) => pending.push({ resolve, reject }));
      }
    }
  };
  const document = {
    getElementById: id => id === 'atDetailRoot' ? host : id === 'atBasicParent' ? parent : null,
    querySelectorAll: () => []
  };
  vm.runInNewContext(source, { window, document });
  const oldRequest = window.atLoadDetail(1), latestRequest = window.atLoadDetail(2);
  pending[1].resolve({ data: { id: 2, name: '当前小组', canEdit: true, parentId: 3, parentName: '上级', parentOptions: [{ id: 3, name: '上级' }] } });
  await latestRequest;
  const latestMarkup = host.innerHTML;
  if (rejectOld) pending[0].reject(new Error('旧请求失败'));
  else pending[0].resolve({ data: { id: 1, name: '旧小组' } });
  await oldRequest;
  assert.strictEqual(state.lastDetail.id, 2);
  assert.strictEqual(host.innerHTML, latestMarkup);
  assert.strictEqual(picker[0], 'atBasicParentInput');
  assert.deepStrictEqual(Array.from(picker[2], item => item.value), ['0', '3']);
  await window.atSaveBasic(2);
  assert.strictEqual(writes, 0, '未选择有效父级时不能默认脱离父级');
  assert.strictEqual(toast, '请选择有效的父级小组');
}

(async () => {
  await latestDetailWins(false);
  await latestDetailWins(true);
  console.log('agileteam detail race and searchable parent passed');
})().catch(error => { console.error(error); process.exitCode = 1; });
