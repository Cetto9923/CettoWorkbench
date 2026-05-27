// =============================================================================
// 文件: internal/model/loginfailure.go
// 模块: 数据模型
// 类型: model
// 职责: 定义登录失败记录模型及表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// LoginFailure 表示一次登录失败事件。
type LoginFailure struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Account   string    `gorm:"column:account"`
	IP        string    `gorm:"column:ip"`
	FailedAt  time.Time `gorm:"column:failedAt"`
	CreatedAt time.Time `gorm:"column:createdDate"`
}

// TableName 指定 zt_login_failures 表。
func (LoginFailure) TableName() string {
	return "zt_login_failures"
}
