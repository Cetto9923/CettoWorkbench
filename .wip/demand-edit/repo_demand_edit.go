// =============================================================================
// 文件: internal/module/po/repo_demand_edit.go
// 模块: PO 工作台
// 类型: repository
// 职责: 业务需求草稿/驳回状态的编辑元数据查询、更新及软删除持久化（主库事务锁行）。
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	errDemandEditNotFound = errors.New("需求不存在或已被删除")
	errDemandEditConflict = errors.New("当前需求状态不允许编辑或删除")
)

// DemandEditRow 仓储层编辑读取实体。
type DemandEditRow struct {
	ID             int64      `gorm:"column:id"`
	Name           string     `gorm:"column:name"`
	Status         string     `gorm:"column:status"`
	Deleted        string     `gorm:"column:deleted"`
	Pri            string     `gorm:"column:pri"`
	Category       string     `gorm:"column:category"`
	Source         string     `gorm:"column:source"`
	SourceNote     string     `gorm:"column:sourceNote"`
	Pool           int        `gorm:"column:pool"`
	PoolName       string     `gorm:"column:pool_name"`
	Product        string     `gorm:"column:product"`
	ProductName    string     `gorm:"column:product_name"`
	Reviewer       string     `gorm:"column:reviewer"`
	ReviewerName   string     `gorm:"column:reviewer_name"`
	EstimateLaunch *time.Time `gorm:"column:estimateLaunch"`
	Desc           string     `gorm:"column:desc"`
	VerifyPlan     string     `gorm:"column:verifyPlan"`
	CreatedBy      string     `gorm:"column:createdBy"`
	CreatedByName  string     `gorm:"column:created_by_name"`
}

// FindDemandForEdit 按 ID 从主库读取需求编辑数据。
func (r *Repo) FindDemandForEdit(ctx context.Context, id int64) (*DemandEditRow, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}
	const query = `
SELECT
  d.id, d.name, d.status, d.deleted, d.pri, d.category, d.source, d.sourceNote,
  d.pool, IFNULL(dp.name, '') AS pool_name,
  d.product, IFNULL(pr.name, '') AS product_name,
  d.reviewer, IFNULL(CONCAT(u_rev.realname, ' (', u_rev.account, ')'), d.reviewer) AS reviewer_name,
  d.estimateLaunch, d.desc, d.verifyPlan,
  d.createdBy, IFNULL(CONCAT(u_cr.realname, ' (', u_cr.account, ')'), d.createdBy) AS created_by_name
FROM zt_demand d
LEFT JOIN zt_demandpool dp ON d.pool = dp.id AND dp.deleted = '0'
LEFT JOIN zt_product pr ON d.product = CAST(pr.id AS CHAR) AND pr.deleted = '0'
LEFT JOIN zt_user u_rev ON d.reviewer = u_rev.account AND u_rev.deleted = '0'
LEFT JOIN zt_user u_cr ON d.createdBy = u_cr.account AND u_cr.deleted = '0'
WHERE d.id = ? AND d.deleted = '0'
LIMIT 1`

	var row DemandEditRow
	if err := db.WithContext(ctx).Raw(query, id).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, errDemandEditNotFound
	}
	return &row, nil
}

// ListDemandEditOptions 获取编辑表单所需的公共字典选项列表。
func (r *Repo) ListDemandEditOptions(ctx context.Context) (*DemandEditOptions, error) {
	db, err := r.homeActionWriter()
	if err != nil {
		return nil, err
	}

	opts := &DemandEditOptions{
		Categories: []DemandEditOptionItem{
			{Value: "feature", Label: "功能需求"},
			{Value: "experience", Label: "体验优化"},
			{Value: "business", Label: "业务需求"},
			{Value: "tecopt", Label: "技术优化"},
			{Value: "safe", Label: "安全"},
			{Value: "performance", Label: "性能"},
			{Value: "datacg", Label: "数据变更"},
			{Value: "research", Label: "调研需求"},
			{Value: "bug", Label: "BUG"},
			{Value: "other", Label: "其他"},
		},
		Priorities: []DemandEditOptionItem{
			{Value: "1", Label: "P1 (紧急)"},
			{Value: "2", Label: "P2 (高)"},
			{Value: "3", Label: "P3 (中)"},
			{Value: "4", Label: "P4 (低)"},
		},
		Sources: []DemandEditOptionItem{
			{Value: "业务提出", Label: "业务提出"},
			{Value: "战略规划", Label: "战略规划"},
			{Value: "客户反馈", Label: "客户反馈"},
			{Value: "运维发现", Label: "运维发现"},
			{Value: "监管要求", Label: "监管要求"},
			{Value: "内部优化", Label: "内部优化"},
			{Value: "其他", Label: "其他"},
		},
		Pools:     []DemandEditOptionItem{},
		Products:  []DemandEditOptionItem{},
		Reviewers: []DemandEditOptionItem{},
	}

	// 1. 需求池选项
	type idNameRow struct {
		ID   string `gorm:"column:id"`
		Name string `gorm:"column:name"`
	}
	var pools []idNameRow
	_ = db.WithContext(ctx).Raw("SELECT id, name FROM zt_demandpool WHERE deleted = '0' ORDER BY id DESC").Scan(&pools).Error
	for _, p := range pools {
		opts.Pools = append(opts.Pools, DemandEditOptionItem{Value: p.ID, Label: p.Name})
	}

	// 2. 产品选项
	var products []idNameRow
	_ = db.WithContext(ctx).Raw("SELECT id, name FROM zt_product WHERE deleted = '0' ORDER BY id DESC").Scan(&products).Error
	for _, pr := range products {
		opts.Products = append(opts.Products, DemandEditOptionItem{Value: pr.ID, Label: pr.Name})
	}

	// 3. 评审人选项
	type userRow struct {
		Value string `gorm:"column:value"`
		Label string `gorm:"column:label"`
	}
	var reviewers []userRow
	_ = db.WithContext(ctx).Raw("SELECT account AS value, CONCAT(realname, ' (', account, ')') AS label FROM zt_user WHERE deleted = '0' AND type = 'inside' ORDER BY account ASC").Scan(&reviewers).Error
	for _, u := range reviewers {
		opts.Reviewers = append(opts.Reviewers, DemandEditOptionItem{Value: u.Value, Label: u.Label})
	}

	return opts, nil
}

