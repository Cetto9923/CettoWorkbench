/**
 * 研发工作台 · 指标管理（Phase 7 canonical readonly）
 *
 * 真源边界：
 * - 指标定义与运行快照只来自 QWShared -> 后端 /workbench/api/metrics?scope=global。
 * - 后端当前没有指标定义持久化写 API，因此本页不提供启停、阈值、口径或负责人编辑。
 * - 不使用 localStorage / sessionStorage / 页面 state 模拟“配置保存成功”。
 * - 详情 Drawer 只展示真实接口字段；需要配置写能力时必须先补服务端合同。
 */
(function () {
  'use strict';

  const S = window.QWShared;
  if (!S) { console.warn('[po-metrics-manage] QWShared 未加载'); return; }

  const { CATEGORY_META, METRICS, escapeHtml, statusLabel, computeStatus } = S;
  let state = [];

  function categoryMeta(key) {
    return CATEGORY_META[key] || { label: key || '未分类', color: 'var(--t3)' };
  }

  function parseSource(src) {
    const s = String(src || '');
    const systems = [];
    const objects = [];
    const pick = function (list, re, label) {
      if (re.test(s) && list.indexOf(label) === -1) list.push(label);
    };
    pick(systems, /禅道|zt_/i, '禅道');
    pick(systems, /devops|gitfox|gitlab|流水线/i, 'DevOps');
    pick(systems, /门禁/i, '质量门禁');
    pick(systems, /发布/i, '发布平台');
    pick(objects, /需求|story|demand/i, '需求');
    pick(objects, /任务|task/i, '任务');
    pick(objects, /bug|缺陷/i, 'Bug');
    pick(objects, /版本|窗口|release|build/i, '版本');
    pick(objects, /迭代|sprint|execution/i, '迭代');
    pick(objects, /排期/i, '排期');
    pick(objects, /项目|project/i, '项目');
    return { systems: systems, objects: objects };
  }

  function composeSource(systems, objects) {
    const sys = (systems || []).join(' / ');
    const obj = (objects || []).join(' / ');
    if (sys && obj) return sys + ' · ' + obj;
    return sys || obj;
  }

  function sourceLabel(src) {
    const s = String(src || '').trim();
    if (!s) return '—';
    if (/devops|gitfox|gitlab|门禁|流水线/i.test(s)) return 'DevOps · 质量门禁';
    const rules = [
      [/zt_demand\b/ig, '禅道 · 需求'],
      [/zt_story\b/ig, '禅道 · 需求'],
      [/zt_task\b/ig, '禅道 · 任务'],
      [/zt_bug\b/ig, '禅道 · Bug'],
      [/zt_project\b|zt_execution\b/ig, '禅道 · 项目'],
      [/zt_build\b|zt_release\b|zt_versionwindow\b/ig, '禅道 · 版本窗口'],
      [/禅道需求/ig, '禅道 · 需求'],
      [/禅道任务/ig, '禅道 · 任务'],
      [/禅道Bug/ig, '禅道 · Bug'],
      [/禅道迭代/ig, '禅道 · 迭代'],
      [/禅道排期/ig, '禅道 · 排期']
    ];
    let out = s;
    rules.forEach(function (rule) { out = out.replace(rule[0], rule[1]); });
    return out;
  }

  function syncStateFromMetrics() {
    state = METRICS.map(function (m) {
      const parsed = parseSource(m.source);
      return {
        metricCode: m.metricCode,
        metricName: m.metricName,
        category: m.category,
        ownerRole: m.owner || '',
        configMaintainer: '',
        period: m.period || '',
        enabled: null,
        source: m.source || '',
        sourceSystems: parsed.systems,
        sourceObjects: parsed.objects,
        formula: m.formula || '',
        caliberNote: m.description || '',
        unit: m.unit || '',
        goodDirection: m.goodDirection || 'down',
        targetValue: m.targetValue,
        targetText: m.targetText || '',
        warningThreshold: m.warningThreshold,
        dangerThreshold: m.dangerThreshold,
        status: m.status,
        severity: m.severity,
        lastCalcTime: m.lastCalcTime || '',
        dataQuality: m.dataQuality || ''
      };
    });
  }

  function sourceDisplay(m) {
    return composeSource(m.sourceSystems, m.sourceObjects) || sourceLabel(m.source);
  }

  function directionLabel(direction) {
    if (direction === 'up') return '越大越好';
    if (direction === 'down') return '越小越好';
    return direction || '—';
  }

  function targetTextOf(m) {
    if (m.targetText) return String(m.targetText);
    if (m.targetValue == null || m.targetValue === '') return '—';
    const unit = m.unit || '';
    if (m.goodDirection === 'up') return '≥' + m.targetValue + unit;
    if (m.goodDirection === 'down') return '≤' + m.targetValue + unit;
    return String(m.targetValue) + unit;
  }

  function thresholdValue(value, unit) {
    if (value == null || value === '') return '—';
    return String(value) + (unit || '');
  }

  function warningRuleLines(m) {
    const unit = m.unit || '';
    const warning = thresholdValue(m.warningThreshold, unit);
    const danger = thresholdValue(m.dangerThreshold, unit);
    if (m.goodDirection === 'up') {
      return ['关注 < ' + warning, '风险 < ' + danger];
    }
    return ['关注 > ' + warning, '风险 > ' + danger];
  }

  function ensureReadonlyNote() {
    const page = document.getElementById('page-metricsManage');
    if (!page || page.querySelector('[data-metrics-readonly-note]')) return;
    const anchor = page.querySelector('.page-hdr') || page.firstElementChild;
    if (!anchor) return;
    const note = document.createElement('div');
    note.dataset.metricsReadonlyNote = '1';
    note.className = 'metrics-readonly-note';
    note.style.cssText = 'margin:0 0 12px;padding:8px 12px;border:1px solid var(--bd,#e5e7eb);border-radius:6px;background:var(--bg2,#f8fafc);font-size:12px;color:var(--t2,#64748b)';
    note.textContent = '指标定义当前为只读真源：页面展示后端配置与运行数据；启停、阈值及口径维护需待服务端配置写接口接入后开放。';
    anchor.insertAdjacentElement('afterend', note);
  }

  function normalizeReadonlyStatusFilter() {
    const select = document.getElementById('mmFilterStatus');
    if (!select) return;
    const current = ['normal', 'warn', 'danger'].indexOf(String(select.value || '')) >= 0 ? String(select.value) : '';
    if (select.dataset && select.dataset.readonlyTruth === '1') {
      if (select.value !== current) select.value = current;
      return;
    }
    select.innerHTML = '<option value="">全部状态</option>'
      + '<option value="normal">正常</option>'
      + '<option value="warn">关注</option>'
      + '<option value="danger">风险</option>';
    select.value = current;
    select.title = '按真实运行快照状态筛选；指标启停配置当前未接入后端写接口';
    if (select.dataset) select.dataset.readonlyTruth = '1';
  }

  function metricRuntimeStatus(metric) {
    return metric ? computeStatus(metric) : 'normal';
  }

  function metricStatusLabel(metric) {
    return metric ? statusLabel(metric.severity || metric.status || 'normal') : '正常';
  }

  function renderSummary() {
    const host = document.getElementById('mmSummary');
    if (!host) return;
    const total = state.length;
    const categories = Object.keys(CATEGORY_META).filter(function (key) {
      return state.some(function (m) { return m.category === key; });
    }).length;
    const snapshots = state.filter(function (m) { return !!m.lastCalcTime; }).length;
    const abnormal = state.filter(function (m) {
      const source = METRICS.find(function (x) { return x.metricCode === m.metricCode; });
      return metricRuntimeStatus(source) !== 'normal';
    }).length;

    host.innerHTML =
      '<div class="mm-sum-line">' +
        '<span class="mm-sum-item"><strong>' + total + '</strong>核心指标</span>' +
        '<span class="mm-sum-item"><strong>' + categories + '</strong>分类</span>' +
        '<span class="mm-sum-item is-ok"><strong>' + snapshots + '</strong>有运行快照</span>' +
        '<span class="mm-sum-item is-warn"><strong>' + abnormal + '</strong>当前异常</span>' +
      '</div>';

    const dot = document.getElementById('mmDataStatus');
    if (dot) {
      const ok = total > 0 && snapshots === total;
      dot.innerHTML = '数据状态：<i class="mm-dq-dot ' + (ok ? 'is-ok' : 'is-bad') + '"></i>' +
        (ok ? '正常' : '部分缺少运行快照') +
        '<span class="mm-data-note">· 指标定义当前只读</span>';
    }
  }

  function readFilter() {
    return {
      category: (document.getElementById('mmFilterCategory') || {}).value || '',
      status: (document.getElementById('mmFilterStatus') || {}).value || '',
      keyword: ((document.getElementById('mmFilterKeyword') || {}).value || '').trim().toLowerCase()
    };
  }

  function filteredRows() {
    const f = readFilter();
    return state.filter(function (m) {
      if (f.category && m.category !== f.category) return false;
      if (f.status) {
        const source = METRICS.find(function (x) { return x.metricCode === m.metricCode; });
        if (metricRuntimeStatus(source) !== f.status) return false;
      }
      if (f.keyword) {
        const blob = [m.metricName, m.metricCode, m.source, m.formula, m.ownerRole].join(' ').toLowerCase();
        if (blob.indexOf(f.keyword) === -1) return false;
      }
      return true;
    });
  }

  function targetThresholdCell(m) {
    const unit = m.unit || '';
    return '<div class="mm-th" title="目标 ' + escapeHtml(targetTextOf(m)) +
      ' · 关注 ' + escapeHtml(thresholdValue(m.warningThreshold, unit)) +
      ' · 风险 ' + escapeHtml(thresholdValue(m.dangerThreshold, unit)) + '">' +
      '<strong>' + escapeHtml(targetTextOf(m)) + '</strong>' +
      '<span>关注 ' + escapeHtml(thresholdValue(m.warningThreshold, unit)) +
      ' · 风险 ' + escapeHtml(thresholdValue(m.dangerThreshold, unit)) + '</span>' +
      '</div>';
  }

  function renderList() {
    const host = document.getElementById('mmListHost');
    if (!host) return;
    const rows = filteredRows();
    if (!rows.length) {
      host.innerHTML = '<div class="rr-empty">无匹配指标</div>';
      return;
    }

    host.innerHTML =
      '<div class="po-list-scroll" style="overflow-x:auto">' +
      '<table class="tbl mm-tbl">' +
      '<thead><tr>' +
        '<th style="width:210px">指标名称 / 编码</th>' +
        '<th style="width:92px">分类</th>' +
        '<th style="width:150px">数据来源</th>' +
        '<th style="width:56px">周期</th>' +
        '<th style="width:168px">目标 / 阈值</th>' +
        '<th style="width:96px">责任角色</th>' +
        '<th style="width:76px">状态</th>' +
        '<th style="width:60px">配置</th>' +
        '<th style="width:76px">操作</th>' +
      '</tr></thead><tbody>' +
      rows.map(function (m) {
        const meta = categoryMeta(m.category);
        const source = METRICS.find(function (x) { return x.metricCode === m.metricCode; });
        const s = metricRuntimeStatus(source);
        return '<tr data-code="' + escapeHtml(m.metricCode) + '">' +
          '<td><div class="mm-metric-name">' + escapeHtml(m.metricName) + '</div>' +
          '<code class="mm-metric-code">' + escapeHtml(m.metricCode) + '</code></td>' +
          '<td><span class="mm-cat-tag" style="--mm-cat-color:' + meta.color + '">' + escapeHtml(meta.label) + '</span></td>' +
          '<td><span class="mm-source">' + escapeHtml(sourceDisplay(m)) + '</span></td>' +
          '<td>' + escapeHtml(m.period || '—') + '</td>' +
          '<td>' + targetThresholdCell(m) + '</td>' +
          '<td>' + escapeHtml(m.ownerRole || '—') + '</td>' +
          '<td><span class="rr-status rr-status-' + s + '">' + escapeHtml(metricStatusLabel(source)) + '</span></td>' +
          '<td><span title="后端未提供可写启停字段">只读</span></td>' +
          '<td><div class="cell-actions">' +
            '<button type="button" class="action-btn mm-detail-btn" data-code="' + escapeHtml(m.metricCode) + '">详情</button>' +
          '</div></td>' +
        '</tr>';
      }).join('') +
      '</tbody></table></div>';

    host.querySelectorAll('.mm-detail-btn').forEach(function (btn) {
      btn.addEventListener('click', function () { openDetailDrawer(btn.dataset.code); });
    });
  }

  function kv(label, valueHtml) {
    return '<div class="mm-kv"><label>' + label + '</label><div>' + valueHtml + '</div></div>';
  }

  function drawerSection(title, bodyHtml) {
    return '<div class="mm-drawer-sec"><div class="mm-drawer-sec-hd">' + title + '</div>' + bodyHtml + '</div>';
  }

  function optionTags(list) {
    if (!list || !list.length) return '—';
    return list.map(function (x) { return '<span class="mm-tag">' + escapeHtml(x) + '</span>'; }).join('');
  }

  function viewBody(m) {
    const meta = categoryMeta(m.category);
    const source = METRICS.find(function (x) { return x.metricCode === m.metricCode; });
    const s = metricRuntimeStatus(source);
    const unit = m.unit || '';
    const rules = warningRuleLines(m);

    return drawerSection('基本信息',
        '<div class="mm-kv-grid">' +
          kv('指标名称', escapeHtml(m.metricName)) +
          kv('指标编码', '<code class="mm-metric-code">' + escapeHtml(m.metricCode) + '</code>') +
          kv('分类', '<span class="mm-cat-tag" style="--mm-cat-color:' + meta.color + '">' + escapeHtml(meta.label) + '</span>') +
          kv('责任角色', escapeHtml(m.ownerRole || '—')) +
          kv('配置维护人', '—') +
          kv('统计周期', escapeHtml(m.period || '—')) +
          kv('配置状态', '<span title="后端当前未提供可写启停字段">只读</span>') +
        '</div>') +
      drawerSection('数据口径',
        '<div class="mm-kv-grid">' +
          kv('来源系统', optionTags(m.sourceSystems)) +
          kv('来源对象', optionTags(m.sourceObjects)) +
          kv('单位', escapeHtml(unit || '—')) +
        '</div>' +
        kv('计算规则', '<div class="mm-formula-box">' + escapeHtml(m.formula || '—') + '</div>') +
        kv('口径说明', escapeHtml(m.caliberNote || '—'))) +
      drawerSection('目标与预警',
        '<div class="mm-kv-grid">' +
          kv('指标方向', escapeHtml(directionLabel(m.goodDirection))) +
          kv('目标值', '<strong>' + escapeHtml(targetTextOf(m)) + '</strong>') +
          kv('关注阈值', escapeHtml(thresholdValue(m.warningThreshold, unit))) +
          kv('风险阈值', escapeHtml(thresholdValue(m.dangerThreshold, unit))) +
        '</div>' +
        kv('预警规则', '<div class="mm-rule-auto"><span>' + escapeHtml(rules[0]) + '</span><span>' +
          escapeHtml(rules[1]) + '</span><em>按后端指标方向与阈值展示</em></div>')) +
      drawerSection('运行信息',
        '<div class="mm-kv-grid">' +
          kv('当前状态', '<span class="rr-status rr-status-' + s + '">' + escapeHtml(metricStatusLabel(source)) + '</span>') +
          kv('最近计算', escapeHtml(m.lastCalcTime || '—')) +
          kv('数据质量', m.dataQuality === 'good' ? '良好' : escapeHtml(m.dataQuality || '—')) +
        '</div>');
  }

  function openDetailDrawer(code) {
    const m = state.find(function (x) { return x.metricCode === code; });
    if (!m) return;

    const overlay = document.createElement('div');
    overlay.className = 'mm-drawer-overlay';
    overlay.innerHTML =
      '<aside class="mm-drawer" role="dialog" aria-modal="true" aria-label="指标详情">' +
        '<div class="mm-drawer-hdr"><div>' +
          '<div class="mm-drawer-title">' + escapeHtml(m.metricName) + '</div>' +
          '<code class="mm-metric-code">' + escapeHtml(m.metricCode) + '</code>' +
          '<span class="mm-drawer-mode">只读</span>' +
        '</div><button type="button" class="modal-close" aria-label="关闭指标详情"><i class="fas fa-xmark"></i></button></div>' +
        '<div class="mm-drawer-body">' + viewBody(m) + '</div>' +
        '<div class="mm-drawer-ft">' +
          '<button type="button" class="action-btn" data-action="metricDetail">查看指标详情</button>' +
        '</div>' +
      '</aside>';
    document.body.appendChild(overlay);

    const close = function () {
      document.removeEventListener('keydown', onKey);
      if (overlay.parentNode) overlay.parentNode.removeChild(overlay);
    };
    const onKey = function (e) { if (e.key === 'Escape') close(); };

    overlay.addEventListener('click', function (e) { if (e.target === overlay) close(); });
    overlay.querySelector('.modal-close').addEventListener('click', close);
    overlay.querySelector('[data-action="metricDetail"]').addEventListener('click', function () {
      close();
      if (typeof navTo === 'function') navTo('metricDetail', { metricCode: m.metricCode });
    });
    document.addEventListener('keydown', onKey);
  }

  function renderMetricsManage() {
    normalizeReadonlyStatusFilter();
    ensureReadonlyNote();
    // 2026-08-31 P1-1: 指标管理页面不再默认走 scope=global。
    // ensureGlobalMetricsLoaded 内部已经按 super-admin 分流；
    // 非 super-admin 会回退到 ensureMetricsLoaded (敏捷小组范围)。
    S.ensureGlobalMetricsLoaded(function () {
      syncStateFromMetrics();
      normalizeReadonlyStatusFilter();
      ensureReadonlyNote();
      renderSummary();
      renderList();
    }).catch(function (err) {
      const host = document.getElementById('mmListHost');
      if (!host) return;
      // 没有可用敏捷小组 (且非 super-admin) → 给明确空状态而不是无限 403。
      const groups = (S.AGILE_GROUPS || []);
      const msg = groups.length === 0
        ? '当前账号未加入敏捷小组，暂无团队指标数据。如需查看全局指标请联系管理员配置权限。'
        : ('指标接口不可用：' + escapeHtml(err && err.message || err));
      host.innerHTML = '<div class="rr-empty">' + msg + '</div>';
    });
  }

  window.renderMetricsManage = renderMetricsManage;
  window.__mmRender = renderMetricsManage;
})();
