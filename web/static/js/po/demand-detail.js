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
    if (!R) return;

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

    // Body by tab
    var bodyEl = $("ddBody");
    if (!bodyEl) return;

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

  function open(demandId) {
    if (!demandId) return;
    currentDemandId = demandId;
    currentTab = "overview";

    var drawer = getDrawer();
    drawer.classList.add("active");

    var bodyEl = $("ddBody");
    if (bodyEl) {
      bodyEl.innerHTML = '<div style="text-align:center;padding:60px 0;color:#8a99ad;font-size:13px;">正在加载需求详情...</div>';
    }

    var cleanId = String(demandId).replace(/^US/i, "");
    fetch("/demands/" + cleanId + "/detail", {
      headers: { "Accept": "application/json" }
    })
      .then(function (res) {
        if (!res.ok) throw new Error("加载需求详情失败 (" + res.status + ")");
        return res.json();
      })
      .then(function (data) {
        if (!data || !data.success) {
          throw new Error((data && data.message) || "需求详情加载异常");
        }
        currentData = data;
        renderContent();
      })
      .catch(function (err) {
        if (bodyEl) {
          bodyEl.innerHTML = [
            '<div style="text-align:center;padding:40px 0;">',
            '  <div style="color:#d32f2f;font-weight:600;margin-bottom:8px;">' + (err.message || "加载失败") + '</div>',
            '  <button class="dd-btn" onclick="DemandDetail.open(' + demandId + ')">点击重试</button>',
            '</div>'
          ].join("");
        }
      });
  }

  function close() {
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

  // 全局事件委托：捕获页面中的业务需求卡片与链接点击
  document.addEventListener("click", function (e) {
    var trigger = e.target.closest("[data-demand-id], [data-open-demand-detail]");
    if (trigger) {
      var did = trigger.getAttribute("data-demand-id") || trigger.getAttribute("data-open-demand-detail");
      if (did && (/^US\d+/i.test(did) || /^\d+$/.test(did))) {
        e.preventDefault();
        open(did);
        return;
      }
    }

    var node = e.target.closest(".demand-grid .node-title-line");
    if (node) {
      var codeEl = node.querySelector(".code");
      var raw = codeEl ? (codeEl.textContent || "").trim() : "";
      if (/^US\d+/i.test(raw) || /^\d+$/.test(raw)) {
        e.preventDefault();
        open(raw);
        return;
      }
    }
  });

  window.DemandDetail = {
    open: open,
    close: close,
    switchTab: switchTab,
    renderContent: renderContent
  };
})();
