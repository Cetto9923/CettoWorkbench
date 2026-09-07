function demandTreeItemMatchesBarFilters(it) {
  const scopeRaw = document.getElementById('demandScopeFilter')?.value || '';
  const scope = scopeRaw === 'subBiz' ? 'biz' : scopeRaw;
  const keyword = (document.getElementById('demandKeywordFilter')?.value || '').trim().toLowerCase();
  const selectedSystems = window.demandSystemSelection || new Set();
  const selectedAgiles = window.demandAgileSelection || new Set();
  const selectedTeams = window.demandTeamSelection || new Set();
  const selectedStatuses = window.demandZentaoStatusSelection || new Set();
  const selectedPools = window.demandPoolSelection || new Set();
  const valueStreamKey = window.demandValueStreamStageSelection || document.getElementById('demandValueStreamStageFilter')?.value || '';
  const todayStr = new Date().toISOString().slice(0, 10);

  const agileName = normalizeDemandAgile(it.agile);
  const valueStage = getDemandItemValueStage(it);
  const statusKey = String(it.status || it.zentaoStatus || '').toLowerCase();
  const relation = getDemandItemRelation(it);
  const iteration = getDemandItemSchedulePlanDate(it) === '-' ? getDemandItemIteration(it) : getDemandItemSchedulePlanDate(it);
  const deadline = String(getDemandItemDeadline(it) || '');
  const estimateLaunch = String(getDemandItemEstimateLaunch(it) || '');
  const compareDate = (estimateLaunch && estimateLaunch !== '-') ? estimateLaunch.slice(0, 10) : (deadline && deadline !== '-' ? deadline.slice(0, 10) : '');
  const delayed = !!compareDate && compareDate < todayStr;
  const suspended = getDemandItemHangLabel(it) === '是';
  const changing = getDemandItemChangeLabel(it) === '是';
  const approval = getDemandItemManagerReviewLabel(it) === '是';
  const mainSystem = getDemandItemMainSystem(it);
  const assignee = getDemandItemAssignee(it);
  const createdBy = it.createdBy || '';
  const proposer = String(it.proposer || '');
  const demandOwner = getDemandItemOwner(it);
  const tester = getDemandItemTester(it);
  const importance = String(it.severity || it.importance || '');
  const category = it.category || '';
  const source = it.source || '';
  const container = String(it.pool || it.containerType || '');
  const closedReason = String(it.closedReason || '');

  if (currentDemandPerspective === 'myLead' && relation !== '我牵头') return false;
  if (currentDemandPerspective === 'mySupport' && relation !== '我配合') return false;
  if (currentDemandPerspective === 'myCreate' && createdBy !== getDemandCurrentUser()) return false;
  if (currentDemandPerspective === 'my' && !['我牵头','我配合','我创建','我关注'].includes(relation)) return false;

  // 业务需求 = 业务需求 + 子业务需求；研发需求单独
  if (scope === 'biz') {
    if (it.nodeType !== 'biz' && it.nodeType !== 'subBiz') return false;
  } else if (scope && it.nodeType !== scope) {
    return false;
  }
  if (selectedAgiles.size && !selectedAgiles.has(agileName)) return false;
  if (selectedTeams.size && !selectedTeams.has(it.team)) return false;
  if (selectedSystems.size) {
    const hit = (it.systems || []).some((name) => selectedSystems.has(name));
    if (!hit) return false;
  }
  if (selectedPools.size) {
    const poolKey = String(it.poolId || it.pool || '');
    if (!selectedPools.has(poolKey)) return false;
  }
  if (valueStreamKey && String(it.valueStreamStage || '').toLowerCase() !== String(valueStreamKey).toLowerCase()) return false;
  if (selectedStatuses.size) {
    const ztMatched = Array.from(selectedStatuses).some((status) => String(status).toLowerCase() === statusKey);
    if (!ztMatched) return false;
  }
  if (currentDemandActionFilter !== 'all') {
    const actionMatch = {
      today: () => delayed || String(it.actionType || '') === 'today',
      overdue: () => delayed || String(it.riskType || '') === 'overdue' || String(it.status || '').includes('超期'),
      blocked: () => ['flowBlocked','testBlocked','blocked'].includes(String(it.riskType || '')) || String(it.status || '').includes('阻塞'),
      bizConfirm: () => /业务|评审委员会|主管部门|李总/.test(String(it.nextOwner || '') + String(it.owner || '')),
      unscheduled: () => !getDemandItemSchedulePlanDate(it) || getDemandItemSchedulePlanDate(it) === '-',
      unaccepted: () => valueStage === '验收' || /待验收|验收/.test(String(it.phase || '') + String(it.status || '')),
      undelivered: () => valueStage === '交付' || /待交付|交付/.test(String(it.phase || '') + String(it.status || '')),
      issueRisk: () => getDemandItemFocusLabel(it) === '是' || ['flowBlocked','testBlocked','blocked','overdue'].includes(String(it.riskType || '')) || /风险|阻塞|超期/.test(String(it.status || ''))
    }[currentDemandActionFilter];
    if (actionMatch && !actionMatch()) return false;
  }
  if ((document.getElementById('demandCategoryFilter')?.value || '') && category !== document.getElementById('demandCategoryFilter').value) return false;
  if ((document.getElementById('demandImportanceFilter')?.value || '') && getDemandItemSeverityLabel({severity: importance}) !== document.getElementById('demandImportanceFilter').value) return false;
  if ((document.getElementById('demandPriorityFilter')?.value || '') && String(it.priority || '') !== document.getElementById('demandPriorityFilter').value) return false;
  if ((document.getElementById('demandDeptFilter')?.value || '') && source !== document.getElementById('demandDeptFilter').value) return false;
  if ((document.getElementById('demandIterationFilter')?.value || '') && !String(iteration || '').includes(document.getElementById('demandIterationFilter').value)) return false;
  if ((document.getElementById('demandWindowFilter')?.value || '') && closedReason !== document.getElementById('demandWindowFilter').value) return false;
  if ((document.getElementById('demandContainerFilter')?.value || '') && container !== document.getElementById('demandContainerFilter').value) return false;
  if ((document.getElementById('demandDelayedFilter')?.value || '') === 'yes' && !delayed) return false;
  if ((document.getElementById('demandDelayedFilter')?.value || '') === 'no' && delayed) return false;
  if ((document.getElementById('demandSuspendedFilter')?.value || '') === 'yes' && !suspended) return false;
  if ((document.getElementById('demandSuspendedFilter')?.value || '') === 'no' && suspended) return false;
  if ((document.getElementById('demandChangingFilter')?.value || '') === 'yes' && !changing) return false;
  if ((document.getElementById('demandChangingFilter')?.value || '') === 'no' && changing) return false;
  if ((document.getElementById('demandApprovalFilter')?.value || '') === 'yes' && !approval) return false;
  if ((document.getElementById('demandApprovalFilter')?.value || '') === 'no' && approval) return false;
  const includesFilter = (id, target) => {
    const value = (document.getElementById(id)?.value || '').trim().toLowerCase();
    return !value || String(target || '').toLowerCase().includes(value);
  };
  if (!includesFilter('demandAssigneeFilter', assignee)) return false;
  if (!includesFilter('demandCreatedByFilter', createdBy)) return false;
  if (!includesFilter('demandProposerFilter', proposer)) return false;
  if (!includesFilter('demandOwnerFilter', demandOwner)) return false;
  if (!includesFilter('demandTesterFilter', tester)) return false;
  const betweenDate = (value, startId, endId) => {
    const start = document.getElementById(startId)?.value || '';
    const end = document.getElementById(endId)?.value || '';
    if (!start && !end) return true;
    if (!value) return false;
    if (start && value < start) return false;
    if (end && value > end) return false;
    return true;
  };
  if (!betweenDate(String(it.createdDate || ''), 'demandCreatedFromFilter', 'demandCreatedToFilter')) return false;
  if (!betweenDate(String(it.deliverDate || it.deliveryDate || ''), 'demandDeliveryFromFilter', 'demandDeliveryToFilter')) return false;
  if (!betweenDate(String(getDemandItemEstimateLaunch(it) || ''), 'demandOnlineFromFilter', 'demandOnlineToFilter')) return false;
  if (keyword) {
    const hay = `${it.id} ${it.reqCode || ''} ${it.title} ${it.desc || ''} ${mainSystem} ${(it.systems || []).join(' ')} ${agileName} ${it.phase || ''} ${valueStage} ${it.statusLabel || ''} ${statusKey}`.toLowerCase();
    if (!hay.includes(keyword)) return false;
  }
  return true;
}

