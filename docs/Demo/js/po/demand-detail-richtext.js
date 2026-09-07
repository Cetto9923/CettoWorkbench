// =============================================================================
// 文件: web/static/js/po/demand-detail-richtext.js
// 模块: PO 工作台
// 职责: F02 详情富文本净化辅助：sanitizeRichText / esc / 内联 DOMParser
//       白名单兜底，独立成文件以保持 demand-detail-render.js 的职责
//       边界与 500 行硬性上限；与 html-sanitize.js 共享同一组白名单语义。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.DemandDetailRichText = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  var sanitizer = null;
  if (typeof HtmlSanitize !== "undefined" && HtmlSanitize && HtmlSanitize.sanitizeHtml) {
    sanitizer = HtmlSanitize;
  } else if (typeof require === "function") {
    try { sanitizer = require("./html-sanitize.js"); } catch (e) { /* ignore */ }
  }

  function esc(str) {
    if (str === null || str === undefined) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function escapeTextNode(v) {
    return String(v)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function sanitizeRichTextFallback(input) {
    if (typeof DOMParser === "undefined") return null;
    if (typeof document === "undefined" || !document.createElement) return null;
    var doc = new DOMParser().parseFromString("<!doctype html><body>" + String(input) + "</body>", "text/html");
    if (!doc || !doc.body) return null;
    var out = [];
    var walk = function (node) {
      var children = Array.prototype.slice.call(node.childNodes);
      for (var i = 0; i < children.length; i++) {
        var c = children[i];
        if (c.nodeType === 3) { out.push(escapeTextNode(c.nodeValue || "")); continue; }
        if (c.nodeType !== 1) continue;
        var t = (c.tagName || "").toLowerCase();
        if (t === "script" || t === "iframe" || t === "object" || t === "embed" ||
            t === "style" || t === "svg" || t === "form" || t === "input" ||
            t === "button" || t === "link" || t === "meta" || t === "math" ||
            t === "foreignobject" || t === "applet" || t === "frame" || t === "frameset" ||
            t === "textarea" || t === "select" || t === "option" || t === "base") {
          continue;
        }
        out.push("<" + t + ">");
        walk(c);
        out.push("</" + t + ">");
      }
    };
    walk(doc.body);
    if (out.length === 0) return null;
    return out.join("");
  }

  function sanitizeRichText(raw) {
    if (raw === null || raw === undefined) return "";
    var s = String(raw);
    if (!s) return "";
    if (sanitizer && typeof sanitizer.sanitizeHtml === "function") {
      var out = sanitizer.sanitizeHtml(s);
      if (!out) return esc(s);
      return out;
    }
    var fallback = sanitizeRichTextFallback(s);
    if (fallback === null) return esc(s);
    return fallback;
  }

  function excerpt(text, max) {
    if (!text || text === "—") return "";
    var plain = String(text).replace(/<[^>]+>/g, " ").replace(/\s+/g, " ").trim();
    if (!plain) return "";
    var lim = max || 140;
    return plain.length > lim ? (plain.slice(0, lim) + "…") : plain;
  }

  return {
    esc: esc,
    sanitizeRichText: sanitizeRichText,
    excerpt: excerpt
  };
});