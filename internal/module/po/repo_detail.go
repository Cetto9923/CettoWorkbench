// =============================================================================
// 文件: internal/module/po/repo_detail.go
// 模块: PO 工作台
// 类型: repository
// 职责: 业务需求统一详情高性能读模型查询（防 N+1，全部聚合 SQL）。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// DemandDetailRepo 统一详情仓储。
type DemandDetailRepo struct {
	db *gorm.DB
}

// NewDemandDetailRepo 创建仓储。
func NewDemandDetailRepo(db *gorm.DB) *DemandDetailRepo {
	return &DemandDetailRepo{db: db}
}

// DemandDetailRow 需求主记录及关联扩展字段。
type DemandDetailRow struct {
	ID             uint       `gorm:"column:id"`
	Parent         uint       `gorm:"column:parent"`
	Pool           uint       `gorm:"column:pool"`
	PoolName       string     `gorm:"column:pool_name"`
	Pri            string     `gorm:"column:pri"`
	Category       string     `gorm:"column:category"`
	Source         string     `gorm:"column:source"`
	SourceNote     string     `gorm:"column:sourceNote"`
	Name           string     `gorm:"column:name"`
	Desc           string     `gorm:"column:desc"`
	VerifyPlan     string     `gorm:"column:verifyPlan"`
	FeedbackBy     string     `gorm:"column:feedbackBy"`
	FeedbackedBy   string     `gorm:"column:feedbackedBy"`
	AssignedTo     string     `gorm:"column:assignedTo"`
	AssignedToName string     `gorm:"column:assigned_to_name"`
	QD             string     `gorm:"column:QD"`
	QDName         string     `gorm:"column:qd_name"`
	RD             string     `gorm:"column:RD"`
	BRA            string     `gorm:"column:BRA"`
	BRAName        string     `gorm:"column:bra_name"`
	MainSystem     string     `gorm:"column:mainSystem"`
	Reviewer       string     `gorm:"column:reviewer"`
	ReviewedDate   *time.Time `gorm:"column:reviewedDate"`
	Status         string     `gorm:"column:status"`
	Stage          string     `gorm:"column:stage"`
	Originator     string     `gorm:"column:originator"`
	OriginatorName string     `gorm:"column:originator_name"`
	OriginatorDept string     `gorm:"column:originator_dept"`
	CreatedBy      string     `gorm:"column:createdBy"`
	CreatedDate    *time.Time `gorm:"column:createdDate"`
	ClosedBy       string     `gorm:"column:closedBy"`
	ClosedDate     *time.Time `gorm:"column:closedDate"`
	ClosedReason   string     `gorm:"column:closedReason"`
	EditedBy       string     `gorm:"column:editedBy"`
	EditedDate     *time.Time `gorm:"column:editedDate"`
	Duration       string     `gorm:"column:duration"`
	BSA            string     `gorm:"column:BSA"`
	EstimateLaunch *time.Time `gorm:"column:estimateLaunch"`
	PublishWindow  *time.Time `gorm:"column:publishWindow"`
	DeliverDate    *time.Time `gorm:"column:deliverDate"`
	Product        string     `gorm:"column:product"`
	ProductName    string     `gorm:"column:product_name"`
}

