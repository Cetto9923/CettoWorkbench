/* =============================================================================
   文件: web/static/js/kanban/story.js
   模块: 工作看板
   职责: 需求看板页需求树渲染（价值流五列）、顶部指标联动，协同成员组件与问题抽屉。
============================================================================= */
(function () {
  "use strict";

  var DEMANDS_URL = "/kanban/story/demands";

  var root = document.querySelector(".po-board");
  if (!root) return;

  var chipsWrap = root.querySelector("[data-teamgroup-chips]");
  var demandHost = document.getElementById("demandGroups");
  var selectedAccount = "";
  var loadSeq = 0;

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function firstRune(s) {
    var t = String(s || "").trim();
    if (!t) return "?";
    return Array.from(t)[0] || "?";
  }

  function priNum(pri) {
    var m = String(pri || "").match(/(\d+)/);
    return m ? m[1] : "";
  }

  // 价值流标签 → 看板 5 列（1=受理澄清 … 5=交付评价）
  function stageColOf(label) {
    var stage = String(label || "");
    if (/受理|澄清/.test(stage)) return 1;
    if (/排期/.test(stage)) return 2;
    if (/提测|研发|开发/.test(stage)) return 3;
    if (/联调|验收|测试/.test(stage)) return 4;
    if (/交付|评价|发布/.test(stage)) return 5;
    return 1;
  }

  // 业需 stage-mini：对齐原型 workboard-demand.js 文案
  function demandStageMini(label) {
    var stage = String(label || "");
    if (/受理|澄清/.test(stage)) return "科技侧需求梳理尚未完成";
    if (/排期/.test(stage)) return "未绑定版本窗口";
    if (/联调|验收/.test(stage)) return "测试完成 · 等待业务验收";
    return "";
  }

  function ownerBadgeHtml(owner) {
    var name = String(owner || "").trim();
    if (!name) return "";
    return (
      '<span class="owner-badge">' +
      '<span class="owner-dot">' +
      escapeHtml(firstRune(name)) +
      "</span>" +
      escapeHtml(name) +
      "</span>"
    );
  }

  function stageCellsHtml(item) {
    var stream = (item && item.valueStream) || "";
    var col = stageColOf(stream);
    var name = escapeHtml(stream || "受理");
    var mini = demandStageMini(stream);
    var miniHtml = mini
      ? '<div class="stage-mini">' + escapeHtml(mini) + "</div>"
      : "";
    var html = "";
    for (var c = 1; c <= 5; c++) {
      if (c !== col) {
        html += '<div class="stage-cell"></div>';
        continue;
      }
      html +=
        '<div class="stage-cell"><div class="stage-card actionable">' +
        '<div class="stage-top"><span class="stage-name">' +
        name +
        '</span><button type="button" class="stage-action" disabled>查看</button></div>' +
        miniHtml +
        "</div></div>";
    }
    return html;
  }

  function extractNumericId(raw) {
    var s = String(raw == null ? "" : raw).trim().replace(/^#/, "");
    if (!s) return "";
    var m = s.match(/^(?:US|REQ|SUB)-?(\d+)$/i);
    if (m) return m[1];
    m = s.match(/^(?:RD|U)-?(\d+)$/i);
    if (m) return m[1];
    m = s.match(/^(\d+)$/);
    return m ? m[1] : "";
  }

  function isBusinessDemandItem(item) {
    var kind = String((item && item.kind) || "").toLowerCase();
    if (kind === "demand" || kind === "business" || kind === "sub_demand") {
      return true;
    }
    if (kind === "story" || kind === "independent_story" || kind === "task") {
      return false;
    }
    var raw = String((item && item.id) || "").trim().replace(/^#/, "");
    if (/^U\d+$/i.test(raw) || /^(?:RD)-?\d+$/i.test(raw)) {
      return false;
    }
    return /^US/i.test(raw) || /^(?:REQ|SUB)-?/i.test(raw);
  }

  function formatChipId(item) {
    var num = extractNumericId(item && item.id);
    if (!num) return "";
    return isBusinessDemandItem(item) ? "#US" + num : "#" + num;
  }

  function idChipHtml(item) {
    var isStory = !isBusinessDemandItem(item);
    var label = isStory ? "研需" : "业需";
    var cls = isStory ? "wb-type-story" : "wb-type-business";
    var idText = formatChipId(item);
    var idInner = idText
      ? '<span class="wb-type-id">' + escapeHtml(idText) + "</span>"
      : "";
    return (
      '<span class="wb-type ' +
      cls +
      '"><span class="wb-type-tag">' +
      label +
      "</span>" +
      idInner +
      "</span>"
    );
  }

  function renderDemandRow(item) {
    var title = escapeHtml(item.title || "");
    var id = escapeHtml(item.id || "");
    var pri = escapeHtml(item.pri || "");
    var pNum = priNum(item.pri);
    var priAttr = pNum ? ' data-priority="' + escapeHtml(pNum) + '"' : "";

    var cnt = "";
    if (isBusinessDemandItem(item) && item.storyCount > 0) {
      cnt = ' <span class="summary">[' + (Number(item.storyCount) || 0) + "条研发需求]</span>";
    } else if (!isBusinessDemandItem(item) && item.taskTotal > 0) {
      var done = Number(item.taskDone) || 0;
      var total = Number(item.taskTotal) || 0;
      cnt = ' <span class="summary">[' + done + "/" + total + "任务]</span>";
    }

    var titleEl = item.zentaoUrl
      ? '<a class="node-title is-link" href="' +
        escapeHtml(item.zentaoUrl) +
        '" target="_blank" rel="noopener noreferrer" title="' +
        title +
        '">' +
        title +
        "</a>"
      : '<span class="node-title" title="' + title + '">' + title + "</span>";

    var ownerHtml = ownerBadgeHtml(item.owner);
    var metaHtml = ownerHtml
      ? '<div class="node-meta">' + ownerHtml + "</div>"
      : "";

    return (
      '<div class="demand-row is-standalone" data-demand-id="' +
      id +
      '" data-kind="' +
      escapeHtml(item.kind || "") +
      '">' +
      '<div class="tree-cell ind0"><div class="node-main">' +
      '<div class="node-title-line">' +
      idChipHtml(item) +
      (pri
        ? '<span class="wb-priority is-compact"' +
          priAttr +
          ">" +
          pri +
          "</span>"
        : "") +
      titleEl +
      cnt +
      "</div>" +
      metaHtml +
      "</div></div>" +
      stageCellsHtml(item) +
      "</div>"
    );
  }

  function showDemandEmpty(msg) {
    if (!demandHost) return;
    demandHost.innerHTML =
      '<div class="demand-empty demand-empty--visible">' +
      escapeHtml(msg) +
      "</div>";
  }

  function renderDemands(items) {
    if (!demandHost) return;
    if (!items || !items.length) {
      showDemandEmpty("暂无需求");
      return;
    }
    demandHost.innerHTML = items.map(renderDemandRow).join("");
  }

  function renderSummary(summary) {
    var stats = root.querySelectorAll(".stats .stat");
    if (!stats || stats.length < 4) return;
    stats[0].innerHTML = "待澄清 <strong>" + ((summary && summary.clarify) || 0) + "</strong>";
    stats[1].innerHTML = "待排期 <strong>" + ((summary && summary.schedule) || 0) + "</strong>";
    stats[2].innerHTML = "阻塞 <strong>" + ((summary && summary.blocked) || 0) + "</strong>";
    stats[3].innerHTML = "超期 <strong>—</strong>"; // 保留 —, 绝不伪造 0
  }

  function activeTeamgroupID() {
    if (!chipsWrap) return "";
    var active = chipsWrap.querySelector(".chip.active[data-teamgroup-id]");
    return active ? String(active.getAttribute("data-teamgroup-id") || "").trim() : "";
  }

  function demandsUrl(account) {
    var params = new URLSearchParams();
    var acc = String(account || "").trim();
    if (acc) params.set("account", acc);
    var tg = activeTeamgroupID();
    if (tg) params.set("teamgroupId", tg);
    var q = params.toString();
    return q ? DEMANDS_URL + "?" + q : DEMANDS_URL;
  }

  function loadDemands(account) {
    if (!demandHost) return;
    var acc = String(account || "").trim();
    if (!acc) {
      showDemandEmpty("暂无成员");
      renderSummary(null);
      if (window.StoryPeople && typeof window.StoryPeople.clearCounts === "function") {
        window.StoryPeople.clearCounts();
      }
      return;
    }
    var seq = ++loadSeq;
    var hasRows = !!demandHost.querySelector(".demand-row");
    if (hasRows) {
      demandHost.classList.add("is-loading");
    } else {
      showDemandEmpty("加载中…");
    }
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    fetchFn(demandsUrl(acc), { method: "GET", credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok) throw new Error("http");
        return r.json();
      })
      .then(function (payload) {
        if (seq !== loadSeq) return;
        demandHost.classList.remove("is-loading");
        if (!payload || payload.success !== true) throw new Error("payload");
        renderDemands(payload.items || []);
        renderSummary(payload.summary);
        if (window.StoryPeople && typeof window.StoryPeople.updateCounts === "function") {
          window.StoryPeople.updateCounts(payload.memberCounts || {});
        }
      })
      .catch(function () {
        if (seq !== loadSeq) return;
        demandHost.classList.remove("is-loading");
        showDemandEmpty("需求加载失败");
        renderSummary(null);
        if (window.StoryPeople && typeof window.StoryPeople.clearCounts === "function") {
          window.StoryPeople.clearCounts();
        }
      });
  }

  function setSelectedAccount(account, reload) {
    var next = String(account || "").trim();
    if (!next) return;
    var changed = next !== selectedAccount;
    selectedAccount = next;
    if (reload || changed) {
      loadDemands(selectedAccount);
      if (window.KanbanIssue && typeof window.KanbanIssue.setAccount === "function") {
        window.KanbanIssue.setAccount(selectedAccount);
      }
    }
  }

  function ensureDefaultTeamgroup() {
    if (!chipsWrap) {
      if (window.StoryPeople && typeof window.StoryPeople.showGroup === "function") {
        window.StoryPeople.showGroup("");
      }
      return;
    }
    var chips = chipsWrap.querySelectorAll(".chip[data-teamgroup-id]");
    if (!chips.length) {
      showDemandEmpty("暂无所属敏捷小组");
      renderSummary(null);
      return;
    }
    var active = chipsWrap.querySelector(".chip.active");
    if (!active) {
      chips[0].classList.add("active");
      active = chips[0];
    }
    if (window.StoryPeople && typeof window.StoryPeople.showGroup === "function") {
      window.StoryPeople.showGroup(active.getAttribute("data-teamgroup-id"));
    }
  }

  if (chipsWrap) {
    chipsWrap.addEventListener("click", function (ev) {
      var chip = ev.target.closest(".chip");
      if (!chip || !chipsWrap.contains(chip)) return;
      if (!chip.getAttribute("data-teamgroup-id")) return;

      chipsWrap.querySelectorAll(".chip.active").forEach(function (el) {
        el.classList.remove("active");
      });
      chip.classList.add("active");
      if (window.StoryPeople && typeof window.StoryPeople.showGroup === "function") {
        window.StoryPeople.showGroup(chip.getAttribute("data-teamgroup-id"));
      }
    });
  }

  if (window.StoryPeople && typeof window.StoryPeople.init === "function") {
    window.StoryPeople.init({
      onAccountChange: function (acc) {
        setSelectedAccount(acc, true);
      }
    });
  }

  if (window.KanbanIssue && typeof window.KanbanIssue.init === "function") {
    window.KanbanIssue.init({ getTeamgroupId: activeTeamgroupID });
  }

  ensureDefaultTeamgroup();
})();
