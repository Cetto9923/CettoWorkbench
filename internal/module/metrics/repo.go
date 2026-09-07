// =============================================================================
// 文件: internal/module/metrics/repo.go
// 模块: 指标管理 (metrics)
// 类型: repo
// 职责: 禅道只读聚合；一次 SQL 拉取所有指标的运行快照，
//       不引入 query-per-metric fan-out（database.md）。
// 依赖: 无
// =============================================================================

package metrics

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Repo 是 metrics 模块的只读仓储：全部数据来自禅道 zt_story / zt_bug / zt_task。
// 不写本地状态；不引入 schema 变更；不重新解释业务。
type Repo struct{ db *gorm.DB }

// NewRepo 装配 Repo；db 为 nil 时 Snapshot 返回明确错误。
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// snapshot 是 12 条指标派生所需的全部原子计数。
// 一行 SQL 拉齐（11 个 COUNT 子查询），避免 query-per-metric fan-out。
type snapshot struct {
	// 需求治理
	Stories             int64 `gorm:"column:stories"`                // 研发需求总量（deleted='0'）
	StoriesActive       int64 `gorm:"column:stories_active"`         // 进行中（status NOT IN closed/released）
	StoriesDone         int64 `gorm:"column:stories_done"`           // 已完成（status IN closed/released）
	StoriesClosedOnTime int64 `gorm:"column:stories_closed_on_time"` // 已完成中 closedDate<=estimateLaunch

	// 研发质量
	Bugs         int64 `gorm:"column:bugs"`          // 未关闭缺陷
	BugsP1P2     int64 `gorm:"column:bugs_p1p2"`     // 未关闭且 severity IN (1,2)
	BugsTotal    int64 `gorm:"column:bugs_total"`    // 缺陷总数
	BugsResolved int64 `gorm:"column:bugs_resolved"` // 已解决/已关闭

	// 效能 / 交付效率
	Tasks        int64 `gorm:"column:tasks"`         // 任务总量
	TasksOpen    int64 `gorm:"column:tasks_open"`    // 未完成
	TasksOverdue int64 `gorm:"column:tasks_overdue"` // 逾期（deadline<today）
}

// Snapshot 一次 SQL 拉所有指标的运行快照（11 个 COUNT 子查询）。
// 所有字符串字面量（status / deleted='0' / severity）都是服务端闭合常量；
// today 由 SQL DATE 函数取得（无用户输入），遵循 database.md「不插值用户值」。
func (r *Repo) Snapshot(ctx context.Context) (snapshot, error) {
	var out snapshot
	if r == nil || r.db == nil {
		return out, fmt.Errorf("metrics database unavailable")
	}
	const query = `SELECT
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0') AS stories,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status NOT IN ('closed','released')) AS stories_active,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status IN ('closed','released')) AS stories_done,
		(SELECT COUNT(*) FROM zt_story WHERE deleted='0' AND status IN ('closed','released') AND estimateLaunch IS NOT NULL AND estimateLaunch != '0000-00-00' AND DATE(closedDate) <= DATE(estimateLaunch)) AS stories_closed_on_time,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status NOT IN ('closed','cancelled')) AS bugs,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status NOT IN ('closed','cancelled') AND severity IN ('1','2')) AS bugs_p1p2,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0') AS bugs_total,
		(SELECT COUNT(*) FROM zt_bug WHERE deleted='0' AND status IN ('resolved','closed')) AS bugs_resolved,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0') AS tasks,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0' AND status NOT IN ('closed','cancel')) AS tasks_open,
		(SELECT COUNT(*) FROM zt_task WHERE deleted='0' AND status NOT IN ('closed','cancel') AND deadline IS NOT NULL AND deadline != '0000-00-00' AND DATE(deadline) < CURDATE()) AS tasks_overdue`
	err := r.db.WithContext(ctx).Raw(query).Scan(&out).Error
	return out, err
}

// RadarSummary 直接复用 Snapshot（一次 SQL），不引入第二次查询。
//
// 设计依据（database.md）：radar 与 manage 共享同一份指标定义与运行快照，
// 仅派生逻辑不同（manage = 单条维度；radar = 5 分类聚合）。第二次 fan-out
// 不会带来新信息，只会把 11 个 COUNT 重算一遍，因此 RadarSummary = Snapshot
// + 明确语义的别名，让 Handler/Service 拿到「这是给 radar 用的快照」的契约。
func (r *Repo) RadarSummary(ctx context.Context) (snapshot, error) {
	return r.Snapshot(ctx)
}
