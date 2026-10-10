// =============================================================================
// 文件: internal/module/teamleader/repo.go
// 模块: 团队长工作台
// 类型: repo
// 职责: 禅道 zt_teamgroup / zt_team / zt_user 的只读查询与参数化检索。
// 依赖: gorm.io/gorm
// =============================================================================

package teamleader

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

// Repo 团队长模块数据访问层。
type Repo struct {
	db *gorm.DB
}

// NewRepo 创建 Repo 实例。
func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

// TeamgroupRaw 团队组原始数据行。
type TeamgroupRaw struct {
	ID      uint   `gorm:"column:id"`
	Type    string `gorm:"column:type"`
	Name    string `gorm:"column:name"`
	Manager string `gorm:"column:manager"`
	PO      string `gorm:"column:PO"`
	Parent  uint   `gorm:"column:parent"`
	Grade   int    `gorm:"column:grade"`
	Status  string `gorm:"column:status"`
}

// TeamMemberRaw 团队成员原始数据行。
type TeamMemberRaw struct {
	GroupID uint   `gorm:"column:group_id"`
	Account string `gorm:"column:account"`
	Role    string `gorm:"column:role"`
}

// FindParentTeams 查询所有有效的三级团队（type='parent'）。
func (r *Repo) FindParentTeams(ctx context.Context) ([]TeamgroupRaw, error) {
	var rows []TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("type = ? AND deleted = ?", "parent", "0").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindParentTeamByID 查询指定 ID 的三级团队。
func (r *Repo) FindParentTeamByID(ctx context.Context, id uint) (*TeamgroupRaw, error) {
	var row TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("id = ? AND type = ? AND deleted = ?", id, "parent", "0").
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// FindSubGroupsByParentID 查询指定父团队下属的所有子小组。
func (r *Repo) FindSubGroupsByParentID(ctx context.Context, parentID uint) ([]TeamgroupRaw, error) {
	if parentID == 0 {
		return []TeamgroupRaw{}, nil
	}
	var rows []TeamgroupRaw
	err := r.db.WithContext(ctx).
		Table("zt_teamgroup").
		Select("id, type, name, manager, PO, parent, grade, status").
		Where("parent = ? AND type = ? AND deleted = ?", parentID, "child", "0").
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}

// FindMembersByGroupIDs 批量参数化查询指定小组列表的所有成员。
// 当 groupIDs 为空时直接短路返回空切片，避免无效查询。
func (r *Repo) FindMembersByGroupIDs(ctx context.Context, groupIDs []uint) ([]TeamMemberRaw, error) {
	if len(groupIDs) == 0 {
		return []TeamMemberRaw{}, nil
	}
	var rows []TeamMemberRaw
	err := r.db.WithContext(ctx).
		Table("zt_team").
		Select("root AS group_id, account, COALESCE(role, '') AS role").
		Where("type = ? AND root IN ?", "teamgroup", groupIDs).
		Order("root ASC, account ASC").
		Find(&rows).Error
	return rows, err
}

// FindUserParentTeamIDsByAccount 查找用户关联的所有三级团队 ID 集合：
// 1. 直接在 parent 团队担任 manager 或 PO；
// 2. 在 child 小组担任 manager 或 PO，向上反推 parent；
// 3. 在 child 小组作为正式成员（zt_team），向上反推 parent。
func (r *Repo) FindUserParentTeamIDsByAccount(ctx context.Context, account string) ([]uint, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return []uint{}, nil
	}
	pattern := "(^|[[:space:],;])" + account + "([[:space:],;]|$)"
	type idRow struct {
		ID uint `gorm:"column:id"`
	}
	var rows []idRow

	// 联合查询三种归属路径，严格使用参数化占位符
	query := `
SELECT DISTINCT p.id
FROM zt_teamgroup p
WHERE p.type = 'parent' AND p.deleted = '0'
  AND (p.manager REGEXP ? OR p.PO = ?)
UNION
SELECT DISTINCT p.id
FROM zt_teamgroup c
JOIN zt_teamgroup p ON p.id = c.parent AND p.type = 'parent' AND p.deleted = '0'
WHERE c.type = 'child' AND c.deleted = '0'
  AND (c.manager REGEXP ? OR c.PO = ?)
UNION
SELECT DISTINCT p.id
FROM zt_team t
JOIN zt_teamgroup c ON c.id = t.root AND c.type = 'child' AND c.deleted = '0'
JOIN zt_teamgroup p ON p.id = c.parent AND p.type = 'parent' AND p.deleted = '0'
WHERE t.type = 'teamgroup' AND t.account = ?
ORDER BY id ASC`

	err := r.db.WithContext(ctx).Raw(query, pattern, account, pattern, account, account).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]uint, 0, len(rows))
	for _, it := range rows {
		out = append(out, it.ID)
	}
	return out, nil
}

// ResolveRealnames 批量解析工号列表对应的真实姓名。
func (r *Repo) ResolveRealnames(ctx context.Context, accounts []string) (map[string]string, error) {
	out := make(map[string]string)
	if len(accounts) == 0 {
		return out, nil
	}
	// 账号去重清洗
	uniq := make([]string, 0, len(accounts))
	seen := make(map[string]bool)
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a != "" && !seen[a] {
			seen[a] = true
			uniq = append(uniq, a)
		}
	}
	if len(uniq) == 0 {
		return out, nil
	}
	type userRow struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []userRow
	err := r.db.WithContext(ctx).
		Table("zt_user").
		Select("account, realname").
		Where("account IN ? AND deleted = ?", uniq, "0").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, u := range rows {
		name := strings.TrimSpace(u.Realname)
		if name == "" {
			name = u.Account
		}
		out[u.Account] = name
	}
	return out, nil
}
