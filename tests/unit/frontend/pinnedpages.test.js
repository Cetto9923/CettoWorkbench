/* =============================================================================
   文件: tests/unit/frontend/pinnedpages.test.js
   模块: 前端回归测试
   职责: 公共导航固定页面的持久化回归
   ============================================================================= */
'use strict';
const assert = require('assert/strict');
const fs = require('fs');
const vm = require('vm');
const source = fs.readFileSync('web/static/js/layout/pinned-pages.js', 'utf8');
const saved = new Map();
function openPage(path) {
  const button = { style: {}, classList: {add(){},remove(){}}, querySelector(){return this.text;},
    text: {}, addEventListener(name, callback){this.click=callback;} };
  const links = ['/version-follow', '/admin/roles'].map(href => ({
    getAttribute(){return href;}, querySelector(selector){
      return selector === '.nav-text' ? {textContent: href} : {className:'fas fa-file'};
    }
  }));
  const window = {location:{pathname:path}, addEventListener(){}, dispatchEvent(){} };
  vm.runInNewContext(source, {window, CustomEvent: function(){}, console,
    localStorage:{getItem:key=>saved.get(key),setItem:(key,value)=>saved.set(key,value)},
    document:{readyState:'complete',getElementById(){return button;},querySelector(){return null;},
      querySelectorAll(selector){return selector === '.nav-item[href]' ? links : [];}}
  });
  return {button, pinned:window.WorkbenchPinned};
}
const version = openPage('/version-follow');
assert.equal(version.button.style.display, 'inline-flex');
version.button.click({preventDefault(){}});
assert.equal(version.button.text.textContent, '已在工作台');
const task = openPage('/board/task');
assert.equal(task.button.style.display, 'inline-flex');
assert.equal(task.pinned.isPinned('/version-follow'), true);
assert.equal(task.pinned.PINNABLE_PAGES.board_task.path, '/board/task');
assert.equal(openPage('/admin/roles/2/edit').button.style.display, 'inline-flex');
assert.equal(task.pinned.keyFromPathOrKey('/pmo'), 'agileteam');
assert.equal(task.pinned.keyFromPathOrKey('/issue-risk'), 'issues_risk');
console.log('shared page pin navigation and persistence passed');
