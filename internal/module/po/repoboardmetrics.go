// =============================================================================
// 文件: internal/module/po/repoboardmetrics.go
// 模块: PO 工作台
// 类型: action
// 职责: 小组效能快照，当前看板展示的 4 个交付节奏指标，按敏捷小组成员真实聚合。
//       禁止 mock：无真实数据源的指标返回空值"-"，由前端展示为 "—"。
//       SQL 一律参数化（IN (?) 占位，不拼接成员账号）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"

	"workbench/internal/module/metrics"
)

// boardMetricCodes 看板 4 项指标与 metrics 目录条目的对应关系。
// 名称、达标线文案、达标/危险阈值一律读 catalog（metrics.SpecOf），看板不再另写一份魔法数。
var boardMetricCodes = []struct{ key, code string }{
	{"delivery", "delivery.cycle"},
	{"implement", "implement.cycle"},
	{"overIteration", "story.overIteration"},
	{"unscheduled", "story.unscheduled"},
}

// boardGroupMetrics 返回当前看板展示的 4 项指标。
func boardGroupMetrics() []*BoardMetric {
	out := make([]*BoardMetric, 0, len(boardMetricCodes))
	for _, d := range boardMetricCodes {
		m := &BoardMetric{Key: d.key, Value: "-", Trend: "", State: "flat"}
		if spec, ok := metrics.SpecOf(d.code); ok {
			m.Name, m.Target = spec.Name, spec.Target
			m.higherIsBetter = false // 4 项在目录中方向均为 down
		}
		out = append(out, m)
	}
	return out
}

// boardSpec 取看板 key 对应的目录定义；不存在时返回零值（调用方按阈值缺失处理）。
func boardSpec(key string) metrics.Spec {
	for _, d := range boardMetricCodes {
		if d.key == key {
			spec, _ := metrics.SpecOf(d.code)
			return spec
		}
	}
	return metrics.Spec{}
}

// FindBoardTeamMetrics 返回某小组的真实效能指标；teamgroupID=0 时返回空（前端展示 Empty State）。
func (r *Repo) FindBoardTeamMetrics(ctx context.Context, teamgroupID uint) ([]*BoardMetric, error) {
	if r == nil || r.db == nil || teamgroupID == 0 {
		return nil, nil
	}
	members, err := r.FindBoardTeamgroupMembers(ctx, teamgroupID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, nil
	}
	ms := boardGroupMetrics()
	r.computeCycles(ctx, ms, members)
	r.computeUnscheduled(ctx, ms, members)
	return ms, nil
}

// computeCycles 交付/实施周期 + 超预迭代周期占比（小组完成研发需求的时间度量）。
func (r *Repo) computeCycles(ctx context.Context, ms []*BoardMetric, members []string) {
	row := struct {
		Delivery    *float64 `gorm:"column:delivery"`
		Implement   *float64 `gorm:"column:implement"`
		FinishedCnt int64    `gorm:"column:finishedCnt"`
		OverIterCnt int64    `gorm:"column:overIterCnt"`
	}{}
	// zt_story 没有 finishedDate（该列在 zt_task）。完成研发交付以 releasedDate 为准，
	// 与交付周期同一终点；实施周期从指派（进入实施）算到发布，不把任务完成日挪来充当需求事件。
	sql := `SELECT
			AVG(CASE WHEN st.status='released' AND st.openedDate!='0000-00-00 00:00:00' AND st.releasedDate!='0000-00-00 00:00:00' AND st.releasedDate>=st.openedDate
				THEN DATEDIFF(st.releasedDate, st.openedDate) ELSE NULL END) AS delivery,
			AVG(CASE WHEN st.status='released' AND st.releasedDate!='0000-00-00 00:00:00' AND st.assignedDate!='0000-00-00 00:00:00' AND st.releasedDate>=st.assignedDate
				THEN DATEDIFF(st.releasedDate, st.assignedDate) ELSE NULL END) AS implement,
			SUM(CASE WHEN st.status='released' AND st.releasedDate!='0000-00-00 00:00:00' THEN 1 ELSE 0 END) AS finishedCnt,
			SUM(CASE WHEN st.status='released' AND st.releasedDate!='0000-00-00 00:00:00' AND st.openedDate!='0000-00-00 00:00:00'
				AND st.releasedDate>=st.openedDate AND DATEDIFF(st.releasedDate, st.openedDate)>? THEN 1 ELSE 0 END) AS overIterCnt
			FROM zt_story st
			WHERE st.deleted='0' AND (st.openedBy IN (?) OR st.assignedTo IN (?))`
	if err := r.db.WithContext(ctx).Raw(sql, metrics.OverIterationDays, members, members).Scan(&row).Error; err != nil {
		return
	}
	if row.Delivery != nil {
		s := boardSpec("delivery")
		setMetricValue(ms, "delivery", fmt.Sprintf("%.1f天", *row.Delivery), *row.Delivery, s.DangerValue, s.TargetValue)
	} else {
		metricOf(ms, "delivery").Value = "-"
	}
	if row.Implement != nil {
		s := boardSpec("implement")
		setMetricValue(ms, "implement", fmt.Sprintf("%.1f天", *row.Implement), *row.Implement, s.DangerValue, s.TargetValue)
	} else {
		metricOf(ms, "implement").Value = "-"
	}
	if row.FinishedCnt > 0 && row.OverIterCnt >= 0 {
		pct := float64(row.OverIterCnt) * 100 / float64(row.FinishedCnt)
		s := boardSpec("overIteration")
		setMetricValue(ms, "overIteration", fmt.Sprintf("%.1f%%", pct), pct, s.DangerValue, s.TargetValue)
	} else {
		metricOf(ms, "overIteration").Value = "-"
	}
}

