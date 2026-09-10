/*
 * 文件: web/static/js/po/homereview.js
 * 模块: PO工作台
 * 职责: 在工作台抽屉展示上下文并提交阶段动作。
 */
(function ($) {
  "use strict";

  var MODAL_IDS = ["poDemandReviewModal", "poDemandReviewOverlay"];
  var ACCEPTANCE_IDS = ["poDemandAcceptanceModal", "poDemandAcceptanceOverlay"];
  var ACTION_IDS = ["poActionDrawer", "poActionDrawerOverlay"];

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  var currentItem = null;
  var currentAction = null;

  // 列表 id 是展示号 US{主键}，接口路径要用数字主键。
  function closeModal() {
    if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
      window.DemandDetail.close();
    }
  }

  function openModal(item) {
    var id = String((item && item.id) || "").trim();
    if (id && window.DemandDetail && typeof window.DemandDetail.open === "function") {
      window.DemandDetail.open(id, { mode: "review" });
    }
  }

  function bindModal() {
    function actionEndpoint(key, id) {
      var routes = { clarify: "clarify", accept: "acceptance", remind_accept: "urge", deliver: "deliver" };
      return routes[key] && id ? "/demands/" + encodeURIComponent(id) + "/" + routes[key] : "";
    }

    function fillActionContext(item, key) {
      var id = String((item && item.id) || "").replace(/^US/i, "");
      var title = String((item && item.title) || "").trim();
      var stage = String((item && (item.valueStream || item.stage)) || "—");
      var owner = String((item && (item.nextOwner || item.owner)) || "—");
      var summary = String((item && (item.summary || item.description || item.title)) || "—").trim();
      var labels = { clarify: "澄清", schedule: "排期", submit_test: "提测", accept: "验收", remind_accept: "催办验收", deliver: "发起交付" };
      $("#poActionDrawerTitle").text(labels[key] || "办理");
      $("#poActionDrawerContext").text("需求 US" + id + " · " + title);
      $("#poActionDrawerStage").text(stage);
      $("#poActionDrawerOwner").text(owner);
      $("#poActionDrawerSummary").text(summary);
      return { id: id, title: title, stage: stage, owner: owner, summary: summary };
    }

    $("#poDemandAcceptanceForm").on("submit", function (event) {
      event.preventDefault();
      if (!currentAction || !currentAction.id) { showToast("需求信息不可用", "error"); return; }
      var form = this, button = $(form).find("button[type=submit]");
      button.prop("disabled", true).text("提交中…");
      (window.appFetch || window.fetch)("/demands/" + encodeURIComponent(currentAction.id) + "/acceptance", { method: "POST", headers: { "Content-Type": "application/json", "Accept": "application/json" }, body: JSON.stringify({ comment: form.comment.value }) })
        .then(function (res) { return res.json().then(function (data) { if (!res.ok) { throw new Error(data.message || "验收失败"); } return data; }); })
        .then(function (data) { showToast(data.message || "验收成功", "success"); form.reset(); if (typeof window.closeShowModals === "function") { window.closeShowModals(ACCEPTANCE_IDS); } return typeof window.refreshPoHomeDemands === "function" ? window.refreshPoHomeDemands() : null; })
        .catch(function (err) { showToast(err.message || "验收失败", "error"); })
        .then(function () { button.prop("disabled", false).text("确认验收"); });
    });

    $(document).on("click", ".js-po-drawer-action[data-action-key='accept']", function () {
      var btn = $(this), url = String(btn.attr("data-action-url") || "");
      var id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      currentAction = { id: id, key: "accept", url: url, title: String(btn.closest("tr").find("td").eq(1).text() || "").trim(), stage: "验收" };
      $("#poDemandAcceptanceContext").text("需求 US" + (id || "—") + " · " + currentAction.title);
      $("#poAcceptanceCases, #poAcceptanceBugs").text("加载中");
      if (id) {
        fetch("/demands/" + encodeURIComponent(id) + "/detail", { method: "GET", headers: { "Accept": "application/json" } }).then(function (res) { return res.ok ? res.json() : null; }).then(function (data) {
          var e = data && (data.execution || data.detail || data);
          var c = e && (e.testCaseSummary || e.testCases || e.summary) || {};
          var b = e && (e.bugSummary || e.bugs || e.summary) || {};
          $("#poAcceptanceCases").text((c.executedCount != null ? c.executedCount : (c.casesExecuted != null ? c.casesExecuted : "—")) + " / " + (c.totalCount != null ? c.totalCount : (c.casesTotal != null ? c.casesTotal : "—")));
          $("#poAcceptanceBugs").text(b.activeCount != null ? b.activeCount : (b.bugsUnresolved != null ? b.bugsUnresolved : "—"));
        }).catch(function () { $("#poAcceptanceCases, #poAcceptanceBugs").text("暂不可用"); });
      }
      if (typeof window.openShowModals === "function") { window.openShowModals(ACCEPTANCE_IDS); }
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='approve']", function () {
      var btn = $(this), id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      var row = btn.closest("tr");
      openModal({
        id: id ? "US" + id : "",
        title: String(row.find("td").eq(1).text() || "").trim(),
        stage: "受理",
        owner: String(row.find("td").eq(5).text() || "").trim()
      });
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='withdraw_review'], .js-po-drawer-action[data-action-key='submit_review']", function () {
      var btn = $(this), id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      var row = btn.closest("tr");
      openModal({
        id: id ? "US" + id : "",
        title: String(row.find("td").eq(1).text() || "").trim(),
        stage: "评审",
        owner: String(row.find("td").eq(5).text() || "").trim()
      });
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='clarify']", function () {
      var btn = $(this), id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      if (typeof window.openPoDemandClarifyModal === "function") {
        window.openPoDemandClarifyModal(id);
      }
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='deliver']", function (e) {
      e.preventDefault();
      var btn = $(this), id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      if (typeof window.openPoDeliverModal === "function") {
        window.openPoDeliverModal(id);
      }
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='submit_test'], .js-submit-test", function (e) {
      e.preventDefault();
      var btn = $(this), id = String(btn.attr("data-demand-id") || "").replace(/^US/i, "");
      var row = btn.closest("tr");
      var title = String(row.find("td").eq(1).text() || "").trim();
      var stage = String(row.find("td").eq(3).text() || "").trim();
      var item = { id: id ? "US" + id : "", title: title, stage: stage };
      if (typeof window.openPoSubmitTestModal === "function") {
        window.openPoSubmitTestModal(item);
      }
    });
    $("#poDemandAcceptanceCloseBtn, #poDemandAcceptanceOverlay").on("click", function () {
      if (typeof window.closeShowModals === "function") { window.closeShowModals(ACCEPTANCE_IDS); }
    });
    $(document).on("click", ".js-po-drawer-action:not([data-action-key='accept']):not([data-action-key='approve']):not([data-action-key='withdraw_review']):not([data-action-key='submit_review']):not([data-action-key='clarify']):not([data-action-key='submit_test']):not([data-action-key='deliver'])", function () {
      var btn = $(this), url = String(btn.attr("data-action-url") || "");
      var key = String(btn.attr("data-action-key") || "");
      var ctx = fillActionContext({ id: btn.attr("data-demand-id"), title: String(btn.closest("tr").find("td").eq(1).text() || "").trim(), valueStream: String(btn.closest("tr").find("td").eq(3).text() || "").trim(), owner: String(btn.closest("tr").find("td").eq(5).text() || "").trim() }, key);
      currentAction = { id: ctx.id, key: key, url: url, title: ctx.title, stage: ctx.stage, owner: ctx.owner, summary: ctx.summary };
      var endpoint = actionEndpoint(key, ctx.id);
      $("#poActionForm").toggle(!!endpoint);
      $("#poActionForm")[0].reset();
      $("#poActionOpenPage").toggle(!endpoint && !!url).attr("href", url || "#");
      $("#poActionDrawerUnavailable").toggle(!endpoint && !url);
      if (typeof window.openShowModals === "function") { window.openShowModals(ACTION_IDS); }
    });
    $("#poActionForm").on("submit", function (event) {
      event.preventDefault();
      if (!currentAction || !currentAction.id) { showToast("需求信息不可用", "error"); return; }
      var form = this, button = $(form).find("button[type=submit]"), endpoint = actionEndpoint(currentAction.key, currentAction.id);
      if (!endpoint) { showToast("该办理页面暂不可用", "error"); return; }
      button.prop("disabled", true).text("提交中…");
      (window.appFetch || window.fetch)(endpoint, { method: "POST", headers: { "Content-Type": "application/json", "Accept": "application/json" }, body: JSON.stringify({ comment: form.comment.value }) })
        .then(function (res) { return res.json().then(function (data) { if (!res.ok) { throw new Error(data.message || "办理失败"); } return data; }); })
        .then(function (data) { showToast(data.message || "办理成功", "success"); form.reset(); if (typeof window.closeShowModals === "function") { window.closeShowModals(ACTION_IDS); } return typeof window.refreshPoHomeDemands === "function" ? window.refreshPoHomeDemands() : null; })
        .catch(function (err) { showToast(err.message || "办理失败", "error"); })
        .then(function () { button.prop("disabled", false).text("确认办理"); });
    });
    $("#poActionDrawerCloseBtn, #poActionDrawerOverlay").on("click", function () {
      if (typeof window.closeShowModals === "function") { window.closeShowModals(ACTION_IDS); }
    });
  }

  window.openPoDemandReviewModal = openModal;

  $(bindModal);
})(jQuery);
