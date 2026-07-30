// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流阶段业需的 MySQL 统计与列表查询（按 QD/RD/BRA + 阶段条件）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
)

// mysqlStageFilter 走 MySQL 的价值流阶段过滤条件。
type mysqlStageFilter struct {
	statuses         []string
	overall          *string
	parent           *string
	developFinishDue bool // true：今天 >= developFinish（且 developFinish 非空）
	noClarify        bool // true：无 zt_demandclarify 记录
	acceptanceStage  bool // true：验收阶段复合条件
}

var (
	releasedOverallEmpty = "0"
	releasedParent       = "-1"
)

// mysqlStageFilters 价值流阶段 → MySQL 查询条件。
var mysqlStageFilters = map[string]mysqlStageFilter{
	"accept":     {statuses: []string{"draft", "wait", "refuse"}},
	"clarify":    {statuses: []string{"active"}, noClarify: true},
	"developing": {statuses: []string{"developing"}, developFinishDue: true},
	"testing":    {statuses: []string{"testing"}},
	"waitacceptance": {acceptanceStage: true},
	"released": {
		statuses: []string{"released"},
		overall:  &releasedOverallEmpty,
		parent:   &releasedParent,
	},
}

// Repo PO 工作台数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// DemandRow 业需列表投影。
type DemandRow struct {
	ID   int    `gorm:"column:id"`
	Name string `gorm:"column:name"`
	Pri  string `gorm:"column:pri"`
}

func (r *Repo) roleDemandScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	q := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("(QD = ? OR RD = ? OR BRA = ?)", account, account, account)

	if filter.acceptanceStage {
		today := time.Now().Format("2006-01-02")
		// (status=testing AND 今天>=testFinish) OR (status=waitacceptance AND (RD|BRA)=账号)
		q = q.Where(`(
			(status = ? AND testFinish IS NOT NULL AND testFinish <= ?)
			OR (status = ? AND (RD = ? OR BRA = ?))
		)`, "testing", today, "waitacceptance", account, account)
		return q
	}

	q = q.Where("status IN ?", filter.statuses)
	if filter.overall != nil {
		q = q.Where("overall = ?", *filter.overall)
	}
	if filter.parent != nil {
		q = q.Where("parent != ?", *filter.parent)
	}
	if filter.developFinishDue {
		today := time.Now().Format("2006-01-02")
		q = q.Where("developFinish IS NOT NULL AND developFinish <= ?", today)
	}
	if filter.noClarify {
		// 等价于 (SELECT COUNT(*) FROM zt_demandclarify WHERE demand = 需求id) = 0
		q = q.Where("NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id)")
	}
	return q
}

func filterReady(account string, filter mysqlStageFilter) bool {
	if strings.TrimSpace(account) == "" {
		return false
	}
	if filter.acceptanceStage {
		return true
	}
	return len(filter.statuses) > 0
}

// CountRoleDemands 按阶段过滤条件统计业需数量。
func (r *Repo) CountRoleDemands(ctx context.Context, account string, filter mysqlStageFilter) (int64, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return 0, nil
	}
	var total int64
	err := r.roleDemandScope(ctx, account, filter).Count(&total).Error
	return total, err
}

// FindRoleDemands 按阶段过滤条件查询业需列表。
func (r *Repo) FindRoleDemands(ctx context.Context, account string, filter mysqlStageFilter) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var rows []DemandRow
	err := r.roleDemandScope(ctx, account, filter).
		Select("id", "name", "pri").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
