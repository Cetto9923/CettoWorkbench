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

// snapshot 是 12 条指标派生所需的原子计数（单 SQL、11 个 COUNT 子查询）。
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

// Snapshot 一次 SQL 拉所有指标的运行快照（11 个 COUNT 子查询）。
// 所有字符串字面量（status / deleted='0' / severity）都是服务端闭合常量；
// today 由 SQL DATE 函数取得（无用户输入），遵循 database.md「不插值用户值」。
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
	const query = `SELECT
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0') AS stories,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status NOT IN ('closed','released')) AS stories_active,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status IN ('closed','released')) AS stories_done,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status IN ('closed','released') AND CAST(closedDate AS CHAR) NOT LIKE '0000-00-00%' AND DATE(closedDate) >= DATE_SUB(CURDATE(), INTERVAL ? DAY)) AS stories_done_recent,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status IN ('closed','released') AND estimateLaunch IS NOT NULL AND CAST(estimateLaunch AS CHAR) NOT LIKE '0000-00-00%' AND CAST(closedDate AS CHAR) NOT LIKE '0000-00-00%' AND DATE(closedDate) >= DATE_SUB(CURDATE(), INTERVAL ? DAY) AND DATE(closedDate) <= DATE(estimateLaunch)) AS stories_closed_on_time,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status NOT IN ('closed','cancelled')) AS bugs,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status NOT IN ('closed','cancelled') AND severity IN ('1','2')) AS bugs_p1p2,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND CAST(openedDate AS CHAR) NOT LIKE '0000-00-00%' AND DATE(openedDate) >= DATE_SUB(CURDATE(), INTERVAL ? DAY)) AS bugs_total,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status IN ('resolved','closed') AND CAST(openedDate AS CHAR) NOT LIKE '0000-00-00%' AND DATE(openedDate) >= DATE_SUB(CURDATE(), INTERVAL ? DAY)) AS bugs_resolved,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0') AS tasks,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0' AND status NOT IN ('closed','cancel')) AS tasks_open,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0' AND status NOT IN ('done','closed','cancel') AND deadline IS NOT NULL AND CAST(deadline AS CHAR) NOT LIKE '0000-00-00%' AND DATE(deadline) < CURDATE()) AS tasks_overdue`
	err := r.db.WithContext(ctx).Raw(query, RecentPeriodDays, RecentPeriodDays, RecentPeriodDays, RecentPeriodDays).Scan(&out).Error
	return out, err
}
