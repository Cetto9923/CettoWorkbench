// =============================================================================
// 文件: internal/model/demandwindow.go
// 模块: 数据模型
// 类型: model
// 职责: 定义业务需求-窗口关联模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
	"time"

	"gorm.io/gorm"
)

// DemandWindow 表示 zt_demandwindow 业务需求-窗口关联表。
// 语义为「业需级单值」：demand + story(=0) 唯一确定一条当前窗口归属。
type DemandWindow struct {
	ID          uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DemandID    uint           `gorm:"column:demand;not null;index:idx_demand;uniqueIndex:uk_demand_story" json:"demandId"`
	StoryID     uint           `gorm:"column:story;not null;default:0;index:idx_story;uniqueIndex:uk_demand_story" json:"storyId"`
	WindowID    uint64         `gorm:"column:versionWindow;not null;index:idx_versionWindow" json:"windowId"`
	CreatedBy   string         `gorm:"column:createdBy;size:30;not null;default:''" json:"createdBy"`
	UpdatedBy   string         `gorm:"column:updatedBy;size:30;not null;default:''" json:"updatedBy"`
	CreatedDate time.Time      `gorm:"column:createdDate;autoCreateTime" json:"createdDate"`
	UpdatedDate time.Time      `gorm:"column:updatedDate;autoUpdateTime" json:"updatedDate"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deletedAt;index" json:"-"`
}

// TableName 指定 zt_demandwindow 表。
func (DemandWindow) TableName() string {
	return "zt_demandwindow"
}
