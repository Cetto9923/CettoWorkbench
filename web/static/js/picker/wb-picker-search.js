/* =============================================================================
 * 文件: web/static/workbench/shared/picker/wb-picker-search.js
 * 职责: shared picker 搜索索引，支持中文、account、拼音全拼、首字母、多 token AND。
 * ============================================================================= */
(function (global) {
  'use strict';

  function text(v) { return String(v == null ? '' : v).trim(); }
  function lower(v) { return text(v).toLowerCase(); }
  function tokens(q) {
    return lower(q).replace(/\s+/g, ' ').split(' ').filter(Boolean);
  }
  function pinyinFull(s) {
    var P = global.PinyinLite;
    return P && typeof P.fullPinyin === 'function' ? P.fullPinyin(s) : '';
  }
  function pinyinInitials(s) {
    var P = global.PinyinLite;
    return P && typeof P.initials === 'function' ? P.initials(s) : '';
  }
  function buildSearchText(values) {
    var parts = [];
    (values || []).forEach(function (v) {
      var s = text(v);
      if (!s) return;
      parts.push(s);
      var full = pinyinFull(s);
      var init = pinyinInitials(s);
      if (full) parts.push(full);
      if (init) parts.push(init);
    });
    return parts.join(' ').toLowerCase();
  }
  function optionSearchText(option) {
    if (!option) return '';
    if (option.searchText) return lower(option.searchText);
    var meta = option.meta || {};
    // 服务端权威拼音（zt_user.pinyin 列，禅道原生维护）优先；
    // 前端 pinyin-lite 高频字典兜底覆盖元数据字段，避免生僻字搜索盲区。
    var parts = [
      option.label, option.value, option.id,
      meta.account, meta.realname, meta.pinyin, meta.deptName, meta.teamName, meta.role
    ];
    // 服务端拼音（如 "maoyuzhang(001196) myz0"）单独原样拼一份，
    // 既保留前端 pinyin-lite 拼音索引，也包含禅道权威拼音全拼/首字母组合。
    if (meta.pinyin) parts.push(meta.pinyin);
    return buildSearchText(parts);
  }
  function flatten(options, out) {
    out = out || [];
    (options || []).forEach(function (o) {
      if (!o) return;
      out.push(o);
      if (Array.isArray(o.children)) flatten(o.children, out);
    });
    return out;
  }
  function matches(option, query) {
    var ts = tokens(query);
    if (!ts.length) return true;
    var hay = optionSearchText(option);
    for (var i = 0; i < ts.length; i += 1) {
      if (hay.indexOf(ts[i]) < 0) return false;
    }
    return true;
  }
  function filter(options, query) {
    var ts = tokens(query);
    if (!ts.length) return options || [];
    return (options || []).map(function (o) {
      if (!o) return null;
      var children = Array.isArray(o.children) ? filter(o.children, query) : [];
      if (matches(o, query) || children.length) {
        var next = {};
        Object.keys(o).forEach(function (k) { next[k] = o[k]; });
        if (Array.isArray(o.children)) next.children = children;
        return next;
      }
      return null;
    }).filter(Boolean);
  }

  global.WbPickerSearch = {
    tokens: tokens,
    buildSearchText: buildSearchText,
    optionSearchText: optionSearchText,
    flatten: flatten,
    matches: matches,
    filter: filter
  };
})(typeof window !== 'undefined' ? window : globalThis);
