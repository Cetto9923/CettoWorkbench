// =============================================================================
// 文件: web/static/js/po/demand-detail-labels.js
// 模块: PO 工作台
// 职责: 需求详情页禅道原始码 → 中文的对照表（研发需求 status / 审计动作 action）。
//       真源：禅道 lang 文件，页面只读不改；查不到的一律回显原文，不丢数据。
//       由 demand_detail.html 在 demand-detail-render.js 之前加载。
// =============================================================================

(function (root, factory) {
  if (typeof define === "function" && define.amd) {
    define([], factory);
  } else if (typeof module === "object" && module.exports) {
    module.exports = factory();
  } else {
    root.DemandDetailLabels = factory();
  }
})(typeof self !== "undefined" ? self : this, function () {
  "use strict";

  // 研发需求状态：禅道 module/story/lang/zh-cn.php 的 $lang->story->statusList
  // 与 $lang->story->stageList 合并（status 在前，stage 补其余，closed 两处同值）。
  var STORY_STATUS = {
    draft: "草稿",
    reviewing: "评审中",
    active: "激活",
    changing: "变更中",
    closed: "已关闭",
    wait: "未开始",
    planned: "已计划",
    projected: "研发立项",
    designing: "设计中",
    designed: "设计完毕",
    developing: "研发中",
    developed: "研发完毕",
    testing: "测试中",
    tested: "测试完毕",
    verified: "已验收",
    rejected: "验收失败",
    delivering: "交付中",
    delivered: "已交付",
    released: "已发布",
  };

  // 禅道操作动作：zentao/extension/custom/action/ext/lang/zh-cn/changshu.php 与 repo.php
  // 的 $lang->action->label->* 优先，其次禅道通用 module/action/lang/zh-cn.php，
  // 两者皆无时退回同文件 $lang->action->dynamicAction->demand->* 的动作名。
  var ACTION_LABEL = {
    acceptance: "验收了",
    acceptanced: "验收了",
    activated: "激活了",
    activatedbychild: "激活了",
    addmember: "添加了团队成员",
    adminadjust: "调整了",
    appraise: "待您进行满意度评价",
    approvalnodepass: "审批通过了",
    assigned: "指派了",
    autotoclosed: "自动更新为已关闭状态",
    autotoreleased: "自动更新为已发布状态",
    braChanged: "需求负责人由【%s】变更为【%s】，请及时处理。",
    canceled: "取消了",
    changed: "变更了",
    changedelivery: "修改交付信息",
    changereason: "变更原因",
    clarify: "澄清了",
    closed: "关闭了",
    closedbychild: "关闭了",
    commented: "评论了",
    createbaselinerelease: "调整基线，联动创建了",
    createchildrendemand: "创建子需求",
    created: "创建了",
    createdfromoa: "创建并评审通过",
    deleted: "删除了",
    deletedfile: "删除了附件",
    deleteestimate: "删除预估时间",
    deletesumbit: "删除了交付物",
    deletetesttask: "删除测试单",
    delist: "下架了",
    deliver: "发起了交付",
    demand: "需求池需求|demand|view|id=%s",
    disbanded: "解散了",
    dispatch: "分派了",
    editbaselinerelease: "调整基线，联动编辑了",
    edited: "编辑了",
    editedbybaseline: "调整了",
    editedproduct: "更新了",
    editestimate: "编辑了工时",
    editfile: "编辑了附件",
    enabled: "启用了",
    finished: "完成了",
    fromdemand: "由需求池需求创建",
    fromfeedback: "来自反馈",
    hangup: "挂起了",
    hidden: "隐藏了",
    importeddemand: "导入了",
    labelreview: "专题评审",
    linkbugapp: "创建了分支从",
    linkobjective: "关联目标到了",
    linkplanapp: "关联了应用从",
    linkreleaseapp: "关联了应用从",
    linkrequirement: "关联用户需求到了",
    linkstory: "关联研发需求到了",
    linkstoryapp: "创建了分支从",
    nofilled: "未填报工时",
    noticeto: "备注并通知",
    notifymanager: "通知相关部门验收",
    opened: "创建了",
    paused: "暂停了",
    picard: "PI卡片|pi|plan|kanbanid=%s",
    plan: "计划了",
    publish: "发布了",
    qualitycheck: "质量检查豁免",
    qualityscanfeedback: "更新了修复进度",
    reboot: "重启了",
    recalled: "撤销了评审",
    recalledchange: "撤销了变更",
    recordestimate: "记录了工时",
    releasedbyallstories: "发布了",
    releasedbyticket: "发布转化工单",
    reminded: "提醒执行将在延期后自动关闭。",
    removemember: "移除了团队成员",
    reviewbymanager: "主管部门审批",
    reviewchange: "评审了变更",
    reviewed: "评审了",
    reviewpassed: "确认通过",
    reviewrejected: "拒绝了",
    startacceptance: "提交验收",
    submitbymanager: "发起了系统主管部门审批",
    submited: "发起了审批",
    suffixplan: "计划",
    testtaskblocked: "测试任务被阻止",
    toReview: "待您进行评审",
    todemand: "转业务需求",
    tostory: "转研发需求",
    toticket: "转工单",
    unbind: "解绑了",
    unlinkbugbranch: "移除了分支从",
    unlinkplanapp: "移除了应用从",
    unlinkreleaseapp: "移除了应用从",
    unlinkrequirement: "移除了用户需求从",
    unlinkstory: "移除了研发需求从",
    unlinkstorybranch: "移除了分支从",
    updateclarified: "更新已澄清",
    updatedeveloping: "变更为研发中",
    updateinfo: "架构评审",
    updatetesting: "变更为测试中",
    upload: "上传了",
    uploadcustom: "上传了自定义内容",
    withdrawacceptance: "撤销验收",
    withdrawbymanager: "撤回主管审批",
  };

  function pick(table, value) {
    var raw = String(value == null ? "" : value).trim();
    if (!raw) return "—";
    return Object.prototype.hasOwnProperty.call(table, raw) ? table[raw] : raw;
  }

  return {
    storyStatusLabel: function (value) { return pick(STORY_STATUS, value); },
    actionLabel: function (value) { return pick(ACTION_LABEL, value); },
    STORY_STATUS: STORY_STATUS,
    ACTION_LABEL: ACTION_LABEL
  };
});
