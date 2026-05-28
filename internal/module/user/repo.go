// =============================================================================
// 文件: internal/module/user/repo.go
// 模块: 用户
// 类型: crud
// 职责: 封装用户数据访问。
// 依赖: internal/model
// =============================================================================

package user

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装用户数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// ListQuery 定义列表查询条件。
type ListQuery struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

type RepoFindAllReq struct {
	Account     string
	Email       string
	DisplayName string
	IsActive    *bool
	Page        int
	PageSize    int
}

// FindAll 按条件查询用户列表及总数。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.User, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.User{}).Where("deleted = 0")
	account := strings.TrimSpace(req.Account)
	if account != "" {
		db = db.Where("account LIKE ?", "%"+account+"%")
	}
	email := strings.TrimSpace(req.Email)
	if email != "" {
		db = db.Where("email LIKE ?", "%"+email+"%")
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName != "" {
		db = db.Where("realname LIKE ?", "%"+displayName+"%")
	}

	if req.IsActive != nil {
		// 直接传递 bool，GORM 正确映射至 tinyint(1)，无需中间 int 变量
		db = db.Where("isActive = ?", *req.IsActive)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	var users []model.User
	offset := 0
	if req.Page > 1 {
		offset = (req.Page - 1) * req.PageSize
	}
	if err := db.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("find users: %w", err)
	}
	return users, total, nil
}

// FindAllForExport 查询导出所需的全部用户记录。
func (r *Repo) FindAllForExport(ctx context.Context) ([]model.User, error) {
	var users []model.User
	if err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("deleted = 0").
		Order("id DESC").
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("find users for export: %w", err)
	}
	return users, nil
}

// FindByID 按 ID 查询用户详情。
func (r *Repo) FindByID(ctx context.Context, id int64) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted = 0", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建用户。
func (r *Repo) Create(ctx context.Context, m *model.User) error {
	return r.createWithTx(r.db.WithContext(ctx), m)
}

func (r *Repo) createWithTx(tx *gorm.DB, m *model.User) error {
	// 某些历史库未包含 lastLoginDate/lastLoginIP 列。
	// 创建用户时使用显式列白名单，避免 GORM 根据模型字段自动拼接不存在的列导致 1054。
	if err := tx.
		Model(&model.User{}).
		Create(map[string]any{
			"account":      m.Account,
			"email":        m.Email,
			"password":     m.PasswordHash,
			"realname":     m.DisplayName,
			"gender":       m.Gender,
			"position":     m.Position,
			"manager":      m.ManagerID,
			"phone":        m.Phone,
			"createdBy":    m.CreatedByName,
			"updatedBy":    m.UpdatedByName,
			"isSuperAdmin": m.IsSuperAdminDB,
			"isActive":     m.IsActiveDB,
			"deptID":       m.DeptID,
		}).
		Error; err != nil {
		return err
	}

	var id int64
	if err := tx.
		Table("zt_gf_user").
		Select("id").
		Where("account = ? AND deleted = 0", m.Account).
		Order("id DESC").
		Limit(1).
		Scan(&id).
		Error; err != nil {
		return err
	}
	m.ID = id
	return nil
}

// Update 更新用户。
func (r *Repo) Update(ctx context.Context, m *model.User) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = 0", m.ID).
		Updates(map[string]any{
			"account":     m.Account,
			"email":       m.Email,
			"realname":    m.DisplayName,
			"isActive":    m.IsActiveDB,
			"deptID":      m.DeptID,
			"updatedBy":   m.UpdatedByName,
			"updatedDate": m.UpdatedAt,
		}).
		Error
}

// Delete 软删除用户（deleted=1）。
// 未匹配到记录时返回 nil error。
func (r *Repo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = 0", id).
		Updates(map[string]any{
			"deleted":     1,
			"updatedDate": gorm.Expr("NOW()"),
		}).
		Error
}

// UpdateStatus 按 ID 更新用户启用状态。
func (r *Repo) UpdateStatus(ctx context.Context, id uint64, isActive bool) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = 0", id).
		Updates(map[string]any{
			// isActive -> model.User.IsActiveDB
			"isActive": isActive,
		}).
		Error
}

// UpdatePassword 按 ID 更新用户密码哈希。
func (r *Repo) UpdatePassword(ctx context.Context, id uint64, hashedPwd string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = 0", id).
		Updates(map[string]any{
			// password -> model.User.PasswordHash
			"password": hashedPwd,
		}).
		Error
}

// ExistsByAccount 检查账号是否已存在。
func (r *Repo) ExistsByAccount(ctx context.Context, account string, excludeID int64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.User{}).Where("account = ? AND deleted = 0", account)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListRoles 返回可分配的启用角色。
func (r *Repo) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("isActive = ?", 1).
		Order("sortOrder ASC, id ASC").
		Find(&roles).
		Error
	return roles, err
}

// ListUserRoleIDs 查询用户已分配角色。
func (r *Repo) ListUserRoleIDs(ctx context.Context, userID int64) ([]int64, error) {
	var roleIDs []int64
	err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("userId = ?", userID).
		Pluck("roleId", &roleIDs).
		Error
	return roleIDs, err
}

// ReplaceUserRoles 全量替换用户角色关系。
func (r *Repo) ReplaceUserRoles(ctx context.Context, userID int64, roleIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.replaceUserRolesWithTx(tx, userID, roleIDs)
	})
}

func (r *Repo) replaceUserRolesWithTx(tx *gorm.DB, userID int64, roleIDs []int64) error {
	if err := tx.Where("userId = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]model.UserRole, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, model.UserRole{
			UserID: userID,
			RoleID: roleID,
		})
	}
	return tx.Create(&rows).Error
}

// BatchCreate 批量创建用户并分配角色（单事务）。
func (r *Repo) BatchCreate(ctx context.Context, users []*model.User, roleIDsList [][]int64) error {
	if len(users) == 0 {
		return nil
	}
	if len(users) != len(roleIDsList) {
		return fmt.Errorf("batch create users: roleIDs length mismatch")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range users {
			if err := r.createWithTx(tx, users[i]); err != nil {
				return err
			}
			if err := r.replaceUserRolesWithTx(tx, users[i].ID, roleIDsList[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// ListDepts 返回可选部门列表。
func (r *Repo) ListDepts(ctx context.Context) ([]model.Dept, error) {
	var depts []model.Dept
	err := r.db.WithContext(ctx).
		Model(&model.Dept{}).
		Where("deletedAt IS NULL").
		Order("sort ASC, id ASC").
		Find(&depts).Error
	return depts, err
}
