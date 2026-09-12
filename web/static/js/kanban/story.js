/* =============================================================================
   文件: web/static/js/kanban/story.js
   模块: 工作看板
   职责: 需求看板页交互（敏捷小组、成员折叠）+ ajax 加载价值流业务需求。
============================================================================= */
(function () {
  "use strict";

  var VISIBLE_COUNT = 5;
  var DEMANDS_URL = "/kanban/story/demands";

  var root = document.querySelector(".po-board");
  if (!root) return;

  var chipsWrap = root.querySelector("[data-teamgroup-chips]");
  var peopleWrap = root.querySelector("[data-people]");
  var demandHost = document.getElementById("demandGroups");
  var currentAccount = peopleWrap
    ? (peopleWrap.getAttribute("data-current-account") || "").trim()
    : "";

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

  function ownerBadgeHtml(owner) {
    var name = String(owner || "").trim();
    if (!name) return "";
    return (
      '<span class="meta-sep">·</span><span class="owner-badge">' +
      '<span class="owner-dot">' +
      escapeHtml(firstRune(name)) +
      "</span>PO " +
      escapeHtml(name) +
      "</span>"
    );
  }

  function stageCellsHtml(item) {
    var col = stageColOf(item && item.valueStream);
    var name = escapeHtml((item && item.valueStream) || "受理");
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
        '<div class="stage-mini">价值流推进中</div></div></div>';
    }
    return html;
  }

  function renderDemandRow(item) {
    var title = escapeHtml(item.title || "");
    var id = escapeHtml(item.id || "");
    var pri = escapeHtml(item.pri || "");
    var pNum = priNum(item.pri);
    var priAttr = pNum ? ' data-priority="' + escapeHtml(pNum) + '"' : "";
    var titleEl = item.zentaoUrl
      ? '<a class="node-title is-link" href="' +
        escapeHtml(item.zentaoUrl) +
        '" target="_blank" rel="noopener noreferrer" title="' +
        title +
        '">' +
        title +
        "</a>"
      : '<span class="node-title" title="' + title + '">' + title + "</span>";

    return (
      '<div class="demand-row is-standalone" data-demand-id="' +
      id +
      '">' +
      '<div class="tree-cell ind0"><div class="node-main">' +
      '<div class="node-title-line">' +
      '<span class="kb-type kb-type-biz">业务需求</span>' +
      (pri
        ? '<span class="wb-priority is-compact"' +
          priAttr +
          ">" +
          pri +
          "</span>"
        : "") +
      titleEl +
      "</div>" +
      '<div class="node-meta"><span class="code"># ' +
      id +
      "</span>" +
      ownerBadgeHtml(item.owner) +
      "</div></div></div>" +
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
      showDemandEmpty("暂无业务需求");
      return;
    }
    demandHost.innerHTML = items.map(renderDemandRow).join("");
  }

  function loadDemands() {
    if (!demandHost) return;
    showDemandEmpty("加载中…");
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    fetchFn(DEMANDS_URL, { method: "GET", credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok) throw new Error("http");
        return r.json();
      })
      .then(function (payload) {
        if (!payload || payload.success !== true) throw new Error("payload");
        renderDemands(payload.items || []);
      })
      .catch(function () {
        showDemandEmpty("业务需求加载失败");
      });
  }

  function showPeopleGroup(teamgroupID) {
    if (!peopleWrap) return;
    peopleWrap.querySelectorAll(".people-group").forEach(function (group) {
      var match = group.getAttribute("data-teamgroup-id") === String(teamgroupID);
      group.classList.toggle("is-hidden", !match);
      if (match) {
        group.removeAttribute("hidden");
        selectCurrentUser(group, true);
      } else {
        group.setAttribute("hidden", "hidden");
        group.querySelectorAll(".person.active").forEach(function (el) {
          el.classList.remove("active");
        });
      }
    });
  }

  function selectCurrentUser(group, allowExpand) {
    group.querySelectorAll(".person.active").forEach(function (el) {
      el.classList.remove("active");
    });
    if (!currentAccount) return;

    var person = group.querySelector(
      '.person[data-account="' + cssEscape(currentAccount) + '"]'
    );
    if (!person) return;

    if (person.classList.contains("is-overflow")) {
      if (!allowExpand) return;
      expandPeople(group);
    }
    person.classList.add("active");
  }

  function cssEscape(value) {
    if (window.CSS && typeof window.CSS.escape === "function") {
      return window.CSS.escape(value);
    }
    return String(value).replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function makeToggleBtn(expanded, hiddenCount) {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "person more";
    if (expanded) {
      btn.setAttribute("data-people-collapse", "");
      btn.textContent = "收起 ‹";
    } else {
      btn.setAttribute("data-people-more", "");
      btn.textContent = "更多 " + hiddenCount + " ›";
    }
    return btn;
  }

  function expandPeople(group) {
    group.querySelectorAll(".person[data-account]").forEach(function (el, idx) {
      if (idx < VISIBLE_COUNT) return;
      el.classList.remove("is-overflow");
      el.removeAttribute("hidden");
    });
    var more = group.querySelector("[data-people-more]");
    if (more) more.remove();
    var collapse = group.querySelector("[data-people-collapse]");
    if (collapse) collapse.remove();
    group.appendChild(makeToggleBtn(true, 0));
  }

  function collapsePeople(group) {
    var hidden = 0;
    group.querySelectorAll(".person[data-account]").forEach(function (el, idx) {
      if (idx < VISIBLE_COUNT) {
        el.classList.remove("is-overflow");
        el.removeAttribute("hidden");
        return;
      }
      el.classList.add("is-overflow");
      el.setAttribute("hidden", "hidden");
      el.classList.remove("active");
      hidden += 1;
    });
    var more = group.querySelector("[data-people-more]");
    if (more) more.remove();
    var collapse = group.querySelector("[data-people-collapse]");
    if (collapse) collapse.remove();
    if (hidden > 0) {
      group.appendChild(makeToggleBtn(false, hidden));
    }
    selectCurrentUser(group, false);
  }

  function ensureDefaultTeamgroup() {
    if (!chipsWrap) return;
    var chips = chipsWrap.querySelectorAll(".chip[data-teamgroup-id]");
    if (!chips.length) return;
    var active = chipsWrap.querySelector(".chip.active");
    if (!active) {
      chips[0].classList.add("active");
      active = chips[0];
    }
    showPeopleGroup(active.getAttribute("data-teamgroup-id"));
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
      showPeopleGroup(chip.getAttribute("data-teamgroup-id"));
    });
  }

  if (peopleWrap) {
    peopleWrap.addEventListener("click", function (ev) {
      var more = ev.target.closest("[data-people-more]");
      if (more && peopleWrap.contains(more)) {
        var moreGroup = more.closest(".people-group");
        if (moreGroup && !moreGroup.classList.contains("is-hidden")) {
          expandPeople(moreGroup);
        }
        return;
      }

      var collapse = ev.target.closest("[data-people-collapse]");
      if (collapse && peopleWrap.contains(collapse)) {
        var collapseGroup = collapse.closest(".people-group");
        if (collapseGroup && !collapseGroup.classList.contains("is-hidden")) {
          collapsePeople(collapseGroup);
        }
        return;
      }

      var person = ev.target.closest(".person[data-account]");
      if (!person || !peopleWrap.contains(person)) return;

      var group = person.closest(".people-group");
      if (!group || group.classList.contains("is-hidden")) return;

      var wasActive = person.classList.contains("active");
      group.querySelectorAll(".person.active").forEach(function (el) {
        el.classList.remove("active");
      });
      if (!wasActive) {
        person.classList.add("active");
      }
    });
  }

  ensureDefaultTeamgroup();
  loadDemands();
})();
