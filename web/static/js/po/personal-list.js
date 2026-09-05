/* =============================================================================
   文件: web/static/js/po/personal-list.js
   模块: PO 个人工作台 (UI-02 列表请求与分页共享原语)
   职责: 提供三列表（/todos, /done, /notice）有界且纯粹的 JSON 异步请求时序保护、
         统一状态（loading/empty/error）切换与统一分页组件渲染。
   禁止: 框架化、通用列表引擎、跨模块数据总线或业务逻辑接管。
   ============================================================================= */

(function () {
  "use strict";

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  /**
   * createController: 创建带时序保护（E06）和状态隔离（E05）的列表数据控制器。
   * options: {
   *   fetchUrl: function(state) -> string,
   *   summaryEl: HTMLElement,
   *   tbodyEl: HTMLElement,
   *   emptyEl: HTMLElement,
   *   errorEl: HTMLElement,
   *   retryBtn: HTMLElement,
   *   onSuccess: function(payload, isLatest),
   *   onError: function(err, isLatest)
   * }
   */
  function createController(opts) {
    var currentSeq = 0;
    var abortCtrl = null;

    function execute(url, state, onDone) {
      currentSeq += 1;
      var reqSeq = currentSeq;

      if (abortCtrl && typeof abortCtrl.abort === "function") {
        try { abortCtrl.abort(); } catch (e) {}
      }
      if (typeof AbortController !== "undefined") {
        abortCtrl = new AbortController();
      } else {
        abortCtrl = null;
      }

      // 状态切换：进入加载中，清空旧错误，不展示假空态
      if (opts.summaryEl) { opts.summaryEl.textContent = "加载中…"; }
      if (opts.emptyEl) { opts.emptyEl.hidden = true; }
      if (opts.errorEl) { opts.errorEl.hidden = true; }
      if (opts.tbodyEl && typeof opts.tbodyEl.setAttribute === "function") {
        opts.tbodyEl.setAttribute("aria-busy", "true");
      }

      var fetchFn = window.appFetch || window.fetch;
      var init = { method: "GET", headers: { "Accept": "application/json" } };
      if (abortCtrl) { init.signal = abortCtrl.signal; }

      fetchFn(url, init)
        .then(function (res) {
          if (res.status === 401) {
            var authErr = new Error("登录已过期，请重新登录");
            authErr.isAuth = true;
            throw authErr;
          }
          if (res.status === 403) {
            throw new Error("暂无权限访问该列表");
          }
          if (!res.ok) {
            throw new Error("服务响应异常 (" + res.status + ")");
          }
          return res.json().catch(function () {
            throw new Error("数据格式解析失败");
          });
        })
        .then(function (payload) {
          if (reqSeq !== currentSeq) { return; } // 旧请求已过期，直接丢弃（防乱序）
          if (opts.tbodyEl && typeof opts.tbodyEl.setAttribute === "function") {
            opts.tbodyEl.setAttribute("aria-busy", "false");
          }
          if (!payload || payload.success === false) {
            throw new Error((payload && (payload.error || payload.message)) || "返回数据无效");
          }
          if (opts.errorEl) { opts.errorEl.hidden = true; }
          var count = 0;
          if (typeof payload.filteredTotal === "number") {
            count = payload.filteredTotal;
          } else if (typeof payload.total === "number") {
            count = payload.total;
          } else if (Array.isArray(payload.items)) {
            count = payload.items.length;
          }
          if (opts.emptyEl) {
            opts.emptyEl.hidden = (count > 0);
          }
          if (opts.summaryEl) {
            opts.summaryEl.textContent = "共 " + count + " 条";
          }
          if (typeof opts.onSuccess === "function") {
            opts.onSuccess(payload);
          }
          if (typeof onDone === "function") { onDone(null, payload); }
        })
        .catch(function (err) {
          if (err && err.name === "AbortError") { return; } // 用户主动取消/切换，不报错
          if (reqSeq !== currentSeq) { return; }
          if (opts.tbodyEl && typeof opts.tbodyEl.setAttribute === "function") {
            opts.tbodyEl.setAttribute("aria-busy", "false");
          }
          // 失败状态处理：显示错误及重试，绝不能伪装为 0 结果空态（E05）
          if (opts.tbodyEl) { opts.tbodyEl.innerHTML = ""; }
          if (opts.emptyEl) { opts.emptyEl.hidden = true; }
          if (opts.errorEl) {
            opts.errorEl.hidden = false;
            var msgSpan = opts.errorEl.querySelector("span");
            if (msgSpan) {
              if (err && err.isAuth) {
                msgSpan.innerHTML = '登录已过期，请 <a href="/login" class="table-action-btn primary" style="display:inline-block;vertical-align:middle;margin:0 4px;">重新登录</a>';
              } else {
                msgSpan.textContent = err.message || "加载失败";
              }
            }
          }
          if (opts.summaryEl) { opts.summaryEl.textContent = "加载失败"; }
          if (typeof opts.onError === "function") {
            opts.onError(err);
          }
          if (typeof onDone === "function") { onDone(err, null); }
        });
    }

    return {
      fetch: execute,
      getSeq: function () { return currentSeq; }
    };
  }

  /**
   * renderPagination: 统一渲染最多 7 个槽位（含省略号）的分页栏。
   * options: {
   *   container: HTMLElement,
   *   page: number,
   *   pageSize: number,
   *   total: number,
   *   onPageChange: function(newPage),
   *   onPageSizeChange: function(newPageSize)
   * }
   */
  function renderPagination(opts) {
    var host = opts.container;
    if (!host) { return; }
    var total = opts.total || 0;
    var pageSize = opts.pageSize || 15;
    var page = opts.page || 1;
    var pages = Math.max(1, Math.ceil(total / pageSize));

    if (total === 0) {
      host.hidden = true;
      host.innerHTML = "";
      return;
    }
    host.hidden = false;

    var start = (page - 1) * pageSize + 1;
    var end = Math.min(total, page * pageSize);

    var html = '<div class="pager-meta"><span>显示 ' + start + "–" + end + "，共 " + total + " 条</span></div>";
    html += '<div class="pager-controls">';
    html += '<select class="pager-size-select" aria-label="每页条数">';
    [10, 15, 20, 30, 50].forEach(function (size) {
      html += '<option value="' + size + '"' + (size === pageSize ? " selected" : "") + ">" + size + " 条/页</option>";
    });
    html += "</select>";

    // 上一页
    html += '<button type="button" class="pager-btn" data-page="' + (page - 1) + '"' + (page <= 1 ? " disabled" : "") + ' aria-label="上一页">‹</button>';

    // 页码窗口计算 (最多 7 个页码按钮/省略号)
    if (pages <= 7) {
      for (var p = 1; p <= pages; p++) {
        html += '<button type="button" class="pager-btn' + (p === page ? " active" : "") + '" data-page="' + p + '"' + (p === page ? ' aria-current="page"' : "") + ">" + p + "</button>";
      }
    } else {
      // 首尾固定 + 当前页邻近
      html += '<button type="button" class="pager-btn' + (1 === page ? " active" : "") + '" data-page="1"' + (1 === page ? ' aria-current="page"' : "") + ">1</button>";
      if (page > 4) {
        html += '<span class="pager-ellipsis">…</span>';
      }
      var pStart = Math.max(2, page - 1);
      var pEnd = Math.min(pages - 1, page + 1);
      if (page <= 4) { pStart = 2; pEnd = 5; }
      if (page >= pages - 3) { pStart = pages - 4; pEnd = pages - 1; }
      for (var i = pStart; i <= pEnd; i++) {
        html += '<button type="button" class="pager-btn' + (i === page ? " active" : "") + '" data-page="' + i + '"' + (i === page ? ' aria-current="page"' : "") + ">" + i + "</button>";
      }
      if (page < pages - 3) {
        html += '<span class="pager-ellipsis">…</span>';
      }
      html += '<button type="button" class="pager-btn' + (pages === page ? " active" : "") + '" data-page="' + pages + '"' + (pages === page ? ' aria-current="page"' : "") + ">" + pages + "</button>";
    }

    // 下一页
    html += '<button type="button" class="pager-btn" data-page="' + (page + 1) + '"' + (page >= pages ? " disabled" : "") + " aria-label=\"下一页\">›</button>";
    html += "</div>";

    host.innerHTML = html;

    // 事件委托绑定
    var sel = host.querySelector(".pager-size-select");
    if (sel) {
      sel.addEventListener("change", function (e) {
        if (typeof opts.onPageSizeChange === "function") {
          opts.onPageSizeChange(Number(e.target.value));
        }
      });
    }
    host.querySelectorAll(".pager-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        if (btn.disabled || btn.classList.contains("active")) { return; }
        var targetPage = Number(btn.dataset.page);
        if (targetPage >= 1 && targetPage <= pages && typeof opts.onPageChange === "function") {
          opts.onPageChange(targetPage);
        }
      });
    });
  }

  window.PersonalList = {
    escapeHtml: escapeHtml,
    createController: createController,
    renderPagination: renderPagination
  };

  if (typeof module !== "undefined" && module.exports) {
    module.exports = window.PersonalList;
  }
})();
