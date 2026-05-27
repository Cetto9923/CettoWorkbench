// =============================================================================
// 文件: internal/model/cronlog.go
// 模块: 数据模型
// 类型: model
// 职责: 定义定时任务执行日志模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// CronLog 表示 zt_cron_logs 任务执行日志表。
type CronLog struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	JobName   string     `gorm:"column:jobName;size:64;not null;default:''"`
	StartAt   time.Time  `gorm:"column:startAt;not null"`
	EndAt     *time.Time `gorm:"column:endAt"`
	Success   bool       `gorm:"column:success;not null;default:false"`
	Message   string     `gorm:"column:message;type:text"`
	CreatedAt time.Time  `gorm:"column:createdAt;autoCreateTime:milli"`
}

// TableName 指定 zt_cron_logs 表。
func (CronLog) TableName() string {
	return "zt_cron_logs"
}
