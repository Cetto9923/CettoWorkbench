// =============================================================================
// 文件: internal/model/operationlog.go
// 模块: 数据模型
// 类型: model
// 职责: 定义操作日志模型及表映射。
// 依赖: 无
// =============================================================================

package model

import "time"

// OperationLog 表示一次操作审计记录。
type OperationLog struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	TenantID   int64     `gorm:"column:tenantId;not null;default:0"`
	UserID     uint64    `gorm:"column:userId;not null;default:0"`
	Account    string    `gorm:"column:account;type:varchar(64);not null;default:''"`
	Method     string    `gorm:"column:method;type:varchar(10);not null;default:''"`
	Path       string    `gorm:"column:path;type:varchar(255);not null;default:''"`
	Query      string    `gorm:"column:query;type:varchar(500);not null;default:''"`
	Body       string    `gorm:"column:body;type:text"`
	IP         string    `gorm:"column:ip;type:varchar(64);not null;default:''"`
	UserAgent  string    `gorm:"column:userAgent;type:varchar(255);not null;default:''"`
	StatusCode int       `gorm:"column:statusCode;not null;default:0"`
	CreatedAt  time.Time `gorm:"column:createdAt;autoCreateTime:milli"`
}

// TableName 指定 zt_operation_logs 表。
func (OperationLog) TableName() string {
	return "zt_operation_logs"
}
