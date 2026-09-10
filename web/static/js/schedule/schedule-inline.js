(function ($) {
  "use strict";

  function readJSON(url) {
    return window.scheduleFetch(url, { headers: { Accept: "application/json" } }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok || !data.success) {
          throw new Error(data.error || "加载失败，请重试");
        }
        return data;
      });
    });
  }

  function showError(error) {
    if (error.message !== "session expired") window.showToast(error.message, "error");
  }

  $(document).on("click", ".js-home-schedule-action", function () {
    var demandID = Number($(this).attr("data-demand-id"));
    var storyID = Number($(this).attr("data-story-id"));
    window.openScheduleIntegratedModal({
      id: storyID ? String(storyID) : "US" + demandID,
      demandId: demandID,
      storyId: storyID,
      isIndependent: storyID > 0
    });
  });

  function fillOptions(data) {
    var $groups = $("#scheduleWindowTeamgroup").empty();
    data.teamgroups.forEach(function (group) {
      $("<option>").val(group.id).text(group.name).appendTo($groups);
    });
    var $products = $("#scheduleWindowSystemsGrid").empty();
    data.products.forEach(function (product) {
      var $label = $("<label>").addClass("chk").attr({ "data-product-id": product.id, "data-product-name": product.name });
      $("<input>").attr({ type: "checkbox", name: "productIds", value: product.id }).appendTo($label);
      $("<span>").text(product.name).appendTo($label);
      $label.appendTo($products);
    });
    if (!data.products.length) $("<div>").addClass("schedule-create-empty").text("暂无关联系统").appendTo($products);
  }

  function refreshWindows(url, previousIDs, payload, resData) {
    return readJSON(url).then(function (data) {
      var $select = $("#scheduleIntegratedWindowSelect");
      var candidates = [];
      var createdId = resData && resData.windowId ? String(resData.windowId) : "";
      (data.windows || []).forEach(function (item) {
        var strId = String(item.id);
        if (!previousIDs.has(strId)) {
          $("<option>").val(item.id).text(item.name).attr("data-release-date", item.releaseDate).appendTo($select);
          if (createdId && strId === createdId) {
            candidates.push(item.id);
          } else if (item.name === payload.name && item.releaseDate === payload.releaseDate) {
            candidates.push(item.id);
          }
        }
      });
      if (candidates.length > 0) {
        $select.val(candidates[0]).trigger("change");
      } else if (createdId) {
        if (!$select.find("option[value='" + createdId + "']").length) {
          $("<option>")
            .val(createdId)
            .text(payload.name)
            .attr("data-release-date", payload.releaseDate)
            .appendTo($select);
        }
        $select.val(createdId).trigger("change");
      }
    }).catch(function () {
      if (resData && resData.windowId) {
        var $select = $("#scheduleIntegratedWindowSelect");
        var createdId = String(resData.windowId);
        if (!$select.find("option[value='" + createdId + "']").length) {
          $("<option>")
            .val(createdId)
            .text(payload.name)
            .attr("data-release-date", payload.releaseDate)
            .appendTo($select);
        }
        $select.val(createdId).trigger("change");
      } else {
        window.showToast("窗口已保存，但窗口列表刷新失败；请重新打开排期后选择", "error");
      }
    });
  }

  $(document).on("click", ".js-schedule-inline-create", function () {
    var shared = window.ScheduleIntegratedShared;
    if (!shared.isSchedulingDetailLoaded || !shared.currentCanEditWindow) {
      window.showToast("当前需求暂不能变更版本窗口", "error");
      return;
    }
    var $button = $(this).prop("disabled", true);
    var url = shared.currentStoryId
      ? "/schedule/stories/" + shared.currentStoryId + "/scheduling"
      : "/schedule/demands/" + shared.currentDemandId + "/scheduling";
    var previousIDs = new Set($("#scheduleIntegratedWindowSelect option").map(function () { return this.value; }).get());
    readJSON("/schedule/window-options").then(function (data) {
      fillOptions(data);
      if (!data.teamgroups.length) throw new Error("暂无可用敏捷小组，无法创建窗口");
      window.openScheduleCreateVersionWindowModal(function (payload, resData) {
        return refreshWindows(url, previousIDs, payload, resData);
      });
    }).catch(showError).finally(function () { $button.prop("disabled", false); });
  });
})(jQuery);
