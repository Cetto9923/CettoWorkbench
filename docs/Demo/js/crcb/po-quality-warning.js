/**
 * 研发工作台 · 质效预警（Phase 7 canonical truth）
 *
 * 业务边界：
 * - 后端 ALERTS 中有真实 id 的记录才是可办理预警，可调用 handleAlert。
 * - 当没有真实 Alert 时，可从真实指标快照派生 W-*“指标实时异常”，仅用于提示，不是持久化预警记录。
 * - 建议动作当前没有对应写接口，只显示建议文本，不提供按钮 + toast 假办理。
 */
(function () {
  'use strict';

  const S = window.QWShared;
  if (!S) { console.warn('[po-quality-warning] QWShared 未加载'); return; }
  const { CATEGORY_META, METRICS, ALERTS, escapeHtml, statusLabel, statusCls, thresholdText, computeStatus, isTrendBad } = S;

  function isPersistedWarning(w) {
    return !!(w && w.alert && w.alert.id && String(w.alert.id).trim());
  }

  function deriveWarnings() {
    const persistedAlerts = ALERTS.filter(function (a) { return !!String(a && a.id || '').trim(); });
    if (persistedAlerts.length) {
      return persistedAlerts.map(function (a) {
        const m = METRICS.find(function (x) { return x.metricCode === a.metricCode; }) || {
          metricCode: a.metricCode,
          metricName: a.metricName,
          category: a.category,
          currentText: a.currentValue,
          currentValue: a.currentValue,
          unit: '',
          owner: a.owner,
          trend: a.trend,
          reason: a.reason,
          suggestion: a.suggestAction,
          affectedProjects: 0,
          affectedDemands: 0
        };
        m.reason = a.reason || m.reason;
        m.suggestion = a.suggestAction || m.suggestion;
        m.severity = a.severity || m.severity;
        m.status = statusCls(a.severity || m.status);
        m.warningStatus = statusLabel(a.severity || m.severity || m.status);
        m.owner = a.owner || m.owner;
        return {
          id: String(a.id || ''),
          metric: m,
          alert: a,
          persisted: true,
          tagCls: statusCls(a.severity),
          tagLabel: statusLabel(a.severity),
          title: a.metricName + (a.handled ? '已处理' : '预警'),
          threshold: a.threshold || thresholdText(m)
        };
      }).sort(function (a, b) {
        if (a.alert.handled !== b.alert.handled) return a.alert.handled ? 1 : -1;
        if (a.tagCls !== b.tagCls) return a.tagCls === 'danger' ? -1 : 1;
        return 0;
      });
    }

    return METRICS
      .filter(function (m) { return computeStatus(m) !== 'normal'; })
      .map(function (m) {
        // computeStatus 已经返回 normal / warn / danger，不能再交给 statusCls 二次归一化，
        // 否则 warning -> warn -> statusCls('warn') 会被错误降成 normal。
        const tagCls = computeStatus(m);
        const tagLabel = statusLabel(m.severity || m.status || tagCls);
        const bad = isTrendBad(m);
        return {
          id: 'W-' + m.metricCode,
          metric: m,
          alert: null,
          persisted: false,
          tagCls: tagCls,
          tagLabel: tagLabel,
          title: m.metricName + (bad ? '恶化' : '异常'),
          threshold: thresholdText(m)
        };
      })
      .sort(function (a, b) {
        if (a.tagCls !== b.tagCls) return a.tagCls === 'danger' ? -1 : 1;
        return 0;
      });
  }

  const qwFilter = { level: '', category: '', handled: '' };

  function filterWarnings(list) {
    return list.filter(function (w) {
      if (qwFilter.level && w.tagCls !== qwFilter.level) return false;
      if (qwFilter.category && w.metric.category !== qwFilter.category) return false;
      if (qwFilter.handled) {
        if (!isPersistedWarning(w)) return false;
        const handled = !!w.alert.handled;
        if (qwFilter.handled === 'open' && handled) return false;
        if (qwFilter.handled === 'done' && !handled) return false;
      }
      return true;
    });
  }

  function renderSummaryBar() {
    const host = document.getElementById('qwHealthSummary');
    if (!host) return;
    const list = deriveWarnings();
    const persistedCount = list.filter(isPersistedWarning).length;
    const risk = list.filter(function (w) { return w.tagCls === 'danger'; }).length;
    const warn = list.filter(function (w) { return w.tagCls === 'warn'; }).length;
    const projects = list.reduce(function (s, w) { return s + (Number(w.metric.affectedProjects) || 0); }, 0);
    const demands = list.reduce(function (s, w) { return s + (Number(w.metric.affectedDemands) || 0); }, 0);

    const chip = function (key, label, active) {
      return '<button type="button" class="qw-sum-chip' + (active ? ' active' : '') + '" data-filter="' + key + '">' + label + '</button>';
    };
    const catOptions = Object.keys(CATEGORY_META).map(function (k) {
      return '<option value="' + k + '"' + (qwFilter.category === k ? ' selected' : '') + '>' + escapeHtml(CATEGORY_META[k].label) + '</option>';
    }).join('');
    const headline = persistedCount > 0
      ? ('当前 ' + persistedCount + ' 项预警记录')
      : ('当前 ' + list.length + ' 项指标实时异常');

    host.innerHTML =
      '<div class="qw-sum">' +
        '<div class="qw-sum-main">' +
          '<strong class="qw-sum-count">' + headline + '</strong>' +
          '<span class="qw-sum-risk"><i></i>风险 ' + risk + '</span>' +
          '<span class="qw-sum-warn"><i></i>关注 ' + warn + '</span>' +
          '<span class="qw-sum-scope">涉及 ' + projects + ' 个项目 · ' + demands + ' 个需求</span>' +
        '</div>' +
        '<div class="qw-sum-filters">' +
          chip('all', '全部', !qwFilter.level && !qwFilter.handled) +
          chip('danger', '风险', qwFilter.level === 'danger') +
          chip('warn', '关注', qwFilter.level === 'warn') +
          (persistedCount ? chip('open', '未处理记录', qwFilter.handled === 'open') : '') +
          '<select class="qw-sum-select" id="qwFilterCategory" title="按指标分类筛选">' +
            '<option value="">全部分类</option>' + catOptions +
          '</select>' +
        '</div>' +
      '</div>';

    host.querySelectorAll('.qw-sum-chip').forEach(function (btn) {
      btn.addEventListener('click', function () {
        const key = btn.dataset.filter;
        if (key === 'all') { qwFilter.level = ''; qwFilter.handled = ''; }
        else if (key === 'danger' || key === 'warn') { qwFilter.level = key; qwFilter.handled = ''; }
        else if (key === 'open') { qwFilter.handled = 'open'; qwFilter.level = ''; }
        renderSummaryBar();
        renderWarningList();
      });
    });
    const catSel = host.querySelector('#qwFilterCategory');
    if (catSel) {
      catSel.addEventListener('change', function () {
        qwFilter.category = catSel.value;
        renderWarningList();
      });
    }
  }

  function trendDescOf(m) {
    const series = m.trendSeries || [];
    let up = 0, down = 0;
    for (let i = series.length - 1; i > 0; i--) {
      if (series[i] > series[i - 1]) up++; else break;
    }
    for (let i = series.length - 1; i > 0; i--) {
      if (series[i] < series[i - 1]) down++; else break;
    }
    const n = Math.max(up, down, 1);
    if (m.goodDirection === 'down') {
      if (m.trend === 'up') return '连续 ' + n + ' 月上升（恶化）';
      if (m.trend === 'down') return '连续 ' + n + ' 月下降（好转）';
    } else {
      if (m.trend === 'up') return '连续 ' + n + ' 月上升（好转）';
      if (m.trend === 'down') return '连续 ' + n + ' 月下降（恶化）';
    }
    return '近月持平';
  }

  function suggestionTextHtml(m) {
    const items = (m.suggestion || '查看需求 / 查看项目 / 发起风险 / 调整计划')
      .split('/').map(function (a) { return a.trim(); }).filter(Boolean);
    return items.map(function (text) {
      return '<span class="qw-action-suggestion" title="当前仅展示建议，工作台未提供该操作写接口">' + escapeHtml(text) + '（建议）</span>';
    }).join('');
  }

  function renderWarningList() {
    const host = document.getElementById('qwWarningList');
    if (!host) return;
    const all = deriveWarnings();
    const list = filterWarnings(all);
    if (!list.length) {
      host.innerHTML = '<div class="qw-empty">' + (all.length ? '当前筛选下无预警/异常' : '暂无预警或指标异常') + '</div>';
      return;
    }

    host.innerHTML = '<div class="qw-warning-list">' + list.map(function (w) {
      const m = w.metric;
      const persisted = isPersistedWarning(w);
      const handled = persisted && !!w.alert.handled;
      const derivedNote = persisted ? '' : '<span class="qw-derived-note" data-derived-warning-note="1">指标实时异常 · 非预警记录</span>';
      const panelHelp = persisted ? '' : '<div class="qw-derived-help" data-derived-warning-help="1">该项由真实指标实时值派生，当前没有后端预警记录，因此不可执行“标记处理”。</div>';
      return '<div class="qw-w-item qw-level-' + w.tagCls + '-row' + (handled ? ' is-handled' : '') + '" data-warn-id="' + escapeHtml(w.id) + '">' +
        '<div class="qw-w-row1">' +
          '<span class="qw-level-tag qw-level-' + w.tagCls + '"><i class="fas fa-circle"></i>' + w.tagLabel + '</span>' +
          '<span class="qw-w-title">' + escapeHtml(w.title) + '</span>' + derivedNote +
          '<span class="qw-w-owner">' + escapeHtml(m.owner || '') + (handled ? ' · 已处理' : '') + '</span>' +
          '<span class="qw-w-btns">' +
            '<button type="button" class="qw-w-detail-btn" data-warn-id="' + escapeHtml(w.id) + '">查看详情</button>' +
            ((persisted && !handled) ? '<button type="button" class="qw-w-handle-btn" data-warn-id="' + escapeHtml(w.id) + '" aria-expanded="false">处理 <i class="fas fa-chevron-down"></i></button>' :
              '<button type="button" class="qw-w-handle-btn" data-warn-id="' + escapeHtml(w.id) + '" aria-expanded="false">查看建议 <i class="fas fa-chevron-down"></i></button>') +
          '</span>' +
        '</div>' +
        '<div class="qw-w-row2">' +
          '<span class="qw-w-meta"><label>指标</label><strong>' + escapeHtml(m.metricName) + ' ' + m.currentValue + escapeHtml(m.unit) + '</strong></span>' +
          '<span class="qw-w-meta"><label>风险线</label>' + escapeHtml(w.threshold) + '</span>' +
          '<span class="qw-w-meta"><label>趋势</label>' + escapeHtml(trendDescOf(m)) + '</span>' +
          '<span class="qw-w-meta"><label>影响</label>' + (m.affectedProjects || 0) + ' 项目 · ' + (m.affectedDemands || 0) + ' 需求</span>' +
        '</div>' +
        (m.reason ? '<div class="qw-w-reason"><label>原因</label>' + escapeHtml(m.reason) + '</div>' : '') +
        '<div class="qw-w-panel">' +
          '<div class="qw-w-panel-hd">建议处理</div>' + panelHelp +
          '<div class="qw-w-actions">' + suggestionTextHtml(m) +
            ((persisted && !handled) ? '<button type="button" class="qw-action-btn qw-handle-btn" data-handle-id="' + escapeHtml(w.id) + '">标记处理</button>' : '') +
          '</div>' +
        '</div>' +
      '</div>';
    }).join('') + '</div>';

    host.querySelectorAll('.qw-w-detail-btn').forEach(function (btn) {
      btn.addEventListener('click', function () { openWarningDetail(btn.dataset.warnId); });
    });
    host.querySelectorAll('.qw-w-handle-btn[data-warn-id]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        const item = btn.closest('.qw-w-item');
        const panel = item && item.querySelector('.qw-w-panel');
        if (!panel) return;
        const open = item.classList.toggle('is-open');
        btn.setAttribute('aria-expanded', open ? 'true' : 'false');
      });
    });
    host.querySelectorAll('.qw-handle-btn[data-handle-id]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        const id = String(btn.dataset.handleId || '').trim();
        const warning = deriveWarnings().find(function (w) { return String(w.id) === id; });
        if (!warning || !isPersistedWarning(warning)) {
          showToast('该项不是后端预警记录，未执行处理');
          return;
        }
        S.handleAlert(id).then(function () {
          showToast('已标记处理');
          return S.ensureAlertsLoaded(function () {
            renderSummaryBar();
            renderWarningList();
          });
        }).catch(function (err) {
          showToast('处理失败：' + ((err && err.message) || err));
        });
      });
    });
  }

  function openWarningDetail(id) {
    const w = deriveWarnings().find(function (x) { return String(x.id) === String(id); });
    if (!w) return;
    const m = w.metric;
    const persisted = isPersistedWarning(w);
    const abnormalHtml = (m.abnormalDemands || []).slice(0, 5).map(function (d) {
      return '<tr><td><code>' + escapeHtml(d.id) + '</code></td><td>' + escapeHtml(d.title) + '</td><td>' + escapeHtml(d.owner) + '</td><td>' + escapeHtml(d.stage) + '</td><td>' + escapeHtml(d.delayDays || '—') + '</td><td><button type="button" class="action-btn" onclick="showDemandDetail && showDemandDetail(\'' + escapeHtml(d.id) + '\')">查看</button></td></tr>';
    }).join('');

    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay qw-detail-overlay';
    overlay.innerHTML = '<div class="modal qw-detail-modal" role="dialog" aria-modal="true">' +
      '<div class="modal-hdr"><span class="qw-level-tag qw-level-' + w.tagCls + '"><i class="fas fa-circle"></i>' + w.tagLabel + '</span>' +
        '<h3 style="margin:0 0 0 8px">' + escapeHtml(w.title) + '</h3>' +
        '<button type="button" class="modal-close" aria-label="关闭预警详情"><i class="fas fa-xmark"></i></button></div>' +
      '<div class="modal-body">' +
        (persisted ? '' : '<div class="qw-derived-help" data-derived-warning-help="1">指标实时异常 · 非预警记录；当前不可执行“标记处理”。</div>') +
        '<div class="qw-detail-basic">' +
          '<div><label>指标</label>' + escapeHtml(m.metricName) + '</div>' +
          '<div><label>指标编码</label><code>' + escapeHtml(m.metricCode) + '</code></div>' +
          '<div><label>当前值</label><strong>' + m.currentValue + escapeHtml(m.unit) + '</strong></div>' +
          '<div><label>目标值</label>' + m.targetValue + escapeHtml(m.unit) + '</div>' +
          '<div><label>黄色阈值</label>' + m.warningThreshold + escapeHtml(m.unit) + '</div>' +
          '<div><label>红色阈值</label>' + m.dangerThreshold + escapeHtml(m.unit) + '</div>' +
          '<div><label>趋势</label>' + escapeHtml(trendDescOf(m)) + '</div>' +
          '<div><label>影响项目</label>' + (m.affectedProjects || 0) + ' 个</div>' +
          '<div><label>影响需求</label>' + (m.affectedDemands || 0) + ' 个</div>' +
          '<div><label>责任对象</label>' + escapeHtml(m.owner || '—') + '</div>' +
          '<div><label>最近计算</label>' + escapeHtml(m.lastCalcTime || '-') + '</div>' +
          '<div><label>数据来源</label><code>' + escapeHtml(m.source || '—') + '</code></div>' +
        '</div>' +
        '<div class="qw-detail-section"><h4>计算口径</h4><div class="qw-formula">' + escapeHtml(m.formula || '—') + '</div></div>' +
        (m.reason ? '<div class="qw-detail-section"><h4>原因分析</h4><div class="qw-formula">' + escapeHtml(m.reason) + '</div></div>' : '') +
        (abnormalHtml ? '<div class="qw-detail-section"><h4>关联异常需求（点击查看明细）</h4><table class="tbl qw-detail-tbl"><thead><tr><th style="width:120px">编号</th><th>标题</th><th style="width:100px">负责人</th><th style="width:100px">当前阶段</th><th style="width:80px">延期天数</th><th style="width:80px">操作</th></tr></thead><tbody>' + abnormalHtml + '</tbody></table></div>' : '') +
        '<div class="qw-detail-section"><h4>建议动作</h4><div class="qw-detail-actions">' + suggestionTextHtml(m) + '</div></div>' +
      '</div>' +
      '<div class="modal-footer"><button type="button" class="action-btn" data-action="close">关闭</button><button type="button" class="action-btn primary" data-action="metricDetail">查看指标明细</button></div>' +
      '</div>';
    document.body.appendChild(overlay);
    const close = function () { if (overlay.parentNode) overlay.parentNode.removeChild(overlay); };
    overlay.addEventListener('click', function (e) { if (e.target === overlay) close(); });
    overlay.querySelector('.modal-close').addEventListener('click', close);
    overlay.querySelector('[data-action="close"]').addEventListener('click', close);
    overlay.querySelector('[data-action="metricDetail"]').addEventListener('click', function () {
      close();
      if (typeof navTo === 'function') navTo('metricDetail', { metricCode: m.metricCode });
    });
  }

  function renderQualityWarning() {
    Promise.all([S.ensureMetricsLoaded(), S.ensureAlertsLoaded()]).then(function () {
      if (S.renderMetricAgileChips) S.renderMetricAgileChips();
      renderSummaryBar();
      renderWarningList();
      const old = document.getElementById('qualityWarningList');
      if (old) old.innerHTML = '';
    }).catch(function (err) {
      const host = document.getElementById('qwWarningList');
      if (host) host.innerHTML = '<div class="qw-empty">预警接口不可用：' + escapeHtml(err && err.message || err) + '</div>';
    });
  }

  window.renderQualityWarning = renderQualityWarning;
  window.openQualityWarningDetail = openWarningDetail;
  if (typeof window.syncQualityWarningFromWorkItems === 'undefined') {
    window.syncQualityWarningFromWorkItems = function () {};
  }
})();
