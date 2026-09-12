/**
 * 敏捷小组详情 / 确认 Drawer
 */
(function () {
  function S() { return window.__at || {}; }
  function esc(s) { return S().esc(s); }
  function toast(m) { return S().toast(m); }
  function val(id) { return S().val(id); }
  function apiFetch(p, o) { return S().apiFetch(p, o); }
  function isLeadView() { return S().isLeadView(); }
  function person(n, a) { return S().person(n, a); }
  function st() { return S().state || {}; }

  window.atLoadDetail = async function(id) {
    const host = document.getElementById("atDetailRoot");
    if (!host) return;
    host.innerHTML = '<div class="at-empty">加载中…</div>';
    try {
      const json = await apiFetch(S().API + "/" + id + (isLeadView() ? "?view=lead" : ""));
      const d = (json && json.data) || {};
      st().canConfirm = !!d.canConfirm;
      st().canEdit = !!d.canEdit && !isLeadView();
      st().lastDetail = d;
      renderDetail(host, d);
    } catch (e) {
      host.innerHTML = '<div class="at-empty">' + esc(e.message || "加载失败") + "</div>";
    }
  }

  function renderDetail(host, d) {
    const logoChar = (d.name || "?").slice(0, 1);
    const canEdit = !!d.canEdit && !isLeadView();
    const saveBtn = canEdit
      ? '<button type="button" class="at-btn primary" onclick="atSaveBasic(' + d.id + ')">保存基本信息</button>'
      : "";
    const banner = isLeadView()
      ? '<div class="at-readonly-banner">当前为团队管理视图：团队长 / 部室负责人仅查看，不提供敏捷小组资料、成员和角色编辑权限。</div>'
      : "";
    host.innerHTML =
      '<button type="button" class="at-back" onclick="atShowList()"><i class="fas fa-arrow-left"></i> 返回列表</button>' +
      '<div class="at-detail-card">' +
      '<div class="at-detail-header">' +
      '<div class="at-logo">' + esc(logoChar) + "</div>" +
      '<div class="at-detail-title"><h3>' + esc(d.name) + "</h3>" +
      '<div class="at-meta"><span>父级小组：' + esc(d.parentName || "—") +
      "</span><span>教练：" + esc(person(d.coachName, d.coachAccount)) +
      "</span><span>PO：" + esc(person(d.poName, d.poAccount)) +
      '</span><span class="at-tag enabled">' + esc(d.statusLabel || "启用") + "</span></div></div>" +
      '<div class="at-page-actions">' + saveBtn + "</div>" +
      "</div>" +
      banner +
      '<div class="at-tabs">' +
      tabBtn("basic", "基本信息") + tabBtn("members", "成员管理") + tabBtn("history", "调整记录") +
      "</div>" +
      '<div class="at-tab-panel' + (st().detailTab === "basic" ? " active" : "") + '" id="atTab-basic">' + renderBasic(d, canEdit) + "</div>" +
      '<div class="at-tab-panel' + (st().detailTab === "members" ? " active" : "") + '" id="atTab-members">' + renderMembers(d, canEdit) + "</div>" +
      '<div class="at-tab-panel' + (st().detailTab === "history" ? " active" : "") + '" id="atTab-history">' + renderHistory(d) + "</div>" +
      "</div>";
  }

  function tabBtn(key, label) {
    return (
      '<button type="button" class="at-tab' + (st().detailTab === key ? " active" : "") +
      '" onclick="atSwitchTab(\'' + key + "')\">" + label + "</button>"
    );
  }

  window.atSwitchTab = function (key) {
    st().detailTab = key;
    document.querySelectorAll(".at-tab").forEach(function (b) {
      b.classList.toggle("active", b.textContent.indexOf(key === "basic" ? "基本" : key === "members" ? "成员" : "调整") >= 0);
    });
    // simpler: re-toggle panels
    ["basic", "members", "history"].forEach(function (k) {
      const p = document.getElementById("atTab-" + k);
      if (p) p.classList.toggle("active", k === key);
    });
    document.querySelectorAll(".at-tabs .at-tab").forEach(function (b, i) {
      const keys = ["basic", "members", "history"];
      b.classList.toggle("active", keys[i] === key);
    });
  };

  function renderBasic(d, canEdit) {
    const ro = canEdit ? "" : " readonly disabled";
    const parentField = canEdit
      ? '<select class="input" id="atBasicParent">' +
        '<option value="0"' + (!d.parentId ? " selected" : "") + ">无（作为父级小组）</option>" +
        (d.parentOptions || []).map(function (o) {
          return '<option value="' + esc(o.id) + '"' + (Number(d.parentId) === Number(o.id) ? " selected" : "") + ">" + esc(o.name) + "</option>";
        }).join("") +
        "</select>"
      : "<div>" + esc(d.parentName || "—") + "</div>";
    return (
      '<div class="at-info-grid">' +
      '<div class="label">父级小组</div>' + parentField +
      '<div class="label">状态</div><div>' + esc(d.statusLabel || "启用") + "</div>" +
      '<div class="label">团队名称</div><input class="input" id="atBasicName" value="' + esc(d.name || "") + '"' + ro + " />" +
      '<div class="label">创建日期</div><div>' + esc(d.createdDate || "—") + "</div>" +
      '<div class="label">敏捷教练</div><div>' + esc(person(d.coachName, d.coachAccount)) + "</div>" +
      '<div class="label">产品负责人</div><div>' + esc(person(d.poName, d.poAccount)) + "</div>" +
      '<div class="label">团队口号</div><input class="input span3" id="atBasicSlogan" value="' + esc(d.slogan || "") + '"' + ro + " />" +
      '<div class="label">团队信条</div><textarea class="textarea span3" id="atBasicDecl"' + ro + ">" + esc(d.declaration || "") + "</textarea>" +
      '<div class="label">团队 Logo</div><textarea class="textarea span3" id="atBasicLogo" placeholder="Logo 文本/URL"' + ro + ">" + esc(d.logo || "") + "</textarea>" +
      "</div>"
    );
  }

  function renderMembers(d, canEdit) {
    let html = "";
    if (d.pending) {
      const p = d.pending;
      html +=
        '<div class="at-pending-box"><div class="at-pending-title"><span class="at-tag pending">待确认</span> 成员调整 ' +
        esc(p.adjustNo) + "</div>" +
        '<div style="font-size:11px;color:#7f735f;margin-bottom:8px">发起人 ' + esc(p.submittedBy) +
        " · " + esc(p.submittedAt) + (p.reason ? " · " + esc(p.reason) : "") + "</div>" +
        '<div style="font-size:12px">新增 ' + esc(p.addCount) + " · 移除 " + esc(p.removeCount) +
        " · 角色调整 " + esc(p.changeCount) + "</div>" +
        '<div style="margin-top:10px"><button type="button" class="at-btn small primary" onclick="atOpenReview(' +
        p.adjustmentId + ')">查看调整</button></div></div>';
    }
    const editBtn = canEdit
      ? '<div class="right"><span class="at-section-note">正式生效仍需组织级敏捷教练确认</span>' +
        '<button type="button" class="at-btn small primary" onclick="atOpenMemberEdit(' + d.id + ')">编辑成员与角色</button></div>'
      : "";
    html +=
      '<div class="at-section-head"><h4>正式成员</h4><span class="at-section-note">仅统计已确认正式生效的成员 · ' +
      esc(d.formalCount || 0) + " 人</span>" + editBtn + "</div>" +
      memberTable(d.formal || [], false);
    html +=
      '<div class="at-section-head"><h4>待加入成员</h4><span class="at-section-note">已可进入任务看板参与协作，但不计入正式成员数</span></div>' +
      memberTable(d.pendingJoin || [], true);
    return html;
  }

  function memberTable(rows, pendingJoin) {
    if (!rows.length) return '<div class="at-empty">暂无</div>';
    const head = pendingJoin
      ? "<tr><th>成员</th><th>拟任角色</th><th>可用工时/天</th><th>发起人</th><th>状态</th></tr>"
      : "<tr><th>成员</th><th>角色</th><th>可用工时/天</th><th>加入日期</th><th>当前状态</th></tr>";
    const body = rows.map(function (m) {
      const st =
        m.status === "pendingRemove"
          ? '<span class="at-tag pending">待移除</span>'
          : m.status === "pendingAdd"
            ? '<span class="at-tag pending">待确认</span>'
            : '<span class="at-tag enabled">正式</span>';
      const cls = m.status === "pendingAdd" || m.status === "pendingRemove" ? ' class="at-pending-member"' : "";
      if (pendingJoin) {
        return (
          "<tr" + cls + "><td>" + esc(person(m.name, m.account)) + "</td><td>" + esc(m.role || "—") +
          "</td><td>" + esc(m.hours) + "</td><td>" + esc(m.submitter || "—") + "</td><td>" + st + "</td></tr>"
        );
      }
      return (
        "<tr" + cls + "><td>" + esc(person(m.name, m.account)) + "</td><td>" + esc(m.role || "—") +
        "</td><td>" + esc(m.hours) + "</td><td>" + esc(m.joinDate || "—") + "</td><td>" + st + "</td></tr>"
      );
    }).join("");
    return '<div class="at-member-table"><table class="at-table"><thead>' + head + "</thead><tbody>" + body + "</tbody></table></div>";
  }

  function renderHistory(d) {
    const rows = d.history || [];
    if (!rows.length) return '<div class="at-empty">暂无调整记录</div>';
    return (
      '<div class="at-timeline">' +
      rows.map(function (h) {
        return (
          '<div class="at-event"><div class="at-event-title">' + esc(h.summary) +
          '</div><div class="at-event-meta">' + esc(h.createdAt) + " · " + esc(h.actorName || h.actor) + "</div></div>"
        );
      }).join("") +
      "</div>"
    );
  }

  window.atSaveBasic = async function (id) {
    if (isLeadView()) return;
    const parentEl = document.getElementById("atBasicParent");
    const body = {
      name: val("atBasicName"),
      slogan: val("atBasicSlogan"),
      declaration: (document.getElementById("atBasicDecl") || {}).value || "",
      logo: (document.getElementById("atBasicLogo") || {}).value || "",
    };
    if (parentEl) body.parentId = Number(parentEl.value || 0);
    try {
      await apiFetch(S().API + "/" + id + "/basic", { method: "PUT", body: body });
      toast("基本信息已保存");
      window.atLoadDetail(id);
    } catch (e) {
      toast(e.message || "保存失败");
    }
  };

  window.atOpenReview = async function (adjustmentId) {
    const mask = document.getElementById("atReviewMask");
    const body = document.getElementById("atReviewBody");
    const foot = document.getElementById("atReviewFoot");
    if (!mask || !body) return;
    body.innerHTML = '<div class="at-empty">加载中…</div>';
    foot.innerHTML = "";
    mask.classList.add("open");
    try {
      const json = await apiFetch(S().API + "/adjustments/" + adjustmentId + (isLeadView() ? "?view=lead" : ""));
      const d = (json && json.data) || {};
      renderReview(body, foot, d);
    } catch (e) {
      body.innerHTML = '<div class="at-empty">' + esc(e.message || "加载失败") + "</div>";
    }
  };

  window.atCloseReview = function () {
    const mask = document.getElementById("atReviewMask");
    if (mask) mask.classList.remove("open");
  };

  function renderReview(body, foot, d) {
    const groups = { add: [], remove: [], roleChange: [] };
    (d.items || []).forEach(function (it) {
      if (groups[it.actionType]) groups[it.actionType].push(it);
    });
    body.innerHTML =
      '<div style="font-size:12px;margin-bottom:12px">' +
      "<div><b>" + esc(d.teamName) + "</b> · " + esc(d.adjustNo) + "</div>" +
      "<div style=\"color:#8894a6;margin-top:4px\">发起人 " + esc(d.submittedBy) + " · " + esc(d.submittedAt) + "</div>" +
      (d.reason ? '<div style="margin-top:8px">说明：' + esc(d.reason) + "</div>" : "") +
      "</div>" +
      diffGroup("新增人员", groups.add, "add") +
      diffGroup("移除人员", groups.remove, "remove") +
      diffGroup("角色 / 工时变化", groups.roleChange, "change");

    let actions = '<button type="button" class="at-btn" onclick="atCloseReview()">关闭</button>';
    if (d.canConfirm && d.status === "pending") {
      actions =
        '<button type="button" class="at-btn danger" onclick="atReject(' + d.id + ')">驳回</button>' +
        '<button type="button" class="at-btn primary" onclick="atConfirm(' + d.id + ')">确认生效</button>';
    }
    foot.innerHTML = actions;
  }

  function diffGroup(title, items, kind) {
    if (!items.length) return "";
    return (
      '<div class="at-diff-group"><h4>' + title + "</h4>" +
      items.map(function (it) {
        let sub = esc(it.account);
        if (kind === "change") {
          sub = esc(it.prevRole || "—") + " → " + esc(it.role || "—") +
            " · 工时 " + esc(it.prevHours) + " → " + esc(it.availableHours);
        } else if (kind === "add") {
          sub = esc(it.role || "—") + " · " + esc(it.availableHours) + "h/天";
        }
        return (
          '<div class="at-person-card"><div class="at-person-name">' + esc(it.name) +
          '</div><div class="at-person-sub">' + sub + "</div></div>"
        );
      }).join("") +
      "</div>"
    );
  }

  window.atConfirm = async function (id) {
    try {
      await apiFetch(S().API + "/adjustments/" + id + "/confirm", { method: "PUT", body: {} });
      toast("成员调整已确认生效");
      atCloseReview();
      if (st().detailId) window.atLoadDetail(st().detailId);
      else S().loadList();
    } catch (e) {
      toast(e.message || "确认失败");
    }
  };

  window.atReject = async function (id) {
    const reason = window.prompt("请填写驳回原因");
    if (reason == null) return;
    if (!String(reason).trim()) {
      toast("驳回原因不能为空");
      return;
    }
    try {
      await apiFetch(S().API + "/adjustments/" + id + "/reject", { method: "PUT", body: { reason: String(reason).trim() } });
      toast("成员调整已驳回");
      atCloseReview();
      if (st().detailId) window.atLoadDetail(st().detailId);
      else S().loadList();
    } catch (e) {
      toast(e.message || "驳回失败");
    }
  };

})();
