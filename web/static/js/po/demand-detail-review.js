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

  var esc = (RichText && RichText.esc) || function (str) {
    if (str === null || str === undefined) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  };
  var sanitizeRichText = (RichText && RichText.sanitizeRichText) || function (raw) { return esc(raw); };

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
    var canWithdrawReview = pa.key === "withdraw_review" && pa.enabled !== false;
    var isCreator = !!summary.isCreator || canWithdrawReview || canSubmitReview;
    var cleanId = String(summary.demandId || summary.id || "").replace(/^US/i, "");
    var zentaoEditUrl = String(summary.zentaoEditUrl || "").trim();

    var safeSpecHtml = sanitizeRichText(req.specHtml || summary.desc);
    var safeVerifyHtml = sanitizeRichText(req.verifyHtml || summary.verifyPlan);

    var filesRows = (req.attachments || []).map(function (f) {
      return '<li><a href="' + esc(f.download) + '" target="_blank">' + esc(f.title) + '</a> (' + esc(f.size) + ')</li>';
    }).join("");

    var bannerBadge = "待业务评审";
    var bannerText = "";
    var badgeStyle = "";

    if (status === "wait") {
      if (canReview) {
        bannerBadge = "待我评审";
        bannerText = "该需求处于待业务评审阶段，您是业务评审人。请核对需求背景、业务描述与验收标准后，在底部进行评审通过或驳回。";
      } else if (isCreator) {
        bannerBadge = "待业务评审";
        badgeStyle = "background:#e6f7ff;color:#096dd9;border:1px solid #91d5ff;";
        bannerText = "该需求已提交业务评审，正在等待业务评审人出具评审结果。您是该需求的创建人，如有需要可撤回评审。";
      } else {
        bannerBadge = "待业务评审";
        badgeStyle = "background:#f5f5f5;color:#595959;border:1px solid #d9d9d9;";
        bannerText = "该需求处于待业务评审阶段，正在等待业务评审人处理。当前仅支持查看。";
      }
    } else if (status === "refuse") {
      bannerBadge = "已驳回";
      badgeStyle = "background:#fff2f0;color:#cf1322;border:1px solid #ffa39e;";
      if (isCreator) {
        bannerText = "该需求已被评审驳回。您是该需求的创建人，可点击编辑前往修改需求内容，或重新提交业务评审。";
      } else {
        bannerText = "该需求已被评审驳回。当前仅支持查看。";
      }
    } else {
      bannerBadge = "草稿 / 暂存";
      badgeStyle = "background:#fffbe6;color:#d46b08;border:1px solid #ffe58f;";
      if (isCreator) {
        bannerText = "该需求处于草稿/暂存状态。您是该需求的创建人，核对信息无误后可直接提交业务评审，或前往编辑。";
      } else {
        bannerText = "该需求处于草稿/暂存状态。当前仅支持查看。";
      }
    }

    var footerHtml = "";
    if (status === "wait" && canReview) {
      footerHtml = [
        '<div class="dd-review-footer">',
        '  <form class="dd-review-form" id="ddReviewInlineForm" onsubmit="DemandDetailReview.handleSubmit(event)">',
        '    <div class="dd-review-form-row">',
        '      <div class="dd-review-form-item">',
        '        <span class="dd-review-label">评审结果：</span>',
        '        <label class="dd-radio-label"><input type="radio" name="result" value="pass" checked> 确认通过</label>',
        '        <label class="dd-radio-label"><input type="radio" name="result" value="refuse"> 驳回拒绝</label>',
        '      </div>',
        '      <div class="dd-review-form-item">',
        '        <span class="dd-review-label">重点关注：</span>',
        '        <label class="dd-radio-label"><input type="radio" name="isNeedFocus" value="0" checked> 否</label>',
        '        <label class="dd-radio-label"><input type="radio" name="isNeedFocus" value="1"> 是</label>',
        '      </div>',
        '    </div>',
        '    <div class="dd-review-form-row">',
        '      <div class="dd-review-form-item" style="flex:1;">',
        '        <input type="text" name="comment" class="dd-review-input" placeholder="输入评审意见 / 备注（驳回必填，通过可选）" maxlength="1000">',
        '      </div>',
        '      <button type="submit" class="dd-btn primary dd-review-submit-btn" id="ddReviewSubmitBtn">评审需求</button>',
        '    </div>',
        '  </form>',
        '</div>'
      ].join("");
    } else if (status === "wait" && canWithdrawReview) {
      footerHtml = [
        '<div class="dd-review-footer" style="display:flex;align-items:center;justify-content:space-between;">',
        '  <span style="color:#595959;font-size:13px;">如需修改需求内容或暂停推进，您可以撤回当前评审申请：</span>',
        '  <button type="button" class="dd-btn danger dd-withdraw-btn" id="ddWithdrawBtn" onclick="DemandDetailReview.handleWithdraw(\'' + esc(cleanId) + '\')">撤回评审</button>',
        '</div>'
      ].join("");
    } else if ((status === "draft" || status === "refuse") && canSubmitReview) {
      footerHtml = [
        '<div class="dd-review-footer" style="display:flex;align-items:center;justify-content:space-between;">',
        '  <span style="color:#595959;font-size:13px;">您可以编辑完善需求，或直接提交给业务评审人进行评审：</span>',
        '  <div style="display:flex;gap:12px;">',
        (zentaoEditUrl ? '    <a href="' + esc(zentaoEditUrl) + '" target="_blank" rel="noopener noreferrer" class="dd-btn">编辑需求 ↗</a>' : ''),
        '    <button type="button" class="dd-btn primary dd-submit-review-btn" id="ddSubmitReviewBtn" onclick="DemandDetailReview.handleSubmitReview(\'' + esc(cleanId) + '\')">提交评审</button>',
        '  </div>',
        '</div>'
      ].join("");
    }

    return [
      '<div class="dd-review-container">',
      '  <div class="dd-review-banner">',
      '    <div class="dd-review-banner-text">',
      '      <span class="dd-review-badge"' + (badgeStyle ? ' style="' + esc(badgeStyle) + '"' : '') + '>' + esc(bannerBadge) + '</span>',
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
      '          <div class="k">所属产品</div><div class="v">' + esc(summary.product) + '</div>',
      '          <div class="k">所属需求池</div><div class="v">' + esc(summary.poolName) + '</div>',
      '          <div class="k">期望上线</div><div class="v">' + esc(summary.estimateLaunch || "—") + '</div>',
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
      '</div>'
    ].join("");
  }

  function handleSubmit(event) {
    if (event && event.preventDefault) event.preventDefault();
    var form = document.getElementById("ddReviewInlineForm");
    if (!form) return;
    var btn = document.getElementById("ddReviewSubmitBtn");
    var demandId = window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : null;
    var cleanId = String(demandId || "").replace(/^US/i, "");
    if (!cleanId) {
      if (typeof window.showToast === "function") window.showToast("需求信息不可用", "error");
      return;
    }

    var result = (form.querySelector("input[name='result']:checked") || {}).value || "pass";
    var isNeedFocus = (form.querySelector("input[name='isNeedFocus']:checked") || {}).value || "0";
    var comment = (form.querySelector("input[name='comment']") || {}).value || "";

    if (result === "refuse" && !comment.trim()) {
      if (typeof window.showToast === "function") window.showToast("驳回时请输入评审意见", "warning");
      var commentInput = form.querySelector("input[name='comment']");
      if (commentInput) commentInput.focus();
      return;
    }

    if (btn) {
      btn.disabled = true;
      btn.textContent = "评审中…";
    }

    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/review", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify({
        result: result,
        isNeedFocus: isNeedFocus,
        comment: comment
      })
    })
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "评审提交失败");
          return data;
        });
      })
      .then(function (data) {
        if (typeof window.showToast === "function") {
          window.showToast((data && data.message) || "评审成功", "success");
        }
        if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        }
      })
      .catch(function (err) {
        if (typeof window.showToast === "function") {
          window.showToast(err.message || "评审失败", "error");
        }
      })
      .then(function () {
        if (btn) {
          btn.disabled = false;
          btn.textContent = "评审需求";
        }
      });
  }

  function handleWithdraw(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    if (!window.confirm("确定要撤回该需求的评审申请吗？撤回后需求将退回草稿状态。")) {
      return;
    }
    var btn = document.getElementById("ddWithdrawBtn");
    if (btn) {
      btn.disabled = true;
      btn.textContent = "撤回中…";
    }
    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/withdraw-review", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify({
        comment: "工作台创建人撤回评审"
      })
    })
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "撤回评审失败");
          return data;
        });
      })
      .then(function (data) {
        if (typeof window.showToast === "function") {
          window.showToast((data && data.message) || "撤回评审成功", "success");
        }
        if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        }
      })
      .catch(function (err) {
        if (typeof window.showToast === "function") {
          window.showToast(err.message || "撤回评审失败", "error");
        }
      })
      .then(function () {
        if (btn) {
          btn.disabled = false;
          btn.textContent = "撤回评审";
        }
      });
  }

  function handleSubmitReview(demandId) {
    var cleanId = String(demandId || (window.DemandDetail && window.DemandDetail.getCurrentDemandId ? window.DemandDetail.getCurrentDemandId() : "")).replace(/^US/i, "");
    if (!cleanId) return;
    if (!window.confirm("确定要将该需求提交业务评审吗？")) {
      return;
    }
    var btn = document.getElementById("ddSubmitReviewBtn");
    if (btn) {
      btn.disabled = true;
      btn.textContent = "提交中…";
    }
    var fetchFn = (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/submit-review", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json"
      },
      body: JSON.stringify({
        comment: "工作台创建人提交评审"
      })
    })
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "提交评审失败");
          return data;
        });
      })
      .then(function (data) {
        if (typeof window.showToast === "function") {
          window.showToast((data && data.message) || "提交评审成功", "success");
        }
        if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        }
      })
      .catch(function (err) {
        if (typeof window.showToast === "function") {
          window.showToast(err.message || "提交评审失败", "error");
        }
      })
      .then(function () {
        if (btn) {
          btn.disabled = false;
          btn.textContent = "提交评审";
        }
      });
  }

  return {
    renderReviewView: renderReviewView,
    handleSubmit: handleSubmit,
    handleWithdraw: handleWithdraw,
    handleSubmitReview: handleSubmitReview
  };
});