// computeUnscheduled 超 2 周未排期：口径与 metrics 目录 story.unscheduled 一致——
// 业务评审通过（reviewedDate）超过 metrics.UnscheduledClarifyDays 天、且尚未进入已澄清及其后阶段的需求数。
//
// 假设（表字段所限）：目录的「未澄清」没有独立状态列，这里取 status 停留在受理/澄清阶段
// （draft/wait/active）作为未澄清的等价判定，reviewedDate 作为「评审通过」时点，
// 成员归属沿用看板其余指标的口径（QD/RD/BRA/createdBy 命中小组任一成员）。
func (r *Repo) computeUnscheduled(ctx context.Context, ms []*BoardMetric, members []string) {
	clarifyBefore := time.Now().AddDate(0, 0, -metrics.UnscheduledClarifyDays).Format("2006-01-02")
	var n int64
	q := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted='0' AND status IN ?", []string{"draft", "wait", "active"}).
		Where("reviewedDate IS NOT NULL AND CAST(reviewedDate AS CHAR) NOT LIKE '0000-00-00%' AND DATE(reviewedDate) <= ?", clarifyBefore).
		Where("(QD IN ? OR RD IN ? OR BRA IN ? OR createdBy IN ?)", members, members, members, members)
	if err := q.Count(&n).Error; err == nil {
		s := boardSpec("unscheduled")
		setMetricCount(ms, "unscheduled", n, int64(s.DangerValue), int64(s.TargetValue))
	}
}

// metricOf 按 key 取指标，便于前向填充。
func metricOf(ms []*BoardMetric, key string) *BoardMetric {
	for _, m := range ms {
		if m.Key == key {
			return m
		}
	}
	return nil
}

// setMetricValue 记录真实值并按阈值折算 state。
// goal 为"达标"严格线、warnLimit 为"预警"宽松线；higherIsBetter=true 时越高越好，false 越低越好。
func setMetricValue(ms []*BoardMetric, key, display string, v, warnLimit, goal float64) {
	m := metricOf(ms, key)
	if m == nil {
		return
	}
	m.Value = display
	m.hasValue = true
	m.measured = v
	if m.higherIsBetter {
		switch {
		case v >= goal:
			m.State = "good"
		case v >= warnLimit:
			m.State = "warn"
		default:
			m.State = "risk"
		}
		return
	}
	switch {
	case v <= goal:
		m.State = "good"
	case v <= warnLimit:
		m.State = "warn"
	default:
		m.State = "risk"
	}
}

// setMetricCount 计数类指标（整型）。goal 为达标严格线、warnLimit 为预警宽松线。
func setMetricCount(ms []*BoardMetric, key string, v int64, warnLimit, goal int64) {
	m := metricOf(ms, key)
	if m == nil {
		return
	}
	m.Value = fmt.Sprintf("%d个", v)
	m.measured = float64(v)
	m.hasValue = true
	if m.higherIsBetter {
		if float64(v) >= float64(goal) {
			m.State = "good"
		} else if float64(v) >= float64(warnLimit) {
			m.State = "warn"
		} else {
			m.State = "risk"
		}
		return
	}
	if float64(v) <= float64(goal) {
		m.State = "good"
	} else if float64(v) <= float64(warnLimit) {
		m.State = "warn"
	} else {
		m.State = "risk"
	}
}