// UpdateDemandDraft 事务更新草稿/驳回需求内容。
func (r *Repo) UpdateDemandDraft(ctx context.Context, id int64, actor string, updates map[string]any, comment string) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked demandReviewRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("zt_demand").
			Select("id, status, deleted, createdBy, product").
			Where("id = ?", id).
			Take(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDemandEditNotFound
			}
			return err
		}
		if locked.Deleted != "0" {
			return errDemandEditNotFound
		}
		st := strings.ToLower(strings.TrimSpace(locked.Status))
		if st != "draft" && st != "refuse" {
			return errDemandEditConflict
		}

		updates["lastEditedBy"] = actor
		updates["lastEditedDate"] = nowStr
		updates["editedBy"] = actor
		updates["editedDate"] = nowStr

		if err := tx.Table("zt_demand").Where("id = ? AND deleted = '0'", id).Updates(updates).Error; err != nil {
			return fmt.Errorf("update zt_demand: %w", err)
		}

		// 插入 zt_action 审计日志
		productField := ",0,"
		if p := strings.TrimSpace(locked.Product); p != "" && p != "0" {
			productField = "," + p + ","
		}
		c := strings.TrimSpace(comment)
		if c == "" {
			c = "工作台更新业务需求内容"
		}
		return tx.Create(&demandActionRow{
			ObjectType: "demand",
			ObjectID:   uint(id),
			Product:    productField,
			Actor:      actor,
			Action:     "edited",
			Date:       now,
			Comment:    c,
		}).Error
	})
}

// DeleteDemandDraft 事务软删除草稿/驳回需求。
func (r *Repo) DeleteDemandDraft(ctx context.Context, id int64, actor string, comment string) error {
	db, err := r.homeActionWriter()
	if err != nil {
		return err
	}
	now := time.Now()
	nowStr := now.Format("2006-01-02 15:04:05")

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked demandReviewRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("zt_demand").
			Select("id, status, deleted, createdBy, product").
			Where("id = ?", id).
			Take(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDemandEditNotFound
			}
			return err
		}
		if locked.Deleted != "0" {
			return errDemandEditNotFound
		}
		st := strings.ToLower(strings.TrimSpace(locked.Status))
		if st != "draft" && st != "refuse" {
			return errDemandEditConflict
		}

		// 1. 软删除 zt_demand
		delUpdates := map[string]any{
			"deleted":        "1",
			"lastEditedBy":   actor,
			"lastEditedDate": nowStr,
		}
		if err := tx.Table("zt_demand").Where("id = ?", id).Updates(delUpdates).Error; err != nil {
			return fmt.Errorf("delete zt_demand: %w", err)
		}

		// 2. 解除可能关联的研发需求绑定（对齐禅道原生 delete 逻辑）
		if err := tx.Table("zt_story").
			Where("fromDemand = ? AND deleted = '0'", id).
			Update("fromDemand", 0).Error; err != nil {
			return fmt.Errorf("unlink zt_story fromDemand: %w", err)
		}

		// 3. 记录 zt_action 审计日志
		productField := ",0,"
		if p := strings.TrimSpace(locked.Product); p != "" && p != "0" {
			productField = "," + p + ","
		}
		c := strings.TrimSpace(comment)
		if c == "" {
			c = "工作台创建人删除业务需求"
		}
		return tx.Create(&demandActionRow{
			ObjectType: "demand",
			ObjectID:   uint(id),
			Product:    productField,
			Actor:      actor,
			Action:     "deleted",
			Date:       now,
			Comment:    c,
		}).Error
	})
}
