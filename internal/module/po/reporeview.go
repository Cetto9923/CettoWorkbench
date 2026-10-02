// =============================================================================
// 文件: internal/module/po/reporeview.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需评审读库（zt_demand / zt_demandreview）。只走主库 writeDB。
// 依赖: 无
// =============================================================================

package po

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// demandReviewRow 对应禅道 zt_demand 评审时用到的列（不必映射整张宽表）。
type demandReviewRow struct {
	ID          int64  `gorm:"column:id"`
	Status      string `gorm:"column:status"`
	Deleted     string `gorm:"column:deleted"`
	CreatedBy   string `gorm:"column:createdBy"`
	AssignedTo  string `gorm:"column:assignedTo"`
	ReviewedBy  string `gorm:"column:reviewedBy"`
	Reviewer    string `gorm:"column:reviewer"`
	Mailto      string `gorm:"column:mailto"`
	IsNeedFocus string `gorm:"column:isNeedFocus"`
	Product     string `gorm:"column:product"`
}

// demandActionRow 对应 zt_action，给禅道详情页「历史记录」用。
type demandActionRow struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   uint      `gorm:"column:objectID"`
	Product    string    `gorm:"column:product"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	Actor      string    `gorm:"column:actor"`
	Action     string    `gorm:"column:action"`
	Date       time.Time `gorm:"column:date"`
	Comment    string    `gorm:"column:comment"`
	Extra      string    `gorm:"column:extra"`
}

func (demandActionRow) TableName() string { return "zt_action" }

func (demandReviewRow) TableName() string { return "zt_demand" }

// demandReviewerRow 对应 zt_demandreview：每个评审人一行结果。
type demandReviewerRow struct {
	Demand     int64      `gorm:"column:demand"`
	Version    int        `gorm:"column:version"`
	Reviewer   string     `gorm:"column:reviewer"`
	Result     string     `gorm:"column:result"`
	ReviewDate *time.Time `gorm:"column:reviewDate"`
}

func (demandReviewerRow) TableName() string { return "zt_demandreview" }

func (r *Repo) writer() (*gorm.DB, error) {
	if r == nil || r.writeDB == nil {
		return nil, errors.New("主库未就绪，无法评审")
	}
	return r.writeDB, nil
}

// FindDemandForReview 按 ID 读业需（主库，避免备库延迟看到旧状态）。
func (r *Repo) FindDemandForReview(ctx context.Context, id int64) (*demandReviewRow, error) {
	db, err := r.writer()
	if err != nil {
		return nil, err
	}
	var row demandReviewRow
	err = db.WithContext(ctx).
		Select("id, status, deleted, createdBy, assignedTo, reviewedBy, reviewer, mailto, isNeedFocus, product").
		Where("id = ?", id).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load demand %d: %w", id, err)
	}
	return &row, nil
}

// FindDemandPoolBusinessReviewer 读业需所属需求池的业务评审人（zt_demandpool.businessReviewer，逗号分隔账号）。
func (r *Repo) FindDemandPoolBusinessReviewer(ctx context.Context, demandID int64) (string, error) {
	db, err := r.writer()
	if err != nil {
		return "", err
	}
	var raw string
	err = db.WithContext(ctx).Table("zt_demand AS d").
		Select("COALESCE(dp.businessReviewer, '')").
		Joins("LEFT JOIN zt_demandpool AS dp ON d.pool = dp.id AND dp.deleted = ?", "0").
		Where("d.id = ?", demandID).
		Limit(1).
		Scan(&raw).Error
	if err != nil {
		return "", fmt.Errorf("load demand pool reviewer %d: %w", demandID, err)
	}
	return raw, nil
}

// FindDemandReviewerResult 查当前账号在 zt_demandreview 中的一行。found=false 表示不是业务评审人。
func (r *Repo) FindDemandReviewerResult(ctx context.Context, demandID int64, account string) (result string, found bool, err error) {
	db, err := r.writer()
	if err != nil {
		return "", false, err
	}
	var row demandReviewerRow
	err = db.WithContext(ctx).
		Select("result").
		Where("demand = ? AND reviewer = ?", demandID, account).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load demandreview %d %s: %w", demandID, account, err)
	}
	return row.Result, true, nil
}

// FindPendingReviewDemandIDs 当前账号尚未给出结果的业务评审需求（用于列表「评审」按钮）。
func (r *Repo) FindPendingReviewDemandIDs(ctx context.Context, account string, demandIDs []int) (map[int]struct{}, error) {
	out := map[int]struct{}{}
	if r == nil || r.db == nil || strings.TrimSpace(account) == "" || len(demandIDs) == 0 {
		return out, nil
	}
	var ids []int
	err := r.db.WithContext(ctx).
		Model(&demandReviewerRow{}).
		Where("demand IN ? AND reviewer = ? AND (result = ? OR result IS NULL)", demandIDs, account, "").
		Pluck("demand", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, nil
}
