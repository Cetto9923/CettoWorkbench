// =============================================================================
// 文件: internal/module/po/repovaluestream.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流 9 阶段业需/研需的个人行动范围统计与列表查询；「全部」计数仅 Pluck id。
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
	statuses           []string
	overall            *string
	parent             *string
	developFinishDue   bool // true：今天 >= developFinish（且 developFinish 非空）
	deliverDateDue     bool // true：今天 >= deliverDate（且 deliverDate 非空）
	braRequired        bool // true：BRA 必须等于当前账号
	noClarify          bool // true：无 zt_demandclarify 记录
	acceptanceStage    bool // true：验收阶段复合条件
	publishStage       bool // true：发布阶段复合条件（waitdeliver 或已发布未评价）
	scheduleIncomplete bool // true：排期未完成（关键日期/QD/主研未填）
	deliverStories     bool // true：合并交付阶段独立研发需求
}

var (
	releasedOverallEmpty = "0"
	releasedParent       = "-1"
)

// mysqlStageFilters 价值流阶段 → MySQL 查询条件。
var mysqlStageFilters = map[string]mysqlStageFilter{
	"accept":         {statuses: []string{"draft", "wait", "refuse"}},
	"clarify":        {statuses: []string{"active"}, noClarify: true},
	"schedule":       {statuses: []string{"clarified"}, scheduleIncomplete: true},
	"developing":     {statuses: []string{"developing"}, developFinishDue: true},
	"testing":        {statuses: []string{"testing"}},
	"waitacceptance": {acceptanceStage: true},
	"acceptanced": {
		statuses:       []string{"acceptanced"},
		deliverDateDue: true,
		deliverStories: true,
	},
	"publish": {publishStage: true},
	"released": {
		statuses: []string{"released"},
		overall:  &releasedOverallEmpty,
		parent:   &releasedParent,
	},
}

// DemandRow 业需列表投影（账号字段；展示名由 Service 用用户 map 解析，避免 JOIN zt_user）。
type DemandRow struct {
	ID         int    `gorm:"column:id"`
	Name       string `gorm:"column:name"`
	Pri        string `gorm:"column:pri"`
	Status     string `gorm:"column:status"`
	AssignedTo string `gorm:"column:assignedTo"`
	QD         string `gorm:"column:QD"`
	RD         string `gorm:"column:RD"`
	BRA        string `gorm:"column:BRA"`
	PM         string `gorm:"column:pm"` // zt_demandclarify.PM，多账号逗号分隔
}

// StoryRow 研发需求列表投影。
type StoryRow struct {
	ID     int    `gorm:"column:id"`
	Title  string `gorm:"column:title"`
	Pri    int    `gorm:"column:pri"`
	Status string `gorm:"column:status"`
}

// roleDemandBase 返回当前账号可推动且未关闭的业务办理单元。
// 个人责任包括指派、派单、质量、研发、验收和澄清 PM 及星标关注，剔除无个人责任的纯 BRA；存在有效子需求时父需求只汇总，不重复统计。
func (r *Repo) roleDemandBase(ctx context.Context, account string) *gorm.DB {
	return r.db.WithContext(ctx).Table("zt_demand").
		Where("deleted = ?", "0").
		Where("status NOT IN ?", []string{"closed"}).
		Where("NOT EXISTS (SELECT 1 FROM zt_demand child WHERE child.deleted = ? AND child.parent = zt_demand.id)", "0").
		Where(`(
			assignedTo = ?
			OR distributedBy = ?
			OR QD = ?
			OR RD = ?
			OR accepter = ?
			OR id IN (SELECT demand FROM zt_demandclarify WHERE PM = ?)
			OR id IN (SELECT objectID FROM zt_starinfo WHERE objectType = 'demand' AND account = ? AND followed = '1')
		)`, account, account, account, account, account, account, account)
}

