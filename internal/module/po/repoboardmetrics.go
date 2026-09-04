// =============================================================================
// 文件: internal/module/po/repoboardmetrics.go
// 模块: PO 工作台
// 类型: action
// 职责: 小组效能快照，8 个紧凑指标，按敏捷小组成员真实聚合。
//       禁止 mock：无真实数据源的指标返回空值"-"，由前端展示为 "—"。
//       SQL 一律参数化（IN (?) 占位，不拼接成员账号）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"
)

// boardTeamMetric 一个小组效能指标的数值样式定义。
type boardTeamMetric struct {
	Key    string `gorm:"-"`
	Name   string `gorm:"-"`
	Value  string `gorm:"-"`
	Target string `gorm:"-"`
	Trend  string `gorm:"-"`
	State  string `gorm:"-"` // good / warn / risk / flat
	// higherIsBetter=true 时数值越高越好；false 越低越好。
	higherIsBetter bool
	targetValue    float64
	measured       float64
	hasValue       bool
}

// boardGroupMetrics 每组返回的 8 项指标（顺序即界面 1..8）。
func boardGroupMetrics() []*BoardMetric {
	defs := []struct {
		key, name, target string
		higher            bool
	}{
		{"delivery", "交付周期", "≤30天", false},
		{"implement", "实施周期", "≤20天", false},
		{"overIteration", "超预迭代周期占比", "≤10%", false},
		{"unscheduled", "超2周未排期", "≤3个", false},
		{"gate", "质量门禁通过率", "≥90%", true},
		{"bugClose", "缺陷关闭率", "≥90%", true},
		{"bugResponse", "缺陷响应效率", "≥85%", true},
		{"onlineDelay", "上线延期数", "0个", false},
	}
	out := make([]*BoardMetric, 0, len(defs))
	for _, d := range defs {
		out = append(out, &BoardMetric{
			Key: d.key, Name: d.name, Target: d.target,
			Value: "-", Trend: "", State: "flat",
			higherIsBetter: d.higher,
		})
	}
	return out
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
	r.computeBug(ctx, ms, members)
	r.computeOnlineDelay(ctx, ms, members)
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
	sql := `SELECT
			AVG(CASE WHEN st.status='released' AND st.openedDate!='0000-00-00 00:00:00' AND st.releasedDate!='0000-00-00 00:00:00' AND st.releasedDate>=st.openedDate
				THEN DATEDIFF(st.releasedDate, st.openedDate) ELSE NULL END) AS delivery,
			AVG(CASE WHEN st.finishedDate!='0000-00-00 00:00:00' AND st.assignedDate!='0000-00-00 00:00:00' AND st.finishedDate>=st.assignedDate
				THEN DATEDIFF(st.finishedDate, st.assignedDate) ELSE NULL END) AS implement,
			COUNT(st.id) AS finishedCnt,
			SUM(CASE WHEN st.finishedDate!='0000-00-00 00:00:00' AND st.openedDate!='0000-00-00 00:00:00'
				AND st.finishedDate>=st.openedDate AND DATEDIFF(st.finishedDate, st.openedDate)>14 THEN 1 ELSE 0 END) AS overIterCnt
			FROM zt_story st
			WHERE st.deleted='0' AND (st.openedBy IN (?) OR st.assignedTo IN (?))`
	if err := r.db.WithContext(ctx).Raw(sql, members, members).Scan(&row).Error; err != nil {
		return
	}
	if row.Delivery != nil {
		setMetricValue(ms, "delivery", fmt.Sprintf("%.1f天", *row.Delivery), *row.Delivery, 40, 30)
	} else {
		metricOf(ms, "delivery").Value = "-"
	}
	if row.Implement != nil {
		setMetricValue(ms, "implement", fmt.Sprintf("%.1f天", *row.Implement), *row.Implement, 28, 20)
	} else {
		metricOf(ms, "implement").Value = "-"
	}
	if row.FinishedCnt > 0 && row.OverIterCnt >= 0 {
		pct := float64(row.OverIterCnt) * 100 / float64(row.FinishedCnt)
		setMetricValue(ms, "overIteration", fmt.Sprintf("%.1f%%", pct), pct, 20, 10)
	} else {
		metricOf(ms, "overIteration").Value = "-"
	}
}

