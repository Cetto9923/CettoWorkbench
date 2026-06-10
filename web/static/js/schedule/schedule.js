(function ($) {
  "use strict";

  var $root = $("#page-schedule");
  if (!$root.length) {
    return;
  }

  var listTitles = {
    bizReq: "业务需求列表",
    independentRD: "独立研发需求列表",
  };

  var scheduleVersionWindowModalMode = "edit";
  var scheduleVersionCreateDraft = null;

  var windowTypeLabels = {
    regular: "常规",
    fast: "快速",
    urgent: "紧急",
  };

  var SCHEDULE_MATCHING_PLANS_URL = "/po/schedule/matching-plans";

  var scheduleIterationDefinitions = [
    {
      key: "current",
      label: "26-0524窗口",
      start: "2026-05-11",
      end: "2026-05-24",
      planTestDone: "2026-05-20",
      testDone: "2026-05-22",
      acceptDone: "2026-05-23",
      online: "2026-05-24",
    },
    {
      key: "next",
      label: "26-0607窗口",
      start: "2026-05-25",
      end: "2026-06-07",
      planTestDone: "2026-06-03",
      testDone: "2026-06-05",
      acceptDone: "2026-06-06",
      online: "2026-06-07",
    },
    {
      key: "nextnext",
      label: "26-0621窗口",
      start: "2026-06-08",
      end: "2026-06-21",
      planTestDone: "2026-06-17",
      testDone: "2026-06-19",
      acceptDone: "2026-06-20",
      online: "2026-06-21",
    },
    {
      key: "recent",
      label: "26-0510已发",
      start: "2026-04-27",
      end: "2026-05-10",
      planTestDone: "2026-05-06",
      testDone: "2026-05-08",
      acceptDone: "2026-05-09",
      online: "2026-05-10",
    },
  ];

  function escapeJsString(value) {
    return String(value == null ? "" : value)
      .replace(/\\/g, "\\\\")
      .replace(/'/g, "\\'");
  }

  function getScheduleProductOptions() {
    var tpl = document.getElementById("scheduleProductOptions");
    if (!tpl || !tpl.content) {
      return [];
    }
    var labels = tpl.content.querySelectorAll("label[data-product-id]");
    var items = [];
    labels.forEach(function (label) {
      var id = Number(label.getAttribute("data-product-id"));
      if (!id) {
        return;
      }
      items.push({
        id: id,
        name: label.getAttribute("data-product-name") || "",
      });
    });
    return items;
  }

  function getProductNameById(productId) {
    var id = Number(productId);
    var found = getScheduleProductOptions().find(function (item) {
      return item.id === id;
    });
    return found ? found.name : "系统";
  }

  function buildProductCheckboxHtml(selectedIds) {
    var tpl = document.getElementById("scheduleProductOptions");
    if (!tpl || !tpl.content) {
      return '<div class="schedule-create-empty">暂无关联系统</div>';
    }
    var labels = tpl.content.querySelectorAll("label[data-product-id]");
    if (!labels.length) {
      return '<div class="schedule-create-empty">暂无关联系统</div>';
    }
    var selected = new Set((selectedIds || []).map(Number));
    var parts = [];
    labels.forEach(function (label) {
      var id = Number(label.getAttribute("data-product-id"));
      var name = label.getAttribute("data-product-name") || "";
      var span = label.querySelector("span");
      var displayName = span ? span.textContent : name;
      parts.push(
        '<label class="chk">' +
          '<input type="checkbox" name="productIds" value="' +
          id +
          '" ' +
          (selected.has(id) ? "checked" : "") +
          ' onchange="toggleScheduleCreateSystem(' +
          id +
          ", '" +
          escapeJsString(name) +
          '\', this.checked)">' +
          "<span>" +
          escapeHtml(displayName) +
          "</span>" +
          "</label>"
      );
    });
    return parts.join("");
  }

  function setScopeChip($chip) {
    $root.find(".schedule-scope-chip").removeClass("active");
    $chip.addClass("active");
  }

  function setDataTab($tab) {
    $root.find(".schedule-data-tab").removeClass("active");
    $tab.addClass("active");
    var type = $tab.data("type") || "bizReq";
    $("#scheduleListTitle").text(listTitles[type] || listTitles.bizReq);
  }

  function toggleMoreFilters() {
    $("#scheduleMoreFilters").toggleClass("open");
  }

  function clearFilters() {
    $root.find(".schedule-scope-chip").removeClass("active");
    $root.find('.schedule-scope-chip[data-scope="notClosed"]').addClass("active");
    $("#scheduleMoreFilters").removeClass("open");
    $("#scheduleFilterAgile, #scheduleFilterSystem, #scheduleFilterStatus, #scheduleFilterAnomaly").val("");
    $("#scheduleSearch, #scheduleCreatorFilter, #scheduleSourceFilter, #scheduleDeptFilter").val("");
    $("#scheduleFilterPri, #scheduleFilterWindow, #scheduleFilterDev, #scheduleFilterTest, #scheduleFilterAccept").val("");
    $("#scheduleFilterRole").val("all");
  }

  function toggleBizChildren(parentId, $toggle) {
    var $children = $root.find('tr.schedule-child-row[data-parent="' + parentId + '"]');
    var collapsed = $toggle.hasClass("is-collapsed");

    if (collapsed) {
      $toggle.removeClass("is-collapsed fa-chevron-right").addClass("fa-chevron-down");
      $children.removeClass("is-hidden");
      return;
    }

    $toggle.addClass("is-collapsed").removeClass("fa-chevron-down").addClass("fa-chevron-right");
    $children.addClass("is-hidden");
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function scheduleAddDaysIso(isoDate, deltaDays) {
    var text = String(isoDate || "").slice(0, 10);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(text)) {
      return "";
    }
    var parts = text.split("-");
    var date = new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
    date.setDate(date.getDate() + deltaDays);
    var month = String(date.getMonth() + 1).padStart(2, "0");
    var day = String(date.getDate()).padStart(2, "0");
    return date.getFullYear() + "-" + month + "-" + day;
  }

  function getDefaultTeamgroupID() {
    var tpl = document.getElementById("scheduleTeamgroupOptions");
    if (!tpl || !tpl.content) {
      return "";
    }
    var first = tpl.content.querySelector("option");
    return first ? String(first.value) : "";
  }

  function buildTeamgroupSelectOptions(selectedID) {
    var tpl = document.getElementById("scheduleTeamgroupOptions");
    if (!tpl || !tpl.content) {
      return '<option value="">暂无敏捷小组</option>';
    }
    var options = tpl.content.querySelectorAll("option");
    if (!options.length) {
      return '<option value="">暂无敏捷小组</option>';
    }
    var parts = [];
    options.forEach(function (opt) {
      var value = String(opt.value);
      var selected = String(selectedID || getDefaultTeamgroupID()) === value ? " selected" : "";
      parts.push(
        '<option value="' +
          escapeHtml(value) +
          '"' +
          selected +
          ">" +
          escapeHtml(opt.textContent || "") +
          "</option>"
      );
    });
    return parts.join("");
  }

  function buildDefaultScheduleWindowNameFromOnline(online) {
    var d = String(online || "").slice(0, 10);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(d)) {
      return "新版本窗口";
    }
    var yy = d.slice(2, 4);
    var md = d.slice(5, 7) + d.slice(8, 10);
    return yy + "-" + md + "窗口";
  }

  function createScheduleVersionCreateDraft() {
    return {
      name: "",
      start: "",
      end: "",
      useOrgTemplate: false,
      teamgroupId: getDefaultTeamgroupID(),
      windowType: "regular",
      planTestDone: "",
      testDone: "",
      acceptDone: "",
      online: "",
      productIds: [],
      systemPlanDraft: {},
      systemPlanMatch: {},
      _lastOnlineAuto: "",
      _lastPlanNameOnline: "",
      _nameAutoGenerated: true,
      _startManual: false,
      _startAutoGenerated: false,
    };
  }

  function hasValidScheduleOnlineDate(draft) {
    var online = String((draft || {}).online || "").slice(0, 10);
    return /^\d{4}-\d{2}-\d{2}$/.test(online);
  }

  function formatScheduleDate(value) {
    var text = String(value == null ? "" : value).slice(0, 10);
    return /^\d{4}-\d{2}-\d{2}$/.test(text) ? text : "-";
  }

  function getScheduleVersionCreateDraft() {
    if (!scheduleVersionCreateDraft) {
      scheduleVersionCreateDraft = createScheduleVersionCreateDraft();
    }
    return scheduleVersionCreateDraft;
  }

  function getScheduleVersionModalDraft() {
    if (scheduleVersionWindowModalMode === "create") {
      return getScheduleVersionCreateDraft();
    }
    return null;
  }

  function getScheduleVersionModalWindowMeta() {
    var d = getScheduleVersionCreateDraft();
    return {
      windowKey: "",
      online: d.online || "",
      windowName: d.name || buildDefaultScheduleWindowNameFromOnline(d.online),
      planTestDone: d.planTestDone || "",
      testDone: d.testDone || "",
      acceptDone: d.acceptDone || "",
    };
  }

  function syncScheduleDraftPlanNamesFromWindow(d) {
    var windowName = d.name || buildDefaultScheduleWindowNameFromOnline(d.online);
    Object.keys(d.systemPlanDraft || {}).forEach(function (sys) {
      var row = d.systemPlanDraft[sys];
      if (row && !row._customName) {
        row.planName = windowName;
      }
    });
  }

  function syncScheduleDraftPlanBeginFromWindow(d) {
    Object.keys(d.systemPlanDraft || {}).forEach(function (sys) {
      var row = d.systemPlanDraft[sys];
      if (row && !row._customBegin) {
        row.planBegin = d.start || "";
      }
    });
  }

  function resetSchedulePlanAreaFromOnline(d) {
    var windowName = d.name || buildDefaultScheduleWindowNameFromOnline(d.online);
    Object.keys(d.systemPlanDraft || {}).forEach(function (sys) {
      var row = d.systemPlanDraft[sys];
      if (!row) {
        return;
      }
      row.planName = windowName;
      row.planBegin = d.start || "";
      row._customName = false;
      row._customBegin = false;
    });
  }

  function applyScheduleOnlineDerivedFields(d, online) {
    if (!/^\d{4}-\d{2}-\d{2}$/.test(online)) {
      return false;
    }
    d.name = buildDefaultScheduleWindowNameFromOnline(online);
    d._nameAutoGenerated = true;
    d._lastOnlineAuto = online;
    d.end = online;
    d.start = scheduleAddDaysIso(online, -13);
    d._startAutoGenerated = true;
    d._startManual = false;
    if (d.useOrgTemplate) {
      applyScheduleCreateOrgTemplateDates();
    } else {
      d.planTestDone = scheduleAddDaysIso(online, -4);
      d.testDone = scheduleAddDaysIso(online, -2);
      d.acceptDone = scheduleAddDaysIso(online, -1);
    }
    resetSchedulePlanAreaFromOnline(d);
    d._lastPlanNameOnline = online;
    return true;
  }

  function clearScheduleOnlineDerivedFields(d) {
    if (d._nameAutoGenerated) {
      d.name = "";
    }
    d._lastOnlineAuto = "";
    d.end = "";
    if (!d.useOrgTemplate) {
      d.planTestDone = "";
      d.testDone = "";
      d.acceptDone = "";
    }
    if (d._startAutoGenerated) {
      d.start = "";
      d._startManual = false;
    }
    syncScheduleDraftPlanBeginFromWindow(d);
    Object.keys(d.systemPlanMatch || {}).forEach(function (key) {
      delete d.systemPlanMatch[key];
    });
  }

  function ensureScheduleModalSystemPlanDraft(productId, productName) {
    var d = getScheduleVersionModalDraft();
    if (!d) {
      return { syncCreate: true, planName: "", planBegin: "" };
    }
    var meta = getScheduleVersionModalWindowMeta();
    var key = String(productId);
    if (!d.systemPlanDraft) {
      d.systemPlanDraft = {};
    }
    if (!d.systemPlanDraft[key]) {
      d.systemPlanDraft[key] = {
        syncCreate: true,
        planName: meta.windowName || buildDefaultScheduleWindowNameFromOnline(meta.online),
        planBegin: d.start || "",
        productName: productName || getProductNameById(productId),
      };
    }
    return d.systemPlanDraft[key];
  }

  function fetchMatchingPlansForProduct(productId, productName) {
    var d = getScheduleVersionCreateDraft();
    var endDate = String(d.online || "").slice(0, 10);
    if (!/^\d{4}-\d{2}-\d{2}$/.test(endDate)) {
      return;
    }
    var key = String(productId);
    if (!d.systemPlanMatch) {
      d.systemPlanMatch = {};
    }
    d.systemPlanMatch[key] = {
      loading: true,
      productName: productName || getProductNameById(productId),
    };
    renderScheduleVersionWindowModalBody();

    $.ajax({
      url: SCHEDULE_MATCHING_PLANS_URL,
      method: "GET",
      data: {
        product_id: productId,
        end_date: endDate,
      },
    })
      .done(function (resp) {
        var draft = getScheduleVersionCreateDraft();
        var plans = (resp && resp.plans) || [];
        draft.systemPlanMatch[key] = {
          loading: false,
          productName: productName || getProductNameById(productId),
          hasMatch: !!(resp && resp.has_match),
          plans: plans,
          selectedPlanIds: plans
            .map(function (plan) {
              return Number(plan.id);
            })
            .filter(function (id) {
              return id > 0;
            }),
        };
        if (!resp || !resp.has_match) {
          var pendingDraft = ensureScheduleModalSystemPlanDraft(productId, productName);
          pendingDraft.planName = draft.name || buildDefaultScheduleWindowNameFromOnline(draft.online);
          pendingDraft.planBegin = draft.start || "";
          pendingDraft._customName = false;
          pendingDraft._customBegin = false;
        }
        renderScheduleVersionWindowModalBody();
      })
      .fail(function () {
        var draft = getScheduleVersionCreateDraft();
        draft.systemPlanMatch[key] = {
          loading: false,
          productName: productName || getProductNameById(productId),
          hasMatch: false,
          plans: [],
          error: true,
        };
        var pendingDraft = ensureScheduleModalSystemPlanDraft(productId, productName);
        pendingDraft.planName = draft.name || buildDefaultScheduleWindowNameFromOnline(draft.online);
        pendingDraft.planBegin = draft.start || "";
        pendingDraft._customName = false;
        pendingDraft._customBegin = false;
        renderScheduleVersionWindowModalBody();
      });
  }

  function refreshMatchingPlansForSelectedProducts() {
    var d = getScheduleVersionCreateDraft();
    (d.productIds || []).forEach(function (productId) {
      var key = String(productId);
      var name =
        ((d.systemPlanMatch || {})[key] && d.systemPlanMatch[key].productName) ||
        getProductNameById(productId);
      fetchMatchingPlansForProduct(productId, name);
    });
  }

  function applyScheduleCreateOrgTemplateDates() {
    var d = getScheduleVersionCreateDraft();
    var tpl =
      scheduleIterationDefinitions.find(function (item) {
        return item.online === d.online;
      }) ||
      scheduleIterationDefinitions.find(function (item) {
        return d.online >= item.start && d.online <= item.end;
      });
    if (tpl) {
      d.planTestDone = tpl.planTestDone || "";
      d.testDone = tpl.testDone || "";
      d.acceptDone = tpl.acceptDone || "";
      d.start = tpl.start || d.start;
      d.end = tpl.end || d.end;
      return;
    }
    if (d.online) {
      d.planTestDone = scheduleAddDaysIso(d.online, -4);
      d.testDone = scheduleAddDaysIso(d.online, -2);
      d.acceptDone = scheduleAddDaysIso(d.online, -1);
      d.start = scheduleAddDaysIso(d.online, -13);
      d.end = d.online;
    }
  }

  function renderScheduleVersionSystemsAndPlansSection() {
    var d = getScheduleVersionModalDraft();
    if (!d) {
      return "";
    }
    var meta = getScheduleVersionModalWindowMeta();
    var systemChecks = buildProductCheckboxHtml(d.productIds || []);

    var planRows =
      (d.productIds || [])
        .map(function (productId) {
          var key = String(productId);
          var match = (d.systemPlanMatch || {})[key] || {};
          var productName = match.productName || getProductNameById(productId);

          if (match.loading) {
            return (
              '<div class="schedule-create-plan-card">' +
              '<div class="schedule-create-plan-title">' +
              "<span>" +
              escapeHtml(productName) +
              "</span>" +
              "<span>查询中</span>" +
              "</div>" +
              '<div class="schedule-create-plan-summary">' +
              '<i class="fas fa-spinner fa-spin"></i>' +
              "<span>正在匹配系统计划…</span>" +
              "</div>" +
              "</div>"
            );
          }

          if (!hasValidScheduleOnlineDate(d) && !match.loading && !match.error) {
            return (
              '<div class="schedule-create-plan-card">' +
              '<div class="schedule-create-plan-title">' +
              "<span>" +
              escapeHtml(productName) +
              "</span>" +
              "<span>待匹配</span>" +
              "</div>" +
              '<div class="schedule-create-plan-meta">请先填写预计上线日期，再匹配系统计划。</div>' +
              "</div>"
            );
          }

          if (match.hasMatch && match.plans && match.plans.length) {
            return match.plans
              .map(function (plan) {
                var onlineDate = formatScheduleDate(plan.end || meta.online);
                return (
                  '<div class="schedule-create-plan-card schedule-create-plan-card--existing">' +
                  '<div class="schedule-create-plan-title">' +
                  "<span>" +
                  escapeHtml(productName) +
                  "</span>" +
                  '<span class="schedule-create-plan-tag">已有计划</span>' +
                  "</div>" +
                  '<div class="schedule-create-plan-summary">' +
                  '<i class="fas fa-check-circle"></i>' +
                  "<span>已有系统计划</span>" +
                  '<strong class="schedule-create-plan-name">' +
                  escapeHtml(plan.title || "") +
                  "</strong>" +
                  "<span>上线 " +
                  escapeHtml(onlineDate) +
                  "</span>" +
                  "</div>" +
                  "</div>"
                );
              })
              .join("");
          }

          var row = ensureScheduleModalSystemPlanDraft(productId, productName);
          return (
            '<div class="schedule-create-plan-card">' +
            '<div class="schedule-create-plan-title">' +
            "<span>" +
            escapeHtml(productName) +
            "</span>" +
            "<span>待建计划</span>" +
            "</div>" +
            '<label class="chk schedule-create-plan-toggle">' +
            '<input type="checkbox" ' +
            (row.syncCreate ? "checked" : "") +
            ' onchange="toggleScheduleCreateSystemPlanSync(' +
            productId +
            ",this.checked)\">" +
            "<span>同步创建对应系统计划（日期与本窗口一致）</span>" +
            "</label>" +
            '<div class="form-group" style="margin:0;' +
            (row.syncCreate ? "" : "opacity:.5") +
            '">' +
            '<label class="form-label">计划名称</label>' +
            '<input type="text" class="form-input" value="' +
            escapeHtml(row.planName || "") +
            '" ' +
            (row.syncCreate ? "" : "disabled") +
            ' placeholder="默认与窗口名称一致，可修改" oninput="updateScheduleCreateSystemPlanName(' +
            productId +
            ",this.value)\">" +
            "</div>" +
            '<div class="schedule-create-plan-meta">里程碑：开始 ' +
            escapeHtml(formatScheduleDate(row.planBegin || d.start)) +
            " / 提测 " +
            escapeHtml(formatScheduleDate(meta.planTestDone)) +
            " / 测试 " +
            escapeHtml(formatScheduleDate(meta.testDone)) +
            " / 验收 " +
            escapeHtml(formatScheduleDate(meta.acceptDone)) +
            " / 上线 " +
            escapeHtml(formatScheduleDate(meta.online)) +
            "</div>" +
            "</div>"
          );
        })
        .join("") ||
      '<div class="schedule-create-empty">请先勾选关联系统，将显示各系统的计划情况。</div>';

    return (
      '<div class="schedule-create-section">' +
      '<div class="schedule-create-section-head"><strong>关联系统</strong></div>' +
      '<div class="schedule-create-grid schedule-create-grid--systems schedule-create-systems">' +
      systemChecks +
      "</div>" +
      "</div>" +
      '<div class="schedule-create-section">' +
      '<div class="schedule-create-section-head"><strong>系统计划</strong></div>' +
      '<div class="schedule-create-plan-grid">' +
      planRows +
      "</div>" +
      "</div>"
    );
  }

  function renderScheduleCreateVersionWindowModalBody() {
    var d = getScheduleVersionCreateDraft();
    var host = document.getElementById("scheduleVersionWindowModalBody");
    if (!host) {
      return;
    }
    var dis = d.useOrgTemplate ? "disabled" : "";

    host.innerHTML =
      '<div class="schedule-create-modal">' +
      '<div class="schedule-create-note">' +
      "创建<strong>版本窗口</strong>并可选同步创建各系统的<strong>系统计划</strong>。窗口名称默认按预计上线日生成（如 26-0701窗口），系统计划名称默认与窗口名称一致，可按需微调。" +
      "</div>" +
      '<div class="schedule-create-grid schedule-create-grid--4">' +
      '<div class="form-group">' +
      '<label class="form-label">预计上线 <span style="color:var(--red)">*</span></label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.online || "") +
      "\" onchange=\"updateScheduleCreateDraftField('online',this.value)\">" +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">窗口名称 <span style="color:var(--red)">*</span></label>' +
      '<input type="text" class="form-input" value="' +
      escapeHtml(d.name || "") +
      '" placeholder="根据预计上线自动生成" onchange="updateScheduleCreateDraftField(\'name\',this.value)">' +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">窗口开始</label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.start || "") +
      "\" onchange=\"updateScheduleCreateDraftField('start',this.value)\">" +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">窗口结束</label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.end || "") +
      "\" onchange=\"updateScheduleCreateDraftField('end',this.value)\">" +
      "</div>" +
      "</div>" +
      '<div class="schedule-create-section">' +
      '<div class="schedule-create-section-head"><strong>窗口归属</strong></div>' +
      '<div class="schedule-create-inline-row">' +
      '<div class="form-group">' +
      '<label class="form-label">敏捷小组（窗口默认）</label>' +
      '<select class="form-select" style="width:100%" onchange="updateScheduleCreateDraftField(\'teamgroupId\',this.value)">' +
      buildTeamgroupSelectOptions(d.teamgroupId) +
      "</select>" +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">发布类型（窗口参考）</label>' +
      '<select class="form-select" style="width:100%" onchange="updateScheduleCreateDraftField(\'windowType\',this.value)">' +
      ["regular", "fast", "urgent"]
        .map(function (wt) {
          return (
            '<option value="' +
            wt +
            '" ' +
            (d.windowType === wt ? "selected" : "") +
            ">" +
            escapeHtml(windowTypeLabels[wt]) +
            "</option>"
          );
        })
        .join("") +
      "</select>" +
      "</div>" +
      '<label class="chk schedule-create-inline-check">' +
      '<input type="checkbox" ' +
      (d.useOrgTemplate ? "checked" : "") +
      ' onchange="toggleScheduleCreateUseOrgTemplate(this.checked)">' +
      "<span>里程碑跟随组织窗口</span>" +
      "</label>" +
      "</div>" +
      "</div>" +
      '<div class="schedule-create-section">' +
      '<div class="schedule-create-section-head">' +
      "<strong>关键时间（本窗口）</strong>" +
      '<span class="schedule-create-inline-meta">上线日期：' +
      escapeHtml(d.online || "-") +
      "</span>" +
      "</div>" +
      '<div class="schedule-create-grid schedule-create-grid--3">' +
      '<div class="form-group">' +
      '<label class="form-label">预计提测 / 开发完成</label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.planTestDone || "") +
      '" ' +
      dis +
      " onchange=\"updateScheduleCreateDraftField('planTestDone',this.value);getScheduleVersionCreateDraft().useOrgTemplate=false\">" +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">预计测试完成</label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.testDone || "") +
      '" ' +
      dis +
      " onchange=\"updateScheduleCreateDraftField('testDone',this.value);getScheduleVersionCreateDraft().useOrgTemplate=false\">" +
      "</div>" +
      '<div class="form-group">' +
      '<label class="form-label">预计验收完成</label>' +
      '<input type="date" class="form-input" value="' +
      escapeHtml(d.acceptDone || "") +
      '" ' +
      dis +
      " onchange=\"updateScheduleCreateDraftField('acceptDone',this.value);getScheduleVersionCreateDraft().useOrgTemplate=false\">" +
      "</div>" +
      "</div>" +
      "</div>" +
      renderScheduleVersionSystemsAndPlansSection() +
      "</div>";
  }

  function renderScheduleVersionWindowModalBody() {
    if (scheduleVersionWindowModalMode === "create") {
      renderScheduleCreateVersionWindowModalBody();
    }
  }

  function openScheduleCreateVersionWindowModal() {
    scheduleVersionWindowModalMode = "create";
    scheduleVersionCreateDraft = createScheduleVersionCreateDraft();
    var titleEl = document.getElementById("scheduleVersionWindowModalTitle");
    var saveBtn = document.getElementById("scheduleVersionWindowModalSaveBtn");
    if (titleEl) {
      titleEl.textContent = "新建版本窗口（SM/PO）";
    }
    if (saveBtn) {
      saveBtn.textContent = "保存";
    }
    renderScheduleVersionWindowModalBody();
    document.getElementById("scheduleVersionWindowModal").classList.add("show");
    document.getElementById("scheduleVersionWindowModalOverlay").classList.add("show");
  }

  function closeScheduleVersionWindowModal() {
    var modal = document.getElementById("scheduleVersionWindowModal");
    var overlay = document.getElementById("scheduleVersionWindowModalOverlay");
    if (modal) {
      modal.classList.remove("show");
    }
    if (overlay) {
      overlay.classList.remove("show");
    }
    scheduleVersionWindowModalMode = "edit";
    scheduleVersionCreateDraft = null;
    var saveBtn = document.getElementById("scheduleVersionWindowModalSaveBtn");
    if (saveBtn) {
      saveBtn.textContent = "保存";
    }
  }

  function collectScheduleCreateSavePayload() {
    var d = getScheduleVersionCreateDraft();
    return {
      releaseDate: d.online || "",
      name: d.name || "",
      startDate: d.start || "",
      endDate: d.end || "",
      teamgroupId: d.teamgroupId || "",
      windowType: d.windowType || "regular",
      planTestDone: d.planTestDone || "",
      testDone: d.testDone || "",
      acceptDone: d.acceptDone || "",
      products: (d.productIds || []).map(function (productId) {
        var key = String(productId);
        var match = (d.systemPlanMatch || {})[key] || {};
        var draft = (d.systemPlanDraft || {})[key] || {};
        return {
          productId: Number(productId),
          selectedPlanIds: (match.selectedPlanIds || []).map(Number),
          syncPlan: !!draft.syncCreate,
          planTitle: draft.planName || "",
          planBegin: draft.planBegin || d.start || "",
        };
      }),
    };
  }

  function saveScheduleVersionWindowModal() {
    if (scheduleVersionWindowModalMode === "create") {
      var payload = collectScheduleCreateSavePayload();
      if (typeof window.showToast === "function") {
        window.showToast("保存功能待实现");
      }
      if (typeof console !== "undefined" && console.log) {
        console.log("[schedule] create window payload", payload);
      }
      return payload;
    }
    return null;
  }

  function updateScheduleCreateDraftField(field, value) {
    var d = getScheduleVersionCreateDraft();
    d[field] = value;
    var shouldRender = false;
    if (field === "online") {
      shouldRender = true;
      var online = String(value || "").slice(0, 10);
      if (applyScheduleOnlineDerivedFields(d, online)) {
        if ((d.productIds || []).length) {
          refreshMatchingPlansForSelectedProducts();
        }
      } else {
        clearScheduleOnlineDerivedFields(d);
      }
    }
    if (field === "start") {
      d._startManual = true;
      d._startAutoGenerated = false;
      syncScheduleDraftPlanBeginFromWindow(d);
      shouldRender = true;
    }
    if (field === "name") {
      d._nameAutoGenerated = false;
      syncScheduleDraftPlanNamesFromWindow(d);
      shouldRender = true;
    }
    if (["planTestDone", "testDone", "acceptDone"].indexOf(field) >= 0) {
      d.useOrgTemplate = false;
      shouldRender = true;
    }
    if (shouldRender) {
      renderScheduleVersionWindowModalBody();
    }
  }

  function toggleScheduleCreateUseOrgTemplate(checked) {
    var d = getScheduleVersionCreateDraft();
    d.useOrgTemplate = !!checked;
    if (d.useOrgTemplate) {
      applyScheduleCreateOrgTemplateDates();
    }
    renderScheduleVersionWindowModalBody();
  }

  function toggleScheduleCreateSystem(productId, productName, checked) {
    var d = getScheduleVersionModalDraft();
    if (!d) {
      return;
    }
    var id = Number(productId);
    var set = new Set((d.productIds || []).map(Number));
    if (checked) {
      set.add(id);
      d.productIds = Array.from(set);
      if (hasValidScheduleOnlineDate(d)) {
        fetchMatchingPlansForProduct(id, productName);
      } else {
        renderScheduleVersionWindowModalBody();
      }
      return;
    }
    set.delete(id);
    d.productIds = Array.from(set);
    delete (d.systemPlanDraft || {})[String(id)];
    delete (d.systemPlanMatch || {})[String(id)];
    renderScheduleVersionWindowModalBody();
  }

  function toggleScheduleCreateSystemPlanSync(productId, checked) {
    var row = ensureScheduleModalSystemPlanDraft(productId, getProductNameById(productId));
    row.syncCreate = !!checked;
    renderScheduleVersionWindowModalBody();
  }

  function updateScheduleCreateSystemPlanName(productId, value) {
    var row = ensureScheduleModalSystemPlanDraft(productId, getProductNameById(productId));
    row.planName = value;
    row._customName = true;
  }

  window.getScheduleVersionCreateDraft = getScheduleVersionCreateDraft;
  window.collectScheduleCreateSavePayload = collectScheduleCreateSavePayload;
  window.updateScheduleCreateDraftField = updateScheduleCreateDraftField;
  window.toggleScheduleCreateUseOrgTemplate = toggleScheduleCreateUseOrgTemplate;
  window.toggleScheduleCreateSystem = toggleScheduleCreateSystem;
  window.toggleScheduleCreateSystemPlanSync = toggleScheduleCreateSystemPlanSync;
  window.updateScheduleCreateSystemPlanName = updateScheduleCreateSystemPlanName;
  window.openScheduleCreateVersionWindowModal = openScheduleCreateVersionWindowModal;
  window.closeScheduleVersionWindowModal = closeScheduleVersionWindowModal;
  window.saveScheduleVersionWindowModal = saveScheduleVersionWindowModal;

  $root.on("click", ".schedule-scope-chip", function () {
    setScopeChip($(this));
  });

  $root.on("click", ".schedule-data-tab", function () {
    setDataTab($(this));
  });

  $root.on("click", ".schedule-row-expand", function (e) {
    e.stopPropagation();
    var $btn = $(this);
    var $icon = $btn.find("i");
    var parentId = $btn.data("target") || $btn.closest("tr").data("id");
    if (!parentId || !$icon.length) {
      return;
    }
    toggleBizChildren(parentId, $icon);
  });

  $root.on("click", ".js-open-create-version-window", function () {
    openScheduleCreateVersionWindowModal();
  });

  $("#scheduleVersionWindowModalOverlay").on("click", closeScheduleVersionWindowModal);
  $("#scheduleVersionWindowModalCloseBtn, #scheduleVersionWindowModalDismissBtn").on("click", closeScheduleVersionWindowModal);
  $("#scheduleVersionWindowModalSaveBtn").on("click", saveScheduleVersionWindowModal);

  $("#scheduleMoreFiltersBtn").on("click", toggleMoreFilters);
  $("#scheduleClearFilters").on("click", clearFilters);
})(jQuery);
