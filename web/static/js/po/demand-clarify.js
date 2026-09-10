// PO 工作台 - 需求澄清交互逻辑 (对齐禅道校验与产品成员推荐)

(function () {
  "use strict";

  var currentDemandId = null, currentFormData = null, productRowIndex = 0, storyRowIndex = 0, isSubmitting = false;

  function esc(s) {
    return String(s == null ? "" : s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function showToast(msg, type) {
    if (typeof window.showToast === "function") window.showToast(msg, type); else alert(msg);
  }

  function handleClarifySuccess(msg) {
    closeModal();
    if (window.DemandDetail && typeof window.DemandDetail.close === "function") window.DemandDetail.close();
    showToast(msg || "需求澄清保存成功", "success");
    if (typeof window.refreshPoHomeDemands === "function") window.refreshPoHomeDemands();
    else setTimeout(function () { window.location.reload(); }, 600);
  }

  function openModal(demandId) {
    if (!demandId) return;
    currentDemandId = String(demandId).replace(/^US/i, "").trim();
    $("#poClarifyDemandBadge").text("US" + currentDemandId);
    $("#poDemandClarifyModal").addClass("show").css("display", "block").attr("aria-hidden", "false");
    $("#poClarifyLoadingState").show();
    $("#poClarifyErrorState, #poDemandClarifyForm").hide();
    loadClarifyData(currentDemandId);
  }

  function closeModal() {
    var $modal = $("#poDemandClarifyModal");
    $modal.removeClass("show").attr("aria-hidden", "true");
    setTimeout(function () { if (!$modal.hasClass("show")) $modal.css("display", "none"); }, 240);
    if (typeof window.destroyAutocomplete === "function") {
      $("#poClarifyProductTbody tr").each(function () {
        window.destroyAutocomplete("poClarifyProductInput_" + $(this).attr("data-row-idx"));
        window.destroyAutocomplete("poClarifyPmInput_" + $(this).attr("data-row-idx"));
      });
      ["poClarifyBRAInput", "poClarifyQDInput", "poClarifyRDInput"].forEach(window.destroyAutocomplete);
    }
    currentDemandId = null; currentFormData = null;
  }

  function findUserLabel(users, val) {
    if (!val) return "";
    for (var i = 0; i < (users || []).length; i++) { if (String(users[i].value) === String(val)) return users[i].label; }
    return val;
  }
  function findProduct(products, pId) {
    for (var i = 0; i < (products || []).length; i++) { if (String(products[i].id) === String(pId)) return products[i]; }
    return null;
  }
  function buildPmOptionsForProduct(pId) {
    var allUsers = (currentFormData && currentFormData.userOptions) || [];
    var members = (currentFormData && currentFormData.productMembers && currentFormData.productMembers[String(pId)]) || [];
    if (!members.length) return allUsers;
    var memberMap = {}, recItems = [], otherUsers = [];
    members.forEach(function (m) {
      memberMap[m.account] = true;
      var name = m.realname || m.account;
      recItems.push({ value: m.account, label: name, badge: m.role || "相关人员", selectedLabel: name + "(" + m.account + ")" });
    });
    allUsers.forEach(function (u) { if (!memberMap[u.value]) otherUsers.push(u); });
    var items = [{ label: "★ 本产品相关人员 (推荐)", value: "", isGroupHeader: true }].concat(recItems);
    if (otherUsers.length) items = items.concat([{ label: "全部人员", value: "", isGroupHeader: true }]).concat(otherUsers);
    return items;
  }
  function bindUserAutocomplete(inputId, hiddenId, users, selectedVal) {
    if (typeof window.initAutocomplete !== "function") return;
    window.initAutocomplete(inputId, hiddenId, users || [], { value: selectedVal || "", label: findUserLabel(users, selectedVal), placeholder: "输入姓名或工号搜索" });
  }
  function bindProductAutocomplete(inputId, hiddenId, products, selectedVal) {
    if (typeof window.initAutocomplete !== "function") return;
    var items = (products || []).map(function (p) { return { value: String(p.id), label: p.name || String(p.id) }; }), found = findProduct(products, selectedVal);
    window.initAutocomplete(inputId, hiddenId, items, { value: selectedVal || "", label: found ? (found.name || String(found.id)) : "", placeholder: "搜索产品名称或 ID" });
  }
  function loadClarifyData(id) {
    (window.appFetch || window.fetch)("/demands/" + encodeURIComponent(id) + "/clarify", { method: "GET", headers: { Accept: "application/json" } })
      .then(function (res) {
        if (!res.ok) return res.json().then(function (d) { throw new Error(d.message || "加载失败"); });
        return res.json();
      })
      .then(function (data) {
        currentFormData = data; renderForm(data);
        $("#poClarifyLoadingState").hide(); $("#poDemandClarifyForm").show();
      })
      .catch(function (err) {
        $("#poClarifyLoadingState").hide(); $("#poClarifyErrorMsg").text(err.message || "加载失败，请稍后重试"); $("#poClarifyErrorState").show();
      });
  }
  function normalizeLegalPersonLogo(val) {
    if (val === "changshu" || val === "0") return "0";
    if (val === "village" || val === "1") return "1";
    if (val === "changshu_village" || val === "2") return "2";
    return "0";
  }

  function renderForm(data) {
    $("#poClarifyDemandId").val(data.id);
    $("#poClarifyDemandBadge").text(data.code || ("US" + data.id));
    $("#poClarifyDemandTitle").text(data.name || "需求澄清");
    populateSelect("#poClarifyCategory", data.categoryOptions, data.category);
    bindUserAutocomplete("poClarifyBRAInput", "poClarifyBRA", data.userOptions, data.bra);
    bindUserAutocomplete("poClarifyQDInput", "poClarifyQD", data.userOptions, data.qd);
    bindUserAutocomplete("poClarifyRDInput", "poClarifyRD", data.userOptions, data.rd);
    $("#poClarifyDescContent").html(data.desc || "<em>暂无描述</em>").hide();
    $("#poClarifyDescPreview").text(data.desc ? data.desc.replace(/<[^>]+>/g, "").slice(0, 40) + "…" : "暂无描述");
    $("#poClarifyDescArrow").removeClass("expanded");
    renderQuickProductBar(data.frequentProducts || []);
    $("#poClarifyProductTbody").empty(); productRowIndex = 0;
    (data.products && data.products.length ? data.products : [{}]).forEach(addProductRow);
    $("#poClarifyStoryTbody").empty(); storyRowIndex = 0;
    (data.userStories && data.userStories.length ? data.userStories : [{}]).forEach(addStoryRow);
    $("#poClarifyDesc").val(data.clarifyDesc || "");
    ["isNewProduct", "isRelatedAccounts", "isNewFunction", "isOtherImportantOrder"].forEach(function (k) { setRadioValue(k, data[k] === "1" ? "1" : "0"); });
    setRadioValue("multiLegalPersonLogo", normalizeLegalPersonLogo(data.multiLegalPersonLogo));
    toggleFinanceAlert(); toggleCategoryRule(); updateTotalScale();
  }
  function renderQuickProductBar(frequentList) {
    var $bar = $("#poClarifyQuickBar");
    if (!frequentList || !frequentList.length) { $bar.hide().empty(); return; }
    var html = '<span class="clarify-quick-title">★ 常用产品快捷选择:</span>';
    frequentList.forEach(function (p) {
      var poText = p.po ? ' <span class="clarify-quick-chip-po">(' + esc(p.po) + ')</span>' : "";
      html += '<button type="button" class="clarify-quick-chip js-quick-prod-chip" data-prod-id="' + esc(p.id) + '" data-prod-name="' + esc(p.name) + '" data-prod-po="' + esc(p.po || "") + '" title="快速选择并添加产品行">' +
        '<span class="clarify-quick-chip-icon">★</span> ' + esc(p.name) + poText + '<span class="clarify-quick-chip-plus">+</span></button>';
    });
    $bar.html(html).show();
  }
  function populateSelect(selector, options, selectedVal) {
    var $sel = $(selector), firstOption = $sel.find("option:first").clone();
    $sel.empty().append(firstOption);
    (options || []).forEach(function (opt) {
      var val = String(opt.value != null ? opt.value : opt.id), lbl = String(opt.label != null ? opt.label : opt.name);
      $sel.append('<option value="' + esc(val) + '"' + (String(selectedVal) === val ? " selected" : "") + ">" + esc(lbl) + "</option>");
    });
  }
  function setRadioValue(name, val) { $('input[name="' + name + '"][value="' + val + '"]').prop("checked", true); }
  function getRadioValue(name) { return $('input[name="' + name + '"]:checked').val() || "0"; }

  function addProductRow(item) {
    item = item || {};
    var idx = productRowIndex++, products = (currentFormData && currentFormData.productOptions) || [];
    var isMain = item.isMainSystem ? " checked" : "", isAdd = item.isAdditionalInfo === "1" ? " checked" : "";
    var reqCls = item.isAdditionalInfo === "1" ? " required" : "";
    var html = '<tr class="clarify-prod-row" data-row-idx="' + idx + '">' +
      '<input type="hidden" name="clarifyIds[' + idx + ']" value="' + esc(item.id || "") + '">' +
      '<td><div class="ui-autocomplete"><input type="text" id="poClarifyProductInput_' + idx + '" class="clarify-input js-clarify-prod-input" placeholder="搜索产品名称或 ID" autocomplete="off"><input type="hidden" class="js-clarify-prod-select" name="products[' + idx + ']" id="poClarifyProductValue_' + idx + '" value="' + esc(item.productId || "") + '"></div></td>' +
      '<td><div class="ui-autocomplete"><input type="text" id="poClarifyPmInput_' + idx + '" class="clarify-input js-clarify-pm-input" placeholder="输入姓名或工号搜索" autocomplete="off"><input type="hidden" name="pm[' + idx + ']" id="poClarifyPmValue_' + idx + '" class="js-clarify-pm-value" required></div></td>' +
      '<td><input type="date" class="clarify-input" name="demandCompletionDate[' + idx + ']" value="' + esc(item.demandCompletionDate || "") + '"></td>' +
      '<td><input type="text" class="clarify-input" name="systemClarifyDesc[' + idx + ']" placeholder="说明" value="' + esc(item.systemClarifyDesc || "") + '"></td>' +
      '<td style="text-align:center;"><input type="checkbox" class="js-clarify-is-add" name="isAdditionalInfo[' + idx + ']" value="1"' + isAdd + '></td>' +
      '<td><input type="text" class="clarify-input js-clarify-add-info' + reqCls + '" name="additionalInfo[' + idx + ']" placeholder="总领文档链接" value="' + esc(item.additionalInfo || "") + '"></td>' +
      '<td style="text-align:center;"><input type="radio" class="js-clarify-main-radio" name="isMainSystemRadio" value="' + idx + '"' + isMain + '></td>' +
      '<td style="text-align:center;"><button type="button" class="clarify-action-btn js-add-prod-row" title="添加产品">+</button> <button type="button" class="clarify-action-btn del js-del-prod-row" title="删除产品">&minus;</button></td></tr>';
    $("#poClarifyProductTbody").append(html);
    bindProductAutocomplete("poClarifyProductInput_" + idx, "poClarifyProductValue_" + idx, products, item.productId);
    bindUserAutocomplete("poClarifyPmInput_" + idx, "poClarifyPmValue_" + idx, buildPmOptionsForProduct(item.productId), item.pm);
    syncStoryProductOptions();
  }

  function addStoryRow(item) {
    item = item || {};
    var idx = storyRowIndex++, isChecked = item.checked !== false ? " checked" : "";
    var roleVal = item.role || "", gvVal = item.gv || "", prodVal = item.productId || "";
    var ptVal = item.point || 2, revVal = item.revpoint || ptVal, srcVal = item.sourceType || "MANUAL", aiCode = item.aiCode != null ? item.aiCode : "";
    var ptKw = item.pointKeyword || (currentFormData && currentFormData.pointToKeyword && currentFormData.pointToKeyword[String(ptVal)]) || "微型";
    var revList = (currentFormData && currentFormData.allPointList) || [2, 3, 5, 8];
    if (srcVal === "AI_GENERATED" && currentFormData && currentFormData.revpointList && currentFormData.revpointList[String(ptVal)]) {
      revList = currentFormData.revpointList[String(ptVal)];
    }
    var revOpts = "";
    revList.forEach(function (p) {
      var kw = (currentFormData && currentFormData.pointToKeyword && currentFormData.pointToKeyword[String(p)]) || p;
      revOpts += '<option value="' + p + '"' + (String(revVal) === String(p) ? " selected" : "") + ">" + esc(kw) + " [" + p + "]</option>";
    });
    var html = '<tr class="clarify-story-row" data-row-idx="' + idx + '">' +
      '<input type="hidden" name="userStoryId[' + idx + ']" value="' + esc(item.id || "") + '">' +
      '<input type="hidden" name="sourceType[' + idx + ']" value="' + esc(srcVal) + '">' +
      '<input type="hidden" name="aiCode[' + idx + ']" value="' + esc(aiCode) + '">' +
      '<input type="hidden" name="point[' + idx + ']" value="' + esc(ptVal) + '">' +
      '<td style="text-align:center;"><input type="checkbox" class="js-story-check" name="userStoryChecked[' + idx + ']" value="1"' + isChecked + '></td>' +
      '<td><input type="text" class="clarify-input" name="role[' + idx + ']" placeholder="输入角色名称（如：系统用户）" value="' + esc(roleVal) + '" required></td>' +
      '<td><textarea class="clarify-textarea" name="gv[' + idx + ']" rows="1" placeholder="输入用户目标与业务价值" required>' + esc(gvVal) + '</textarea></td>' +
      '<td><select class="clarify-select js-story-prod-select" name="entryProductID[' + idx + ']" data-saved-val="' + esc(prodVal) + '" required><option value="">请选择产品</option></select></td>' +
      '<td style="text-align:center;"><span class="clarify-badge-tag">' + esc(ptKw) + '</span></td>' +
      '<td style="text-align:center;"><select class="clarify-select js-story-revpoint" name="revpoint[' + idx + ']">' + revOpts + '</select></td>' +
      '<td style="text-align:center;"><button type="button" class="clarify-action-btn js-add-story-row" title="添加故事">+</button> <button type="button" class="clarify-action-btn del js-del-story-row" title="删除故事">&minus;</button></td></tr>';
    $("#poClarifyStoryTbody").append(html);
    syncStoryProductOptions();
    updateTotalScale();
  }

  function syncStoryProductOptions() {
    var selectedProds = [];
    $("#poClarifyProductTbody tr").each(function () {
      var val = $(this).find(".js-clarify-prod-select").val();
      var txt = $(this).find(".js-clarify-prod-input").val() || val;
      if (val && val !== "0") selectedProds.push({ id: val, name: txt });
    });

    $(".js-story-prod-select").each(function () {
      var $sel = $(this);
      var cur = $sel.val() || $sel.attr("data-saved-val") || "";
      $sel.empty().append('<option value="">请选择产品</option>');
      selectedProds.forEach(function (p) {
        var isSel = String(p.id) === String(cur) ? " selected" : "";
        $sel.append('<option value="' + esc(p.id) + '"' + isSel + ">" + esc(p.name) + "</option>");
      });
      if (cur) $sel.val(cur);
    });
  }

  function updateTotalScale() {
    var total = 0;
    $("#poClarifyStoryTbody tr").each(function () {
      var $row = $(this);
      if ($row.find(".js-story-check").is(":checked")) {
        total += parseInt($row.find(".js-story-revpoint").val(), 10) || 0;
      }
    });
    $("#poClarifyScaleEstimation").val(total);
  }

  function toggleCategoryRule() {
    var cat = $("#poClarifyCategory").val();
    var noAi = (currentFormData && currentFormData.noAiCategories) || [];
    if (noAi.indexOf(cat) >= 0) {
      $("#poClarifyStorySection").slideUp(150);
      $("#poClarifyScaleEstimation").val(0);
    } else {
      $("#poClarifyStorySection").slideDown(150);
      updateTotalScale();
    }
  }

  function toggleFinanceAlert() { $("#poClarifyFinanceAlert").toggle(getRadioValue("isRelatedAccounts") === "1"); }

  function bindEvents() {
    $(document).on("click", "#poDemandClarifyCloseBtn, #poClarifyCancelBtn, #poDemandClarifyBackdrop", closeModal);
    $(document).on("keydown", function (e) {
      if (e.key === "Escape" && $("#poDemandClarifyModal").hasClass("show")) { closeModal(); }
    });
    $(document).on("click", ".js-po-drawer-action[data-action-key='clarify'], .js-drawer-clarify-btn", function (e) {
      e.preventDefault();
      var id = String($(this).attr("data-demand-id") || "").replace(/^US/i, "").trim();
      if (id) openModal(id);
    });
    $(document).on("click", "#poClarifyRetryBtn", function () {
      if (currentDemandId) {
        $("#poClarifyLoadingState").show();
        $("#poClarifyErrorState, #poDemandClarifyForm").hide();
        loadClarifyData(currentDemandId);
      }
    });
    $(document).on("click", "#poClarifyDescToggle", function () {
      $("#poClarifyDescContent").slideToggle(150);
      $("#poClarifyDescArrow").toggleClass("expanded");
    });
    $(document).on("change", "#poClarifyCategory", toggleCategoryRule);
    $(document).on("change", 'input[name="isRelatedAccounts"]', toggleFinanceAlert);

    function onProductSelected(idx) {
      var $row = $("#poClarifyProductTbody tr[data-row-idx='" + idx + "']");
      var pId = $row.find(".js-clarify-prod-select").val();
      var selectedProduct = findProduct((currentFormData && currentFormData.productOptions) || [], pId);
      var pmOptions = buildPmOptionsForProduct(pId);
      var defaultPm = (selectedProduct && selectedProduct.po) || "";
      var currentPm = $row.find(".js-clarify-pm-value").val();
      bindUserAutocomplete("poClarifyPmInput_" + idx, "poClarifyPmValue_" + idx, pmOptions, currentPm || defaultPm);
      syncStoryProductOptions();
    }

    $(document).on("click", '.ui-autocomplete-dropdown[data-autocomplete-for^="poClarifyProductInput_"] .ui-autocomplete-option', function () {
      var inputID = $(this).closest(".ui-autocomplete-dropdown").attr("data-autocomplete-for");
      var idx = inputID.replace("poClarifyProductInput_", "");
      onProductSelected(idx);
    });

    $(document).on("change", ".js-clarify-prod-select", function () {
      var idx = $(this).closest("tr").attr("data-row-idx");
      if (idx !== undefined && idx !== null) {
        onProductSelected(idx);
      }
    });

    $(document).on("click", ".js-quick-prod-chip", function () {
      var pId = $(this).attr("data-prod-id");
      var pPo = $(this).attr("data-prod-po") || "";
      if (!pId) return;

      var emptyRowIdx = null;
      $("#poClarifyProductTbody tr").each(function () {
        var v = $(this).find(".js-clarify-prod-select").val();
        if (!v || v === "0") {
          emptyRowIdx = $(this).attr("data-row-idx");
          return false;
        }
      });

      if (emptyRowIdx !== null) {
        var products = (currentFormData && currentFormData.productOptions) || [];
        bindProductAutocomplete("poClarifyProductInput_" + emptyRowIdx, "poClarifyProductValue_" + emptyRowIdx, products, pId);
        var pmOptions = buildPmOptionsForProduct(pId);
        bindUserAutocomplete("poClarifyPmInput_" + emptyRowIdx, "poClarifyPmValue_" + emptyRowIdx, pmOptions, pPo);
        syncStoryProductOptions();
      } else {
        addProductRow({ productId: pId, pm: pPo });
      }
    });

    $(document).on("change", ".js-clarify-is-add", function () { $(this).closest("tr").find(".js-clarify-add-info").toggleClass("required", $(this).is(":checked")); });
    $(document).on("change", ".js-clarify-main-radio", function () { $(this).closest("tr").find(".js-clarify-is-add").prop("checked", true).trigger("change"); });
    $(document).on("click", ".js-add-prod-row", function () { addProductRow(); });
    $(document).on("click", ".js-del-prod-row", function () {
      if ($("#poClarifyProductTbody tr").length <= 1) { showToast("涉及产品至少保留一行", "error"); return; }
      var $tr = $(this).closest("tr"), idx = $tr.attr("data-row-idx");
      if (typeof window.destroyAutocomplete === "function") {
        window.destroyAutocomplete("poClarifyProductInput_" + idx);
        window.destroyAutocomplete("poClarifyPmInput_" + idx);
      }
      $tr.remove(); syncStoryProductOptions();
    });
    $(document).on("click", ".js-add-story-row", function () { addStoryRow(); });
    $(document).on("click", ".js-del-story-row", function () {
      if ($("#poClarifyStoryTbody tr").length <= 1) { showToast("用户故事至少保留一行", "error"); return; }
      $(this).closest("tr").remove(); updateTotalScale();
    });
    $(document).on("change", ".js-story-check, .js-story-revpoint", updateTotalScale);

    // AI 生成用户故事
    $(document).on("click", "#poClarifyAiGenBtn", function () {
      var prods = [];
      $("#poClarifyProductTbody tr").each(function () {
        var v = $(this).find(".js-clarify-prod-select").val();
        if (v && v !== "0") prods.push(v);
      });
      if (prods.length === 0) { showToast("请先在上方至少选择一个涉及产品", "error"); return; }

      var $btn = $("#poClarifyAiGenBtn"), $txt = $("#poClarifyAiBtnText");
      $btn.prop("disabled", true); $txt.text("AI生成中…");

      (window.appFetch || window.fetch)("/demands/" + encodeURIComponent(currentDemandId) + "/clarify/ai-generate", {
        method: "POST", headers: { "Content-Type": "application/json", "Accept": "application/json" },
        body: JSON.stringify({
          demandId: parseInt(currentDemandId, 10), products: prods,
          desc: (currentFormData && currentFormData.desc) || "", clarifyDesc: $("#poClarifyDesc").val() || ""
        })
      })
        .then(function (res) { return res.json().then(function (d) { if (!res.ok) throw new Error(d.message || d.error || "AI生成失败"); return d; }); })
        .then(function (data) {
          if (data && data.content) {
            parseAndAppendAiStories(data.content, data.aiCodes || []);
            showToast("AI用户故事条目已生成并追加", "success");
          } else { showToast("AI未返回有效故事内容", "error"); }
        })
        .catch(function (err) { showToast(err.message || "AI生成异常", "error"); })
        .then(function () { $btn.prop("disabled", false); $txt.text("AI辅助生成条目"); });
    });

    // 提交保存澄清
    $(document).on("submit", "#poDemandClarifyForm", function (e) {
      e.preventDefault();
      if (isSubmitting) return;
      var err = validateForm();
      if (err) { showToast(err, "error"); return; }

      var payload = buildPayload(), $btn = $("#poClarifySubmitBtn"), $cancelBtn = $("#poClarifyCancelBtn");
      isSubmitting = true; $btn.prop("disabled", true).text("保存中…"); $cancelBtn.prop("disabled", true);

      (window.appFetch || window.fetch)("/demands/" + encodeURIComponent(currentDemandId) + "/clarify", {
        method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(payload)
      })
        .then(function (res) {
          return res.json().then(function (d) {
            if (!res.ok) { var errObj = new Error(d.message || d.error || "保存失败"); errObj.status = res.status; throw errObj; }
            return d;
          });
        })
        .then(function (data) { handleClarifySuccess(data && data.message); })
        .catch(function (err) {
          if (err.status === 422 || (err.message && err.message.indexOf("当前需求不是待澄清状态") >= 0)) {
            handleClarifySuccess("该需求已完成澄清或状态已更新");
          } else { showToast(err.message || "保存异常", "error"); }
        })
        .then(function () { isSubmitting = false; $btn.prop("disabled", false).text("保存澄清"); $cancelBtn.prop("disabled", false); });
    });
  }

  function parseAndAppendAiStories(csvContent, aiCodes) {
    var lines = csvContent.split("EOF"), dataLines = (lines.length >= 2 ? lines[1] : csvContent).trim().split("\n");
    var firstPid = $("#poClarifyProductTbody tr:first .js-clarify-prod-select").val() || "";

    // 若当前只有一行且未填任何内容（空白默认行），先移除该行以避免残留空行导致提交校验失败
    var $existingRows = $("#poClarifyStoryTbody tr");
    if ($existingRows.length === 1) {
      var $firstRow = $existingRows.first();
      if (!$firstRow.find('input[name^="role["]').val().trim() && !$firstRow.find('textarea[name^="gv["]').val().trim()) {
        $firstRow.remove();
      }
    }

    for (var i = 0; i < dataLines.length; i++) {
      var line = dataLines[i].trim();
      if (!line || line.indexOf("|") < 0) continue;
      var cols = line.split("|");
      if (cols.length >= 5) {
        addStoryRow({
          role: cols[0].trim(), gv: cols[1].trim(), productId: cols[2].trim() || firstPid,
          point: parseInt(cols[4].trim(), 10) || 2, revpoint: parseInt(cols[4].trim(), 10) || 2,
          sourceType: "AI_GENERATED", aiCode: aiCodes[i] != null ? aiCodes[i] : "", checked: true
        });
      }
    }
  }

  function validateForm() {
    if (!$("#poClarifyCategory").val()) return "请选择需求类别";
    if (!$("#poClarifyBRA").val()) return "请选择需求负责人";
    var mainRadio = $('input[name="isMainSystemRadio"]:checked').val();
    if (mainRadio === undefined || mainRadio === null) return "必须指定一个主系统";

    var prodCount = 0, prodSet = {}, prodErr = "";
    $("#poClarifyProductTbody tr").each(function () {
      var idx = $(this).attr("data-row-idx"), p = $(this).find(".js-clarify-prod-select").val();
      var pm = $(this).find(".js-clarify-pm-value").val(), isAdd = $(this).find(".js-clarify-is-add").is(":checked");
      var addInfo = $(this).find(".js-clarify-add-info").val().trim();
      if (!p || p === "0") { prodErr = "涉及产品不能为空"; return false; }
      if (prodSet[p]) { prodErr = "涉及产品不能重复选择"; return false; }
      prodSet[p] = true;
      if (!pm) { prodErr = "需求分析人员不能为空"; return false; }
      if (isAdd && !addInfo) { prodErr = "勾选涉及总领文档时，总领文档链接不能为空"; return false; }
      if (String(idx) === String(mainRadio) && !p) { prodErr = "主系统对应涉及产品不能为空"; return false; }
      prodCount++;
    });
    if (prodErr) return prodErr;
    if (prodCount === 0) return "至少需要一条涉及产品";

    var cat = $("#poClarifyCategory").val(), noAi = (currentFormData && currentFormData.noAiCategories) || [];
    var aiList = (currentFormData && currentFormData.aiCategories) || [];
    if (noAi.indexOf(cat) < 0) {
      var scale = parseInt($("#poClarifyScaleEstimation").val(), 10) || 0;
      if (scale <= 0) return "需求规模估算不能为空且必须大于0";
      var checkedStories = 0, storyErr = "";
      $("#poClarifyStoryTbody tr").each(function () {
        var $r = $(this);
        if ($r.find(".js-story-check").is(":checked")) {
          checkedStories++;
          if (!$r.find('input[name^="role["]').val().trim()) { storyErr = "采纳的用户故事角色不能为空"; return false; }
          if (!$r.find('textarea[name^="gv["]').val().trim()) { storyErr = "采纳的用户故事目标和价值不能为空"; return false; }
          if (!$r.find('.js-story-prod-select').val()) { storyErr = "采纳的用户故事涉及产品不能为空"; return false; }
          if (!$r.find('.js-story-revpoint').val()) { storyErr = "采纳的用户故事校准故事点不能为空"; return false; }
        }
      });
      if (storyErr) return storyErr;
      if (aiList.indexOf(cat) >= 0 && checkedStories === 0) return "当前类别必须至少采纳一条用户故事条目";
    }
    return null;
  }

  function buildPayload() {
    var prods = [], pms = [], dates = [], descs = [], isAdds = [], addInfos = [], clarifyIds = [];
    var isMain = {}, mainRadioVal = $('input[name="isMainSystemRadio"]:checked').val();
    $("#poClarifyProductTbody tr").each(function (arrIdx) {
      var $r = $(this);
      prods.push($r.find(".js-clarify-prod-select").val() || ""); pms.push($r.find(".js-clarify-pm-value").val() || "");
      dates.push($r.find('input[name^="demandCompletionDate["]').val() || ""); descs.push($r.find('input[name^="systemClarifyDesc["]').val() || "");
      isAdds.push($r.find(".js-clarify-is-add").is(":checked") ? "1" : "0"); addInfos.push($r.find(".js-clarify-add-info").val() || "");
      clarifyIds.push($r.find('input[name^="clarifyIds["]').val() || "");
      if (String($r.attr("data-row-idx")) === String(mainRadioVal)) isMain[String(arrIdx)] = "1";
    });

    var sIds = [], sNos = [], sChecked = {}, roles = [], gvs = [], eProds = [], pts = [], revs = [], srcs = [], aiCodes = [];
    $("#poClarifyStoryTbody tr").each(function (arrIdx) {
      var $r = $(this);
      sIds.push($r.find('input[name^="userStoryId["]').val() || ""); sNos.push(arrIdx + 1);
      roles.push($r.find('input[name^="role["]').val() || ""); gvs.push($r.find('textarea[name^="gv["]').val() || "");
      eProds.push($r.find(".js-story-prod-select").val() || ""); pts.push($r.find('input[name^="point["]').val() || "2");
      revs.push($r.find(".js-story-revpoint").val() || "2"); srcs.push($r.find('input[name^="sourceType["]').val() || "MANUAL");
      aiCodes.push($r.find('input[name^="aiCode["]').val() || null);
      if ($r.find(".js-story-check").is(":checked")) sChecked[String(arrIdx)] = "1";
    });

    return {
      id: parseInt(currentDemandId, 10), category: $("#poClarifyCategory").val(), bra: $("#poClarifyBRA").val(), qd: $("#poClarifyQD").val(), rd: $("#poClarifyRD").val(), clarifyDesc: $("#poClarifyDesc").val(), scaleEstimation: parseInt($("#poClarifyScaleEstimation").val(), 10) || 0,
      isNewProduct: getRadioValue("isNewProduct"), isRelatedAccounts: getRadioValue("isRelatedAccounts"), isNewFunction: getRadioValue("isNewFunction"), isOtherImportantOrder: getRadioValue("isOtherImportantOrder"), multiLegalPersonLogo: getRadioValue("multiLegalPersonLogo"), comment: "工作台需求澄清",
      products: prods, pm: pms, demandCompletionDate: dates, systemClarifyDesc: descs, isAdditionalInfo: isAdds, additionalInfo: addInfos, isMainSystem: isMain, clarifyIds: clarifyIds,
      userStoryNo: sNos, userStoryChecked: sChecked, userStoryId: sIds, role: roles, gv: gvs, entryProductID: eProds, point: pts, revpoint: revs, sourceType: srcs, aiCode: aiCodes
    };
  }

  window.openPoDemandClarifyModal = openModal;
  $(bindEvents);
})();
