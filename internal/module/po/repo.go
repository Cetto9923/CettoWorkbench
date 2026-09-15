// =============================================================================
// 文件: internal/module/po/repo.go
// 模块: PO 工作台
// 类型: action
// 职责: 价值流阶段业需/研发需求的只读库统计与列表查询（业需范围：澄清 PM 或 QD/RD/BRA，排除 closed；「全部」计数仅 Pluck id）。
// 依赖: internal/model
//       internal/model/zentao
// =============================================================================

package po

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"workbench/internal/model"
	zentaomodel "workbench/internal/model/zentao"
)

// mysqlStageFilter 走 MySQL 的价值流阶段过滤条件。
type mysqlStageFilter struct {
	statuses           []string
	statusOrder        []string // 非空时按该顺序排 status，其次 id DESC（受理：待评审→已驳回→草稿）
	overall            *string
	parent             *string
	developFinishDue   bool // true：今天 >= developFinish（且 developFinish 非空）
	deliverDateDue     bool // true：今天 >= deliverDate（且 deliverDate 非空）
	braRequired        bool // true：BRA 必须等于当前账号
	noClarify          bool // true：无 zt_demandclarify 记录
	acceptanceStage    bool // true：验收阶段复合条件
	scheduleIncomplete bool // true：排期未完成（关键日期/QD/主研未填）
	deliverStories     bool // true：合并交付阶段独立研发需求
}

var (
	releasedOverallEmpty = "0"
	releasedParent       = "-1"
)

// mysqlStageFilters 价值流阶段 → MySQL 查询条件。
var mysqlStageFilters = map[string]mysqlStageFilter{
	"accept": {
		statuses:    []string{"draft", "wait", "refuse"},
		statusOrder: []string{"wait", "refuse", "draft"}, // 待评审 → 已驳回 → 草稿
	},
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
	"publish": {statuses: []string{"waitdeliver"}},
	"released": {
		statuses: []string{"released"},
		overall:  &releasedOverallEmpty,
		parent:   &releasedParent,
	},
}

// Repo PO 工作台数据访问。
type Repo struct {
	// db 只读备库：首页列表/统计走这里，避免压主库。
	db *gorm.DB
	// writeDB 主库：评审等写操作必须走这里。备库失败时可能为 nil。
	writeDB *gorm.DB
}

// NewRepo 创建 Repo。readDB 用于查询；writeDB 用于写入（可与 readDB 相同）。
func NewRepo(readDB, writeDB *gorm.DB) *Repo {
	return &Repo{db: readDB, writeDB: writeDB}
}

// DemandRow 业需列表投影（账号字段；展示名由 Service 用用户 map 解析，避免 JOIN zt_user）。
type DemandRow struct {
	ID         int    `gorm:"column:id"`
	Name       string `gorm:"column:name"`
	Pri        string `gorm:"column:pri"`
	Status     string `gorm:"column:status"`
	CreatedBy  string `gorm:"column:createdBy"`
	AssignedTo string `gorm:"column:assignedTo"`
	QD         string `gorm:"column:QD"`
	RD         string `gorm:"column:RD"`
	BRA        string `gorm:"column:BRA"`
	MainSystem string `gorm:"column:mainSystem"` // 主系统产品 ID（字符串）
	PM         string `gorm:"column:pm"`         // zt_demandclarify.PM，多账号逗号分隔
}

// StoryRow 研发需求列表投影。
type StoryRow struct {
	ID     int    `gorm:"column:id"`
	Title  string `gorm:"column:title"`
	Pri    int    `gorm:"column:pri"`
	Status string `gorm:"column:status"`
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
			(status = ? AND `+dateSetExpr("testFinish")+` AND testFinish <= ?)
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
		q = q.Where(dateSetExpr("developFinish")+" AND developFinish <= ?", today)
	}
	if filter.deliverDateDue {
		today := time.Now().Format("2006-01-02")
		q = q.Where(dateSetExpr("deliverDate")+" AND deliverDate <= ?", today)
	}
	if filter.braRequired {
		q = q.Where("BRA = ?", account)
	}
	if filter.noClarify {
		// 等价于 (SELECT COUNT(*) FROM zt_demandclarify WHERE demand = 需求id) = 0
		q = q.Where("NOT EXISTS (SELECT 1 FROM zt_demandclarify dc WHERE dc.demand = zt_demand.id)")
	}
	if filter.scheduleIncomplete {
		// 日期未填：NULL / 零日期；或 QD、mainDevelopers 为空
		q = q.Where("(" + strings.Join([]string{
			dateUnsetExpr("developFinish"),
			dateUnsetExpr("testFinish"),
			dateUnsetExpr("verifyFinish"),
			dateUnsetExpr("estimateLaunch"),
			"QD = ''",
			"mainDevelopers = ''",
		}, " OR ") + ")")
	}
	return q
}

// scheduleStoryScope 排期阶段独立研发需求：非需求池、指派人或 ReqM 为当前用户、关键日期未填。
func (r *Repo) scheduleStoryScope(ctx context.Context, account string) *gorm.DB {
	return r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("(assignedTo = ? OR ReqM = ?)", account, account).
		Where("(" + strings.Join([]string{
			dateUnsetExpr("developFinish"),
			dateUnsetExpr("testFinish"),
			dateUnsetExpr("verifyFinish"),
		}, " OR ") + ")")
}

