// =============================================================================
// 文件: internal/module/role/repo.go
// 模块: 角色管理
// 类型: crud
// 职责: 封装角色及角色权限关系的数据访问。
// 依赖: internal/model
// =============================================================================

package role

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"workbench/internal/model"
)

// Repo 封装角色数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// RepoFindAllReq 角色列表查询条件。
type RepoFindAllReq struct {
	Keyword  string
	Page     int
	PageSize int
}

// FindAll 分页查询角色列表及总数。
func (r *Repo) FindAll(ctx context.Context, req RepoFindAllReq) ([]model.Role, int64, error) {
	db := r.db.WithContext(ctx).Model(&model.Role{}).Where("deleted = 0")
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ?", like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}

	var rows []model.Role
	offset := 0
	if req.Page > 1 {
		offset = (req.Page - 1) * req.PageSize
	}
	if err := db.Order("sortOrder ASC, id ASC").Offset(offset).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("find roles: %w", err)
	}
	return rows, total, nil
}

// FindByID 按 ID 查询未删除的角色。
func (r *Repo) FindByID(ctx context.Context, id int64) (*model.Role, error) {
	var m model.Role
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted = 0", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 插入角色。
func (r *Repo) Create(ctx context.Context, m *model.Role) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// Update 更新角色名称与备注等字段。
func (r *Repo) Update(ctx context.Context, m *model.Role) error {
	return r.db.WithContext(ctx).
		Model(&model.Role{}).
		Where("id = ? AND deleted = 0", m.ID).
		Updates(map[string]any{
			"name":        m.Name,
			"description": m.Remark,
			"updatedBy":   m.UpdatedBy,
			"updatedDate": m.UpdatedDate,
		}).
		Error
}

// Delete 在事务内移除角色权限关联并软删除角色。
func (r *Repo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("roleId = ? AND deleted = 0", id).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		return tx.Model(&model.Role{}).
			Where("id = ? AND deleted = 0", id).
			Updates(map[string]any{
				"deleted":     1,
				"updatedDate": gorm.Expr("CURRENT_TIMESTAMP"),
			}).
			Error
	})
}

// HasUsers 判断角色下是否仍有用户关联（未删除关系）。
func (r *Repo) HasUsers(ctx context.Context, roleID int64) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).
		Model(&model.UserRole{}).
		Where("roleId = ? AND deleted = 0", roleID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ExistsByName 检查租户内角色名称是否已存在。
func (r *Repo) ExistsByName(ctx context.Context, tenantID int64, name string, excludeID int64) (bool, error) {
	q := r.db.WithContext(ctx).Model(&model.Role{}).
		Where("tenantId = ? AND name = ? AND deleted = 0", tenantID, strings.TrimSpace(name))
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// ListPermissions 返回角色已绑定的权限码列表。
func (r *Repo) ListPermissions(ctx context.Context, roleID int64) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).
		Model(&model.RolePermission{}).
		Where("roleId = ? AND deleted = 0", roleID).
		Order("permCode ASC").
		Pluck("permCode", &codes).Error
	if err != nil {
		return nil, err
	}
	return dedupeNonEmptyCodes(codes), nil
}

// ListAllMenus 返回全部未删除菜单，按 sort、id 排序，供权限分配页挂载模块权限。
func (r *Repo) ListAllMenus(ctx context.Context) ([]model.Menu, error) {
	var rows []model.Menu
	err := r.db.WithContext(ctx).
		Model(&model.Menu{}).
		Where("deletedAt IS NULL").
		Order("sort ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ReplacePermissions 在事务内全量替换角色的权限绑定（先删后插）。
func (r *Repo) ReplacePermissions(ctx context.Context, roleID int64, permCodes []string) error {
	uniq := dedupeNonEmptyCodes(permCodes)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("roleId = ? AND deleted = 0", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if len(uniq) == 0 {
			return nil
		}
		rows := make([]model.RolePermission, 0, len(uniq))
		for _, code := range uniq {
			if strings.TrimSpace(code) == "" {
				return fmt.Errorf("存在无效的权限编码")
			}
			rows = append(rows, model.RolePermission{
				RoleID:   roleID,
				PermCode: code,
			})
		}
		return tx.Create(&rows).Error
	})
}

func dedupeNonEmptyCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}
