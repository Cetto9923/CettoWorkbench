// =============================================================================
// 文件: web/static/js/layout/badges.test.js
// 模块: 公共导航
// 职责: 验证计数异步补取、真实零值与失败重试。
// =============================================================================
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const source = fs.readFileSync(__dirname + '/badges.js', 'utf8');
async function run() {
  let resolve;
  let calls = 0;
  let fail = true;
  const badges = ['todos', 'notice', 'notice'].map(key => ({dataset:{sidebarBadge:key,badgePending:'true'}, addEventListener(_,fn){this.click=fn;}}));
  const pending = new Promise(done => { resolve = done; });
  vm.runInNewContext(source, {document:{querySelectorAll:()=>badges}, window:{appFetch:async () => {
    calls++; await pending;
    return {ok:!fail,json:async()=>({success:!fail,data:{todos:17,notice:0}})};
  }}});
  assert.equal(calls,1); assert.equal(badges[0].textContent,'…');
  resolve(); await new Promise(done=>setImmediate(done));
  assert.equal(badges[0].textContent,'重试');
  fail=false;
  let prevented=false;
  badges[0].click({preventDefault(){prevented=true;},stopPropagation(){}});
  await new Promise(done=>setImmediate(done));
  assert.equal(prevented,true); assert.equal(calls,2);
  assert.equal(badges[0].textContent,'17');
  assert.equal(badges[1].textContent,'0'); assert.equal(badges[2].textContent,'0');
  assert.equal(badges[0].dataset.badgeFailed,'false');
}
run().catch(error=>{console.error(error);process.exitCode=1;});
