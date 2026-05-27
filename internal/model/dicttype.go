// =============================================================================
// 文件: internal/model/dicttype.go
// 模块: 数据模型
// 类型: model
// 职责: 定义字典类型模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// DictType 表示 zt_dict_types 字典类型表。
type DictType struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	TenantID  int64      `gorm:"column:tenantId;not null;default:0"`
	Code      string     `gorm:"column:code;size:64;not null;default:''"`
	Name      string     `gorm:"column:name;size:64;not null;default:''"`
	Remark    string     `gorm:"column:remark;size:255;not null;default:''"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime:milli"`
	DeletedAt *time.Time `gorm:"column:deletedAt"`
}

// TableName 指定 zt_dict_types 表。
func (DictType) TableName() string {
	return "zt_dict_types"
}
