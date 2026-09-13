(function ($) {
  "use strict";
  var esc = window.escapeHtml;

  function renderDynamicForm(data) {
    var $r = window.PoTesttaskCore.$root();
    var isJointTest = Number($r.find('input[name="isJointTest"]:checked').val()) || 0;
    var demandId = data.demandId != null ? String(data.demandId) : "";
    var launchDate = $.trim(data.estimateLaunch || "");
    if (launchDate.indexOf("0001") === 0 || launchDate.indexOf("0000") === 0 || launchDate === "—") {
      launchDate = "";
    }
    var systems = Array.isArray(data.systems) ? data.systems : [];
    if (!systems.length) {
      if (data.mainSystemName && data.mainSystemName !== "—") {
        systems = [{ id: 0, name: data.mainSystemName, isMain: true, stories: [] }];
      }
    }

    $r.find("#poTesttaskSysMeta").text("共 " + systems.length + " 个系统");

    var todayStr = new Date().toISOString().slice(0, 10).replace(/-/g, "");
    var qdName = $.trim(data.qdName || data.qd || "");
    if (qdName === "—") qdName = "";

    // 1. 动态生成步骤 2：按系统配置版本基础信息
    var unitsHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var badgeHtml = isMain ? '<span class="po-testtask-sys-badge is-main">主</span>' : '<span class="po-testtask-sys-badge is-sub">配</span>';
      var enabled = isMain || isJointTest === 1;
      var toggleHtml = isMain
        ? '<label class="po-testtask-sys-toggle"><input type="checkbox" name="sys' + unitNo + 'Enable" value="1" checked disabled /> <span>主系统（默认提测）</span></label>'
        : '<label class="po-testtask-sys-toggle"><input type="checkbox" name="sys' + unitNo + 'Enable" value="1" ' + (enabled ? 'checked' : '') + ' /> <span>参与本次提测</span></label>';

      var defVerName = todayStr + "-US" + demandId + "-" + sysName + "-提测版本";

      unitsHtml +=
        '<div class="po-testtask-unit" data-tt-unit="' + unitNo + '" data-tt-unit-product="' + (sys.id || "") + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-head">' +
        '    <div class="po-testtask-unit-name"><i class="fas fa-folder-open"></i> <span>' + sysName + '</span> ' + badgeHtml + '</div>' +
        '    ' + toggleHtml +
        '  </div>' +
        '  <div class="po-testtask-unit-skipped ' + (enabled ? 'po-testtask-hidden' : '') + '" data-tt-unit-skipped="' + unitNo + '">' +
        '    <i class="fas fa-circle-info"></i> 已跳过该系统提测，不创建版本与测试单，不显示该系统需求' +
        '  </div>' +
        '  <div class="po-testtask-unit-body ' + (enabled ? '' : 'po-testtask-hidden') + '">' +
        '    <div class="po-testtask-field-label"><span class="req">*</span> 版本模式</div>' +
        '    <div class="po-testtask-radio-row">' +
        '      <label><input type="radio" name="ver' + unitNo + 'Mode" value="new" checked /> 创建新版本</label>' +
        '      <label><input type="radio" name="ver' + unitNo + 'Mode" value="exist" /> 使用已有版本</label>' +
        '    </div>' +
        (isMain ? '' : '    <div class="po-testtask-check-row" data-tt-integrate-wrap="' + unitNo + '"><label><input type="checkbox" name="ver' + unitNo + 'IsIntegrate" value="1" /> 集成版本</label></div>') +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 所属执行</div>' +
        '      <select data-tt-exec="' + unitNo + '"><option value="">加载中…</option></select>' +
        '    </div>' +
        '    <div data-tt-new-base="' + unitNo + '">' +
        '      <div class="po-testtask-grid-2">' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 新版本名称</div>' +
        '          <input type="text" data-tt-ver-name="' + unitNo + '" value="' + esc(defVerName) + '" />' +
        '        </div>' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 计划上线日期</div>' +
        '          <input type="date" data-tt-ver-date="' + unitNo + '" value="' + esc(launchDate) + '" />' +
        '        </div>' +
        '      </div>' +
        '      <div class="po-testtask-field">' +
        '        <div class="po-testtask-field-label">版本说明 (可选)</div>' +
        '        <input type="text" data-tt-ver-desc="' + unitNo + '" placeholder="说明此版本主要内容（可选）" />' +
        '      </div>' +
        '    </div>' +
        '    <div class="po-testtask-hidden" data-tt-exist-base="' + unitNo + '">' +
        '      <div class="po-testtask-grid-2">' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label"><span class="req">*</span> 已有版本名称</div>' +
        '          <select data-tt-exist-ver="' + unitNo + '"><option value="">请先选择所属执行，再选取版本</option></select>' +
        '        </div>' +
        '        <div class="po-testtask-field">' +
        '          <div class="po-testtask-field-label">计划上线日期</div>' +
        '          <input type="date" class="is-readonly" data-tt-exist-date="' + unitNo + '" readonly />' +
        '        </div>' +
        '      </div>' +
        '      <div class="po-testtask-field">' +
        '        <div class="po-testtask-field-label">版本说明 (可选)</div>' +
        '        <input type="text" class="is-readonly" data-tt-exist-desc="' + unitNo + '" readonly />' +
        '      </div>' +
        '    </div>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskUnitsWrap").html(unitsHtml);

    // 2. 动态生成步骤 3：各系统版本关联需求配置
    var linksHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var badgeHtml = isMain ? '<span class="po-testtask-sys-badge is-main">主</span>' : '<span class="po-testtask-sys-badge is-sub">配</span>';
      var stories = Array.isArray(sys.stories) ? sys.stories : [];

      var storiesListHtml = "";
      if (stories.length > 0) {
        stories.forEach(function (st) {
          storiesListHtml += '<label><input type="checkbox" checked value="' + esc(st.id) + '" /> #' + esc(st.id) + ' ' + esc(st.title) + '</label>';
        });
      } else {
        storiesListHtml = '<div class="po-testtask-empty-hint" style="color:var(--t3);font-size:12px;padding:6px 0">该系统下暂未关联转出的研发需求</div>';
      }

      var enabled = isMain || isJointTest === 1;
      var shouldHide = !enabled;

      linksHtml +=
        '<div class="po-testtask-unit ' + (shouldHide ? 'po-testtask-hidden' : '') + '" data-tt-link-unit="' + unitNo + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-name" style="margin-bottom:8px">' +
        '    <i class="fas fa-folder-open"></i> <span>' + sysName + '</span> ' + badgeHtml +
        '  </div>' +
        '  <div class="po-testtask-unit-body">' +
        '    <div data-tt-link-edit="' + unitNo + '">' +
        '      <div class="po-testtask-link-actions">' +
        '        <div class="po-testtask-field-label">勾选研发需求以追加关联到版本（不会移除已有关联）</div>' +
        '      </div>' +
        '      <div class="po-testtask-link-list">' + storiesListHtml + '</div>' +
        '    </div>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskLinksWrap").html(linksHtml);

    // 3. 动态生成步骤 4：各系统独立测试单
    var tasksHtml = "";
    systems.forEach(function (sys, idx) {
      var unitNo = idx + 1;
      var isMain = !!sys.isMain;
      var sysName = esc(sys.name || ("系统" + unitNo));
      var defTaskName = todayStr + "-US" + demandId + "-" + sysName + "-测试单";
      var enabled = isMain || isJointTest === 1;
      var shouldHide = !enabled;

      tasksHtml +=
        '<div class="po-testtask-unit ' + (shouldHide ? 'po-testtask-hidden' : '') + '" data-tt-task-unit="' + unitNo + '" data-is-main="' + (isMain ? "1" : "0") + '">' +
        '  <div class="po-testtask-unit-title">' + sysName + ' - 测试单</div>' +
        '  <div class="po-testtask-grid-4">' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 测试单名称</div>' +
        '      <input type="text" data-tt-task-name="' + unitNo + '" value="' + esc(defTaskName) + '" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label"><span class="req">*</span> 测试负责人</div>' +
        '      <input type="text" data-tt-task-owner="' + unitNo + '" value="' + esc(qdName) + '" placeholder="输入姓名或工号搜索" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label">开始日期</div>' +
        '      <input type="date" data-tt-task-begin="' + unitNo + '" value="' + todayStr.slice(0,4)+"-"+todayStr.slice(4,6)+"-"+todayStr.slice(6,8) + '" />' +
        '    </div>' +
        '    <div class="po-testtask-field">' +
        '      <div class="po-testtask-field-label">结束日期</div>' +
        '      <input type="date" data-tt-task-end="' + unitNo + '" value="' + esc(launchDate) + '" />' +
        '    </div>' +
        '  </div>' +
        '  <div class="po-testtask-field">' +
        '    <div class="po-testtask-field-label">测试说明 / 重点 (可选)</div>' +
        '    <textarea data-tt-task-desc="' + unitNo + '" placeholder="说明本次提测重点（可选）"></textarea>' +
        '  </div>' +
        '</div>';
    });
    $r.find("#poTesttaskTasksWrap").html(tasksHtml);

    window.PoTesttaskFields.init(data);
  }

  window.PoTesttaskRender = renderDynamicForm;
})(jQuery);
