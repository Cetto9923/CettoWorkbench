// =============================================================================
// 文件: internal/model/zentao/teamgroup.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道敏捷小组表 zt_teamgroup 只读字段映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtTeamgroup 敏捷小组表。
type ZtTeamgroup struct {
	ID            uint       `gorm:"column:id;primaryKey" json:"id"`
	Type          string     `gorm:"column:type" json:"type"`
	Name          string     `gorm:"column:name" json:"name"`
	PO            string     `gorm:"column:PO" json:"PO"`
	Manager       string     `gorm:"column:manager" json:"manager"` // 敏捷教练，可多人逗号分隔
	Parent        uint       `gorm:"column:parent" json:"parent"`
	Grade         int        `gorm:"column:grade" json:"grade"`
	Path          string     `gorm:"column:path" json:"path"`
	Status        string     `gorm:"column:status" json:"status"`
	CreatedBy     string     `gorm:"column:createdBy" json:"createdBy"`
	CreatedDate   *time.Time `gorm:"column:createdDate" json:"createdDate"`
	DisbandedDate *time.Time `gorm:"column:disbandedDate" json:"disbandedDate"`
	Deleted       string     `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtTeamgroup) TableName() string { return "zt_teamgroup" }
