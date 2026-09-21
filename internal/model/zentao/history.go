// =============================================================================
// 文件: internal/model/zentao/history.go
// 模块: 数据模型
// 类型: model
// 职责: 定义禅道字段变更表 zt_history 字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtHistory 操作的字段变更，对应禅道详情历史面板中的变更内容。
type ZtHistory struct {
	ID     uint   `gorm:"column:id;primaryKey;autoIncrement"`
	Action uint   `gorm:"column:action"`
	Field  string `gorm:"column:field"`
	Old    string `gorm:"column:old"`
	New    string `gorm:"column:new"`
	Diff   string `gorm:"column:diff"`
}

// TableName 指定表名。
func (ZtHistory) TableName() string { return "zt_history" }
