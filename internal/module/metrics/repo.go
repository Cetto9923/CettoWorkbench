// =============================================================================
// 文件: internal/module/metrics/repo.go
// 模块: 指标管理 (metrics)
// 类型: repo
// 职责: 禅道只读聚合；单次 SQL 拉取指标运行快照。
// 依赖: 无
// =============================================================================

package metrics

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Repo 是 metrics 模块的只读仓储：数据来自禅道 zt_story / zt_bug / zt_task。
type Repo struct{ db *gorm.DB }

// NewRepo 装配 Repo；db 为 nil 时 Snapshot 返回明确错误。
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// snapshot 是 12 条指标派生所需的原子计数（单 SQL、三个条件聚合）。
type snapshot struct {
	// 需求治理
	Stories             int64 `gorm:"column:stories"`                // 研发需求总量（deleted='0'）
	StoriesActive       int64 `gorm:"column:stories_active"`         // 进行中（status NOT IN closed/released）
	StoriesDone         int64 `gorm:"column:stories_done"`           // 已完成（status IN closed/released）
	StoriesDoneRecent   int64 `gorm:"column:stories_done_recent"`    // 近 RecentPeriodDays 天已完成（按时关闭率的分母）
	StoriesClosedOnTime int64 `gorm:"column:stories_closed_on_time"` // 同上窗口内 closedDate<=estimateLaunch

	// 研发质量
	Bugs         int64 `gorm:"column:bugs"`          // 未关闭缺陷
	BugsP1P2     int64 `gorm:"column:bugs_p1p2"`     // 未关闭且 severity IN (1,2)
	BugsTotal    int64 `gorm:"column:bugs_total"`    // 近 RecentPeriodDays 天新建缺陷（解决率分母）
	BugsResolved int64 `gorm:"column:bugs_resolved"` // 同批新建缺陷中已解决/已关闭的（解决率分子）

	// 效能 / 交付效率
	Tasks        int64 `gorm:"column:tasks"`         // 任务总量
	TasksOpen    int64 `gorm:"column:tasks_open"`    // 未完成
	TasksOverdue int64 `gorm:"column:tasks_overdue"` // 逾期（deadline<today，且非 done/closed/cancel）
}

// Snapshot 一次 SQL 拉所有指标的运行快照（三个条件聚合）。
// 所有字符串字面量（status / deleted='0' / severity）都是服务端闭合常量；
// today 由 SQL CURDATE 函数取得（无用户输入），遵循 database.md「不插值用户值」。
//
// story.closedOnTimeRate 与 bug.resolutionRate 的目录 Period 为「近 N 天」，故其口径与 SQL 对齐：
//   - 按时关闭率：取近 RecentPeriodDays 天内 closedDate 的已完成需求为分母，分子为其中 closedDate<=estimateLaunch 的。
//   - 缺陷解决率：取近 RecentPeriodDays 天内 openedDate 的新建缺陷为分母，分子为其中已解决/已关闭的。
//
// 两者都是「同一批样本的分子/分母」，分母与分子同窗口同日期列，不会退化成恒为 100%。
// 其余 Period 为「实时 / 月度」的指标不加窗口条件。
func (r *Repo) Snapshot(ctx context.Context) (snapshot, error) {
	var out snapshot
	if r == nil || r.db == nil {
		return out, fmt.Errorf("metrics database unavailable")
	}
	const query = `SELECT * FROM (
 SELECT COUNT(*) AS stories,
 COUNT(CASE WHEN status NOT IN ('closed','released') THEN 1 END) AS stories_active,
 COUNT(CASE WHEN status IN ('closed','released') THEN 1 END) AS stories_done,
 COUNT(CASE WHEN status IN ('closed','released') AND CAST(closedDate AS CHAR) NOT LIKE '0000-00-00%' AND closedDate >= DATE_SUB(CURDATE(), INTERVAL ? DAY) THEN 1 END) AS stories_done_recent,
 COUNT(CASE WHEN status IN ('closed','released') AND CAST(closedDate AS CHAR) NOT LIKE '0000-00-00%' AND CAST(estimateLaunch AS CHAR) NOT LIKE '0000-00-00%' AND closedDate >= DATE_SUB(CURDATE(), INTERVAL ? DAY) AND closedDate < DATE_ADD(DATE(estimateLaunch), INTERVAL 1 DAY) THEN 1 END) AS stories_closed_on_time
 FROM zt_story WHERE deleted = '0'
 ) s CROSS JOIN (
 SELECT COUNT(CASE WHEN status NOT IN ('closed','cancelled') THEN 1 END) AS bugs,
 COUNT(CASE WHEN status NOT IN ('closed','cancelled') AND severity IN ('1','2') THEN 1 END) AS bugs_p1p2,
 COUNT(CASE WHEN CAST(openedDate AS CHAR) NOT LIKE '0000-00-00%' AND openedDate >= DATE_SUB(CURDATE(), INTERVAL ? DAY) THEN 1 END) AS bugs_total,
 COUNT(CASE WHEN status IN ('resolved','closed') AND CAST(openedDate AS CHAR) NOT LIKE '0000-00-00%' AND openedDate >= DATE_SUB(CURDATE(), INTERVAL ? DAY) THEN 1 END) AS bugs_resolved
 FROM zt_bug WHERE deleted = '0'
 ) b CROSS JOIN (
 SELECT COUNT(*) AS tasks,
 COUNT(CASE WHEN status NOT IN ('closed','cancel') THEN 1 END) AS tasks_open,
 COUNT(CASE WHEN status NOT IN ('done','closed','cancel') AND CAST(deadline AS CHAR) NOT LIKE '0000-00-00%' AND deadline < CURDATE() THEN 1 END) AS tasks_overdue
 FROM zt_task WHERE deleted = '0'
 ) t`

	err := r.db.WithContext(ctx).Raw(query, RecentPeriodDays, RecentPeriodDays, RecentPeriodDays, RecentPeriodDays).Scan(&out).Error
	return out, err
}
