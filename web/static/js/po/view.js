/*
 * 文件: web/static/js/po/view.js
 * 模块: PO工作台
 * 职责: 业需评审详情右侧抽屉开关与列表行字段回填（静态演示，不接详情/提交 API）。
 */
(function ($) {
  "use strict";

  var DRAWER_ID = "poDemandViewDrawer";
  var REJECT_MODAL_ID = "poDemandViewRejectModal";
  var currentItem = null;

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function dash(value) {
    var text = String(value == null ? "" : value).trim();
    return text || "—";
  }

  function setText(id, value) {
    var el = document.getElementById(id);
    if (el) {
      el.textContent = dash(value);
    }
  }

  function categoryLabel(value) {
    var key = String(value || "").trim().toLowerCase();
    var labels = {
      experience: "体验优化",
      feature: "功能需求",
      request: "业务需求",
      business: "业务需求",
      research: "调研需求",
      bug: "BUG",
      tecopt: "技术优化",
      performance: "性能",
      safe: "安全",
      datacg: "数据变更",
      datachange: "数据变更",
      dataexport: "数据导出",
      other: "其他"
    };
    return labels[key] || value || "—";
  }

  function zentaoStatusLabel(value) {
    var raw = String(value || "").trim();
    var key = raw.toLowerCase();
    var labels = {
      developing: "开发中",
      testing: "测试中",
      wait: "待评审",
      draft: "草稿",
      active: "已评审",
      closed: "已关闭",
      canceled: "已取消",
      cancelled: "已取消",
      suspended: "已挂起",
      blocked: "已阻塞",
      done: "已完成",
      resolved: "已解决",
      verified: "已验证",
      reviewing: "评审中",
      changed: "已变更",
      postponed: "已延期",
      refuse: "已驳回"
    };
    return labels[key] || raw || "—";
  }

  function spotStageLabel(item) {
    var zt = String((item && item.zentaoStatus) || "").trim().toLowerCase();
    if (zt === "wait") {
      return "待受理";
    }
    return dash(item && (item.valueStream || item.stage || item.valueStageLabel || item.valueStage));
  }

  function fillFromItem(item) {
    currentItem = item || null;
    var id = dash(item && item.id);
    var title = dash(item && item.title);
    var owner = dash(item && (item.ownerName || item.nextOwner || item.owner));
    var proposer = dash(item && (item.proposerName || item.proposer));
    var updated = dash(item && (item.editedDate || item.updatedDate || item.updatedAt));
    var reviewer = dash(item && (item.reviewer || item.nextOwner || item.owner));
    var pri = dash(item && item.pri);
    var category = categoryLabel(item && item.category);
    var source = dash(item && item.source);
    var product = dash(item && item.product);
    var pool = dash(item && (item.poolName || item.pool));
    var launch = dash(item && (item.estimateLaunch || item.launchDate || item.expectedLaunch));

    setText("poDemandViewId", id);
    setText("poDemandViewTitle", title);
    setText("poDemandViewPri", pri === "—" ? "P2" : pri);
    setText("poDemandViewPriSide", pri === "—" ? "P2" : pri);
    setText("poDemandViewProposerMeta", proposer);
    setText("poDemandViewPoMeta", owner);
    setText("poDemandViewUpdatedMeta", updated);
    setText("poDemandViewCurrentOwner", dash(item && (item.nextOwner || item.owner || item.ownerName)));
    setText("poDemandViewBra", owner);
    setText("poDemandViewProposer", proposer);
    setText("poDemandViewReviewer", reviewer);

    setText("poDemandViewSpotStage", spotStageLabel(item));
    setText("poDemandViewSpotReviewer", reviewer === "—" ? "待确认" : reviewer);
    setText("poDemandViewSpotLaunch", launch);
    setText("poDemandViewStripCategory", category);
    setText("poDemandViewStripSource", source);
    setText("poDemandViewStripProduct", product);
    setText("poDemandViewStripPool", pool);
    setText("poDemandViewNativeState", zentaoStatusLabel(item && item.zentaoStatus));

    setText("poDemandViewCategory", category);
    setText("poDemandViewSource", source);
    setText("poDemandViewProduct", product);
    setText("poDemandViewPool", pool);
    setText("poDemandViewLaunch", launch);
  }

  function openRejectModal() {
    var modal = document.getElementById(REJECT_MODAL_ID);
    if (!modal) {
      return;
    }
    modal.style.display = "flex";
    modal.setAttribute("aria-hidden", "false");
    var ta = document.getElementById("poDemandViewRejectComment");
    if (ta) {
      ta.value = "";
      ta.focus();
    }
  }

  function closeRejectModal() {
    var modal = document.getElementById(REJECT_MODAL_ID);
    if (!modal) {
      return;
    }
    modal.style.display = "none";
    modal.setAttribute("aria-hidden", "true");
  }

  function closeDrawer() {
    closeRejectModal();
    var drawer = document.getElementById(DRAWER_ID);
    if (drawer) {
      drawer.classList.remove("active");
      drawer.setAttribute("aria-hidden", "true");
    }
    document.body.style.overflow = "";
    currentItem = null;
  }

  function openDrawer(item) {
    fillFromItem(item);
    var drawer = document.getElementById(DRAWER_ID);
    if (!drawer) {
      return;
    }
    drawer.classList.add("active");
    drawer.setAttribute("aria-hidden", "false");
    document.body.style.overflow = "hidden";
    var body = document.getElementById("poDemandViewBody");
    if (body) {
      body.scrollTop = 0;
    }
  }

  function bindEvents() {
    var drawer = document.getElementById(DRAWER_ID);
    if (!drawer) {
      return;
    }

    $("#poDemandViewCloseBtn").on("click", function () {
      closeDrawer();
    });
    $(drawer).on("click", function (e) {
      if (e.target === drawer) {
        closeDrawer();
      }
    });

    $("#poDemandViewRejectBtn").on("click", function () {
      openRejectModal();
    });
    $("#poDemandViewRejectCloseBtn, #poDemandViewRejectCancelBtn").on("click", function () {
      closeRejectModal();
    });
    $("#poDemandViewRejectConfirmBtn").on("click", function () {
      var comment = String($("#poDemandViewRejectComment").val() || "").trim();
      if (!comment) {
        showToast("驳回时请输入评审意见", "warning");
        $("#poDemandViewRejectComment").focus();
        return;
      }
      showToast("静态演示：驳回未提交", "info");
      closeRejectModal();
    });
    $("#poDemandViewPassBtn").on("click", function () {
      showToast("静态演示：评审通过未提交", "info");
    });

    $(document).on("keydown.poDemandView", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var rejectModal = document.getElementById(REJECT_MODAL_ID);
      if (rejectModal && rejectModal.style.display === "flex") {
        closeRejectModal();
        return;
      }
      if (drawer.classList.contains("active")) {
        closeDrawer();
      }
    });
  }

  window.openPoDemandViewDrawer = openDrawer;
  window.closePoDemandViewDrawer = closeDrawer;

  $(bindEvents);
})(jQuery);