// deliverStoryScope 交付阶段独立研发需求：非需求池、指派人或 ReqM 为当前用户、今天 >= deliverDate。
func (r *Repo) deliverStoryScope(ctx context.Context, account string) *gorm.DB {
	today := time.Now().Format("2006-01-02")
	return r.db.WithContext(ctx).Table("zt_story").
		Where("deleted = ?", "0").
		Where("IFNULL(sourceType, '') != ?", "demandpool").
		Where("type = ?", "story").
		Where("(assignedTo = ? OR ReqM = ?)", account, account).
		Where(dateSetExpr("deliverDate")+" AND deliverDate <= ?", today)
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

// FindRoleDemandIDs 按阶段过滤条件只查业需 ID（供「全部」去重计数，避免拉全字段）。
func (r *Repo) FindRoleDemandIDs(ctx context.Context, account string, filter mysqlStageFilter) ([]int, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	var ids []int
	err := applyDemandListOrder(r.roleDemandScope(ctx, account, filter), filter).
		Pluck("zt_demand.id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FindRoleDemands 按阶段过滤条件查询业需列表（只取账号字段，不 JOIN zt_user）。
// limit>0 时应用 LIMIT/OFFSET；limit<=0 表示不分页拉全量（供合并列表场景）。
func (r *Repo) FindRoleDemands(ctx context.Context, account string, filter mysqlStageFilter, limit, offset int) ([]DemandRow, error) {
	if r == nil || r.db == nil || !filterReady(account, filter) {
		return nil, nil
	}
	q := applyDemandListOrder(
		r.roleDemandScope(ctx, account, filter).
			Select(`zt_demand.id, zt_demand.name, zt_demand.pri, zt_demand.status, zt_demand.createdBy,
			zt_demand.assignedTo, zt_demand.QD, zt_demand.RD, zt_demand.BRA, zt_demand.mainSystem,
			clarify_pm.PM AS pm`).
			Joins(`LEFT JOIN (
			SELECT demand, GROUP_CONCAT(PM) AS PM
			FROM zt_demandclarify
			WHERE PM IS NOT NULL AND PM <> ''
			GROUP BY demand
		) AS clarify_pm ON clarify_pm.demand = zt_demand.id`),
		filter,
	)
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		q = q.Limit(limit).Offset(offset)
	}
	var rows []DemandRow
	err := q.Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// applyDemandListOrder 列表排序：有 statusOrder 时按自定义状态序，否则 id DESC。
// CASE + ? 占位，兼容 OceanBase；同状态内仍 id DESC。
func applyDemandListOrder(q *gorm.DB, filter mysqlStageFilter) *gorm.DB {
	if len(filter.statusOrder) == 0 {
		return q.Order("zt_demand.id DESC")
	}
	var b strings.Builder
	b.WriteString("CASE zt_demand.status")
	args := make([]any, 0, len(filter.statusOrder))
	for i, st := range filter.statusOrder {
		b.WriteString(" WHEN ? THEN ")
		b.WriteString(strconv.Itoa(i + 1))
		args = append(args, st)
	}
	b.WriteString(" ELSE 999 END ASC, zt_demand.id DESC")
	return q.Order(clause.OrderBy{
		Expression: clause.Expr{SQL: b.String(), Vars: args},
	})
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

// FindMaxTesttaskIDByProducts 按产品取最大测试单 id（ORDER BY id DESC LIMIT 1）。
func (r *Repo) FindMaxTesttaskIDByProducts(ctx context.Context, productIDs []uint) (map[uint]uint, error) {
	out := make(map[uint]uint)
	if r == nil || r.db == nil || len(productIDs) == 0 {
		return out, nil
	}
	uniq := make([]uint, 0, len(productIDs))
	seen := make(map[uint]struct{}, len(productIDs))
	for _, id := range productIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	for _, productID := range uniq {
		var taskID uint
		err := r.db.WithContext(ctx).Model(&zentaomodel.ZtTesttask{}).
			Select("id").
			Where("product = ? AND deleted = ?", productID, "0").
			Order("id DESC").
			Limit(1).
			Scan(&taskID).Error
		if err != nil {
			return nil, err
		}
		if taskID > 0 {
			out[productID] = taskID
		}
	}
	return out, nil
}

// ListVersionWindows 查询上线时间（releaseDate）>= 今天的未删除窗口（zt_versionwindow）。
func (r *Repo) ListVersionWindows(ctx context.Context) ([]model.VersionWindow, error) {
	var rows []model.VersionWindow
	if err := r.db.WithContext(ctx).
		Where("releaseDate >= CURDATE()").
		Order("releaseDate ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// dateUnsetExpr 判断 DATE 列未填（NULL 或零日期）。
// 不能写 col = '0000-00-00'：MySQL 8 / OceanBase 在 NO_ZERO_DATE 下会把字面量转 DATE，触发 Error 1525。
func dateUnsetExpr(col string) string {
	return col + " IS NULL OR CAST(" + col + " AS CHAR) LIKE '0000-00-00%'"
}

// dateSetExpr 判断 DATE 列已填有效日期（非 NULL、非零日期）。
func dateSetExpr(col string) string {
	return col + " IS NOT NULL AND CAST(" + col + " AS CHAR) NOT LIKE '0000-00-00%'"
}
