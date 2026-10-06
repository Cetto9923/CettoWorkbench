// =============================================================================
// 文件: web/static/js/po/version-follow.js
// 模块: PO 工作台
// 职责: 版本跟进页交互：窗口条、判断带、筛选、需求行渲染、展开行、依据弹层。
//       AI 块仅在后端返回 aiPilotOn=true 时渲染，开关关闭时页面不出现任何 AI 元素。
// =============================================================================

(function () {
  "use strict";

  var state = {
    windowId: 0,
    scope: "",
    stage: "",
    system: "",
    owner: "",
    judgement: "",
    page: 1,
    pageSize: 20,
    data: null,
  };

  var el = {};
  var requestSerial = 0;

  function $(id) {
    return document.getElementById(id);
  }

  function cacheEls() {
    [
      "vfWindowChips", "vfDaysNum", "vfDaysLabel", "vfOnTrack", "vfRisk", "vfBlocked",
      "vfStageBar", "vfAiBlock", "vfTbody", "vfEmptyState", "vfEmptyTitle",
      "vfEmptyDesc", "vfPager", "vfTable", "vfFilterStage", "vfFilterSystem",
      "vfFilterOwner", "vfFilterClear", "vfBasisPopover", "vfBasisBody",
      "vfBasisTitle", "vfBasisClose", "vfWindowSearch", "vfWindowId",
    ].forEach(function (id) {
      el[id] = $(id);
    });
  }

  var esc = window.escapeHtml;

  function fetchList() {
    var q = new URLSearchParams();
    if (state.windowId) q.set("windowId", state.windowId);
    if (state.scope) q.set("scope", state.scope);
    if (state.stage) q.set("stage", state.stage);
    if (state.system) q.set("system", state.system);
    if (state.owner) q.set("owner", state.owner);
    if (state.judgement) q.set("judgement", state.judgement);
    q.set("page", state.page);
    q.set("pageSize", state.pageSize);
    var serial = ++requestSerial;
    return window.appJson("/version-follow/items?" + q.toString())
      .then(function (j) {
        if (serial !== requestSerial) return;
        state.data = j.data || null;
        render();
      })
      .catch(function (err) {
        if (serial === requestSerial) window.showToast(err.message || "版本窗口加载失败", "error");
      });
  }

  function render() {
    var d = state.data;
    if (!d) return;
    state.windowId = d.windowId || state.windowId;
    renderWindows(d.windows || []);
    renderBand(d);
    renderAI(d);
    renderRows(d);
    renderPager(d);
  }

  function renderWindows(windows) {
    if (!el.vfWindowChips) return;
    var selected = windows.find(function (w) { return w.id === state.windowId; });
    window.initAutocomplete("vfWindowSearch", "vfWindowId", windows.map(function (w) {
      return { value: String(w.id), label: w.name + " · " + w.releaseDate + (w.status === "closed" ? " · 已关闭" : w.released ? " · 已过期" : "") };
    }), { value: String(state.windowId), label: selected ? selected.name + " · " + selected.releaseDate : "", labelOnly: true });
    var visible = windows.filter(function (w) { return w.status !== "closed"; }).slice(0, 12);
    if (selected && !visible.some(function (w) { return w.id === selected.id; })) visible.push(selected);
    el.vfWindowChips.innerHTML = visible
      .map(function (w) {
        var cls = "vf-window-chip" + (w.id === state.windowId ? " is-active" : "") + (w.released ? " is-released" : "");
        return (
          '<button type="button" class="' + cls + '" data-window-id="' + w.id + '">' +
          esc(w.name) +
          ' <span>' + esc(w.releaseDate) + "</span>" +
          " <span>·</span><span> " + w.demandCount + " 需</span>" +
          "</button>"
        );
      })
      .join("");
  }

  function renderBand(d) {
    if (el.vfDaysNum) {
      el.vfDaysNum.textContent = typeof d.distanceDays === "number" ? Math.abs(d.distanceDays) : "—";
      if (el.vfDaysLabel) el.vfDaysLabel.textContent = d.distanceDays < 0 ? "已过上线日（天）" : "距上线（天）";
    }
    if (el.vfOnTrack) el.vfOnTrack.textContent = d.onTrack || 0;
    if (el.vfRisk) el.vfRisk.textContent = d.risk || 0;
    if (el.vfBlocked) el.vfBlocked.textContent = d.blocked || 0;
    if (el.vfStageBar) {
      el.vfStageBar.innerHTML = (d.stageCounts || [])
        .map(function (s) {
          var cls = "vf-stage-seg" + (state.stage === s.stage ? " is-active" : "");
          return (
            '<button type="button" class="' + cls + '" data-stage="' + esc(s.stage) + '">' +
            '<span class="vf-stage-seg-name">' + esc(s.stage) + "</span>" +
            '<span class="vf-stage-seg-count">' + s.count + "</span></button>"
          );
        })
        .join("");
    }
    fillStageFilter(d.stageCounts || []);
  }

  function fillStageFilter(stages) {
    var sel = el.vfFilterStage;
    if (!sel || sel.dataset.filled === "1") return;
    sel.innerHTML = '<option value="">全部</option>' + stages
      .map(function (s) {
        return '<option value="' + esc(s.stage) + '">' + esc(s.stage) + "</option>";
      })
      .join("");
    sel.dataset.filled = "1";
  }

  // renderAI 仅在后端 aiPilotOn 为真时渲染 AI 块；关闭时整块不出现。
  function renderAI(d) {
    if (!el.vfAiBlock) return;
    if (!d.aiPilotOn || !d.orphanTestedCount) {
      el.vfAiBlock.hidden = true;
      el.vfAiBlock.innerHTML = "";
      return;
    }
    el.vfAiBlock.hidden = false;
    el.vfAiBlock.innerHTML =
      '<span class="vf-ai-text">AI 发现 · 待核实：共有 ' + d.orphanTestedCount +
      " 条需求已到测试后阶段但未挂版本窗口</span>" +
      '<button type="button" class="vf-ai-detail-btn" id="vfAiDetail">查看依据</button>';
    var btn = $("vfAiDetail");
    if (btn) btn.addEventListener("click", openOrphanBasis);
  }

  function openOrphanBasis() {
    fetch("/version-follow/orphan-evidence", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    })
      .then(function (r) {
        return r.json();
      })
      .then(function (j) {
        showBasis("已到测试后阶段但未挂版本窗口", j.data || []);
      })
      .catch(function () {
        // 依据拉取失败静默。
      });
  }

  function showBasis(title, findings) {
    if (!el.vfBasisPopover) return;
    el.vfBasisTitle.textContent = title;
    el.vfBasisBody.innerHTML = findings.length
      ? findings
          .map(function (f) {
            return (
              '<div class="vf-basis-item">' +
              '<div class="vf-basis-item-title">' + esc(f.ruleName) + "</div>" +
              '<div class="vf-basis-item-desc">' + esc(f.field) + "：" + esc(f.value) + "</div>" +
              '<div class="vf-basis-item-desc"><a href="' + esc(f.targetUrl) + '">查看对象</a></div>' +
              "</div>"
            );
          })
          .join("")
      : '<p class="vf-empty-desc">暂无依据</p>';
    el.vfBasisPopover.hidden = false;
  }

  function closeBasis() {
    if (el.vfBasisPopover) el.vfBasisPopover.hidden = true;
  }

  function judgementCell(it) {
    if (!it.judgement) {
      return '<span class="vf-judgement is-none">暂无判断</span>';
    }
    var map = { ontrack: "可按期", risk: "风险", blocked: "阻塞" };
    return (
      '<span class="vf-judgement is-' + esc(it.judgement) + '">' + esc(map[it.judgement] || it.judgement) + "</span>" +
      (it.reason ? '<div class="vf-reason">' + esc(it.reason) + "</div>" : "")
    );
  }

  function findingsCell(it) {
    if (!it.findings || !it.findings.length) return "";
    return (
      '<div class="vf-findings">' +
      it.findings
        .map(function (f, i) {
          return (
            '<button type="button" class="vf-finding-link" data-finding="' + i + '">AI 发现 · 待核实：' +
            esc(f.ruleName) + "</button>"
          );
        })
        .join("") +
      "</div>"
    );
  }

  function rowHtml(it, i) {
    var stay = it.stayDays > 0 ? "已停留 " + it.stayDays + " 天" : "—";
    var disabled = it.actionEnabled ? "" : " aria-disabled=\"true\" disabled";
    return (
      '<tr class="vf-row" data-index="' + i + '">' +
      '<td><button type="button" class="vf-expand-btn" data-expand="' + i + '" aria-label="展开">▸</button></td>' +
      "<td>" +
      '<span class="vf-req-no">#' + esc(it.demandNo) + "</span> " +
      '<div class="vf-req-title">' + esc(it.title) + "</div>" +
      '<div class="vf-req-meta">' + esc(it.system) + " · " + esc(it.priority) + "</div>" +
      "</td>" +
      "<td>" + esc(it.stage) + "</td>" +
      '<td><span class="vf-stay-days">' + stay + "</span>" +
      (it.overdue ? '<div class="vf-overdue">' + esc(it.overdue) + "</div>" : "") + "</td>" +
      "<td>" + judgementCell(it) + findingsCell(it) + "</td>" +
      "<td>" + esc(it.owner) + "</td>" +
      '<td><button type="button" class="vf-action-btn"' + disabled + ">" + esc(it.actionLabel) + "</button>" +
      (it.actionReason ? '<span class="vf-btn-hint">' + esc(it.actionReason) + "</span>" : "") +
      "</td></tr>"
    );
  }

  function detailHtml(d) {
    if (!d) return "";
    var checks = (d.checks || [])
      .map(function (c) {
        return (
          '<div class="vf-check-item"><span class="vf-check-label">' + esc(c.label) +
          (c.phase2 ? ' <span class="vf-phase2-tag">二期</span>' : "") +
          '</span><span class="vf-check-state is-' + esc(c.state) + '">' +
          esc(c.detail) + "</span></div>"
        );
      })
      .join("");
    return (
      '<tr class="vf-detail-row"><td colspan="7"><div class="vf-detail">' +
      '<div class="vf-checklist">' + checks + "</div>" +
      '<div class="vf-source-block">来源需求：本期不展示' +
      '<span class="vf-phase2-tag">二期</span></div>' +
      "</div></td></tr>"
    );
  }

  function renderRows(d) {
    var items = d.items || [];
    var hasRows = items.length > 0;
    if (el.vfTable) el.vfTable.hidden = !hasRows;
    if (el.vfEmptyState) el.vfEmptyState.hidden = hasRows;
    if (!hasRows) {
      if (el.vfTbody) el.vfTbody.innerHTML = "";
      if (el.vfEmptyTitle) {
        el.vfEmptyTitle.textContent =
          (d.windows || []).length === 0 ? "暂无版本窗口" : "该版本窗口暂无关联需求";
      }
      if (el.vfEmptyDesc) {
        el.vfEmptyDesc.textContent =
          (d.windows || []).length === 0
            ? "请先在需求排期中创建版本窗口。"
            : "可在需求排期中把需求关联到当前版本窗口。";
      }
      var act = $("vfEmptyAction");
      if (act) act.hidden = (d.windows || []).length === 0;
      return;
    }
    if (el.vfTbody) {
      el.vfTbody.innerHTML = items
        .map(function (it, i) {
          return rowHtml(it, i);
        })
        .join("");
    }
    fillOwnerFilter(items);
  }

  function fillOwnerFilter(items) {
    var sel = el.vfFilterOwner;
    if (!sel || sel.dataset.filled === "1") return;
    var owners = [];
    items.forEach(function (it) {
      if (it.owner && it.owner !== "暂无" && owners.indexOf(it.owner) < 0) owners.push(it.owner);
    });
    sel.innerHTML = '<option value="">全部</option>' + owners
      .map(function (o) {
        return '<option value="' + esc(o) + '">' + esc(o) + "</option>";
      })
      .join("");
    sel.dataset.filled = "1";
  }

  function renderPager(d) {
    if (!el.vfPager) return;
    var total = d.total || 0;
    var pages = Math.max(1, Math.ceil(total / (d.pageSize || 20)));
    if (pages <= 1) {
      el.vfPager.innerHTML = "";
      return;
    }
    var html = '<button type="button" class="vf-pager-btn" data-page="' + (state.page - 1) + '"' +
      (state.page <= 1 ? " disabled" : "") + ">上一页</button>";
    for (var p = 1; p <= pages; p += 1) {
      html += '<button type="button" class="vf-pager-btn' + (p === state.page ? " is-active" : "") +
        '" data-page="' + p + '">' + p + "</button>";
    }
    html += '<button type="button" class="vf-pager-btn" data-page="' + (state.page + 1) + '"' +
      (state.page >= pages ? " disabled" : "") + ">下一页</button>";
    el.vfPager.innerHTML = html;
  }

  function selectWindow(id) {
    if (!Number(id)) return;
    state.windowId = Number(id);
    state.page = 1;
    fetchList();
  }

  function bindEvents() {
    if (el.vfWindowId) el.vfWindowId.addEventListener("change", function () { if (Number(el.vfWindowId.value) !== state.windowId) selectWindow(el.vfWindowId.value); });
    if (el.vfWindowChips) {
      el.vfWindowChips.addEventListener("click", function (e) {
        var b = e.target.closest("[data-window-id]");
        if (!b) return;
        selectWindow(b.dataset.windowId);
      });
    }
    if (el.vfStageBar) {
      el.vfStageBar.addEventListener("click", function (e) {
        var b = e.target.closest("[data-stage]");
        if (!b) return;
        state.stage = state.stage === b.dataset.stage ? "" : b.dataset.stage;
        if (el.vfFilterStage) el.vfFilterStage.value = state.stage;
        state.page = 1;
        fetchList();
      });
    }
    document.querySelectorAll(".vf-scope-chip").forEach(function (b) {
      b.addEventListener("click", function () {
        document.querySelectorAll(".vf-scope-chip").forEach(function (x) {
          x.classList.remove("is-active");
        });
        b.classList.add("is-active");
        state.scope = b.dataset.scope || "";
        state.page = 1;
        fetchList();
      });
    });
    document.querySelectorAll(".vf-count-chip").forEach(function (b) {
      b.addEventListener("click", function () {
        document.querySelectorAll(".vf-count-chip").forEach(function (x) {
          x.classList.remove("is-active");
        });
        b.classList.add("is-active");
        state.judgement = b.dataset.judgement || "";
        state.page = 1;
        fetchList();
      });
    });
    [["vfFilterStage", "stage"], ["vfFilterSystem", "system"], ["vfFilterOwner", "owner"]]
      .forEach(function (pair) {
        var node = el[pair[0]];
        if (!node) return;
        node.addEventListener("change", function () {
          state[pair[1]] = node.value;
          state.page = 1;
          fetchList();
        });
      });
    if (el.vfFilterClear) {
      el.vfFilterClear.addEventListener("click", function () {
        state.stage = state.system = state.owner = state.judgement = "";
        state.page = 1;
        if (el.vfFilterStage) el.vfFilterStage.value = "";
        if (el.vfFilterSystem) el.vfFilterSystem.value = "";
        if (el.vfFilterOwner) el.vfFilterOwner.value = "";
        document.querySelectorAll(".vf-count-chip, .vf-scope-chip").forEach(function (x) {
          x.classList.remove("is-active");
        });
        if (document.querySelector('.vf-scope-chip[data-scope=""]')) {
          document.querySelector('.vf-scope-chip[data-scope=""]').classList.add("is-active");
          state.scope = "";
        }
        fetchList();
      });
    }
    if (el.vfPager) {
      el.vfPager.addEventListener("click", function (e) {
        var b = e.target.closest("[data-page]");
        if (!b || b.disabled) return;
        state.page = Number(b.dataset.page);
        fetchList();
      });
    }
    if (el.vfTbody) {
      el.vfTbody.addEventListener("click", function (e) {
        var exp = e.target.closest("[data-expand]");
        if (exp) {
          toggleRow(Number(exp.dataset.expand));
          return;
        }
        var fnd = e.target.closest("[data-finding]");
        if (fnd) {
          var row = fnd.closest("tr");
          var idx = Number((row && row.dataset.index) || 0);
          var it = ((state.data && state.data.items) || [])[idx] || {};
          showBasis("发现依据", it.findings || []);
        }
      });
    }
    if (el.vfBasisClose) el.vfBasisClose.addEventListener("click", closeBasis);
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") closeBasis();
    });
  }

  function toggleRow(i) {
    var row = el.vfTbody.querySelector('.vf-row[data-index="' + i + '"]');
    if (!row) return;
    var next = row.nextElementSibling;
    if (next && next.classList.contains("vf-detail-row")) {
      next.remove();
      row.classList.remove("is-open");
      return;
    }
    var details = (state.data && state.data.details) || [];
    row.insertAdjacentHTML("afterend", detailHtml(details[i]));
    row.classList.add("is-open");
  }

  function init() {
    if (!$("vfTbody")) return;
    cacheEls();
    bindEvents();
    fetchList();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
