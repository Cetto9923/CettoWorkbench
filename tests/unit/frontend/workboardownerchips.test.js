/* =============================================================================
   文件: tests/unit/frontend/workboardownerchips.test.js
   模块: 前端回归测试
   职责: 任务看板零任务成员与展开去重回归
   ============================================================================= */
'use strict';
const assert = require('assert/strict');
const fs = require('fs');
const vm = require('vm');
const host = {
  children: [], innerHTML: '', addEventListener() {},
  appendChild(child) { this.children.push(child); },
  removeChild(child) { this.children = this.children.filter(item => item !== child); },
  querySelectorAll() { return this.children; }
};
const window = { location: {search:''}, escapeHtml: String, PersonalList: {}, PoWB: {} };
vm.runInNewContext(fs.readFileSync('web/static/js/po/workboard-core.js','utf8'), {
  window, location: {pathname:'/board/task'}, URLSearchParams,
  localStorage: {getItem(){return null;}},
  document: { getElementById(){return host;}, createElement(){
    return {dataset:{},classList:{remove(){},add(){}},addEventListener(event,action){this.click=action;}};
  } }
});
const people = [{value:'',display:'',count:0}].concat(Array.from({length:9},(_,i)=>({value:'member'+i,display:'成员'+i,count:i===0?2:0})));
window.PoWB.renderOwnerChips('taskOwners',people,()=>{});
assert.equal(host.children.length,8);
host.children.at(-1).click();
assert.equal(host.children.length,10);
assert.equal(new Set(host.children.map(item=>item.dataset.value)).size,10);
assert.ok(host.children.at(-1).innerHTML.includes('class="count">0</span>'));
console.log('workboard owners include zero counts and expand without duplication');
