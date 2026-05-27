// =============================================================================
// 文件: internal/model/dept.go
// 模块: 数据模型
// 类型: model
// 职责: 定义部门模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// Dept 表示 zt_depts 部门表。
type Dept struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TenantID  int64      `gorm:"column:tenantId;not null;default:0" json:"tenantId"`
	ParentID  uint64     `gorm:"column:parentId;not null;default:0" json:"parentId"`
	Name      string     `gorm:"column:name;size:64;not null;default:''" json:"name"`
	Leader    string     `gorm:"column:leader;size:64;not null;default:''" json:"leader"`
	Phone     string     `gorm:"column:phone;size:32;not null;default:''" json:"phone"`
	Email     string     `gorm:"column:email;size:64;not null;default:''" json:"email"`
	Status    uint8      `gorm:"column:status;not null;default:0" json:"status"`
	Sort      int        `gorm:"column:sort;not null;default:0" json:"sort"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime:milli" json:"updatedAt"`
	DeletedAt *time.Time `gorm:"column:deletedAt" json:"deletedAt"`
}

// TableName 指定 zt_depts 表。
func (Dept) TableName() string {
	return "zt_depts"
}
