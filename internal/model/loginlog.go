// =============================================================================
// 文件: internal/model/loginlog.go
// 模块: 数据模型
// 类型: model
// 职责: 定义登录审计日志模型及表映射。
// 依赖: 无
// =============================================================================

package model

import (
	"database/sql"
	"time"
)

// LoginLog 表示一次登录行为（成功或失败）。
type LoginLog struct {
	ID         int64         `gorm:"primaryKey;autoIncrement"`
	Account    string        `gorm:"column:account"`
	UserID     sql.NullInt64 `gorm:"column:userId"`
	IP         string        `gorm:"column:ip"`
	UserAgent  string        `gorm:"column:userAgent"`
	Success    bool          `gorm:"column:success"`
	FailReason string        `gorm:"column:failReason"`
	CreatedAt  time.Time     `gorm:"column:createdDate"`
}

// TableName 指定 zt_login_logs 表。
func (LoginLog) TableName() string {
	return "zt_login_logs"
}
