// =============================================================================
// 文件: internal/model/permission.go
// 模块: 数据模型
// 类型: model
// 职责: 定义权限模型字段与表映射。
// 依赖: internal/model/basemodel.go
// =============================================================================

package model

// Permission 表示 zt_permissions 权限表。
type Permission struct {
	BaseModel
	Code        string `gorm:"column:code;size:64;not null"`
	Name        string `gorm:"column:name;size:64;not null"`
	Module      string `gorm:"column:module;size:64;not null;default:''"`
	Description string `gorm:"column:description;size:255;not null;default:''"`
}

// TableName 指定 zt_permissions 表。
func (Permission) TableName() string {
	return "zt_permissions"
}
