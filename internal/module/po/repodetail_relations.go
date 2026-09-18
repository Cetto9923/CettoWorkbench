// =============================================================================
// 文件: internal/module/po/repodetail_relations.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需详情关联块查询（转化研发需求/用户故事/评审记录/工单）。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"
)

// demandStoryDetailRow 转化研发需求投影。
type demandStoryDetailRow struct {
	ID          uint       `gorm:"column:id"`
	Title       string     `gorm:"column:title"`
	Stage       string     `gorm:"column:stage"`
	ProductName string     `gorm:"column:product_name"`
	ReleaseDate *time.Time `gorm:"column:release_date"`
}

// demandUserStoryDetailRow 用户故事条目投影。
type demandUserStoryDetailRow struct {
	ID          uint   `gorm:"column:id"`
	Role        string `gorm:"column:role"`
	GV          string `gorm:"column:gv"`
	ProductName string `gorm:"column:product_name"`
	Point       int    `gorm:"column:point"`
	Revpoint    int    `gorm:"column:revpoint"`
}

// demandReviewRecordRow 评审记录投影。
type demandReviewRecordRow struct {
	ID           uint       `gorm:"column:id"`
	ReviewType   string     `gorm:"column:reviewType"`
	ReviewDate   *time.Time `gorm:"column:reviewDate"`
	ReviewResult string     `gorm:"column:reviewResult"`
	CreatedBy    string     `gorm:"column:createdBy"`
	CreatedDate  *time.Time `gorm:"column:createdDate"`
	ReviewStatus string     `gorm:"column:reviewStatus"`
}

// demandTicketRow 工单投影。
type demandTicketRow struct {
	ID     uint   `gorm:"column:id"`
	Title  string `gorm:"column:title"`
	Pri    uint8  `gorm:"column:pri"`
	Status string `gorm:"column:status"`
}

// FindDemandStoriesByDemandID 查询转化的研发需求（对齐禅道 demand->SRS）。
func (r *Repo) FindDemandStoriesByDemandID(ctx context.Context, demandID int64) ([]demandStoryDetailRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var rows []demandStoryDetailRow
	// 发布时间取关联发布首条 date（对齐禅道 view：stories 逗号列表含 story id）。
	// 不用 FIND_IN_SET：CAST AS CHAR 与 zt_release.stories（utf8mb4_unicode_ci）会触发 collation 冲突。
	err := r.db.WithContext(ctx).Raw(`
SELECT
  s.id,
  s.title,
  s.stage,
  COALESCE(p.name, '') AS product_name,
  (
    SELECT r.date
    FROM zt_release r
    WHERE r.deleted = '0'
      AND CONCAT(',', IFNULL(r.stories, ''), ',') LIKE CONCAT('%,', s.id, ',%')
      AND r.date IS NOT NULL
      AND r.date > '1970-01-01'
    ORDER BY r.id ASC
    LIMIT 1
  ) AS release_date
FROM zt_story s
LEFT JOIN zt_product p ON p.id = s.product AND p.deleted = '0'
WHERE s.fromDemand = ?
  AND s.type = 'story'
  AND s.deleted = '0'
ORDER BY s.id ASC`, demandID).Scan(&rows).Error
	return rows, err
}

// FindDemandUserStoriesByDemandID 查询用户故事条目。
func (r *Repo) FindDemandUserStoriesByDemandID(ctx context.Context, demandID int64) ([]demandUserStoryDetailRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var rows []demandUserStoryDetailRow
	err := r.db.WithContext(ctx).Raw(`
SELECT
  us.id,
  us.role,
  us.gv,
  COALESCE(p.name, '') AS product_name,
  us.point,
  COALESCE(us.revpoint, 0) AS revpoint
FROM zt_demanduserstory us
LEFT JOIN zt_product p ON p.id = us.product AND p.deleted = '0'
WHERE us.demand = ?
ORDER BY us.id ASC`, demandID).Scan(&rows).Error
	return rows, err
}

// FindDemandReviewRecordsByDemandID 查询评审信息。
func (r *Repo) FindDemandReviewRecordsByDemandID(ctx context.Context, demandID int64) ([]demandReviewRecordRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var rows []demandReviewRecordRow
	err := r.db.WithContext(ctx).Raw(`
SELECT
  id,
  reviewType,
  reviewDate,
  reviewResult,
  createdBy,
  createdDate,
  reviewStatus
FROM zt_demandreviewrecord
WHERE objectID = ?
  AND deleted = '0'
ORDER BY createdDate DESC, id DESC`, demandID).Scan(&rows).Error
	return rows, err
}

// FindDemandTicketsByDemandID 查询工单信息。
func (r *Repo) FindDemandTicketsByDemandID(ctx context.Context, demandID int64) ([]demandTicketRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var rows []demandTicketRow
	err := r.db.WithContext(ctx).Raw(`
SELECT id, title, pri, status
FROM zt_ticket
WHERE demand = ?
  AND deleted = '0'
ORDER BY id ASC`, demandID).Scan(&rows).Error
	return rows, err
}
