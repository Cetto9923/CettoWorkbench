/* =============================================================================
   文件: docs/PRD/Demo/js/mock-service.js
   模块: PRD Demo 静态 Mock 引擎
   职责: 拦截原生 fetch 与 XMLHttpRequest，自动路由并返回离线 Mock 数据
   ============================================================================= */

(function () {
  "use strict";

  var MOCK_DB = window.MOCK_DATA || {};

  function matchRoute(url, options) {
    var rawUrl = String(url || "");
    var path = rawUrl.split("?")[0];
    var query = rawUrl.indexOf("?") >= 0 ? rawUrl.split("?")[1] : "";
    var params = new URLSearchParams(query);
    var method = (options && options.method) ? options.method.toUpperCase() : "GET";

    // 1. Demands 列表 (PO 首页)
    if (path === "/demands") {
      var status = params.get("status") || "all";
      var key = "demands_" + status;
      if (MOCK_DB[key]) {
        return { status: 200, json: MOCK_DB[key] };
      }
      return { status: 200, json: MOCK_DB["demands_all"] || { items: [], total: 0 } };
    }

    // 2. 需求详情 /demands/:id/detail
    var demandDetailMatch = path.match(/\/demands\/(\d+)\/detail/);
    if (demandDetailMatch) {
      var did = demandDetailMatch[1];
      var detailKey = "demand_" + did + "_detail";
      if (MOCK_DB[detailKey]) {
        return { status: 200, json: MOCK_DB[detailKey] };
      }
      return { status: 200, json: MOCK_DB["demand_63421_detail"] || { success: false } };
    }

    // 3. 我的待办 /todos/items
    if (path === "/todos/items") {
      var tab = params.get("tab") || "all";
      var base = JSON.parse(JSON.stringify(MOCK_DB["todos_all"] || { items: [] }));
      if (tab !== "all" && base.items) {
        base.items = base.items.filter(function (it) {
          return it.objectType === tab || it.tab === tab || it.kind === tab;
        });
        base.total = base.items.length;
      }
      return { status: 200, json: base };
    }

    // 4. 我的已办 /done/items
    if (path === "/done/items" || path === "/workbench/api/done") {
      var doneBase = JSON.parse(JSON.stringify(MOCK_DB["done_all"] || { items: [] }));
      var doneTab = params.get("tab") || "all";
      var keyword = (params.get("keyword") || "").trim().toLowerCase();
      var dItems = doneBase.items || [];
      if (doneTab !== "all") {
        dItems = dItems.filter(function (it) { return it.objectType === doneTab || it.actionKey === doneTab; });
      }
      if (keyword) {
        dItems = dItems.filter(function (it) {
          return (it.objectName && it.objectName.toLowerCase().indexOf(keyword) >= 0) ||
                 (it.actionName && it.actionName.toLowerCase().indexOf(keyword) >= 0) ||
                 (it.objectCode && it.objectCode.toLowerCase().indexOf(keyword) >= 0);
        });
      }
      doneBase.items = dItems;
      doneBase.total = dItems.length;
      return { status: 200, json: doneBase };
    }

    // 5. 已办元数据 /done/meta
    if (path === "/done/meta" || path === "/workbench/api/done/meta") {
      return { status: 200, json: MOCK_DB["done_meta"] || { data: {} } };
    }

    // 6. 已办详情 /done/detail/:actionId
    var doneDetailMatch = path.match(/\/done\/detail\/(\d+)/);
    if (doneDetailMatch) {
      var aid = parseInt(doneDetailMatch[1], 10);
      var allDone = (MOCK_DB["done_all"] && MOCK_DB["done_all"].items) || [];
      var matched = allDone.find(function (it) { return it.id === aid || it.sourceActionId === aid; });
      if (!matched && allDone.length > 0) matched = allDone[0];
      return {
        status: 200,
        json: {
          success: true,
          data: matched || {
            action: "操作记录",
            actor: "程统(003030)",
            date: "2026-09-07 14:00:00",
            objectName: "示例业务需求",
            commentSummary: "已于测试环境完成验证"
          }
        }
      };
    }

    // 7. 通知中心 /notice/items
    if (path === "/notice/items") {
      var qv = params.get("quickView") || "all";
      if (qv === "unread" && MOCK_DB["notice_unread"]) {
        return { status: 200, json: MOCK_DB["notice_unread"] };
      }
      if (qv === "action" && MOCK_DB["notice_action"]) {
        return { status: 200, json: MOCK_DB["notice_action"] };
      }
      return { status: 200, json: MOCK_DB["notice_all"] || { items: [], total: 0 } };
    }

    // 8. 通知单条已读 /notice/:id/read & 全部已读 /notice/read-all
    if (path.match(/\/notice\/\d+\/read/) || path === "/notice/read-all") {
      return { status: 200, json: { success: true, message: "已标记为已读" } };
    }

    // 9. 我的关注 /follow/items
    if (path === "/follow/items") {
      return { status: 200, json: MOCK_DB["follow_demand"] || { items: [], total: 0 } };
    }

    // 10. 关注项目周报 /follow/project-weeklies
    if (path === "/follow/project-weeklies" || path === "/workbench/api/watches/project-weeklies") {
      return { status: 200, json: MOCK_DB["follow_project_weeklies"] || { items: [], total: 0 } };
    }

    // 11. 周报团队 /follow/project-weeklies/teams
    if (path === "/follow/project-weeklies/teams" || path === "/workbench/api/watches/project-weeklies/teams") {
      return { status: 200, json: MOCK_DB["follow_teams"] || { items: [] } };
    }

    // 12. 工作看板 /board/demand/items
    if (path === "/board/demand/items") {
      return { status: 200, json: MOCK_DB["board_demand"] || { tree: [] } };
    }

    // 13. 看板右侧问题 /board/issues
    if (path === "/board/issues") {
      return { status: 200, json: MOCK_DB["board_issues"] || { items: [], total: 0 } };
    }

    // 14. 看板敏捷组指标 /board/group/metrics
    if (path === "/board/group/metrics") {
      return { status: 200, json: MOCK_DB["board_group_metrics"] || { metrics: {} } };
    }

    // 默认空响应
    return { status: 200, json: { success: true, items: [], total: 0 } };
  }

  // Hook window.fetch
  var originalFetch = window.fetch;
  window.fetch = function (input, init) {
    var url = typeof input === "string" ? input : (input && input.url ? input.url : "");
    if (url.startsWith("/demands") ||
        url.startsWith("/todos") ||
        url.startsWith("/done") ||
        url.startsWith("/notice") ||
        url.startsWith("/follow") ||
        url.startsWith("/board") ||
        url.startsWith("/workbench/api")) {
      var result = matchRoute(url, init);
      return Promise.resolve({
        ok: result.status >= 200 && result.status < 300,
        status: result.status,
        statusText: "OK",
        headers: new Headers({ "Content-Type": "application/json" }),
        json: function () { return Promise.resolve(result.json); },
        text: function () { return Promise.resolve(JSON.stringify(result.json)); }
      });
    }
    if (originalFetch) {
      return originalFetch.apply(this, arguments);
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: function () { return Promise.resolve({}); },
      text: function () { return Promise.resolve("{}"); }
    });
  };

  // Hook XMLHttpRequest
  var OriginalXHR = window.XMLHttpRequest;
  function MockXHR() {
    var xhr = new OriginalXHR();
    var self = this;
    this._xhr = xhr;
    this._url = "";
    this._method = "GET";
    this.status = 200;
    this.statusText = "OK";
    this.responseText = "";
    this.response = "";
    this.readyState = 0;
    this.onreadystatechange = null;
    this.onload = null;
    this.onerror = null;

    this.open = function (method, url, async, user, password) {
      self._method = method;
      self._url = url;
      self.readyState = 1;
      if (self.onreadystatechange) self.onreadystatechange();
    };

    this.setRequestHeader = function () {};

    this.send = function (body) {
      var res = matchRoute(self._url, { method: self._method, body: body });
      self.status = res.status;
      self.responseText = JSON.stringify(res.json);
      self.response = self.responseText;
      self.readyState = 4;
      setTimeout(function () {
        if (self.onreadystatechange) self.onreadystatechange();
        if (self.onload) self.onload();
      }, 20);
    };

    this.getResponseHeader = function (header) {
      if (header && header.toLowerCase() === "content-type") return "application/json";
      return null;
    };
    this.getAllResponseHeaders = function () {
      return "content-type: application/json\r\n";
    };
  }
  window.XMLHttpRequest = MockXHR;

})();
