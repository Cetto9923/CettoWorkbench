/* Homepage acceptance urge: batch-modal aligned with delivery. */
(function () {
  "use strict";

  var modal = document.getElementById("poUrgeModal");
  var overlay = document.getElementById("poUrgeOverlay");
  var form = document.getElementById("poUrgeForm");
  if (!modal || !overlay || !form) { return; }

  var context = document.getElementById("poUrgeContext");
  var demandCodeEl = document.getElementById("poUrgeDemandCode");
  var demandTitleEl = document.getElementById("poUrgeDemandTitle");
  var statusTagEl = document.getElementById("poUrgeStatusTag");
  var recipient = document.getElementById("poUrgeRecipient");
  var recipientTip = document.getElementById("poUrgeRecipientTip");
  var recipientTipText = document.getElementById("poUrgeRecipientTipText");
  var recipientBadge = document.getElementById("poUrgeRecipientBadge");
  var preview = document.getElementById("poUrgePreview");
  var channel = form.elements.inapp;
  var channelError = document.getElementById("poUrgeChannelError");
  var commentInput = form.elements.comment;
  var currentID = "";
  var basePreview = "";

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || window.escapeHtml || function (s) { return String(s == null ? "" : s); };

  function toast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function close() {
    overlay.classList.remove("show");
    overlay.setAttribute("aria-hidden", "true");
    currentID = "";
    basePreview = "";
  }

  function composedPreview() {
    var note = String(commentInput ? commentInput.value : "").trim();
    if (!basePreview) {
      var code = demandCodeEl ? demandCodeEl.textContent : (currentID ? ("US" + currentID) : "—");
      var title = demandTitleEl ? demandTitleEl.textContent : "";
      return note ? ("【催办验收】业务需求 " + code + "《" + title + "》：当前状态「待验收」，请验收责任人尽快完成业务验收。\n补充说明：" + note) : "正在加载通知正文…";
    }
    return note ? (basePreview + "\n补充说明：" + note) : basePreview;
  }

  function updatePreview() {
    if (preview) { preview.textContent = composedPreview(); }
  }

  function validChannels() {
    var ok = !!(channel && channel.checked);
    if (channelError) { channelError.hidden = ok; }
    return ok;
  }

  function setLoadingState() {
    if (recipient) { recipient.innerHTML = '<span class="po-urge-recipient-loading">正在解析验收责任人…</span>'; }
    if (recipientTip) { recipientTip.hidden = true; }
    if (recipientBadge) { recipientBadge.hidden = true; }
    basePreview = "";
    updatePreview();
  }

  function applyPreview(data) {
    data = data || {};
    var did = data.demandId || currentID || "";
    var code = did ? ("US" + did) : "—";
    var title = String(data.title || "").trim();
    var statusLabel = String(data.statusLabel || "待验收").trim();

    if (demandCodeEl) { demandCodeEl.textContent = code; }
    if (demandTitleEl) { demandTitleEl.textContent = title || "（未命名需求）"; }
    if (statusTagEl) { statusTagEl.textContent = statusLabel; }
    if (context) { context.textContent = code + " · 催办验收" + (title ? " · " + title : ""); }

    var recs = Array.isArray(data.recipients) ? data.recipients : [];
    if (recipient) {
      recipient.innerHTML = recs.length
        ? recs.map(function (r) {
            return '<span class="po-urge-recipient-chip"><i class="fas fa-user-check" aria-hidden="true"></i> ' + esc(r.label || r.account) + '</span>';
          }).join("")
        : '<span class="po-urge-no-recipient">未找到验收责任人</span>';
    }

    // 收件人兜底说明：严格呈现在催办对象下方，绝不污染催办原因
    var tip = String(data.recipientTip || "").trim();
    // 兼容降级处理：若后端仍返回旧格式且 reason 包含兜底说明
    if (!tip && data.reason && (data.reason.indexOf("未配置") !== -1 || data.reason.indexOf("推荐") !== -1)) {
      tip = data.reason;
    }

    if (recipientTip && recipientTipText) {
      if (tip) {
        recipientTipText.textContent = tip;
        recipientTip.hidden = false;
        if (recipientBadge) { recipientBadge.hidden = false; }
      } else {
        recipientTip.hidden = true;
        if (recipientBadge) { recipientBadge.hidden = true; }
      }
    }

    basePreview = String(data.messagePreview || "").trim();
    updatePreview();
  }

  function loadPreview() {
    setLoadingState();
    var fetchFn = (typeof window.appFetch === "function") ? window.appFetch : fetch;
    fetchFn("/demands/" + encodeURIComponent(currentID) + "/urge-preview", {
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
      if (recipient) { recipient.innerHTML = '<span class="po-urge-no-recipient">加载失败</span>'; }
      toast(error.message || "加载催办信息失败", "error");
    });
  }

  function open(buttonOrId, maybeTitle) {
    var did = "";
    var title = "";
    if (typeof buttonOrId === "object" && buttonOrId && buttonOrId.getAttribute) {
      did = String(buttonOrId.getAttribute("data-demand-id") || "").replace(/^US/i, "");
      title = buttonOrId.getAttribute("data-demand-title") || "";
      if (!title) {
        var row = buttonOrId.closest("tr");
        if (row) {
          var titleLink = row.querySelector(".table-title-link") || row.cells[1];
          title = titleLink ? titleLink.innerText.trim() : "";
        }
      }
    } else if (typeof buttonOrId === "string" || typeof buttonOrId === "number") {
      did = String(buttonOrId).replace(/^US/i, "");
      title = maybeTitle || "";
    }

    currentID = did;
    var code = currentID ? ("US" + currentID) : "—";
    if (demandCodeEl) { demandCodeEl.textContent = code; }
    if (demandTitleEl) { demandTitleEl.textContent = title || "加载中…"; }
    if (statusTagEl) { statusTagEl.textContent = "待验收"; }
    if (context) { context.textContent = code + " · 催办验收" + (title ? " · " + title : ""); }

    form.reset();
    if (channel) { channel.checked = true; }
    validChannels();
    overlay.classList.add("show");
    overlay.setAttribute("aria-hidden", "false");
    window.setTimeout(function () {
      if (commentInput) {
        commentInput.focus();
      } else {
        modal.focus();
      }
    }, 60);
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
    var fetchFn = (typeof window.appFetch === "function") ? window.appFetch : fetch;
    var submit = form.querySelector("button[type='submit']");
    if (submit) {
      submit.disabled = true;
      submit.textContent = "发送中…";
    }

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
      if (submit) {
        submit.disabled = false;
        submit.textContent = "确认催办";
      }
    });
  });

  window.PoUrgeModal = {
    open: open,
    close: close,
    applyPreview: applyPreview
  };
})();
