// =============================================================================
// 文件: internal/module/dept/repo_dept_override.go
// 模块: 部门管理
// 类型: repository
// 职责: 科技本部部门负责人补缺覆盖表（zt_wb_dept_manager_override）持久层访问。
// =============================================================================

package dept

import (
	"context"
	"strings"
	"time"

	"workbench/internal/model"

	"gorm.io/gorm"
)

// DeptOverrideView 部门负责人补缺展示视图
type DeptOverrideView struct {
	ID          uint64     `json:"id" gorm:"column:id"`
	Dept        uint       `json:"dept" gorm:"column:dept"`
	DeptName    string     `json:"deptName" gorm:"column:dept_name"`
	Account     string     `json:"account" gorm:"column:account"`
	Realname    string     `json:"realname" gorm:"column:realname"`
	Remark      string     `json:"remark" gorm:"column:remark"`
	CreatedBy   string     `json:"createdBy" gorm:"column:createdBy"`
	CreatedDate *time.Time `json:"createdDate" gorm:"column:createdDate"`
}

// DeptOption 科技本部可选部门选项
type DeptOption struct {
	ID   uint   `json:"id" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`
	Path string `json:"path" gorm:"column:path"`
}

// ListOverrides 查询所有有效的部门负责人补缺记录
func (r *Repo) ListOverrides(ctx context.Context) ([]DeptOverrideView, error) {
	var rows []DeptOverrideView
	err := r.db.WithContext(ctx).Table("zt_wb_dept_manager_override o").
		Select("o.id, o.dept, COALESCE(d.name, '') AS dept_name, o.account, COALESCE(u.realname, o.account) AS realname, o.remark, o.createdBy, o.createdDate").
		Joins("LEFT JOIN zt_dept d ON d.id = o.dept").
		Joins("LEFT JOIN zt_user u ON u.account = o.account").
		Where("o.deleted = '0'").
		Order("o.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListTechDepts 查询科技部本部(52)及其子孙部门列表，供下拉框选择
func (r *Repo) ListTechDepts(ctx context.Context) ([]DeptOption, error) {
	var depts []DeptOption
	err := r.db.WithContext(ctx).Table("zt_dept").
		Select("id, name, path").
		Where("id = 52 OR path LIKE '%,52,%'").
		Order("grade ASC, id ASC").
		Scan(&depts).Error
	if err != nil {
		return nil, err
	}
	return depts, nil
}

// SaveOverride 保存或更新补缺记录（同一部门只保留一条有效覆盖记录）
func (r *Repo) SaveOverride(ctx context.Context, dept uint, account, remark, operator string) error {
	account = strings.TrimSpace(account)
	operator = strings.TrimSpace(operator)
	remark = strings.TrimSpace(remark)
	now := time.Now()

	// 查找该部门现有的有效记录
	var existing model.DeptManagerOverride
	err := r.db.WithContext(ctx).
		Where("dept = ? AND deleted = '0'", dept).
		First(&existing).Error

	if err == nil && existing.ID > 0 {
		// 更新已有记录
		return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
			"account":     account,
			"remark":      remark,
			"updatedBy":   operator,
			"updatedDate": &now,
		}).Error
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	// 插入新记录
	record := model.DeptManagerOverride{
		Dept:        dept,
		Account:     account,
		Remark:      remark,
		CreatedBy:   operator,
		CreatedDate: &now,
		UpdatedBy:   operator,
		UpdatedDate: &now,
		Deleted:     "0",
	}
	return r.db.WithContext(ctx).Create(&record).Error
}

// DeleteOverride 软删除指定补缺记录
func (r *Repo) DeleteOverride(ctx context.Context, id uint64, operator string) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.DeptManagerOverride{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted":     "1",
			"updatedBy":   strings.TrimSpace(operator),
			"updatedDate": &now,
		}).Error
}
