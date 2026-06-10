// =============================================================================
// 文件: internal/model/versionwindow.go
// 模块: 数据模型
// 类型: model
// 职责: 定义版本窗口及窗口-产品关联模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// VersionWindow 表示 version_window 版本窗口表。
type VersionWindow struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"column:name;size:100;not null" json:"name"`
	ReleaseDate time.Time  `gorm:"column:release_date;type:date;not null" json:"releaseDate"`
	StartDate   *time.Time `gorm:"column:start_date;type:date" json:"startDate"`
	TeamgroupID uint       `gorm:"column:teamgroup_id;not null;index:idx_teamgroup" json:"teamgroupId"`
	GroupSize   uint       `gorm:"column:group_size;not null;default:1" json:"groupSize"`
	CreatedBy   string     `gorm:"column:created_by;size:30;not null;index:idx_created_by" json:"createdBy"`
	Status      string     `gorm:"column:status;size:20;not null;default:planning" json:"status"`
	SortOrder   int        `gorm:"column:sort_order;not null;default:0" json:"sortOrder"`
	Deleted     uint8      `gorm:"column:deleted;not null;default:0" json:"deleted"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

// TableName 指定 version_window 表。
func (VersionWindow) TableName() string {
	return "version_window"
}

// VersionWindowProduct 表示 version_window_product 窗口-系统关联表。
type VersionWindowProduct struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	WindowID   uint64    `gorm:"column:window_id;not null;index:idx_window;uniqueIndex:uk_window_product" json:"windowId"`
	ProductID  uint      `gorm:"column:product_id;not null;index:idx_product;uniqueIndex:uk_window_product" json:"productId"`
	PlanID     *uint     `gorm:"column:plan_id" json:"planId"`
	PlanSynced uint8     `gorm:"column:plan_synced;not null;default:0" json:"planSynced"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

// TableName 指定 version_window_product 表。
func (VersionWindowProduct) TableName() string {
	return "version_window_product"
}
