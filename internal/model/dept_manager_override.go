// =============================================================================
// 文件: internal/model/dept_manager_override.go
// 模块: 数据模型
// 类型: model
// 职责: 定义科技本部部门负责人补缺覆盖表模型与映射。
// =============================================================================

package model

import "time"

// DeptManagerOverride 科技本部部门负责人补缺覆盖表
type DeptManagerOverride struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Dept        uint       `gorm:"column:dept;not null;default:0" json:"dept"`
	Account     string     `gorm:"column:account;size:30;not null;default:''" json:"account"`
	Remark      string     `gorm:"column:remark;size:255;not null;default:''" json:"remark"`
	CreatedBy   string     `gorm:"column:createdBy;size:30;not null;default:''" json:"createdBy"`
	CreatedDate *time.Time `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	UpdatedBy   string     `gorm:"column:updatedBy;size:30;not null;default:''" json:"updatedBy"`
	UpdatedDate *time.Time `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate"`
	Deleted     string     `gorm:"column:deleted;size:1;not null;default:'0'" json:"deleted"`
}

// TableName 指定 zt_wb_dept_manager_override 表。
func (DeptManagerOverride) TableName() string {
	return "zt_wb_dept_manager_override"
}
