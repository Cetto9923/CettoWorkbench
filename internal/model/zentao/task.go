// =============================================================================
// 文件: internal/model/zentao/task.go
// 模块: 数据模型
// 类型: model
// 职责: 任务 zt_task 的插入列映射，只包含排期创建时写入的字段。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtTask 是 zt_task 的插入投影，避免把未赋值列写成零值。
type ZtTask struct {
	ID         uint      `gorm:"column:id;primaryKey;autoIncrement"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	Pri        int       `gorm:"column:pri"`
	Story      uint      `gorm:"column:story"`
	Project    uint      `gorm:"column:project"`
	Execution  uint      `gorm:"column:execution"`
	AssignedTo string    `gorm:"column:assignedTo"`
	Estimate   float64   `gorm:"column:estimate"`
	Consumed   float64   `gorm:"column:consumed"`
	Left       float64   `gorm:"column:left"`
	EstStarted string    `gorm:"column:estStarted"`
	Deadline   string    `gorm:"column:deadline"`
	Status     string    `gorm:"column:status"`
	OpenedBy   string    `gorm:"column:openedBy"`
	OpenedDate time.Time `gorm:"column:openedDate"`
	Version    int       `gorm:"column:version"`
	Deleted    string    `gorm:"column:deleted"`
}

// TableName 指定表名。
func (ZtTask) TableName() string { return "zt_task" }
