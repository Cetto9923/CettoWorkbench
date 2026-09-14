/*
 * 文件: web/static/js/po/review.js
 * 模块: PO工作台
 * 职责: 业需审批右侧抽屉；打开时拉取 GET /demands/:id 回填。
 */
(function ($) {
  "use strict";

  var DRAWER_ID = "poDemandReviewDrawer";
  var REJECT_MODAL_ID = "poDemandReviewRejectModal";
  var currentItem = null;
  var detailAbort = null;

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
    var ul = document.getElementById("poDemandReviewFiles");
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

    setText("poDemandReviewUsId", id);
    setText("poDemandReviewDrawerTitle", title);
    setPriority("poDemandReviewPri", pri);
    setText("poDemandReviewPriSide", pri === "—" ? "P2" : pri);
    setText("poDemandReviewProposerMeta", proposer);
    setText("poDemandReviewPoMeta", owner);
    setText("poDemandReviewCurrentOwner", dash(data && data.currentOwner));
    setText("poDemandReviewBra", owner);
    setText("poDemandReviewProposer", proposer);
    setText("poDemandReviewDept", dash(data && data.proposerDept));
    setText("poDemandReviewReviewer", reviewer);
    setText("poDemandReviewCreator", dash(data && data.createdName));

    setText("poDemandReviewSpotStage", stage);
    setText("poDemandReviewSpotReviewer", reviewer === "—" ? "待确认" : reviewer);
    setText("poDemandReviewSpotLaunch", launch);
    setText("poDemandReviewStripCategory", category);
    setText("poDemandReviewStripSource", source);
    setText("poDemandReviewStripPool", pool);
    setText("poDemandReviewNativeState", ztLabel);

    setText("poDemandReviewCategory", category);
    setText("poDemandReviewSource", source);
    setText("poDemandReviewPool", pool);
    setText("poDemandReviewLaunch", launch);
  }

  function fillBodyContent(data) {
    setRichHtml("poDemandReviewSpec", data && data.specHtml, "暂无详细描述");
    setRichHtml("poDemandReviewVerify", data && data.verifyHtml, "暂无验收标准说明");
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
    setRichHtml("poDemandReviewSpec", "", "加载中…");
    setRichHtml("poDemandReviewVerify", "", "加载中…");
    renderAttachments([]);
    var filesEl = document.getElementById("poDemandReviewFiles");
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
    var editBtn = document.getElementById("poDemandReviewEditBtn");
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
          setRichHtml("poDemandReviewSpec", "", "加载失败");
          setRichHtml("poDemandReviewVerify", "", "加载失败");
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
        setRichHtml("poDemandReviewSpec", "", "加载失败");
        setRichHtml("poDemandReviewVerify", "", "加载失败");
      });
  }

  function openRejectModal() {
    var modal = document.getElementById(REJECT_MODAL_ID);
    if (!modal) {
      return;
    }
    modal.style.display = "flex";
    modal.setAttribute("aria-hidden", "false");
    var ta = document.getElementById("poDemandReviewRejectComment");
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
    var body = document.getElementById("poDemandReviewBody");
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

    $("#poDemandReviewDrawerCloseBtn").on("click", function () {
      closeDrawer();
    });
    $(drawer).on("click", function (e) {
      if (e.target === drawer) {
        closeDrawer();
      }
    });

    $("#poDemandReviewRejectBtn").on("click", function () {
      openRejectModal();
    });
    $("#poDemandReviewRejectCloseBtn, #poDemandReviewRejectCancelBtn").on("click", function () {
      closeRejectModal();
    });
    $("#poDemandReviewRejectConfirmBtn").on("click", function () {
      var comment = String($("#poDemandReviewRejectComment").val() || "").trim();
      if (!comment) {
        showToast("驳回时请输入评审意见", "warning");
        $("#poDemandReviewRejectComment").focus();
        return;
      }
      showToast("静态演示：驳回未提交", "info");
      closeRejectModal();
    });
    $("#poDemandReviewPassBtn").on("click", function () {
      showToast("静态演示：评审通过未提交", "info");
    });

    $(document).on("keydown.poDemandReview", function (e) {
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

  window.openPoDemandReviewDrawer = openDrawer;
  window.closePoDemandReviewDrawer = closeDrawer;

  $(bindEvents);
})(jQuery);
