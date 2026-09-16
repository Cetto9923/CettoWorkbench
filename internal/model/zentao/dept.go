// =============================================================================
// 文件: internal/model/zentao/dept.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道部门表 zt_dept 字段映射（与工作台 zt_depts 不同）。
// 依赖: 无
// =============================================================================

package model

// ZtDept 禅道部门表。
type ZtDept struct {
	ID       uint   `gorm:"column:id;primaryKey" json:"id"`
	Name     string `gorm:"column:name" json:"name"`
	Parent   uint   `gorm:"column:parent" json:"parent"`
	Path     string `gorm:"column:path" json:"path"`
	Grade    uint8  `gorm:"column:grade" json:"grade"`
	Order    uint16 `gorm:"column:order" json:"order"`
	Position string `gorm:"column:position" json:"position"`
	Function string `gorm:"column:function" json:"function"`
	Manager  string `gorm:"column:manager" json:"manager"`
}

// TableName 指定表名。
func (ZtDept) TableName() string { return "zt_dept" }
