/**
 * 顶栏全局检索：仅支持纯数字编号，跳转禅道对象详情（对齐常熟禅道 globalsearch + id-only）。
 */
(function () {
  "use strict";

  var DEFAULT_TYPE = "bug";

  // searchObjects（去掉 all），顺序与禅道一致
  var SEARCH_OBJECTS = [
    { key: "demand", label: "业务需求" },
    { key: "mr", label: "合并请求" },
    { key: "bug", label: "Bug" },
    { key: "story", label: "研发需求" },
    { key: "task", label: "任务" },
    { key: "testcase", label: "用例" },
    { key: "product", label: "产品" },
    { key: "build", label: "版本" },
    { key: "release", label: "发布" },
    { key: "productplan", label: "产品计划" },
    { key: "testtask", label: "测试单" },
    { key: "doc", label: "文档" },
    { key: "caselib", label: "用例库" },
    { key: "testreport", label: "测试报告" },
    { key: "program", label: "项目集" },
    { key: "project", label: "项目" },
    { key: "execution", label: "执行" },
    { key: "user", label: "用户" },
    { key: "aiapp", label: "AI" },
    { key: "feedback", label: "反馈" },
    { key: "ticket", label: "运维工单" },
    { key: "practice", label: "实践" },
    { key: "service", label: "服务" },
    { key: "deploy", label: "上线" },
    { key: "deploystep", label: "上线步骤" },
    { key: "issue", label: "问题" },
    { key: "risk", label: "风险" },
    { key: "opportunity", label: "机会" },
    { key: "trainplan", label: "培训计划" },
  ];

  var TYPE_OVERRIDE = {
    program: { module: "program", method: "product" },
    deploystep: { module: "deploy", method: "viewstep" },
    feedback: { module: "feedback", method: "adminView" },
  };

  var MSG_ID_ONLY = "仅支持编号搜索，请输入数字编号";

  var input;
  var results;
  var clearBtn;
  var activeKey = DEFAULT_TYPE;

  function meta(name) {
    var el = document.querySelector('meta[name="' + name + '"]');
    return el ? (el.getAttribute("content") || "").trim() : "";
  }

  function toast(message) {
    if (typeof window.showToast === "function") {
      window.showToast(message, "error");
      return;
    }
    window.alert(message);
  }

  function isPathInfo() {
    return meta("zentao-request-type").toUpperCase() === "PATH_INFO";
  }

  function zentaoBase() {
    return meta("zentao-url").replace(/\/+$/, "");
  }

  function pathInfoValues(raw) {
    var values = [];
    raw.split("&").forEach(function (pair) {
      if (!pair) return;
      var cut = pair.indexOf("=");
      values.push(cut >= 0 ? pair.slice(cut + 1) : pair);
    });
    return values;
  }

  // 对齐 internal/pkg/zentao.URL
  function createLink(module, method, params) {
    var base = zentaoBase();
    if (!base || !module || !method) return "";
    params = params || "";
    if (isPathInfo()) {
      var parts = [module, method];
      if (params) parts = parts.concat(pathInfoValues(params));
      return base + "/" + parts.join("-") + ".html";
    }
    var query = "m=" + encodeURIComponent(module) + "&f=" + encodeURIComponent(method);
    if (params) query += "&" + params;
    return base + "/index.php?" + query;
  }

  function resolveType(type) {
    var ov = TYPE_OVERRIDE[type];
    if (ov) return ov;
    return { module: type, method: "view" };
  }

  function buildObjectUrl(type, id) {
    var resolved = resolveType(type);
    return createLink(resolved.module, resolved.method, "id=" + id);
  }

  function labelOf(key) {
    for (var i = 0; i < SEARCH_OBJECTS.length; i++) {
      if (SEARCH_OBJECTS[i].key === key) return SEARCH_OBJECTS[i].label;
    }
    return key;
  }

  function isDigits(value) {
    return /^\d+$/.test(value);
  }

  function syncClearBtn() {
    if (!clearBtn) return;
    var has = !!(input && input.value.trim());
    clearBtn.hidden = !has;
  }

  function hideResults() {
    if (!results) return;
    results.classList.remove("show");
    results.innerHTML = "";
  }

  function openUrl(url) {
    if (!url) {
      toast("禅道地址未配置");
      return;
    }
    window.open(url, "_blank", "noopener,noreferrer");
  }

  function onSelect(type, id) {
    if (!type) return;
    openUrl(buildObjectUrl(type, id));
    hideResults();
  }

  function setActive(key) {
    activeKey = key;
    if (!results) return;
    var items = results.querySelectorAll(".gs-item");
    for (var i = 0; i < items.length; i++) {
      items[i].classList.toggle("is-active", items[i].getAttribute("data-key") === key);
    }
  }

  function renderResults(id) {
    if (!results) return;
    activeKey = DEFAULT_TYPE;
    var html = [];

    html.push(
      '<button type="button" class="gs-item gs-item-primary is-active" data-key="' +
        DEFAULT_TYPE +
        '">' +
        labelOf(DEFAULT_TYPE) +
        " #" +
        id +
        "</button>"
    );
    html.push('<div class="gs-grid">');
    SEARCH_OBJECTS.forEach(function (obj) {
      if (obj.key === DEFAULT_TYPE) return;
      html.push(
        '<button type="button" class="gs-item" data-key="' +
          obj.key +
          '">' +
          obj.label +
          " #" +
          id +
          "</button>"
      );
    });
    html.push("</div>");

    results.innerHTML = html.join("");
    results.classList.add("show");

    results.onclick = function (e) {
      var btn = e.target.closest(".gs-item");
      if (!btn || !results.contains(btn)) return;
      var key = btn.getAttribute("data-key") || "";
      setActive(key);
      onSelect(key, id);
    };

    results.onmouseover = function (e) {
      var btn = e.target.closest(".gs-item");
      if (!btn || !results.contains(btn)) return;
      setActive(btn.getAttribute("data-key") || "");
    };
  }

  function handleInput() {
    syncClearBtn();
    var value = (input.value || "").trim();
    if (!value) {
      hideResults();
      return;
    }
    if (!isDigits(value)) {
      hideResults();
      return;
    }
    renderResults(value);
  }

  function handleKeydown(e) {
    var value = (input.value || "").trim();
    if (e.key === "Escape") {
      hideResults();
      return;
    }
    if (e.key === "Enter") {
      e.preventDefault();
      if (!value) return;
      if (!isDigits(value)) {
        toast(MSG_ID_ONLY);
        return;
      }
      if (!results || !results.classList.contains("show")) {
        renderResults(value);
      }
      onSelect(activeKey || DEFAULT_TYPE, value);
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      if (!results || !results.classList.contains("show")) return;
      e.preventDefault();
      var items = Array.prototype.slice.call(results.querySelectorAll(".gs-item"));
      if (!items.length) return;
      var idx = items.findIndex(function (el) {
        return el.classList.contains("is-active");
      });
      if (idx < 0) idx = 0;
      idx = e.key === "ArrowDown" ? (idx + 1) % items.length : (idx - 1 + items.length) % items.length;
      setActive(items[idx].getAttribute("data-key") || "");
    }
  }

  function handleBlur() {
    // 延迟关闭，以便点击下拉项生效
    window.setTimeout(function () {
      if (!document.activeElement || !document.activeElement.closest || !document.activeElement.closest("#searchBox")) {
        hideResults();
      }
    }, 150);
  }

  function init() {
    input = document.getElementById("globalSearch");
    results = document.getElementById("searchResults");
    clearBtn = document.getElementById("globalSearchClear");
    if (!input || !results) return;

    input.addEventListener("input", handleInput);
    input.addEventListener("keydown", handleKeydown);
    input.addEventListener("focus", handleInput);
    input.addEventListener("blur", handleBlur);

    if (clearBtn) {
      clearBtn.addEventListener("mousedown", function (e) {
        e.preventDefault();
      });
      clearBtn.addEventListener("click", function () {
        input.value = "";
        syncClearBtn();
        hideResults();
        input.focus();
      });
    }

    document.addEventListener("click", function (e) {
      var box = document.getElementById("searchBox");
      if (box && !box.contains(e.target)) hideResults();
    });

    syncClearBtn();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
