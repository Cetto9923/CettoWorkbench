// =============================================================================
// 文件: web/static/js/po/home-team-drawer.js
// 模块: PO工作台 - 团队视角
// 职责: 负责晨会作战抽屉（周报审阅归档、项目延期明细、人员负荷明细）交互与展现
// =============================================================================
(function () {
  "use strict";

  var esc = window.escapeHtml || function (value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, function (char) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char];
    });
  };

  function showToast(msg) {
    if (typeof document === "undefined" || !document.createElement) return;
    var toast = document.createElement("div");
    toast.className = "team-toast";
    toast.textContent = msg;
    document.body.appendChild(toast);
    setTimeout(function () { toast.remove(); }, 2500);
  }

  function getDrawer() {
    return document.getElementById("teamDrawer");
  }

  function getOverlay() {
    return document.getElementById("teamDrawerOverlay");
  }

  function closeDrawer() {
    var drawer = getDrawer();
    var overlay = getOverlay();
    if (drawer) drawer.classList.remove("show");
    if (overlay) overlay.classList.remove("show");
  }

  function openWeeklyDrawer(project) {
    var drawer = getDrawer();
    var overlay = getOverlay();
    if (!drawer || !overlay) return;
    drawer.innerHTML = '<div class="team-drawer-head">' +
      '<span class="team-drawer-title">项目周报审阅与归档 · ' + esc(project) + ' <span class="wb-phase2-tag">二期</span></span>' +
      '<button type="button" class="team-drawer-close" data-action="closeDrawer">✕</button>' +
    '</div>' +
    '<div class="team-drawer-body">' +
      '<div class="chip orange">待团队长审阅归档</div>' +
      '<div class="team-drawer-kv"><span>承建项目</span><strong>' + esc(project) + '</strong></div>' +
      '<div class="team-drawer-kv"><span>汇报周期</span><strong>2026年10月第2周</strong></div>' +
      '<div class="team-drawer-kv"><span>填报人</span><strong>张航 (PM)</strong></div>' +
      '<div class="team-drawer-kv"><span>提交时间</span><strong>2026-10-09 17:30</strong></div>' +
      '<div class="team-drawer-section">' +
        '<h4>本周进展与里程碑达成</h4>' +
        '<div class="team-drawer-desc">核心审批链路及风控改造已完成 85%，前端页面联调基本就绪，准备进入全量压测。</div>' +
      '</div>' +
      '<div class="team-drawer-section">' +
        '<h4>风险与资源瓶颈披露</h4>' +
        '<div class="team-drawer-desc">核心开发人员承担跨项目紧急支持任务，联调环境数据库表锁偶发超时，提请团队长协调联调排期。</div>' +
      '</div>' +
      '<div class="team-drawer-section">' +
        '<h4>下周工作重点</h4>' +
        '<div class="team-drawer-desc">完成性能基线测试，锁定 10-25 试运行投产窗口并组织发布演练。</div>' +
      '</div>' +
    '</div>' +
    '<div class="team-drawer-foot">' +
      '<button type="button" class="btn small" data-action="closeDrawer">取消</button>' +
      '<button type="button" class="btn small primary" data-action="confirmWeeklyArchive">确认周报并归档</button>' +
    '</div>';
    drawer.classList.add("show");
    overlay.classList.add("show");
  }

  function openProjectDrawer(project) {
    var drawer = getDrawer();
    var overlay = getOverlay();
    if (!drawer || !overlay) return;
    drawer.innerHTML = '<div class="team-drawer-head">' +
      '<span class="team-drawer-title">延期根因明细 · ' + esc(project) + ' <span class="wb-phase2-tag">二期</span></span>' +
      '<button type="button" class="team-drawer-close" data-action="closeDrawer">✕</button>' +
    '</div>' +
    '<div class="team-drawer-body">' +
      '<div class="chip red">延期高风险</div>' +
      '<div class="team-drawer-kv"><span>关联项目</span><strong>' + esc(project) + '</strong></div>' +
      '<div class="team-drawer-kv"><span>责任人</span><strong>付昊 (后端)</strong></div>' +
      '<div class="team-drawer-kv"><span>计划截止</span><strong>2026-10-08 (逾期 2 天)</strong></div>' +
      '<div class="team-drawer-kv"><span>计划发版</span><strong>2026-10-18 (窗口临近)</strong></div>' +
      '<div class="team-drawer-section">' +
        '<h4>延期根因分析</h4>' +
        '<div class="team-drawer-desc">对公二代支付批量对账模块联调环境数据库表锁导致集成阻塞，接口死锁频发，已申请 DBA 协助排查隔离。</div>' +
      '</div>' +
      '<div class="team-drawer-section">' +
        '<h4>影响交付范围</h4>' +
        '<div class="team-drawer-desc">涉及支付重构子模块 2 个在研任务，需在 10-12 前完成修复，否则需申请顺延发版窗口。</div>' +
      '</div>' +
    '</div>' +
    '<div class="team-drawer-foot">' +
      '<button type="button" class="btn small" data-action="closeDrawer">关闭</button>' +
      '<a href="/follow" class="btn small primary">进入周报中心追溯</a>' +
    '</div>';
    drawer.classList.add("show");
    overlay.classList.add("show");
  }

  function openCoordDrawer(project) {
    var drawer = getDrawer();
    var overlay = getOverlay();
    if (!drawer || !overlay) return;
    drawer.innerHTML = '<div class="team-drawer-head">' +
      '<span class="team-drawer-title">跨组协同阻塞 · ' + esc(project) + ' <span class="wb-phase2-tag">二期</span></span>' +
      '<button type="button" class="team-drawer-close" data-action="closeDrawer">✕</button>' +
    '</div>' +
    '<div class="team-drawer-body">' +
      '<div class="chip orange">外部依赖阻塞</div>' +
      '<div class="team-drawer-kv"><span>受阻项目</span><strong>' + esc(project) + '</strong></div>' +
      '<div class="team-drawer-kv"><span>项目责任人</span><strong>陈诺 (PM)</strong></div>' +
      '<div class="team-drawer-kv"><span>依赖方团队</span><strong>数仓平台研发组</strong></div>' +
      '<div class="team-drawer-section">' +
        '<h4>阻塞事项说明</h4>' +
        '<div class="team-drawer-desc">数仓平台客户画像聚合表接口定义尚未基线化，联调任务被动挂起，需两组主管协调统一接口协议。</div>' +
      '</div>' +
    '</div>' +
    '<div class="team-drawer-foot">' +
      '<button type="button" class="btn small" data-action="closeDrawer">关闭</button>' +
      '<button type="button" class="btn small primary" data-action="confirmCoordAction">记录协调待办</button>' +
    '</div>';
    drawer.classList.add("show");
    overlay.classList.add("show");
  }

  function openMemberDrawer(name) {
    var drawer = getDrawer();
    var overlay = getOverlay();
    if (!drawer || !overlay) return;
    var memberData = {
      "王睿": { role: "后端研发", load: "115%", level: "red", status: "超负荷冲突", proj: 4, tasks: 9, late: 2, desc: "承担营销活动引擎、手机银行等4个重点项目，核心任务并发集中，存在单点延期风险。" },
      "张航": { role: "后端研发", load: "85%", level: "orange", status: "重载关注", proj: 3, tasks: 8, late: 2, desc: "承担普惠金融与核心接口改造任务，负荷处于高位警戒线，需关注任务闭环进度。" },
      "赵宁": { role: "测试骨干", load: "75%", level: "green", status: "平稳", proj: 3, tasks: 6, late: 1, desc: "测试任务分配均衡，整体推进正常。" },
      "李晴": { role: "前端研发", load: "55%", level: "green", status: "平稳", proj: 2, tasks: 5, late: 0, desc: "主要承接企业手机银行前端迭代，工时利用率合理。" },
      "孙悦": { role: "需求分析", load: "待补工时", level: "grey", status: "需补计划", proj: 2, tasks: 4, late: 0, desc: "需求澄清与评审任务进行中，本周可用工时与排期需在敏捷小组补全。" }
    };
    var m = memberData[name] || { role: "研发骨干", load: "80%", level: "green", status: "平稳", proj: 2, tasks: 5, late: 0, desc: "承接任务正常推进中。" };
    drawer.innerHTML = '<div class="team-drawer-head">' +
      '<span class="team-drawer-title">成员作战与负荷明细 · ' + esc(name) + ' <span class="wb-phase2-tag">二期</span></span>' +
      '<button type="button" class="team-drawer-close" data-action="closeDrawer">✕</button>' +
    '</div>' +
    '<div class="team-drawer-body">' +
      '<div class="chip ' + m.level + '">' + m.status + '</div>' +
      '<div class="team-drawer-kv"><span>成员姓名</span><strong>' + esc(name) + '</strong></div>' +
      '<div class="team-drawer-kv"><span>岗位角色</span><strong>' + m.role + '</strong></div>' +
      '<div class="team-drawer-kv"><span>计划负荷率</span><strong>' + m.load + '</strong></div>' +
      '<div class="team-drawer-kv"><span>承接在建项目</span><strong>' + m.proj + ' 个项目</strong></div>' +
      '<div class="team-drawer-kv"><span>在研任务总数</span><strong>' + m.tasks + ' 项 (逾期 ' + m.late + ' 项)</strong></div>' +
      '<div class="team-drawer-section">' +
        '<h4>负荷与风险评估</h4>' +
        '<div class="team-drawer-desc">' + m.desc + '</div>' +
      '</div>' +
    '</div>' +
    '<div class="team-drawer-foot">' +
      '<button type="button" class="btn small" data-action="closeDrawer">关闭</button>' +
    '</div>';
    drawer.classList.add("show");
    overlay.classList.add("show");
  }

  if (typeof document !== "undefined" && typeof document.addEventListener === "function") {
    document.addEventListener("click", function (e) {
      var target = e.target;
      if (!target || typeof target.closest !== "function") return;

      if (target.closest('[data-action="closeDrawer"]') || target.closest("#teamDrawerOverlay")) {
        closeDrawer();
        return;
      }
      if (target.closest('[data-action="confirmWeeklyArchive"]')) {
        closeDrawer();
        showToast("【二期规划】周报归档接口尚未对接后端服务，未执行归档操作");
        return;
      }
      if (target.closest('[data-action="confirmCoordAction"]')) {
        closeDrawer();
        showToast("【二期规划】协同待办接口尚未对接后端服务，未执行登记操作");
        return;
      }

      var reviewWeekly = target.closest('[data-action="reviewWeekly"]');
      if (reviewWeekly) {
        var project = reviewWeekly.getAttribute("data-target") || "普惠金融二期";
        openWeeklyDrawer(project);
        return;
      }

      var projDrawer = target.closest('[data-action="openProjectDrawer"]');
      if (projDrawer) {
        var pName = projDrawer.getAttribute("data-target") || "对公二代支付";
        openProjectDrawer(pName);
        return;
      }

      var coord = target.closest('[data-action="coordSupport"]');
      if (coord) {
        var cName = coord.getAttribute("data-target") || "客户画像数据治理";
        openCoordDrawer(cName);
        return;
      }

      var memberDrawer = target.closest('[data-action="openMemberDrawer"]');
      if (memberDrawer) {
        var name = memberDrawer.getAttribute("data-name") || "团队成员";
        openMemberDrawer(name);
        return;
      }
    });

    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") closeDrawer();
    });
  }
})();