// FindDemandDetailByID 查询单个需求及左连扩展字段。
func (r *DemandDetailRepo) FindDemandDetailByID(ctx context.Context, id uint) (*DemandDetailRow, error) {
	var row DemandDetailRow
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id, d.parent, d.pool, d.pri, d.category, d.source, d.sourceNote,
       d.name, d.desc, d.verifyPlan, d.feedbackBy, d.feedbackedBy,
       d.assignedTo, d.QD, d.RD, d.BRA, d.mainSystem, d.reviewer, d.reviewedDate,
       d.status, d.stage, d.originator, d.createdBy, d.createdDate,
       d.closedBy, d.closedDate, d.closedReason, d.editedBy, d.editedDate,
       d.duration, d.BSA, d.estimateLaunch, d.publishWindow, d.deliverDate, d.product,
       COALESCE(u_assign.realname, d.assignedTo) AS assigned_to_name,
       COALESCE(u_bra.realname, d.BRA) AS bra_name,
       COALESCE(u_qd.realname, d.QD) AS qd_name,
       COALESCE(u_orig.realname, d.originator) AS originator_name,
       COALESCE(dept.name, '—') AS originator_dept,
       COALESCE(dp.name, '—') AS pool_name,
       COALESCE(prod.name, d.product) AS product_name
FROM zt_demand d
LEFT JOIN zt_user u_assign ON d.assignedTo = u_assign.account AND u_assign.deleted = '0'
LEFT JOIN zt_user u_bra ON d.BRA = u_bra.account AND u_bra.deleted = '0'
LEFT JOIN zt_user u_qd ON d.QD = u_qd.account AND u_qd.deleted = '0'
LEFT JOIN zt_user u_orig ON d.originator = u_orig.account AND u_orig.deleted = '0'
LEFT JOIN zt_dept dept ON u_orig.dept = dept.id
LEFT JOIN zt_demandpool dp ON d.pool = dp.id AND dp.deleted = '0'
LEFT JOIN zt_product prod ON d.product = prod.id AND prod.deleted = '0'
WHERE d.id = ? AND d.deleted = '0'
LIMIT 1`, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

// DemandChildRow 子需求概要。
type DemandChildRow struct {
	ID             uint       `gorm:"column:id"`
	Parent         uint       `gorm:"column:parent"`
	Name           string     `gorm:"column:name"`
	Status         string     `gorm:"column:status"`
	Stage          string     `gorm:"column:stage"`
	AssignedTo     string     `gorm:"column:assignedTo"`
	AssignedToName string     `gorm:"column:assigned_to_name"`
	EstimateLaunch *time.Time `gorm:"column:estimateLaunch"`
}

// FindChildDemands 查询直接子需求列表。
func (r *DemandDetailRepo) FindChildDemands(ctx context.Context, parentID uint) ([]DemandChildRow, error) {
	var rows []DemandChildRow
	err := r.db.WithContext(ctx).Raw(`
SELECT d.id, d.parent, d.name, d.status, d.stage, d.assignedTo,
       COALESCE(u.realname, d.assignedTo) AS assigned_to_name,
       d.estimateLaunch
FROM zt_demand d
LEFT JOIN zt_user u ON d.assignedTo = u.account AND u.deleted = '0'
WHERE d.parent = ? AND d.deleted = '0'
ORDER BY d.id ASC`, parentID).Scan(&rows).Error
	return rows, err
}

// DemandStoryRow 关联研发需求。
type DemandStoryRow struct {
	ID             uint   `gorm:"column:id"`
	Title          string `gorm:"column:title"`
	Status         string `gorm:"column:status"`
	Stage          string `gorm:"column:stage"`
	AssignedTo     string `gorm:"column:assignedTo"`
	AssignedToName string `gorm:"column:assigned_to_name"`
	Product        uint   `gorm:"column:product"`
	ProductName    string `gorm:"column:product_name"`
}

// FindDemandStories 批量查询从该业务需求分发的研发需求 (fromDemand)。
func (r *DemandDetailRepo) FindDemandStories(ctx context.Context, demandID uint) ([]DemandStoryRow, error) {
	var rows []DemandStoryRow
	err := r.db.WithContext(ctx).Raw(`
SELECT s.id, s.title, s.status, s.stage, s.assignedTo,
       COALESCE(u.realname, s.assignedTo) AS assigned_to_name,
       s.product,
       COALESCE(prod.name, '') AS product_name
FROM zt_story s
LEFT JOIN zt_user u ON s.assignedTo = u.account AND u.deleted = '0'
LEFT JOIN zt_product prod ON s.product = prod.id AND prod.deleted = '0'
WHERE s.fromDemand = ? AND s.deleted = '0'
ORDER BY s.id ASC`, demandID).Scan(&rows).Error
	return rows, err
}

// TaskCountRow 研发任务统计。
type TaskCountRow struct {
	StoryID uint `gorm:"column:story"`
	Total   int  `gorm:"column:total"`
	Done    int  `gorm:"column:done"`
}

// FindStoryTaskCounts 批量统计研发需求下的任务进度。
func (r *DemandDetailRepo) FindStoryTaskCounts(ctx context.Context, storyIDs []uint) (map[uint]TaskCountRow, error) {
	out := make(map[uint]TaskCountRow)
	if len(storyIDs) == 0 {
		return out, nil
	}
	var rows []TaskCountRow
	err := r.db.WithContext(ctx).Raw(`
SELECT story,
       COUNT(*) AS total,
       SUM(CASE WHEN status IN ('done', 'closed') THEN 1 ELSE 0 END) AS done
FROM zt_task
WHERE story IN ? AND deleted = '0'
GROUP BY story`, storyIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.StoryID] = row
	}
	return out, nil
}

// BugCountRow 缺陷统计。
type BugCountRow struct {
	StoryID          uint `gorm:"column:story"`
	Total            int  `gorm:"column:total"`
	Active           int  `gorm:"column:active"`
	Resolved         int  `gorm:"column:resolved"`
	DeliveryBlocking int  `gorm:"column:delivery_blocking"`
}

// FindStoryBugCounts 批量统计研发需求关联的缺陷。
func (r *DemandDetailRepo) FindStoryBugCounts(ctx context.Context, storyIDs []uint) (map[uint]BugCountRow, error) {
	out := make(map[uint]BugCountRow)
	if len(storyIDs) == 0 {
		return out, nil
	}
	var rows []BugCountRow
	err := r.db.WithContext(ctx).Raw(`
SELECT story,
       COUNT(*) AS total,
       SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END) AS active,
       SUM(CASE WHEN status IN ('resolved', 'closed') THEN 1 ELSE 0 END) AS resolved,
       SUM(CASE WHEN status = 'active' AND severity IN (1, 2) THEN 1 ELSE 0 END) AS delivery_blocking