func (r *Repo) roleDemandScope(ctx context.Context, account string, filter mysqlStageFilter) *gorm.DB {
	// 业需个人行动范围由 roleDemandBase 统一限定。
	q := r.roleDemandBase(ctx, account)

	if filter.acceptanceStage {
		today := time.Now().Format("2006-01-02")
		// (status=testing AND 今天>=testFinish) OR (status=waitacceptance AND RD=账号)
		q = q.Where(`(
			(status = ? AND testFinish IS NOT NULL AND testFinish <= ?)
			OR (status = ? AND RD = ?)
		)`, "testing", today, "waitacceptance", account)
		return q
	}
	if filter.publishStage {
		// status=waitdeliver OR (status=released AND 无有效评价记录)
		q = q.Where(`(
			status = ?
			OR (
				status = ?
				AND NOT EXISTS (
					SELECT 1 FROM zt_demandappraise
					WHERE demand = zt_demand.id
						AND appraiseBy <> '' AND appraiseBy IS NOT NULL
						AND appraiseTime IS NOT NULL
				)
			)
		)`, "waitdeliver", "released")
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
	if filter.acceptanceStage || filter.publishStage {
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

// FindRoleDemandIDs 按阶段过滤条件只查业需 ID（供「全部」去重计数，避免拉全字段）。
func (r *Repo) FindRoleDemandIDs(ctx context.Context, account string, filter mysqlStageFilter) ([]int, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var ids []int
	err := r.roleDemandScope(ctx, account, filter).
		Order("zt_demand.id DESC").
		Pluck("zt_demand.id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindRoleDemands 按阶段过滤条件查询业需列表（只取账号字段，不 JOIN zt_user）。
func (r *Repo) FindRoleDemands(ctx context.Context, account string, filter mysqlStageFilter) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var rows []DemandRow
	err := r.roleDemandScope(ctx, account, filter).
		Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA,
			clarify_pm.PM AS pm`).
		Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`).
		Order("zt_demand.id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// FindRoleDemandsPaged 按阶段过滤条件分页查询业需列表（只取账号字段，不 JOIN zt_user）。
func (r *Repo) FindRoleDemandsPaged(ctx context.Context, account string, filter mysqlStageFilter, offset, limit int) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var rows []DemandRow
	q := r.roleDemandScope(ctx, account, filter).
		Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA,
			clarify_pm.PM AS pm`).
		Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`).
		Order("zt_demand.id DESC")
	if offset > 0 {
		q = q.Offset(offset)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&rows).Error
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

// FindScheduleStoryIDs 查询排期阶段独立研发需求 ID。
func (r *Repo) FindScheduleStoryIDs(ctx context.Context, account string) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := r.scheduleStoryScope(ctx, account).
		Order("id DESC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindScheduleStories 查询排期阶段独立研发需求列表。
func (r *Repo) FindScheduleStories(ctx context.Context, account string) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.scheduleStoryScope(ctx, account).
		Select("id", "title", "pri", "status").
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

// FindDeliverStoryIDs 查询交付阶段独立研发需求 ID。
func (r *Repo) FindDeliverStoryIDs(ctx context.Context, account string) ([]int, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var ids []int
	err := r.deliverStoryScope(ctx, account).
		Order("id DESC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindDeliverStories 查询交付阶段独立研发需求列表。
func (r *Repo) FindDeliverStories(ctx context.Context, account string) ([]StoryRow, error) {
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" {
		return nil, nil
	}
	var rows []StoryRow
	err := r.deliverStoryScope(ctx, account).
		Select("id", "title", "pri", "status").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// FindRoleDemandsByIDs 按 ID 列表批量查询业需详情。
func (r *Repo) FindRoleDemandsByIDs(ctx context.Context, ids []int) ([]DemandRow, error) {
	if r == nil || r.db == nil || len(ids) == 0 {
		return nil, nil
	}
	var rows []DemandRow
	err := r.db.WithContext(ctx).Table("zt_demand").
		Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA,
			clarify_pm.PM AS pm`).
		Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`).
		Where("zt_demand.id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// FindStoriesByIDs 按 ID 列表批量查询研发需求详情。
func (r *Repo) FindStoriesByIDs(ctx context.Context, ids []int) ([]StoryRow, error) {
	if r == nil || r.db == nil || len(ids) == 0 {
		return nil, nil
	}
	var rows []StoryRow
	err := r.db.WithContext(ctx).Table("zt_story").
		Select("id", "title", "pri", "status").
		Where("id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CountKPIToday 统计今日必推：今日到期 OR 已逾期 且未完成（业务需求）。
// V10.1 01 节：今日到期 OR 已逾期 且未完成。actor role scope 与 value stream 一致。
