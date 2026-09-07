/* =============================================================================
   文件: docs/PRD/Demo/js/po/primary-action.js
   模块: PO 个人工作台 - 主操作动作渲染
   职责: 按照业务动作矩阵 (9大阶段)，为每行需求派生具体可办理的交互主动作。
         在静态原型环境下，支持打开对应的弹窗/Drawer/提测/排期页面。
   ============================================================================= */

(function () {
  'use strict';

  var esc = (window.PersonalList && window.PersonalList.escapeHtml) || function (v) { 
    return String(v == null ? '' : v).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  };

  function renderEnabled(item, pa, isStory) {
    var label = String(pa.label || '').trim();
    if (!label) { return '<span class="home-unavailable">—</span>'; }
    var key = String(pa.key || '').toLowerCase();
    var kind = String(pa.kind || '').toLowerCase();
    var did = String(pa.demandId || item.id || '').replace(/^US/i, '');

    // 1. 排期
    if (key === 'schedule' || kind === 'schedule') {
      return '<a class="table-action-btn primary" href="schedule.html">' + esc(label) + '</a>';
    }

    // 2. 提测 -> submit-test.html
    if (key === 'submit_test' || kind === 'page') {
      return '<a class="table-action-btn primary" href="submit-test.html">' + esc(label) + '</a>';
    }

    // 3. 联调测试 -> 测试单 ↗
    if (key === 'test_order' || kind === 'external') {
      return '<a class="table-action-btn primary" href="#test-order" onclick="alert('打开联调测试单详情：' + esc(did) + '');return false">' + esc(label) + ' ↗</a>';
    }

    // 4. 受理 (审批)
    if (key === 'approve') {
      return '<button type="button" class="table-action-btn primary" onclick="PrimaryAction.handleApprove('' + esc(did) + '')">' + esc(label) + '</button>';
    }

    // 5. 澄清
    if (key === 'clarify') {
      return '<button type="button" class="table-action-btn primary" onclick="PrimaryAction.handleClarify('' + esc(did) + '')">' + esc(label) + '</button>';
    }

    // 6. 验收 / 催验收
    if (key === 'accept') {
      return '<button type="button" class="table-action-btn primary" onclick="PrimaryAction.handleAccept('' + esc(did) + '')">' + esc(label) + '</button>';
    }
    if (key === 'urge_accept') {
      return '<button type="button" class="table-action-btn" onclick="PrimaryAction.handleUrge('' + esc(did) + '')">' + esc(label) + '</button>';
    }

    // 7. 发起交付
    if (key === 'deliver') {
      return '<button type="button" class="table-action-btn primary" onclick="PrimaryAction.handleDeliver('' + esc(did) + '')">' + esc(label) + '</button>';
    }

    // 8. 评价 / 查看评价
    if (key === 'feedback') {
      return '<button type="button" class="table-action-btn primary" onclick="PrimaryAction.handleFeedback('' + esc(did) + '')">' + esc(label) + '</button>';
    }
    if (key === 'view_feedback') {
      return '<button type="button" class="table-action-btn" onclick="PrimaryAction.handleViewFeedback('' + esc(did) + '')">' + esc(label) + '</button>';
    }

    // 抽屉或内部事件
    if (kind === 'drawer' || kind === 'internal') {
      return '<button type="button" class="table-action-btn primary" data-demand-id="' + esc(did) + '">' + esc(label) + '</button>';
    }

    return '<button type="button" class="table-action-btn primary" data-demand-id="' + esc(did) + '">' + esc(label) + '</button>';
  }

  function primaryActionHtml(item, isStory) {
    var pa = item && item.primaryAction;
    if (pa && typeof pa === 'object') {
      if (pa.enabled === false) {
        var reason = String(pa.reason || '无办理权限').trim();
        var label = String(pa.label || '').trim() || '—';
        return '<span class="home-unavailable" title="' + esc(reason) + '">' + esc(label) + '</span>';
      }
      return renderEnabled(item, pa, isStory);
    }
    return '<span class="home-unavailable">—</span>';
  }

  // 动作交互办理弹窗/Drawer
  function handleApprove(did) {
    if (window.DemandDetail && typeof window.DemandDetail.open === 'function') {
      window.DemandDetail.open(did);
    } else {
      alert('正在办理需求审批 (ID: ' + did + ')');
    }
  }

  function handleClarify(did) {
    if (window.DemandDetail && typeof window.DemandDetail.open === 'function') {
      window.DemandDetail.open(did);
      setTimeout(function() {
        if (typeof window.DemandDetail.switchTab === 'function') {
          window.DemandDetail.switchTab('requirement', 'clarificationSection');
        }
      }, 100);
    } else {
      alert('打开需求澄清 (ID: ' + did + ')');
    }
  }

  function handleAccept(did) {
    alert('【业务验收办理】
需求 ID: US' + did + '
当前状态: 联调测试已通过
点击确认完成验收签章');
  }

  function handleUrge(did) {
    alert('【催验收通知】
已通过工作台消息向责任人发送催办提醒！');
  }

  function handleDeliver(did) {
    alert('【发起交付办理】
需求 ID: US' + did + '
系统已核验：测试报告已通过、分支已合并。
已提交发布投产窗口审批！');
  }

  function handleFeedback(did) {
    alert('【需求评价反馈】
需求已在生产稳定运行，请输入 1~5 星满意度评价并提交');
  }

  function handleViewFeedback(did) {
    alert('【查看历史评价】
综合评价：5星满分
业务反馈：上线平稳，大幅提高业务处理效率！');
  }

  window.PrimaryAction = {
    primaryActionHtml: primaryActionHtml,
    handleApprove: handleApprove,
    handleClarify: handleClarify,
    handleAccept: handleAccept,
    handleUrge: handleUrge,
    handleDeliver: handleDeliver,
    handleFeedback: handleFeedback,
    handleViewFeedback: handleViewFeedback
  };
})();
