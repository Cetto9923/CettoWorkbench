(function ($) {
  "use strict";

  var DEFAULT_VISIBLE = 5;

  var TOP5_ITEMS = [
    { id: "REQ-24018", pri: "P1", title: "核心交易链路对账差异修复", stage: "提测", blocker: "功能待开发，敬请期待...", next: "协调联调环境", owner: "张三", alert: true },
    { id: "REQ-23986", pri: "P1", title: "会员积分批量补发与核对", stage: "验收", blocker: "业务确认超时", next: "催业务确认", owner: "李四", alert: true },
    { id: "REQ-23952", pri: "P1", title: "订单导出性能优化（百万级）", stage: "排期", blocker: "排期未锁定", next: "锁定版本窗口", owner: "王五", alert: true },
    { id: "REQ-23901", pri: "P2", title: "渠道结算报表字段扩展", stage: "联调测试", blocker: "测试阻塞", next: "推动缺陷修复", owner: "赵六" },
    { id: "REQ-23877", pri: "P2", title: "大额转账审批流程优化", stage: "澄清", blocker: "需求口径未统一", next: "组织澄清会", owner: "张明远" },
    { id: "REQ-23840", pri: "P2", title: "移动端人脸识别登录", stage: "受理", blocker: "待业务补材料", next: "跟进材料提交", owner: "李工" },
    { id: "REQ-23812", pri: "P3", title: "统一认证 OAuth2.0 升级", stage: "排期", blocker: "依赖版本未确认", next: "确认依赖版本", owner: "周工" },
    { id: "REQ-23795", pri: "P3", title: "信用卡账单分期 V2.0", stage: "交付", blocker: "发布窗口待定", next: "确认发布窗口", owner: "王经理" },
    { id: "REQ-23760", pri: "P3", title: "对公账户批量转账优化", stage: "评价反馈", blocker: "反馈收集未完成", next: "发起满意度回访", owner: "陈敏" },
    { id: "REQ-23721", pri: "P4", title: "电子票据签发流程重构", stage: "受理", blocker: "方案评审中", next: "完成方案评审", owner: "刘洋" },
    { id: "REQ-23688", pri: "P4", title: "外汇兑换汇率实时展示", stage: "澄清", blocker: "外部接口待评估", next: "完成接口评估", owner: "孙婷" },
    { id: "REQ-23650", pri: "P2", title: "开放银行 API 接口升级", stage: "验收", blocker: "验收用例未齐", next: "补齐验收用例", owner: "马超" },
    { id: "REQ-23611", pri: "P3", title: "转账结果页文案优化", stage: "评价反馈", blocker: "待客户确认", next: "发送确认邮件", owner: "吴倩" },
    { id: "REQ-23580", pri: "P4", title: "历史遗留接口兼容适配", stage: "排期", blocker: "挂起超 45 天", next: "评估是否解除挂起", owner: "郑凯" },
    { id: "REQ-23542", pri: "P2", title: "企业网银限额白名单配置", stage: "提测", blocker: "配置未下发", next: "下发测试配置", owner: "钱峰" },
  ];

  function escapeHtml(text) {
    return $("<div>").text(text).html();
  }

  function buildRowHtml(item) {
    var rowClass = item.alert ? "js-top5-row top5-row--alert" : "js-top5-row";
    return (
      "<tr class=\"" + rowClass + "\">" +
      "<td class=\"table-id col-id\">" + escapeHtml(item.id) + "</td>" +
      "<td class=\"text-strong cell-ellipsis\"><span class=\"priority-tag " + escapeHtml(item.pri) + "\">" +
      escapeHtml(item.pri) + "</span>" + escapeHtml(item.title) + "</td>" +
      "<td><span class=\"table-tag\">" + escapeHtml(item.stage) + "</span></td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.blocker) + "</td>" +
      "<td class=\"text-muted\">" + escapeHtml(item.next) + "</td>" +
      "<td>" + escapeHtml(item.owner) + "</td>" +
      "<td class=\"col-action\"><button type=\"button\" class=\"table-row-btn\">处理</button></td>" +
      "</tr>"
    );
  }

  function renderTop5Rows() {
    var $tbody = $("#top5TableBody");
    if (!$tbody.length) {
      return $();
    }

    var html = $.map(TOP5_ITEMS, buildRowHtml).join("");
    $tbody.html(html);
    return $tbody.find(".js-top5-row");
  }

  function initTop5Toggle($rows) {
    var $wrap = $("#top5ToggleWrap");
    var $btn = $("#top5Toggle");
    if (!$wrap.length || !$btn.length || !$rows.length) {
      return;
    }

    if ($rows.length <= DEFAULT_VISIBLE) {
      $wrap.prop("hidden", true);
      return;
    }

    $wrap.prop("hidden", false);
    var expanded = false;

    function apply() {
      $rows.each(function (index) {
        $(this).toggleClass("is-row-hidden", !expanded && index >= DEFAULT_VISIBLE);
      });
      $btn
        .attr("aria-expanded", expanded ? "true" : "false")
        .text(expanded ? "收起 ↑" : "查看全部 " + $rows.length + " 条 →");
    }

    $btn.on("click", function () {
      expanded = !expanded;
      apply();
    });

    apply();
  }

  $(function () {
    initTop5Toggle(renderTop5Rows());
  });
})(jQuery);
