/* =============================================================================
   文件: web/static/js/kanban/story.js
   模块: 工作看板
   职责: 需求看板页交互（敏捷小组默认选中、成员折叠展开与选中态）。
============================================================================= */
(function () {
  "use strict";

  var VISIBLE_COUNT = 5;

  var root = document.querySelector(".po-board");
  if (!root) return;

  var chipsWrap = root.querySelector("[data-teamgroup-chips]");
  var peopleWrap = root.querySelector("[data-people]");
  if (!chipsWrap || !peopleWrap) return;

  function showPeopleGroup(teamgroupID) {
    peopleWrap.querySelectorAll(".people-group").forEach(function (group) {
      var match = group.getAttribute("data-teamgroup-id") === String(teamgroupID);
      group.classList.toggle("is-hidden", !match);
      if (match) {
        group.removeAttribute("hidden");
      } else {
        group.setAttribute("hidden", "hidden");
        group.querySelectorAll(".person.active").forEach(function (el) {
          el.classList.remove("active");
        });
      }
    });
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
  }

  function ensureDefaultTeamgroup() {
    var chips = chipsWrap.querySelectorAll(".chip[data-teamgroup-id]");
    if (!chips.length) return;
    var active = chipsWrap.querySelector(".chip.active");
    if (!active) {
      chips[0].classList.add("active");
      active = chips[0];
    }
    showPeopleGroup(active.getAttribute("data-teamgroup-id"));
  }

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

  ensureDefaultTeamgroup();
})();
