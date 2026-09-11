// =============================================================================
// 文件: internal/model/zentao/productplan.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道产品计划表 zt_productplan 只读字段映射。
// 依赖: 无
// =============================================================================

package model

// ZtProductplan 产品计划表。
type ZtProductplan struct {
	ID      uint   `gorm:"column:id;primaryKey" json:"id"`
	Product uint   `gorm:"column:product" json:"product"`
	Branch  string `gorm:"column:branch" json:"branch"`
	Title   string `gorm:"column:title" json:"title"`
	Status  string `gorm:"column:status" json:"status"`
	Deleted string `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtProductplan) TableName() string { return "zt_productplan" }
