// =============================================================================
// 文件: internal/model/rolepermission.go
// 模块: 数据模型
// 类型: model
// 职责: 定义角色权限关系模型字段与表映射。
// 依赖: internal/model/basemodel.go
// =============================================================================

package model

// RolePermission 表示 zt_role_permissions 角色权限关联表。
type RolePermission struct {
	BaseModel
	RoleID   int64  `gorm:"column:roleId;not null"`
	PermCode string `gorm:"column:permCode;size:64;not null;default:''"`
}

// TableName 指定 zt_role_permissions 表。
func (RolePermission) TableName() string {
	return "zt_role_permissions"
}
