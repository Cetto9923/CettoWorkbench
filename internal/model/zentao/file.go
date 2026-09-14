// =============================================================================
// 文件: internal/model/zentao/file.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道附件表 zt_file 字段映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtFile 禅道附件表。
type ZtFile struct {
	ID         uint       `gorm:"column:id;primaryKey" json:"id"`
	Pathname   string     `gorm:"column:pathname" json:"pathname"`
	Title      string     `gorm:"column:title" json:"title"`
	Extension  string     `gorm:"column:extension" json:"extension"`
	Size       int        `gorm:"column:size" json:"size"`
	ObjectType string     `gorm:"column:objectType" json:"objectType"`
	ObjectID   int        `gorm:"column:objectID" json:"objectID"`
	AddedBy    string     `gorm:"column:addedBy" json:"addedBy"`
	AddedDate  *time.Time `gorm:"column:addedDate" json:"addedDate"`
	Downloads  int        `gorm:"column:downloads" json:"downloads"`
	Extra      string     `gorm:"column:extra" json:"extra"`
	Deleted    string     `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtFile) TableName() string { return "zt_file" }
