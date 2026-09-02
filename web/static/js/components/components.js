// =============================================================================
// 文件: web/static/js/components/components.js
// 模块: 前端组件
// 职责: 统一循环加载 components 目录下的组件脚本
// =============================================================================

(function () {
  var scripts = ["form.js", "search.js", "batchform.js"];
  var basePath = "/static/js/components/";

  scripts.forEach(function (name) {
    if (name === "components.js") return;
    var script = document.createElement("script");
    script.src = basePath + name;
    // 关闭 async，保证按数组顺序执行。
    script.async = false;
    document.body.appendChild(script);
  });
})();
