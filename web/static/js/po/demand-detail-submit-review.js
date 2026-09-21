// 提交业务评审弹框：复用共享用户选择器，支持多人会签。
(function (root) {
  "use strict";

  var Review = root.DemandDetailReview;
  if (!Review) return;
  var esc = root.escapeHtml;
  var state = null;
  var clearingPicker = false;

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

  function modalHtml(cleanId) {
    return [
      '<div id="ddSubmitReviewModal" class="dd-reject-modal-overlay" style="display:none;">',
      '  <div class="dd-reject-modal-dialog dd-submit-review-dialog">',
      '    <div class="dd-reject-modal-header"><div><h3>提交需求评审</h3><p class="dd-submit-review-hint">请选择业务评审人，支持多人会签；选择结果将同步至禅道。</p></div>',
      '      <button type="button" class="ui-close-btn" onclick="DemandDetailReview.closeSubmitReviewModal()">×</button>',
      '    </div>',
      '    <div class="dd-reject-modal-body">',
      '      <label class="dd-reject-label"><span class="dd-required">*</span> 业务评审人</label>',
      '      <div class="dd-review-picker" id="ddReviewPicker"><div class="dd-review-selected" id="ddReviewSelected" aria-live="polite"></div>',
      '        <div class="dd-review-picker-input"><input id="ddReviewReviewerInput" type="text" autocomplete="off" placeholder="输入姓名、工号或拼音搜索"><input id="ddReviewReviewerValue" type="hidden"></div>',
      '      </div>',
      '      <div id="ddReviewCandidatesLoading" class="dd-review-picker-state">正在加载可选评审人…</div>',
      '      <div id="ddReviewCandidatesError" class="dd-review-picker-state dd-review-picker-error" style="display:none;"></div>',
      '      <label class="dd-reject-label dd-submit-comment-label">评审说明（选填）</label>',
      '      <textarea id="ddSubmitReviewComment" class="dd-reject-textarea" rows="3" placeholder="可补充本次提交评审的说明，将同步至禅道…">工作台创建人提交评审</textarea>',
      '    </div>',
      '    <div class="dd-reject-modal-footer"><button type="button" class="dd-btn" onclick="DemandDetailReview.closeSubmitReviewModal()">取消</button>',
      '      <button type="button" class="dd-btn primary" id="ddConfirmSubmitReviewBtn" onclick="DemandDetailReview.confirmSubmitReview(\'' + esc(cleanId) + '\')">确认提交</button></div>',
      '  </div>',
      '</div>'
    ].join("");
  }

  function items(data) {
    return (data && data.users || []).map(function (item) {
      return { value: String(item.value || "").trim(), label: item.label || item.value || "", dept: item.dept || "", pinyin: item.pinyin || "" };
    }).filter(function (item) { return item.value; });
  }

  function label(account) {
    if (!state) return account;
    var item = state.users.find(function (user) { return user.value === account; });
    return item ? item.label : account;
  }

  function renderChips() {
    var host = document.getElementById("ddReviewSelected");
    if (!host || !state) return;
    host.innerHTML = state.selected.map(function (account) {
      return '<span class="dd-review-chip">' + esc(label(account)) + '<button type="button" class="dd-review-chip-remove" data-reviewer-account="' + esc(account) + '" aria-label="移除 ' + esc(label(account)) + '">×</button></span>';
    }).join("");
    host.querySelectorAll(".dd-review-chip-remove").forEach(function (button) {
      button.addEventListener("click", function () { remove(button.getAttribute("data-reviewer-account")); });
    });
  }

  function add(account) {
    account = String(account || "").trim();
    if (!state || !account || state.selected.indexOf(account) >= 0) return;
    state.selected.push(account);
    renderChips();
  }

  function remove(account) {
    if (!state) return;
    state.selected = state.selected.filter(function (item) { return item !== account; });
    renderChips();
  }

  function destroy() {
    if (typeof root.destroyUserPicker === "function") root.destroyUserPicker("ddReviewReviewerInput");
    state = null;
  }

  function close() {
    var modal = document.getElementById("ddSubmitReviewModal");
    if (modal) modal.style.display = "none";
    destroy();
  }

  function bind(data) {
    var users = items(data);
    state = { users: users, selected: [] };
    (data.selected || []).forEach(function (account) {
      account = String(account || "").trim();
      if (users.some(function (item) { return item.value === account; })) add(account);
    });
    renderChips();
    root.initUserPicker("ddReviewReviewerInput", "ddReviewReviewerValue", users, { placeholder: "输入姓名、工号或拼音搜索", maxShow: 20 });
    var value = document.getElementById("ddReviewReviewerValue");
    if (value) value.addEventListener("change", function () {
      if (clearingPicker) return;
      add(value.value);
      if (typeof root.clearAutocomplete === "function") {
        clearingPicker = true;
        try {
          root.clearAutocomplete("ddReviewReviewerInput");
        } finally {
          clearingPicker = false;
        }
      }
    });
    var loading = document.getElementById("ddReviewCandidatesLoading");
    if (loading) loading.style.display = "none";
    var button = document.getElementById("ddConfirmSubmitReviewBtn");
    if (button) button.disabled = false;
  }

  function renderLoadError(error, errorEl) {
    if (!errorEl) return;
    if (error && error.isAuth) {
      var location = root.location || {};
      var redirect = (location.pathname || "/home") + (location.search || "");
      errorEl.innerHTML = '登录已过期，请 <a href="/login?redirect=' + encodeURIComponent(redirect) + '">重新登录</a>后再试';
      errorEl.style.display = "flex";
      return;
    }
    errorEl.textContent = (error && error.message) || "加载评审人失败，请稍后重试";
    errorEl.style.display = "flex";
  }

  function open(demandId) {
    var modal = document.getElementById("ddSubmitReviewModal");
    if (!modal) return;
    destroy();
    modal.style.display = "flex";
    var loading = document.getElementById("ddReviewCandidatesLoading");
    var error = document.getElementById("ddReviewCandidatesError");
    var button = document.getElementById("ddConfirmSubmitReviewBtn");
    if (loading) loading.style.display = "block";
    if (error) { error.style.display = "none"; error.textContent = ""; }
    if (button) button.disabled = true;
    var fetchFn = root.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(demandId) + "/review-candidates", { method: "GET", credentials: "same-origin", headers: { Accept: "application/json" } })
      .then(function (res) {
        if (res.status === 401) {
          var authErr = new Error("登录已过期，请重新登录");
          authErr.isAuth = true;
          throw authErr;
        }
        return res.json().then(function (data) {
          if (!res.ok || !data || data.success === false) {
            throw new Error((data && data.message) || "加载评审人失败");
          }
          return data.data || data;
        });
      })
      .then(bind)
      .catch(function (err) { if (loading) loading.style.display = "none"; renderLoadError(err, error); });
  }

  function confirm(demandId) {
    var cleanId = String(demandId || "").replace(/^US/i, "");
    if (!cleanId || !state) return;
    if (!state.selected.length) { root.showToast("请至少选择一位业务评审人", "warning"); return; }
    var button = document.getElementById("ddConfirmSubmitReviewBtn");
    if (button) { button.disabled = true; button.textContent = "提交中…"; }
    var fetchFn = root.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(cleanId) + "/submit-review", { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify({ reviewer: state.selected, comment: (document.getElementById("ddSubmitReviewComment") || {}).value || "工作台创建人提交评审" }) })
      .then(function (res) { return parseActionResponse(res, "提交评审失败"); })
      .then(function (data) {
        close();
        root.showToast((data && data.message) || "提交评审成功", "success");
        if (root.DemandDetail && typeof root.DemandDetail.open === "function") root.DemandDetail.open(cleanId);
        if (typeof root.refreshPoHomeDemands === "function") root.refreshPoHomeDemands();
        else if (root.QueryList && typeof root.QueryList.search === "function") root.QueryList.search();
        else if (root.PersonalList && typeof root.PersonalList.refresh === "function") root.PersonalList.refresh();
      })
      .catch(function (err) { root.showToast(err.message || "提交评审失败", "error"); })
      .then(function () { if (button) { button.disabled = false; button.textContent = "确认提交"; } });
  }

  root.DemandDetailReviewSubmitReview = { modalHtml: modalHtml, open: open };
  Review.openSubmitReviewModal = open;
  Review.closeSubmitReviewModal = close;
  Review.confirmSubmitReview = confirm;
  Review.removeReviewReviewer = remove;
  Review.handleSubmitReview = open;
})(typeof window !== "undefined" ? window : this);
