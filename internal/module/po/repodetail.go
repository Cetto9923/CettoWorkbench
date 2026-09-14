// =============================================================================
// 文件: internal/module/po/repodetail.go
// 模块: PO 工作台
// 类型: action
// 职责: 业需详情主记录与附件查询（只读备库）。
// 依赖: internal/model/zentao
// =============================================================================

package po

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	ztmodel "workbench/internal/model/zentao"
)

// DemandDetailRow 业需详情投影（对齐禅道 demand-view 抽屉所需列）。
type DemandDetailRow struct {
	ID          int64      `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Pri         string     `gorm:"column:pri"`
	Category    string     `gorm:"column:category"`
	Source      string     `gorm:"column:source"`
	Desc        string     `gorm:"column:desc"`
	VerifyPlan  string     `gorm:"column:verifyPlan"`
	Status      string     `gorm:"column:status"`
	Originator  string     `gorm:"column:originator"`
	ProposeDept string     `gorm:"column:propose_dept"`
	BRA         string     `gorm:"column:BRA"`
	QD          string     `gorm:"column:QD"`
	RD          string     `gorm:"column:RD"`
	Reviewer    string     `gorm:"column:reviewer"`
	AssignedTo  string     `gorm:"column:assignedTo"`
	CreatedBy   string     `gorm:"column:createdBy"`
	CreatedDate *time.Time `gorm:"column:createdDate"`
	Deadline    *time.Time `gorm:"column:deadline"`
	PoolName    string     `gorm:"column:pool_name"`
}

// FindDemandDetailByID 查询单个业需及池/提出部门展示名。
func (r *Repo) FindDemandDetailByID(ctx context.Context, id int64) (*DemandDetailRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var row DemandDetailRow
	err := r.db.WithContext(ctx).
		Table(ztmodel.ZtDemand{}.TableName()+" AS d").
		Select(
			"d.id, d.name, d.pri, d.category, d.source, d.`desc`, d.verifyPlan, d.status, "+
				"d.originator, d.BRA, d.QD, d.RD, d.reviewer, d.assignedTo, d.createdBy, "+
				"d.createdDate, d.deadline, "+
				"COALESCE(dept.name, '') AS propose_dept, "+
				"COALESCE(dp.name, '') AS pool_name",
		).
		Joins("LEFT JOIN "+ztmodel.ZtDept{}.TableName()+" AS dept ON d.proposeDept = dept.id").
		Joins("LEFT JOIN "+ztmodel.ZtDemandpool{}.TableName()+" AS dp ON d.pool = dp.id AND dp.deleted = ?", "0").
		Where("d.id = ? AND d.deleted = ?", id, "0").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

// FindDemandFiles 查询业需附件（排除 appendFiles）。
func (r *Repo) FindDemandFiles(ctx context.Context, demandID int64) ([]ztmodel.ZtFile, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("po repo db is nil")
	}
	var rows []ztmodel.ZtFile
	err := r.db.WithContext(ctx).
		Model(&ztmodel.ZtFile{}).
		Select("id", "title", "size").
		Where("objectType = ? AND objectID = ? AND deleted = ?", "demand", demandID, "0").
		Where("extra IS NULL OR extra <> ?", "appendFiles").
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}
