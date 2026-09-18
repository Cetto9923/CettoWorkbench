/* =============================================================================
   文件: web/static/js/kanban/story.js
   模块: 工作看板
   职责: 需求看板页交互（敏捷小组、成员必选）+ 按选中负责人加载价值流业需/研需。
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
    var isBusiness = isBusinessDemandItem(item);
    var mini = "";

    if (isBusiness) {
      if (item && item.storyCount > 0) {
        mini = item.storyCount + " 研发需求推进中";
      } else {
        mini = demandStageMini(stream);
      }
    } else {
      if (item && item.taskTotal > 0) {
        mini = "任务 " + item.taskDone + "/" + item.taskTotal;
        if (item.owner) {
          mini += " · " + item.owner + "负责";
        }
      } else if (/研发|提测/.test(stream)) {
        mini = "尚未创建研发任务";
      } else {
        mini = demandStageMini(stream);
      }
    }

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
    var countHtml = "";
    if (isBusinessDemandItem(item) && item.storyCount > 0) {
      countHtml = '<span class="summary" style="font-size:10px;color:var(--t3);margin-left:6px;">' + escapeHtml(String(item.storyCount)) + " 研需</span>";
    }
    var metaHtml = (ownerHtml || countHtml)
      ? '<div class="node-meta">' + ownerHtml + countHtml + "</div>"
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
      })
      .catch(function () {
        if (seq !== loadSeq) return;
        demandHost.classList.remove("is-loading");
        showDemandEmpty("需求加载失败");
      });
  }

  function setSelectedAccount(account, reload) {
    var next = String(account || "").trim();
    if (!next) return;
    var changed = next !== selectedAccount;
    selectedAccount = next;
    if (reload || changed) {
      loadDemands(selectedAccount);
    }
  }

  function showPeopleGroup(teamgroupID) {
    if (!peopleWrap) return;
    peopleWrap.querySelectorAll(".people-group").forEach(function (group) {
      var match = group.getAttribute("data-teamgroup-id") === String(teamgroupID);
      group.classList.toggle("is-hidden", !match);
      if (match) {
        group.removeAttribute("hidden");
        selectDefaultPerson(group, true);
      } else {
        group.setAttribute("hidden", "hidden");
        group.querySelectorAll(".person.active").forEach(function (el) {
          el.classList.remove("active");
        });
      }
    });
  }

  // 必选一人：优先当前登录用户，否则「全部」；不可取消选中。
  function selectDefaultPerson(group, allowExpand) {
    group.querySelectorAll(".person.active").forEach(function (el) {
      el.classList.remove("active");
    });

    var person = null;
    if (currentAccount) {
      person = group.querySelector(
        '.person[data-account="' + cssEscape(currentAccount) + '"]'
      );
    }
    if (!person) {
      person = group.querySelector('.person[data-account="all"]');
    }
    if (!person) {
      person = group.querySelector(".person[data-account]");
    }
    if (!person) {
      selectedAccount = "";
      showDemandEmpty("暂无成员");
      return;
    }

    if (person.classList.contains("is-overflow")) {
      if (allowExpand) {
        expandPeople(group);
      }
    }
    person.classList.add("active");
    setSelectedAccount(person.getAttribute("data-account"), true);
  }

  function memberPersons(group) {
    return Array.prototype.filter.call(
      group.querySelectorAll(".person[data-account]"),
      function (el) {
        return el.getAttribute("data-account") !== "all";
      }
    );
  }

  function selectPerson(group, person) {
    if (!group || !person) return;
    group.querySelectorAll(".person.active").forEach(function (el) {
      el.classList.remove("active");
    });
    person.classList.add("active");
    setSelectedAccount(person.getAttribute("data-account"), true);
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
    memberPersons(group).forEach(function (el, idx) {
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
    var activeAccount = "";
    var active = group.querySelector(".person.active[data-account]");
    if (active) {
      activeAccount = active.getAttribute("data-account") || "";
    }
    memberPersons(group).forEach(function (el, idx) {
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
    // 收起后若选中人被藏起，保持选中账号并展开回选中；否则重新点亮可见选中人
    if (activeAccount) {
      var still = group.querySelector(
        '.person[data-account="' + cssEscape(activeAccount) + '"]'
      );
      if (still && still.classList.contains("is-overflow")) {
        expandPeople(group);
        still = group.querySelector(
          '.person[data-account="' + cssEscape(activeAccount) + '"]'
        );
      }
      if (still) {
        group.querySelectorAll(".person.active").forEach(function (el) {
          el.classList.remove("active");
        });
        still.classList.add("active");
      }
    }
  }

  function ensureDefaultTeamgroup() {
    if (!chipsWrap) {
      if (peopleWrap) {
        var only = peopleWrap.querySelector(".people-group:not(.is-hidden)");
        if (only) selectDefaultPerson(only, true);
      }
      return;
    }
    var chips = chipsWrap.querySelectorAll(".chip[data-teamgroup-id]");
    if (!chips.length) {
      showDemandEmpty("暂无所属敏捷小组");
      return;
    }
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

      // 不可取消选中：已选中再点仍保持选中，不重复请求
      if (person.classList.contains("active")) return;
      selectPerson(group, person);
    });
  }

  ensureDefaultTeamgroup();
})();
