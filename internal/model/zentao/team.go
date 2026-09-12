// =============================================================================
// 文件: internal/model/zentao/team.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道团队成员表 zt_team 只读字段映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtTeam 团队成员表（项目 / 执行 / 敏捷小组等）。
type ZtTeam struct {
	ID        uint       `gorm:"column:id;primaryKey" json:"id"`
	Root      uint       `gorm:"column:root" json:"root"`
	Type      string     `gorm:"column:type" json:"type"`
	Teamgroup uint       `gorm:"column:teamgroup" json:"teamgroup"`
	Account   string     `gorm:"column:account" json:"account"`
	Role      string     `gorm:"column:role" json:"role"`
	Position  string     `gorm:"column:position" json:"position"`
	Limited   string     `gorm:"column:limited" json:"limited"`
	Join      *time.Time `gorm:"column:join" json:"join"`
	Days      uint       `gorm:"column:days" json:"days"`
	Hours     float32    `gorm:"column:hours" json:"hours"`
	Order     int8       `gorm:"column:order" json:"order"`
}

// TableName 指定表名。
func (ZtTeam) TableName() string { return "zt_team" }
