/* =============================================================================
   文件: web/static/js/kanban/story-people.js
   模块: 工作看板
   职责: 敏捷小组成员交互（必选一人、更多/收起）与需求角标动态联动。
============================================================================= */
(function () {
  "use strict";

  var VISIBLE_COUNT = 5;
  var root = document.querySelector(".po-board");
  if (!root) return;

  var peopleWrap = root.querySelector("[data-people]");
  if (!peopleWrap) return;

  var currentAccount = (peopleWrap.getAttribute("data-current-account") || "").trim();
  var onAccountChangeCallback = null;

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

  function memberPersons(group) {
    return Array.prototype.filter.call(
      group.querySelectorAll(".person[data-account]"),
      function (el) {
        return el.getAttribute("data-account") !== "all";
      }
    );
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

  function selectPerson(group, person) {
    if (!group || !person) return;
    group.querySelectorAll(".person.active").forEach(function (el) {
      el.classList.remove("active");
    });
    person.classList.add("active");
    var acc = person.getAttribute("data-account") || "";
    if (typeof onAccountChangeCallback === "function") {
      onAccountChangeCallback(acc);
    }
  }

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
      if (typeof onAccountChangeCallback === "function") {
        onAccountChangeCallback("");
      }
      return;
    }

    if (person.classList.contains("is-overflow") && allowExpand) {
      expandPeople(group);
    }
    person.classList.add("active");
    var acc = person.getAttribute("data-account") || "";
    if (typeof onAccountChangeCallback === "function") {
      onAccountChangeCallback(acc);
    }
  }

  function showGroup(teamgroupID) {
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

  function updateCounts(countsMap) {
    var grp = peopleWrap.querySelector(".people-group:not(.is-hidden)");
    if (!grp) return;
    grp.querySelectorAll(".person[data-account]").forEach(function (btn) {
      var acc = (btn.getAttribute("data-account") || "").trim();
      if (!acc || acc === "all") return;
      var num = (countsMap && countsMap[acc]) || 0;
      var el = btn.querySelector(".count");
      if (num > 0) {
        if (!el) {
          el = document.createElement("span");
          el.className = "count";
          btn.appendChild(el);
        }
        el.textContent = num;
      } else if (el) {
        el.remove();
      }
    });
  }

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

    if (person.classList.contains("active")) return;
    selectPerson(group, person);
  });

  window.StoryPeople = {
    init: function (options) {
      if (options && typeof options.onAccountChange === "function") {
        onAccountChangeCallback = options.onAccountChange;
      }
    },
    showGroup: showGroup,
    updateCounts: updateCounts,
    clearCounts: function () {
      updateCounts({});
    }
  };
})();
