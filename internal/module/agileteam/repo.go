// =============================================================================
// 文件: internal/module/agileteam/repo.go
// 模块: 敏捷小组治理
// 类型: action
// 职责: 读取禅道 zt_teamgroup / zt_team / zt_user（正式真源只读 + 基本信息更新 + 确认后写成员）。
// 依赖: 无
// =============================================================================

package agileteam

import (
	"context"
	"html"
	"strings"

	"gorm.io/gorm"
)

// TeamgroupRow 禅道敏捷小组详情行。
type TeamgroupRow struct {
	ID          uint   `gorm:"column:id"`
	Name        string `gorm:"column:name"`
	Parent      uint   `gorm:"column:parent"`
	ParentName  string `gorm:"column:parent_name"`
	Type        string `gorm:"column:type"`
	Grade       int    `gorm:"column:grade"`
	Path        string `gorm:"column:path"`
	PO          string `gorm:"column:PO"`
	Manager     string `gorm:"column:manager"`
	Slogan      string `gorm:"column:slogan"`
	Declaration string `gorm:"column:declaration"`
	Logo        string `gorm:"column:logo"`
	Status      string `gorm:"column:status"`
	CreatedDate string `gorm:"column:createdDate"`
}

// TeamMemberRow 正式成员行。
type TeamMemberRow struct {
	Account string  `gorm:"column:account"`
	Name    string  `gorm:"column:name"`
	Role    string  `gorm:"column:role"`
	Hours   float64 `gorm:"column:hours"`
	Days    int     `gorm:"column:days"`
	Join    string  `gorm:"column:join_date"`
}

// Repo 敏捷小组数据访问。
type Repo struct {
	db            *gorm.DB
	dbRead        *gorm.DB
	tableOverride string
}

// NewRepo 创建 Repo；dbRead 为空时回退到 db。
func NewRepo(db, dbRead *gorm.DB) *Repo {
	if dbRead == nil {
		dbRead = db
	}
	return &Repo{db: db, dbRead: dbRead}
}

func (r *Repo) read() *gorm.DB {
	if r.dbRead != nil {
		return r.dbRead
	}
	return r.db
}

// ListTeamgroups 查询敏捷小组列表（含父组名）。
func (r *Repo) ListTeamgroups(ctx context.Context) ([]TeamgroupRow, error) {
	var rows []TeamgroupRow
	err := r.read().WithContext(ctx).Raw(`
SELECT tg.id, tg.name, tg.parent,
       COALESCE(p.name, '') AS parent_name,
       tg.type, tg.grade, COALESCE(tg.path, '') AS path,
       tg.PO, tg.manager, tg.slogan, tg.declaration, tg.logo, tg.status,
       COALESCE(DATE_FORMAT(tg.createdDate, '%Y-%m-%d'), '') AS createdDate
FROM zt_teamgroup tg
LEFT JOIN zt_teamgroup p ON p.id = tg.parent AND p.deleted = '0'
WHERE tg.deleted = '0'
ORDER BY tg.id ASC`).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []TeamgroupRow{}
	}
	for i := range rows {
		rows[i].Name = html.UnescapeString(rows[i].Name)
		rows[i].ParentName = html.UnescapeString(rows[i].ParentName)
	}
	return rows, nil
}

// FindTeamgroupsByIDs 按 ID 批量查小组名称。
func (r *Repo) FindTeamgroupsByIDs(ctx context.Context, ids []uint) ([]TeamgroupRow, error) {
	out := []TeamgroupRow{}
	if len(ids) == 0 {
		return out, nil
	}
	err := r.read().WithContext(ctx).Raw(`
SELECT tg.id, tg.name, tg.parent,
       COALESCE(p.name, '') AS parent_name,
       tg.type, tg.grade, COALESCE(tg.path, '') AS path,
       tg.PO, tg.manager, tg.slogan, tg.declaration, tg.logo, tg.status,
       COALESCE(DATE_FORMAT(tg.createdDate, '%Y-%m-%d'), '') AS createdDate
FROM zt_teamgroup tg
LEFT JOIN zt_teamgroup p ON p.id = tg.parent AND p.deleted = '0'
WHERE tg.deleted = '0' AND tg.id IN ?
ORDER BY tg.id ASC`, ids).Scan(&out).Error
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []TeamgroupRow{}
	}
	for i := range out {
		out[i].Name = html.UnescapeString(out[i].Name)
		out[i].ParentName = html.UnescapeString(out[i].ParentName)
	}
	return out, nil
}

