// =============================================================================
// 文件: internal/model/role.go
// 模块: 数据模型
// 类型: model
// 职责: 定义角色模型字段与表映射。
// 依赖: internal/model/basemodel.go
// =============================================================================

package model

// Role 表示 zt_roles 角色表。
// 时间字段与删除标记来自 BaseModel（createdDate / updatedDate / deleted），
// 对应业务上的创建、更新与软删语义；TenantID 对应 tenantId。
type Role struct {
	BaseModel
	Code      string `gorm:"column:code;size:64;not null"`
	Name      string `gorm:"column:name;size:64;not null"`
	Remark    string `gorm:"column:description;size:255;not null;default:''"`
	IsBuiltin bool   `gorm:"column:isBuiltin;not null;default:false"`
	IsActive  bool   `gorm:"column:isActive;not null;default:true"`
	SortOrder int32  `gorm:"column:sortOrder;not null;default:0"`
}

// TableName 指定 zt_roles 表。
func (Role) TableName() string {
	return "zt_roles"
}