function renderDemandListView() {
  closeAllDemandRowMenus();
  const container = document.getElementById('demandListView');
  renderDemandHeaderCounts();
  const useApiRows = (typeof demandWorkspaceInitialized !== 'undefined' && demandWorkspaceInitialized)
    || ((typeof demandWorkspaceRows !== 'undefined' && Array.isArray(demandWorkspaceRows)) && demandWorkspaceRows.length > 0);
  const sourceRows = useApiRows ? ((typeof demandWorkspaceRows !== 'undefined' && demandWorkspaceRows) || DEMAND_TREE_DATA) : DEMAND_TREE_DATA;
  ensureDefaultDemandExpansion(sourceRows);
  const { nodeMap, childrenMap } = getDemandNodeMeta(sourceRows);
  const matchedNodeIds = new Set();
  const matchesNode = (it) => useApiRows ? true : demandTreeItemMatchesBarFilters(it);

  sourceRows.forEach((it) => {
    if (matchesNode(it)) matchedNodeIds.add(it.id);
  });
  const visibleNodeIds = new Set();
  if (useApiRows) {
    sourceRows.forEach((it) => visibleNodeIds.add(it.id));
  } else {
    matchedNodeIds.forEach((id) => {
      visibleNodeIds.add(id);
      let current = nodeMap.get(id);
      while (current && current.parentId) {
        visibleNodeIds.add(current.parentId);
        current = nodeMap.get(current.parentId);
      }
      collectNodeWithDescendants(id, childrenMap).forEach((descId) => visibleNodeIds.add(descId));
    });
  }

  const rootNodes = useApiRows
    ? sourceRows.filter((it) => visibleNodeIds.has(it.id))
    : sourceRows.filter((it) => visibleNodeIds.has(it.id) && (it.nodeType === 'biz' || (it.nodeType === 'rd' && !it.parentId)));
  if (!useApiRows) {
    demandTotal = rootNodes.length;
  }
  const totalPages = WBPagination.totalPages(demandTotal, demandPageSize);
  demandPage = WBPagination.clampPage(demandPage, totalPages);
  const start = useApiRows ? 0 : WBPagination.offset(demandPage, demandPageSize);
  const end = useApiRows ? rootNodes.length : start + demandPageSize;
  const pageData = useApiRows ? rootNodes : rootNodes.slice(start, end);
  
  const expandedIds = window.expandedDemandIds || new Set();
  
  const demandVisible = loadColumnConfig('demand');
  const showDemandCol = (key) => key !== 'action' && demandVisible.has(key);
  const demandHeaderDefs = [
    ['id','编号','88px'],['title','需求主题','400px'],['stageDelay','阶段延期','88px'],['priority','优先级','52px'],['severity','重要程度','80px'],['category','需求类别','88px'],['source','提出部门','88px'],['status','状态','88px'],['phase','价值流阶段','108px'],['assignee','指派人','80px'],['owner','需求负责人','96px'],['tester','测试负责人','80px'],['acceptOwner','验收负责人','80px'],['reviewer','业务评审人','80px'],['deadline','期望完成时间','108px'],['online','预计上线时间','108px'],['schedulePlan','排定计划日期','114px'],['delivery','交付时间','108px'],['hang','挂起','52px'],['change','变更中','52px'],['managerReview','系统主管审批','96px'],['watch','操作','52px'],['created','创建时间','108px'],['creator','由谁创建','80px'],['reason','卡点原因','140px']
  ];
  const typeLabel = { biz: '业务需求', subBiz: '子业务需求', rd: '研发需求' };
  const versionLabel = (item) => getDemandItemIteration(item);
  const systemCell = (item) => {
    const list = item.systems || [];
    if (!list.length) return '-';
    if (item.nodeType === 'rd') return `<span style="font-size:12px;color:var(--t2)">${list[0]}</span>`;
    const main = list[0];
    const more = list.slice(1);
    return `<span style="font-size:12px;color:var(--t2)">${main}</span>${more.length ? `<span class="status-tag st-pending" style="margin-left:6px">+${more.length}</span><span style="font-size:11px;color:var(--t3);margin-left:4px">${more.join(' / ')}</span>` : ''}`;
  };
  const indentByType = { biz: 0, subBiz: 24, rd: 48 };
  /* 标题旁：系统只落在研发需求上；业务需求 / 子业务需求不写系统与主配角色 */
  const systemRoleHint = (item) => {
    if (item.nodeType !== 'rd') return '';
    const sys = getDemandItemPrimarySystem(item);
    const isMain = getDemandItemIsMainRole(item);
    // 独立排期研需无真实系统 (system='—'), 主配角色 hint 无意义, 不渲染。
    if (!sys || sys === '—' || sys === '-') return '';
    return `<span class="dw-system-hint">${vfH(sys)}·${isMain ? '主' : '配'}</span>`;
  };
  const titleBlockerTag = (blocker) => {
    if (!blocker || blocker.type === 'overdue') return '';
    if (blocker.type === 'blocked' || blocker.type === 'approve' || blocker.type === 'test' || blocker.type === 'gate') {
      return `<span class="dw-inline-alert">${vfH(blocker.label)}</span>`;
    }
    if (blocker.type === 'risk') return `<span class="dw-inline-alert dw-inline-alert--risk">${vfH(blocker.label)}</span>`;
    return '';
  };
  const yesBadge = (label, kind) => label === '是'
    ? `<span class="dw-yes-badge${kind ? ' ' + kind : ''}">是</span>`
    : '<span class="dw-no-quiet">—</span>';
  const renderDemandRowActions = (item) => {
    const id = item.id;
    const title = item.title || item.reqCode || id;
    const stage = getDemandItemValueStage(item);
    const rowStatusLabel = item.statusLabel || item.status || '-';
    const blocker = getDemandItemBlockerInfo(item);
    const stageText = stage + rowStatusLabel;
    let cta = null;
    if (/待验收|已验收/.test(stageText)) {
      cta = { label: '验收', kind: 'primary', handler: `showToast('已进入验收跟进：${id}')`, aria: `处理 ${title} 验收` };
    } else if (/待排期|已澄清|待评审/.test(stageText) || !getDemandItemSchedulePlanDate(item) || getDemandItemSchedulePlanDate(item) === '-') {
      cta = { label: '排期', kind: 'primary', handler: `navTo('schedule');showToast('已带入需求排期：${id}')`, aria: `为 ${title} 进入排期` };
    }
    /* 口径（主人 2026-08-12 修订）：
       - 最多 3 个可见；行间左对齐（CTA 槽）
       - 有主操作（去排期/建任务等）时：催办/挂起进「…」
       - 无主操作时：催办/挂起直出，禁止只剩一个「…」 */
    const primaryInline = [];
    const softFold = [];
    if (item.nodeType !== 'biz' || /开发中|待开发|测试中/.test(stageText)) {
      primaryInline.push({ label: '建任务', kind: 'warn', handler: `showToast('已生成任务草稿：${id}')` });
    }
    if (blocker || /阻塞|超期|挂起/.test(String(item.status || '') + String(item.riskType || '') + rowStatusLabel)) {
      softFold.push({ label: '催办', handler: `showToast('已生成催办草稿：${id}')` });
    }
    if (getDemandItemChangeLabel(item) === '是') {
      primaryInline.push({ label: '变更记录', handler: `showToast('已打开变更记录：${id}')` });
    } else if (getDemandItemHangLabel(item) !== '是') {
      softFold.push({ label: '挂起', handler: `showToast('已生成挂起申请草稿：${id}')` });
    }
    const MAX_VISIBLE = 3;
    const hasPrimary = !!(cta || primaryInline.length);
    let inlineExtras = [];
    let overflow = [];
    if (hasPrimary) {
      overflow = softFold.slice();
      const ctaCount = cta ? 1 : 0;
      let moreNeeded = overflow.length > 0;
      let budget = Math.max(0, MAX_VISIBLE - ctaCount - (moreNeeded ? 1 : 0));
      if (primaryInline.length > budget) {
        moreNeeded = true;
        budget = Math.max(0, MAX_VISIBLE - ctaCount - 1);
      }
      inlineExtras = primaryInline.slice(0, budget);
      overflow.push(...primaryInline.slice(budget));
    } else {
      /* 无主按钮：催办/挂起直接露出 */
      inlineExtras = softFold.slice(0, MAX_VISIBLE);
      overflow = softFold.slice(MAX_VISIBLE);
    }
    /* 兜底：绝不出现「唯一可见控件是 …」 */
    if (!cta && !inlineExtras.length && overflow.length) {
      const take = Math.min(MAX_VISIBLE, overflow.length);
      inlineExtras = overflow.splice(0, take);
    }
    const extraHtml = inlineExtras.map((m) => {
      const cls = m.kind ? `action-btn dw-action-${m.kind}` : 'action-btn';
      return `<button type="button" class="${cls}" aria-label="${vfH(m.label)} ${vfH(title)}" onclick="event.stopPropagation();${m.handler}">${m.label}</button>`;
    }).join('');
    const menuHtml = overflow.length
      ? `<div class="dw-action-more"><button type="button" class="action-btn dw-more-btn" aria-label="更多操作" title="更多操作" onclick="event.stopPropagation();toggleDemandRowMenu(this)"><i class="fas fa-ellipsis"></i></button><div class="dw-action-menu" hidden>${overflow.map((m) => `<button type="button" class="dw-action-menu-item" onclick="event.stopPropagation();${m.handler};closeAllDemandRowMenus()">${m.label}</button>`).join('')}</div></div>`
      : '';
    const ctaHtml = cta
      ? `<button type="button" class="action-btn dw-action-${cta.kind}" aria-label="${vfH(cta.aria)}" onclick="event.stopPropagation();${cta.handler}">${cta.label}</button>`
      : '';
    const ctaSlotHtml = cta ? `<span class="dw-action-cta-slot">${ctaHtml}</span>` : '';
    return `<div class="dw-row-actions ${cta ? 'has-cta' : 'no-cta'}">${ctaSlotHtml}${extraHtml}${menuHtml}</div>`;
  };
  const renderRow = (item) => {
    const priClass = item.priority === 'P0' ? 'p0' : item.priority === 'P1' ? 'p1' : item.priority === 'P3' ? 'p3' : 'p2';
    const statusClass = SM2[item.status] || 'st-pending';
    const hasChildren = (childrenMap.get(item.id) || []).some((child) => visibleNodeIds.has(child.id));
    const isExpanded = expandedIds.has(item.id);
    // 无父节点 (biz 根 / 独立研需根) 不缩进; 缩进仅用于挂父的 subBiz/rd。
    const indent = item.parentId ? (indentByType[item.nodeType] || 0) : 0;
    const toggleIcon = hasChildren
      ? `<button type="button" class="dw-tree-toggle" aria-label="${isExpanded ? '收起' : '展开'}" title="${isExpanded ? '收起' : '展开'}" onclick="event.stopPropagation();toggleDemandExpand('${item.id}', event)"><i class="fas ${isExpanded ? 'fa-chevron-down' : 'fa-chevron-right'}"></i></button>`
      : '<span class="dw-tree-toggle empty" aria-hidden="true"></span>';
    /* 2026-08-12 ISSUE：与排期工作台样式对齐。
     * BR / SBR / RD 三类行在标题前加层级 chip（业务需求 / 子业务需求 / 研发需求），
     * 风格与排期 .schedule-rd-micro 一致；RD 行额外加 .schedule-rd-branch 分支线。 */
    const tierChipHtml = `<span class="dw-tier-chip dw-tier-chip--${item.nodeType}">${vfH(typeLabel[item.nodeType] || '需求')}</span>`;
    const branchHtml = item.nodeType === 'rd'
      ? '<span class="dw-tier-branch"></span>'
      : '';
    const displayId = item.reqCode || item.id;
    const blocker = getDemandItemBlockerInfo(item);
    const statusRiskTag = (() => {
      if (!blocker) return '';
      if (blocker.type === 'overdue') return '<span class="dw-status-risk dw-status-risk--overdue">超期</span>';
      if (blocker.type === 'blocked' || blocker.type === 'approve') return '<span class="dw-status-risk dw-status-risk--blocked">阻塞</span>';
      return '<span class="dw-status-risk dw-status-risk--risk">卡点</span>';
    })();
    const reasonCell = blocker ? blocker.reason : bottleneckReasonFromItem(item);
    const childCount = (childrenMap.get(item.id) || []).filter((c) => visibleNodeIds.has(c.id)).length;
    const countHint = item.nodeType === 'biz' && childCount ? `<span class="dw-child-count">· ${childCount} 子项</span>` : '';
    const rowTier = item.nodeType === 'biz' ? 'dw-tier-biz' : item.nodeType === 'subBiz' ? 'dw-tier-sub' : 'dw-tier-rd';
    const owner = getDemandItemOwner(item);
    const assignee = getDemandItemAssignee(item);
    const onlineDate = getDemandItemEstimateLaunch(item);
    const createdDate = getDemandItemCreatedDate(item);
    const creator = item.createdBy || getDemandItemProposer(item);
    const deliveryDate = getDemandItemDeliveryDate(item);
    const tester = getDemandItemTester(item);
    const acceptOwner = getDemandItemAcceptOwner(item);
    const reviewer = getDemandItemReviewer(item);
    const statusLabel = item.statusLabel || item.status || '-';
    const statusClassName = typeof demandStatusCssClass === 'function'
      ? demandStatusCssClass(item.status || item.zentaoStatus)
      : ('status-tag ' + (SM2[item.status] || 'st-pending'));
    // r2：挂起/变更中——默认列里已移除，标题旁补轻量 tag，避免信息丢失
    const isHang = getDemandItemHangLabel(item) === '是';
    const isChange = getDemandItemChangeLabel(item) === '是';
    const stateInlineTags = (isHang || isChange)
      ? `<span class="dw-state-inline">${isHang ? '<span class="dw-state-tag dw-state-tag--hang">已挂起</span>' : ''}${isChange ? '<span class="dw-state-tag dw-state-tag--change">变更中</span>' : ''}</span>`
      : '';
    const stageDelayLabel = getDemandStageDelayLabel(item);
    const stageDelayCell = stageDelayLabel ? `<span class="dw-stage-delay">${vfH(stageDelayLabel)}</span>` : '<span style="color:var(--t3)">—</span>';
    const demandJsId = typeof escapeJsString === 'function' ? escapeJsString(item.id) : String(item.id || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'");
    const zentaoJsId = typeof escapeJsString === 'function' ? escapeJsString(displayId) : String(displayId || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'");
    const zentaoType = item.nodeType === 'rd' ? 'story' : 'demand';
    const detailHandler = item.nodeType === 'rd'
      ? `showDemandDetail('${demandJsId}')`
      : `openBusinessDemandDetail('${demandJsId}')`;
    const cellMap = {
      id: `<td data-col="id" style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--t2);white-space:nowrap"><button type="button" class="dw-id-link" title="在禅道中查看原始详情" onclick="event.stopPropagation();viewInZentao('${zentaoJsId}','${zentaoType}')">${vfH(displayId)}</button></td>`,
      title: `<td data-col="title" style="padding-left:${indent}px"><span class="dw-title-line">${toggleIcon}${systemRoleHint(item)}<button type="button" class="dw-title-text" title="${vfH(item.title)}" onclick="event.stopPropagation();${detailHandler}">${vfH(item.title)}</button>${stateInlineTags}${countHint}</span></td>`,
      stageDelay: `<td data-col="stageDelay" style="font-size:12px;white-space:nowrap">${stageDelayCell}</td>`,
      priority: `<td data-col="priority"><span class="pri-tag ${priClass}" style="font-size:10px">${item.priority || '-'}</span></td>`,
      severity: `<td data-col="severity" style="font-size:12px">${getDemandItemSeverityLabel(item)}</td>`,
      category: `<td data-col="category" style="font-size:12px">${getDemandItemCategoryLabel(item)}</td>`,
      source: `<td data-col="source" style="font-size:12px">${getDemandItemSourceLabel(item)}</td>`,
      status: `<td data-col="status"><div class="dw-status-stack"><span class="${statusClassName}">${vfH(statusLabel)}</span>${statusRiskTag}</div></td>`,
      phase: `<td data-col="phase"><span class="status-tag st-progress">${getDemandItemValueStage(item)}</span></td>`,
      assignee: `<td data-col="assignee" style="font-size:12px">${(assignee && assignee !== '-') ? assignee : '—'}</td>`,
      owner: `<td data-col="owner" style="font-size:12px">${owner}</td>`,
      tester: `<td data-col="tester" style="font-size:12px;color:var(--t2)">${tester}</td>`,
      acceptOwner: `<td data-col="acceptOwner" style="font-size:12px;color:var(--t2)">${acceptOwner}</td>`,
      reviewer: `<td data-col="reviewer" style="font-size:12px;color:var(--t2)">${reviewer}</td>`,
      deadline: `<td data-col="deadline" style="font-size:12px;color:var(--t2);white-space:nowrap">${getDemandItemDeadline(item)}</td>`,
      online: `<td data-col="online" style="font-size:12px;color:var(--t2);white-space:nowrap">${onlineDate}</td>`,
      schedulePlan: `<td data-col="schedulePlan" style="font-size:12px;color:var(--t2);white-space:nowrap">${getDemandItemSchedulePlanDate(item)}</td>`,
      delivery: `<td data-col="delivery" style="font-size:12px;color:var(--t2);white-space:nowrap">${deliveryDate}</td>`,
      hang: `<td data-col="hang" style="font-size:12px">${yesBadge(getDemandItemHangLabel(item), 'hang')}</td>`,
      change: `<td data-col="change" style="font-size:12px">${yesBadge(getDemandItemChangeLabel(item), 'change')}</td>`,
      managerReview: `<td data-col="managerReview" style="font-size:12px">${yesBadge(getDemandItemManagerReviewLabel(item))}</td>`,
      watch: (() => { const watched = !!item.followed; const jsId = typeof escapeJsString==='function' ? escapeJsString(item.id) : String(item.id || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'"); return `<td data-col="watch" style="font-size:12px;text-align:center"><button type="button" class="demand-watch-toggle${watched?' is-watched':''}" title="${watched?'取消关注':'关注'}" aria-label="${watched?'取消关注':'关注'}" onclick="event.stopPropagation();toggleDemandWatch('${jsId}', ${watched})"><i class="${watched?'fas':'far'} fa-star"></i></button></td>`; })(),
      created: `<td data-col="created" style="font-size:12px;color:var(--t2);white-space:nowrap">${createdDate}</td>`,
      creator: `<td data-col="creator" style="font-size:12px;color:var(--t2)">${creator}</td>`,
      reason: `<td data-col="reason" style="font-size:12px;color:var(--t2)">${reasonCell}</td>`
    };
    html += `<tr class="${item.parentId ? 'child-row' : 'parent-row'} ${rowTier} ${item.nodeType === 'biz' ? 'biz-row' : ''} ${isExpanded ? 'expanded' : ''}">${demandHeaderDefs.filter(([k]) => showDemandCol(k)).map(([k]) => cellMap[k]).join('')}</tr>`;
    if (!hasChildren || !isExpanded) return;
    (childrenMap.get(item.id) || []).forEach((child) => {
      if (!visibleNodeIds.has(child.id)) return;
      renderRow(child);
    });
  };
  let html = '';
  if (demandWorkspaceLoading && !pageData.length) {
    html = '<div class="po-list-empty" style="padding:40px 12px;text-align:center;color:var(--t3)"><i class="fas fa-spinner fa-spin"></i> 正在加载需求列表…</div>';
  } else if (!pageData.length) {
    html = '<div class="po-list-empty" style="padding:40px 12px;text-align:center;color:var(--t3)"><div class="empty-state-title">暂无匹配需求</div><div style="font-size:12px;margin-top:6px">可调整状态、关键词或高级筛选条件</div></div>';
  } else {
    html = '<div class="dw-table-scroll po-list-scroll has-scroll-hint" data-po-list-scroll><table data-wb-colkey="wb-colw:po:demand"><thead><tr>' +
      demandHeaderDefs.filter(([k]) => showDemandCol(k)).map(([k,label,w]) => `<th data-col="${k}" style="width:${w}">${label}</th>`).join('') +
      '</tr></thead><tbody>';
    pageData.forEach((item) => {
      renderRow(item);
    });
    html += '</tbody></table></div>';
  }
  container.innerHTML = html;
  if (window.WorkbenchCols) WorkbenchCols.bindPreset(document.querySelector('#demandListView table'), 'po:demand');
  renderDemandPaginationBars();
  renderDemandAnalyticsPanel();
  updateDemandPageInfo();
}

function toggleDemandExpand(id, event) {
  if (event) {
    event.preventDefault();
    event.stopPropagation();
  }
  window.expandedDemandIds = window.expandedDemandIds || new Set();
  window.__demandExpandInitialized = true;
  if (window.expandedDemandIds.has(id)) {
    window.expandedDemandIds.delete(id);
  } else {
    window.expandedDemandIds.add(id);
  }
  renderDemandListView();
}

document.addEventListener('DOMContentLoaded', function() {
  // Workbench iframe 嵌入：标记后隐藏与壳重复的文案
  try {
    if (new URLSearchParams(location.search).has('embed') || window.self !== window.top) {
      document.body.classList.add('workbench-embed');
    }
  } catch (e) { document.body.classList.add('workbench-embed'); }
  initDemandDefaultFilters();
  try { window.demandSavedQueries = JSON.parse(localStorage.getItem('demand_saved_queries_v1') || '[]') || []; } catch (e) { window.demandSavedQueries = []; }
  try { syncDemandFieldViewFromRole({ silent: true, skipRender: true, force: true }); } catch (e) {}
  renderDemandHeaderCounts();
  renderDemandDimensionSummary();
  renderDemandSavedQueries();
  try { setRole(detectInitWorkbenchRole(), { skipNav: true }); } catch (e) { console.warn('setRole init failed', e); }
  try {
    window.addEventListener('storage', function (ev) {
      if (ev && ev.key === 'wb_active_role') {
        lastSyncedWorkbenchRole = null;
        syncDemandFieldViewFromRole({ silent: true });
      }
    });
  } catch (e2) {}
});

function treeItemToKanbanCol(it) {
  const lbl = String(it.phase || '');
  if (/澄清|待澄清|新需求/.test(lbl)) return 'discover';
  if (/评审|审批|驳回|待评审|主管部门/.test(lbl)) return 'review';
  if (/排期|待排期/.test(lbl)) return 'plan';
  if (/研发|开发|设计|联调|开发中|待开发|设计中/.test(lbl)) return 'build';
  if (/测试|验收|交付/.test(lbl)) return 'accept';
  if (/发布|上线|关闭|评价|已完成/.test(lbl)) return 'release';
  return 'discover';
}

function poReqToKanbanCol(r) {
  const ph = r.phase;
  if (ph === 'clarify') return 'discover';
  if (ph === 'wait' || ph === 'refuse' || ph === 'approve') return 'review';
  if (ph === 'schedule') return 'plan';
  if (ph === 'dev') return 'build';
  if (ph === 'test' || ph === 'accept') return 'accept';
  if (ph === 'delivery' || ph === 'release' || ph === 'closed') return 'release';
  if (ph === 'hang') return 'plan';
  return 'discover';
}

function renderDemandKanbanView() {
  const board = document.querySelector('#demandKanbanView .kanban-board');
  if (!board) return;

  const columns = [
    { key: 'discover', title: '需求提出/澄清', desc: '需求成型与范围确认', wip: 30, items: [] },
    { key: 'review', title: '评审/审批', desc: '评审会与审批流', wip: 30, items: [] },
    { key: 'plan', title: '排期/转研', desc: '进入时间盒与转研', wip: 40, items: [] },
    { key: 'build', title: '研发/联调', desc: '研发与联调验证', wip: 60, items: [] },
    { key: 'accept', title: '测试/验收/交付', desc: '测试、验收、发起交付', wip: 50, items: [] },
    { key: 'release', title: '发布/评价', desc: '发布后观察与评价', wip: 30, items: [] }
  ];
  const colMap = Object.fromEntries(columns.map((c) => [c.key, c]));
  const roots = DEMAND_TREE_DATA.filter((item) => item.nodeType === 'biz').filter((it) => demandTreeItemMatchesBarFilters(it));
  const usedIds = new Set();

  roots.forEach((item) => {
    const key = treeItemToKanbanCol(item);
    usedIds.add(item.id);
    colMap[key].items.push({
      id: item.reqCode || item.id,
      title: item.title,
      priority: item.priority,
      agile: item.agile,
      risk: item.riskType ? 'warning' : 'normal',
      riskLabel: item.riskType === 'flowBlocked' ? '审批阻塞' : item.riskType === 'testBlocked' ? '测试阻塞' : item.riskType || ''
    });
  });

  (poReqData || []).forEach((r) => {
    if (usedIds.has(r.id)) return;
    const key = poReqToKanbanCol(r);
    if (colMap[key].items.length === 0) {
      usedIds.add(r.id);
      colMap[key].items.push({
        id: r.id,
        title: r.title,
        priority: r.pri,
        agile: r.agile,
        risk: (r.risks && r.risks.length) ? 'warning' : 'normal',
        riskLabel: (r.risks || []).join(' ')
      });
    }
  });

  columns.forEach((col) => {
    if (col.items.length) return;
    const add = (poReqData || []).find((r) => !usedIds.has(r.id) && poReqToKanbanCol(r) === col.key);
    if (add) {
      usedIds.add(add.id);
      col.items.push({
        id: add.id,
        title: add.title,
        priority: add.pri,
        agile: add.agile,
        risk: 'normal',
        riskLabel: ''
      });
    }
  });

  let html = '';
  columns.forEach((col) => {
    html += `<div class="kanban-column">
      <div class="kanban-column-hdr">${col.title}<span class="kc-count">${col.items.length}/${col.wip}</span></div>
      <div style="padding:0 14px 8px;color:var(--t3);font-size:11px">${col.desc}</div>
      <div class="kanban-column-body">`;
    if (col.items.length === 0) {
      html += `<div class="empty-state">
        <div class="empty-state-icon"><i class="fas fa-inbox"></i></div>
        <div class="empty-state-title">暂无需求</div>
      </div>`;
    } else {
      col.items.forEach((item) => {
        const priClass = item.priority === 'P0' ? 'p0' : item.priority === 'P1' ? 'p1' : item.priority === 'P3' ? 'p3' : 'p2';
        const riskClass = item.risk ? `risk-${item.risk}` : '';
        const riskIcon = item.risk === 'danger' ? '<i class="fas fa-exclamation-circle" style="color:var(--red)"></i>' :
          item.risk === 'warning' ? '<i class="fas fa-clock" style="color:var(--orange)"></i>' : '';
        const safeId = String(item.id).replace(/'/g, "\\'");
        html += `<div class="kanban-card ${riskClass}" onclick="showDemandDetail('${safeId}')">
          <div class="kc-title">${item.title}</div>
          <div class="kc-meta">
            <span class="pri-tag ${priClass}">${item.priority}</span>
            <span>${item.agile}</span>
            ${riskIcon ? `<span style="margin-left:auto">${riskIcon}</span>` : ''}
          </div>
          <div style="margin-top:8px;display:flex;justify-content:flex-end">
            <button type="button" class="action-btn primary" onclick="event.stopPropagation();navTo('schedule');showToast('已跳转需求排期：${safeId}')">排期</button>
          </div>
        </div>`;
      });
    }
    html += '</div></div>';
  });
  board.innerHTML = html;
  clearDemandPaginationBars();
}

function renderDemandGroupView() {
  const container = document.querySelector('#demandGroupView .group-container');
  if (!container) return;

  let flat = DEMAND_TREE_DATA.filter((row) => demandTreeItemMatchesBarFilters(row));
  const groups = [
    { title: 'P0 - 紧急', pri: 'P0', items: [] },
    { title: 'P1 - 重要', pri: 'P1', items: [] },
    { title: 'P2 - 普通', pri: 'P2', items: [] }
  ];
  groups.forEach((g) => {
    g.items = flat.filter((it) => (it.priority || 'P2') === g.pri);
  });
  groups.forEach((g) => {
    if (g.items.length) return;
    const extra = (poReqData || []).filter((r) => r.pri === g.pri).slice(0, 5);
    extra.forEach((r) => {
      g.items.push({ id: r.id, title: r.title, phase: r.phaseLabel, deadline: r.deadlineStr || r.deadline || '-' });
    });
  });

  let html = '';
  groups.forEach((group, idx) => {
    html += `<div class="group-section${idx === 0 ? ' open' : ''}" onclick="toggleGroupSection(this)">
      <div class="group-section-hdr">
        <i class="fas fa-chevron-right gs-toggle"></i>
        <span class="gs-title">${group.title}</span>
        <span class="gs-count">${group.items.length}</span>
      </div>
      <div class="group-section-body">
        <div style="display:flex;flex-direction:column;gap:8px">`;
    group.items.forEach((item) => {
      const sid = String(item.id).replace(/'/g, "\\'");
      html += `<div style="padding:10px 12px;border:1px solid var(--border-lt);border-radius:4px;cursor:pointer" onclick="showDemandDetail('${sid}')">
        <div style="font-weight:500;margin-bottom:4px">${item.id} - ${item.title}</div>
        <div style="font-size:11px;color:var(--t3);display:flex;gap:12px">
          <span>阶段：${item.phase || '-'}</span>
          <span>截止：${item.deadline || '-'}</span>
        </div>
      </div>`;
    });
    html += '</div></div></div>';
  });
  container.innerHTML = html;
  clearDemandPaginationBars();
}

function toggleGroupSection(el) {
  el.classList.toggle('open');
}

function demandPagePrev() {
  if (demandPage > 1) {
    demandPage--;
    if (demandCurrentView === 'list') renderDemandListView();
  }
}

function demandPageNext() {
  const totalPages = Math.ceil(demandTotal / demandPageSize);
  if (demandPage < totalPages) {
    demandPage++;
    if (demandCurrentView === 'list') renderDemandListView();
  }
}

function updateDemandPageInfo() {
  const totalPages = Math.ceil(demandTotal / demandPageSize) || 1;
  const pageInfo = document.getElementById('demandPageInfo');
  if (pageInfo) pageInfo.textContent = `第 ${demandPage} 页 / 共 ${totalPages} 页`;
  const countEl = document.getElementById('demandResultCount');
  if (countEl) countEl.textContent = String(demandTotal || 0);
}

function renderDemandPaginationBars() {
  const total = Number(demandTotal || 0);
  const page = Number(demandPage || 1);
  const pageSize = Number(demandPageSize || 20);
  const renderBar = (id) => {
    const host = document.getElementById(id);
    if (!host) return;
    host.innerHTML = renderPagination({
      total,
      page,
      pageSize,
      pageSizeOptions: [5, 10, 15, 20, 30, 50],
      onPageChange: (newPage) => { demandPage = newPage; if (typeof triggerDemandWorkspaceRefresh === 'function') triggerDemandWorkspaceRefresh({ page: newPage }); else renderDemandListView(); },
      onPageSizeChange: (newSize) => { demandPage = 1; demandPageSize = newSize; if (typeof triggerDemandWorkspaceRefresh === 'function') triggerDemandWorkspaceRefresh({ page: 1, pageSize: newSize }); else renderDemandListView(); }
    }) || '';
  };
  renderBar('demandPaginationBottom');
}

function clearDemandPaginationBars() {
  ['demandPaginationBottom'].forEach((id) => {
    const host = document.getElementById(id);
    if (host) host.innerHTML = '';
  });
}

function closeAllDemandRowMenus() {
  document.querySelectorAll('.dw-action-menu').forEach((m) => {
    m.hidden = true;
    m.classList.remove('is-ported');
    m.style.left = '';
    m.style.top = '';
    m.style.right = '';
    m.style.bottom = '';
    m.style.position = '';
    m.style.transform = '';
    m.style.zIndex = '';
    if (m._home && m.parentElement !== m._home) m._home.appendChild(m);
  });
}
function placeDemandRowMenu(btn, menu) {
  const r = btn.getBoundingClientRect();
  const mw = Math.max(menu.offsetWidth || 0, 108);
  const mh = menu.offsetHeight || 72;
  const pad = 8;
  let left = r.right - mw;
  let top = r.bottom + 4;
  if (top + mh > window.innerHeight - pad) top = Math.max(pad, r.top - mh - 4);
  if (left < pad) left = pad;
  if (left + mw > window.innerWidth - pad) left = Math.max(pad, window.innerWidth - mw - pad);
  menu.style.position = 'fixed';
  menu.style.left = left + 'px';
  menu.style.top = top + 'px';
  menu.style.right = 'auto';
  menu.style.bottom = 'auto';
  menu.style.transform = 'none';
  menu.style.zIndex = '4000';
}
function toggleDemandRowMenu(btn) {
  const more = btn && btn.closest('.dw-action-more');
  const menu = more && more.querySelector('.dw-action-menu');
  if (!menu) return;
  const willOpen = menu.hidden;
  closeAllDemandRowMenus();
  if (!willOpen) return;
  menu._home = more;
  document.body.appendChild(menu);
  menu.classList.add('is-ported');
  menu.hidden = false;
  placeDemandRowMenu(btn, menu);
}
document.addEventListener('click', (e) => {
  if (!e.target.closest) return;
  if (e.target.closest('.dw-action-more') || e.target.closest('.dw-action-menu')) return;
  closeAllDemandRowMenus();
});
document.addEventListener('scroll', () => {
  if (document.querySelector('.dw-action-menu.is-ported')) closeAllDemandRowMenus();
}, true);
window.addEventListener('resize', closeAllDemandRowMenus);

// ═══ PRODUCT WORKSPACE ═══
