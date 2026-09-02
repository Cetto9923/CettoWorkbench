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

// disabledLockedUntil 表示管理员禁用账号时写入 locked 的占位时间。
var disabledLockedUntil = time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

// User 表示禅道 zt_user 用户表。
type User struct {
	ID            int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Company       uint64     `gorm:"column:company;not null;default:0"`
	Type          string     `gorm:"column:type;size:30;not null;default:inside"`
	DeptID        uint64     `gorm:"column:dept;not null;default:0"`
	Account       string     `gorm:"column:account;size:30;not null"`
	PasswordHash  string     `gorm:"column:password;size:32;not null"`
	Role          string     `gorm:"column:role;size:10;not null;default:''"`
	DisplayName   string     `gorm:"column:realname;size:100;not null"`
	Gender        string     `gorm:"column:gender;type:enum('f','m');not null;default:f"`
	Email         string     `gorm:"column:email;size:90;not null"`
	Phone         string     `gorm:"column:phone;size:20;not null"`
	Mobile        string     `gorm:"column:mobile;size:11;not null"`
	Deleted       string     `gorm:"column:deleted;type:enum('0','1');not null;default:0;index"`
	Locked        *time.Time `gorm:"column:locked"`
	LastLoginUnix uint32     `gorm:"column:last;not null;default:0"`
	LastLoginIP   string     `gorm:"column:ip;size:255;not null"`

	// ── 以下字段不持久化，由 AfterFind 根据 DB 字段计算回填 ──────────────
	// 禁止对这些字段添加 gorm tag 或在 Where 条件中直接引用。
	IsSuperAdmin    bool       `gorm:"-"`
	IsActive        bool       `gorm:"-"`
	IsActiveDB      bool       `gorm:"-"`
	IsSuperAdminDB  bool       `gorm:"-"`
	LastLoginDate   *time.Time `gorm:"-"`
	LastLoginDateDB *time.Time `gorm:"-"`
}

// TableName 指定 zt_user 表。
func (User) TableName() string {
	return "zt_user"
}

// AfterFind 回填兼容状态字段。
func (u *User) AfterFind(_ *gorm.DB) error {
	now := time.Now()
	u.IsActive = u.Deleted == "0" && (u.Locked == nil || u.Locked.Before(now))
	u.IsActiveDB = u.IsActive
	u.IsSuperAdmin = u.Account == "admin" || u.Role == "admin"
	u.IsSuperAdminDB = u.IsSuperAdmin
	if u.LastLoginUnix > 0 {
		t := time.Unix(int64(u.LastLoginUnix), 0)
		u.LastLoginDate = &t
		u.LastLoginDateDB = &t
	}
	return nil
}

// SetActive 更新用户启用状态（通过 locked 字段映射禅道禁用语义）。
func (u *User) SetActive(active bool) {
	u.IsActiveDB = active
	if active {
		u.Locked = nil
		return
	}
	locked := disabledLockedUntil
	u.Locked = &locked
}

// LockedForActive 返回与启用状态对应的 locked 列值。
func (u *User) LockedForActive() *time.Time {
	if u.IsActiveDB {
		return nil
	}
	locked := disabledLockedUntil
	return &locked
}
