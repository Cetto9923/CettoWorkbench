// =============================================================================
// 文件: internal/model/versionwindow.go
// 模块: 数据模型
// 类型: model
// 职责: 定义版本窗口及窗口-产品关联模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
	"time"

	"gorm.io/gorm"
)

// VersionWindow 表示 zt_versionwindow 版本窗口表。
type VersionWindow struct {
	ID          uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"column:name;size:100;not null" json:"name"`
	ReleaseDate time.Time      `gorm:"column:releaseDate;type:date;not null" json:"releaseDate"`
	StartDate   *time.Time     `gorm:"column:startDate;type:date" json:"startDate"`
	TeamgroupID uint           `gorm:"column:teamgroup;not null;index:idx_teamgroup" json:"teamgroupId"`
	GroupSize   uint           `gorm:"column:groupSize;not null;default:1" json:"groupSize"`
	CreatedBy   string         `gorm:"column:createdBy;size:30;not null;index:idx_createdBy" json:"createdBy"`
	UpdatedBy   string         `gorm:"column:updatedBy;size:30;not null;default:''" json:"updatedBy"`
	Status      string         `gorm:"column:status;size:20;not null;default:planning" json:"status"`
	Order       int            `gorm:"column:order;not null;default:0" json:"order"`
	CreatedDate time.Time      `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	UpdatedDate time.Time      `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deletedAt;index" json:"-"`
}

// TableName 指定 zt_versionwindow 表。
func (VersionWindow) TableName() string {
	return "zt_versionwindow"
}

// VersionWindowProduct 表示 zt_versionwindowproduct 窗口-系统关联表。
type VersionWindowProduct struct {
	ID         uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WindowID   uint64         `gorm:"column:versionWindow;not null;index:idx_versionWindow;uniqueIndex:uk_versionWindow_product" json:"windowId"`
	ProductID  uint           `gorm:"column:product;not null;index:idx_product;uniqueIndex:uk_versionWindow_product" json:"productId"`
	PlanID     *uint          `gorm:"column:plan" json:"planId"`
	PlanSynced uint8          `gorm:"column:planSynced;not null;default:0" json:"planSynced"`
	CreatedBy  string         `gorm:"column:createdBy;size:30;not null;default:''" json:"createdBy"`
	UpdatedBy  string         `gorm:"column:updatedBy;size:30;not null;default:''" json:"updatedBy"`
	CreatedDate time.Time     `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	UpdatedDate time.Time     `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deletedAt;index" json:"-"`
}

// TableName 指定 zt_versionwindowproduct 表。
func (VersionWindowProduct) TableName() string {
	return "zt_versionwindowproduct"
}
