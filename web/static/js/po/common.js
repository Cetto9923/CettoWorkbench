/*
 * 文件: web/static/js/po/common.js
 * 模块: PO工作台
 * 职责: 业需详情抽屉公共能力（回填、富文本、拉详情）；评审/提交评审/撤销等复用
 */
(function (root) {
  "use strict";

  function showToast(message, level) {
    if (typeof root.showToast === "function") {
      root.showToast(message, level || "info");
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

  function demandNumericId(item) {
    var raw = String((item && (item.demandId || item.id)) || "").trim();
    return raw.replace(/^US/i, "");
  }

  function listItemToDetail(item) {
    return {
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
    };
  }

  /**
   * 按 DOM id 前缀创建抽屉回填器。
   * 前缀示例：poDemandReview / poDemandSubmitReview / poDemandCanal
   */
  function create(prefix) {
    function eid(suffix) {
      return prefix + suffix;
    }

    function renderAttachments(list) {
      var ul = document.getElementById(eid("Files"));
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

      setText(eid("UsId"), id);
      setText(eid("DrawerTitle"), title);
      setPriority(eid("Pri"), pri);
      setText(eid("PriSide"), pri === "—" ? "P2" : pri);
      setText(eid("ProposerMeta"), proposer);
      setText(eid("PoMeta"), owner);
      setText(eid("CurrentOwner"), dash(data && data.currentOwner));
      setText(eid("Bra"), owner);
      setText(eid("Proposer"), proposer);
      setText(eid("Dept"), dash(data && data.proposerDept));
      setText(eid("Reviewer"), reviewer);
      setText(eid("Creator"), dash(data && data.createdName));
      setText(eid("SpotStage"), stage);
      setText(eid("SpotReviewer"), reviewer === "—" ? "待确认" : reviewer);
      setText(eid("SpotLaunch"), launch);
      setText(eid("StripCategory"), category);
      setText(eid("StripSource"), source);
      setText(eid("StripPool"), pool);
      setText(eid("NativeState"), ztLabel);
      setText(eid("Category"), category);
      setText(eid("Source"), source);
      setText(eid("Pool"), pool);
      setText(eid("Launch"), launch);
    }

    function fillBodyContent(data) {
      setRichHtml(eid("Spec"), data && data.specHtml, "暂无详细描述");
      setRichHtml(eid("Verify"), data && data.verifyHtml, "暂无验收标准说明");
      renderAttachments(data && data.attachments);
    }

    function resetEditBtn() {
      var editBtn = document.getElementById(eid("EditBtn"));
      if (!editBtn) {
        return;
      }
      editBtn.disabled = true;
      editBtn.setAttribute("title", "暂无编辑链接");
      editBtn.onclick = null;
    }

    function bindEditBtn(data) {
      var editBtn = document.getElementById(eid("EditBtn"));
      if (!editBtn) {
        return;
      }
      var editUrl = String((data && data.zentaoEditUrl) || "").trim();
      if (!editUrl) {
        resetEditBtn();
        return;
      }
      editBtn.disabled = false;
      editBtn.removeAttribute("title");
      editBtn.onclick = function () {
        window.open(editUrl, "_blank", "noopener,noreferrer");
      };
    }

    function fillFromListItem(item) {
      fillHeaderAndAside(listItemToDetail(item));
      setRichHtml(eid("Spec"), "", "加载中…");
      setRichHtml(eid("Verify"), "", "加载中…");
      var filesEl = document.getElementById(eid("Files"));
      if (filesEl) {
        filesEl.innerHTML = '<li class="text-muted">加载中…</li>';
      }
      resetEditBtn();
    }

    function applyDetail(data) {
      if (!data) {
        return;
      }
      fillHeaderAndAside(data);
      fillBodyContent(data);
      bindEditBtn(data);
    }

    function setFailedBody() {
      setRichHtml(eid("Spec"), "", "加载失败");
      setRichHtml(eid("Verify"), "", "加载失败");
      renderAttachments([]);
    }

    function fetchDetail(item, ctrl, onSuccess) {
      var id = demandNumericId(item);
      if (!id) {
        return null;
      }
      if (ctrl && ctrl.abort && typeof ctrl.abort.abort === "function") {
        ctrl.abort.abort();
      }
      var abort = typeof AbortController !== "undefined" ? new AbortController() : null;
      if (ctrl) {
        ctrl.abort = abort;
      }
      var opts = { credentials: "same-origin", headers: { Accept: "application/json" } };
      if (abort) {
        opts.signal = abort.signal;
      }
      fetch("/demands/" + encodeURIComponent(id), opts)
        .then(function (res) {
          return res.json().then(function (json) {
            return { ok: res.ok, json: json };
          });
        })
        .then(function (ret) {
          if (!ret.ok || !ret.json || !ret.json.success || !ret.json.data) {
            showToast((ret.json && ret.json.message) || "获取需求详情失败", "error");
            setFailedBody();
            return;
          }
          applyDetail(ret.json.data);
          if (typeof onSuccess === "function") {
            onSuccess(ret.json.data);
          }
        })
        .catch(function (err) {
          if (err && err.name === "AbortError") {
            return;
          }
          showToast("获取需求详情失败", "error");
          setFailedBody();
        });
      return abort;
    }

    function openDrawer(item) {
      fillFromListItem(item);
      var drawer = document.getElementById(eid("Drawer"));
      if (!drawer) {
        return false;
      }
      drawer.classList.add("active");
      drawer.setAttribute("aria-hidden", "false");
      document.body.style.overflow = "hidden";
      var body = document.getElementById(eid("Body"));
      if (body) {
        body.scrollTop = 0;
      }
      return true;
    }

    function closeDrawer() {
      var drawer = document.getElementById(eid("Drawer"));
      if (drawer) {
        drawer.classList.remove("active");
        drawer.setAttribute("aria-hidden", "true");
      }
      document.body.style.overflow = "";
    }

    function showModal(modalId) {
      var modal = document.getElementById(modalId);
      if (!modal) {
        return;
      }
      modal.style.display = "flex";
      modal.setAttribute("aria-hidden", "false");
    }

    function hideModal(modalId) {
      var modal = document.getElementById(modalId);
      if (!modal) {
        return;
      }
      modal.style.display = "none";
      modal.setAttribute("aria-hidden", "true");
    }

    return {
      eid: eid,
      fillHeaderAndAside: fillHeaderAndAside,
      fillBodyContent: fillBodyContent,
      fillFromListItem: fillFromListItem,
      applyDetail: applyDetail,
      fetchDetail: fetchDetail,
      openDrawer: openDrawer,
      closeDrawer: closeDrawer,
      showModal: showModal,
      hideModal: hideModal,
      setFailedBody: setFailedBody
    };
  }

  root.PoDemandDrawer = {
    showToast: showToast,
    csrfHeaders: csrfHeaders,
    dash: dash,
    escapeHtml: escapeHtml,
    demandNumericId: demandNumericId,
    create: create
  };
})(typeof window !== "undefined" ? window : this);
