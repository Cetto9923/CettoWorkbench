// =============================================================================
// 文件: internal/model/basemodel.go
// 模块: 数据模型
// 类型: model
// 职责: 定义模型公共字段基类。
// 依赖: 无
// =============================================================================

package model

import "time"

import "gorm.io/gorm"

// BaseModel 定义业务模型公共字段。
type BaseModel struct {
	ID int64 `gorm:"column:id;primaryKey;autoIncrement`
	// TODO 后续删除，修改为DeletedAt
	// Deleted     uint8          `gorm:"column:deleted;not null;default:0;index"`
	CreatedBy   string         `gorm:"column:createdBy;size:30;not null"`
	CreatedDate time.Time      `gorm:"column:createdDate;autoCreateTime"`
	UpdatedBy   string         `gorm:"column:updatedBy;size:30;not null"`
	UpdatedDate time.Time      `gorm:"column:updatedDate;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deletedAt"`
}
