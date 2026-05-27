// =============================================================================
// 文件: internal/model/cronjob.go
// 模块: 数据模型
// 类型: model
// 职责: 定义定时任务模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// CronJob 表示 zt_cron_jobs 定时任务配置表。
type CronJob struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string     `gorm:"column:name;size:64;not null;default:''"`
	Spec      string     `gorm:"column:spec;size:64;not null;default:''"`
	IsEnabled bool       `gorm:"column:isEnabled;not null;default:true"`
	Remark    string     `gorm:"column:remark;size:255;not null;default:''"`
	LastRunAt *time.Time `gorm:"column:lastRunAt"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli"`
	UpdatedAt time.Time  `gorm:"column:updatedAt;autoUpdateTime:milli"`
}

// TableName 指定 zt_cron_jobs 表。
func (CronJob) TableName() string {
	return "zt_cron_jobs"
}
