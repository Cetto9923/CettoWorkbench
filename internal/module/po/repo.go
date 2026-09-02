// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流阶段业需/研发需求的只读库统计与列表查询（业需范围：澄清 PM 或 QD/RD/BRA，排除 closed）。
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
	statuses            []string
	overall             *string
	parent              *string
	overallNotFive      bool // true：排除 overall=5（已完成五星评价的）
	developFinishDue    bool // true：今天 >= developFinish（且 developFinish 非空）
	deliverDateDue      bool // true：今天 >= deliverDate（且 deliverDate 非空）
	braRequired         bool // true：BRA 必须等于当前账号
	noClarify           bool // true：无 zt_demandclarify 记录
	acceptanceStage     bool // true：验收阶段复合条件
	scheduleIncomplete  bool // true：排期未完成（关键日期/QD/主研未填）
	deliverStories      bool // true：合并交付阶段独立研发需求
}

var (
	releasedOverallEmpty = "0"
	releasedParent       = "-1"
)

// mysqlStageFilters 价值流阶段 → MySQL 查询条件。
// 「released」段（PO 评价页）排除已被系统自动完成五星评价（overall=5）的记录，
// 保留 PO 真正需要看的人为评价项。
var mysqlStageFilters = map[string]mysqlStageFilter{
	"accept":         {statuses: []string{"draft", "wait", "refuse"}},
	"clarify":        {statuses: []string{"active"}, noClarify: true},
	"schedule":       {statuses: []string{"clarified"}, scheduleIncomplete: true},
	"developing":     {statuses: []string{"developing"}, developFinishDue: true},
	"testing":        {statuses: []string{"testing"}},
	"waitacceptance": {acceptanceStage: true},
	"acceptanced": {
		statuses:       []string{"acceptanced"},
		braRequired:    true,
		deliverDateDue: true,
		deliverStories: true,
	},
	"waitdeliver": {statuses: []string{"waitdeliver"}},
	"released": {
		statuses:       []string{"released"},
		overall:        &releasedOverallEmpty,
		parent:         &releasedParent,
		overallNotFive: true,
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

// DemandRow 业需列表投影（仅选择业务要展示/排序/责任追溯所需要的列，避免 SELECT *）。
// 列名与数据库列名一致（小驼峰），由 zt_demand 真实 schema 校准。
type DemandRow struct {
	ID           int     `gorm:"column:id"`
	Name         string  `gorm:"column:name"`
	Pri          string  `gorm:"column:pri"`              // '1'/'2'/'3'/'4'（字符串），空字符串表示未设
	Status       string  `gorm:"column:status"`
	AssignedTo   string  `gorm:"column:assignedTo"`        // 业务负责人
	QD           string  `gorm:"column:QD"`                // 牵头人
	RD           string  `gorm:"column:RD"`                // 主研发责任人
	BRA          string  `gorm:"column:BRA"`               // 业务需求负责人
	IsNeedFocus  string  `gorm:"column:isNeedFocus"`       // '0'/'1'
	DevelopFinish *time.Time `gorm:"column:developFinish"` // 计划开发完成日
	TestFinish    *time.Time `gorm:"column:testFinish"`    // 计划联调完成日
	VerifyFinish  *time.Time `gorm:"column:verifyFinish"`  // 计划验收完成日
	DeliverDate   *time.Time `gorm:"column:deliverDate"`   // 计划交付日
}

// StoryRow 研发需求列表投影（pri 为 tinyint 数字）。
type StoryRow struct {
	ID            int        `gorm:"column:id"`
	Title         string     `gorm:"column:title"`
	Pri           int        `gorm:"column:pri"`
	Status        string     `gorm:"column:status"`
	AssignedTo    string     `gorm:"column:assignedTo"`
	DevelopFinish *time.Time `gorm:"column:developFinish"`
	TestFinish    *time.Time `gorm:"column:testFinish"`
	VerifyFinish  *time.Time `gorm:"column:verifyFinish"`
	DeliverDate   *time.Time `gorm:"column:deliverDate"`
}

func (r *Repo) roleDemandScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	// 业需可见范围：澄清表 PM = 当前账号，或 QD/RD/BRA = 当前账号；排除已关闭
	q := r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("status NOT IN ?", []string{"closed"}).
		Where(`(
			id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR QD = ?
			OR RD = ?
			OR BRA = ?
		)`, account, account, account, account)

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
	if filter.overallNotFive {
		// 排除已被系统完成五星评价的（PO 评价页不该再出现这些项）
		q = q.Where("overall <> ?", "5")
	}
	if filter.developFinishDue {
		today := time.Now().Format("2006-01-02")
		q = q.Where("developFinish IS NOT NULL AND developFinish <= ?", today)
	}
	if filter.deliverDateDue {
		today := time.Now().Format("2006-01-02")
		q = q.Where("deliverDate IS NOT NULL AND deliverDate != '0000-00-00' AND deliverDate <= ?", today)
	}
	if filter.braRequired {
		q = q.Where("BRA = ?", account)
	}
	if filter.noClarify {
		// 等价于 (SELECT COUNT(*) FROM zt_demandclarify WHERE demand = 需求id) = 0
		q = q.Where("NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id)")
	}
	if filter.scheduleIncomplete {
		// 日期未填：NULL / 0000-00-00（DATE 不可与 '' 比较，会触发 Error 1525）；或 QD、mainDevelopers 为空
		q = q.Where(`(
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
			OR estimateLaunch IS NULL OR estimateLaunch = '0000-00-00'
			OR QD = ''
			OR mainDevelopers = ''
		)`)
	}
	return q
}

// scheduleStoryScope 排期阶段独立研发需求：非需求池、指派给当前用户、关键日期未填。
func (r *Repo) scheduleStoryScope(ctx context.Context, account string) *gorm.DB {
	return r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("assignedTo = ?", account).
		Where(`(
			developFinish IS NULL OR developFinish = '0000-00-00'
			OR testFinish IS NULL OR testFinish = '0000-00-00'
			OR verifyFinish IS NULL OR verifyFinish = '0000-00-00'
		)`)
}

// deliverStoryScope 交付阶段独立研发需求：非需求池、指派给当前用户、今天 >= deliverDate。
func (r *Repo) deliverStoryScope(ctx context.Context, account string) *gorm.DB {
	today := time.Now().Format("2006-01-02")
	return r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("assignedTo = ?", account).
		Where("deliverDate IS NOT NULL AND deliverDate != '0000-00-00' AND deliverDate <= ?", today)
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
		Select(
			"id", "name", "pri", "status",
			"assignedTo", "QD", "RD", "BRA", "isNeedFocus",
			"developFinish", "testFinish", "verifyFinish", "deliverDate",
		).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountScheduleStories 统计排期阶段独立研发需求数量。
func (r *Repo) CountScheduleStories(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.scheduleStoryScope(ctx, account).Count(&total).Error
	return total, err
}

// FindScheduleStories 查询排期阶段独立研发需求列表。
func (r *Repo) FindScheduleStories(ctx context.Context, account string) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.scheduleStoryScope(ctx, account).
		Select(
			"id", "title", "pri", "status", "assignedTo",
			"developFinish", "testFinish", "verifyFinish",
		).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountDeliverStories 统计交付阶段独立研发需求数量。
func (r *Repo) CountDeliverStories(ctx context.Context, account string) (int64, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return 0, nil
	}
	var total int64
	err := r.deliverStoryScope(ctx, account).Count(&total).Error
	return total, err
}

// FindDeliverStories 查询交付阶段独立研发需求列表。
func (r *Repo) FindDeliverStories(ctx context.Context, account string) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.deliverStoryScope(ctx, account).
		Select(
			"id", "title", "pri", "status", "assignedTo",
			"developFinish", "testFinish", "verifyFinish", "deliverDate",
		).
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
