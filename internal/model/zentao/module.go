// =============================================================================
// 文件: internal/model/zentao/module.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道模块表 zt_module 只读字段映射。
// 依赖: 无
// =============================================================================

package model

// ZtModule 模块表。
type ZtModule struct {
	ID      uint   `gorm:"column:id;primaryKey" json:"id"`
	Root    uint   `gorm:"column:root" json:"root"`
	Branch  uint   `gorm:"column:branch" json:"branch"`
	Name    string `gorm:"column:name" json:"name"`
	Parent  uint   `gorm:"column:parent" json:"parent"`
	Path    string `gorm:"column:path" json:"path"`
	Grade   uint8  `gorm:"column:grade" json:"grade"`
	Type    string `gorm:"column:type" json:"type"`
	Deleted string `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtModule) TableName() string { return "zt_module" }
