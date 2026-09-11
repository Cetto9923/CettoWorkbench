// =============================================================================
// 文件: web/static/js/po/demand-detail-edit.js
// 模块: PO 工作台
// 职责: 业务需求草稿/驳回状态的编辑抽屉表单交互与防误触删除二次确认弹窗。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.DemandDetailEdit = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  var currentDemandId = null;
  var currentDemandTitle = "";

  function $(id) { return document.getElementById(id); }

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function getAppFetch() {
    return (typeof window !== "undefined" && window.appFetch) ? window.appFetch : fetch;
  }

  function toast(msg, type) {
    if (typeof window.showToast === "function") {
      window.showToast(msg, type || "info");
    } else {
      alert(msg);
    }
  }

  // 1. 初始化 DOM 抽屉与删除模态框容器
  function ensureContainers() {
    var drawer = $("demandDetailEditDrawer");
    if (!drawer) {
      drawer = document.createElement("div");
      drawer.id = "demandDetailEditDrawer";
      drawer.className = "dd-edit-mask";
      drawer.innerHTML = [
        '<div class="dd-edit-panel" onclick="event.stopPropagation()">',
        '  <header class="dd-edit-head">',
        '    <div class="dd-edit-title-box">',
        '      <h3 id="ddeTitle">编辑业务需求</h3>',
        '      <span class="dd-tag blue" id="ddeStatusTag">草稿 / 暂存</span>',
        '    </div>',
        '    <button type="button" class="dd-edit-close" onclick="DemandDetailEdit.close()" aria-label="关闭">&times;</button>',
        '  </header>',
        '  <div class="dd-edit-banner" id="ddeBanner">',
        '    <i class="fas fa-info-circle" aria-hidden="true"></i>',
        '    <span>您正在编辑草稿需求，修改后可随时暂存，或完善后直接提交业务评审。</span>',
        '  </div>',
        '  <form class="dd-edit-body" id="ddeForm" onsubmit="return false;">',
        '    <div class="dd-edit-grid">',
        '      <div class="dd-edit-main">',
        '        <div class="dd-edit-card">',
        '          <div class="dd-edit-card-title"><i class="fas fa-file-alt"></i> 需求核心信息</div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeName">需求名称 / 标题 <span class="req">*</span></label>',
        '            <input type="text" class="dd-edit-input" id="ddeName" placeholder="简要概括该业务需求的核心目标（100字以内）" maxlength="200" required>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeDesc">业务需求描述 <span class="req">*</span></label>',
        '            <textarea class="dd-edit-textarea" id="ddeDesc" placeholder="请详细阐述该业务需求的业务背景、痛点问题与预期效果..." style="min-height:160px;" required></textarea>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeVerifyPlan">验收标准 (Verify Plan)</label>',
        '            <textarea class="dd-edit-textarea" id="ddeVerifyPlan" placeholder="明确可量化的验收标准与业务验收要点（若直接提交评审建议填写）" style="min-height:100px;"></textarea>',
        '          </div>',
        '        </div>',
        '      </div>',
        '      <aside class="dd-edit-aside">',
        '        <div class="dd-edit-card">',
        '          <div class="dd-edit-card-title"><i class="fas fa-sliders-h"></i> 业务属性</div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeCategory">需求类别</label>',
        '            <select class="dd-edit-select" id="ddeCategory"></select>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddePri">优先级</label>',
        '            <select class="dd-edit-select" id="ddePri"></select>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddePool">所属需求池</label>',
        '            <select class="dd-edit-select" id="ddePool"></select>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeProduct">所属产品</label>',
        '            <select class="dd-edit-select" id="ddeProduct"></select>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeEstimateLaunch">期望上线日期</label>',
        '            <input type="date" class="dd-edit-input" id="ddeEstimateLaunch">',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeSource">需求来源</label>',
        '            <select class="dd-edit-select" id="ddeSource"></select>',
        '          </div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeSourceNote">来源备注</label>',
        '            <input type="text" class="dd-edit-input" id="ddeSourceNote" placeholder="补充来源单号或背景说明">',
        '          </div>',
        '        </div>',
        '        <div class="dd-edit-card">',
        '          <div class="dd-edit-card-title"><i class="fas fa-user-check"></i> 评审与协同</div>',
        '          <div class="dd-edit-field">',
        '            <label class="dd-edit-label" for="ddeReviewer">业务评审人 <span class="req">*</span></label>',
        '            <select class="dd-edit-select" id="ddeReviewer" required></select>',
        '          </div>',
        '        </div>',
        '      </aside>',
        '    </div>',
        '  </form>',
        '  <footer class="dd-edit-footer">',
        '    <button type="button" class="dd-btn danger dd-delete-btn" id="ddeDeleteBtn" onclick="DemandDetailEdit.openDeleteConfirm()"><i class="fas fa-trash-alt"></i> 删除需求</button>',
        '    <div class="dd-edit-footer-right">',
        '      <button type="button" class="dd-btn" onclick="DemandDetailEdit.close()">取消</button>',
        '      <button type="button" class="dd-btn" id="ddeSaveDraftBtn" onclick="DemandDetailEdit.submit(false)"><i class="fas fa-save"></i> 暂存修改</button>',
        '      <button type="button" class="dd-btn primary" id="ddeSubmitReviewBtn" onclick="DemandDetailEdit.submit(true)"><i class="fas fa-paper-plane"></i> 保存并提交评审</button>',
        '    </div>',
        '  </footer>',
        '</div>'
      ].join("");
      drawer.onclick = function (e) {
        if (e.target === drawer) DemandDetailEdit.close();
      };
      document.body.appendChild(drawer);
    }

    var delModal = $("poDemandDeleteModalOverlay");
    if (!delModal) {
      delModal = document.createElement("div");
      delModal.id = "poDemandDeleteModalOverlay";
      delModal.className = "dd-delete-modal-overlay";
      delModal.setAttribute("role", "dialog");
      delModal.setAttribute("aria-modal", "true");
      delModal.innerHTML = [
        '<div class="dd-delete-modal" onclick="event.stopPropagation()">',
        '  <div class="dd-del-hdr">',
        '    <h4 class="dd-del-title"><i class="fas fa-exclamation-triangle" aria-hidden="true"></i> 删除业务需求确认</h4>',
        '    <button type="button" class="dd-del-close" onclick="DemandDetailEdit.closeDeleteConfirm()" aria-label="关闭">&times;</button>',
        '  </div>',
        '  <div class="dd-del-body">',
        '    <p style="margin:0;">您确认要删除以下业务需求草稿吗？</p>',
        '    <div class="dd-del-target-box">',
        '      <div class="dd-del-target-code" id="ddeDelCode">US--</div>',
        '      <div style="font-size:12px;color:var(--color-text-primary);margin-top:4px;" id="ddeDelTitle">--</div>',
        '    </div>',
        '    <div class="dd-del-warning">',
        '      <strong>⚠️ 高危操作风险提示：</strong><br>',
        '      1. 该操作不可撤销，删除后该需求草稿数据将从工作台彻底移除；<br>',
        '      2. 关联的草稿内容及历史动作将被清理。',
        '    </div>',
        '  </div>',
        '  <div class="dd-del-footer">',
        '    <button type="button" class="dd-btn" id="ddeDelCancelBtn" onclick="DemandDetailEdit.closeDeleteConfirm()">取消</button>',
        '    <button type="button" class="dd-btn danger" id="ddeDelConfirmBtn" onclick="DemandDetailEdit.executeDelete()"><i class="fas fa-trash-alt"></i> 确认删除</button>',
        '  </div>',
        '</div>'
      ].join("");
      delModal.onclick = function (e) {
        if (e.target === delModal) DemandDetailEdit.closeDeleteConfirm();
      };
      document.body.appendChild(delModal);
    }
  }

  function populateSelect(el, items, selectedVal) {
    if (!el) return;
    el.innerHTML = "";
    (items || []).forEach(function (it) {
      var opt = document.createElement("option");
      opt.value = it.value;
      opt.textContent = it.label;
      if (String(it.value) === String(selectedVal)) opt.selected = true;
      el.appendChild(opt);
    });
  }

  // 2. 打开编辑抽屉
  function open(demandId) {
    var cleanId = String(demandId || "").replace(/^US/i, "").trim();
    if (!cleanId) return;
    currentDemandId = cleanId;

    ensureContainers();
    var drawer = $("demandDetailEditDrawer");
    if (!drawer) return;

    // 获取并填充数据
    var fetchFn = getAppFetch();
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/edit-data")
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "获取编辑数据失败");
          return data;
        });
      })
      .then(function (data) {
        var d = data.demand || {};
        var opts = data.options || {};
        currentDemandTitle = d.name || "";

        $("ddeTitle").textContent = "编辑业务需求 · " + (d.code || ("US" + cleanId));
        $("ddeStatusTag").textContent = d.statusLabel || "草稿 / 暂存";
        $("ddeName").value = d.name || "";
        $("ddeDesc").value = d.desc || "";
        $("ddeVerifyPlan").value = d.verifyPlan || "";
        $("ddeEstimateLaunch").value = d.estimateLaunch || "";
        $("ddeSourceNote").value = d.sourceNote || "";

        populateSelect($("ddeCategory"), opts.categories, d.category);
        populateSelect($("ddePri"), opts.priorities, d.pri);
        populateSelect($("ddePool"), opts.pools, d.pool);
        populateSelect($("ddeProduct"), opts.products, d.product);
        populateSelect($("ddeSource"), opts.sources, d.source);
        populateSelect($("ddeReviewer"), opts.reviewers, d.reviewer);

        drawer.classList.add("active");
      })
      .catch(function (err) {
        toast(err.message || "无法打开编辑页面", "error");
      });
  }

  function close() {
    var drawer = $("demandDetailEditDrawer");
    if (drawer) drawer.classList.remove("active");
  }

  // 3. 保存提交处理
  function submit(submitReview) {
    if (!currentDemandId) return;
    var name = $("ddeName").value.trim();
    var desc = $("ddeDesc").value.trim();
    var reviewer = $("ddeReviewer").value.trim();

    if (!name) {
      toast("请输入需求名称 / 标题", "warning");
      $("ddeName").focus();
      return;
    }
    if (submitReview) {
      if (!desc) {
        toast("提交业务评审前请完善业务需求描述", "warning");
        $("ddeDesc").focus();
        return;
      }
      if (!reviewer) {
        toast("提交业务评审时必须指定业务评审人", "warning");
        $("ddeReviewer").focus();
        return;
      }
      if (!window.confirm("确定要保存修改并直接提交给业务评审人「" + ($("ddeReviewer").selectedOptions[0] ? $("ddeReviewer").selectedOptions[0].text : reviewer) + "」进行评审吗？")) {
        return;
      }
    }

    var btn = submitReview ? $("ddeSubmitReviewBtn") : $("ddeSaveDraftBtn");
    var origText = btn.innerHTML;
    btn.disabled = true;
    btn.innerHTML = submitReview ? '<i class="fas fa-spinner fa-spin"></i> 提交中…' : '<i class="fas fa-spinner fa-spin"></i> 保存中…';

    var payload = {
      id: parseInt(currentDemandId, 10),
      name: name,
      desc: $("ddeDesc").value,
      verifyPlan: $("ddeVerifyPlan").value,
      category: $("ddeCategory").value,
      pri: $("ddePri").value,
      pool: parseInt($("ddePool").value, 10) || 0,
      product: $("ddeProduct").value,
      estimateLaunch: $("ddeEstimateLaunch").value,
      source: $("ddeSource").value,
      sourceNote: $("ddeSourceNote").value,
      reviewer: reviewer,
      submitReview: !!submitReview,
      comment: submitReview ? "工作台编辑并提交业务评审" : "工作台保存需求草稿修改"
    };

    var fetchFn = getAppFetch();
    fetchFn("/demands/" + encodeURIComponent(currentDemandId) + "/edit", {
      method: "POST",
      headers: { "Content-Type": "application/json", "Accept": "application/json" },
      body: JSON.stringify(payload)
    })
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "保存失败");
          return data;
        });
      })
      .then(function (data) {
        toast((data && data.message) || "保存成功", "success");
        close();
        // 刷新详情抽屉
        if (window.DemandDetail && typeof window.DemandDetail.open === "function") {
          window.DemandDetail.open(currentDemandId);
        }
        // 刷新工作台列表
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        }
      })
      .catch(function (err) {
        toast(err.message || "保存失败", "error");
      })
      .then(function () {
        btn.disabled = false;
        btn.innerHTML = origText;
      });
  }

  // 4. 删除模态弹窗与执行
  function openDeleteConfirm(demandId, demandTitle) {
    ensureContainers();
    var did = demandId || currentDemandId;
    if (!did) return;
    currentDemandId = did;
    var title = demandTitle || currentDemandTitle || ("需求 US" + did);

    $("ddeDelCode").textContent = "US" + did;
    $("ddeDelTitle").textContent = title;

    var modal = $("poDemandDeleteModalOverlay");
    if (modal) {
      modal.classList.add("show");
      var cancelBtn = $("ddeDelCancelBtn");
      if (cancelBtn) cancelBtn.focus();
    }
  }

  function closeDeleteConfirm() {
    var modal = $("poDemandDeleteModalOverlay");
    if (modal) modal.classList.remove("show");
  }

  function executeDelete() {
    if (!currentDemandId) return;
    var btn = $("ddeDelConfirmBtn");
    if (btn) {
      btn.disabled = true;
      btn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> 删除中…';
    }

    var fetchFn = getAppFetch();
    fetchFn("/demands/" + encodeURIComponent(currentDemandId) + "/delete", {
      method: "POST",
      headers: { "Content-Type": "application/json", "Accept": "application/json" },
      body: JSON.stringify({ comment: "工作台创建人删除需求草稿" })
    })
      .then(function (res) {
        return res.json().then(function (data) {
          if (!res.ok) throw new Error((data && data.message) || "删除失败");
          return data;
        });
      })
      .then(function (data) {
        toast((data && data.message) || "需求已成功删除", "success");
        closeDeleteConfirm();
        close();
        if (window.DemandDetail && typeof window.DemandDetail.close === "function") {
          window.DemandDetail.close();
        }
        if (typeof window.refreshPoHomeDemands === "function") {
          window.refreshPoHomeDemands();
        }
        // 如果是在独立详情页上，退回首页
        if (window.location.pathname.indexOf("/demands/") !== -1 && window.location.pathname.indexOf("/detail") === -1) {
          setTimeout(function () { window.location.href = "/home"; }, 800);
        }
      })
      .catch(function (err) {
        toast(err.message || "删除失败", "error");
      })
      .then(function () {
        if (btn) {
          btn.disabled = false;
          btn.innerHTML = '<i class="fas fa-trash-alt"></i> 确认删除';
        }
      });
  }

  return {
    open: open,
    close: close,
    submit: submit,
    openDeleteConfirm: openDeleteConfirm,
    closeDeleteConfirm: closeDeleteConfirm,
    executeDelete: executeDelete
  };
});
