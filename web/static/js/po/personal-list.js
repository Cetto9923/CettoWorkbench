/* =============================================================================
   文件: web/static/js/po/personal-list.js
   模块: PO 个人工作台 (UI-02 列表请求与分页共享原语)
   职责: 提供三列表（/todos, /done, /notice）有界且纯粹的 JSON 异步请求时序保护、
         统一状态（loading/empty/error）切换与统一分页组件渲染。
   禁止: 框架化、通用列表引擎、跨模块数据总线或业务逻辑接管。
   ============================================================================= */

(function () {
  "use strict";

  // 每页条数选项：与 components/pager.html 的 <select name="pageSize"> 保持同一套取值。
  var PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

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
          if (typeof onDone === "function") {
            if (onDone.length >= 2) {
              onDone(null, payload);
            } else {
              onDone(payload);
            }
          }
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
          if (typeof onDone === "function") {
            if (onDone.length >= 2) {
              onDone(err, null);
            } else {
              onDone(null);
            }
          }
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
    var pageSize = opts.pageSize || 20;
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
    PAGE_SIZE_OPTIONS.forEach(function (size) {
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
    html += '<span class="pager-jump-label">跳至</span><input type="number" class="pager-jump-input" min="1" max="' + pages + '" value="' + page + '" aria-label="跳至页码"><span class="pager-jump-label">页</span><button type="button" class="pager-jump-btn">前往</button>';
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

    var jumpInput = host.querySelector(".pager-jump-input");
    var jumpBtn = host.querySelector(".pager-jump-btn");
    function jumpToPage() {
      if (!jumpInput) { return; }
      var t = Number(jumpInput.value);
      if (t >= 1 && t <= pages && t !== page && typeof opts.onPageChange === "function") { opts.onPageChange(t); }
    }
    if (jumpBtn) { jumpBtn.addEventListener("click", jumpToPage); }
    if (jumpInput) { jumpInput.addEventListener("keydown", function (e) { if (e.key === "Enter") { e.preventDefault(); jumpToPage(); } }); }
  }

  /**
   * loadPageSize / savePageSize:
   * 让每个列表页（/todos /done /notice /home /follow 等）的每页条数
   * 在刷新页面和下次登录后保持一致。
   * key 是命名空间前缀（如 "po.todos.pageSize"）；allowed 是允许值白名单；
   * 不在白名单或解析失败时返回 fallback。
   */
  function loadPageSize(key, fallback, allowed) {
    try {
      var raw = window.localStorage.getItem(key);
      var n = parseInt(raw, 10);
      if (!Array.isArray(allowed)) { allowed = null; }
      if (!isNaN(n) && (!allowed || allowed.indexOf(n) >= 0)) { return n; }
    } catch (e) { /* localStorage 不可用 */ }
    return fallback;
  }
  function savePageSize(key, value) {
    try { window.localStorage.setItem(key, String(value)); } catch (e) { /* 静默忽略 */ }
  }

  /**
   * Stage 3 公共优先级 / 对象类型 helpers。
   * 入口 (handlers/services) 输出 P1–P4 字符串值 ("1","2","3","4") 或数字; 也可能
   * 返回 null/undefined/"" / 越界值。规范:
   *   - normalizePriority(raw) -> 1..4 数字 / null (无默认 fallback)
   *   - priorityBadge(raw)    -> <span class="wb-priority" data-priority="N">P{N}</span>
   *                              raw 无效时返回 <span class="wb-priority" data-priority="">—</span>
   *   - objectTypeBadge(kind) -> <span class="wb-type wb-type-{kind}">中文 label</span>
   *                              kind 未知时返回 wb-type-unknown。
   */
  function normalizePriority(raw) {
    if (raw === null || raw === undefined) { return null; }
    var s = String(raw).trim();
    if (!s) { return null; }
    var stripped = s.replace(/^p/i, "");
    var n = parseInt(stripped, 10);
    if (isNaN(n)) { return null; }
    if (n < 1 || n > 4) { return null; }
    return n;
  }

  function priorityBadge(raw) {
    var n = normalizePriority(raw);
    if (n === null) {
      return '<span class="wb-priority" data-priority="">—</span>';
    }
    return '<span class="wb-priority" data-priority="' + n + '">P' + n + "</span>";
  }

  var OBJECT_TYPE_LABELS = {
    business: "业务需求",
    sub_demand: "子需求",
    story: "研发需求",
    independent_story: "独立研需",
    task: "任务",
    issue: "问题",
    bug: "Bug",
    approval: "审批",
    todo: "待办",
    testtask: "测试单",
    charter: "章程",
    feedback: "反馈",
    project: "项目",
    mail: "邮件",
    risk: "风险"
  };

  /* chip 内「对象 #ID」单色标签使用的缩写版标签。
     只在 chip 上下文使用；独立 badge / tab 标题仍走 OBJECT_TYPE_LABELS 全称。 */
  var OBJECT_TYPE_SHORT_LABELS = {
    business: "业需",
    sub_demand: "子需",
    story: "研需",
    independent_story: "独立研需",
    task: "任务",
    issue: "问题",
    bug: "Bug",
    approval: "审批",
    todo: "待办",
    testtask: "测单",
    charter: "章程",
    feedback: "反馈",
    project: "项目",
    mail: "邮件",
    risk: "风险"
  };

  /* 待办/列表 API 的 kind 字段 → objectTypeBadge 的 canonical key */
  var OBJECT_KIND_FROM_API = {
    demand: "business",
    business: "business",
    sub_demand: "sub_demand",
    story: "story",
    independent_story: "independent_story",
    task: "task",
    bug: "bug",
    issue: "issue",
    approval: "approval",
    todo: "todo",
    test: "testtask",
    testtask: "testtask",
    charter: "charter",
    feedback: "feedback",
    project: "project",
    mail: "mail",
    risk: "risk"
  };

  var REVERSE_CHINESE_MAP = {
    "业务需求": "business", "业需": "business",
    "子需求": "sub_demand", "子需": "sub_demand",
    "研发需求": "story", "研需": "story", "独立研需": "independent_story",
    "任务": "task", "问题": "issue", "缺陷": "bug", "bug": "bug",
    "审批": "approval", "待办": "todo",
    "测试单": "testtask", "测单": "testtask", "测试": "testtask",
    "章程": "charter", "反馈": "feedback", "项目": "project",
    "邮件": "mail", "风险": "risk"
  };

  function normalizeKind(kind) {
    var raw = String(kind || "").trim();
    var lower = raw.toLowerCase();
    return OBJECT_KIND_FROM_API[lower] || REVERSE_CHINESE_MAP[raw] || REVERSE_CHINESE_MAP[lower] || lower;
  }

  function objectTypeBadgeFromKind(kind) {
    return objectTypeBadge(normalizeKind(kind));
  }

  function objectTypeBadge(kind) {
    var k = normalizeKind(kind);
    if (!k) { return '<span class="wb-type wb-type-unknown">—</span>'; }
    if (OBJECT_TYPE_LABELS[k]) {
      return '<span class="wb-type wb-type-' + k + '">' + escapeHtml(OBJECT_TYPE_LABELS[k]) + "</span>";
    }
    return '<span class="wb-type wb-type-unknown">' + escapeHtml(k) + "</span>";
  }

  function formatChipId(safeId) {
    if (!safeId) { return ""; }
    if (safeId.charAt(0) === "#") { return safeId; }
    if (/^<([a-zA-Z0-9]+)\b([^>]*)>(\s*[-—]+\s*)<\/\1>$/i.test(safeId) || /^\s*[-—]+\s*$/.test(safeId)) {
      return safeId;
    }
    if (/^<([a-zA-Z0-9]+)\b[^>]*>\s*#/.test(safeId)) {
      return safeId;
    }
    if (/^<([a-zA-Z0-9]+)\b([^>]*)>(.*)$/s.test(safeId)) {
      return safeId.replace(/^<([a-zA-Z0-9]+)\b([^>]*)>(.*)$/s, "<$1$2>#$3");
    }
    return "#" + safeId;
  }

  /**
   * idChipHtml: 渲染「对象 #ID」双段式工程微标（Option B: Linear Two-tone Badge）。
   *   - kind   : canon key / API 字段 / 中文类型，统一由 normalizeKind 解析
   *   - idHtml : 已经构造好的 ID HTML（典型为 <a class="table-id-link"> 或 <span>）。
   *   左段 .wb-type-tag 包裹缩写类型（白字实底），右段 .wb-type-id 包裹带 # 编号（白底深色等宽）。
   */
  function idChipHtml(kind, idHtml) {
    var k = normalizeKind(kind);
    var safeId = typeof idHtml === "string" ? idHtml.trim() : (idHtml != null ? String(idHtml).trim() : "");
    var label = OBJECT_TYPE_SHORT_LABELS[k] || OBJECT_TYPE_LABELS[k] || k || "—";
    var cls = k && OBJECT_TYPE_LABELS[k] ? ("wb-type-" + k) : "wb-type-unknown";
    if (!safeId) {
      return '<span class="wb-type ' + cls + '"><span class="wb-type-tag">' + escapeHtml(label) + "</span></span>";
    }
    return '<span class="wb-type ' + cls + '">' +
      '<span class="wb-type-tag">' + escapeHtml(label) + "</span>" +
      '<span class="wb-type-id">' + formatChipId(safeId) + "</span>" +
      "</span>";
  }

  // 系统级模板提醒兜底：「提醒：您有 Bug(9)」「您有 Task(3)」「您有 需求(2)」
  // 这类 subject 不符合 TYPE #ID 形态，后端 objType 可能仍为空。命中时返回
  // 一个可渲染 notice-tag 的 canon kind；subject 文本保持原样。
  var REMINDER_KIND_ALIASES = {
    bug: "bug", task: "task",
    story: "story", demand: "business", issue: "issue",
    feedback: "feedback", charter: "charter", project: "project",
    testtask: "testtask", risk: "risk",
    "研发需求": "story", "业务需求": "business", "需求": "business",
    "任务": "task", "缺陷": "bug", "测试": "testtask", "问题": "issue",
    "风险": "risk", "反馈": "feedback", "章程": "charter", "项目": "project"
  };
  var REMINDER_PATTERN = /(?:Bug|Task|Story|Demand|Issue|Feedback|Charter|Project|TestTask|Risk|研发需求|业务需求|需求|任务|缺陷|测试|问题|风险|反馈|章程|立项|项目)\s*[\(（]\s*\d+\s*[\)）]/i;
  function reminderKindFromSubject(subject) {
    var text = String(subject || "");
    if (!text) { return ""; }
    var m = text.match(REMINDER_PATTERN);
    if (!m) { return ""; }
    var token = String(m[0]).replace(/[\(（]\s*\d+\s*[\)）]/, "").trim();
    return REMINDER_KIND_ALIASES[token] || REMINDER_KIND_ALIASES[token.toLowerCase()] || "";
  }

  /**
   * statusTagHtml: 统一渲染状态微标（微圆点 + 语义文字）。
   * 自动按关键词识别五大语义：success, processing, warning, danger, neutral。
   */
  function statusTagHtml(text) {
    var raw = String(text == null ? "" : text).trim();
    if (!raw || raw === "—" || raw === "--") {
      return '<span class="wb-status-tag wb-status-neutral">—</span>';
    }
    var lower = raw.toLowerCase();
    var semantic = "neutral";
    if (lower.indexOf("关闭") >= 0 || lower.indexOf("closed") >= 0 || lower.indexOf("暂存") >= 0 || lower.indexOf("草稿") >= 0 || lower.indexOf("draft") >= 0 || lower.indexOf("取消") >= 0) {
      semantic = "neutral";
    } else if (lower.indexOf("澄清") >= 0 || lower.indexOf("完成") >= 0 || lower.indexOf("done") >= 0 || lower.indexOf("验收") >= 0 || lower.indexOf("发布") >= 0 || lower.indexOf("通过") >= 0 || lower.indexOf("正常") >= 0 || lower.indexOf("已解决") >= 0 || lower.indexOf("已闭环") >= 0 || lower.indexOf("激活") >= 0) {
      semantic = "success";
    } else if (lower.indexOf("待") >= 0 || lower.indexOf("wait") >= 0 || lower.indexOf("排期") >= 0 || lower.indexOf("评审中") >= 0 || lower.indexOf("审批中") >= 0 || lower.indexOf("预警") >= 0 || lower.indexOf("关注") >= 0) {
      semantic = "warning";
    } else if (lower.indexOf("挂起") >= 0 || lower.indexOf("驳回") >= 0 || lower.indexOf("阻塞") >= 0 || lower.indexOf("超期") >= 0 || lower.indexOf("逾期") >= 0 || lower.indexOf("失败") >= 0 || lower.indexOf("风险") >= 0 || lower.indexOf("异常") >= 0 || lower.indexOf("延期") >= 0) {
      semantic = "danger";
    } else if (lower.indexOf("开发") >= 0 || lower.indexOf("doing") >= 0 || lower.indexOf("测试") >= 0 || lower.indexOf("处理") >= 0 || lower.indexOf("进行") >= 0 || lower.indexOf("评审") >= 0 || lower.indexOf("active") >= 0) {
      semantic = "processing";
    }
    return '<span class="wb-status-tag wb-status-' + semantic + '"><i class="wb-status-dot"></i>' + escapeHtml(raw) + '</span>';
  }

  window.PersonalList = {
    escapeHtml: escapeHtml,
    createController: createController,
    renderPagination: renderPagination,
    loadPageSize: loadPageSize,
    savePageSize: savePageSize,
    PAGE_SIZE_OPTIONS: PAGE_SIZE_OPTIONS,
    normalizePriority: normalizePriority,
    priorityBadge: priorityBadge,
    objectTypeBadge: objectTypeBadge,
    objectTypeBadgeFromKind: objectTypeBadgeFromKind,
    idChipHtml: idChipHtml,
    statusTagHtml: statusTagHtml,
    reminderKindFromSubject: reminderKindFromSubject,
    OBJECT_TYPE_LABELS: OBJECT_TYPE_LABELS,
    OBJECT_TYPE_SHORT_LABELS: OBJECT_TYPE_SHORT_LABELS,
    OBJECT_KIND_FROM_API: OBJECT_KIND_FROM_API
  };

  if (typeof module !== "undefined" && module.exports) {
    module.exports = window.PersonalList;
  }
})();
