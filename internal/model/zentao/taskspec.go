// =============================================================================
// 文件: internal/model/zentao/taskspec.go
// 模块: 数据模型
// 类型: model
// 职责: 定义任务描述表 zt_taskspec 字段与表映射。
// 依赖: 无
// =============================================================================

package model

// ZtTaskspec 任务描述表。
type ZtTaskspec struct {
	Task       uint   `gorm:"column:task;primaryKey"`
	Version    int    `gorm:"column:version;primaryKey"`
	Name       string `gorm:"column:name"`
	EstStarted string `gorm:"column:estStarted"`
	Deadline   string `gorm:"column:deadline"`
}

// TableName 指定表名。
func (ZtTaskspec) TableName() string { return "zt_taskspec" }
