// =============================================================================
// 文件: internal/module/profile/repo.go
// 模块: 个人资料
// 类型: action
// 职责: 读写 zt_user 资料字段、mainTeam，加载本人已加入的敏捷小组，以及读写工作台偏好。
// 依赖: gorm.io/gorm
// =============================================================================

package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"workbench/internal/model"
	"workbench/internal/pkg/workbenchroles"
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

func (r *Repo) SaveSelfProfile(ctx context.Context, actor *model.User, req UpdateReq, preferred []string) error {
	fields := map[string]any{"email": req.Email, "mainTeam": req.MainTeamID}
	if !req.ApplyGenderSkip() {
		fields["gender"] = req.Gender
	}
	if req.DisplayName != "" {
		fields["realname"] = req.DisplayName
	}
	if req.Mobile != "" {
		fields["mobile"] = req.Mobile
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).Where("id = ? AND account = ? AND deleted = ?", actor.ID, actor.Account, "0").Updates(fields).Error; err != nil {
			return err
		}
		return (&Repo{db: tx}).UpsertPreferredRoles(ctx, actor.Account, preferred)
	})
}

// FindPreferredRoles 读取自选视图偏好。
func (r *Repo) FindPreferredRoles(ctx context.Context, account string) ([]string, error) {
	if r == nil || r.db == nil || account == "" {
		return []string{}, nil
	}
	value, found, err := r.ReadPreference(ctx, account, "preferred_roles")
	if err != nil {
		return nil, err
	}
	if !found {
		// Keep existing deployments compatible: older releases stored role names
		// as a comma-separated value in zt_wb_profile_prefs.preferredRoles.
		var legacy string
		err := r.db.WithContext(ctx).Table("zt_wb_profile_prefs").Select("preferredRoles").Where("account = ?", account).Limit(1).Scan(&legacy).Error
		if err != nil || legacy == "" {
			return []string{}, nil
		}
		roles := splitRoles(legacy)
		return roles, nil
	}
	var roles []string
	if err := json.Unmarshal(value, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// UpsertPreferredRoles stores validated display roles using the unified schema.
func (r *Repo) UpsertPreferredRoles(ctx context.Context, account string, roles []string) error {
	if r == nil || r.db == nil || account == "" {
		return fmt.Errorf("profile repo is not configured")
	}
	raw, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	return r.WritePreference(ctx, account, "preferred_roles", raw)
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
		opts[i] = RoleOption{Key: row.RoleKey, Label: workbenchroles.DisplayLabel(row.RoleKey, row.Label)}
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
