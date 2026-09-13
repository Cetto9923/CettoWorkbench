(function ($) {
  "use strict";
  function init(data) {
    var $r = window.PoTesttaskCore.$root();
    var now = new Date();
    var today = now.getFullYear() + "-" + String(now.getMonth() + 1).padStart(2, "0") + "-" + String(now.getDate()).padStart(2, "0");
    var users = (data.users || []).map(function (u) {
      return { value: u.account, label: (u.realname || u.account) + "(" + u.account + ")", pinyin: u.pinyin || "" };
    });
    $r.find('[data-tt-task-name="joint"]').val(today.replace(/-/g, "") + "-US" + data.demandId + "-联调总测试单");
    $r.find('[data-tt-task-begin="joint"]').val(today);
    $r.find('[data-tt-task-end="joint"]').val(data.estimateLaunch || "");
    $r.find('[data-tt-task-desc="joint"]').val("");
    $r.find("[data-tt-task-owner]").each(function () {
      var unit = $(this).attr("data-tt-task-owner");
      var inputId = "poTesttaskOwnerInput" + unit, hiddenId = "poTesttaskOwnerValue" + unit;
      $(this).attr("id", inputId);
      $r.find("#" + hiddenId).remove();
      $("<input type='hidden'>").attr({ id: hiddenId, "data-tt-owner-account": unit }).insertAfter(this);
      var label = users.find(function (u) { return u.value === data.qd; });
      window.initUserPicker(inputId, hiddenId, users, { value: data.qd || "", label: label ? label.label : "", placeholder: "输入姓名或工号搜索" });
      var $panel = $(this).closest(".po-testtask-unit");
      $panel.find("[data-tt-task-options]").remove();
      var $options = $("<div class='po-testtask-grid-2' data-tt-task-options></div>");
      var $type = $("<select></select>").attr("data-tt-task-type", unit);
      [["integrate", "SIT测试"], ["system", "UAT测试"], ["check", "验收测试"], ["performance", "性能测试"], ["safety", "安全测试"], ["automat", "自动化测试"]].forEach(function (o) {
        $type.append($("<option></option>").val(o[0]).text(o[1]));
      });
      var $pri = $("<select></select>").attr("data-tt-task-pri", unit);
      [1, 2, 3, 4].forEach(function (v) { $pri.append($("<option></option>").val(v).text(v)); });
      $pri.val("3");
      $options.append($("<div class='po-testtask-field'><div class='po-testtask-field-label'>测试类型</div></div>").append($type), $("<div class='po-testtask-field'><div class='po-testtask-field-label'>优先级</div></div>").append($pri));
      $panel.append($options);
    });
    $r.find("[data-tt-members-wrap]").remove();
    var $members = $("<select multiple aria-label='联调成员'></select>").attr("id", "poTesttaskMembers");
    users.forEach(function (u) { $members.append($("<option></option>").val(u.value).text(u.label)); });
    $r.find("[data-tt-joint-global] .po-testtask-unit").append($("<div class='po-testtask-field' data-tt-members-wrap><div class='po-testtask-field-label'>联调成员（可多选，负责人自动加入）</div></div>").append($members));

  }
  window.PoTesttaskFields = { init: init };
})(jQuery);
