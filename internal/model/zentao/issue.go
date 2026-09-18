// =============================================================================
// 文件: internal/model/zentao/issue.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道问题表 zt_issue 只读字段映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtIssue 禅道问题表。
type ZtIssue struct {
	ID           uint       `gorm:"column:id;primaryKey" json:"id"`
	Project      string     `gorm:"column:project" json:"project"`
	Execution    uint       `gorm:"column:execution" json:"execution"`
	Title        string     `gorm:"column:title" json:"title"`
	Pri          string     `gorm:"column:pri" json:"pri"`
	Severity     string     `gorm:"column:severity" json:"severity"`
	Type         string     `gorm:"column:type" json:"type"`
	Deadline     *time.Time `gorm:"column:deadline" json:"deadline"`
	Status       string     `gorm:"column:status" json:"status"`
	Owner        string     `gorm:"column:owner" json:"owner"`
	CreatedBy    string     `gorm:"column:createdBy" json:"createdBy"`
	CreatedDate  *time.Time `gorm:"column:createdDate" json:"createdDate"`
	AssignedTo   string     `gorm:"column:assignedTo" json:"assignedTo"`
	AssignedDate *time.Time `gorm:"column:assignedDate" json:"assignedDate"`
	ResolvedBy   string     `gorm:"column:resolvedBy" json:"resolvedBy"`
	ResolvedDate *time.Time `gorm:"column:resolvedDate" json:"resolvedDate"`
	ClosedBy     string     `gorm:"column:closedBy" json:"closedBy"`
	ClosedDate   *time.Time `gorm:"column:closedDate" json:"closedDate"`
	Deleted      string     `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtIssue) TableName() string { return "zt_issue" }
