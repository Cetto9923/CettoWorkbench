// =============================================================================
// 文件: internal/model/dictitem.go
// 模块: 数据模型
// 类型: model
// 职责: 定义字典值模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// DictItem 表示 zt_dict_items 字典值表。
type DictItem struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	TenantID  int64      `gorm:"column:tenantId;not null;default:0"`
	TypeCode  string     `gorm:"column:typeCode;size:64;not null;default:''"`
	Label     string     `gorm:"column:label;size:64;not null;default:''"`
	Value     string     `gorm:"column:value;size:64;not null;default:''"`
	Sort      int        `gorm:"column:sort;not null;default:0"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime:milli"`
	DeletedAt *time.Time `gorm:"column:deletedAt"`
}

// TableName 指定 zt_dict_items 表。
func (DictItem) TableName() string {
	return "zt_dict_items"
}
