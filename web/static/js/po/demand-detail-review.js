// =============================================================================
// 文件: web/static/js/po/demand-detail-review.js
// 模块: PO 工作台
// 职责: 待评审需求详情的减法视图（精简双栏概览 + 底部评审操作栏）与评审提交交互。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define(["./demand-detail-richtext"], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory(require("./demand-detail-richtext.js"));
  } else {
    root.DemandDetailReview = factory(root.DemandDetailRichText);
  }
})(typeof self !== "undefined" ? self : this, function (RichText) {
  "use strict";

  var esc = window.escapeHtml;
  var sanitizeRichText = (RichText && RichText.sanitizeRichText) || function (raw) { return esc(raw); };

  function parseActionResponse(res, fallbackMsg) {
    return res.text().then(function (text) {
      var data = null;
      if (text) {
        try {
          data = JSON.parse(text);
        } catch (e) {
          data = { message: text.replace(/<[^>]*>/g, "").trim() };
        }
      }
      if (!res.ok || (data && data.success === false)) {
        throw new Error((data && data.message) || fallbackMsg);
      }
      return data || {};
    });
  }

  function categoryLabel(value) {
    var key = String(value || "").trim().toLowerCase();
    var labels = { experience: "体验优化", feature: "功能需求", request: "业务需求", business: "业务需求", research: "调研需求", bug: "BUG", tecopt: "技术优化", performance: "性能", safe: "安全", datacg: "数据变更", datachange: "数据变更", dataexport: "数据导出", other: "其他" };
    return labels[key] || value || "—";
  }

  // 渲染做减法后的待评审专享视图
  function renderReviewView(data) {
    var summary = (data && data.summary) || {};
    var req = (data && data.requirement) || {};
    var pa = (data && data.primaryAction) || {};
    var status = String(summary.status || "").trim().toLowerCase();
    var canReview = !!summary.canReview || (pa.key === "approve" && pa.enabled !== false);
    var canSubmitReview = pa.key === "submit_review" && pa.enabled !== false;
    var canWithdrawReview = !!summary.canWithdrawReview || (pa.key === "withdraw_review" && pa.enabled !== false);
    var isCreator = !!summary.isCreator || canWithdrawReview || canSubmitReview;
    var cleanId = String(summary.demandId || summary.id || "").replace(/^US/i, "");
    var zentaoEditUrl = String(summary.zentaoEditUrl || "").trim();
    var zentaoUrl = String(summary.zentaoUrl || "").trim();
    if (!/^https?:\/\//i.test(zentaoEditUrl)) {
      if (zentaoUrl) {
        var derivedEdit = zentaoUrl.replace(/\/demand-view-(\d+)\.html/i, "/demand-edit-$1.html")
                                   .replace(/([?&]f=)view(&|$)/i, "$1edit$2");
        if (derivedEdit !== zentaoUrl) {
          zentaoEditUrl = derivedEdit;
        } else if (zentaoEditUrl && zentaoEditUrl.charAt(0) === "/" && /^https?:\/\//i.test(zentaoUrl)) {
          try {
            var parsedUrl = new URL(zentaoUrl);
            var hash = zentaoEditUrl.indexOf("#") === -1 ? "#app=demandpool" : "";
            zentaoEditUrl = parsedUrl.origin + zentaoEditUrl + hash;
          } catch (e) {}
        }
      }
    }

    var safeSpecHtml = sanitizeRichText(req.specHtml || summary.desc);
    var safeVerifyHtml = sanitizeRichText(req.verifyHtml || summary.verifyPlan);

    var filesRows = (req.attachments || []).map(function (f) {
      return '<li><a href="' + esc(f.download) + '" target="_blank">' + esc(f.title) + '</a> (' + esc(f.size) + ')</li>';
    }).join("");

    var bannerBadge = "待业务评审";
    var bannerText = "";
    var bannerMod = "dd-review-banner--wait";
    var badgeMod = "";

    if (status === "wait") {
      if (canReview) {
        bannerBadge = "待我评审";
        bannerText = isCreator
          ? "该需求处于待业务评审阶段。您是业务评审人，也是创建人。可在底部统一办理（评审通过、驳回拒绝、编辑或撤回）。"
          : "该需求处于待业务评审阶段，您是业务评审人。请核对需求背景与验收标准后，在底部进行评审通过或驳回。";
      } else if (isCreator) {
        bannerBadge = "待业务评审";
        badgeMod = "dd-review-badge--creator";
        bannerText = "该需求已提交业务评审，正在等待业务评审人出具评审结果。您是该需求的创建人，可在底部编辑需求或撤回评审。";
      } else {
        bannerBadge = "待业务评审";
        badgeMod = "dd-review-badge--muted";
        bannerText = "该需求处于待业务评审阶段，正在等待业务评审人处理。当前仅支持查看。";
      }
    } else if (status === "refuse") {
      bannerBadge = "已驳回";
      bannerMod = "dd-review-banner--refuse";
      badgeMod = "dd-review-badge--refuse";
      bannerText = isCreator
        ? "该需求已被评审驳回。您是该需求的创建人，可在底部点击编辑前往修改需求内容，或重新提交业务评审。"
        : "该需求已被评审驳回。当前仅支持查看。";
    } else {
      bannerBadge = "草稿 / 暂存";
      bannerMod = "dd-review-banner--draft";
      badgeMod = "dd-review-badge--draft";
      bannerText = isCreator
        ? "该需求处于草稿/暂存状态。您是该需求的创建人，核对信息无误后可在底部提交业务评审，或前往编辑。"
        : "该需求处于草稿/暂存状态。当前仅支持查看。";
    }

    var leftActions = [];
    var rightActions = [];

    // 发起人/创建人操作组 (左侧)
    if (isCreator || canWithdrawReview) {
      if (summary.canEdit && zentaoEditUrl) {
        leftActions.push('<a href="' + esc(zentaoEditUrl) + '" target="_blank" rel="noopener noreferrer" class="dd-btn dd-btn-edit">✏️ 编辑需求 ↗</a>');
      } else if (summary.hasReviewed || summary.reviewedCount > 0) {
        var lockTip = summary.editDisabledReason || "已有评审人出具评审意见，需求已锁定修改；如需修改请先撤回评审申请";
        leftActions.push('<button type="button" class="dd-btn disabled" disabled title="' + esc(lockTip) + '">🔒 编辑已锁定 ↗</button>');
      } else if (zentaoEditUrl && (status === "draft" || status === "refuse")) {
        leftActions.push('<a href="' + esc(zentaoEditUrl) + '" target="_blank" rel="noopener noreferrer" class="dd-btn dd-btn-edit">✏️ 编辑需求 ↗</a>');
      }
      if (canWithdrawReview) {
        leftActions.push('<button type="button" class="dd-btn dd-btn-ghost-danger dd-withdraw-btn" id="ddWithdrawBtn" onclick="DemandDetailReview.openWithdrawModal(\'' + esc(cleanId) + '\')">撤销评审</button>');
      }
    }

    // 审批人/决断操作组 (右侧)
    if (status === "wait" && canReview) {
      rightActions.push('<button type="button" class="dd-btn danger-outline dd-reject-btn" id="ddRejectBtn" onclick="DemandDetailReview.openRejectModal(\'' + esc(cleanId) + '\')">驳回拒绝</button>');
      rightActions.push('<button type="button" class="dd-btn primary dd-pass-btn" id="ddPassBtn" onclick="DemandDetailReview.handlePass(\'' + esc(cleanId) + '\')">评审通过</button>');
    } else if ((status === "draft" || status === "refuse") && canSubmitReview) {
      rightActions.push('<button type="button" class="dd-btn primary dd-submit-review-btn" id="ddSubmitReviewBtn" onclick="DemandDetailReview.handleSubmitReview(\'' + esc(cleanId) + '\')">提交评审</button>');
    }

    var footerHtml = "";
    if (leftActions.length > 0 || rightActions.length > 0) {
      footerHtml = [
        '<div class="dd-review-footer dd-review-action-bar">',
        '  <div class="dd-review-footer-left">' + leftActions.join("") + '</div>',
        '  <div class="dd-review-footer-right">' + rightActions.join("") + '</div>',
        '</div>'
      ].join("");
    }

    var rejectModalHtml = [
      '<div id="ddRejectModal" class="dd-reject-modal-overlay" style="display:none;">',
      '  <div class="dd-reject-modal-dialog">',
      '    <div class="dd-reject-modal-header">',
      '      <h3>驳回需求评审</h3>',
      '      <button type="button" class="ui-close-btn" onclick="DemandDetailReview.closeRejectModal()">×</button>',
      '    </div>',
      '    <div class="dd-reject-modal-body">',
      '      <label class="dd-reject-label"><span class="dd-required">*</span> 请输入驳回原因 / 意见（必填）：</label>',
      '      <textarea id="ddRejectComment" class="dd-reject-textarea" rows="4" placeholder="请详细说明驳回原因，将作为评审记录同步至禅道并通知创建人..."></textarea>',
      '    </div>',
      '    <div class="dd-reject-modal-footer">',
      '      <button type="button" class="dd-btn" onclick="DemandDetailReview.closeRejectModal()">取消</button>',
      '      <button type="button" class="dd-btn danger" id="ddConfirmRejectBtn" onclick="DemandDetailReview.confirmReject(\'' + esc(cleanId) + '\')">确认驳回</button>',
      '    </div>',
      '  </div>',
      '</div>'
    ].join("");

    var withdrawModalHtml = [
      '<div id="ddWithdrawModal" class="dd-reject-modal-overlay" style="display:none;">',
      '  <div class="dd-reject-modal-dialog">',
      '    <div class="dd-reject-modal-header">',
      '      <h3>撤销需求评审</h3>',
      '      <button type="button" class="ui-close-btn" onclick="DemandDetailReview.closeWithdrawModal()">×</button>',
      '    </div>',
      '    <div class="dd-reject-modal-body">',
      '      <p class="dd-reject-modal-lead">确定要撤销该业务需求的评审申请吗？撤销后需求将退回<strong>草稿</strong>状态，评审流程中止。</p>',
      '      <label class="dd-reject-label">撤销原因 / 说明（选填）：</label>',
      '      <textarea id="ddWithdrawComment" class="dd-reject-textarea" rows="3" placeholder="选填，默认为“创建人撤销评审”，将作为评审记录同步至禅道..."></textarea>',
      '    </div>',
      '    <div class="dd-reject-modal-footer">',
      '      <button type="button" class="dd-btn" onclick="DemandDetailReview.closeWithdrawModal()">取消</button>',
      '      <button type="button" class="dd-btn danger" id="ddConfirmWithdrawBtn" onclick="DemandDetailReview.confirmWithdraw(\'' + esc(cleanId) + '\')">确认撤销</button>',
      '    </div>',
      '  </div>',
      '</div>'
    ].join("");

    var submitReviewModalHtml = window.DemandDetailReviewSubmitReview
      ? window.DemandDetailReviewSubmitReview.modalHtml(cleanId)
      : "";

    return [
      '<div class="dd-review-container">',
      '  <div class="dd-review-banner ' + esc(bannerMod) + '">',
      '    <div class="dd-review-banner-text">',
      '      <span class="dd-review-badge' + (badgeMod ? ' ' + esc(badgeMod) : '') + '">' + esc(bannerBadge) + '</span>',
      '      <span>' + esc(bannerText) + '</span>',
      '    </div>',
      '  </div>',
      '  <div class="dd-grid dd-review-grid">',
      '    <div class="dd-main">',
      '      <div class="dd-card"><div class="dd-card-body">',
      '        <div class="dd-cardhead"><h3>业务需求描述</h3></div>',
      '        <div class="dd-review-richtext">' + (safeSpecHtml || '<span class="text-muted">暂无详细描述</span>') + '</div>',
      '      </div></div>',
      '      <div class="dd-card"><div class="dd-card-body">',
      '        <div class="dd-cardhead"><h3>验收标准 (Verify Plan)</h3></div>',
      '        <div class="dd-review-richtext">' + (safeVerifyHtml || '<span class="text-muted">暂无验收标准说明</span>') + '</div>',
      '      </div></div>',
      filesRows ? ('      <div class="dd-card"><div class="dd-card-body"><div class="dd-cardhead"><h3>需求附件</h3></div><ul class="dd-file-list">' + filesRows + '</ul></div></div>') : '',
      '    </div>',
      '    <aside class="dd-sidebar">',
      '      <div class="dd-card"><div class="dd-card-body">',
      '        <div class="dd-cardhead"><h3>基本信息</h3></div>',
      '        <div class="dd-aside-title">业务属性</div>',
      '        <div class="dd-kv-list compact">',
      '          <div class="k">需求类别</div><div class="v">' + esc(categoryLabel(summary.category)) + '</div>',
      '          <div class="k">需求来源</div><div class="v">' + esc(summary.source) + '</div>',
      '          <div class="k">优先级</div><div class="v">' + esc(summary.priority) + '</div>',
      '          <div class="k">所属需求池</div><div class="v">' + esc(summary.poolName) + '</div>',
      '          <div class="k">所属模块</div><div class="v">' + esc(summary.moduleName || "—") + '</div>',
      '          <div class="k">期望上线日期</div><div class="v">' + esc(summary.estimateLaunch || "—") + '</div>',
      '        </div>',
      '        <div class="dd-aside-title" style="margin-top:14px">相关责任人</div>',
      '        <div class="dd-kv-list compact">',
      '          <div class="k">提出人</div><div class="v">' + esc(summary.proposerName) + '</div>',
      '          <div class="k">提出部门</div><div class="v">' + esc(summary.proposerDept || "—") + '</div>',
      '          <div class="k">负责人 / PO</div><div class="v">' + esc(summary.ownerName) + '</div>',
      '          <div class="k">业务评审人</div><div class="v">' + esc(summary.reviewer) + '</div>',
      '          <div class="k">创建人</div><div class="v">' + esc(summary.createdName || summary.createdBy || "—") + '</div>',
      '          <div class="k">当前责任人</div><div class="v">' + esc(summary.currentOwner || "待确认") + '</div>',
      '        </div>',
      '      </div></div>',
      '    </aside>',
      '  </div>',
      footerHtml,
      rejectModalHtml,
      withdrawModalHtml,
      submitReviewModalHtml,
      '</div>'
    ].join("");
  }

  function handlePass(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    if (!window.confirm("确认评审通过该需求吗？")) return;
    var btn = document.getElementById("ddPassBtn");
    if (btn) { btn.disabled = true; btn.textContent = "通过中…"; }
    submitReviewAction(cleanId, "pass", "确认通过", function () {
      if (btn) { btn.disabled = false; btn.textContent = "评审通过"; }
    });
  }

  function openRejectModal(demandId) {
    var modal = document.getElementById("ddRejectModal");
    if (!modal) return;
    modal.style.display = "flex";
    var ta = document.getElementById("ddRejectComment");
    if (ta) { ta.value = ""; ta.focus(); }
  }

  function closeRejectModal() {
    var modal = document.getElementById("ddRejectModal");
    if (modal) modal.style.display = "none";
  }

  function confirmReject(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    var ta = document.getElementById("ddRejectComment");
    var comment = (ta && ta.value || "").trim();
    if (!comment) {
      window.showToast("驳回时请输入评审意见", "warning");
      if (ta) ta.focus();
      return;
    }
    var btn = document.getElementById("ddConfirmRejectBtn");
    if (btn) { btn.disabled = true; btn.textContent = "驳回中…"; }
    submitReviewAction(cleanId, "refuse", comment, function () {
      if (btn) { btn.disabled = false; btn.textContent = "确认驳回"; }
      closeRejectModal();
    });
  }

  function submitReviewAction(cleanId, result, comment, onDone) {
    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/review", {
      method: "POST",
      headers: { "Content-Type": "application/json", "Accept": "application/json" },
      body: JSON.stringify({ result: result, isNeedFocus: "0", comment: comment })
    })
      .then(function (res) { return parseActionResponse(res, "评审提交失败"); })
      .then(function (data) {
        window.showToast((data && data.message) || (result === "pass" ? "评审通过成功" : "驳回成功"), "success");
        if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        } else if (window.QueryList && typeof window.QueryList.search === "function") {
          window.QueryList.search();
        } else if (window.PersonalList && typeof window.PersonalList.refresh === "function") {
          window.PersonalList.refresh();
        }
      })
      .catch(function (err) {
        window.showToast(err.message || "评审失败", "error");
      })
      .then(function () {
        if (typeof onDone === "function") onDone();
      });
  }

  function handleSubmit(event) {
    if (event && event.preventDefault) event.preventDefault();
    var demandId = window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : null;
    handlePass(demandId);
  }

  function openWithdrawModal(demandId) {
    var modal = document.getElementById("ddWithdrawModal");
    if (!modal) {
      handleWithdrawFallback(demandId);
      return;
    }
    modal.style.display = "flex";
    var ta = document.getElementById("ddWithdrawComment");
    if (ta) {
      ta.value = "";
      ta.focus();
    }
  }

  function closeWithdrawModal() {
    var modal = document.getElementById("ddWithdrawModal");
    if (modal) modal.style.display = "none";
  }

  function confirmWithdraw(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    var ta = document.getElementById("ddWithdrawComment");
    var comment = (ta && ta.value || "").trim();
    var btn = document.getElementById("ddConfirmWithdrawBtn");
    if (btn) {
      btn.disabled = true;
      btn.textContent = "撤销中…";
    }
    var actionBtn = document.getElementById("ddWithdrawBtn");
    if (actionBtn) {
      actionBtn.disabled = true;
      actionBtn.textContent = "撤销中…";
    }
    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/withdraw-review", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify({
        comment: comment || "工作台创建人撤销评审"
      })
    })
      .then(function (res) { return parseActionResponse(res, "撤销评审失败"); })
      .then(function (data) {
        closeWithdrawModal();
        window.showToast((data && data.message) || "撤销评审成功，需求已退回草稿状态", "success");
        if (window.DemandDetail && typeof window.DemandDetail.open === "function") {
          window.DemandDetail.open(cleanId);
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        } else if (window.QueryList && typeof window.QueryList.search === "function") {
          window.QueryList.search();
        } else if (window.PersonalList && typeof window.PersonalList.refresh === "function") {
          window.PersonalList.refresh();
        }
      })
      .catch(function (err) {
        window.showToast(err.message || "撤销评审失败", "error");
      })
      .then(function () {
        if (btn) {
          btn.disabled = false;
          btn.textContent = "确认撤销";
        }
        if (actionBtn) {
          actionBtn.disabled = false;
          actionBtn.textContent = "撤销评审";
        }
      });
  }

  function handleWithdrawFallback(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    if (!window.confirm("确定要撤销该需求的评审申请吗？撤销后需求将退回草稿状态。")) {
      return;
    }
    confirmWithdraw(cleanId);
  }

  function handleWithdraw(demandId) {
    openWithdrawModal(demandId);
  }

  function handleSubmitReview(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    if (window.DemandDetailReviewSubmitReview) {
      window.DemandDetailReviewSubmitReview.open(cleanId);
    }
  }

  return {
    renderReviewView: renderReviewView,
    handleSubmit: handleSubmit,
    handlePass: handlePass,
    openRejectModal: openRejectModal,
    closeRejectModal: closeRejectModal,
    confirmReject: confirmReject,
    openWithdrawModal: openWithdrawModal,
    closeWithdrawModal: closeWithdrawModal,
    confirmWithdraw: confirmWithdraw,
    handleWithdraw: handleWithdraw,
    handleSubmitReview: handleSubmitReview
  };
});
