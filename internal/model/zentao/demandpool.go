// =============================================================================
// 文件: internal/model/zentao/demandpool.go
// 模块: 数据模型
// 类型: model
// 职责: 禅道需求池表 zt_demandpool 字段映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// ZtDemandpool 禅道需求池表。
type ZtDemandpool struct {
	ID          int        `gorm:"column:id;primaryKey" json:"id"`
	Name        string     `gorm:"column:name" json:"name"`
	Desc        *string    `gorm:"column:desc" json:"desc"`
	Status      string     `gorm:"column:status" json:"status"`
	Products    string     `gorm:"column:products" json:"products"`
	CreatedBy   string     `gorm:"column:createdBy" json:"createdBy"`
	CreatedDate *time.Time `gorm:"column:createdDate" json:"createdDate"`
	Owner       *string    `gorm:"column:owner" json:"owner"`
	Reviewer    *string    `gorm:"column:reviewer" json:"reviewer"`
	ACL         *string    `gorm:"column:acl" json:"acl"`
	Deleted     string     `gorm:"column:deleted" json:"deleted"`
}

// TableName 指定表名。
func (ZtDemandpool) TableName() string { return "zt_demandpool" }