// FindTeamgroupByID 按 ID 查小组详情。
func (r *Repo) FindTeamgroupByID(ctx context.Context, id uint) (*TeamgroupRow, error) {
	var row TeamgroupRow
	err := r.read().WithContext(ctx).Raw(`
SELECT tg.id, tg.name, tg.parent,
       COALESCE(p.name, '') AS parent_name,
       tg.type, tg.grade, COALESCE(tg.path, '') AS path,
       tg.PO, tg.manager, tg.slogan, tg.declaration, tg.logo, tg.status,
       COALESCE(DATE_FORMAT(tg.createdDate, '%Y-%m-%d'), '') AS createdDate
FROM zt_teamgroup tg
LEFT JOIN zt_teamgroup p ON p.id = tg.parent AND p.deleted = '0'
WHERE tg.id = ? AND tg.deleted = '0'
LIMIT 1`, id).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	row.Name = html.UnescapeString(row.Name)
	row.ParentName = html.UnescapeString(row.ParentName)
	return &row, nil
}

// ListMembers 查询正式成员。
func (r *Repo) ListMembers(ctx context.Context, teamgroupID uint) ([]TeamMemberRow, error) {
	var rows []TeamMemberRow
	err := r.read().WithContext(ctx).Raw(`
SELECT t.account,
       COALESCE(NULLIF(u.realname, ''), t.account) AS name,
       t.role, t.hours, t.days,
       COALESCE(DATE_FORMAT(t.`+"`join`"+`, '%Y-%m-%d'), '') AS join_date
FROM zt_team t
LEFT JOIN zt_user u ON u.account = t.account AND u.deleted = '0'
WHERE t.type = 'teamgroup' AND t.root = ?
ORDER BY t.account ASC`, teamgroupID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []TeamMemberRow{}
	}
	return rows, nil
}

// ListMembersByGroupIDs 批量查正式成员。
func (r *Repo) ListMembersByGroupIDs(ctx context.Context, ids []uint) (map[uint][]TeamMemberRow, error) {
	out := make(map[uint][]TeamMemberRow, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	type row struct {
		GroupID uint    `gorm:"column:group_id"`
		Account string  `gorm:"column:account"`
		Name    string  `gorm:"column:name"`
		Role    string  `gorm:"column:role"`
		Hours   float64 `gorm:"column:hours"`
		Days    int     `gorm:"column:days"`
		Join    string  `gorm:"column:join_date"`
	}
	var rows []row
	err := r.read().WithContext(ctx).Raw(`
SELECT t.root AS group_id, t.account,
       COALESCE(NULLIF(u.realname, ''), t.account) AS name,
       t.role, t.hours, t.days,
       COALESCE(DATE_FORMAT(t.`+"`join`"+`, '%Y-%m-%d'), '') AS join_date
FROM zt_team t
LEFT JOIN zt_user u ON u.account = t.account AND u.deleted = '0'
WHERE t.type = 'teamgroup' AND t.root IN ?
ORDER BY t.root ASC, t.account ASC`, ids).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		out[it.GroupID] = append(out[it.GroupID], TeamMemberRow{
			Account: it.Account, Name: it.Name, Role: it.Role,
			Hours: it.Hours, Days: it.Days, Join: it.Join,
		})
	}
	return out, nil
}

// ResolveRealnames 批量解析账号 → 姓名。
func (r *Repo) ResolveRealnames(ctx context.Context, accounts []string) (map[string]string, error) {
	out := map[string]string{}
	uniq := make([]string, 0, len(accounts))
	seen := map[string]bool{}
	for _, a := range accounts {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		uniq = append(uniq, a)
	}
	if len(uniq) == 0 {
		return out, nil
	}
	type row struct {
		Account  string `gorm:"column:account"`
		Realname string `gorm:"column:realname"`
	}
	var rows []row
	err := r.read().WithContext(ctx).
		Table("zt_user").
		Select("account, realname").
		Where("account IN ? AND deleted = '0'", uniq).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, it := range rows {
		name := strings.TrimSpace(it.Realname)
		if name == "" {
			name = it.Account
		}
		out[it.Account] = name
	}
	for _, a := range uniq {
		if _, ok := out[a]; !ok {
			out[a] = a
		}
	}
	return out, nil
}

// FindMember 查单个正式成员。
func (r *Repo) FindMember(ctx context.Context, teamgroupID uint, account string) (*TeamMemberRow, error) {
	var row TeamMemberRow
	err := r.read().WithContext(ctx).Raw(`
SELECT t.account,
       COALESCE(NULLIF(u.realname, ''), t.account) AS name,
       t.role, t.hours, t.days,
       COALESCE(DATE_FORMAT(t.`+"`join`"+`, '%Y-%m-%d'), '') AS join_date
FROM zt_team t
LEFT JOIN zt_user u ON u.account = t.account AND u.deleted = '0'
WHERE t.type = 'teamgroup' AND t.root = ? AND t.account = ?
LIMIT 1`, teamgroupID, account).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.Account == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}
