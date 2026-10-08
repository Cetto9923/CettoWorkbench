/* =============================================================================
   文件: tests/unit/frontend/badges-quiet401.test.js
   模块: 前端回归测试
   职责: 侧栏角标补取遇到 401 / 超时 / 5xx 时必须降级为「重试」态，
         且不得触发 appFetch 的全站跳登录；真实业务请求的 401 仍须跳登录。
   ============================================================================= */
'use strict';
const assert = require('assert/strict');
const fs = require('fs');
const vm = require('vm');

const appJs = fs.readFileSync('web/static/js/app.js', 'utf8');
const badgesJs = fs.readFileSync('web/static/js/layout/badges.js', 'utf8');

// 只抽取 app.js 的 appFetch 行为：真实文件还包含表单/分页/搜索绑定，
// 这些依赖真实 DOM，与本用例无关，故按函数边界切片执行。
function extractAppFetch() {
  const start = appJs.indexOf('  function redirectToLogin()');
  const end = appJs.indexOf('function setSubmittingState');
  assert.ok(start > 0 && end > start, '未能定位 appFetch 源码片段');
  return appJs.slice(start, end);
}

// 运行一次角标加载：注入假 fetch 与 DOM，返回是否发生跳登录。
async function runBadges(handler) {
  let navigatedTo = null;
  const badge = {
    dataset: { sidebarBadge: 'todos', badgePending: 'true' },
    textContent: '',
    title: '',
    addEventListener(name, cb) { if (name === 'click') this.click = cb; }
  };
  // appFetch 取自 app.js 真实源码；其 401 分支通过 location 赋值记录跳登录。
  const appSandbox = {
    window: {
      location: {
        pathname: '/home', search: '',
        get href() { return navigatedTo; },
        set href(v) { navigatedTo = v; }
      },
      getCsrfToken: () => 'test-token'
    },
    Headers: class { constructor() {} set() {} },
    fetch: handler,
    Promise, Object, Error, console
  };
  const sandbox = {
    window: {
      appFetch: appFetchFromSandbox(appSandbox),
      location: { pathname: '/home', search: '' },
      setTimeout: () => 0,
      clearTimeout: () => {}
    },
    document: {
      querySelectorAll: sel => (sel === '[data-sidebar-badge]' ? [badge] : [])
    },
    AbortController: function () {
      this.signal = { addEventListener() {} };
      this.abort = () => {};
    },
    console, Promise, Object, Error, JSON, String, Array, Boolean
  };
  vm.runInNewContext(badgesJs, sandbox);
  // 让 load() 内部的 await 链（fetch → json）全部走完。
  for (let i = 0; i < 8; i++) await new Promise(resolve => setImmediate(resolve));
  return { badge, navigatedTo };
}

// 用 app.js 真实源码构造 appFetch，确保被测的正是线上那份 401 分支，
// 而不是测试里另写一份可能漂移的副本。
function appFetchFromSandbox(sandbox) {
  const source = '(function(){' + extractAppFetch() + '\nreturn appFetch;})()';
  return vm.runInNewContext(source, sandbox);
}

function jsonResponse(status, body) {
  return {
    status,
    ok: status >= 200 && status < 300,
    json: () => (body === undefined ? Promise.reject(new Error('empty')) : Promise.resolve(body))
  };
}

async function main() {
  // b. 角标 401：不跳登录，角标进入可重试态。
  const unauthorized = await runBadges(() => Promise.resolve(jsonResponse(401, undefined)));
  assert.equal(unauthorized.navigatedTo, null, '角标 401 不得触发跳登录');
  assert.equal(unauthorized.badge.dataset.badgeFailed, 'true', '角标 401 应标记为失败可重试');
  assert.equal(unauthorized.badge.textContent, '重试', '角标 401 应显示重试文案');

  // 业务不可用（200 + unavailable）沿用同一降级路径。
  const unavailable = await runBadges(() => Promise.resolve(
    jsonResponse(200, { success: true, data: { unavailable: true } })));
  assert.equal(unavailable.navigatedTo, null, '角标 unavailable 不得触发跳登录');
  assert.equal(unavailable.badge.dataset.badgeFailed, 'true', '角标 unavailable 应标记失败');

  // 5xx 优雅降级，不弹错。
  const serverError = await runBadges(() => Promise.resolve(jsonResponse(503, { success: false })));
  assert.equal(serverError.navigatedTo, null, '角标 5xx 不得触发跳登录');
  assert.equal(serverError.badge.dataset.badgeFailed, 'true', '角标 5xx 应标记失败');

  // 网络错误同样降级。
  const networkError = await runBadges(() => Promise.reject(new Error('network down')));
  assert.equal(networkError.navigatedTo, null, '角标网络错误不得触发跳登录');
  assert.equal(networkError.badge.dataset.badgeFailed, 'true', '网络错误应标记失败');

  // a. 正常响应：角标显示真实数字。
  const ok = await runBadges(() => Promise.resolve(
    jsonResponse(200, { success: true, data: { todos: 7, notice: 120 } })));
  assert.equal(ok.navigatedTo, null, '正常响应不得跳登录');
  assert.equal(ok.badge.textContent, '7', '正常响应应显示待办数字');
  assert.equal(ok.badge.dataset.badgeFailed, 'false', '正常响应不应标记失败');

  // c. 对照组：不带 quietUnauthorized 的真实业务请求，401 仍然跳登录。
  // appFetch 跳登录后返回的是「永不 settle」的 Promise，这里不能 await，
  // 否则用例自身会挂住；只断言跳登录已发生。
  let redirected = null;
  const appFetch = appFetchFromSandbox({
    window: {
      location: { pathname: '/schedule', search: '', set href(v) { redirected = v; }, get href() { return redirected; } },
      getCsrfToken: () => 'test-token'
    },
    Headers: class { constructor() {} set() {} },
    fetch: () => Promise.resolve(jsonResponse(401, undefined)),
    Promise, Object, Error, console
  });
  appFetch('/schedule/stories/1/tasks', { method: 'POST' });
  await new Promise(resolve => setImmediate(resolve));
  assert.ok(String(redirected).startsWith('/login'), '真实业务请求 401 仍须跳登录，实际 ' + redirected);

  // 显式开启静默时，appFetch 必须把 401 原样交回调用方（且不下发无效字段）。
  let quietRedirected = null;
  let sentOptions = null;
  const quietAppFetch = appFetchFromSandbox({
    window: {
      location: { pathname: '/home', search: '', set href(v) { quietRedirected = v; }, get href() { return quietRedirected; } },
      getCsrfToken: () => 'test-token'
    },
    Headers: class { constructor() {} set() {} },
    fetch: (url, options) => { sentOptions = options; return Promise.resolve(jsonResponse(401, undefined)); },
    Promise, Object, Error, console
  });
  const quietResp = await quietAppFetch('/navigation/badges', { quietUnauthorized: true });
  assert.equal(quietResp.status, 401, '静默模式下 401 应原样返回');
  assert.equal(quietRedirected, null, '静默模式不得跳登录');
  assert.equal('quietUnauthorized' in sentOptions, false, 'quietUnauthorized 是内部开关，不得传给 fetch');

  console.log('sidebar badge 401 degrades without redirecting to login passed');
}

main().catch(err => { console.error(err); process.exit(1); });