FROM zt_bug
WHERE story IN ? AND deleted = '0'
GROUP BY story`, storyIDs).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.StoryID] = row
	}
	return out, nil
}

// TestCaseCountRow 用例执行统计。
type TestCaseCountRow struct {
	TotalCount    int `gorm:"column:total_count"`
	ExecutedCount int `gorm:"column:executed_count"`
	PassedCount   int `gorm:"column:passed_count"`
}

// FindStoryTestCaseCounts 聚合统计关联研发需求的所有测试用例执行状况。
func (r *DemandDetailRepo) FindStoryTestCaseCounts(ctx context.Context, storyIDs []uint) (TestCaseCountRow, error) {
	var row TestCaseCountRow
	if len(storyIDs) == 0 {
		return row, nil
	}
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) AS total_count,
       SUM(CASE WHEN lastRunResult IS NOT NULL AND lastRunResult != '' THEN 1 ELSE 0 END) AS executed_count,
       SUM(CASE WHEN lastRunResult = 'pass' THEN 1 ELSE 0 END) AS passed_count
FROM zt_case
WHERE story IN ? AND deleted = '0'`, storyIDs).Scan(&row).Error
	return row, err
}

// DemandActionRow 禅道操作历史。
type DemandActionRow struct {
	Date      *time.Time `gorm:"column:date"`
	Actor     string     `gorm:"column:actor"`
	ActorName string     `gorm:"column:actor_name"`
	Action    string     `gorm:"column:action"`
	Extra     string     `gorm:"column:extra"`
	Comment   string     `gorm:"column:comment"`
}

// FindDemandActions 获取需求的最近操作审计记录。
func (r *DemandDetailRepo) FindDemandActions(ctx context.Context, demandID uint) ([]DemandActionRow, error) {
	var rows []DemandActionRow
	err := r.db.WithContext(ctx).Raw(`
SELECT a.date, a.actor, a.action, a.extra, a.comment,
       COALESCE(u.realname, a.actor) AS actor_name
FROM zt_action a
LEFT JOIN zt_user u ON a.actor = u.account AND u.deleted = '0'
WHERE a.objectType = 'demand' AND a.objectID = ?
ORDER BY a.date DESC
LIMIT 60`, demandID).Scan(&rows).Error
	return rows, err
}

// DemandClarifyRow 澄清扩展说明。
type DemandClarifyRow struct {
	Product           string     `gorm:"column:product"`
	ProductName       string     `gorm:"column:product_name"`
	PM                string     `gorm:"column:PM"`
	SystemClarifyDesc string     `gorm:"column:systemClarifyDesc"`
	DevEnd            *time.Time `gorm:"column:devEnd"`
	TestEnd           *time.Time `gorm:"column:testEnd"`
}

// FindDemandClarifications 查询系统/产品维度的澄清记录。
func (r *DemandDetailRepo) FindDemandClarifications(ctx context.Context, demandID uint) ([]DemandClarifyRow, error) {
	var rows []DemandClarifyRow
	err := r.db.WithContext(ctx).Raw(`
SELECT dc.product, dc.PM, dc.systemClarifyDesc, dc.devEnd, dc.testEnd,
       COALESCE(p.name, dc.product) AS product_name
FROM zt_demandclarify dc
LEFT JOIN zt_product p ON dc.product = p.id AND p.deleted = '0'
WHERE dc.demand = ?
ORDER BY dc.id ASC`, demandID).Scan(&rows).Error
	return rows, err
}

// DemandFileRow 附件行。
type DemandFileRow struct {
	ID        uint       `gorm:"column:id"`
	Title     string     `gorm:"column:title"`
	Size      int        `gorm:"column:size"`
	AddedDate *time.Time `gorm:"column:addedDate"`
	Extension string     `gorm:"column:extension"`
}

// FindDemandFiles 查询关联附件。
func (r *DemandDetailRepo) FindDemandFiles(ctx context.Context, demandID uint) ([]DemandFileRow, error) {
	var rows []DemandFileRow
	err := r.db.WithContext(ctx).Raw(`
SELECT f.id, f.title, f.size, f.addedDate, f.extension
FROM zt_file f
WHERE f.objectType = 'demand' AND f.objectID = ? AND f.deleted = '0'
ORDER BY f.id DESC`, demandID).Scan(&rows).Error
	return rows, err
}
