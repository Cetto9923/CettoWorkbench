// =============================================================================
// 文件: internal/module/profile/repo.go
// 模块: 个人资料
// 类型: action
// 职责: 读写 zt_user 资料字段、mainTeam，加载本人已加入的敏捷小组，以及读写 zt_wb_profile_prefs 自选视图偏好。
// 依赖: gorm.io/gorm
// =============================================================================

package profile

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"workbench/internal/model"
)

// Repo 个人资料数据访问。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

type profileRow struct {
	ID       int64  `gorm:"column:id"`
	Account  string `gorm:"column:account"`
	Realname string `gorm:"column:realname"`
	Email    string `gorm:"column:email"`
	Mobile   string `gorm:"column:mobile"`
	Gender   string `gorm:"column:gender"`
	DeptID   uint64 `gorm:"column:dept"`
	DeptName string `gorm:"column:deptName"`
	MainTeam uint64 `gorm:"column:mainTeam"`
}

// FindByID 读取用户资料（含部门名与默认小组）。
func (r *Repo) FindByID(ctx context.Context, id int64) (*profileRow, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("profile repo is not configured")
	}
	var row profileRow
	err := r.db.WithContext(ctx).Raw(`
SELECT u.id, u.account, u.realname, u.email, u.mobile, u.gender, u.dept, u.mainTeam,
       COALESCE(d.name, '') AS deptName
FROM zt_user u
LEFT JOIN zt_dept d ON d.id = u.dept
WHERE u.id = ? AND u.deleted = '0'
LIMIT 1`, id).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("find profile: %w", err)
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

// FindUserAgileGroups 本人已加入的敏捷小组（对齐禅道 getByAccountTeam）。
// 对应 SQL：zt_teamgroup tg INNER JOIN zt_team t ON t.root = tg.id AND t.type='teamgroup'。
func (r *Repo) FindUserAgileGroups(ctx context.Context, account string) ([]AgileGroupOption, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("profile repo is not configured")
	}
	var rows []AgileGroupOption
	err := r.db.WithContext(ctx).Raw(`
SELECT tg.id AS id, tg.name AS name
FROM zt_teamgroup tg
INNER JOIN zt_team t ON t.root = tg.id AND t.type = 'teamgroup'
WHERE t.account = ? AND tg.deleted = '0'
ORDER BY tg.id ASC`, account).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find agile groups: %w", err)
	}
	if rows == nil {
		rows = []AgileGroupOption{}
	}
	return rows, nil
}

// UpdateSelfContact 更新邮箱与性别。
// gender 传 "" 且 genderSkip 为 true 时，跳过 gender 字段更新。
func (r *Repo) UpdateSelfContact(ctx context.Context, id int64, email, gender string, genderSkip bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("profile repo is not configured")
	}
	updates := map[string]any{"email": email}
	if !genderSkip {
		updates["gender"] = gender
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = ?", id, "0").
		Updates(updates).Error
}

// UpdateMainTeam 更新禅道默认小组 zt_user.mainTeam。
func (r *Repo) UpdateMainTeam(ctx context.Context, id int64, mainTeamID uint64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("profile repo is not configured")
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = ?", id, "0").
		Updates(map[string]any{"mainTeam": mainTeamID}).Error
}

// UpdateMobile 更新手机号。
func (r *Repo) UpdateMobile(ctx context.Context, id int64, mobile string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("profile repo is not configured")
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = ?", id, "0").
		Updates(map[string]any{"mobile": mobile}).Error
}

// UpdateDisplayName 更新显示名（realname）。
func (r *Repo) UpdateDisplayName(ctx context.Context, id int64, realname string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("profile repo is not configured")
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = ?", id, "0").
		Updates(map[string]any{"realname": realname}).Error
}

// FindPreferredRoles 读取自选视图偏好。
func (r *Repo) FindPreferredRoles(ctx context.Context, account string) ([]string, error) {
	if r == nil || r.db == nil || account == "" {
		return []string{}, nil
	}
	var raw string
	err := r.db.WithContext(ctx).
		Table("zt_wb_profile_prefs").
		Select("preferredRoles").
		Where("account = ?", account).
		Limit(1).
		Scan(&raw).Error
	if err != nil {
		// 降级返回空切片，保证在迁移未完成时界面正常加载
		return []string{}, nil
	}
	return splitRoles(raw), nil
}

// UpsertPreferredRoles 写入自选视图偏好。
func (r *Repo) UpsertPreferredRoles(ctx context.Context, account string, roles []string) error {
	if r == nil || r.db == nil || account == "" {
		return nil
	}
	joined := strings.Join(roles, ",")
	return r.db.WithContext(ctx).Exec(`
INSERT INTO zt_wb_profile_prefs (account, preferredRoles)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE preferredRoles = VALUES(preferredRoles), updatedDate = CURRENT_TIMESTAMP`,
		account, joined).Error
}

// FindUserOrgRoles retrieves organization-assigned roles for a user.
func (r *Repo) FindUserOrgRoles(ctx context.Context, account string) ([]RoleOption, error) {
	if r == nil || r.db == nil || account == "" {
		return []RoleOption{}, nil
	}
	var rows []struct {
		RoleKey string `gorm:"column:role_key"`
		Label   string `gorm:"column:label"`
	}
	err := r.db.WithContext(ctx).Raw(`
SELECT r.code AS role_key, r.name AS label
FROM zt_user u
JOIN zt_gf_user_roles ur ON ur.userId = u.id
JOIN zt_roles r ON r.id = ur.roleId
WHERE u.account = ? AND u.deleted = '0' AND ur.deleted = '0' AND r.deleted = '0' AND r.isActive = '1'
ORDER BY r.sortOrder ASC, r.id ASC`, account).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("find org roles: %w", err)
	}
	opts := make([]RoleOption, len(rows))
	for i, row := range rows {
		opts[i] = RoleOption{Key: row.RoleKey, Label: row.Label}
	}
	return opts, nil
}

// FindPasswordHash 读取密码哈希（MD5 hex，32 字符）。
func (r *Repo) FindPasswordHash(ctx context.Context, id int64) (string, error) {
	if r == nil || r.db == nil {
		return "", fmt.Errorf("profile repo is not configured")
	}
	var hash string
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("password").
		Where("id = ? AND deleted = ?", id, "0").
		Scan(&hash).Error
	if err != nil {
		return "", fmt.Errorf("find password hash: %w", err)
	}
	if hash == "" {
		return "", gorm.ErrRecordNotFound
	}
	return hash, nil
}

// UpdatePassword 更新本人密码哈希。
func (r *Repo) UpdatePassword(ctx context.Context, id int64, hashed string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("profile repo is not configured")
	}
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND deleted = ?", id, "0").
		Updates(map[string]any{"password": hashed}).Error
}

func splitRoles(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		key := strings.ToLower(strings.TrimSpace(p))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
