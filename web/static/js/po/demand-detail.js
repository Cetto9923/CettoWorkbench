// =============================================================================
// 文件: web/static/js/po/demand-detail.js
// 模块: PO 工作台
// 职责: 业务需求统一详情（Drawer 抽屉与独立页面交互控制器）。
// =============================================================================

(function () {
  "use strict";

  var currentData = null;
  var currentTab = "overview";
  var currentDemandId = null;
  var currentMode = "normal"; // normal | review
  var requestSeq = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function getDrawer() {
    var drawer = $("demandDetailDrawer");
    if (!drawer) {
      drawer = document.createElement("div");
      drawer.id = "demandDetailDrawer";
      drawer.className = "dd-drawer-mask";
      drawer.innerHTML = [
        '<div class="dd-drawer-panel" onclick="event.stopPropagation()">',
        '  <header class="dd-head" id="ddHead"></header>',
        '  <nav id="ddRelationNav"></nav>',
        '  <nav class="dd-tabs" id="ddTabs">',
        '    <button class="dd-tab active" data-tab="overview" onclick="DemandDetail.switchTab(\'overview\')">整体概览</button>',
        '    <button class="dd-tab" data-tab="requirement" onclick="DemandDetail.switchTab(\'requirement\')">需求与澄清 <span class="dd-tab-count" id="ddTabCountReq"></span></button>',
        '    <button class="dd-tab" data-tab="execution" onclick="DemandDetail.switchTab(\'execution\')">研发与测试 <span class="dd-tab-count" id="ddTabCountExec"></span></button>',
        '    <button class="dd-tab" data-tab="delivery" onclick="DemandDetail.switchTab(\'delivery\')">交付上线</button>',
        '    <button class="dd-tab" data-tab="history" onclick="DemandDetail.switchTab(\'history\')">过程记录</button>',
        '  </nav>',
        '  <main class="dd-body" id="ddBody"></main>',
        '</div>'
      ].join("");

      drawer.onclick = function () {
        DemandDetail.close();
      };

      document.body.appendChild(drawer);
    }
    return drawer;
  }

  function renderContent() {
    if (!currentData) return;
    var R = window.DemandDetailRender;
    if (!R) throw new Error("详情组件未加载，请刷新页面重试");

    // Header
    var headEl = $("ddHead");
    if (headEl) {
      headEl.innerHTML = R.renderHeader(currentData.summary, currentData.mode);
    }

    // Relation Nav
    var relEl = $("ddRelationNav");
    if (relEl) {
      relEl.innerHTML = R.renderRelationNav(currentData.relationContext, currentData.summary.demandId);
    }

    var tabsEl = $("ddTabs");
    var bodyEl = $("ddBody");
    if (!bodyEl) return;

    var summaryStatus = String((currentData.summary && (currentData.summary.status || currentData.summary.zentaoStatus)) || "").trim().toLowerCase();
    var isEarlyStage = (summaryStatus === "wait" || summaryStatus === "draft" || summaryStatus === "refuse");

    // 评审阶段做减法视图：待评审、草稿、驳回状态下统一使用精简视图（隐藏 5 个 Tab）
    if ((currentMode === "review" || isEarlyStage) && window.DemandDetailReview) {
      if (tabsEl) tabsEl.style.display = "none";
      bodyEl.innerHTML = window.DemandDetailReview.renderReviewView(currentData);
      return;
    }

    if (tabsEl) tabsEl.style.display = "";

    // Tab counts
    var reqCount = $("ddTabCountReq");
    if (reqCount && currentData.requirement && currentData.requirement.clarifications) {
      reqCount.textContent = currentData.requirement.clarifications.length || "";
      reqCount.hidden = !currentData.requirement.clarifications.length;
    }
    var execCount = $("ddTabCountExec");
    if (execCount && currentData.execution && currentData.execution.stories) {
      execCount.textContent = currentData.execution.stories.length || "";
      execCount.hidden = !currentData.execution.stories.length;
    }

    if (currentData.mode === "parentAggregate" && currentTab === "overview") {
      bodyEl.innerHTML = R.renderParentAggregate(currentData.parentAggregate);
      return;
    }

    switch (currentTab) {
      case "overview":
        bodyEl.innerHTML = R.renderTabOverview(currentData);
        break;
      case "requirement":
        bodyEl.innerHTML = R.renderTabRequirement(currentData.requirement);
        break;
      case "execution":
        bodyEl.innerHTML = R.renderTabExecution(currentData.execution);
        break;
      case "delivery":
        bodyEl.innerHTML = R.renderTabDelivery(currentData.delivery);
        break;
      case "history":
        bodyEl.innerHTML = R.renderTabHistory(currentData.history);
        break;
      default:
        bodyEl.innerHTML = R.renderTabOverview(currentData);
        break;
    }
  }

  function open(demandId, options) {
    var cleanId = String(demandId || "").replace(/^US/i, "");
    if (!/^\d+$/.test(cleanId) || Number(cleanId) <= 0) return;
    var seq = ++requestSeq;
    currentDemandId = demandId;
    currentTab = "overview";
    currentData = null;
    currentMode = (options && options.mode) === "review" ? "review" : "normal";

    var drawer = getDrawer();
    drawer.classList.add("active");
    $("ddHead").innerHTML = '<span>需求详情</span><button type="button" class="ui-close-btn" aria-label="关闭" onclick="DemandDetail.close()">×</button>';
    $("ddRelationNav").innerHTML = "";
    var tabsEl = $("ddTabs");
    if (tabsEl) tabsEl.style.display = currentMode === "review" ? "none" : "";
    drawer.querySelectorAll(".dd-tab").forEach(function (tab) { tab.classList.toggle("active", tab.getAttribute("data-tab") === "overview"); });
    ["ddTabCountReq", "ddTabCountExec"].forEach(function (id) { $(id).textContent = ""; $(id).hidden = true; });

    var bodyEl = $("ddBody");
    if (bodyEl) {
      bodyEl.innerHTML = '<div style="text-align:center;padding:60px 0;color:#8a99ad;font-size:13px;">正在加载需求详情...</div>';
    }

    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + cleanId + "/detail", {
      method: "GET",
      credentials: "same-origin",
      headers: { "Accept": "application/json" }
    })
      .then(function (res) {
        if (res.status === 401) {
          var authErr = new Error("登录已过期，请重新登录");
          authErr.isAuth = true;
          throw authErr;
        }
        if (res.status === 403) {
          throw new Error("无权查看该业务需求");
        }
        if (res.status === 404) {
          throw new Error("需求不存在");
        }
        if (!res.ok) throw new Error("加载需求详情失败 (" + res.status + ")");
        return res.json();
      })
      .then(function (data) {
        if (seq !== requestSeq) return;
        if (!data || !data.success) {
          throw new Error((data && (data.message || data.error)) || "需求详情加载异常");
        }
        currentData = data;
        renderContent();
      })
      .catch(function (err) {
        if (seq !== requestSeq) return;
        if (bodyEl) {
          var actions = err && err.isAuth
            ? '<a class="dd-btn" href="/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search) + '">去登录</a>'
            : '<button class="dd-btn" onclick="DemandDetail.open(' + cleanId + ')">点击重试</button>';
          bodyEl.innerHTML = [
            '<div style="text-align:center;padding:40px 0;">',
            '  <div class="dd-error-message"></div>',
            '  ' + actions,
            '</div>'
          ].join("");
          bodyEl.querySelector(".dd-error-message").textContent = err.message || "加载失败";
        }
      });
  }

  function close() {
    requestSeq++;
    currentData = null;
    var drawer = $("demandDetailDrawer");
    if (drawer) {
      drawer.classList.remove("active");
    }
  }

  function switchTab(tabKey, targetSection) {
    currentTab = tabKey || "overview";

    var tabs = document.querySelectorAll("#ddTabs .dd-tab");
    for (var i = 0; i < tabs.length; i++) {
      var t = tabs[i];
      if (t.getAttribute("data-tab") === currentTab) {
        t.classList.add("active");
      } else {
        t.classList.remove("active");
      }
    }

    renderContent();

    if (targetSection) {
      setTimeout(function () {
        var el = $(targetSection);
        if (el && typeof el.scrollIntoView === "function") {
          el.scrollIntoView({ behavior: "smooth", block: "start" });
        }
      }, 50);
    }
  }

  // 绑定全局键盘 Esc 关闭
  window.addEventListener("keydown", function (e) {
    if (e.key === "Escape" || e.keyCode === 27) {
      close();
    }
  });

  // 需求详情链接使用同一个抽屉；禅道原文和办理链接保留正常导航。
  document.addEventListener("click", function (e) {
    if (e.target.closest(".js-po-drawer-action, [data-action-key], .table-action-btn, .js-demand-review, [data-review-demand-id]") && !e.target.closest("[data-open-demand-detail]")) {
      return;
    }

    var clarifyBtn = e.target.closest(".js-drawer-clarify-btn");
    if (clarifyBtn) {
      e.preventDefault();
      var cDid = clarifyBtn.getAttribute("data-demand-id");
      if (cDid && typeof window.openPoDemandClarifyModal === "function") {
        window.openPoDemandClarifyModal(cDid);
      }
      return;
    }

    var trigger = e.target.closest("[data-demand-id], [data-open-demand-detail], a[href^='/demands/']");
    if (trigger) {
      var tag = (trigger.tagName || "").toUpperCase();
      var href = tag === "A" ? trigger.getAttribute("href") : "";
      var detailMatch = href && href.match(/^\/demands\/((?:US)?\d+)(?:[?#].*)?$/i);
      if (detailMatch && (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey)) { return; }
      if (tag === "A" && href && !detailMatch) { return; }
      var did = trigger.getAttribute("data-demand-id") || trigger.getAttribute("data-open-demand-detail");
      if (!did && detailMatch) { did = detailMatch[1]; }
      if (did && (/^US\d+/i.test(did) || /^\d+$/.test(did))) {
        e.preventDefault();
        open(did);
        return;
      }
    }

    // 看板标题行：仅业务/子需求（US…）打开详情抽屉。
    // 研需 displayId 为纯数字且行上带 data-rd，不可误打 /demands/:storyId/detail。
    var node = e.target.closest(".demand-grid .node-title-line");
    if (node) {
      var row = node.closest(".demand-row, .biz-collapsed-row, .biz-head, .biz-group");
      if (row && row.getAttribute("data-rd")) {
        return;
      }
      var codeEl = node.querySelector(".code");
      var raw = codeEl ? (codeEl.textContent || "").trim() : "";
      if (/^US\d+/i.test(raw)) {
        e.preventDefault();
        open(raw);
        return;
      }
    }
  });

  document.addEventListener("DOMContentLoaded", function () {
    var did = new URLSearchParams(window.location.search).get("openDemand");
    if (did && (/^US\d+$/i.test(did) || /^\d+$/.test(did))) { open(did); }
  });

  window.DemandDetail = {
    open: open,
    close: close,
    switchTab: switchTab,
    renderContent: renderContent,
    getCurrentDemandId: function () {
      return currentDemandId;
    }
  };
})();
