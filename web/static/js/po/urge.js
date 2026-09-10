/* Homepage acceptance urge: batch-modal aligned with delivery. */
(function () {
  "use strict";

  var modal = document.getElementById("poUrgeModal");
  var overlay = document.getElementById("poUrgeOverlay");
  var form = document.getElementById("poUrgeForm");
  if (!modal || !overlay || !form) { return; }

  var context = document.getElementById("poUrgeContext");
  var recipient = document.getElementById("poUrgeRecipient");
  var reason = document.getElementById("poUrgeReason");
  var preview = document.getElementById("poUrgePreview");
  var channel = form.elements.inapp;
  var channelError = document.getElementById("poUrgeChannelError");
  var commentInput = form.elements.comment;
  var currentID = "";
  var basePreview = "";

  function toast(message, level) {
    if (typeof window.showToast === "function") { window.showToast(message, level || "info"); }
  }

  function close() {
    overlay.classList.remove("show");
    overlay.setAttribute("aria-hidden", "true");
    currentID = "";
    basePreview = "";
  }

  function composedPreview() {
    var note = String(commentInput.value || "").trim();
    if (!basePreview) {
      return note || "—";
    }
    return note ? (basePreview + "\n补充说明：" + note) : basePreview;
  }

  function updatePreview() {
    preview.textContent = composedPreview();
  }

  function validChannels() {
    var ok = !!(channel && channel.checked);
    if (channelError) { channelError.hidden = ok; }
    return ok;
  }

  function setLoadingState() {
    if (recipient) { recipient.textContent = "正在解析验收责任人…"; }
    if (reason) { reason.textContent = "验收待办理，请及时处理"; }
    basePreview = "";
    updatePreview();
  }

  function applyPreview(data) {
    data = data || {};
    var title = String(data.title || "").trim();
    context.textContent = "US" + (data.demandId || currentID || "—") + " · 催办验收" + (title ? " · " + title : "");
    var recs = Array.isArray(data.recipients) ? data.recipients : [];
    if (recipient) {
      recipient.textContent = recs.length
        ? recs.map(function (r) { return r.label || r.account; }).join("、")
        : "未找到验收责任人";
    }
    if (reason) {
      reason.textContent = String(data.reason || "验收待办理，请及时处理");
    }
    basePreview = String(data.messagePreview || "").trim();
    updatePreview();
  }

  function loadPreview() {
    setLoadingState();
    if (typeof window.appFetch !== "function") {
      toast("网络组件未就绪，请刷新后重试", "error");
      return;
    }
    window.appFetch("/demands/" + encodeURIComponent(currentID) + "/urge-preview", {
      method: "GET",
      headers: { Accept: "application/json", "X-Requested-With": "XMLHttpRequest" }
    }).then(function (response) {
      return response.json().then(function (body) {
        if (!response.ok || !body.success) {
          throw new Error((body && body.message) || "加载催办信息失败");
        }
        return body.data || {};
      });
    }).then(applyPreview).catch(function (error) {
      if (recipient) { recipient.textContent = "加载失败"; }
      toast(error.message || "加载催办信息失败", "error");
    });
  }

  function open(button) {
    currentID = String(button.getAttribute("data-demand-id") || "").replace(/^US/i, "");
    var row = button.closest("tr");
    var title = row && row.cells[1] ? row.cells[1].innerText.trim() : "";
    context.textContent = "US" + (currentID || "—") + " · 催办验收" + (title ? " · " + title : "");
    form.reset();
    if (channel) { channel.checked = true; }
    validChannels();
    overlay.classList.add("show");
    overlay.setAttribute("aria-hidden", "false");
    window.setTimeout(function () { modal.focus(); }, 0);
    if (currentID) { loadPreview(); }
  }

  document.addEventListener("click", function (event) {
    var button = event.target.closest(".js-po-drawer-action[data-action-key='remind_accept']");
    if (button) {
      event.preventDefault();
      open(button);
    }
  });

  [document.getElementById("poUrgeClose"), document.getElementById("poUrgeCancel"), overlay].forEach(function (element) {
    if (!element) { return; }
    element.addEventListener("click", function (event) {
      if (element !== overlay || event.target === overlay) { close(); }
    });
  });

  if (channel) { channel.addEventListener("change", validChannels); }
  if (commentInput) { commentInput.addEventListener("input", updatePreview); }
  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && overlay.classList.contains("show")) { close(); }
  });

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    if (!currentID || !validChannels()) { return; }
    if (typeof window.appFetch !== "function") { toast("网络组件未就绪，请刷新后重试", "error"); return; }
    var submit = form.querySelector("button[type='submit']");
    submit.disabled = true;
    submit.textContent = "发送中…";
    // 补充说明单独提交；完整通知正文由服务端按验收人/需求组装，与预览一致。
    window.appFetch("/demands/" + encodeURIComponent(currentID) + "/urge", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ urgeType: "accept", channels: ["inapp"], comment: commentInput ? commentInput.value : "" })
    }).then(function (response) {
      return response.json().then(function (body) {
        if (!response.ok) { throw new Error(body.message || "催办失败"); }
        return body;
      });
    }).then(function (body) {
      toast(body.message || "已发起催办", body.duplicate ? "info" : "success");
      close();
      if (typeof window.refreshPoHomeDemands === "function") { return window.refreshPoHomeDemands(); }
    }).catch(function (error) {
      toast(error.message || "催办失败", "error");
    }).then(function () {
      submit.disabled = false;
      submit.textContent = "确认催办";
    });
  });
})();
