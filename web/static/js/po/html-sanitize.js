// =============================================================================
// 文件: web/static/js/po/html-sanitize.js
// 模块: PO 工作台
// 职责: 共享 HTML 净化器（白名单标签 + 属性 + 协议；用于富文本字段
//       在 DOMParser 中重建节点，剔除 script / iframe / svg / on* /
//       javascript: 等危险形态）。F02 富文本注入修复的浏览器侧防线。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.HtmlSanitize = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  var ALLOWED_TAGS = {
    p: true, br: true, strong: true, em: true, u: true, s: true,
    ol: true, ul: true, li: true,
    h1: true, h2: true, h3: true, h4: true, h5: true, h6: true,
    blockquote: true, pre: true, code: true,
    table: true, thead: true, tbody: true, tfoot: true, tr: true, td: true, th: true,
    span: true, div: true,
    a: true, img: true
  };

  var ALLOWED_ATTRS = {
    a: ["href", "target", "rel", "title"],
    img: ["src", "alt", "title", "width", "height"],
    th: ["colspan", "rowspan", "align", "valign"],
    td: ["colspan", "rowspan", "align", "valign"],
    tr: ["align", "valign"],
    table: ["align", "width", "border", "cellpadding", "cellspacing"],
    th_: ["align"],
    span: ["align", "style"],
    div: ["align", "style"],
    p: ["align", "style"],
    h1: ["align", "style"],
    h2: ["align", "style"],
    h3: ["align", "style"],
    h4: ["align", "style"],
    h5: ["align", "style"],
    h6: ["align", "style"]
  };

  var SAFE_URL_SCHEMES = ["http:", "https:", "mailto:"];

  function getAllowedAttrs(tagName) {
    if (tagName === "th" || tagName === "td") return ALLOWED_ATTRS[tagName];
    return ALLOWED_ATTRS[tagName] || [];
  }

  function isSafeHref(value) {
    if (typeof value !== "string") return false;
    var trimmed = value.trim();
    if (trimmed === "" || trimmed.charAt(0) === "#") return true;
    var lower = trimmed.toLowerCase();
    if (lower.indexOf("javascript:") === 0) return false;
    if (lower.indexOf("data:") === 0) {
      if (lower.indexOf("data:image/") === 0) return true;
      return false;
    }
    if (lower.indexOf("vbscript:") === 0) return false;
    for (var i = 0; i < SAFE_URL_SCHEMES.length; i++) {
      var scheme = SAFE_URL_SCHEMES[i];
      if (lower.indexOf(scheme) === 0) return true;
    }
    return false;
  }

  function isSafeSrc(value) {
    if (typeof value !== "string") return false;
    var trimmed = value.trim();
    if (trimmed === "") return false;
    var lower = trimmed.toLowerCase();
    if (lower.indexOf("javascript:") === 0) return false;
    if (lower.indexOf("data:") === 0) {
      if (lower.indexOf("data:image/") === 0) return true;
      return false;
    }
    if (lower.indexOf("vbscript:") === 0) return false;
    for (var i = 0; i < SAFE_URL_SCHEMES.length; i++) {
      var scheme = SAFE_URL_SCHEMES[i];
      if (lower.indexOf(scheme) === 0) return true;
    }
    return false;
  }

  function sanitizeNode(node) {
    if (!node) return null;
    if (node.nodeType === 3) {
      return node.ownerDocument.createTextNode(node.nodeValue || "");
    }
    if (node.nodeType !== 1) {
      return null;
    }

    var tagName = (node.tagName || "").toLowerCase();

    // Treat SVG / foreignObject as opaque text (never embed svg via innerHTML).
    if (
      tagName === "svg" || tagName === "foreignobject" ||
      tagName === "math" || tagName === "script" || tagName === "iframe" ||
      tagName === "object" || tagName === "embed" || tagName === "style" ||
      tagName === "link" || tagName === "meta" || tagName === "form" ||
      tagName === "input" || tagName === "button" || tagName === "textarea" ||
      tagName === "select" || tagName === "option" || tagName === "frame" ||
      tagName === "frameset" || tagName === "base" || tagName === "applet"
    ) {
      return null;
    }

    if (!ALLOWED_TAGS[tagName]) {
      // Unknown tag → unwrap, keep safe text descendants.
      var frag = node.ownerDocument.createDocumentFragment();
      var children = Array.prototype.slice.call(node.childNodes);
      for (var i = 0; i < children.length; i++) {
        var cleaned = sanitizeNode(children[i]);
        if (cleaned) frag.appendChild(cleaned);
      }
      return frag;
    }

    var safe = node.ownerDocument.createElement(tagName);
    var allowed = getAllowedAttrs(tagName);

    if (node.attributes) {
      for (var j = 0; j < node.attributes.length; j++) {
        var attr = node.attributes[j];
        var attrName = (attr.name || "").toLowerCase();
        // Reject any on* handler by attribute name prefix.
        if (attrName.indexOf("on") === 0) continue;
        if (allowed.indexOf(attrName) === -1) continue;

        if (attrName === "href") {
          if (!isSafeHref(attr.value)) continue;
        } else if (attrName === "src") {
          if (!isSafeSrc(attr.value)) continue;
        } else if (attrName === "style") {
          // Permit only simple alignment declarations to avoid CSS injection.
          var safeStyle = sanitizeStyle(attr.value || "");
          if (!safeStyle) continue;
          safe.setAttribute("style", safeStyle);
          continue;
        }

        safe.setAttribute(attrName, attr.value);
      }
    }

    // Auto-add rel for external anchors (defense-in-depth even if target=_blank).
    if (tagName === "a" && safe.getAttribute("target") === "_blank") {
      if (!safe.getAttribute("rel")) {
        safe.setAttribute("rel", "noopener noreferrer");
      }
    }

    var children = Array.prototype.slice.call(node.childNodes);
    for (var k = 0; k < children.length; k++) {
      var cleanedChild = sanitizeNode(children[k]);
      if (cleanedChild) safe.appendChild(cleanedChild);
    }
    return safe;
  }

  function sanitizeStyle(raw) {
    if (typeof raw !== "string") return "";
    // Drop declarations with url() / expression() / @import / javascript:
    if (/expression\s*\(/i.test(raw)) return "";
    if (/url\s*\(/i.test(raw)) return "";
    if (/javascript:/i.test(raw)) return "";
    if (/@import/i.test(raw)) return "";
    // Only allow text-align / vertical-align / text-decoration related keywords.
    var safe = [];
    var parts = raw.split(";");
    for (var i = 0; i < parts.length; i++) {
      var decl = parts[i].trim();
      if (!decl) continue;
      var colon = decl.indexOf(":");
      if (colon < 0) continue;
      var key = decl.substring(0, colon).trim().toLowerCase();
      var val = decl.substring(colon + 1).trim();
      if (
        (key === "text-align" || key === "vertical-align" || key === "color" || key === "background") &&
        !/[<>]/.test(val)
      ) {
        safe.push(key + ": " + val);
      }
    }
    return safe.join("; ");
  }

  function getParser() {
    if (typeof DOMParser !== "undefined") return new DOMParser();
    if (typeof window !== "undefined" && window.DOMParser) return new window.DOMParser();
    return null;
  }

  // sanitizeHtml: parse the input HTML and rebuild only safe nodes.
  // Returns a sanitized HTML string. If no DOMParser is available, returns "".
  function sanitizeHtml(input) {
    if (input === null || input === undefined) return "";
    var raw = String(input);
    if (!raw) return "";
    var parser = getParser();
    if (!parser) return "";
    var doc = parser.parseFromString("<!doctype html><body>" + raw + "</body>", "text/html");
    if (!doc || !doc.body) return "";
    var out = [];
    var children = Array.prototype.slice.call(doc.body.childNodes);
    for (var i = 0; i < children.length; i++) {
      var clean = sanitizeNode(children[i]);
      if (clean) {
        if (clean.nodeType === 11) {
          // DocumentFragment: serialize children.
          var inner = Array.prototype.slice.call(clean.childNodes);
          for (var j = 0; j < inner.length; j++) {
            out.push(inner[j].outerHTML || inner[j].nodeValue || "");
          }
        } else {
          out.push(clean.outerHTML || clean.nodeValue || "");
        }
      }
    }
    return out.join("");
  }

  return {
    sanitizeHtml: sanitizeHtml,
    ALLOWED_TAGS: ALLOWED_TAGS
  };
});