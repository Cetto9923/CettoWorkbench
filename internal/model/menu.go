// =============================================================================
// 文件: internal/model/menu.go
// 模块: 数据模型
// 类型: model
// 职责: 定义菜单模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// Menu 表示 zt_menus 菜单表。
type Menu struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	ParentID  uint64     `gorm:"column:parentId;not null;default:0;uniqueIndex:idx_parent_title_perm"`
	Title     string     `gorm:"column:title;size:64;not null;default:'';uniqueIndex:idx_parent_title_perm"`
	Icon      string     `gorm:"column:icon;size:64;not null;default:''"`
	Path      string     `gorm:"column:path;size:255;not null;default:''"`
	Perm      string     `gorm:"column:perm;size:64;not null;default:'';uniqueIndex:idx_parent_title_perm;index:idx_zt_menus_perm"`
	Type      string     `gorm:"column:type;type:char(1);not null;default:'C'"`
	Sort      int        `gorm:"column:sort;not null;default:0"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime:milli"`
	DeletedAt *time.Time `gorm:"column:deletedAt"`
}

// TableName 指定 zt_menus 表。
func (Menu) TableName() string {
	return "zt_menus"
}
