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

// 侧栏二级菜单由服务端按 zt_menus 渲染，这里用最小 DOM 模拟其结构。
function linkNode(href, title) {
  return {
    getAttribute(name) { return name === 'href' ? href : null; },
    querySelector(selector) {
      if (selector === '.nav-text') return {textContent: title};
      if (selector === '.nav-icon') return {className: 'fas fa-file'};
      return null;
    },
    closest() { return null; },
    textContent: title
  };
}

function panelNode(group, groupTitle, links) {
  return {
    getAttribute(name) { return name === 'data-subnav-panel' ? group : null; },
    querySelector(selector) { return selector === '.po-subnav-title' ? {textContent: groupTitle} : null; },
    querySelectorAll(selector) {
      return selector === '.po-subnav-body a.nav-item[href]' ? links : [];
    }
  };
}

const panels = [
  panelNode('menu_110', '需求规划', [linkNode('/schedule', '需求排期'), linkNode('/version-follow', '版本跟进')]),
  panelNode('menu_120', '团队协作', [linkNode('/board/demand', '工作看板'), linkNode('/board/task', '任务看板')]),
  panelNode('menu_140', '治理分析', [linkNode('/issues/risk', '问题风险')])
];

function openPage(path) {
  const button = { style: {}, classList: {add(){},remove(){}}, querySelector(){return this.text;},
    text: {}, addEventListener(name, callback){this.click=callback;} };
  const sidebar = {
    querySelectorAll(selector) { return selector === '.po-subnav-panel' ? panels : []; }
  };
  const window = {location:{pathname:path}, addEventListener(){}, dispatchEvent(){} };
  vm.runInNewContext(source, {window, CustomEvent: function(){}, console,
    localStorage:{getItem:key=>saved.get(key),setItem:(key,value)=>saved.set(key,value)},
    document:{readyState:'complete',getElementById(id){return id === 'sidebar' ? sidebar : button;},
      querySelector(){return null;}, querySelectorAll(){return [];}}
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
assert.equal(task.pinned.PINNABLE_PAGES.board_task.title, '任务看板', '标题取自侧栏配置');
assert.equal(openPage('/admin/roles/2/edit').button.style.display, 'none', '未配置菜单不显示固定按钮');
assert.equal(task.pinned.keyFromPathOrKey('/pmo'), 'agileteam');
assert.equal(task.pinned.keyFromPathOrKey('/issue-risk'), 'issues_risk');
console.log('shared page pin navigation and persistence passed');
