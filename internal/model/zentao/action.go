// =============================================================================
// 文件: internal/model/zentao/action.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道操作日志表 zt_action 字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtAction 操作日志表，供禅道详情历史记录使用。
type ZtAction struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	ObjectType string    `gorm:"column:objectType"`
	ObjectID   uint      `gorm:"column:objectID"`
	Product    string    `gorm:"column:product"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	Actor      string    `gorm:"column:actor"`
	Action     string    `gorm:"column:action"`
	Date       time.Time `gorm:"column:date"`
	Comment    string    `gorm:"column:comment"`
	Extra      string    `gorm:"column:extra"`
}

// TableName 指定表名。
func (ZtAction) TableName() string { return "zt_action" }
