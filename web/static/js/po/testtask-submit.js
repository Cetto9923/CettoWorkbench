(function ($) {
  "use strict";
  var running = false, blocked = false, linked = {}, versions = {}, outcomes = {};
  function root() { return window.PoTesttaskCore.$root(); }
  function value(key, unit) { return String(root().find('[data-tt-' + key + '="' + unit + '"]').val() || "").trim(); }
  function result(message) { root().find("#poTesttaskResult").text(message); }
  function enabledUnits() {
    var rows = [];
    root().find("[data-tt-unit]").each(function () {
      var unit = $(this).attr("data-tt-unit");
      if ($(this).attr("data-is-main") === "1" || root().find('[name="sys' + unit + 'Enable"]').is(":checked")) {
        rows.push({ unit: unit, product: Number($(this).attr("data-tt-unit-product")) });
      }
    });
    if (!rows.length || rows.some(function (r) { return !r.product; })) throw new Error("请等待系统信息加载完成，并选择有效系统");
    return rows;
  }
  function basePath() { return "/demands/" + encodeURIComponent(window.PoTesttaskCore.getDemandId()) + "/testtask/"; }
  async function post(path, body) {
    var response;
    try {
      response = await window.appFetch(path, { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(body) });
    } catch (err) {
      blocked = true;
      throw new Error("请求结果未知，请先到禅道核对，勿重复提交");
    }
    var data;
    try { data = await response.json(); } catch (err) {
      blocked = true;
      throw new Error("服务响应无法确认，请先到禅道核对，勿重复提交");
    }
    if (!response.ok || !data.success) {
      // 除明确的表单校验 422 外，失败响应也可能已有远端写入。
      if (response.status !== 422) blocked = true;
      var succeeded = data.succeeded || [];
      var detail = succeeded.length ? "；已创建：" + succeeded.map(function (t) { return "#" + t.testtaskId; }).join("、") : "";
      throw new Error((data.message || "提交失败") + detail + (data.errors ? "；" + data.errors.map(function (e) { return e.message; }).join("；") : ""));
    }
    return data.data;
  }
  async function run(action) {
    if (running || blocked) return false;
    running = true;
    var $controls = root().find("input:enabled, select:enabled, textarea:enabled, button:enabled");
    $controls.prop("disabled", true);
    try { await action(); return true; }
    catch (err) { result(err.message || "操作失败"); window.PoTesttaskCore.showToast(err.message || "操作失败", "error"); return false; }
    finally { if (blocked) outcomes[basePath()] = root().find("#poTesttaskResult").text(); running = false; $controls.prop("disabled", false); if (blocked) root().find("#poTesttaskSubmitBtn, #poTesttaskNextBtn").prop("disabled", true); }
  }
  async function prepareVersions() {
    var rows = enabledUnits();
    // 全部字段先校验，再开始创建，避免前几项已落库才发现末项漏填。
    rows.forEach(function (row) {
      var mode = root().find('[name="ver' + row.unit + 'Mode"]:checked').val();
      row.ids = (root().find('[data-tt-exist-ver="' + row.unit + '"]').val() || []);
      if (!Array.isArray(row.ids)) row.ids = row.ids ? [row.ids] : [];
      row.ids = row.ids.map(Number).filter(Boolean);
      if (mode === "exist") { if (!row.ids.length) throw new Error("请选择系统已有版本"); return; }
      var execution = value("exec", row.unit).split("-");
      row.build = { productId: row.product, projectId: Number(execution[0]), executionId: Number(execution[1]), name: value("ver-name", row.unit), date: value("ver-date", row.unit), desc: value("ver-desc", row.unit) };
      if (!row.build.projectId || !row.build.executionId || !row.build.name || !row.build.date) throw new Error("请补齐版本名称、所属执行与计划上线日期");
    });
    for (var row of rows) {
      if (row.build) {
        var data = await post(basePath() + "builds", { builds: [row.build] });
        var build = data && data.builds && data.builds[0];
        if (!build || !build.buildId) { blocked = true; throw new Error("未收到版本 ID，请先到禅道核对"); }
        row.ids = [build.buildId];
        var $select = root().find('[data-tt-exist-ver="' + row.unit + '"]');
        $select.append($("<option></option>").val(build.buildId).text(build.name)).val(String(build.buildId)).prop("disabled", false);
        root().find('[name="ver' + row.unit + 'Mode"][value="exist"]').prop("checked", true).trigger("change");
        result("版本 #" + build.buildId + " 已创建，可继续关联需求");
      }
      versions[row.unit] = row.ids;
      await showLinked(row);
    }
  }
  async function showLinked(row) {
    var $box = root().find('[data-tt-link-unit="' + row.unit + '"]');
    $box.find('[data-tt-existing-stories]').remove();
    for (var id of row.ids) {
      var response = await window.appFetch('/builds/' + id + '/linkedstories', { headers: { Accept: 'application/json' } });
      var body = await response.json();
      if (!response.ok || !body.success) throw new Error(body.message || '已有研发需求加载失败，请重试');
      var list = Array.isArray(body.data) ? body.data : [];
      var text = list.length ? list.map(function (s) { return '#' + s.id + ' ' + (s.title || ''); }).join('、') : '暂无关联需求';
      $box.append($('<div data-tt-existing-stories></div>').text('版本 #' + id + ' 已关联：' + text));
    }
  }
  async function linkStories() {
    for (var row of enabledUnits()) {
      var ids = versions[row.unit];
      if (!ids || !ids.length) throw new Error("请返回第二步确认版本");
      var stories = root().find('[data-tt-link-unit="' + row.unit + '"] input[type="checkbox"]:checked').map(function () { return $(this).val(); }).get().join(",");
      if (!stories) continue;
      for (var id of ids) {
        var key = id + ":" + stories;
        if (linked[key]) continue;
        await post("/builds/" + id + "/linkstories", { stories: stories });
        linked[key] = true;
      }
    }
  }
  function taskFields(unit) {
    var item = { name: value("task-name", unit), owner: value("owner-account", unit), begin: value("task-begin", unit), end: value("task-end", unit), type: value("task-type", unit), pri: Number(value("task-pri", unit)), desc: value("task-desc", unit) };
    if (!item.name || !item.owner || !item.begin || !item.end || !item.type || !item.pri) throw new Error("请补齐测试单名称、负责人、日期、类型和优先级");
    if (item.end < item.begin) throw new Error("结束日期不能早于开始日期");
    return item;
  }
  async function submit() {
    var rows = enabledUnits(), joint = Number(root().find('[name="isJointTest"]:checked').val()) === 1;
    var payload = joint ? Object.assign(taskFields("joint"), { joint: 1, products: [], builds: [], members: root().find("#poTesttaskMembers").val() || [] }) : { joint: 0, tasks: [] };
    rows.forEach(function (row) {
      var ids = versions[row.unit];
      if (!ids || !ids.length) throw new Error("请返回第二步确认版本");
      if (joint) { payload.products.push(row.product); payload.builds.push(ids); }
      else { if (ids.length !== 1) throw new Error("独立测试单每个系统只能选择一个版本"); payload.tasks.push(Object.assign(taskFields(row.unit), { productId: row.product, buildId: ids[0] })); }
    });
    var data = await post(basePath() + "tasks", payload);
    if (!data || !data.tasks || !data.tasks.length) { blocked = true; throw new Error("未收到测试单 ID，请先到禅道核对"); }
    blocked = true;
    result("提测成功，已创建禅道测试单：" + data.tasks.map(function (t) { return "#" + t.testtaskId; }).join("、"));
  }
  window.PoTesttaskSubmit = {
    beforeNext: function (step) { return run(function () { if (step === 2) return prepareVersions(); if (step === 3) return linkStories(); enabledUnits(); }); },
    busy: function () { return running; },
    reset: function () { blocked = !!outcomes[basePath()]; linked = {}; versions = {}; result(outcomes[basePath()] || ""); root().find("#poTesttaskSubmitBtn, #poTesttaskNextBtn").prop("disabled", blocked); },
    submit: function () { return run(submit); }
  };
  $(document).on("click", "#poTesttaskSubmitBtn", function () { window.PoTesttaskSubmit.submit(); });
  $(document).on("change", '[name="isJointTest"]', function () {
    var joint = Number($(this).val()) === 1;
    root().find("[data-tt-exist-ver]").prop("multiple", joint);
  });
})(jQuery);
