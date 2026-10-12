// =============================================================================
// 文件: internal/model/usermenu.go
// 模块: 数据模型
// 类型: model
// 职责: 定义用户固定工作台菜单模型（zt_usermenu）。
// 依赖: 无
// =============================================================================

package model

import (
	"time"
)

// UserMenu 表示 zt_usermenu 用户固定菜单表。
// title/path/icon 不落库，展示时 JOIN zt_menus。
type UserMenu struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Account     string    `gorm:"column:account;size:30;not null;uniqueIndex:uk_zt_usermenu_account_menu;index:idx_zt_usermenu_account"`
	MenuID      uint64    `gorm:"column:menuId;not null;uniqueIndex:uk_zt_usermenu_account_menu;index:idx_zt_usermenu_menuId"`
	Sort        int       `gorm:"column:sort;not null;default:0"`
	CreatedBy   string    `gorm:"column:createdBy;size:30;not null;default:''"`
	CreatedDate time.Time `gorm:"column:createdDate;autoCreateTime"`
	UpdatedBy   string    `gorm:"column:updatedBy;size:30;not null;default:''"`
	UpdatedDate time.Time `gorm:"column:updatedDate;autoUpdateTime"`
}

// TableName 指定 zt_usermenu 表。
func (UserMenu) TableName() string {
	return "zt_usermenu"
}
