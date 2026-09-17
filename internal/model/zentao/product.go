// =============================================================================
// 文件: internal/model/zentao/product.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道产品表 zt_product 只读字段映射。
// 依赖: 无
// =============================================================================

package model

// ZtProduct 产品表。
type ZtProduct struct {
	ID      uint   `gorm:"column:id;primaryKey" json:"id"`
	Name    string `gorm:"column:name" json:"name"`
	Type    string `gorm:"column:type" json:"type"`
	ReqM    string `gorm:"column:ReqM;type:varchar(30);default:''" json:"reqM"` // 需求负责人
	Deleted string `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtProduct) TableName() string { return "zt_product" }
