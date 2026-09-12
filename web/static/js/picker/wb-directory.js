/* =============================================================================
 * 文件: web/static/workbench/shared/picker/wb-directory.js
 * 职责: shared picker 前端目录缓存。只做页面生命周期 Promise 缓存，不落 localStorage。
 * ============================================================================= */
(function (global) {
  'use strict';

  var cache = {};
  function fetchJson(url) {
    if (typeof global.apiFetch === 'function') return global.apiFetch(url);
    return fetch(url, { credentials: 'include', headers: { 'X-Requested-With': 'XMLHttpRequest' } })
      .then(function (res) { return res.ok ? res.json() : Promise.reject(new Error('HTTP ' + res.status)); });
  }
  function load(key, url) {
    if (!cache[key]) {
      cache[key] = fetchJson(url).then(function (json) {
        if (!json || json.success === false) throw new Error((json && json.message) || '目录加载失败');
        return Array.isArray(json.items) ? json.items : [];
      }).catch(function (err) {
        cache[key] = null;
        throw err;
      });
    }
    return cache[key];
  }
  function users() { return load('users', '/workbench/api/directory/users'); }
  function depts() { return load('depts', '/workbench/api/directory/depts'); }
  function agileTeams() { return load('agileTeams', '/workbench/api/directory/agile-teams'); }
  function reset() { cache = {}; }

  global.WbDirectory = { users: users, depts: depts, agileTeams: agileTeams, reset: reset };
})(typeof window !== 'undefined' ? window : globalThis);
