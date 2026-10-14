/* ==========================================================================
   文件: web/static/js/schedule/scheduleintegratediteration.js
   模块: 排期工作台
   职责: 一体化排期「快速创建迭代」字段初始化、默认值与产品联动（不含创建提交）
   ========================================================================== */
(function ($) {
  "use strict";

  var shared = window.ScheduleIntegratedShared;
  var DATE_PLACEHOLDER = "年-月-日";

  function parsePositiveInt(value) {
    var num = parseInt(String(value == null ? "" : value), 10);
    return isNaN(num) || num <= 0 ? 0 : num;
  }

  function shanghaiToday() {
    return new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Shanghai" }).format(new Date());
  }

  function addDays(dateStr, days) {
    var d = new Date(dateStr + "T00:00:00+08:00");
    d.setUTCDate(d.getUTCDate() + days);
    return new Intl.DateTimeFormat("sv-SE", { timeZone: "Asia/Shanghai" }).format(d);
  }

  function normalizeYMD(value) {
    var raw = String(value || "").trim();
    if (!raw) {
      return "";
    }
    var m = raw.match(/^(\d{4})-(\d{1,2})-(\d{1,2})/);
    if (!m) {
      return "";
    }
    return m[1] + "-" + ("0" + m[2]).slice(-2) + "-" + ("0" + m[3]).slice(-2);
  }

  function syncDateDisplay(input) {
    if (!input) {
      return;
    }
    if (shared && typeof shared.syncIntegratedDateInputState === "function") {
      shared.syncIntegratedDateInputState(input);
      return;
    }
    var wrap = input.closest(".task-iteration-date");
    if (!wrap) {
      return;
    }
    var v = normalizeYMD(input.value);
    wrap.setAttribute("data-date", v || DATE_PLACEHOLDER);
    wrap.classList.toggle("is-empty", !v);
  }

  function setDateValue(input, ymd) {
    if (!input) {
      return;
    }
    input.value = normalizeYMD(ymd);
    syncDateDisplay(input);
  }

  function findForm($node) {
    return $node.find(".rd-quick-iteration").first();
  }

  function teamgroupItems() {
    var items = [];
    var seen = {};
    document.querySelectorAll("#scheduleIntegratedTeamgroupOptions [data-value]").forEach(function (el) {
      var value = String(el.getAttribute("data-value") || "").trim();
      if (!value || seen[value]) {
        return;
      }
      seen[value] = true;
      items.push({
        value: value,
        label: String(el.getAttribute("data-label") || value).trim() || value,
        manager: String(el.getAttribute("data-manager") || "").trim(),
      });
    });
    return items;
  }

  function findItemLabel(items, value) {
    value = String(value || "").trim();
    if (!value) {
      return "";
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].value) === value) {
        return items[i].label || value;
      }
    }
    return "";
  }

  function findItemByValue(items, value) {
    value = String(value || "").trim();
    if (!value) {
      return null;
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].value) === value) {
        return items[i];
      }
    }
    return null;
  }

  function findItemValueByLabel(items, label) {
    label = String(label || "").trim();
    if (!label || label === "—") {
      return "";
    }
    for (var i = 0; i < (items || []).length; i++) {
      if (String(items[i].label || "").trim() === label) {
        return String(items[i].value);
      }
    }
    return "";
  }

  function firstManagerAccount(manager) {
    var raw = String(manager || "").trim();
    if (!raw) {
      return "";
    }
    return raw.split(",")[0].trim();
  }

  function resolveDefaultTeam() {
    var items = teamgroupItems();
    var teamId = String((shared && shared.currentTeamgroupId) || "").trim();
    if (teamId === "0") {
      teamId = "";
    }
    var teamName = String((shared && shared.currentTeamgroupName) || "").trim();
    if (!teamId) {
      teamId = findItemValueByLabel(items, teamName);
    }
    if (!teamName) {
      teamName = findItemLabel(items, teamId);
    }
    var item = findItemByValue(items, teamId);
    return {
      items: items,
      teamId: teamId,
      teamName: teamName,
      manager: item ? firstManagerAccount(item.manager) : "",
    };
  }

  function userItems() {
    return (shared && typeof shared.toAutocompleteItems === "function"
      ? shared.toAutocompleteItems(shared.schedulingUsers)
      : []);
  }

  function planItems(plans) {
    return (plans || [])
      .map(function (p) {
        var id = String((p && p.id) || "").trim();
        if (!id || id === "0") {
          return null;
        }
        var title = String((p && p.title) || "").trim();
        return { value: id, label: title || id };
      })
      .filter(Boolean);
  }

  function ensureFieldIds($form) {
    if ($form.attr("data-qi-ids") === "1") {
      return {
        ownerInput: $form.find(".rd-quick-iteration-owner-input").attr("id"),
        ownerHidden: $form.find(".rd-quick-iteration-owner").attr("id"),
        teamInput: $form.find(".rd-quick-iteration-team-input").attr("id"),
        teamHidden: $form.find(".rd-quick-iteration-team").attr("id"),
        planInput: $form.find(".rd-quick-iteration-plan-input").attr("id"),
        planHidden: $form.find(".rd-quick-iteration-plan").attr("id"),
        periodName: $form.find(".rd-quick-iteration-period").first().attr("name"),
      };
    }
    if (shared) {
      shared.quickIterationSeq = (shared.quickIterationSeq || 0) + 1;
    }
    var seq = (shared && shared.quickIterationSeq) || Date.now();
    var ids = {
      ownerInput: "rdQiOwnerInput" + seq,
      ownerHidden: "rdQiOwner" + seq,
      teamInput: "rdQiTeamInput" + seq,
      teamHidden: "rdQiTeam" + seq,
      planInput: "rdQiPlanInput" + seq,
      planHidden: "rdQiPlan" + seq,
      periodName: "rdQiPeriod" + seq,
    };
    $form.find(".rd-quick-iteration-owner-input").attr("id", ids.ownerInput);
    $form.find(".rd-quick-iteration-owner").attr("id", ids.ownerHidden);
    $form.find(".rd-quick-iteration-team-input").attr("id", ids.teamInput);
    $form.find(".rd-quick-iteration-team").attr("id", ids.teamHidden);
    $form.find(".rd-quick-iteration-plan-input").attr("id", ids.planInput);
    $form.find(".rd-quick-iteration-plan").attr("id", ids.planHidden);
    $form.find(".rd-quick-iteration-period").attr("name", ids.periodName);
    $form.attr("data-qi-ids", "1");
    return ids;
  }

  function destroyPickers(ids) {
    if (typeof window.destroyAutocomplete !== "function" || !ids) {
      return;
    }
    [ids.ownerInput, ids.teamInput, ids.planInput].forEach(function (id) {
      if (id) {
        window.destroyAutocomplete(id);
      }
    });
  }

  function readPeriod($form) {
    var $checked = $form.find(".rd-quick-iteration-period:checked").first();
    return ($checked.val() || "2w") === "4w" ? "4w" : "2w";
  }

  function applyPeriodDates($form) {
    var period = readPeriod($form);
    var begin = shanghaiToday();
    var end = addDays(begin, period === "4w" ? 27 : 13);
    setDateValue($form.find(".rd-quick-iteration-begin")[0], begin);
    setDateValue($form.find(".rd-quick-iteration-end")[0], end);
  }

  function initOwnerAutocomplete(ids, ownerAccount) {
    if (typeof window.initAutocomplete !== "function" || !ids) {
      return;
    }
    var users = userItems();
    var owner = String(ownerAccount || "").trim();
    window.initAutocomplete(ids.ownerInput, ids.ownerHidden, users, {
      labelOnly: true,
      value: owner,
      label: findItemLabel(users, owner),
    });
  }

  function initTeamAutocomplete(ids, team) {
    if (typeof window.initAutocomplete !== "function" || !ids) {
      return;
    }
    team = team || {};
    window.initAutocomplete(ids.teamInput, ids.teamHidden, team.items || teamgroupItems(), {
      labelOnly: true,
      placeholder: "请选择",
      value: team.teamId || "",
      label: team.teamName || "",
    });
  }

  function initPlanAutocomplete(ids, plans) {
    if (typeof window.initAutocomplete !== "function" || !ids) {
      return;
    }
    window.initAutocomplete(ids.planInput, ids.planHidden, planItems(plans), {
      labelOnly: true,
      value: "",
      placeholder: "请选择产品计划",
    });
  }

  function clearPlanAutocomplete(ids) {
    initPlanAutocomplete(ids, []);
  }

  function loadProductPlans(productId, done) {
    var pid = parsePositiveInt(productId);
    if (!pid) {
      done([]);
      return;
    }
    $.ajax({
      url: "/schedule/products/" + encodeURIComponent(pid) + "/plans",
      method: "GET",
      dataType: "json",
    })
      .done(function (resp) {
        if (!resp || !resp.success) {
          done([]);
          return;
        }
        done(resp.plans || []);
      })
      .fail(function () {
        done([]);
      });
  }

  function resolveNodeProductId($node) {
    return parsePositiveInt($node.attr("data-product-id") || $node.find(".rd-node-product").val());
  }

  function bindFormEvents($form) {
    if ($form.attr("data-qi-bound") === "1") {
      return;
    }
    $form.attr("data-qi-bound", "1");

    $form.on("change", ".rd-quick-iteration-period", function () {
      if (!this.checked) {
        return;
      }
      applyPeriodDates($form);
    });

    $form.on("change input", ".rd-quick-iteration-begin, .rd-quick-iteration-end", function () {
      syncDateDisplay(this);
    });
  }

  function initForNode($node) {
    if (!$node || !$node.length) {
      return;
    }
    var $form = findForm($node);
    if (!$form.length) {
      return;
    }

    var ids = ensureFieldIds($form);
    destroyPickers(ids);
    bindFormEvents($form);

    $form.find(".rd-quick-iteration-name").val("");
    $form.find(".rd-quick-iteration-period").each(function () {
      this.checked = this.value === "2w";
    });
    applyPeriodDates($form);

    var team = resolveDefaultTeam();
    initOwnerAutocomplete(ids, team.manager);
    initTeamAutocomplete(ids, team);

    var productId = resolveNodeProductId($node);
    loadProductPlans(productId, function (plans) {
      initPlanAutocomplete(ids, plans);
    });
  }

  function onProductChange($node, productId) {
    if (!$node || !$node.length) {
      return;
    }
    // 仅未保存（草稿）研需：改所属产品时清空并重载关联计划
    if (!$node.hasClass("rd-node--draft")) {
      return;
    }
    var $form = findForm($node);
    if (!$form.length) {
      return;
    }
    var ids = ensureFieldIds($form);
    clearPlanAutocomplete(ids);
    loadProductPlans(productId, function (plans) {
      initPlanAutocomplete(ids, plans);
    });
  }

  window.ScheduleIntegratedIteration = {
    initForNode: initForNode,
    onProductChange: onProductChange,
    ensureFieldIds: ensureFieldIds,
    normalizeYMD: normalizeYMD,
    resolveNodeProductId: resolveNodeProductId,
  };
})(jQuery);
