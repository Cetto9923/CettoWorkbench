// =============================================================================
// 文件: internal/module/po/repo_detail.go
// 模块: PO 工作台
// 类型: repository
// 职责: 业务需求详情、关联执行数据及历史记录查询。
// 依赖: gorm.io/gorm
// =============================================================================

package po

import (
	"context"
	"fmt"
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
	ID               uint       `gorm:"column:id"`
	Parent           uint       `gorm:"column:parent"`
	Pool             uint       `gorm:"column:pool"`
	PoolName         string     `gorm:"column:pool_name"`
	Pri              string     `gorm:"column:pri"`
	Category         string     `gorm:"column:category"`
	Source           string     `gorm:"column:source"`
	SourceNote       string     `gorm:"column:sourceNote"`
	Name             string     `gorm:"column:name"`
	Desc             string     `gorm:"column:desc"`
	VerifyPlan       string     `gorm:"column:verifyPlan"`
	FeedbackBy       string     `gorm:"column:feedbackBy"`
	FeedbackedBy     string     `gorm:"column:feedbackedBy"`
	AssignedTo       string     `gorm:"column:assignedTo"`
	AssignedToName   string     `gorm:"column:assigned_to_name"`
	QD               string     `gorm:"column:QD"`
	QDName           string     `gorm:"column:qd_name"`
	RD               string     `gorm:"column:RD"`
	BRA              string     `gorm:"column:BRA"`
	BRAName          string     `gorm:"column:bra_name"`
	MainSystem       string     `gorm:"column:mainSystem"`
	Reviewer         string     `gorm:"column:reviewer"`
	ReviewedDate     *time.Time `gorm:"column:reviewedDate"`
	Status           string     `gorm:"column:status"`
	Stage            string     `gorm:"column:stage"`
	Originator       string     `gorm:"column:originator"`
	OriginatorName   string     `gorm:"column:originator_name"`
	OriginatorDept   string     `gorm:"column:originator_dept"`
	CreatedBy        string     `gorm:"column:createdBy"`
	CreatedDate      *time.Time `gorm:"column:createdDate"`
	ClosedBy         string     `gorm:"column:closedBy"`
	ClosedDate       *time.Time `gorm:"column:closedDate"`
	ClosedReason     string     `gorm:"column:closedReason"`
	EditedBy         string     `gorm:"column:editedBy"`
	EditedDate       *time.Time `gorm:"column:editedDate"`
	Duration         string     `gorm:"column:duration"`
	BSA              string     `gorm:"column:BSA"`
	EstimateLaunch   *time.Time `gorm:"column:estimateLaunch"`
	PublishWindow    *time.Time `gorm:"column:publishWindow"`
	DeliverDate      *time.Time `gorm:"column:deliverDate"`
	Product          string     `gorm:"column:product"`
	ProductName      string     `gorm:"column:product_name"`
	Accepter         string     `gorm:"column:accepter"`
	AccepterName     string     `gorm:"column:accepter_name"`
	ReviewerName     string     `gorm:"column:reviewer_name"`
	MainSystemName   string     `gorm:"column:main_system_name"`
	EstimateDelivery int        `gorm:"column:estimateDelivery"`
	DevelopFinish    *time.Time `gorm:"column:developFinish"`
	TestFinish       *time.Time `gorm:"column:testFinish"`
	VerifyFinish     *time.Time `gorm:"column:verifyFinish"`
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
       d.accepter, d.estimateDelivery, d.developFinish, d.testFinish, d.verifyFinish,
       COALESCE(u_assign.realname, d.assignedTo) AS assigned_to_name,
       COALESCE(u_bra.realname, d.BRA) AS bra_name,
       COALESCE(u_qd.realname, d.QD) AS qd_name,
       COALESCE(u_orig.realname, d.originator) AS originator_name,
       COALESCE(u_acc.realname, d.accepter) AS accepter_name,
       COALESCE(u_rev.realname, d.reviewer) AS reviewer_name,
       COALESCE(dept.name, '—') AS originator_dept,
       COALESCE(dp.name, '—') AS pool_name,
       COALESCE(prod.name, d.product) AS product_name,
       COALESCE(sys.name, d.mainSystem) AS main_system_name
FROM zt_demand d
LEFT JOIN zt_user u_assign ON d.assignedTo = u_assign.account AND u_assign.deleted = '0'
LEFT JOIN zt_user u_bra ON d.BRA = u_bra.account AND u_bra.deleted = '0'
LEFT JOIN zt_user u_qd ON d.QD = u_qd.account AND u_qd.deleted = '0'
LEFT JOIN zt_user u_orig ON d.originator = u_orig.account AND u_orig.deleted = '0'
LEFT JOIN zt_user u_acc ON d.accepter = u_acc.account AND u_acc.deleted = '0'
LEFT JOIN zt_user u_rev ON d.reviewer = u_rev.account AND u_rev.deleted = '0'
LEFT JOIN zt_dept dept ON u_orig.dept = dept.id
LEFT JOIN zt_demandpool dp ON d.pool = dp.id AND dp.deleted = '0'
LEFT JOIN zt_product prod ON d.product = prod.id AND prod.deleted = '0'
LEFT JOIN zt_product sys ON sys.id = CAST(NULLIF(d.mainSystem, '') AS UNSIGNED) AND sys.deleted = '0'
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

// TestCaseCountRow 测试用例聚合行。
type TestCaseCountRow struct {
	TotalCount    int `gorm:"column:total_count"`
	ExecutedCount int `gorm:"column:executed_count"`
	PassedCount   int `gorm:"column:passed_count"`
	FailedCount   int `gorm:"column:failed_count"`
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
       SUM(CASE WHEN lastRunResult = 'pass' THEN 1 ELSE 0 END) AS passed_count,
       SUM(CASE WHEN lastRunResult = 'fail' THEN 1 ELSE 0 END) AS failed_count
FROM zt_case
WHERE story IN ? AND deleted = '0'`, storyIDs).Scan(&row).Error
	return row, err
}

// DemandTestTaskRow 测试单行。
type DemandTestTaskRow struct {
	ID        uint       `gorm:"column:id"`
	Name      string     `gorm:"column:name"`
	Status    string     `gorm:"column:status"`
	Owner     string     `gorm:"column:owner"`
	OwnerName string     `gorm:"column:owner_name"`
	Begin     *time.Time `gorm:"column:begin"`
	End       *time.Time `gorm:"column:end"`
}

// FindStoryTestTasks 查询研发需求关联的测试单。
func (r *DemandDetailRepo) FindStoryTestTasks(ctx context.Context, storyIDs []uint) ([]DemandTestTaskRow, error) {
	if len(storyIDs) == 0 {
		return nil, nil
	}
	var rows []DemandTestTaskRow
	err := r.db.WithContext(ctx).Raw(`
SELECT DISTINCT tt.id, tt.name, tt.status, tt.owner, tt.begin, tt.end,
       COALESCE(u.realname, tt.owner) AS owner_name
FROM zt_testtask tt
JOIN zt_testrun tr ON tr.task = tt.id
JOIN zt_case c ON c.id = tr.case
LEFT JOIN zt_user u ON tt.owner = u.account AND u.deleted = '0'
WHERE c.story IN ? AND tt.deleted = '0'
ORDER BY tt.id DESC`, storyIDs).Scan(&rows).Error
	return rows, err
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

// CheckDemandVisibility 实现 F01 对象级授权查询：判断 actor.Account 是否与
// 该业务需求存在任一 PO/提出/责任/测试/验收/评审/创建/闭环/编辑关系，
// 或与需求看板一致的 RD / 澄清 PM 关系（避免看板可见详情却 403）。
//
// 决策由 Service 层负责（loadDemandIfVisible），Repo 仅投影候选行，不做
// 最终放行判断。返回 true 表示命中至少一条关系；不命中返回 false。
//
// 涉及列（与 zt_demand / zt_demandclarify 实际 schema 对齐）：
//
//	originator / assignedTo / QD / RD / BRA / accepter / reviewer /
//	createdBy / closedBy / editedBy / feedbackedBy /
//	zt_demandclarify.PM
func (r *DemandDetailRepo) CheckDemandVisibility(ctx context.Context, demandID uint, account string) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("demand detail repo is not configured")
	}
	if demandID == 0 || account == "" {
		return false, nil
	}
	var hits int64
	err := r.db.WithContext(ctx).Raw(`
SELECT COUNT(*) FROM zt_demand d
WHERE d.id = ? AND d.deleted = '0'
  AND (
    d.originator    = ?
    OR d.assignedTo  = ?
    OR d.QD          = ?
    OR d.RD          = ?
    OR d.BRA         = ?
    OR d.accepter    = ?
    OR d.reviewer    = ?
    OR d.createdBy   = ?
    OR d.closedBy    = ?
    OR d.editedBy    = ?
    OR d.feedbackedBy= ?
    OR EXISTS (
      SELECT 1 FROM zt_demandclarify c
      WHERE c.demand = d.id AND c.PM = ?
    )
  )`, demandID, account, account, account, account, account,
		account, account, account, account, account, account, account).Scan(&hits).Error
	if err != nil {
		return false, err
	}
	return hits > 0, nil
}
