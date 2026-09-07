/**
 * 研发工作台 · 指标雷达（V1.3 · 引用 QWShared）
 *
 * 职责定位（2026-08-20 质效模块收口）：
 *   回答「当前整体质效表现怎么样？哪里表现不好？」——看整体、找问题的页面。
 *
 * 结构：分析范围 → 五分类紧凑卡（分数 + 状态统计 + 主要风险） → 分组指标明细
 * 表格 7 列：指标 / 当前值 / 目标 / 状态 / 近6期趋势(sparkline) / 环比 / 操作
 *
 * 数据边界：敏捷小组范围由 QWShared 的真实 teamgroup 参数控制；V1 不提供人员“团队”
 * 或前端时间区间过滤，禁止模板里的示例人名/未接后端的时间下拉冒充真实查询条件。
 */
(function () {
  'use strict';

  const S = window.QWShared;
  if (!S) { console.warn('[po-risk-radar] QWShared 未加载'); return; }
  const { CATEGORY_META, METRICS, RADAR, escapeHtml, statusLabel, statusCls, computeStatus } = S;

  function runtimeStatus(metric) {
    const value = metric ? computeStatus(metric) : 'normal';
    return value === 'danger' || value === 'warn' ? value : 'normal';
  }

  function runtimeStatusLabel(metric) {
    if (!metric) return '正常';
    return statusLabel(metric.severity || metric.status || 'normal');
  }

  function normalizeSeverity(value) {
    const raw = String(value || '').toLowerCase();
    if (raw === 'danger' || raw === 'critical') return 'danger';
    if (raw === 'warn' || raw === 'warning') return 'warn';
    return 'normal';
  }

  function sparkline(series, status) {
    const values = Array.isArray(series) ? series.filter(function (v) { return Number.isFinite(Number(v)); }).map(Number) : [];
    if (values.length < 2) return '<span class="rr-delta is-neutral">—</span>';
    const w = 90, h = 26, pad = 3;
    const min = Math.min.apply(null, values);
    const max = Math.max.apply(null, values);
    const range = max - min || 1;
    const stepX = (w - pad * 2) / (values.length - 1);
    const normY = function (v) { return h - pad - ((v - min) / range) * (h - pad * 2); };
    const points = values.map(function (v, i) { return (pad + i * stepX) + ',' + normY(v); }).join(' ');
    return '<svg class="rr-spark" viewBox="0 0 ' + w + ' ' + h + '" preserveAspectRatio="none">' +
      '<polyline class="rr-spark-line rr-status-' + status + '" points="' + points + '" />' +
      '</svg>';
  }

  function statusBadge(metric) {
    const level = runtimeStatus(metric);
    return '<span class="rr-status rr-status-' + level + '">' + escapeHtml(runtimeStatusLabel(metric)) + '</span>';
  }

  function deltaCell(m) {
    const series = m.trendSeries || [];
    if (series.length < 2) return '<span class="rr-delta is-neutral">—</span>';
    const last = series[series.length - 1];
    const prev = series[series.length - 2];
    const delta = Math.round((last - prev) * 10) / 10;
    if (Math.abs(delta) < 0.01) return '<span class="rr-delta is-neutral">0</span>';
    const up = delta > 0;
    const isGood = (up && m.goodDirection === 'up') || (!up && m.goodDirection === 'down');
    const cls = isGood ? 'is-good' : 'is-bad';
    return '<span class="rr-delta ' + cls + '">' + (delta > 0 ? '+' : '') + delta.toFixed(1).replace(/\.0$/, '') + escapeHtml(m.unit) + '</span>';
  }

  function renderCategoryKpi() {
    const host = document.getElementById('rrCategoryKpi');
    if (!host) return;
    const groups = {};
    Object.keys(CATEGORY_META).forEach(function (k) { groups[k] = []; });
    METRICS.forEach(function (m) {
      if (!groups[m.category]) groups[m.category] = [];
      groups[m.category].push(m);
    });
    const card = function (key) {
      const list = groups[key] || [];
      const normal = list.filter(function (m) { return runtimeStatus(m) === 'normal'; }).length;
      const warn   = list.filter(function (m) { return runtimeStatus(m) === 'warn'; }).length;
      const danger = list.filter(function (m) { return runtimeStatus(m) === 'danger'; }).length;
      const meta = CATEGORY_META[key];
      const radar = (RADAR || []).find(function (r) { return r.category === key; }) || {};
      const score = radar.score || 0;
      const sev = radar.severity ? normalizeSeverity(radar.severity) : (danger ? 'danger' : (warn ? 'warn' : 'normal'));
      const mainRisk = list.filter(function (m) { return runtimeStatus(m) === 'danger'; })[0]
        || list.filter(function (m) { return runtimeStatus(m) === 'warn'; })[0] || null;
      return '<div class="rr-cat-card">' +
        '<div class="rr-cat-hdr">' +
          '<i class="fas ' + meta.icon + '"></i>' +
          '<span>' + escapeHtml(meta.label) + '</span>' +
          '<span class="rr-cat-total">' + list.length + ' 项</span>' +
          '<span class="rr-cat-score rr-status-' + sev + '"><strong>' + score + '</strong><em>分</em></span>' +
        '</div>' +
        '<div class="rr-cat-stats">' +
          '<span class="rr-stat-ok"><i></i>正常 ' + normal + '</span>' +
          '<span class="rr-stat-warn"><i></i>关注 ' + warn + '</span>' +
          '<span class="rr-stat-risk"><i></i>风险 ' + danger + '</span>' +
        '</div>' +
        '<div class="rr-cat-mainrisk" title="' + (mainRisk ? '主要风险指标：' + escapeHtml(mainRisk.metricName) : '该分类当前无风险/关注指标') + '">' +
          '<em>主要风险</em>' +
          (mainRisk ? '<span>' + escapeHtml(mainRisk.metricName) + '</span>' : '<span class="is-none">—</span>') +
        '</div>' +
      '</div>';
    };
    host.innerHTML = Object.keys(CATEGORY_META).map(card).join('');
  }

  function hideUnsupportedFilters() {
    ['rrFilterTeam', 'rrFilterRange'].forEach(function (id) {
      const el = document.getElementById(id);
      if (!el) return;
      const label = el.previousElementSibling;
      if (label && String(label.tagName || '').toLowerCase() === 'label') label.hidden = true;
      el.hidden = true;
      el.disabled = true;
      el.setAttribute('aria-hidden', 'true');
    });
  }

  function readFilter() {
    return {
      category: (document.getElementById('rrFilterCategory') || {}).value || '',
      status:   (document.getElementById('rrFilterStatus') || {}).value || ''
    };
  }

  function renderTable() {
    const host = document.getElementById('rrTableHost');
    if (!host) return;
    const f = readFilter();
    const rows = METRICS.filter(function (m) {
      if (f.category && m.category !== f.category) return false;
      if (f.status && runtimeStatus(m) !== f.status) return false;
      return true;
    });

    const groups = {};
    Object.keys(CATEGORY_META).forEach(function (k) { groups[k] = []; });
    rows.forEach(function (m) {
      if (!groups[m.category]) groups[m.category] = [];
      groups[m.category].push(m);
    });

    if (!rows.length) {
      host.innerHTML = '<div class="rr-empty">无匹配指标</div>';
      return;
    }

    let html = '';
    Object.keys(CATEGORY_META).forEach(function (key) {
      const list = groups[key] || [];
      if (!list.length) return;
      const meta = CATEGORY_META[key];
      html += '<div class="rr-tbl-section">' +
        '<h3 class="rr-tbl-section-hdr"><i class="fas ' + meta.icon + '"></i>' + escapeHtml(meta.label) +
        '<span class="rr-tbl-section-meta">· ' + list.length + ' 项</span></h3>' +
        '<div class="po-list-scroll" style="overflow-x:auto">' +
        '<table class="tbl rr-tbl">' +
        '<thead><tr>' +
          '<th style="width:230px">指标</th>' +
          '<th style="width:96px">当前值</th>' +
          '<th style="width:96px">目标</th>' +
          '<th style="width:76px">状态</th>' +
          '<th style="width:150px">近6期趋势</th>' +
          '<th style="width:96px">环比</th>' +
          '<th style="width:88px">操作</th>' +
        '</tr></thead><tbody>' +
        list.map(function (m) {
          const s = runtimeStatus(m);
          const currentText = m.currentText || (m.currentValue + (m.unit || ''));
          const targetText = m.targetText || (m.targetValue + (m.unit || ''));
          return '<tr data-code="' + escapeHtml(m.metricCode) + '">' +
            '<td><a href="javascript:void(0)" class="rr-metric-link" data-code="' + escapeHtml(m.metricCode) + '">' +
              '<span class="rr-metric-name">' + escapeHtml(m.metricName) + '</span>' +
              '<span class="rr-metric-code">' + escapeHtml(m.metricCode) + '</span>' +
            '</a></td>' +
            '<td><span class="rr-current rr-status-' + s + '">' + escapeHtml(currentText) + '</span></td>' +
            '<td>' + escapeHtml(targetText) + '</td>' +
            '<td>' + statusBadge(m) + '</td>' +
            '<td>' + sparkline(m.trendSeries, s) + '</td>' +
            '<td>' + deltaCell(m) + '</td>' +
            '<td><button type="button" class="action-btn rr-detail-btn" data-code="' + escapeHtml(m.metricCode) + '">详情</button></td>' +
          '</tr>';
        }).join('') +
        '</tbody></table>' +
        '</div>' +
      '</div>';
    });
    host.innerHTML = html;

    host.querySelectorAll('.rr-detail-btn').forEach(function (btn) {
      btn.addEventListener('click', function () { goDetail(btn.dataset.code); });
    });
    host.querySelectorAll('.rr-metric-link').forEach(function (a) {
      a.addEventListener('click', function () { goDetail(a.dataset.code); });
    });
  }

  function goDetail(code) {
    if (typeof navTo === 'function') {
      navTo('metricDetail', { metricCode: code });
    } else if (typeof window.openMetricDetail === 'function') {
      window.openMetricDetail(code);
    }
  }

  function renderRiskRadarFull() {
    hideUnsupportedFilters();
    Promise.all([S.ensureMetricsLoaded(), S.ensureRadarLoaded()]).then(function () {
      if (S.renderMetricAgileChips) S.renderMetricAgileChips();
      renderCategoryKpi();
      renderTable();
    }).catch(function (err) {
      const host = document.getElementById('rrTableHost');
      if (host) host.innerHTML = '<div class="rr-empty">指标接口不可用：' + escapeHtml(err && err.message || err) + '</div>';
    });
  }

  window.renderRiskRadarFull = renderRiskRadarFull;
  window.__rrRender = renderRiskRadarFull;
})();