// computeUnscheduled 超 2 周未排期：近 90 天活跃、打开超 14 天、尚无执行任务的研需数量。
func (r *Repo) computeUnscheduled(ctx context.Context, ms []*BoardMetric, members []string) {
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14).Format("2006-01-02")
	ninetyDaysAgo := now.AddDate(0, 0, -90).Format("2006-01-02")
	var n int64
	q := r.db.WithContext(ctx).Table("zt_story st").
		Where("st.deleted='0' AND st.status NOT IN ?", []string{"closed", "cancel"}).
		Where("st.openedDate >= ?", ninetyDaysAgo+" 00:00:00").
		Where("st.openedDate <= ?", twoWeeksAgo+" 23:59:59").
		Where("(st.openedBy IN ? OR st.assignedTo IN ?)", members, members).
		Where("NOT EXISTS (SELECT 1 FROM zt_task t WHERE t.story=st.id AND t.deleted='0')")
	if err := q.Count(&n).Error; err == nil {
		setMetricCount(ms, "unscheduled", n, 6, 3) // 目标 ≤3 个达标，≤6 预警
	}
}

// computeBug 缺陷关闭率 + 缺陷响应效率（小组负责/创建的 Bug 口径）。
func (r *Repo) computeBug(ctx context.Context, ms []*BoardMetric, members []string) {
	row := struct {
		Total       int64 `gorm:"column:total"`
		Closed      int64 `gorm:"column:closed"`
		Responsed   int64 `gorm:"column:responsed"`
		OpenedReali int64 `gorm:"column:openedReasonable"`
	}{}
	sql := `SELECT COUNT(*) AS total,
			SUM(CASE WHEN status='closed' THEN 1 ELSE 0 END) AS closed,
			SUM(CASE WHEN assignedDate!='0000-00-00 00:00:00' AND assignedDate>=openedDate
				AND TIMESTAMPDIFF(HOUR, openedDate, assignedDate)<=8 THEN 1 ELSE 0 END) AS responsed,
			SUM(CASE WHEN assignedDate!='0000-00-00 00:00:00' THEN 1 ELSE 0 END) AS openedReasonable
			FROM zt_bug b
			WHERE b.deleted='0' AND (b.openedBy IN (?) OR b.assignedTo IN (?))`
	if err := r.db.WithContext(ctx).Raw(sql, members, members).Scan(&row).Error; err != nil {
		return
	}
	if row.Total > 0 {
		closePct := float64(row.Closed) * 100 / float64(row.Total)
		setMetricValue(ms, "bugClose", fmt.Sprintf("%.0f%%", closePct), closePct, 85, 90)
	} else {
		metricOf(ms, "bugClose").Value = "-"
	}
	if row.OpenedReali > 0 {
		respPct := float64(row.Responsed) * 100 / float64(row.OpenedReali)
		setMetricValue(ms, "bugResponse", fmt.Sprintf("%.0f%%", respPct), respPct, 80, 85)
	} else {
		metricOf(ms, "bugResponse").Value = "-"
	}
}

// computeOnlineDelay 质效指标：上线延期数 + 质量门禁通过率（已验证交付占比近似）。
func (r *Repo) computeOnlineDelay(ctx context.Context, ms []*BoardMetric, members []string) {
	var delay int64
	if err := r.db.WithContext(ctx).Table("zt_story st").
		Where("st.deleted='0' AND st.deliverDate!='0000-00-00 00:00:00' AND st.releasedDate!='0000-00-00 00:00:00' AND st.releasedDate>st.deliverDate").
		Where("(st.openedBy IN ? OR st.assignedTo IN ?)", members, members).
		Count(&delay).Error; err == nil {
		setMetricCount(ms, "onlineDelay", delay, 2, 0)
	}
	gateRow := struct {
		Total  int64 `gorm:"column:total"`
		Passed int64 `gorm:"column:passed"`
	}{}
	sql := `SELECT COUNT(*) AS total,
			SUM(CASE WHEN st.verifiedDate!='0000-00-00 00:00:00' AND st.verifiedDate IS NOT NULL AND st.verifiedDate<>'' THEN 1 ELSE 0 END) AS passed
			FROM zt_story st
			WHERE st.deleted='0' AND (st.openedBy IN (?) OR st.assignedTo IN (?))`
	if err := r.db.WithContext(ctx).Raw(sql, members, members).Scan(&gateRow).Error; err == nil && gateRow.Total > 0 {
		gatePct := float64(gateRow.Passed) * 100 / float64(gateRow.Total)
		setMetricValue(ms, "gate", fmt.Sprintf("%.0f%%", gatePct), gatePct, 80, 90)
	} else {
		metricOf(ms, "gate").Value = "-"
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
