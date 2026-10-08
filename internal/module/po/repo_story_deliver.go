// =============================================================================
// 文件: internal/module/po/repo_story_deliver.go
// 模块: PO 工作台
// 类型: repository
// 职责: 独立研发需求发起交付的详情、缺陷与已绑定窗口读取。
// =============================================================================

package po

import "context"

// deliverStoryRow 独立研发需求交付表单所需字段。
type deliverStoryRow struct {
	ID          uint   `gorm:"column:id"`
	Title       string `gorm:"column:title"`
	Status      string `gorm:"column:status"`
	Deleted     string `gorm:"column:deleted"`
	AssignedTo  string `gorm:"column:assignedTo"`
	ProductReqM string `gorm:"column:productReqM"`
	DeliverDate string `gorm:"column:deliverDate"`
	IsCarReview string `gorm:"column:isCarReview"`
	VerifyDate  string `gorm:"column:verifyDate"`
	VerifyPlan  string `gorm:"column:verifyPlan"`
	Verifier    string `gorm:"column:veriFier"`
	Delivered   int    `gorm:"column:delivered"`
	WindowBound int    `gorm:"column:windowBound"`
}

// FindDeliverStory 查询独立研发需求交付详情。
func (r *Repo) FindDeliverStory(ctx context.Context, id uint) (*deliverStoryRow, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	var row deliverStoryRow
	query := `
SELECT
  id, title, status, deleted, assignedTo,
  IFNULL((SELECT p.ReqM FROM zt_product p WHERE p.id = zt_story.product AND p.deleted = '0' LIMIT 1), '') AS productReqM,
  IFNULL(DATE_FORMAT(deliverDate, '%Y-%m-%d'), '') AS deliverDate,
  IFNULL(isCarReview, '') AS isCarReview, IFNULL(verifyDate, '') AS verifyDate,
  IFNULL(verifyPlan, '') AS verifyPlan, IFNULL(veriFier, '') AS veriFier,
  CASE WHEN ` + storyDeliveredSQL + ` THEN 1 ELSE 0 END AS delivered,
  CASE WHEN ` + storyWindowBoundSQL + ` THEN 1 ELSE 0 END AS windowBound
FROM zt_story
WHERE id = ? AND deleted = '0' AND fromDemand = 0 AND type = 'story'
  AND isParent = '0' AND IFNULL(sourceType, '') <> 'demandpool'
LIMIT 1`
	if err := db.WithContext(ctx).Raw(query, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errStoryNotFound
	}
	return &row, nil
}

// CheckStoryDeliverBlockers 统计该研发需求下未关闭缺陷。
func (r *Repo) CheckStoryDeliverBlockers(ctx context.Context, storyID uint) (severeBugs int, openBugs int, err error) {
	const query = `
SELECT
  COUNT(CASE WHEN b.severity IN ('1', '2', '10') THEN 1 END) AS severe_count,
  COUNT(1) AS open_count
FROM zt_bug b
WHERE b.story = ?
  AND b.deleted = '0'
  AND b.status IN ('active', 'resolved')`
	return r.readDeliverBugCounts(ctx, query, storyID)
}

// FindStoryLinkedWindow 查询研发需求经产品计划绑定的上线窗口。
func (r *Repo) FindStoryLinkedWindow(ctx context.Context, storyID uint) (windowID uint, windowName, releaseDate string, err error) {
	const query = `
SELECT vw.id AS window_id, vw.name AS window_name, IFNULL(DATE_FORMAT(vw.releaseDate, '%Y-%m-%d'), '') AS release_date
FROM zt_planstory ps
INNER JOIN zt_versionwindowproduct vwp ON vwp.plan = ps.plan AND vwp.deletedAt IS NULL
INNER JOIN zt_versionwindow vw ON vw.id = vwp.versionWindow AND vw.deletedAt IS NULL
WHERE ps.story = ?
ORDER BY vw.releaseDate DESC, vw.id DESC
LIMIT 1`
	return r.readDeliverWindow(ctx, query, storyID)
}
