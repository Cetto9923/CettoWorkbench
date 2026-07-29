// =============================================================================
// 文件: internal/pkg/perm/loader.go
// 模块: 基础设施
// 类型: infra
// 职责: 提供基于用户 ID 的权限集合加载能力。
// 依赖: 无
// =============================================================================

package perm

import (
	"context"

	"gorm.io/gorm"
)

// LoadUserPermissionSet 按用户 ID 从 RBAC 关系表加载权限集合。
// 返回值 key 为权限 code，value 固定为 true。
func LoadUserPermissionSet(ctx context.Context, db *gorm.DB, userID int64) (map[string]bool, error) {
	set := make(map[string]bool)
	if db == nil || userID <= 0 {
		return set, nil
	}

	var permCodes []string
	err := db.WithContext(ctx).
		Table("zt_role_permissions rp").
		Distinct("rp.permCode").
		Joins("JOIN zt_gf_user_roles ur ON ur.roleId = rp.roleId").
		Joins("JOIN zt_roles r ON r.id = ur.roleId").
		Where("ur.userId = ?", userID).
		Where("r.isActive = ?", true).
		Scan(&permCodes).
		Error
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(All()))
	for _, p := range All() {
		allowed[p.String()] = struct{}{}
	}
	for _, code := range permCodes {
		if _, ok := allowed[code]; ok {
			set[code] = true
		}
	}
	return set, nil
}
