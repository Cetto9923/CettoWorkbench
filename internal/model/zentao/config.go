// =============================================================================
// 文件: internal/model/zentao/config.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道系统配置表字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtConfig 禅道配置表（zt_config）。
type ZtConfig struct {
	ID      uint   `gorm:"column:id;primaryKey;autoIncrement;type:mediumint unsigned" json:"id"`
	Vision  string `gorm:"column:vision;type:varchar(10);not null;default:''" json:"vision"`
	Owner   string `gorm:"column:owner;type:char(30);not null;default:''" json:"owner"`
	Module  string `gorm:"column:module;type:varchar(30);not null;default:''" json:"module"`
	Section string `gorm:"column:section;type:char(30);not null;default:''" json:"section"`
	Key     string `gorm:"column:key;type:char(30);not null;default:''" json:"key"`
	Value   string `gorm:"column:value;type:longtext" json:"value"`
}

// TableName 指定表名。
func (ZtConfig) TableName() string {
	return "zt_config"
}
