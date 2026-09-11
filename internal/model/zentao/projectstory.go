// =============================================================================
// 文件: internal/model/zentao/projectstory.go
// 模块: 数据模型
// 类型: model
// 职责: 定义项目/执行与研发需求关联表 zt_projectstory。
// 依赖: 无
// =============================================================================

package model

// ZtProjectstory 项目/执行-需求关联表（zt_projectstory）。
// project 列可为项目或执行 ID。
type ZtProjectstory struct {
	Project uint  `gorm:"column:project;type:mediumint unsigned;not null;default:0" json:"project"`
	Product uint  `gorm:"column:product;type:mediumint unsigned;not null;default:0" json:"product"`
	Branch  uint  `gorm:"column:branch;type:mediumint unsigned;not null;default:0" json:"branch"`
	Story   uint  `gorm:"column:story;type:mediumint unsigned;not null;default:0" json:"story"`
	Version int16 `gorm:"column:version;type:smallint;not null;default:1" json:"version"`
	Order   uint  `gorm:"column:order;type:smallint unsigned;not null;default:0" json:"order"`
}

// TableName 指定表名。
func (ZtProjectstory) TableName() string {
	return "zt_projectstory"
}
