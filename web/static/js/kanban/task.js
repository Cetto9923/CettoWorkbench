/* =============================================================================
   文件: web/static/js/kanban/task.js
   模块: 工作看板
   职责: 任务看板页交互（敏捷小组、成员必选）+ 按选中负责人加载三列任务。
============================================================================= */
(function () {
  "use strict";

  var VISIBLE_COUNT = 5;
  var TASKS_URL = "/kanban/task/items";

  var root = document.querySelector(".po-board");
  if (!root) return;

  var chipsWrap = root.querySelector("[data-teamgroup-chips]");
  var peopleWrap = root.querySelector("[data-people]");
  var taskBoard = root.querySelector(".task-board");
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

  function setStat(flag, value) {
    var el = root.querySelector('[data-stat="' + flag + '"]');
    if (el) el.textContent = String(value);
  }

  function clearColumns(msg) {
    if (!taskBoard) return;
    taskBoard.querySelectorAll(".task-col").forEach(function (col) {
      var body = col.querySelector(".task-col-body");
      var count = col.querySelector(".k-count");
      if (count) count.textContent = "0";
      if (body) {
        body.innerHTML = msg
          ? '<div class="demand-empty demand-empty--visible">' +
            escapeHtml(msg) +
            "</div>"
          : "";
      }
    });
    setStat("blocked", 0);
    setStat("overdue", 0);
  }

  function taskCard(task) {
    var status = String((task && task.status) || "wait");
    var canDrag = status === "wait" || status === "doing";
    var cls =
      "task-card" +
      (canDrag ? " is-draggable" : "") +
      (task && task.blocked ? " blocked" : "") +
      (task && task.overdue && status !== "done" ? " overdue" : "") +
      (status === "done" ? " done" : "");
    var due =
      task && task.overdue && status !== "done"
        ? '<span class="task-due over">已超期</span>'
        : task && task.deadline
          ? "截止 " + escapeHtml(task.deadline)
          : "";
    var storyLink =
      task && task.storyId
        ? '<span class="task-link" title="' +
          escapeHtml(task.storyTitle || String(task.storyId)) +
          '">所属研需 ' +
          escapeHtml(task.storyTitle || String(task.storyId)) +
          "</span>"
        : "";
    var codeText = escapeHtml((task && (task.displayId || task.id)) || "");
    var titleText = escapeHtml((task && task.title) || "");
    var codeHtml = task && task.url
      ? '<a class="code task-code-link" href="' +
        escapeHtml(task.url) +
        '" target="_blank" rel="noopener noreferrer" draggable="false" title="在禅道打开任务详情">' +
        codeText +
        "</a>"
      : '<span class="code">' + codeText + "</span>";
    var titleHtml = task && task.url
      ? '<a class="task-title task-title-link" href="' +
        escapeHtml(task.url) +
        '" target="_blank" rel="noopener noreferrer" draggable="false" title="' +
        titleText +
        '">' +
        titleText +
        "</a>"
      : '<span class="task-title" title="' + titleText + '">' + titleText + "</span>";
    var owner = String((task && task.owner) || "").trim();
    var assignee = owner
      ? '<span class="mini-avatar">' +
        escapeHtml(firstRune(owner)) +
        "</span>" +
        escapeHtml(owner)
      : "";

    return (
      '<div class="' +
      cls +
      '"' +
      (canDrag ? ' draggable="true"' : "") +
      ' data-task-id="' +
      escapeHtml((task && task.id) || "") +
      '" data-task-status="' +
      escapeHtml(status) +
      '">' +
      '<div class="task-head">' +
      '<span class="kb-type kb-type-task">任务</span>' +
      codeHtml +
      titleHtml +
      "</div>" +
      '<div class="task-meta">' +
      storyLink +
      (task && task.type ? "<span>" + escapeHtml(task.type) + "</span>" : "") +
      "</div>" +
      '<div class="task-footer"><span class="task-due">' +
      due +
      '</span><span class="assignee">' +
      assignee +
      "</span></div></div>"
    );
  }

  function renderColumns(columns, summary) {
    if (!taskBoard) return;
    var colMap = {};
    (columns || []).forEach(function (c) {
      if (c && c.key) colMap[c.key] = c.items || [];
    });
    ["wait", "doing", "done"].forEach(function (key) {
      var colEl = taskBoard.querySelector('.task-col[data-col="' + key + '"]');
      if (!colEl) return;
      var items = colMap[key] || [];
      var body = colEl.querySelector(".task-col-body");
      var count = colEl.querySelector(".k-count");
      if (count) count.textContent = String(items.length);
      if (!body) return;
      if (!items.length) {
        body.innerHTML =
          '<div class="demand-empty demand-empty--visible">暂无任务</div>';
      } else {
        body.innerHTML = items.map(taskCard).join("");
      }
    });
    setStat("blocked", (summary && summary.blocked) || 0);
    setStat("overdue", (summary && summary.overdue) || 0);
  }

  function tasksUrl(account) {
    var acc = String(account || "").trim();
    if (!acc) return TASKS_URL;
    return TASKS_URL + "?account=" + encodeURIComponent(acc);
  }

  function loadTasks(account) {
    if (!taskBoard) return;
    var acc = String(account || "").trim();
    if (!acc) {
      clearColumns("暂无成员");
      return;
    }
    var seq = ++loadSeq;
    clearColumns("加载中…");
    var fetchFn = typeof window.appFetch === "function" ? window.appFetch : fetch;
    fetchFn(tasksUrl(acc), { method: "GET", credentials: "same-origin" })
      .then(function (r) {
        if (!r.ok) throw new Error("http");
        return r.json();
      })
      .then(function (payload) {
        if (seq !== loadSeq) return;
        if (!payload || payload.success !== true) throw new Error("payload");
        renderColumns(payload.columns || [], payload.summary || {});
      })
      .catch(function () {
        if (seq !== loadSeq) return;
        clearColumns("任务加载失败");
      });
  }

  function setSelectedAccount(account, reload) {
    var next = String(account || "").trim();
    if (!next) return;
    var changed = next !== selectedAccount;
    selectedAccount = next;
    if (reload || changed) {
      loadTasks(selectedAccount);
    }
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
    var activeAccount = "";
    var active = group.querySelector(".person.active[data-account]");
    if (active) {
      activeAccount = active.getAttribute("data-account") || "";
    }
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

  // 必选一人：优先当前登录用户，否则组内第一人；不可取消选中。
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
      person = group.querySelector(".person[data-account]");
    }
    if (!person) {
      selectedAccount = "";
      clearColumns("暂无成员");
      return;
    }

    if (person.classList.contains("is-overflow") && allowExpand) {
      expandPeople(group);
    }
    person.classList.add("active");
    setSelectedAccount(person.getAttribute("data-account"), true);
  }

  function selectPerson(group, person) {
    if (!group || !person) return;
    group.querySelectorAll(".person.active").forEach(function (el) {
      el.classList.remove("active");
    });
    person.classList.add("active");
    setSelectedAccount(person.getAttribute("data-account"), true);
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
      clearColumns("暂无所属敏捷小组");
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

      if (person.classList.contains("active")) return;
      selectPerson(group, person);
    });
  }

  ensureDefaultTeamgroup();
})();
