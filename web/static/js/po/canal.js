/*
 * 文件: web/static/js/po/canal.js
 * 模块: PO工作台
 * 职责: 撤销评审右侧抽屉；打开时拉取 GET /demands/:id 回填；确认后 POST /demands/:id/withdraw-review。
 */
(function ($) {
  "use strict";

  var DRAWER_ID = "poDemandCanalDrawer";
  var WITHDRAW_MODAL_ID = "poDemandCanalWithdrawModal";
  var currentItem = null;
  var detailAbort = null;
  var submitting = false;

  function showToast(message, level) {
    if (typeof window.showToast === "function") {
      window.showToast(message, level || "info");
    }
  }

  function csrfHeaders() {
    var headers = {
      Accept: "application/json",
      "Content-Type": "application/json",
      "X-Requested-With": "XMLHttpRequest"
    };
    var meta = document.querySelector('meta[name="csrf-token"]');
    var token = meta ? String(meta.getAttribute("content") || "").trim() : "";
    if (token) {
      headers["X-CSRF-Token"] = token;
    }
    return headers;
  }

  function withdrawFailMessage(res, data, text) {
    if (data) {
      var fromApi = String(data.message || data.error || "").trim();
      if (fromApi) {
        return fromApi;
      }
    }
    if (res && res.status === 401) {
      return "未登录或会话已过期，请刷新后重试";
    }
    if (res && res.status === 403) {
      return "没有撤回评审权限";
    }
    if (text && /CSRF/i.test(text)) {
      return "安全校验失败，请刷新页面后重试";
    }
    if (res && res.status) {
      return "撤回评审失败（HTTP " + res.status + "）";
    }
    return "撤回评审失败";
  }

  function setWithdrawButtonsDisabled(disabled) {
    $("#poDemandCanalWithdrawBtn, #poDemandCanalWithdrawConfirmBtn").prop("disabled", !!disabled);
  }

  function submitWithdraw(comment) {
    var id = demandNumericId(currentItem);
    if (!id) {
      showToast("需求 ID 无效", "error");
      return;
    }
    if (submitting) {
      return;
    }
    submitting = true;
    setWithdrawButtonsDisabled(true);
    var fetchFn = window.appFetch || fetch;
    fetchFn("/demands/" + encodeURIComponent(id) + "/withdraw-review", {
      method: "POST",
      headers: csrfHeaders(),
      body: JSON.stringify({
        comment: comment || ""
      })
    })
      .then(function (res) {
        return res.text().then(function (text) {
          var data = {};
          try {
            data = text ? JSON.parse(text) : {};
          } catch (ignore) {
            data = {};
          }
          return { ok: res.ok, data: data, res: res, text: text };
        });
      })
      .then(function (wrap) {
        var data = wrap.data;
        if (wrap.ok && data.success) {
          showToast(data.message || "撤回评审成功", "success");
          closeWithdrawModal();
          closeDrawer();
          if (typeof window.refreshPoHomeDemands === "function") {
            window.refreshPoHomeDemands();
          }
          return;
        }
        showToast(withdrawFailMessage(wrap.res, data, wrap.text), "error");
      })
      .catch(function () {
        showToast("撤回评审失败，请稍后重试", "error");
      })
      .then(function () {
        submitting = false;
        setWithdrawButtonsDisabled(false);
      });
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

  function setPriority(id, pri) {
    var el = document.getElementById(id);
    if (!el) {
      return;
    }
    var raw = String(pri == null ? "" : pri).trim();
    var m = raw.match(/(\d+)/);
    var num = m ? m[1] : "2";
    el.textContent = "P" + num;
    el.setAttribute("data-priority", num);
  }

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function sanitizeRichHtml(html) {
    var tmp = document.createElement("div");
    tmp.innerHTML = String(html || "");
    tmp.querySelectorAll("script,style,iframe,object,embed,form,link,meta").forEach(function (el) {
      el.remove();
    });
    tmp.querySelectorAll("*").forEach(function (el) {
      Array.from(el.attributes).forEach(function (attr) {
        var name = attr.name || "";
        var value = String(attr.value || "");
        if (/^on/i.test(name) || (name === "href" && /^javascript:/i.test(value))) {
          el.removeAttribute(name);
        }
      });
    });
    return tmp.innerHTML;
  }

  function formatRichTextContent(raw) {
    raw = String(raw == null ? "" : raw).trim();
    if (!raw) {
      return "";
    }
    if (/<[a-z][\s\S]*>/i.test(raw)) {
      return sanitizeRichHtml(raw);
    }
    return escapeHtml(raw).replace(/\n/g, "<br>");
  }

  function setRichHtml(id, html, emptyText) {
    var el = document.getElementById(id);
    if (!el) {
      return;
    }
    var safe = formatRichTextContent(html);
    el.innerHTML = safe || '<span class="text-muted">' + escapeHtml(emptyText || "暂无") + "</span>";
  }

  function renderAttachments(list) {
    var ul = document.getElementById("poDemandCanalFiles");
    if (!ul) {
      return;
    }
    var rows = Array.isArray(list) ? list : [];
    if (!rows.length) {
      ul.innerHTML = '<li class="text-muted">暂无附件</li>';
      return;
    }
    ul.innerHTML = rows
      .map(function (f) {
        var title = escapeHtml(f && f.title);
        var size = escapeHtml(f && f.size);
        var href = escapeHtml((f && f.download) || "#");
        return (
          "<li><a href=\"" +
          href +
          "\" target=\"_blank\" rel=\"noopener noreferrer\">" +
          title +
          "</a> (" +
          size +
          ")</li>"
        );
      })
      .join("");
  }

  function demandNumericId(item) {
    var raw = String((item && (item.demandId || item.id)) || "").trim();
    raw = raw.replace(/^US/i, "");
    return raw;
  }

  function fillHeaderAndAside(data) {
    var id = dash(data && data.id);
    var title = dash(data && data.title);
    var owner = dash(data && data.ownerName);
    var proposer = dash(data && data.proposerName);
    var reviewer = dash(data && data.reviewer);
    var pri = dash(data && data.pri);
    var category = dash(data && data.category);
    var source = dash(data && data.source);
    var pool = dash(data && data.poolName);
    var launch = dash(data && data.deadline);
    var stage = dash(data && data.valueStageLabel);
    var ztLabel = dash(data && data.zentaoStatusLabel);

    setText("poDemandCanalUsId", id);
    setText("poDemandCanalDrawerTitle", title);
    setPriority("poDemandCanalPri", pri);
    setText("poDemandCanalPriSide", pri === "—" ? "P2" : pri);
    setText("poDemandCanalProposerMeta", proposer);
    setText("poDemandCanalPoMeta", owner);
    setText("poDemandCanalCurrentOwner", dash(data && data.currentOwner));
    setText("poDemandCanalBra", owner);
    setText("poDemandCanalProposer", proposer);
    setText("poDemandCanalDept", dash(data && data.proposerDept));
    setText("poDemandCanalReviewer", reviewer);
    setText("poDemandCanalCreator", dash(data && data.createdName));

    setText("poDemandCanalSpotStage", stage);
    setText("poDemandCanalSpotReviewer", reviewer === "—" ? "待确认" : reviewer);
    setText("poDemandCanalSpotLaunch", launch);
    setText("poDemandCanalStripCategory", category);
    setText("poDemandCanalStripSource", source);
    setText("poDemandCanalStripPool", pool);
    setText("poDemandCanalNativeState", ztLabel);

    setText("poDemandCanalCategory", category);
    setText("poDemandCanalSource", source);
    setText("poDemandCanalPool", pool);
    setText("poDemandCanalLaunch", launch);
  }

  function fillBodyContent(data) {
    setRichHtml("poDemandCanalSpec", data && data.specHtml, "暂无详细描述");
    setRichHtml("poDemandCanalVerify", data && data.verifyHtml, "暂无验收标准说明");
    renderAttachments(data && data.attachments);
  }

  function fillFromListItem(item) {
    currentItem = item || null;
    fillHeaderAndAside({
      id: item && item.id,
      title: item && item.title,
      pri: item && item.pri,
      ownerName: item && (item.ownerName || item.nextOwner || item.owner),
      proposerName: item && (item.proposerName || item.proposer),
      reviewer: item && item.reviewer,
      category: item && item.category,
      source: item && item.source,
      poolName: item && (item.poolName || item.pool),
      deadline: item && (item.deadline || item.estimateLaunch || item.launchDate || item.expectedLaunch),
      valueStageLabel: item && (item.valueStream || item.stage),
      zentaoStatusLabel: item && item.zentaoStatus,
      currentOwner: item && (item.nextOwner || item.owner || item.ownerName),
      proposerDept: item && item.proposerDept,
      createdName: item && item.createdName
    });
    setRichHtml("poDemandCanalSpec", "", "加载中…");
    setRichHtml("poDemandCanalVerify", "", "加载中…");
    renderAttachments([]);
    var filesEl = document.getElementById("poDemandCanalFiles");
    if (filesEl) {
      filesEl.innerHTML = '<li class="text-muted">加载中…</li>';
    }
  }

  function applyDetail(data) {
    if (!data) {
      return;
    }
    fillHeaderAndAside(data);
    fillBodyContent(data);
    var editBtn = document.getElementById("poDemandCanalEditBtn");
    if (editBtn) {
      var editUrl = String((data && data.zentaoEditUrl) || "").trim();
      if (editUrl) {
        editBtn.disabled = false;
        editBtn.removeAttribute("title");
        editBtn.onclick = function () {
          window.open(editUrl, "_blank", "noopener,noreferrer");
        };
      }
    }
  }

  function fetchDetail(item) {
    var id = demandNumericId(item);
    if (!id) {
      return;
    }
    if (detailAbort && typeof detailAbort.abort === "function") {
      detailAbort.abort();
    }
    detailAbort = typeof AbortController !== "undefined" ? new AbortController() : null;
    var opts = { credentials: "same-origin", headers: { Accept: "application/json" } };
    if (detailAbort) {
      opts.signal = detailAbort.signal;
    }
    fetch("/demands/" + encodeURIComponent(id), opts)
      .then(function (res) {
        return res.json().then(function (json) {
          return { ok: res.ok, status: res.status, json: json };
        });
      })
      .then(function (ret) {
        if (!ret.ok || !ret.json || !ret.json.success || !ret.json.data) {
          var msg = (ret.json && ret.json.message) || "获取需求详情失败";
          showToast(msg, "error");
          setRichHtml("poDemandCanalSpec", "", "加载失败");
          setRichHtml("poDemandCanalVerify", "", "加载失败");
          renderAttachments([]);
          return;
        }
        applyDetail(ret.json.data);
      })
      .catch(function (err) {
        if (err && err.name === "AbortError") {
          return;
        }
        showToast("获取需求详情失败", "error");
        setRichHtml("poDemandCanalSpec", "", "加载失败");
        setRichHtml("poDemandCanalVerify", "", "加载失败");
      });
  }

  function openWithdrawModal() {
    var modal = document.getElementById(WITHDRAW_MODAL_ID);
    if (!modal) {
      return;
    }
    modal.style.display = "flex";
    modal.setAttribute("aria-hidden", "false");
    var ta = document.getElementById("poDemandCanalWithdrawComment");
    if (ta) {
      ta.value = "";
      ta.focus();
    }
  }

  function closeWithdrawModal() {
    var modal = document.getElementById(WITHDRAW_MODAL_ID);
    if (!modal) {
      return;
    }
    modal.style.display = "none";
    modal.setAttribute("aria-hidden", "true");
  }

  function closeDrawer() {
    closeWithdrawModal();
    if (detailAbort && typeof detailAbort.abort === "function") {
      detailAbort.abort();
      detailAbort = null;
    }
    var drawer = document.getElementById(DRAWER_ID);
    if (drawer) {
      drawer.classList.remove("active");
      drawer.setAttribute("aria-hidden", "true");
    }
    document.body.style.overflow = "";
    currentItem = null;
  }

  function openDrawer(item) {
    fillFromListItem(item);
    var drawer = document.getElementById(DRAWER_ID);
    if (!drawer) {
      return;
    }
    drawer.classList.add("active");
    drawer.setAttribute("aria-hidden", "false");
    document.body.style.overflow = "hidden";
    var body = document.getElementById("poDemandCanalBody");
    if (body) {
      body.scrollTop = 0;
    }
    fetchDetail(item);
  }

  function bindEvents() {
    var drawer = document.getElementById(DRAWER_ID);
    if (!drawer) {
      return;
    }

    $("#poDemandCanalDrawerCloseBtn").on("click", function () {
      closeDrawer();
    });
    $(drawer).on("click", function (e) {
      if (e.target === drawer) {
        closeDrawer();
      }
    });

    $("#poDemandCanalWithdrawBtn").on("click", function () {
      openWithdrawModal();
    });
    $("#poDemandCanalWithdrawCloseBtn, #poDemandCanalWithdrawCancelBtn").on("click", function () {
      closeWithdrawModal();
    });
    $("#poDemandCanalWithdrawConfirmBtn").on("click", function () {
      var comment = String($("#poDemandCanalWithdrawComment").val() || "").trim();
      submitWithdraw(comment);
    });

    $(document).on("keydown.poDemandCanal", function (e) {
      if (e.key !== "Escape" && e.keyCode !== 27) {
        return;
      }
      var withdrawModal = document.getElementById(WITHDRAW_MODAL_ID);
      if (withdrawModal && withdrawModal.style.display === "flex") {
        closeWithdrawModal();
        return;
      }
      if (drawer.classList.contains("active")) {
        closeDrawer();
      }
    });
  }

  window.openPoDemandCanalDrawer = openDrawer;
  window.closePoDemandCanalDrawer = closeDrawer;

  $(bindEvents);
})(jQuery);
