// =============================================================================
// 文件: internal/model/user.go
// 模块: 数据模型
// 类型: model
// 职责: 定义用户模型字段与表映射。
// 依赖: 无
// =============================================================================

package model

import (
	"time"

	"gorm.io/gorm"
)

// User 表示 zt_gf_user 用户表。
type User struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Account        string    `gorm:"column:account;size:30;not null"`
	PasswordHash   string    `gorm:"column:password;size:60;not null"`
	DisplayName    string    `gorm:"column:realname;size:30;not null"`
	Gender         string    `gorm:"column:gender;type:enum('f','m');not null;default:m"`
	Position       string    `gorm:"column:position;size:30;not null"`
	ManagerID      int64     `gorm:"column:manager;not null;default:0"`
	Phone          string    `gorm:"column:phone;size:20;not null"`
	Email          string    `gorm:"column:email;size:60;not null"`
	CreatedByName  string    `gorm:"column:createdBy;size:30;not null"`
	CreatedAt      time.Time `gorm:"column:createdDate;autoCreateTime"`
	UpdatedByName  string    `gorm:"column:updatedBy;size:30;not null"`
	UpdatedAt      time.Time `gorm:"column:updatedDate;autoUpdateTime"`
	Deleted        uint8     `gorm:"column:deleted;not null;default:0;index"`
	IsSuperAdminDB bool      `gorm:"column:isSuperAdmin;not null;default:false"`
	// IsActiveDB 是数据库持久化列，对应 isActive 字段。
	// 业务代码请使用 AfterFind 回填的 IsActive 字段（非持久化），
	// 该字段屏蔽了软删除用户，能正确反映用户实际可用状态。
	IsActiveDB      bool       `gorm:"column:isActive;not null;default:true"`
	DeptID          uint64     `gorm:"column:deptID;not null;default:0"`
	LastLoginDateDB *time.Time `gorm:"column:lastLoginDate"`
	LastLoginIPDB   string     `gorm:"column:lastLoginIP;size:45"`

	// ── 以下字段不持久化，由 AfterFind 根据 DB 字段计算回填 ──────────────
	// 禁止对这些字段添加 gorm tag 或在 Where 条件中直接引用。
	IsSuperAdmin  bool       `gorm:"-"`
	IsActive      bool       `gorm:"-"`
	LastLoginDate *time.Time `gorm:"-"`
	LastLoginIP   string     `gorm:"-"`
}

// TableName 指定 zt_gf_user 表。
func (User) TableName() string {
	return "zt_gf_user"
}

// AfterFind 回填兼容状态字段。
func (u *User) AfterFind(_ *gorm.DB) error {
	u.IsSuperAdmin = u.IsSuperAdminDB
	// 用户可用状态由 isActive 字段直接决定，删除状态由 deleted 字段在查询层过滤。
	u.IsActive = u.IsActiveDB
	u.LastLoginDate = u.LastLoginDateDB
	u.LastLoginIP = u.LastLoginIPDB
	return nil
}

// SetActive 更新用户启用状态。
func (u *User) SetActive(active bool) {
	u.IsActiveDB = active
}
