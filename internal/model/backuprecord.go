// =============================================================================
// 文件: internal/model/backuprecord.go
// 模块: 数据模型
// 类型: model
// 职责: 定义数据库备份记录模型与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// BackupRecord 表示 zt_backup_records 备份记录表。
type BackupRecord struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Filename    string    `gorm:"column:filename;size:255;not null;default:''"`
	SizeByte    int64     `gorm:"column:sizeByte;not null;default:0"`
	CreatedBy   int64     `gorm:"column:createdBy;not null;default:0"`
	CreatedDate time.Time `gorm:"column:createdDate;autoCreateTime:milli"`
	UpdatedBy   int64     `gorm:"column:updatedBy;not null;default:0"`
	UpdatedDate time.Time `gorm:"column:updatedDate;autoUpdateTime:milli"`
	Deleted     bool      `gorm:"column:deleted;not null;default:false"`
}

// TableName 指定 zt_backup_records 表。
func (BackupRecord) TableName() string {
	return "zt_backup_records"
}
