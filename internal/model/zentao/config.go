// =============================================================================
// 文件: internal/model/zentao/config.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道配置表 zt_config 字段映射（提测 CRExecution 等开关读取）。
// 依赖: 无
// =============================================================================

package model

// ZtConfig 禅道配置表（zt_config）。
type ZtConfig struct {
	ID      uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Owner   string `gorm:"column:owner;type:varchar(30);not null;default:''" json:"owner"`
	Module  string `gorm:"column:module;type:varchar(30);not null;default:''" json:"module"`
	Section string `gorm:"column:section;type:varchar(30);not null;default:''" json:"section"`
	Key     string `gorm:"column:key;type:varchar(30);not null;default:''" json:"key"`
	Value   string `gorm:"column:value;type:text" json:"value"`
}

// TableName 指定表名。
func (ZtConfig) TableName() string { return "zt_config" }
