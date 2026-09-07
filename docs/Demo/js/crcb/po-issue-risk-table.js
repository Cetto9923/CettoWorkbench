/* =============================================================================
 * 文件: web/static/workbench/po/po-issue-risk-table.js
 * 模块: PO 工作台 · 问题风险列表表格
 * 职责: 高密度表格渲染、分页与行操作按钮
 * ============================================================================= */

function renderIssueRiskTable(items) {
  var wrap = document.getElementById('issueRiskTableWrap');
  if (!wrap) return;
  var total = (issueRiskLastResp && issueRiskLastResp.total) || 0;
  var ps = Math.max(1, issueRiskFilter.pageSize || 20);
  var page = Math.max(1, issueRiskFilter.page || 1);
  var kind = issueRiskFilter.kind === 'risk' ? 'risk' : 'issue';

  if (!items || !items.length) {
    wrap.innerHTML =
      '<div class="ir-empty"><i class="fas fa-inbox"></i>当前筛选下暂无记录<br><span style="font-size:11px;margin-top:6px;display:inline-block">可切换上方类型、关系或闭环条件</span></div>';
  } else {
    var visible = typeof loadColumnConfig === 'function' ? loadColumnConfig('issueRisk') : null;
    var html =
      '<div class="has-scroll-hint"><table class="tbl issue-risk-tbl"><thead><tr>' +
      '<th data-col="id" style="width:72px">编号</th>' +
      '<th data-col="title" class="ir-title-cell" style="min-width:280px;width:320px">' +
      (kind === 'risk' ? '风险名称' : '问题名称') +
      '</th>' +
      '<th data-col="project" style="width:140px">所属项目</th>' +
      '<th data-col="severity" style="width:80px">严重程度</th>' +
      '<th data-col="priority" style="width:72px">优先级</th>' +
      '<th data-col="handler" style="width:108px">处理人</th>' +
      '<th data-col="owner" style="width:108px">提出人</th>' +
      '<th data-col="deadline" style="width:106px">计划解决</th>' +
      '<th data-col="days" style="width:90px">存续/逾期</th>' +
      '<th data-col="result" style="width:90px">状态</th>' +
      '<th data-col="action" class="col-actions" style="width:82px">操作</th>' +
      '</tr></thead><tbody>';

    items.forEach(function (item) {
      var statusClass =
        item.status === '未处理' || item.status === '激活'
          ? 'st-pending'
          : item.status === '处理中' || item.status === '跟踪中'
            ? 'st-progress'
            : item.status === '已解决' || item.status === '已关闭'
              ? 'st-success'
              : 'st-suspended';
      var priClass = item.priority === 'P0' ? 'p0' : item.priority === 'P1' ? 'p1' : item.priority === 'P3' ? 'p3' : 'p2';
      var rid = String(item.displayId || item.id || '-').replace(/'/g, '');
      var titleEsc = issueRiskEsc(item.title || item.name || '-');
      var kindTag = kind === 'risk' ? '<span class="badge blue">风险</span>' : '<span class="badge blue">问题</span>';
      var daysCell = item.isOverdue
        ? '<span class="sev-tag fatal">逾期 ' + Number(item.overdueDays || item.days || 0) + '天</span>'
        : Number(item.days || 0) + '天';
      var action = item.myAction || 'view';
      var actionLabel = action === 'process' ? '处理' : action === 'follow' ? '跟进' : '查看';
      var actionCls = action === 'process' ? 'action-btn primary' : 'action-btn';
      var rowFocus =
        issueRiskHighlightId && (rid === String(issueRiskHighlightId) || String(item.id) === String(issueRiskHighlightId));

      html += '<tr' + (rowFocus ? ' class="ir-row-focus"' : '') + ' data-ir-row="' + rid + '">';
      // 全站列表约定（PAGES-26~30）：编号列跳禅道原始详情，标题列进工作台处理入口
      var idOnClick = 'viewInZentao(&quot;' + rid + '&quot;,&quot;' + kind + '&quot;)';
      html +=
        '<td data-col="id" class="c-id"><button type="button" class="query-id-link" onclick="' +
        idOnClick +
        '">' +
        issueRiskEsc(item.displayId || item.id) +
        '</button></td>';
      html +=
        '<td data-col="title" class="ir-title-cell">' +
        '<div class="ir-title-wrap">' +
        kindTag +
        '<button type="button" class="query-title-link ir-title-main" onclick="openIssueRiskDetail(&quot;' +
        kind +
        '&quot;,&quot;' +
        rid +
        '&quot;)">' +
        titleEsc +
        '</button></div></td>';
      html += '<td data-col="project">' + issueRiskEsc(item.project || '-') + '</td>';
      html +=
        '<td data-col="severity"><span class="sev-tag ' +
        issueRiskSevClass(item.severity) +
        '">' +
        issueRiskSevLabel(item.severity) +
        '</span></td>';
      html += '<td data-col="priority"><span class="pri-tag ' + priClass + '">' + issueRiskEsc(item.priority || '-') + '</span></td>';
      html += '<td data-col="handler">' + issueRiskEsc(item.handler || '-') + '</td>';
      html += '<td data-col="owner">' + issueRiskEsc(item.submitter || item.creator || '-') + '</td>';
      html +=
        '<td data-col="deadline" style="white-space:nowrap;font-family:JetBrains Mono,monospace;font-size:11px">' +
        issueRiskEsc(item.planDate || '-') +
        '</td>';
      html += '<td data-col="days">' + daysCell + '</td>';
      html += '<td data-col="result"><span class="status-tag ' + statusClass + '">' + issueRiskEsc(item.status || '-') + '</span></td>';
      html +=
        '<td data-col="action" class="col-actions"><div class="ir-actions"><button type="button" class="' +
        actionCls +
        '" onclick="openIssueRiskDetail(&quot;' +
        kind +
        '&quot;,&quot;' +
        rid +
        '&quot;)">' +
        actionLabel +
        '</button></div></td>';
      html += '</tr>';
    });
    html += '</tbody></table></div>';
    wrap.innerHTML = html;

    if (visible && typeof applyTableColumnVisibility === 'function') {
      applyTableColumnVisibility(
        '#issueRiskTableWrap table',
        {
          id: 1,
          title: 2,
          project: 3,
          severity: 4,
          priority: 5,
          handler: 6,
          owner: 7,
          deadline: 8,
          days: 9,
          result: 10,
          action: 11
        },
        visible
      );
    }
  }

  var pagerHost = document.getElementById('issueRiskPagination');
  if (pagerHost && typeof renderPagination === 'function') {
    pagerHost.innerHTML = renderPagination({
      total: total,
      page: page,
      pageSize: ps,
      pageSizeOptions: [10, 20, 50],
      onPageChange: function (newPage) {
        issueRiskFilter.page = newPage;
        loadAndRenderIssueRisk();
      },
      onPageSizeChange: function (newSize) {
        issueRiskFilter.page = 1;
        issueRiskFilter.pageSize = Number(newSize) || 20;
        loadAndRenderIssueRisk();
      }
    });
  }
}
