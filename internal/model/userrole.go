// =============================================================================
// 文件: internal/model/userrole.go
// 模块: 数据模型
// 类型: model
// 职责: 定义用户角色关系模型字段与表映射。
// 依赖: internal/model/basemodel.go
// =============================================================================

package model

// UserRole 表示 zt_gf_user_roles 用户角色关联表。
type UserRole struct {
	BaseModel
	UserID int64 `gorm:"column:userId;not null"`
	RoleID int64 `gorm:"column:roleId;not null"`
}

// TableName 指定 zt_gf_user_roles 表。
func (UserRole) TableName() string {
	return "zt_gf_user_roles"
}
